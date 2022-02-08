package backendapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	_ "github.com/twitchtv/twirp"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
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

// Create creates a new backend
func (s *Server) CreateOrUpdateBackend(ctx context.Context, req *gatewayv1.Backend) (*gatewayv1.Empty, error) {
	// defer span.Finish()

	provider.Logger(ctx).Debugw("CreateOrUpdateBackend", map[string]interface{}{
		"request": req.String(),
	})

	createParams := BackendCreateParams{
		ID:                   req.GetId(),
		Scheme:               req.GetScheme().Enum().String(),
		Hostname:             req.GetHostname(),
		ExternalUrl:          req.GetExternalUrl(),
		IsEnabled:            req.GetIsEnabled(),
		IsHealthy:            req.GetIsHealthy(),
		UptimeSchedule:       req.GetUptimeSchedule(),
		ClusterLoad:          req.GetClusterLoad(),
		ThresholdClusterLoad: req.GetThresholdClusterLoad(),
		StatsUpdatedAt:       req.GetStatsUpdatedAt(),
	}

	err := s.core.CreateOrUpdateBackend(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Get retrieves a single backend record
func (s *Server) GetBackend(ctx context.Context, req *gatewayv1.BackendGetRequest) (*gatewayv1.BackendGetResponse, error) {
	provider.Logger(ctx).Debugw("GetBackend", map[string]interface{}{
		"request": req.String(),
	})

	backend, err := s.core.GetBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	backendProto, err := toBackendResponseProto(backend)
	if err != nil {
		return nil, err
	}
	return &gatewayv1.BackendGetResponse{Backend: backendProto}, nil
}

// List fetches a list of filtered backend records
func (s *Server) ListAllBackends(ctx context.Context, req *gatewayv1.Empty) (*gatewayv1.BackendListAllResponse, error) {
	provider.Logger(ctx).Debugw("ListAllBackends", map[string]interface{}{
		"request": req.String(),
	})
	backends, err := s.core.GetAllBackends(ctx)
	if err != nil {
		return nil, err
	}

	backendsProto := make([]*gatewayv1.Backend, len(backends))
	for i, backendModel := range backends {
		backend, err := toBackendResponseProto(&backendModel)
		if err != nil {
			return nil, err
		}
		backendsProto[i] = backend
	}

	response := gatewayv1.BackendListAllResponse{
		Items: backendsProto,
	}

	return &response, nil
}

// Approve marks a backends status to approved

func (s *Server) EnableBackend(ctx context.Context, req *gatewayv1.BackendEnableRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("EnableBackend", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.EnableBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) DisableBackend(ctx context.Context, req *gatewayv1.BackendDisableRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("DisableBackend", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.DisableBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) MarkHealthyBackend(ctx context.Context, req *gatewayv1.BackendMarkHealthyRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("MarkHealthyBackend", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.MarkHealthyBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) MarkUnhealthyBackend(ctx context.Context, req *gatewayv1.BackendMarkUnhealthyRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("MarkUnhealthyBackend", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.MarkUnhealthyBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) UpdateClusterLoadBackend(
	ctx context.Context,
	req *gatewayv1.BackendUpdateClusterLoadRequest,
) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("UpdateClusterLoadBackend", map[string]interface{}{
		"request": req.String(),
	})
	b, err := s.core.GetBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	*b.ClusterLoad = req.GetClusterLoad()
	*b.StatsUpdatedAt = time.Now().Unix()

	if err := s.core.UpdateBackend(ctx, b); err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Delete deletes a backend, soft-delete
func (s *Server) DeleteBackend(ctx context.Context, req *gatewayv1.BackendDeleteRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("DeleteBackend", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.DeleteBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func toBackendResponseProto(backend *models.Backend) (*gatewayv1.Backend, error) {
	if backend == nil {
		return &gatewayv1.Backend{}, nil
	}
	scheme, ok := gatewayv1.Backend_Scheme_value[backend.Scheme]
	if !ok {
		return nil, errors.New(fmt.Sprint("error encoding response: invalid scheme ", backend.Scheme))
	}
	response := gatewayv1.Backend{
		Id:                   backend.ID,
		Hostname:             backend.Hostname,
		Scheme:               *gatewayv1.Backend_Scheme(scheme).Enum(),
		ExternalUrl:          *backend.ExternalUrl,
		IsEnabled:            *backend.IsEnabled,
		UptimeSchedule:       *backend.UptimeSchedule,
		ClusterLoad:          *backend.ClusterLoad,
		ThresholdClusterLoad: *backend.ThresholdClusterLoad,
		StatsUpdatedAt:       *backend.StatsUpdatedAt,
		IsHealthy:            *backend.IsHealthy,
	}

	return &response, nil
}
