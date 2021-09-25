package backendapi

import (
	"context"
	"fmt"

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
	GetAllBackends(ctx context.Context) (*[]models.Backend, error)
	GetAllActiveBackends(ctx context.Context) (*[]models.Backend, error)
	DeleteBackend(ctx context.Context, id string) error
	EnableBackend(ctx context.Context, id string) error
	DisableBackend(ctx context.Context, id string) error

	EvaluateRoutingGroup(ctx context.Context, c *EvaluateClientParams) (string, error)
	EvaluateBackend(ctx context.Context, group string) (string, error)
	FindBackendForQuery(ctx context.Context, q string) (string, error)
}

func NewCore(ctx *context.Context, backend repo.IBackendRepo) *Core {
	_ = ctx
	return &Core{backendRepo: backend}
}

// CreateParams has attributes that are required for backend.Create()
type BackendCreateParams struct {
	ID                      string
	Hostname                string
	Scheme                  string
	ExternalUrl             string
	IsEnabled               bool
	UptimeSchedule          string
	RunningQueries          int32
	QueuedQueries           int32
	ThresholdRunningQueries int32
	ThresholdQueuedQueries  int32
}

func (c *Core) CreateOrUpdateBackend(ctx context.Context, params *BackendCreateParams) error {
	backend := models.Backend{
		Hostname:                params.Hostname,
		Scheme:                  params.Scheme,
		ExternalUrl:             &params.ExternalUrl,
		IsEnabled:               &params.IsEnabled,
		UptimeSchedule:          &params.UptimeSchedule,
		RunningQueries:          &params.RunningQueries,
		QueuedQueries:           &params.QueuedQueries,
		ThresholdRunningQueries: &params.ThresholdRunningQueries,
		ThresholdQueuedQueries:  &params.ThresholdQueuedQueries,
	}
	backend.ID = params.ID

	fmt.Println(backend.IsEnabled)
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

func (c *Core) GetAllBackends(ctx context.Context) (*[]models.Backend, error) {
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

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) (*[]models.Backend, error) {

	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.backendRepo.FindMany(ctx, conditions)
}

func (c *Core) GetAllActiveBackends(ctx context.Context) (*[]models.Backend, error) {
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

type EvaluateClientParams struct {
	ListeningPort int32
}

func (c *Core) EvaluateRoutingGroup(ctx context.Context, params *EvaluateClientParams) (string, error) {
	return "", nil
}
func (c *Core) EvaluateBackend(ctx context.Context, group string) (string, error) {
	return "", nil
}
func (c *Core) FindBackendForQuery(ctx context.Context, q string) (string, error) {
	return "", nil
}
