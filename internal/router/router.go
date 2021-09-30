package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"regexp"
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

type routerServer struct {
	gatewayApiClient *GatewayApiClient
	port             int
	routerHostname   string
}

type clientRequest struct {
	username                   string
	host                       string
	headerConnectionProperties string
	headerClientTags           string
	queryId                    string
	incomingPort               int32
	queryText                  string
	clientIp                   string
}

func Server(ctx *context.Context, port int, apiClient *GatewayApiClient, routerHostname string) *http.Server {

	routerServer := routerServer{
		port:             port,
		gatewayApiClient: apiClient,
		routerHostname:   routerHostname,
	}

	/*
		For data sharing between processClientReq & processClientResponse, we hav following approaches
		1. context - Tried it, modified using req.WithValue() but in resp.Request.Context() modifications were not propagated.
		2. pointers - issues with synchronization, also hinders readability and code is prone to bugs
		3. goroutines + channel - cleaner but extra overhead
	*/
	var query gatewayv1.Query

	reverseProxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req, err := routerServer.processRequest(ctx, req, &query)
			if err != nil {
				provider.Logger(*ctx).Errorw(
					fmt.Sprint(LOG_TAG, "Request Processing failed"),
					map[string]interface{}{
						"error": err.Error(),
					})
				req.URL.Host = "http://invalid:8080"
			}

			provider.Logger(*ctx).Debugw(
				fmt.Sprint(LOG_TAG, "Request Processed, forwarding to server"),
				map[string]interface{}{
					"host": req.URL.Host,
				})

		},
		Transport: nil,
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			msg := "Backend unavailable Or Invalid request"
			resp.Write([]byte(msg))
			provider.Logger(*ctx).WithError(err).Errorw(
				fmt.Sprint(LOG_TAG, msg),
				map[string]interface{}{
					"request": routerServer.stringifyHttpRequest(ctx, req),
				})
		},
		ModifyResponse: func(resp *http.Response) error {
			err := routerServer.processResponse(ctx, resp, &query)
			if err != nil {
				provider.Logger(*ctx).Errorw(
					fmt.Sprint(LOG_TAG, "Unable to process server response"),
					map[string]interface{}{
						"error": err.Error(),
					})
			}
			return err
		},
	}

	return &http.Server{
		Handler: &reverseProxy,
	}
}

func constructQueryFromReq(body string, preparedStmt string) string {
	// TODO
	return body
}

func (r *routerServer) parseClientRequest(ctx *context.Context, req *http.Request) *clientRequest {

	body := ""
	// Assumption that HTTP spec is followed and body in GET is meaningless
	if req.Method == "GET" {
		body = ""
	} else if req.Method == "POST" {
		b, e := parseBody(ctx, &req.Body)
		if e != nil {
			provider.Logger(*ctx).WithError(e).Error(fmt.Sprint(LOG_TAG, "unable to parse body of client request"))
		}
		provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "parsed body of client request"),
			map[string]interface{}{
				"body": body,
			})
		body = b
	}

	query := constructQueryFromReq(body, req.Header.Get("X-Trino-Prepared-Statement"))

	return &clientRequest{
		username:                   req.Header.Get("X-Trino-User"),
		queryId:                    extractQueryId(ctx, query),
		incomingPort:               int32(r.port),
		host:                       req.Host,
		headerConnectionProperties: req.Header.Get("X-Trino-Connection-Properties"),
		headerClientTags:           req.Header.Get("X-Trino-Client-Tags"),
		queryText:                  query,
	}
}

