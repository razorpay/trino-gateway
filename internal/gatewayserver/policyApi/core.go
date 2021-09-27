package policyapi

import (
	"context"
	"fmt"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
)

type Core struct {
	policyRepo repo.IPolicyRepo
}

type ICore interface {
	CreateOrUpdatePolicy(ctx context.Context, params *PolicyCreateParams) error
	GetPolicy(ctx context.Context, id string) (*models.Policy, error)
	GetAllPolicies(ctx context.Context) (*[]models.Policy, error)
	GetAllActivePolicies(ctx context.Context) (*[]models.Policy, error)
	DeletePolicy(ctx context.Context, id string) error
	EnablePolicy(ctx context.Context, id string) error
	DisablePolicy(ctx context.Context, id string) error

	EvaluateRoutingGroup(ctx context.Context, c *EvaluateClientParams) (string, error)
	EvaluatePolicy(ctx context.Context, group string) (string, error)
	FindPolicyForQuery(ctx context.Context, q string) (string, error)
}

func NewCore(ctx *context.Context, policy repo.IPolicyRepo) *Core {
	_ = ctx
	return &Core{policyRepo: policy}
}

// CreateParams has attributes that are required for policy.Create()
type PolicyCreateParams struct {
	ID            string
	RuleType      string
	RuleValue     string
	Group         string
	FallbackGroup string
	IsEnabled     bool
}

func (c *Core) CreateOrUpdatePolicy(ctx context.Context, params *PolicyCreateParams) error {
	policy := models.Policy{
		RuleType:        params.RuleType,
		RuleValue:       params.RuleValue,
		GroupId:         params.Group,
		FallbackGroupId: &params.FallbackGroup,
		IsEnabled:       &params.IsEnabled,
	}
	policy.ID = params.ID

	fmt.Println(policy.IsEnabled)
	_, exists := c.policyRepo.Find(ctx, params.ID)
	if exists == nil { // update
		return c.policyRepo.Update(ctx, &policy)
	} else { // create
		return c.policyRepo.Create(ctx, &policy)
	}
}

func (c *Core) GetPolicy(ctx context.Context, id string) (*models.Policy, error) {
	policy, err := c.policyRepo.Find(ctx, id)
	return policy, err
}

func (c *Core) GetAllPolicies(ctx context.Context) (*[]models.Policy, error) {
	policies, err := c.policyRepo.FindMany(ctx, make(map[string]interface{}))
	return policies, err
}

type IFindManyParams interface {
	// GetCount() int32
	// GetSkip() int32
	// GetFrom() int32
	// GetTo() int32

	// custom
	GetIsEnabled() bool
}

type FindManyParams struct {
	// pagination
	Count int32
	Skip  int32
	From  int32
	To    int32

	// custom
	IsEnabled bool
}

func (p *FindManyParams) GetIsEnabled() bool {
	return p.IsEnabled
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) (*[]models.Policy, error) {

	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.policyRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActivePolicies(ctx context.Context) (*[]models.Policy, error) {
	policies, err := c.FindMany(ctx, &FindManyParams{IsEnabled: true})
	return policies, err
}

func (c *Core) DeletePolicy(ctx context.Context, id string) error {
	return c.policyRepo.Delete(ctx, id)
}

func (c *Core) EnablePolicy(ctx context.Context, id string) error {
	return c.policyRepo.Enable(ctx, id)
}

func (c *Core) DisablePolicy(ctx context.Context, id string) error {
	return c.policyRepo.Disable(ctx, id)
}

type EvaluateClientParams struct {
	ListeningPort int32
}

func (c *Core) EvaluateRoutingGroup(ctx context.Context, params *EvaluateClientParams) (string, error) {
	return "", nil
}
func (c *Core) EvaluatePolicy(ctx context.Context, group string) (string, error) {
	return "", nil
}
func (c *Core) FindPolicyForQuery(ctx context.Context, q string) (string, error) {
	return "", nil
}
