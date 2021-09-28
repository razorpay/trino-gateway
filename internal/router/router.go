package router

import (
	"bytes"
	"context"
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
	ctx              *context.Context
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
		ctx:              ctx,
		port:             port,
		gatewayApiClient: apiClient,
		routerHostname:   routerHostname,
	}
	reverseProxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			_, err := routerServer.processRequest(req)
			if err != nil {
				provider.Logger(*routerServer.ctx).Errorw(
					fmt.Sprint(LOG_TAG, "Request Processing failed"),
					map[string]interface{}{
						"error": err.Error(),
					})
				req.URL.Host = "http://invalid:8080"
			}

		},
		Transport: nil,
		ErrorHandler: func(resp http.ResponseWriter, req *http.Request, err error) {
			msg := "Backend unavailable Or Invalid request"
			resp.Write([]byte(msg))
			provider.Logger(*ctx).Errorw(
				fmt.Sprint(LOG_TAG, msg),
				map[string]interface{}{
					"request": routerServer.stringifyHttpRequest(req),
				})
		},
		ModifyResponse: func(resp *http.Response) error {
			err := routerServer.processResponse(resp)
			if err != nil {
				provider.Logger(*routerServer.ctx).Errorw(
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

func (r *routerServer) parseClientRequest(req *http.Request) *clientRequest {

	var body string
	// Assumption that HTTP spec is followed and body in GET is meaningless
	if req.Method == "GET" {
		body = ""
	} else {
		body = parseBody(req.Body)
	}
	query := constructQueryFromReq(body, req.Header.Get("X-Trino-Prepared-Statement"))

	return &clientRequest{
		username:                   req.Header.Get("X-Trino-User"),
		queryId:                    extractQueryId(query),
		incomingPort:               int32(r.port),
		host:                       req.Host,
		headerConnectionProperties: req.Header.Get("X-Trino-Connection-Properties"),
		headerClientTags:           req.Header.Get("X-Trino-Client-Tags"),
		queryText:                  query,
	}
}

func (r *routerServer) processRequest(req *http.Request) (*http.Request, error) {
	var err error = nil
	provider.Logger(*r.ctx).Infow(
		fmt.Sprint(LOG_TAG, "Request received"),
		map[string]interface{}{
			"request":       r.stringifyHttpRequest(req),
			"listeningPort": r.port,
		})

	clientReq := r.parseClientRequest(req)
	if !isValidRequest(clientReq) {
		return req, errors.New("invalid request")
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
		findBackendResp, err := r.gatewayApiClient.Backend.GetBackend(
			*r.ctx,
			&gatewayv1.BackendGetRequest{Id: backend_id},
		)
		if err != nil {
			err = errors.New(
				fmt.Sprint("Cannot find backend for backend id:",
					backend_id,
					err.Error()),
			)
		}
		backend = findBackendResp.GetBackend()

		req.URL.Host = backend.Hostname
		req.URL.Scheme = backend.Scheme.Enum().String()
		req.Host = backend.Hostname
		provider.Logger(*r.ctx).Infow(
			fmt.Sprint(LOG_TAG, "Request modified, ready to be forwarded"),
			map[string]interface{}{
				"request": r.stringifyHttpRequest(req),
			})

		querySaveReq.BackendId = backend.Id
	}

	if clientReq.queryId == "" {

		evalGrpResp, err := r.gatewayApiClient.Policy.EvaluateGroupsForClient(*r.ctx, &gatewayv1.EvaluateGroupsRequest{
			IncomingPort:               clientReq.incomingPort,
			Host:                       clientReq.host,
			HeaderConnectionProperties: clientReq.headerConnectionProperties,
			HeaderClientTags:           clientReq.headerClientTags,
		})
		if err != nil {
			err = errors.New(fmt.Sprint("Groups Unresolvable for client", req, err.Error()))
		} else {
			evalBackendResp, err := r.gatewayApiClient.Group.EvaluateBackendForGroups(
				*r.ctx,
				&gatewayv1.EvaluateBackendRequest{GroupIds: evalGrpResp.GetGroupIds()},
			)
			if err != nil {
				err = errors.New(
					fmt.Sprint("Backend Unresolvable for groups",
						evalGrpResp.GetGroupIds(),
						err.Error()),
				)
			}
			querySaveReq.GroupId = evalBackendResp.GetGroupId()
			backendFound(evalBackendResp.GetBackendId())
		}
	} else {
		findBackendIdResp, err := r.gatewayApiClient.Query.FindBackendForQuery(
			*r.ctx,
			&gatewayv1.FindBackendForQueryRequest{QueryId: clientReq.queryId},
		)
		if err != nil {
			err = errors.New(fmt.Sprint("Backend Unresolvable for query id:", clientReq.queryId, err.Error()))
		} else {
			backendFound(findBackendIdResp.BackendId)
		}
	}

	_, err2 := r.gatewayApiClient.Query.CreateOrUpdateQuery(*r.ctx, &querySaveReq)
	if err2 != nil {
		provider.Logger(
			*r.ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to save query"),
			map[string]interface{}{
				"query_id": querySaveReq.Id,
				"error":    err2.Error()})
	}

	return req, err
}

func (r *routerServer) processResponse(resp *http.Response) error {
	// Handle Redirects
	// TODO: Clean it up
	regex := regexp.MustCompile(`\w+\:\/\/[^\/]*(.*)`)
	if resp.Header.Get("Location") != ("") {
		oldLoc := resp.Header.Get("Location")
		newLoc := fmt.Sprint("http://", r.routerHostname, regex.ReplaceAllString(oldLoc, "$1"))
		resp.Header.Set("Location", newLoc)
	}

	go func() {
		isQuerySubmissionSuccessful := true
		if isQuerySubmissionSuccessful {
			queryId := extractQueryIdFromServerResponse(parseBody(resp.Body))
			req := gatewayv1.Query{
				// TODO
				Id:          queryId,
				SubmittedAt: time.Now().Unix(),
			}

			_, err := r.gatewayApiClient.Query.CreateOrUpdateQuery(*r.ctx, &req)
			if err != nil {
				provider.Logger(*r.ctx).Errorw(LOG_TAG, map[string]interface{}{"msg": "Unable to save successfully submitted query", "query_id": req.Id, "error": err.Error()})
			}
		}
	}()

	return nil
}

func extractQueryIdFromServerResponse(body string) string {
	return ""
}

func (r *routerServer) stringifyHttpRequest(req *http.Request) string {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		provider.Logger(*r.ctx).Errorw(
			fmt.Sprint(LOG_TAG, "Unable to stringify http request"),
			map[string]interface{}{
				"error": err.Error(),
			})
	}
	return string(requestDump)
}

func parseBody(body io.ReadCloser) string {
	bodyBytes, _ := io.ReadAll(body)
	// since its a ReadCloser type, the stream will be empty after its read once
	// ensure a it is restored in original request
	body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}

func extractQueryId(body string) string {
	// TODO
	return ""
}

func isValidRequest(req *clientRequest) bool {
	if req.username == "" || req.queryText == "" {
		return false
	}
	return true
}
