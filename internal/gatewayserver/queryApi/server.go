package queryapi

import (
	"context"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
	_ "github.com/twitchtv/twirp"
)

// Server has methods implementing of server rpc.
type Server struct {
	core ICore
}

// NewServer returns a server.
func NewServer(core ICore) *Server {
	return &Server{
		core: core,
	}
}

func (s *Server) CreateOrUpdateQuery(ctx context.Context, req *gatewayv1.Query) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("CreateOrUpdateQuery", map[string]interface{}{
		"request": req.String(),
	})

	createParams := QueryCreateParams{
		ID:          req.GetId(),
		Text:        req.GetText(),
		ReceivedAt:  req.GetReceivedAt(),
		ClientIp:    req.GetClientIp(),
		GroupId:     req.GetGroupId(),
		BackendId:   req.GetBackendId(),
		Username:    req.GetUsername(),
		SubmittedAt: req.GetSubmittedAt(),
	}

	err := s.core.CreateOrUpdateQuery(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) GetQuery(ctx context.Context, req *gatewayv1.QueryGetRequest) (*gatewayv1.QueryGetResponse, error) {
	provider.Logger(ctx).Debugw("GetQuery", map[string]interface{}{
		"request": req.String(),
	})
	query, err := s.core.GetQuery(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	queryProto, err := toQueryResponseProto(query)
	if err != nil {
		return nil, err
	}
	return &gatewayv1.QueryGetResponse{Query: queryProto}, nil
}

func (s *Server) ListQueries(ctx context.Context, req *gatewayv1.QueriesListRequest) (*gatewayv1.QueriesListResponse, error) {
	provider.Logger(ctx).Debugw("ListQueries", map[string]interface{}{
		"request": req.String(),
	})
	// TODO
	return nil, nil
}

func toQueryResponseProto(query *models.Query) (*gatewayv1.Query, error) {
	return &gatewayv1.Query{
		Id:          query.ID,
		Text:        query.Text,
		ReceivedAt:  query.ReceivedAt,
		ClientIp:    query.ClientIp,
		GroupId:     query.GroupId,
		BackendId:   query.BackendId,
		Username:    query.Username,
		SubmittedAt: query.SubmittedAt,
	}, nil
}

func (s *Server) FindBackendForQuery(ctx context.Context, req *gatewayv1.FindBackendForQueryRequest) (*gatewayv1.FindBackendForQueryResponse, error) {
	provider.Logger(ctx).Debugw("FindBackendForQuery", map[string]interface{}{
		"request": req.String(),
	})

	backend_id, err := s.core.FindBackendForQuery(ctx, req.QueryId)

	if err != nil {
		return &gatewayv1.FindBackendForQueryResponse{BackendId: backend_id}, nil
	}
	return nil, err
}
