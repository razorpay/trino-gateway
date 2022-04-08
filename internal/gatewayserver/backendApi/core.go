package backendapi

import (
	"context"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
)

type Core struct {
	backendRepo repo.IBackendRepo
}

type ICore interface {
	CreateOrUpdateBackend(ctx context.Context, params *BackendCreateParams) error
	GetBackend(ctx context.Context, id string) (*models.Backend, error)
	GetAllBackends(ctx context.Context) ([]models.Backend, error)
	GetAllActiveBackends(ctx context.Context) ([]models.Backend, error)
	UpdateBackend(ctx context.Context, b *models.Backend) error
	DeleteBackend(ctx context.Context, id string) error
	EnableBackend(ctx context.Context, id string) error
	DisableBackend(ctx context.Context, id string) error
	MarkHealthyBackend(ctx context.Context, id string) error
	MarkUnhealthyBackend(ctx context.Context, id string) error
}

func NewCore(backend repo.IBackendRepo) *Core {
	return &Core{backendRepo: backend}
}

// CreateParams has attributes that are required for backend.Create()
type BackendCreateParams struct {
	ID                   string
	Hostname             string
	Scheme               string
	ExternalUrl          string
	IsEnabled            bool
	IsHealthy            bool
	UptimeSchedule       string
	ClusterLoad          int32
	ThresholdClusterLoad int32
	StatsUpdatedAt       int64
}

func (c *Core) CreateOrUpdateBackend(ctx context.Context, params *BackendCreateParams) error {
	backend := models.Backend{
		Hostname:             params.Hostname,
		Scheme:               params.Scheme,
		ExternalUrl:          &params.ExternalUrl,
		IsEnabled:            &params.IsEnabled,
		IsHealthy:            &params.IsHealthy,
		UptimeSchedule:       &params.UptimeSchedule,
		ClusterLoad:          &params.ClusterLoad,
		ThresholdClusterLoad: &params.ThresholdClusterLoad,
		StatsUpdatedAt:       &params.StatsUpdatedAt,
	}
	backend.ID = params.ID

	_, exists := c.backendRepo.Find(ctx, params.ID)
	if exists == nil { // update
		return c.backendRepo.Update(ctx, &backend)
	} else { // create
		return c.backendRepo.Create(ctx, &backend)
	}
}

func (c *Core) GetBackend(ctx context.Context, id string) (*models.Backend, error) {
	backend, err := c.backendRepo.Find(ctx, id)
	return backend, err
}

func (c *Core) UpdateBackend(ctx context.Context, b *models.Backend) error {
	_, exists := c.backendRepo.Find(ctx, b.ID)
	if exists != nil {
		return exists
	}
	return c.backendRepo.Update(ctx, b)
}

func (c *Core) GetAllBackends(ctx context.Context) ([]models.Backend, error) {
	backends, err := c.backendRepo.FindMany(ctx, make(map[string]interface{}))
	return backends, err
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
	// Count int32
	// Skip  int32
	// From  int32
	// To    int32

	// custom
	IsEnabled bool `json:"is_enabled"`
}

func (p *FindManyParams) GetIsEnabled() bool {
	return p.IsEnabled
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) ([]models.Backend, error) {
	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.backendRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActiveBackends(ctx context.Context) ([]models.Backend, error) {
	backends, err := c.FindMany(ctx, &FindManyParams{IsEnabled: true})
	return backends, err
}

func (c *Core) DeleteBackend(ctx context.Context, id string) error {
	return c.backendRepo.Delete(ctx, id)
}

func (c *Core) EnableBackend(ctx context.Context, id string) error {
	return c.backendRepo.Enable(ctx, id)
}

func (c *Core) DisableBackend(ctx context.Context, id string) error {
	return c.backendRepo.Disable(ctx, id)
}

func (c *Core) MarkHealthyBackend(ctx context.Context, id string) error {
	return c.backendRepo.MarkHealthy(ctx, id)
}

func (c *Core) MarkUnhealthyBackend(ctx context.Context, id string) error {
	return c.backendRepo.MarkUnhealthy(ctx, id)
}

type EvaluateClientParams struct {
	ListeningPort int32
}
