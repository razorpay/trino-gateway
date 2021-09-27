package repo

import (
	"context"
	"errors"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
)

type IGroupRepo interface {
	Create(ctx context.Context, group *models.Group) error
	Update(ctx context.Context, group *models.Group) error
	Find(ctx context.Context, id string) (*models.Group, error)
	FindMany(ctx context.Context, conditions map[string]interface{}) (*[]models.Group, error)
	// GetAll(ctx context.Context) (*[]models.Group, error)
	// GetAllActive(ctx context.Context) (*[]models.Group, error)
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string) error
	Disable(ctx context.Context, id string) error
}

type GroupRepo struct {
	repo dbRepo.IDbRepo
}

// NewCore returns a new instance of *Core
func NewGroupRepo(ctx context.Context, repo dbRepo.IDbRepo) *GroupRepo {
	_ = ctx
	return &GroupRepo{repo: repo}
}

func (c *GroupRepo) Dummy(ctx context.Context) error { return nil }

func (r *GroupRepo) Create(ctx context.Context, group *models.Group) error {
	err := r.repo.Create(ctx, group)
	if err != nil {
		provider.Logger(ctx).WithError(err).Errorw("group create failed", map[string]interface{}{"group_id": group.ID})
		return err
	}

	provider.Logger(ctx).Infow("group created", map[string]interface{}{"group_id": group.ID})

	return nil
}

func (r *GroupRepo) Update(ctx context.Context, group *models.Group) error {
	err := r.repo.Update(ctx, group)
	if err != nil {
		provider.Logger(ctx).WithError(err).Errorw("group update failed", map[string]interface{}{"group_id": group.ID})
		return err
	}

	provider.Logger(ctx).Infow("group updated", map[string]interface{}{"group_id": group.ID})

	return nil
}

func (r *GroupRepo) Find(ctx context.Context, id string) (*models.Group, error) {
	group := models.Group{}

	err := r.repo.FindByID(ctx, &group, id)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (r *GroupRepo) FindMany(ctx context.Context, conditions map[string]interface{}) (*[]models.Group, error) {
	var groups []models.Group

	err := r.repo.FindMany(ctx, &groups, conditions)
	if err != nil {
		return nil, err
	}

	return &groups, nil
}

func (r *GroupRepo) Enable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("group activation triggered", map[string]interface{}{"group_id": id})

	group, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("group activation failed: " + err.Error())
		return err
	}

	if *group.IsEnabled {
		provider.Logger(ctx).Error("group activation failed. Already active")
		return errors.New("Already active")
	}

	*group.IsEnabled = true

	if err := r.repo.Update(ctx, group); err != nil {
		return err
	}

	return nil
}

func (r *GroupRepo) Disable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("group activation triggered", map[string]interface{}{"group_id": id})

	group, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("group activation failed: " + err.Error())
		return err
	}

	if !*group.IsEnabled {
		provider.Logger(ctx).Error("group activation failed. Already active")
		return errors.New("Already active")
	}

	*group.IsEnabled = false

	if err := r.repo.Update(ctx, group); err != nil {
		return err
	}

	return nil
}

func (r *GroupRepo) Delete(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("group delete request", map[string]interface{}{"group_id": id})

	group, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("group delete failed: " + err.Error())
		return err
	}

	// _ = group

	err = r.repo.Delete(ctx, group)
	if err != nil {
		return err
	}

	return nil
}
