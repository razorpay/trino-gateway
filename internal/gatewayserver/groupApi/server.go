package groupapi

import (
	"context"
	"errors"
	"fmt"

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

// Create creates a new group
func (s *Server) CreateOrUpdateGroup(ctx context.Context, req *gatewayv1.Group) (*gatewayv1.Empty, error) {
	// defer span.Finish()

	provider.Logger(ctx).Infow("UpsertGroupRequest", map[string]interface{}{
		"id":         req.GetId(),
		"strategy":   req.GetStrategy().Enum().String(),
		"backends":   req.GetBackends(),
		"is_enabled": req.GetIsEnabled(),
	})

	createParams := GroupCreateParams{
		ID:        req.GetId(),
		Strategy:  req.GetStrategy().Enum().String(),
		Backends:  req.GetBackends(),
		IsEnabled: req.GetIsEnabled(),
	}

	err := s.core.CreateOrUpdateGroup(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Get retrieves a single group record
func (s *Server) GetGroup(ctx context.Context, req *gatewayv1.GroupGetRequest) (*gatewayv1.GroupGetResponse, error) {
	group, err := s.core.GetGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	groupProto, err := toGroupResponseProto(group)
	if err != nil {
		return nil, err
	}
	return &gatewayv1.GroupGetResponse{Group: groupProto}, nil
}

// List fetches a list of filtered group records
func (s *Server) ListAllGroups(ctx context.Context, req *gatewayv1.Empty) (*gatewayv1.GroupListAllResponse, error) {
	groups, err := s.core.GetAllGroups(ctx)
	if err != nil {
		return nil, err
	}

	groupsProto := make([]*gatewayv1.Group, 0, len(*groups))
	for _, groupModel := range *groups {
		group, err := toGroupResponseProto(&groupModel)
		if err != nil {
			return nil, err
		}
		groupsProto = append(groupsProto, group)
	}

	response := gatewayv1.GroupListAllResponse{
		Items: groupsProto,
	}

	return &response, nil
}

// Approve marks a groups status to approved

func (s *Server) EnableGroup(ctx context.Context, req *gatewayv1.GroupEnableRequest) (*gatewayv1.Empty, error) {
	err := s.core.EnableGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) DisableGroup(ctx context.Context, req *gatewayv1.GroupDisableRequest) (*gatewayv1.Empty, error) {
	err := s.core.DisableGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Delete deletes a group, soft-delete
func (s *Server) DeleteGroup(ctx context.Context, req *gatewayv1.GroupDeleteRequest) (*gatewayv1.Empty, error) {
	err := s.core.DeleteGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func toGroupResponseProto(group *models.Group) (*gatewayv1.Group, error) {
	strategy, ok := gatewayv1.Group_RoutingStrategy_value[*group.Strategy]
	if !ok {
		return nil, errors.New(fmt.Sprint("error encoding response: invalid strategy ", group.Strategy))
	}
	var backends []string
	for _, backend := range *group.GroupBackendsMappings {
		backends = append(backends, backend.BackendId)
	}
	response := gatewayv1.Group{
		Id:        group.ID,
		Strategy:  *gatewayv1.Group_RoutingStrategy(strategy).Enum(),
		Backends:  backends,
		IsEnabled: *group.IsEnabled,
	}

	return &response, nil
}

func (s *Server) EvaluateBackendForGroup(ctx context.Context, req *gatewayv1.EvaluateBackendRequest) (*gatewayv1.Backend, error) {
	return nil, nil
}
