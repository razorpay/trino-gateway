package groupapi

import (
	"context"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
)

type Core struct {
	groupRepo repo.IGroupRepo
}

type ICore interface {
	CreateOrUpdateGroup(ctx context.Context, params *GroupCreateParams) error
	GetGroup(ctx context.Context, id string) (*models.Group, error)
	GetAllGroups(ctx context.Context) (*[]models.Group, error)
	GetAllActiveGroups(ctx context.Context) (*[]models.Group, error)
	DeleteGroup(ctx context.Context, id string) error
	EnableGroup(ctx context.Context, id string) error
	DisableGroup(ctx context.Context, id string) error

	EvaluateBackend(ctx context.Context, group string) (string, error)
}

func NewCore(ctx *context.Context, group repo.IGroupRepo) *Core {
	_ = ctx
	return &Core{groupRepo: group}
}

// CreateParams has attributes that are required for group.Create()
type GroupCreateParams struct {
	ID        string
	Strategy  string
	IsEnabled bool
	Backends  []string
}

func (c *Core) CreateOrUpdateGroup(ctx context.Context, params *GroupCreateParams) error {

	var backendMappings []models.GroupBackendsMapping
	for _, backend := range params.Backends {
		backendMappings = append(backendMappings, models.GroupBackendsMapping{
			GroupId:   params.ID,
			BackendId: backend,
		})
	}

	group := models.Group{
		Strategy:  &params.Strategy,
		IsEnabled: &params.IsEnabled,
	}
	group.ID = params.ID
	group.GroupBackendsMappings = &backendMappings
	_, exists := c.groupRepo.Find(ctx, params.ID)
	if exists == nil { // update
		return c.groupRepo.Update(ctx, &group)
	} else { // create
		return c.groupRepo.Create(ctx, &group)
	}
}

func (c *Core) GetGroup(ctx context.Context, id string) (*models.Group, error) {
	group, err := c.groupRepo.Find(ctx, id)
	return group, err
}

func (c *Core) GetAllGroups(ctx context.Context) (*[]models.Group, error) {
	groups, err := c.groupRepo.FindMany(ctx, make(map[string]interface{}))
	return groups, err
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

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) (*[]models.Group, error) {

	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.groupRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActiveGroups(ctx context.Context) (*[]models.Group, error) {
	groups, err := c.FindMany(ctx, &FindManyParams{IsEnabled: true})
	return groups, err
}

func (c *Core) DeleteGroup(ctx context.Context, id string) error {
	return c.groupRepo.Delete(ctx, id)
}

func (c *Core) EnableGroup(ctx context.Context, id string) error {
	return c.groupRepo.Enable(ctx, id)
}

func (c *Core) DisableGroup(ctx context.Context, id string) error {
	return c.groupRepo.Disable(ctx, id)
}

func (c *Core) EvaluateBackend(ctx context.Context, group string) (string, error) {
	return "", nil
}
