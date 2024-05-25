package policyapi

import (
	"context"
	"strconv"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
	"github.com/razorpay/trino-gateway/internal/provider"
)

type Core struct {
	policyRepo repo.IPolicyRepo
}

type ICore interface {
	CreateOrUpdatePolicy(ctx context.Context, params *PolicyCreateParams) error
	GetPolicy(ctx context.Context, id string) (*models.Policy, error)
	GetAllPolicies(ctx context.Context) ([]models.Policy, error)
	GetAllActivePolicies(ctx context.Context) ([]models.Policy, error)
	DeletePolicy(ctx context.Context, id string) error
	EnablePolicy(ctx context.Context, id string) error
	DisablePolicy(ctx context.Context, id string) error

	EvaluateGroupsForClient(ctx context.Context, c *EvaluateClientParams) ([]string, error)
	EvaluateAuthDelegation(ctx context.Context, p int32) (bool, error)
	// EvaluatePolicy(ctx context.Context, group string) (string, error)
	// FindPolicyForQuery(ctx context.Context, q string) (string, error)
}

func NewCore(policy repo.IPolicyRepo) *Core {
	return &Core{policyRepo: policy}
}

// CreateParams has attributes that are required for policy.Create()
type PolicyCreateParams struct {
	ID              string
	RuleType        string
	RuleValue       string
	Group           string
	FallbackGroup   string
	IsEnabled       bool
	IsAuthDelegated bool
}

func (c *Core) CreateOrUpdatePolicy(ctx context.Context, params *PolicyCreateParams) error {
	policy := models.Policy{
		RuleType:        params.RuleType,
		RuleValue:       params.RuleValue,
		GroupId:         params.Group,
		FallbackGroupId: &params.FallbackGroup,
		IsEnabled:       &params.IsEnabled,
		IsAuthDelegated: &params.IsAuthDelegated,
	}
	policy.ID = params.ID

	if policy.FallbackGroupId == nil {
		policy.FallbackGroupId = &boot.Config.Gateway.DefaultRoutingGroup
	}

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

func (c *Core) GetAllPolicies(ctx context.Context) ([]models.Policy, error) {
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
	GetRuleType() string
	GetRuleValue() string
}

type FindManyParams struct {
	// pagination
	// Count int32
	// Skip  int32
	// From  int32
	// To    int32

	// custom
	IsEnabled       bool   `json:"is_enabled"`
	RuleType        string `json:"rule_type"`
	RuleValue       string `json:"rule_value"`
	IsAuthDelegated bool   `json:"is_auth_delegated,omitempty"`
}

func (p *FindManyParams) GetIsEnabled() bool {
	return p.IsEnabled
}

func (p *FindManyParams) GetRuleType() string {
	return p.RuleType
}

func (p *FindManyParams) GetRuleValue() string {
	return p.RuleValue
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) ([]models.Policy, error) {
	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.policyRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActivePolicies(ctx context.Context) ([]models.Policy, error) {
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
	ListeningPort              int32
	Hostname                   string
	HeaderConnectionProperties string
	HeaderClientTags           string
}

func (c *Core) EvaluateGroupsForClient(ctx context.Context, params *EvaluateClientParams) ([]string, error) {
	// policies, err := c.GetAllActivePolicies(ctx)
	var err error

	// Using a map instead of slice for returning groups, to simulate a 'set' data type
	findGroupsForPolicyTypes := func(ruleType string, ruleValue string) (*map[string]struct{}, error) {
		provider.Logger(ctx).Debugw("Fetching policies for rule", map[string]interface{}{
			"ruleType":  ruleType,
			"ruleValue": ruleValue,
		})
		activePolicies, err := c.FindMany(
			ctx,
			&FindManyParams{
				IsEnabled: true,
				RuleType:  ruleType,
				RuleValue: ruleValue,
			})
		if err != nil {
			return nil, err
		}
		gids := make(map[string]struct{})
		for _, policy := range activePolicies {
			gids[policy.GroupId] = struct{}{}
		}
		provider.Logger(ctx).Debugw("Groups matching rule", map[string]interface{}{
			"ruleType":  ruleType,
			"ruleValue": ruleValue,
			"GIDs":      gids,
		})
		if len(gids) == 0 {
			gids = map[string]struct{}(nil)
		}
		return &gids, nil
	}

	// Step 1: find all policies
	listeningPortPolicies, err := findGroupsForPolicyTypes("listening_port", strconv.Itoa(int(params.ListeningPort)))
	if err != nil {
		return nil, err
	}

	hostnamePolicies, err := findGroupsForPolicyTypes("header_host", params.Hostname)
	if err != nil {
		return nil, err
	}

	clientTagsPolicies, err := findGroupsForPolicyTypes("header_client_tags", params.HeaderClientTags)
	if err != nil {
		return nil, err
	}

	clientConnPropsPolicies, err := findGroupsForPolicyTypes("header_connection_properties", params.HeaderConnectionProperties)
	if err != nil {
		return nil, err
	}

	// Step 2: take intersections of all non nil grp sets; a nil set = any grp; all sets nil == route to fallbackGrp;
	provider.Logger(ctx).Debug("Taking intersection of all eligible non-nil groups sets")
	gids := setIntersection(setIntersection(setIntersection(*listeningPortPolicies, *hostnamePolicies), *clientTagsPolicies), *clientConnPropsPolicies)

	res := make([]string, len(gids))
	i := 0
	for k := range gids {
		res[i] = k
		i++
	}
	return res, nil
}

func (c *Core) EvaluateAuthDelegation(ctx context.Context, port int32) (bool, error) {
	res, err := c.FindMany(
		ctx,
		&FindManyParams{
			IsEnabled:       true,
			RuleType:        "listening_port",
			RuleValue:       strconv.Itoa(int(port)),
			IsAuthDelegated: true,
		})
	if err != nil {
		return false, err
	}
	provider.Logger(ctx).Debugw("Is Auth Delegated For Port", map[string]interface{}{
		"listeningPort": port,
		"matchingRules": res,
	})
	if len(res) > 0 {
		return true, nil
	}
	return false, nil
}

// Implementing "set" collection methods here, :)
func setIntersection(s1 map[string]struct{}, s2 map[string]struct{}) map[string]struct{} {
	s_intersection := map[string]struct{}{}
	if len(s1) > len(s2) {
		s1, s2 = s2, s1 // better to iterate over a shorter set
	}
	if s1 == nil {
		return s2
	}
	for k, _ := range s1 {
		if _, found := s2[k]; found {
			s_intersection[k] = struct{}{}
		}
	}
	return s_intersection
}