func (r *routerServer) processRequest(ctx *context.Context, req *http.Request, q *gatewayv1.Query) (*http.Request, error) {
	var err error = nil
	provider.Logger(*ctx).Infow(
		fmt.Sprint(LOG_TAG, "Request received"),
		map[string]interface{}{
			"request":       r.stringifyHttpRequest(ctx, req),
			"listeningPort": r.port,
		})

	clientReq := r.parseClientRequest(ctx, req)
	if !isValidRequest(ctx, clientReq) {
		return req, errors.New("not a valid Trino request")
	}
	querySaveReq := gatewayv1.Query{
		Id:         clientReq.queryId,
		Text:       clientReq.queryText,
		Username:   clientReq.username,
		ClientIp:   clientReq.clientIp,
		ReceivedAt: time.Now().Unix(),
	}

	var backend *gatewayv1.Backend

	backendFound := func(backend_id string) {
		provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "fetching details of resolved backend"))
		findBackendResp, err2 := r.gatewayApiClient.Backend.GetBackend(
			*ctx,
			&gatewayv1.BackendGetRequest{Id: backend_id},
		)
		if err2 != nil {
			err = errors.New(
				fmt.Sprint("Cannot find backend for backend id:",
					backend_id,
					err2.Error()),
			)
		}
		backend = findBackendResp.GetBackend()

		req.URL.Host = backend.Hostname
		req.URL.Scheme = backend.Scheme.Enum().String()
		req.Host = backend.Hostname
		provider.Logger(*ctx).Infow(
			fmt.Sprint(LOG_TAG, "Request modified, ready to be forwarded"),
			map[string]interface{}{
				"request": r.stringifyHttpRequest(ctx, req),
			})

		querySaveReq.BackendId = backend.Id
	}

	if clientReq.queryId == "" {

		provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "evaluating groups for client request"))
		evalGrpResp, err2 := r.gatewayApiClient.Policy.EvaluateGroupsForClient(*ctx, &gatewayv1.EvaluateGroupsRequest{
			IncomingPort:               clientReq.incomingPort,
			Host:                       clientReq.host,
			HeaderConnectionProperties: clientReq.headerConnectionProperties,
			HeaderClientTags:           clientReq.headerClientTags,
		})
		if err2 != nil {
			err = errors.New(fmt.Sprint("Groups resolution encountered error for client", req, err2.Error()))
		} else {
			provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "evaluating backend for groups"), map[string]interface{}{"groups": evalGrpResp.GetGroupIds()})
			evalBackendResp, err2 := r.gatewayApiClient.Group.EvaluateBackendForGroups(
				*ctx,
				&gatewayv1.EvaluateBackendRequest{GroupIds: evalGrpResp.GetGroupIds()},
			)
			if err2 != nil {
				err = errors.New(
					fmt.Sprint("Backend Unresolvable for groups",
						evalGrpResp.GetGroupIds(),
						err2.Error()),
				)
			} else {
				provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "backend resolved"), map[string]interface{}{
					"backend_id": evalBackendResp.GetBackendId(),
					"group_id":   evalBackendResp.GetGroupId(),
				})
				querySaveReq.GroupId = evalBackendResp.GetGroupId()
				backendFound(evalBackendResp.GetBackendId())
			}
		}
	} else {
		findBackendIdResp, err2 := r.gatewayApiClient.Query.FindBackendForQuery(
			*ctx,
			&gatewayv1.FindBackendForQueryRequest{QueryId: clientReq.queryId},
		)
		if err2 != nil {
			err = errors.New(fmt.Sprint("Backend Unresolvable for query id:", clientReq.queryId, err2.Error()))
		} else {
			backendFound(findBackendIdResp.BackendId)
		}
	}

	if err != nil {
		return req, err
	}

	*q = querySaveReq

	return req, err
}

func (r *routerServer) processResponse(ctx *context.Context, resp *http.Response, q *gatewayv1.Query) error {
	// Handle Redirects
	// TODO: Clean it up
	regex := regexp.MustCompile(`\w+\:\/\/[^\/]*(.*)`)
	if resp.Header.Get("Location") != ("") {
		oldLoc := resp.Header.Get("Location")
		newLoc := fmt.Sprint("http://", r.routerHostname, regex.ReplaceAllString(oldLoc, "$1"))
		resp.Header.Set("Location", newLoc)
	}

	func() {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "HAKUNA MATATA"),
			map[string]interface{}{
				"resp": r.stringifyHttpResponse(ctx, resp)})

		isQuerySubmissionSuccessful := true
		if isQuerySubmissionSuccessful {

			req := q
			body, err := parseBody(ctx, &resp.Body)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Error(fmt.Sprint(LOG_TAG, "unable to parse body of server response"))
			}
			req.Id = extractQueryIdFromServerResponse(ctx, body)
			req.SubmittedAt = time.Now().Unix()

			_, err2 := r.gatewayApiClient.Query.CreateOrUpdateQuery(*ctx, req)
			if err2 != nil {
				provider.Logger(
					*ctx).Errorw(
					fmt.Sprint(LOG_TAG, "Unable to save query"),
					map[string]interface{}{
						"query_id": req.Id,
						"error":    err2.Error()})
			}
		}
		return
	}()

	return nil
}

func extractQueryIdFromServerResponse(ctx *context.Context, body string) string {
	provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "extracting queryId from server response"),
		map[string]interface{}{
			"body": body,
		})
	var resp struct{ Id string }
	json.Unmarshal([]byte(body), &resp)
	return resp.Id
}

func (r *routerServer) stringifyHttpRequest(ctx *context.Context, req *http.Request) string {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to stringify http request"),
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(requestDump)
}

func (r *routerServer) stringifyHttpResponse(ctx *context.Context, req *http.Response) string {
	responseDump, err := httputil.DumpResponse(req, true)
	if err != nil {
		provider.Logger(*ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to stringify http response"),
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(responseDump)
}

func parseBody(ctx *context.Context, body *io.ReadCloser) (string, error) {
	//b := req.Body

	bodyBytes, err := io.ReadAll(*body)
	if err != nil {
		return "", err
	}
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure a it is restored in original request
	*body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes), nil
}

func extractQueryId(ctx *context.Context, body string) string {
	// TODO
	return ""
}

func isValidRequest(ctx *context.Context, req *clientRequest) bool {
	if req.username == "" || req.queryText == "" {
		return false
	}
	return true
}
