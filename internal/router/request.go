package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/router/trinoheaders"
	"github.com/razorpay/trino-gateway/internal/utils"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

func extractQueryId(ctx *context.Context, body string) string {
	isKillQuery := func() bool {
		regex := regexp.MustCompile(`(?:call|CALL)[\ ]+(?:(?:system|"system").(?:runtime|"runtime").(?:kill_query|"kill_query"))`)

		if regex.FindString(body) != "" {
			provider.Logger(*ctx).Debugw(" kill_query procedure deteccted", map[string]interface{}{
				"query": body,
			})
			return true
		}
		return false
	}

	if isKillQuery() {
		regex := regexp.MustCompile(`(?:(?:kill_query|"kill_query")[\ ]*\([\ ]*'([\w_-]*)'|(?:query_id[\ ]*=>[\ ]*'([\w_-]*)'))`)

		match := regex.FindStringSubmatch(body)
		if len(match) != 3 {
			provider.Logger(*ctx).Errorw("unable to extract queryId from kill_query procedure", map[string]interface{}{
				"query":      body,
				"regexMatch": match,
			})

			return ""
		}

		qId := match[1]
		if qId == "" {
			qId = match[2] // Exactly one of them will be empty
		}

		provider.Logger(*ctx).Debugw("Extracted queryId from kill_query procedure", map[string]interface{}{
			"query":      body,
			"regexMatch": match,
			"queryId":    qId,
		})

		return qId
	}

	return ""
}

func constructQueryFromReq(ctx *context.Context, req *http.Request) (string, error) {
	if req.Method == "GET" {
		// Assumption that HTTP spec is followed and body in GET is meaningless
		return "", nil
	} else if req.Method == "POST" {
		body, err := utils.ParseHttpPayloadBody(ctx, &req.Body, req.Header.Get("Content-Encoding"))
		if err != nil {
			return "", err
		}
		provider.Logger(*ctx).Debugw(fmt.Sprint(LOG_TAG, "parsed body of client request"),
			map[string]interface{}{
				"body": body,
			})
		preparedStmt := trinoheaders.Get(trinoheaders.PreparedStatement, req)
		// TODO
		_ = preparedStmt
		return body, nil

	}

	// Other request methods are not unsupported
	return "", errors.New(
		fmt.Sprintf(
			"unsupported request method '%s' for extracting query ID from request body",
			req.Method,
		),
	)
}

func (r *RouterServer) ParseClientRequest(ctx *context.Context, req *http.Request) (cReq ClientRequest, err error) {
	if req.Method == "GET" {
		if strings.Contains(req.URL.Path, "ui/") {
			return &UiRequest{
				queryId: req.URL.RawQuery,
			}, nil
		} else if strings.Contains(req.URL.Path, "v1/info") ||
			strings.Contains(req.URL.Path, "v1/status") {
			return &ApiRequest{}, nil
		}
	} else if req.Method == "POST" {

		qText, err := constructQueryFromReq(ctx, req)
		if err != nil {
			provider.Logger(*ctx).WithError(err).Error(fmt.Sprint(LOG_TAG, "unable to construct Sql query from request"))
		}
		queryId := extractQueryId(ctx, qText)

		query := &gatewayv1.Query{
			Id:       queryId,
			Text:     qText,
			Username: trinoheaders.Get(trinoheaders.User, req),
			ClientIp: req.RemoteAddr,
		}

		return &QueryRequest{
			incomingPort:               int32(r.port),
			headerConnectionProperties: trinoheaders.Get(trinoheaders.ConnectionProperties, req),
			headerClientTags:           trinoheaders.Get(trinoheaders.ClientTags, req),
			transactionId:              trinoheaders.Get(trinoheaders.TransactionId, req),
			Query:                      query,
			clientHost:                 req.Host,
		}, nil
	} else if req.Method == "DELETE" && strings.HasPrefix(req.URL.Path, "/v1/query") {
		queryId := strings.TrimPrefix(req.URL.Path, "/v1/query/")
		query := &gatewayv1.Query{
			Id:       queryId,
			Username: trinoheaders.Get(trinoheaders.User, req),
			ClientIp: req.RemoteAddr,
		}

		return &QueryApiRequest{
			incomingPort:               int32(r.port),
			headerConnectionProperties: trinoheaders.Get(trinoheaders.ConnectionProperties, req),
			headerClientTags:           trinoheaders.Get(trinoheaders.ClientTags, req),
			transactionId:              trinoheaders.Get(trinoheaders.TransactionId, req),
			Query:                      query,
			clientHost:                 req.Host,
		}, nil
	}
	return nil, errors.New("client request type not supported by gateway")
}

