package groupapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	provider.Logger(ctx).Debugw("CreateOrUpdateGroup", map[string]interface{}{
		"request": req.String(),
	})

	createParams := GroupCreateParams{
		ID:                req.GetId(),
		Strategy:          req.GetStrategy().Enum().String(),
		Backends:          req.GetBackends(),
		IsEnabled:         req.GetIsEnabled(),
		LastRoutedBackend: req.GetLastRoutedBackend(),
	}

	err := s.core.CreateOrUpdateGroup(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Get retrieves a single group record
func (s *Server) GetGroup(ctx context.Context, req *gatewayv1.GroupGetRequest) (*gatewayv1.GroupGetResponse, error) {
	provider.Logger(ctx).Debugw("GetGroup", map[string]interface{}{
		"request": req.String(),
	})

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
	provider.Logger(ctx).Debugw("ListAllGroups", map[string]interface{}{
		"request": req.String(),
	})
	groups, err := s.core.GetAllGroups(ctx)
	if err != nil {
		return nil, err
	}

	groupsProto := make([]*gatewayv1.Group, len(groups))
	for i, groupModel := range groups {
		group, err := toGroupResponseProto(&groupModel)
		if err != nil {
			return nil, err
		}
		groupsProto[i] = group
	}

	response := gatewayv1.GroupListAllResponse{
		Items: groupsProto,
	}

	return &response, nil
}

// Approve marks a groups status to approved

func (s *Server) EnableGroup(ctx context.Context, req *gatewayv1.GroupEnableRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("EnableGroup", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.EnableGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) DisableGroup(ctx context.Context, req *gatewayv1.GroupDisableRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("DisableGroup", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.DisableGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Delete deletes a group, soft-delete
func (s *Server) DeleteGroup(ctx context.Context, req *gatewayv1.GroupDeleteRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("DeleteGroup", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.DeleteGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func toGroupResponseProto(group *models.Group) (*gatewayv1.Group, error) {
	if group == nil {
		return &gatewayv1.Group{}, nil
	}
	strategy, ok := gatewayv1.Group_RoutingStrategy_value[strings.ToUpper(*group.Strategy)]
	if !ok {
		return nil, errors.New(fmt.Sprint("error encoding response: invalid strategy ", *group.Strategy))
	}
	var backends []string
	for _, backend := range group.GroupBackendsMappings {
		backends = append(backends, backend.BackendId)
	}
	response := gatewayv1.Group{
		Id:                group.ID,
		Strategy:          *gatewayv1.Group_RoutingStrategy(strategy).Enum(),
		Backends:          backends,
		IsEnabled:         *group.IsEnabled,
		LastRoutedBackend: *group.LastRoutedBackend,
	}

	return &response, nil
}

func (s *Server) EvaluateBackendForGroups(ctx context.Context, req *gatewayv1.EvaluateBackendRequest) (*gatewayv1.EvaluateBackendResponse, error) {
	provider.Logger(ctx).Debugw("EvaluateBackendForGroups", map[string]interface{}{
		"request": req.String(),
	})

	backend_id, group_id, err := s.core.EvaluateBackendForGroups(ctx, req.GetGroupIds())
	if err != nil {
		return nil, err

	}
	return &gatewayv1.EvaluateBackendResponse{BackendId: backend_id, GroupId: group_id}, nil
}
