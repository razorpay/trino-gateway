package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/router/trinoheaders"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type ClientRequest struct {
	username                   string
	host                       string
	headerConnectionProperties string
	headerClientTags           string
	queryId                    string
	incomingPort               int32
	queryText                  string
	clientIp                   string
}

func extractQueryId(ctx *context.Context, body string) string {
	// TODO
	return ""
}

func isValidRequest(ctx *context.Context, req *ClientRequest) bool {
	if req.username == "" || req.queryText == "" {
		return false
	}
	return true
}

func constructQueryFromReq(body string, preparedStmt string) string {
	// TODO
	return body
}

func (r *RouterServer) parseClientRequest(ctx *context.Context, req *http.Request) *ClientRequest {
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

	query := constructQueryFromReq(body, trinoheaders.Get(trinoheaders.PreparedStatement, req))

	return &ClientRequest{
		username:                   trinoheaders.Get(trinoheaders.User, req),
		queryId:                    extractQueryId(ctx, query),
		incomingPort:               int32(r.port),
		host:                       req.Host,
		headerConnectionProperties: trinoheaders.Get(trinoheaders.ConnectionProperties, req),
		headerClientTags:           trinoheaders.Get(trinoheaders.ClientTags, req),
		queryText:                  query,
	}
}

func (r *RouterServer) processRequest(
	ctx *context.Context,
	req *http.Request,
) (*gatewayv1.Query, error) {

	var err error = nil
	provider.Logger(*ctx).Infow(
		fmt.Sprint(LOG_TAG, "Request received"),
		map[string]interface{}{
			"request":       stringifyHttpRequest(ctx, req),
			"listeningPort": r.port,
		})

	clientReq := r.parseClientRequest(ctx, req)
	if !isValidRequest(ctx, clientReq) {
		return nil, errors.New("invalid Request - not a valid Trino request")
	}
	querySaveReq := &gatewayv1.Query{
		Id:         clientReq.queryId,
		Text:       clientReq.queryText,
		Username:   clientReq.username,
		ClientIp:   clientReq.clientIp,
		ReceivedAt: time.Now().Unix(),
	}

	if clientReq.queryId != "" {
		findBackendIdResp, err := r.gatewayApiClient.Query.FindBackendForQuery(
			*ctx,
			&gatewayv1.FindBackendForQueryRequest{QueryId: clientReq.queryId},
		)
		if err != nil {
			provider.Logger(*ctx).WithError(err).
				Errorw("Backend Unresolvable for meta query",
					map[string]interface{}{"queryId": clientReq.queryId})
			return nil, err
		}
		bId := findBackendIdResp.BackendId
		r.prepareReqForRouting(ctx, req, bId)
		if err != nil {
			provider.Logger(*ctx).WithError(err).
				Errorw("Backend Unresolvable for meta query",
					map[string]interface{}{"queryId": clientReq.queryId})
			return nil, err
		}
		querySaveReq.BackendId = bId

		return querySaveReq, nil
	}

	provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "evaluating groups for client request"))
	bId, gId, err := r.evaluateRoutingBackend(ctx, clientReq)
	r.prepareReqForRouting(ctx, req, bId)
	if err != nil {
		return nil, err
	}
	querySaveReq.GroupId = gId
	querySaveReq.BackendId = bId

	return querySaveReq, nil
}

func (r *RouterServer) evaluateRoutingBackend(ctx *context.Context, clientReq *ClientRequest) (backendId string, groupId string, err error) {
	evalGrpReq := &gatewayv1.EvaluateGroupsRequest{
		IncomingPort:               clientReq.incomingPort,
		Host:                       clientReq.host,
		HeaderConnectionProperties: clientReq.headerConnectionProperties,
		HeaderClientTags:           clientReq.headerClientTags,
	}
	evalGrpResp, err := r.gatewayApiClient.Policy.
		EvaluateGroupsForClient(*ctx, evalGrpReq)
	if err != nil {
		provider.Logger(*ctx).WithError(err).
			Errorw("Groups resolution encountered error for client", map[string]interface{}{"req": evalGrpReq})
		return "", "", err
	}

	provider.Logger(*ctx).
		Debugw(fmt.Sprint(LOG_TAG, "evaluating backend for groups"), map[string]interface{}{"groups": evalGrpResp.GetGroupIds()})

	evalBackendReq := &gatewayv1.EvaluateBackendRequest{GroupIds: evalGrpResp.GetGroupIds()}
	evalBackendResp, err := r.gatewayApiClient.Group.
		EvaluateBackendForGroups(
			*ctx,
			evalBackendReq,
		)
	if err != nil {
		provider.Logger(*ctx).WithError(err).
			Errorw("Backend Unresolvable for groups", map[string]interface{}{"req": evalBackendReq})
		return "", "", err
	}

	provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "backend resolved"), map[string]interface{}{
		"backend_id": evalBackendResp.GetBackendId(),
		"group_id":   evalBackendResp.GetGroupId(),
	})

	backendId = evalBackendResp.GetBackendId()
	groupId = evalBackendResp.GetGroupId()
	return backendId, groupId, nil
}

func (r *RouterServer) prepareReqForRouting(ctx *context.Context, req *http.Request, backend_id string) error {
	provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "fetching details of resolved backend"))
	findBackendResp, err := r.gatewayApiClient.Backend.GetBackend(
		*ctx,
		&gatewayv1.BackendGetRequest{Id: backend_id},
	)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error(
			fmt.Sprint("Cannot find backend for backend id:",
				backend_id),
		)
		return err
	}
	backend := findBackendResp.GetBackend()

	req.URL.Host = backend.Hostname
	req.URL.Scheme = backend.Scheme.Enum().String()
	req.Host = backend.Hostname
	provider.Logger(*ctx).Infow(
		fmt.Sprint(LOG_TAG, "Request modified, ready to be forwarded"),
		map[string]interface{}{
			"request": stringifyHttpRequest(ctx, req),
		})
	return nil
}
