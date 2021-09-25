package repo

import (
	"context"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
)

type IQueryRepo interface {
	Create(ctx context.Context, query *models.Query) error
	Update(ctx context.Context, query *models.Query) error
	Find(ctx context.Context, id string) (*models.Query, error)
	FindMany(ctx context.Context, conditions map[string]interface{}) (*[]models.Query, error)
	// Find(ctx context.Context, id string) (*Query, error)
	// FindAll(ctx context.Context) (*[]Query, error)
}

type QueryRepo struct {
	repo dbRepo.IDbRepo
}

// NewCore returns a new instance of *Core
func NewQueryRepo(ctx context.Context, repo dbRepo.IDbRepo) *QueryRepo {
	_ = ctx
	return &QueryRepo{repo: repo}
}

func (r *QueryRepo) Create(ctx context.Context, query *models.Query) error {
	err := r.repo.Create(ctx, query)
	if err != nil {
		boot.Logger(ctx).WithError(err).Errorw("user create failed", map[string]interface{}{"user_id": query.ID})
		return err
	}

	boot.Logger(ctx).Infow("query created", map[string]interface{}{"user_id": query.ID})

	return nil
}

func (r *QueryRepo) Update(ctx context.Context, query *models.Query) error {
	err := r.repo.Update(ctx, query)
	if err != nil {
		boot.Logger(ctx).WithError(err).Errorw("user update failed", map[string]interface{}{"query_id": query.ID})
		return err
	}

	boot.Logger(ctx).Infow("query updated", map[string]interface{}{"query_id": query.ID})

	return nil
}

func (r *QueryRepo) Find(ctx context.Context, id string) (*models.Query, error) {
	query := models.Query{}

	err := r.repo.FindByID(ctx, &query, id)
	if err != nil {
		return nil, err
	}

	return &query, nil
}

func (r *QueryRepo) FindMany(ctx context.Context, conditions map[string]interface{}) (*[]models.Query, error) {
	var queries []models.Query

	err := r.repo.FindMany(ctx, &queries, conditions)
	if err != nil {
		return nil, err
	}

	return &queries, nil
}
