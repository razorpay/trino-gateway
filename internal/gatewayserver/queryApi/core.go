package queryapi

import (
	"context"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
)

type Core struct {
	queryRepo repo.IQueryRepo
}

type ICore interface {
	CreateOrUpdateQuery(ctx context.Context, params *QueryCreateParams) error
	GetQuery(ctx context.Context, id string) (*models.Query, error)
	FindMany(ctx context.Context, params IFindManyParams) (*[]models.Query, error)

	FindBackendForQuery(ctx context.Context, q string) (string, error)
}

func NewCore(ctx *context.Context, query repo.IQueryRepo) *Core {
	_ = ctx
	return &Core{queryRepo: query}
}

// CreateParams has attributes that are required for query.Create()
type QueryCreateParams struct {
	ID          string
	Text        string
	ClientIp    string
	BackendId   string
	Username    string
	GroupId     string
	ReceivedAt  int64
	SubmittedAt int64
}

func (c *Core) CreateOrUpdateQuery(ctx context.Context, params *QueryCreateParams) error {
	query := models.Query{
		Text:        params.Text,
		ClientIp:    params.ClientIp,
		BackendId:   params.BackendId,
		Username:    params.Username,
		GroupId:     params.GroupId,
		ReceivedAt:  params.ReceivedAt,
		SubmittedAt: params.SubmittedAt,
	}
	query.ID = params.ID
	_, exists := c.queryRepo.Find(ctx, params.ID)
	if exists == nil { // update
		return c.queryRepo.Update(ctx, &query)
	} else { // create
		return c.queryRepo.Create(ctx, &query)
	}
}

func (c *Core) GetQuery(ctx context.Context, id string) (*models.Query, error) {
	query, err := c.queryRepo.Find(ctx, id)
	return query, err
}

type IFindManyParams interface {
	GetCount() int32
	GetSkip() int32
	GetFrom() int32
	GetTo() int32
	GetOrderBy() string

	// custom
	GetUsername() string
	GetBackendId() string
	GetGroup() string
	GetSuccessfulSubmission() bool
}

type FindManyParams struct {
	// pagination
	Count   int32
	Skip    int32
	From    int32
	To      int32
	OrderBy string

	// custom
	Username             string
	BackendId            string
	Group                string
	SuccessfulSubmission bool
}

func (p *FindManyParams) GetUsername() string {
	return p.Username
}
func (p *FindManyParams) GetBackendId() string {
	return p.BackendId
}
func (p *FindManyParams) GetGroup() string {
	return p.Group
}
func (p *FindManyParams) GetSuccessfulSubmission() bool {
	return p.SuccessfulSubmission
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) (*[]models.Query, error) {

	conditionStr := structs.New(params)
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	return c.queryRepo.FindMany(ctx, conditions)
}

func (c *Core) FindBackendForQuery(ctx context.Context, q string) (string, error) {
	return "", nil
}
