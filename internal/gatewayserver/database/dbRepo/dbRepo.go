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
	FindMany(ctx context.Context, receivers interface{}, condition map[string]interface{}) error
	Delete(ctx context.Context, receiver spine.IModel) error
	Update(ctx context.Context, receiver spine.IModel, attrList ...string) error
}

func NewDbRepo(ctx context.Context, db *db.DB) IDbRepo {
	_ = ctx

	return &DbRepo{
		Repo: spine.Repo{
			Db: db,
		},
	}
}
