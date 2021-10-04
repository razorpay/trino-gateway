package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

const LOG_TAG string = "GATEWAY_ROUTER: "

type GatewayApiClient struct {
	Policy  gatewayv1.PolicyApi
	Backend gatewayv1.BackendApi
	Group   gatewayv1.GroupApi
	Query   gatewayv1.QueryApi
}

type RouterServer struct {
	gatewayApiClient *GatewayApiClient
	port             int
	routerHostname   string
}

type key int

const keyCtxSharedObj key = iota

/*
	For data sharing between processClientReq & processClientResponse, we hav following approaches

	1. context - Be careful with it, twirp clients require the context sent to http.Server whereas sharing should be done on req.Context (req object sent to Director), otherwise the http.Server context will keep getting shared ctx values added to it unless the Server is shutdown, also this ctx will be shared across all requests sent to the server
	TODO - have unit tests ensuring http.Server ctx is not being modified (i.e. none changes *ctx), maybe by checking fmt.Sprintf("%v", *ctx)

	2. pointers - issues with synchronization, also hinders readability and code is prone to bugs
	              won't work with concurrent requests
	3. goroutines + channel - cleaner but extra overhead
*/
type ContextSharedObject struct {
	query          *gatewayv1.Query
	timerSt        *time.Time
	preRoutingErr  *error
	postRoutingErr *error
}

func init() {
	initMetrics()
}

func Server(ctx *context.Context, port int, apiClient *GatewayApiClient, routerHostname string) *http.Server {
	routerServer := RouterServer{
		port:             port,
		gatewayApiClient: apiClient,
		routerHostname:   routerHostname,
	}
	reverseProxy := httputil.ReverseProxy{
		Director:  func(req *http.Request) { routerServer.handleClientRequest(ctx, req) },
		Transport: nil,
		ModifyResponse: func(resp *http.Response) error {
			return routerServer.handleServerResponse(ctx, resp)
		},
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			provider.Logger(*ctx).WithError(err).Errorw(
				fmt.Sprint(LOG_TAG, "HttpReverseProxy ErrorHandler invoked"),
				map[string]interface{}{
					"request": stringifyHttpRequest(ctx, req),
				})
			ctxSharedObj, e := routerServer.extractSharedRequestCtxObject(ctx, req)
			if e != nil {
				provider.Logger(*ctx).WithError(e).
					Error("unable to cast shared object object from context")
				return
			}

			status := http.StatusBadGateway
			var msg string
			defer func(st time.Time) {
				post_d := time.Since(st).Milliseconds()
				tot_d := time.Since(*ctxSharedObj.timerSt).Milliseconds()
				metrics.responsesSentTotal.
					WithLabelValues(metrics.env, req.Method, fmt.Sprint(status)).
					Inc()
				metrics.requestPostRoutingDelays.
					WithLabelValues(metrics.env, req.Method, fmt.Sprint(status)).
					Observe(float64(post_d))
				metrics.responseDurations.
					WithLabelValues(metrics.env, req.Method, fmt.Sprint(status)).
					Observe(float64(tot_d))
			}(time.Now())

			if *ctxSharedObj.preRoutingErr != nil {
				status, msg = routerServer.handlePreRoutingError(ctx, *ctxSharedObj.preRoutingErr)
			} else if *ctxSharedObj.postRoutingErr != nil {
				status, msg = routerServer.handlePostRoutingError(ctx, *ctxSharedObj.postRoutingErr)
			} else {
				status, msg = routerServer.handleServerError(ctx, err)
			}
			resp.WriteHeader(status)
			resp.Write([]byte(msg))
		},
	}

	return &http.Server{
		Handler: &reverseProxy,
	}
}

func (r *RouterServer) extractSharedRequestCtxObject(ctx *context.Context, req *http.Request) (*ContextSharedObject, error) {
	reqCtx := req.Context()
	res, ok := (reqCtx).Value(keyCtxSharedObj).(*ContextSharedObject)
	if !ok {
		err := errors.New("unable to cast shared object object from context")
		provider.Logger(*ctx).WithError(err).Error("unable to cast shared object object from context")
		return nil, err
	}
	return res, nil
}

func (r *RouterServer) handleClientRequest(ctx *context.Context, req *http.Request) {
	var err error
	st := time.Now()
	metrics.requestsReceivedTotal.
		WithLabelValues(metrics.env, req.Method, fmt.Sprint(r.port))
	defer func(st time.Time) {
		duration := time.Since(st).Milliseconds()
		metrics.requestPreRoutingDelays.
			WithLabelValues(metrics.env, req.Method).
			Observe(float64(duration))
	}(st)

	q, err := r.processRequest(ctx, req)
	if err != nil {
		r.handleClientRequestRoutingError(ctx, req, err)
	} else {
		metrics.requestsRoutedTotal.
			WithLabelValues(metrics.env, req.Method, fmt.Sprint(r.port), q.GroupId, q.BackendId).
			Inc()
	}

	provider.Logger(*ctx).Debugw(
		fmt.Sprint(LOG_TAG, "Request Processed, forwarding to server"),
		map[string]interface{}{
			"host": req.URL.Host,
		})

	c := &ContextSharedObject{
		query:         q,
		timerSt:       &st,
		preRoutingErr: &err,
	}
	reqCtx := context.WithValue(req.Context(), keyCtxSharedObj, c)
	*req = *req.WithContext(reqCtx)
}

func (r *RouterServer) handleClientRequestRoutingError(ctx *context.Context, req *http.Request, err error) {
	provider.Logger(*ctx).WithError(err).Errorw(
		fmt.Sprint(LOG_TAG, "Request Processing failed"),
		map[string]interface{}{
			"req": req,
		})
	req.URL.Host = "http://invalid:8080"
}

func (r *RouterServer) handlePreRoutingError(ctx *context.Context, err error) (status int, msg string) {
	return http.StatusBadRequest, "Gateway couldn't process this request"
}

func (r *RouterServer) handlePostRoutingError(ctx *context.Context, err error) (status int, msg string) {
	return http.StatusBadGateway, "Gateway encountered an error parsing server response"
}

func (r *RouterServer) handleServerError(ctx *context.Context, err error) (status int, msg string) {
	return http.StatusBadGateway, "Trino Server unreachable"
}

func (r *RouterServer) handleServerResponse(ctx *context.Context, resp *http.Response) error {
	ctxSharedObj, err := r.extractSharedRequestCtxObject(ctx, resp.Request)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error("unable to cast shared object from context")
		return err
	}
	err = r.processResponse(ctx, resp, ctxSharedObj.query)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to process server response"),
			map[string]interface{}{
				"error": err.Error(),
			})
	} else {
		defer func(st time.Time) {
			post_d := time.Since(st).Milliseconds()
			tot_d := time.Since(*ctxSharedObj.timerSt).Milliseconds()
			metrics.responsesSentTotal.
				WithLabelValues(metrics.env, resp.Request.Method, fmt.Sprint(200)).
				Inc()
			metrics.requestPostRoutingDelays.
				WithLabelValues(metrics.env, resp.Request.Method, fmt.Sprint(200)).
				Observe(float64(post_d))
			metrics.responseDurations.
				WithLabelValues(metrics.env, resp.Request.Method, fmt.Sprint(200)).
				Observe(float64(tot_d))
		}(time.Now())
	}
	ctxSharedObj.postRoutingErr = &err
	return err
}
