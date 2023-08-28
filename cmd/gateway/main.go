package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/twitchtv/twirp"

	"github.com/razorpay/trino-gateway/internal/boot"
	guiserver "github.com/razorpay/trino-gateway/internal/frontend/server"
	backendapi "github.com/razorpay/trino-gateway/internal/gatewayserver/backendApi"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	groupapi "github.com/razorpay/trino-gateway/internal/gatewayserver/groupApi"
	healthapi "github.com/razorpay/trino-gateway/internal/gatewayserver/healthApi"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/hooks"
	policyapi "github.com/razorpay/trino-gateway/internal/gatewayserver/policyApi"
	queryapi "github.com/razorpay/trino-gateway/internal/gatewayserver/queryApi"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
	"github.com/razorpay/trino-gateway/internal/monitor"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/router"
	"github.com/razorpay/trino-gateway/pkg/fetcher"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
	// "github.com/razorpay/trino-gateway/twirpql"
)

const (
	appSwaggerUiPath = "/admin/swaggerui/"
	// appApiPath       = "/api"
	// appHealthPath = "/health"
	// appHealthPath = healthv1.HealthCheckAPIPathPrefix
	// appTwirpqlPath = "/admin/twirpql"
)

func main() {
	// Initialize context
	ctx, cancel := context.WithCancel(boot.NewContext(context.Background()))
	defer cancel()

	// Init app dependencies
	env := boot.GetEnv()
	err := boot.InitApi(ctx, env)
	if err != nil {
		log.Fatalf("failed to init api: %v", err)
	}

	// traceCloser, err := boot.InitTracing(ctx)
	// if err != nil {
	// 	log.Fatalf("error initializing tracer: %v", err)
	// }
	// defer traceCloser.Close()

	provider.Logger(ctx).Debug(fmt.Sprint(boot.Config))

	// Start Api Server
	apiServer := startApiServer(&ctx)

	// Start ReverseProxy Server
	gatewayServers := startGatewayServers(&ctx)

	// Start GUI Server
	// gui server will be on same port as of apiServer till the frontend is client sided
	// guiServer := startGuiServer(&ctx)

	// App metrics server
	metricServer := startMetricsServer(&ctx)

	// start backend health monitor
	startMonitor(&ctx)

	c := make(chan os.Signal, 1)

	// accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM.
	// SIGKILL, SIGQUIT will not be caught.
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	// Block until signal is received.
	<-c
	// shutDown(ctx, httpServers, healthCore)
	shutDown(ctx, append(append(gatewayServers, apiServer), metricServer)...)
}

func startGatewayServers(_ctx *context.Context) []*http.Server {
	// Start trino gateway reverse proxy servers on ports

	gatewayApiUrl := fmt.Sprint("http://localhost:", boot.Config.App.Port)
	gatewayClient := router.GatewayApiClient{
		Group:   gatewayv1.NewGroupApiProtobufClient(gatewayApiUrl, &http.Client{}),
		Policy:  gatewayv1.NewPolicyApiProtobufClient(gatewayApiUrl, &http.Client{}),
		Backend: gatewayv1.NewBackendApiProtobufClient(gatewayApiUrl, &http.Client{}),
		Query:   gatewayv1.NewQueryApiProtobufClient(gatewayApiUrl, &http.Client{}),
	}

	header := make(http.Header)
	header.Set(boot.Config.Auth.TokenHeaderKey, boot.Config.Auth.Token)
	ctx, err := twirp.WithHTTPRequestHeaders(*_ctx, header)
	if err != nil {
		log.Printf("twirp error setting headers: %s", err)
		return nil
	}

	servers := make([]*http.Server, len(boot.Config.Gateway.Ports))
	for i, port := range boot.Config.Gateway.Ports {
		server := router.Server(&ctx, port, &gatewayClient, boot.Config.App.ServiceExternalHostname, boot.Config.Auth.Router.Authenticate)
		servers[i] = server

		go listenHttp(&ctx, server, port)
	}
	return servers
}

func listenHttp(ctx *context.Context, server *http.Server, port int) {
	listener, err := net.Listen("tcp4", fmt.Sprint(boot.Config.Gateway.Network, ":", port))
	if err != nil {
		panic(err)
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		provider.Logger(*ctx).WithContext(*ctx, nil).Fatalw("Failed to start http listener", map[string]interface{}{"error": err})
	}
}

func startMonitor(_ctx *context.Context) {
	// Start backend health check monitors
	gatewayApiUrl := fmt.Sprint("http://localhost:", boot.Config.App.Port)
	client := gatewayv1.NewBackendApiProtobufClient(gatewayApiUrl, &http.Client{})
	core := monitor.NewCore(client)

	header := make(http.Header)
	header.Set(boot.Config.Auth.TokenHeaderKey, boot.Config.Auth.Token)
	ctx, err := twirp.WithHTTPRequestHeaders(*_ctx, header)
	if err != nil {
		log.Printf("twirp error setting headers: %s", err)
		return
	}

	m := monitor.NewMonitor(core)
	err = m.Schedule(&ctx, boot.Config.Monitor.Interval)
	if err != nil {
		provider.Logger(ctx).WithError(err).Fatal(
			"Unable to start Monitoring module",
		)
	}
}

