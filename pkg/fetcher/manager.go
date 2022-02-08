package fetcher

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"gorm.io/gorm"
)

// Client for fetcher package.
type client struct {
	db       *gorm.DB
	entities entityList
}

// IClient exposed methods of fetcher package.
type IClient interface {
	Register(name string, entity IModel, entities interface{}) error
	IsEntityRegistered(name string) bool
	GetEntityList() []string
	Fetch(ctx context.Context, req IFetchRequest) (IFetchResponse, error)
	FetchMultiple(ctx context.Context, req IFetchMultipleRequest) (IFetchMultipleResponse, error)
}

// The entities to be fetched should implement IModel.
type IModel interface {
	TableName() string
}

type entityType struct {
	model  IModel
	models interface{}
}

type entityList struct {
	sync.Mutex
	list map[string]entityType
}

// New : returns a new fetcher client.
func New(db *gorm.DB) IClient {
	return &client{
		db: db,
		entities: entityList{
			list: make(map[string]entityType),
		},
	}
}

// Register : register the entity with the fetcher.
// An entity can be registered only once. Only the
// entities registered can be fetched using fetcher client.
func (c *client) Register(name string, model IModel, models interface{}) error {
	c.entities.Lock()
	defer c.entities.Unlock()

	if _, ok := c.entities.list[name]; ok {
		return fmt.Errorf("entity %s name already registered", name)
	}
	c.entities.list[name] = entityType{
		model:  model,
		models: models,
	}
	return nil
}

func (c *client) IsEntityRegistered(name string) bool {
	if _, ok := c.entities.list[name]; ok {
		return true
	}
	return false
}

// GetEntityList : returns the list of entities registered with fetcher package.
func (c *client) GetEntityList() []string {
	list := make([]string, 0, len(c.entities.list))

	for k := range c.entities.list {
		list = append(list, k)
	}

	return list
}

// FetchMultiple : returns multiple record of the entities based on the fetch multiple request parameters.
func (c *client) FetchMultiple(_ context.Context, req IFetchMultipleRequest) (IFetchMultipleResponse, error) {
	var (
		dataTypes entityType
		ok        bool
	)

	if dataTypes, ok = c.entities.list[req.GetEntityName()]; !ok {
		return nil, fmt.Errorf("entity %s not registered with fetcher", req.GetEntityName())
	}

	models := clone(dataTypes.models)
	var query *gorm.DB
	if req.ContainsCreatedAt() {
		query = c.db.
			Order("created_at DESC").
			Limit(req.GetPagination().GetLimit()).
			Offset(req.GetPagination().GetOffset())

		if req.GetTimeRange().GetFrom() != 0 && req.GetTimeRange().GetTo() != 0 {
			query = query.Where("created_at between ? and ?", req.GetTimeRange().GetFrom(), req.GetTimeRange().GetTo())
		}
	} else {
		query = c.db.
			Limit(req.GetPagination().GetLimit()).
			Offset(req.GetPagination().GetOffset())
	}

	if req.GetTrashed() {
		query = query.Unscoped()
	}

	if len(req.GetFilter()) >= 1 {
		query = query.Where(req.GetFilter())
	}
	query = query.Find(models)
	return FetchMultipleResponse{
		entities: map[string]interface{}{
			req.GetEntityName(): models,
		},
	}, query.Error
}

// Fetch : returns single entity from the using id.
func (c *client) Fetch(_ context.Context, req IFetchRequest) (IFetchResponse, error) {
	var (
		dataTypes entityType
		ok        bool
	)

	if dataTypes, ok = c.entities.list[req.GetEntityName()]; !ok {
		return nil, fmt.Errorf("entity %s not registered with fetcher", req.GetEntityName())
	}

	model := clone(dataTypes.model)
	query := c.db.Where("id = ?", req.GetID())

	if req.GetTrashed() {
		query = query.Unscoped()
	}

	query = query.First(model)
	return FetchResponse{
		entity: model,
	}, query.Error
}

func clone(src interface{}) interface{} {
	return reflect.
		New(reflect.TypeOf(src).Elem()).
		Interface()
}