func (r *RouterServer) ProcessRequest(ctx *context.Context, req *http.Request) (cReq ClientRequest, err error) {
	cReq, err = r.ParseClientRequest(ctx, req)
	if err != nil {
		return nil, errors.New(fmt.Sprint("unable to parse Trino Request - ", err.Error()))
	}
	provider.Logger(*ctx).Infow(
		fmt.Sprint(LOG_TAG, "Request received"),
		map[string]interface{}{
			"request":       utils.StringifyHttpRequestOrResponse(ctx, req),
			"request.Url":   req.URL,
			"listeningPort": r.port,
			"request.Query": req.URL.Query(),
		})

	if err := cReq.Validate(); err != nil {
		return nil, errors.New(fmt.Sprint("invalid Trino Request - ", err.Error()))
	}

	switch nt := cReq.(type) {
	case *ApiRequest:
		// TODO
		return nil, nil

	case *UiRequest:
		findBackendIdResp, err := r.gatewayApiClient.Query.FindBackendForQuery(
			*ctx,
			&gatewayv1.FindBackendForQueryRequest{QueryId: nt.queryId},
		)
		if err != nil {
			provider.Logger(*ctx).WithError(err).
				Errorw("Backend Unresolvable for queryId extracted for Ui Request.",
					map[string]interface{}{"queryId": nt.queryId})
			return nil, err
		}
		bId := findBackendIdResp.GetBackendId()
		err = r.prepareReqForRouting(ctx, req, bId, nt)
		if err != nil {
			return nil, err
		}

		return nt, nil

	case *QueryRequest:
		if nt.Query.GetId() != "" {
			findBackendIdResp, err := r.gatewayApiClient.Query.FindBackendForQuery(
				*ctx,
				&gatewayv1.FindBackendForQueryRequest{QueryId: nt.Query.GetId()},
			)
			if err != nil {
				provider.Logger(*ctx).WithError(err).
					Errorw("Backend Unresolvable for query metadata procedure. Ignoring extracted query_id",
						map[string]interface{}{"queryId": nt.Query.GetId()})
			}
			nt.Query.BackendId = findBackendIdResp.GetBackendId()
			nt.Query.GroupId = findBackendIdResp.GetGroupId()
			err = r.prepareReqForRouting(ctx, req, nt.Query.GetBackendId(), nt)
			if err != nil {
				return nil, err
			}

			return nt, nil
		}

		provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "invoking routing backend evaluation"))

		bId, gId, err := r.evaluateRoutingBackend(ctx, *nt)
		r.prepareReqForRouting(ctx, req, bId, nt)
		if err != nil {
			return nil, err
		}
		nt.Query.GroupId = gId
		nt.Query.BackendId = bId

		return nt, nil
	case *QueryApiRequest:
		findBackendIdResp, err := r.gatewayApiClient.Query.FindBackendForQuery(
			*ctx,
			&gatewayv1.FindBackendForQueryRequest{QueryId: nt.Query.GetId()},
		)
		if err != nil {
			provider.Logger(*ctx).WithError(err).
				Errorw("Backend Unresolvable for query.",
					map[string]interface{}{"queryId": nt.Query.GetId()})
		}
		provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "preparing payload for query Api request"))
		nt.Query.BackendId = findBackendIdResp.GetBackendId()
		nt.Query.GroupId = findBackendIdResp.GetGroupId()
		err = r.prepareReqForRouting(ctx, req, nt.Query.GetBackendId(), nt)
		if err != nil {
			provider.Logger(*ctx).WithError(err).
				Error("Error generating routing payload")
			return nil, err
		}
		return nt, nil

	default:
		return nil, fmt.Errorf("unexpected type %T", nt)
	}
}

func (r *RouterServer) evaluateRoutingBackend(ctx *context.Context, clientReq QueryRequest) (backendId string, groupId string, err error) {
	evalGrpReq := &gatewayv1.EvaluateGroupsRequest{
		IncomingPort:               clientReq.incomingPort,
		Host:                       clientReq.clientHost,
		HeaderConnectionProperties: clientReq.headerConnectionProperties,
		HeaderClientTags:           clientReq.headerClientTags,
	}
	provider.Logger(*ctx).Debug(fmt.Sprint(LOG_TAG, "evaluating groups for client"))

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

// modifies http.req for preparing it for routing
func (r *RouterServer) prepareReqForRouting(ctx *context.Context, req *http.Request, backend_id string, cReq ClientRequest) error {
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

	var host, scheme string
	// TODO: clean it
	switch cr := cReq.(type) {
	case *UiRequest:
		host = backend.GetExternalUrl()
		// TODO: track external url scheme separately
		scheme = backend.GetScheme().Enum().String()
	case *QueryApiRequest:
		host = backend.GetHostname()
		scheme = backend.GetScheme().Enum().String()
		cr.Query.ServerHost = fmt.
			Sprintf("%s://%s", backend.GetScheme().Enum().String(), backend.GetExternalUrl())
	case *QueryRequest:
		host = backend.GetHostname()
		scheme = backend.GetScheme().Enum().String()
		cr.Query.ServerHost = fmt.
			Sprintf("%s://%s", backend.GetScheme().Enum().String(), backend.GetExternalUrl())
	default:
		return fmt.Errorf("unexpected type %T", cr)
	}

	req.URL.Host = host
	req.URL.Scheme = scheme
	req.Host = host
	sourceHeader, err := r.gatewayApiClient.Policy.EvaluateRequestSourceForClient(*ctx, &gatewayv1.EvaluateRequestSourceRequest{
		IncomingPort: int32(r.port),
	})
	if err != nil {
		return err
	}
	if s := sourceHeader.GetSetRequestSource(); s != "" {
		req.Header.Set("X-Trino-Source", s)
	}
	// TODO - validate and refine parsing of X-Forwarded headers
	req.Header.Set("X-Forwarded-Host", host)
	provider.Logger(*ctx).Infow(
		fmt.Sprint(LOG_TAG, "Request modified, ready to be forwarded"),
		map[string]interface{}{
			"request": utils.StringifyHttpRequestOrResponse(ctx, req),
		})

	return nil
}