// Unused, gui is launched from apiServer, till frontend is fixed
// func startGuiServer(ctx *context.Context) *http.Server {
// 	mux := http.NewServeMux()
// 	// mux.HandleFunc("/", gatewayui.HttpHandler)
// 	fs := http.FileServer(http.Dir("./web/frontend"))
// 	appFrontendPath := "/"
// 	mux.Handle(appFrontendPath, http.StripPrefix(appFrontendPath, fs))
// 	httpServer := http.Server{Handler: mux}
// 	go listenHttp(ctx, &httpServer, boot.Config.App.GuiPort)
// 	return &httpServer
// }

func startApiServer(ctx *context.Context) *http.Server {
	// Init http and register servers to mux
	mux := http.NewServeMux()

	// // Define server handlers
	healthServer := healthapi.NewServer(healthapi.NewCore())
	healthServerHandler := gatewayv1.NewHealthCheckAPIServer(healthServer, nil)

	gatewayDbRepo := dbRepo.NewDbRepo(boot.DB)
	gatewayBackendRepo := repo.NewBackendRepo(gatewayDbRepo)

	fetcherClient := fetcher.New(boot.DB.Instance(*ctx))

	gatewayBackendCore := backendapi.NewCore(gatewayBackendRepo)
	gatewayGroupCore := groupapi.NewCore(repo.NewGroupRepo(gatewayDbRepo), gatewayBackendRepo)
	gatewayPolicyCore := policyapi.NewCore(repo.NewPolicyRepo(gatewayDbRepo))
	gatewayQueryCore := queryapi.NewCore(repo.NewQueryRepo(gatewayDbRepo), fetcherClient)

	gatewayBackendServer := backendapi.NewServer(gatewayBackendCore)
	gatewayGroupServer := groupapi.NewServer(gatewayGroupCore)
	gatewayPolicyServer := policyapi.NewServer(gatewayPolicyCore)
	gatewayQueryServer := queryapi.NewServer(gatewayQueryCore)

	gatewayBackendServerHandler := gatewayv1.NewBackendApiServer(gatewayBackendServer, twirpHooks())
	gatewayGroupServerHandler := gatewayv1.NewGroupApiServer(gatewayGroupServer, twirpHooks())
	gatewayPolicyServerHandler := gatewayv1.NewPolicyApiServer(gatewayPolicyServer, twirpHooks())
	gatewayQueryServerHandler := gatewayv1.NewQueryApiServer(gatewayQueryServer, twirpHooks())

	// // Ensure defaultRoutingGroup is present in healthcheck
	mux.Handle(gatewayv1.HealthCheckAPIPathPrefix, healthServerHandler)
	// grp, gatewayGroupCore.GetGroup(*ctx, boot.Config.Gateway.DefaultRoutingGroup)

	mux.Handle(gatewayv1.BackendApiPathPrefix, hooks.WithAuth(gatewayBackendServerHandler))
	mux.Handle(gatewayv1.GroupApiPathPrefix, hooks.WithAuth(gatewayGroupServerHandler))
	mux.Handle(gatewayv1.PolicyApiPathPrefix, hooks.WithAuth(gatewayPolicyServerHandler))
	mux.Handle(gatewayv1.QueryApiPathPrefix, hooks.WithAuth(gatewayQueryServerHandler))

	// Serve the current git commit hash
	mux.HandleFunc("/commit.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, boot.Config.App.GitCommitHash)
	})

	fs := http.FileServer(http.Dir("./third_party/swaggerui"))
	mux.Handle(appSwaggerUiPath, http.StripPrefix(appSwaggerUiPath, fs))

	mux.Handle("/", *guiserver.NewServerHandler(ctx))

	// mux.Handle("/twirpql", twirpql.Handler(gatewayServer, nil))
	// mux.Handle("/admin/twirpql/play", twirpql.Playground("my service", "/twirpql"))

	// Serve request - http.Serve
	httpServer := http.Server{Handler: mux}

	// Start app server listener
	go listenHttp(ctx, &httpServer, boot.Config.App.Port)

	return &httpServer
}

func startMetricsServer(ctx *context.Context) *http.Server {
	httpServer := http.Server{Handler: promhttp.Handler()}
	go listenHttp(ctx, &httpServer, boot.Config.App.MetricsPort)
	return &httpServer
}

// twirpHooks register common twirp hooks applicable to all endpoints.
func twirpHooks() *twirp.ServerHooks {
	return twirp.ChainHooks(
		hooks.Metric(),
		hooks.RequestID(),
		hooks.Auth(),
		hooks.Ctx())
}

// shutDown the application, gracefully
func shutDown(ctx context.Context, servers ...*http.Server) {
	// send unhealthy status to the healthcheck probe and let
	// it mark this pod OOR first before shutting the server down
	// logger.Ctx(ctx).Info("Marking server unhealthy")
	// healthCore.MarkUnhealthy()

	// wait for ShutdownDelay seconds
	time.Sleep(time.Duration(boot.Config.App.ShutdownDelay) * time.Second)

	// Create a deadline to wait for.
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(boot.Config.App.ShutdownTimeout)*time.Second)
	defer cancel()

	provider.Logger(ctx).Info("Shutting down trino-gateway")

	for _, server := range servers {
		go func(server *http.Server) {
			err := server.Shutdown(ctxWithTimeout)
			if err != nil {
				provider.Logger(ctx).Errorw("Failed to initiate shutdown", map[string]interface{}{"error": err})
			}
		}(server)
	}
}
