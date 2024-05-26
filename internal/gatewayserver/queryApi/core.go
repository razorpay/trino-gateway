package queryapi

import (
	"context"
	"time"

	"github.com/fatih/structs"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/repo"
	"github.com/razorpay/trino-gateway/internal/provider"
	fetcherPkg "github.com/razorpay/trino-gateway/pkg/fetcher"
)

var entityName string = (&models.Query{}).EntityName()

type Core struct {
	queryRepo repo.IQueryRepo
	fetcher   fetcherPkg.IClient
}

type ICore interface {
	CreateOrUpdateQuery(ctx context.Context, params *QueryCreateParams) error
	GetQuery(ctx context.Context, id string) (*models.Query, error)
	FindMany(ctx context.Context, params IFindManyParams) ([]models.Query, error)
	PurgeOldQueries(ctx context.Context, maxAge time.Time) error
}

func NewCore(query repo.IQueryRepo, fetcher fetcherPkg.IClient) *Core {
	if !fetcher.IsEntityRegistered(entityName) {
		fetcher.Register(entityName, &models.Query{}, &[]models.Query{})
	}
	return &Core{
		queryRepo: query,
		fetcher:   fetcher,
	}
}

// CreateParams has attributes that are required for query.Create()
type QueryCreateParams struct {
	ID          string
	Text        string
	ClientIp    string
	BackendId   string
	Username    string
	GroupId     string
	ServerHost  string
	SubmittedAt int64
}

func (c *Core) CreateOrUpdateQuery(ctx context.Context, params *QueryCreateParams) error {
	query := models.Query{
		Text:        params.Text,
		ClientIp:    params.ClientIp,
		BackendId:   params.BackendId,
		Username:    params.Username,
		GroupId:     params.GroupId,
		ServerHost:  params.ServerHost,
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
	GetFrom() int64
	GetTo() int64
	// GetOrderBy() string

	// custom
	GetUsername() string
	GetBackendId() string
	GetGroupId() string
}

type Filters struct {
	// custom
	Username  string `json:"username,omitempty"`
	BackendId string `json:"backend_id,omitempty"`
	GroupId   string `json:"group_id,omitempty"`
}

func (c *Core) FindMany(ctx context.Context, params IFindManyParams) ([]models.Query, error) {
	conditionStr := structs.New(Filters{
		Username:  params.GetUsername(),
		BackendId: params.GetBackendId(),
		GroupId:   params.GetGroupId(),
	})
	// use the json tag name, so we can respect omitempty tags
	conditionStr.TagName = "json"
	conditions := conditionStr.Map()

	// return c.queryRepo.FindMany(ctx, conditions)

	pagination := fetcherPkg.Pagination{}

	pagination.Skip = int(params.GetSkip())
	pagination.Limit = int(params.GetCount())

	t := params
	timeRange := fetcherPkg.TimeRange{}
	if params.GetFrom() != int64(0) && params.GetTo() != int64(0) {
		timeRange.From = t.GetFrom()
		timeRange.To = t.GetTo()
	}

	fetchRequest := fetcherPkg.FetchMultipleRequest{
		EntityName:   entityName,
		Filter:       conditions,
		Pagination:   pagination,
		TimeRange:    timeRange,
		IsTrashed:    false,
		HasCreatedAt: true,
	}

	resp, err := c.fetcher.FetchMultiple(ctx, fetchRequest)
	if err != nil {
		return nil, err
	}

	queries := (resp.GetEntities().(map[string]interface{})[entityName]).(*[]models.Query)

	return *queries, nil
}

func (c *Core) PurgeOldQueries(ctx context.Context, maxAge time.Time) error {
	maxAgeUnix := maxAge.Unix()
	provider.Logger(ctx).Debugw("PurgeOldQueries", map[string]interface{}{
		"maxAgeUnix": maxAgeUnix,
	})

	conditions := map[string]interface{}{
		"created_at <= ?": maxAgeUnix,
	}

	var emptyQueryModel []models.Query
	err := c.queryRepo.DeleteMany(ctx, conditions, emptyQueryModel)
	return err
}
