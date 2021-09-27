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

	provider.Logger(ctx).Infow("UpsertPolicyRequest", map[string]interface{}{
		"id":         req.GetId(),
		"rule_type":  req.GetRule().Type.Enum().String(),
		"rule_value": req.GetRule().Value,
		"group":      req.GetGroup(),
		"fallback":   req.GetFallbackGroup(),
		"is_enabled": req.GetIsEnabled(),
	})

	createParams := PolicyCreateParams{
		ID:            req.GetId(),
		RuleType:      req.GetRule().Type.Enum().String(),
		RuleValue:     req.GetRule().Value,
		Group:         req.GetGroup(),
		FallbackGroup: req.GetFallbackGroup(),
		IsEnabled:     req.GetIsEnabled(),
	}

	err := s.core.CreateOrUpdatePolicy(ctx, &createParams)
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Get retrieves a single policy record
func (s *Server) GetPolicy(ctx context.Context, req *gatewayv1.PolicyGetRequest) (*gatewayv1.PolicyGetResponse, error) {
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
	policies, err := s.core.GetAllPolicies(ctx)
	if err != nil {
		return nil, err
	}

	policiesProto := make([]*gatewayv1.Policy, 0, len(*policies))
	for _, policyModel := range *policies {
		policy, err := toPolicyResponseProto(&policyModel)
		if err != nil {
			return nil, err
		}
		policiesProto = append(policiesProto, policy)
	}

	response := gatewayv1.PolicyListAllResponse{
		Items: policiesProto,
	}

	return &response, nil
}

// Approve marks a policies status to approved

func (s *Server) EnablePolicy(ctx context.Context, req *gatewayv1.PolicyEnableRequest) (*gatewayv1.Empty, error) {
	err := s.core.EnablePolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func (s *Server) DisablePolicy(ctx context.Context, req *gatewayv1.PolicyDisableRequest) (*gatewayv1.Empty, error) {
	err := s.core.DisablePolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

// Delete deletes a policy, soft-delete
func (s *Server) DeletePolicy(ctx context.Context, req *gatewayv1.PolicyDeleteRequest) (*gatewayv1.Empty, error) {
	err := s.core.DeletePolicy(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &gatewayv1.Empty{}, nil
}

func toPolicyResponseProto(policy *models.Policy) (*gatewayv1.Policy, error) {
	rule_type, ok := gatewayv1.Policy_Rule_RuleType_value[policy.RuleType]
	if !ok {
		return nil, errors.New(fmt.Sprint("error encoding response: invalid rule_type ", policy.RuleType))
	}
	rule := gatewayv1.Policy_Rule{
		Type:  *gatewayv1.Policy_Rule_RuleType(rule_type).Enum(),
		Value: policy.RuleValue,
	}
	response := gatewayv1.Policy{
		Id:            policy.ID,
		Rule:          &rule,
		Group:         policy.GroupId,
		FallbackGroup: policy.FallbackGroupId,
		IsEnabled:     policy.IsEnabled,
	}

	return &response, nil
}

func (s *Server) EvaluateGroupForClient(ctx context.Context, req *gatewayv1.EvaluateGroupRequest) (*gatewayv1.Group, error) {
	return nil, nil
}
