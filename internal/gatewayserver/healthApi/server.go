package healthapi

import (
	"context"

	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
	"github.com/twitchtv/twirp"
)

// Server has methods implementing of server rpc.
type Server struct {
	core *Core
}

// NewServer returns a server.
func NewServer(core *Core) *Server {
	return &Server{
		core: core,
	}
}

// Check returns service's serving status.
func (s *Server) Check(ctx context.Context, req *gatewayv1.HealthCheckRequest) (*gatewayv1.HealthCheckResponse, error) {
	var status gatewayv1.HealthCheckResponse_ServingStatus
	ok, err := s.core.RunHealthCheck(ctx)
	if !ok {
		status = gatewayv1.HealthCheckResponse_SERVING_STATUS_NOT_SERVING
		return &gatewayv1.HealthCheckResponse{ServingStatus: status}, twirp.NewError(twirp.Unavailable, err.Error())
	}
	status = gatewayv1.HealthCheckResponse_SERVING_STATUS_SERVING
	return &gatewayv1.HealthCheckResponse{ServingStatus: status}, nil
}
