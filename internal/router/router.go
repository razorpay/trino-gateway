package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"

	// "net/http/httputil"
	// "net/url"

	"github.com/razorpay/trino-gateway/internal/boot"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

// type gatewayCtx struct {
// 	ctx  *context.Context
// 	port int32
// }

// func NewGatewayCtx(ctx *context.Context, port int32) *gatewayCtx {
// 	return &gatewayCtx{ctx, port}
// }

func StartRoutingServer(ctx *context.Context, port int) *http.Server {

	//////////////
	// gg, _ := url.ParseRequestURI("http://localhost:8000")
	// httpHandler := httputil.NewSingleHostReverseProxy(gg)

	// log.Fatal(http.ListenAndServe(":8080", httpHandler))

	/////////

	// http.HandleFunc("/", myHandler)
	// log.Fatal(http.ListenAndServe(fmt.Sprint(":", port), nil))

	///////////

	// proxy := goproxy.NewReverseProxyHttpServer()
	// proxy.Verbose = true

	// log.Fatal(http.ListenAndServe(fmt.Sprint(":", port), proxy))

	////////////

	reverseProxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			_, err := processRequest(ctx, req, port)
			if err != nil {
				log.Println(err.Error())
				req.URL.Host = "http://invalid:8080"
			}

		},
		Transport: nil,
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			resp.Write([]byte("Backend unavailable"))
		},
		ModifyResponse: processResponse,
	}

	return &http.Server{
		Handler: &reverseProxy,
	}
}

// func myHandler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("MyHandler")
// 	fmt.Println(r.URL.Scheme)
// 	fmt.Println(r.URL.Host)
// }

// type myProxy struct{ goproxy.ProxyHttpServer }

// func (proxy *myProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	r.URL.Host = "empty"
// 	r.URL.Scheme = "http"
// 	myProxy.ProxyHttpServer.ServeHTTP(w, r)

// }

// func MyReverseProxyHttpServer() *myProxy {
// 	reverseProxy := goproxy.NewReverseProxyHttpServer()

//     reverseProxy.reqHan
// 	return &myProxy{*reverseProxy}
// }

// func goproxyFunc1(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
// 	r, err := processRequest(ctx, r, 8080)
// 	if err != nil {
// 		return r, goproxy.NewResponse(r,
// 			goproxy.ContentTypeText, http.StatusBadGateway,
// 			err.Error())
// 	}
// 	return r, nil
// }

func processRequest(ctx *context.Context, r *http.Request, listeningPort int) (*http.Request, error) {
	debugPrintReq(r)
	body := parseRequestBody(r)
	println("processing Request")
	println(listeningPort)

	queryId := extractQueryId(body)
	// var req gatewayv1.EvaluateBackendRequest
	var backend *gatewayv1.Backend
	var err error
	if queryId == "" {

		// // headers := make(map[string]*structpb.ListValue, len(r.Header))
		// // for k, v := range r.Header {
		// // 	var vals []*structpb.Value
		// // 	for _, i := range v {
		// // 		vals = append(vals, &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: i}})
		// // 	}
		// // 	headers[k] = &structpb.ListValue{Values: vals}
		// // }

		// // req = gatewayv1.FindBackendRequest{ParamsOneof: &gatewayv1.FindBackendRequest_ClientParams_{
		// // 	ClientParams: &gatewayv1.FindBackendRequest_ClientParams{
		// // 		IncomingPort: int32(listeningPort),
		// // 		Host:         r.Host,
		// // 		Headers:      headers,
		// // 	},
		// // },
		// // }

		// req = gatewayv1.EvaluateBackendRequest{ParamsOneof: &gatewayv1.EvaluateBackendRequest_ClientParams_{
		// 	ClientParams: &gatewayv1.EvaluateBackendRequest_ClientParams{
		// 		IncomingPort:               int32(listeningPort),
		// 		Host:                       r.Host,
		// 		HeaderConnectionProperties: r.Header.Get("X-Trino-Connection-Properties"),
		// 		HeaderClientTags:           r.Header.Get("X-Trino-Client-Tags"),
		// 	},
		// },

		policyClient := gatewayv1.NewPolicyApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{})

		req := gatewayv1.EvaluateGroupRequest{
			IncomingPort:               int32(listeningPort),
			Host:                       r.Host,
			HeaderConnectionProperties: r.Header.Get("X-Trino-Connection-Properties"),
			HeaderClientTags:           r.Header.Get("X-Trino-Client-Tags"),
		}
		group, err := policyClient.EvaluateGroupForClient(*ctx, &req)
		if err != nil {
			println(err.Error())
			return r, errors.New(fmt.Sprint("Group Unresolvable for client id:", req, err.Error()))
		}
		groupClient := gatewayv1.NewGroupApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{})

		backend, err = groupClient.EvaluateBackendForGroup(*ctx, &gatewayv1.EvaluateBackendRequest{
			GroupId: group.Id,
		})
		if err != nil {
			println(err.Error())
			return r, errors.New(fmt.Sprint("Backend Unresolvable for query id:", queryId, err.Error()))
		}
	} else {
		// req = gatewayv1.EvaluateBackendRequest{ParamsOneof: &gatewayv1.EvaluateBackendRequest_QueryId{
		// 	QueryId: queryId,
		// }}
		client := gatewayv1.NewQueryApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{})

		req := gatewayv1.FindBackendForQueryRequest{QueryId: queryId}

		backend, err = client.FindBackendForQuery(*ctx, &req)
		if err != nil {
			println(err.Error())
			return r, errors.New(fmt.Sprint("Backend Unresolvable for query id:", queryId, err.Error()))
		}
	}

	// client := gatewayv1.NewGatewayApiProtobufClient(fmt.Sprint("http://localhost:", boot.Config.App.Port), &http.Client{})

	// backend, err := client.EvaluateBackendForClient(*ctx, &req)
	// if err != nil {
	// 	println(err.Error())
	// 	return r, errors.New(fmt.Sprint("Backend Unresolvable", err.Error()))

	// }

	r.URL.Host = backend.Hostname
	r.URL.Scheme = backend.Scheme.Enum().String()
	r.Host = backend.Hostname
	debugPrintReq(r)
	return r, nil
}

func processResponse(resp *http.Response) error {
	// Handle Redirects
	// TODO: Clean it up
	regex := regexp.MustCompile(`\w+\:\/\/[^\/]*(.*)`)
	if resp.Header.Get("Location") != ("") {
		oldLoc := resp.Header.Get("Location")
		newLoc := fmt.Sprint("http://", boot.Config.App.ServiceExternalHostname, regex.ReplaceAllString(oldLoc, "$1"))
		resp.Header.Set("Location", newLoc)
	}
	return nil
}

func debugPrintReq(r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(requestDump))
}

func parseRequestBody(req *http.Request) string {
	// Assumption that HTTP spec is followed and body in GET is meaningless
	if req.Method == "GET" {
		return ""
	}
	bodyBytes, _ := io.ReadAll(req.Body)
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure a it is restored in original request
	req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}

func extractQueryId(body string) string {
	// TODO
	return ""
}

func saveQuery(queryId string, backend string) {
	// TODO
}
