package policyapi

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

// Create creates a new policy
func (s *Server) CreateOrUpdatePolicy(ctx context.Context, req *gatewayv1.Policy) (*gatewayv1.Empty, error) {
	// defer span.Finish()

	provider.Logger(ctx).Debugw("CreateOrUpdatePolicy", map[string]interface{}{
		"request": req.String(),
	})

	createParams := PolicyCreateParams{
		ID:              req.GetId(),
		RuleType:        req.GetRule().GetType().Enum().String(),
		RuleValue:       req.GetRule().GetValue(),
		Group:           req.GetGroup(),
		FallbackGroup:   req.GetFallbackGroup(),
		IsEnabled:       req.GetIsEnabled(),
		IsAuthDelegated: req.GetIsAuthDelegated(),
	}

	err := s.core.CreateOrUpdatePolicy(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Get retrieves a single policy record
func (s *Server) GetPolicy(ctx context.Context, req *gatewayv1.PolicyGetRequest) (*gatewayv1.PolicyGetResponse, error) {
	provider.Logger(ctx).Debugw("GetPolicy", map[string]interface{}{
		"request": req.String(),
	})
	policy, err := s.core.GetPolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	policyProto, err := toPolicyResponseProto(policy)
	if err != nil {
		return nil, err
	}
	return &gatewayv1.PolicyGetResponse{Policy: policyProto}, nil
}

// List fetches a list of filtered policy records
func (s *Server) ListAllPolicies(ctx context.Context, req *gatewayv1.Empty) (*gatewayv1.PolicyListAllResponse, error) {
	provider.Logger(ctx).Debugw("ListAllPolicies", map[string]interface{}{
		"request": req.String(),
	})
	policies, err := s.core.GetAllPolicies(ctx)
	if err != nil {
		return nil, err
	}

	policiesProto := make([]*gatewayv1.Policy, len(policies))
	for i, policyModel := range policies {
		policy, err := toPolicyResponseProto(&policyModel)
		if err != nil {
			return nil, err
		}
		policiesProto[i] = policy
	}

	response := gatewayv1.PolicyListAllResponse{
		Items: policiesProto,
	}

	return &response, nil
}

// Approve marks a policies status to approved

func (s *Server) EnablePolicy(ctx context.Context, req *gatewayv1.PolicyEnableRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("EnablePolicy", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.EnablePolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) DisablePolicy(ctx context.Context, req *gatewayv1.PolicyDisableRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("DisablePolicy", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.DisablePolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Delete deletes a policy, soft-delete
func (s *Server) DeletePolicy(ctx context.Context, req *gatewayv1.PolicyDeleteRequest) (*gatewayv1.Empty, error) {
	provider.Logger(ctx).Debugw("DeletePolicy", map[string]interface{}{
		"request": req.String(),
	})
	err := s.core.DeletePolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func toPolicyResponseProto(policy *models.Policy) (*gatewayv1.Policy, error) {
	if policy == nil {
		return &gatewayv1.Policy{}, nil
	}
	rule_type, ok := gatewayv1.Policy_Rule_RuleType_value[policy.RuleType]
	if !ok {
		return nil, errors.New(fmt.Sprint("error encoding response: invalid rule_type ", policy.RuleType))
	}
	rule := gatewayv1.Policy_Rule{
		Type:  *gatewayv1.Policy_Rule_RuleType(rule_type).Enum(),
		Value: policy.RuleValue,
	}
	response := gatewayv1.Policy{
		Id:              policy.ID,
		Rule:            &rule,
		Group:           policy.GroupId,
		FallbackGroup:   *policy.FallbackGroupId,
		IsEnabled:       *policy.IsEnabled,
		IsAuthDelegated: *policy.IsAuthDelegated,
	}

	return &response, nil
}

func (s *Server) EvaluateGroupsForClient(ctx context.Context, req *gatewayv1.EvaluateGroupsRequest) (*gatewayv1.EvaluateGroupsResponse, error) {
	provider.Logger(ctx).Debugw("EvaluateGroupsForClient", map[string]interface{}{
		"request": req.String(),
	})

	gids, err := s.core.EvaluateGroupsForClient(
		ctx,
		&EvaluateClientParams{
			ListeningPort:              req.GetIncomingPort(),
			Hostname:                   req.GetHost(),
			HeaderConnectionProperties: req.GetHeaderConnectionProperties(),
			HeaderClientTags:           req.GetHeaderClientTags(),
		})

	if err != nil {
		return nil, err

	}
	if gids != nil {
		return &gatewayv1.EvaluateGroupsResponse{GroupIds: gids}, nil
	}
	return &gatewayv1.EvaluateGroupsResponse{}, nil
}

func (s *Server) EvaluateAuthDelegationForClient(ctx context.Context, req *gatewayv1.EvaluateAuthDelegationRequest) (*gatewayv1.EvaluateAuthDelegationResponse, error) {
	provider.Logger(ctx).Debugw("EvaluateAuthDelegation", map[string]interface{}{
		"request": req.String(),
	})

	if req.GetIncomingPort() == 0 {
		err := errors.New("Invalid port defined in `incoming_port`.")
		provider.Logger(ctx).WithError(err).Error(err.Error())
		return &gatewayv1.EvaluateAuthDelegationResponse{IsAuthDelegated: false}, nil
	}

	result, err := s.core.EvaluateAuthDelegation(
		ctx,
		req.GetIncomingPort(),
	)
	if err != nil {
		return nil, err
	}
	return &gatewayv1.EvaluateAuthDelegationResponse{IsAuthDelegated: result}, nil
}
