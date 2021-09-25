package backendapi

import (
	"context"
	"errors"
	"fmt"

	_ "github.com/twitchtv/twirp"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
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

	boot.Logger(ctx).Infow("UpsertBackendRequest", map[string]interface{}{
		"scheme":       req.GetScheme().Enum().String(),
		"hostname":     req.GetHostname(),
		"external_url": req.GetExternalUrl(),
		"is_enabled":   req.GetIsEnabled(),
	})

	createParams := BackendCreateParams{
		ID:             req.GetId(),
		Scheme:         req.GetScheme().Enum().String(),
		Hostname:       req.GetHostname(),
		ExternalUrl:    req.GetExternalUrl(),
		IsEnabled:      req.GetIsEnabled(),
		UptimeSchedule: req.GetUptimeSchedule(),
	}

	err := s.core.CreateOrUpdateBackend(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Get retrieves a single backend record
func (s *Server) GetBackend(ctx context.Context, req *gatewayv1.BackendGetRequest) (*gatewayv1.BackendGetResponse, error) {
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
	backends, err := s.core.GetAllBackends(ctx)
	if err != nil {
		return nil, err
	}

	backendsProto := make([]*gatewayv1.Backend, 0, len(*backends))
	for _, backendModel := range *backends {
		backend, err := toBackendResponseProto(&backendModel)
		if err != nil {
			return nil, err
		}
		backendsProto = append(backendsProto, backend)
	}

	response := gatewayv1.BackendListAllResponse{
		Items: backendsProto,
	}

	return &response, nil
}

// Approve marks a backends status to approved

func (s *Server) EnableBackend(ctx context.Context, req *gatewayv1.BackendEnableRequest) (*gatewayv1.Empty, error) {
	err := s.core.EnableBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) DisableBackend(ctx context.Context, req *gatewayv1.BackendDisableRequest) (*gatewayv1.Empty, error) {
	err := s.core.DisableBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Delete deletes a backend, soft-delete
func (s *Server) DeleteBackend(ctx context.Context, req *gatewayv1.BackendDeleteRequest) (*gatewayv1.Empty, error) {
	err := s.core.DeleteBackend(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func toBackendResponseProto(backend *models.Backend) (*gatewayv1.Backend, error) {
	scheme, ok := gatewayv1.Backend_Scheme_value[backend.Scheme]
	if !ok {
		return nil, errors.New(fmt.Sprint("error encoding response: invalid scheme ", backend.Scheme))
	}
	response := gatewayv1.Backend{
		Id:             backend.ID,
		Hostname:       backend.Hostname,
		Scheme:         *gatewayv1.Backend_Scheme(scheme).Enum(),
		ExternalUrl:    *backend.ExternalUrl,
		IsEnabled:      *backend.IsEnabled,
		UptimeSchedule: *backend.UptimeSchedule,
	}

	return &response, nil
}
