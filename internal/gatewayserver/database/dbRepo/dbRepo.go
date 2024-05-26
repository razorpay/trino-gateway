package dbRepo

import (
	"context"

	"github.com/razorpay/trino-gateway/pkg/spine"
	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

type DbRepo struct {
	spine.Repo
}

type IDbRepo interface {
	Create(ctx context.Context, receiver spine.IModel) error
	FindByID(ctx context.Context, receiver spine.IModel, id string) error
	FindWithConditionByIDs(ctx context.Context, receivers interface{}, condition map[string]interface{}, ids []string) error
	FindMany(ctx context.Context, receivers interface{}, condition map[string]interface{}) error
	Delete(ctx context.Context, receiver spine.IModel) error
	DeleteMany(ctx context.Context, receivers interface{}, conditionStatements map[string]interface{}) error
	Update(ctx context.Context, receiver spine.IModel, attrList ...string) error
	Preload(ctx context.Context, query string, args ...interface{}) *spine.Repo
	ClearAssociations(ctx context.Context, receiver spine.IModel, name string) error
	ReplaceAssociations(ctx context.Context, receiver spine.IModel, name string, ass interface{}) error
}

func NewDbRepo(db *db.DB) IDbRepo {
	return &DbRepo{
		Repo: spine.Repo{
			Db: db,
		},
	}
}
