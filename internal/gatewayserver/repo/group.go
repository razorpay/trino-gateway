package repo

import (
	"context"
	"errors"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/pkg/spine"
	"gorm.io/gorm/clause"
)

type IGroupRepo interface {
	Create(ctx context.Context, group *models.Group) error
	Update(ctx context.Context, group *models.Group) error
	Find(ctx context.Context, id string) (*models.Group, error)
	FindMany(ctx context.Context, conditions map[string]interface{}) ([]models.Group, error)
	// GetAll(ctx context.Context) ([]models.Group, error)
	// GetAllActive(ctx context.Context) ([]models.Group, error)
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string) error
	Disable(ctx context.Context, id string) error
}

type GroupRepo struct {
	repo dbRepo.IDbRepo
}

// NewCore returns a new instance of *Core
func NewGroupRepo(repo dbRepo.IDbRepo) *GroupRepo {
	return &GroupRepo{repo: repo}
}

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
	if group.GroupBackendsMappings != nil {
		err := r.repo.ReplaceAssociations(ctx, group, "GroupBackendsMappings", group.GroupBackendsMappings)
		if err != nil {
			provider.Logger(ctx).WithError(err).Errorw(
				"group update failed, unable to update associations",
				map[string]interface{}{"group_id": group.ID})
			return err
		}
	}
	err := r.repo.Update(ctx, group)
	if err != nil {
		if err == spine.NoRowAffected {
			provider.Logger(ctx).Debugw(
				"no row affected by group update",
				map[string]interface{}{"group_id": group.ID},
			)
			return nil
		}
		provider.Logger(ctx).WithError(err).Errorw(
			"group update failed",
			map[string]interface{}{"group_id": group.ID})
		return err
	}

	provider.Logger(ctx).Infow("group updated", map[string]interface{}{"group_id": group.ID})

	return nil
}

func (r *GroupRepo) Find(ctx context.Context, id string) (*models.Group, error) {
	group := models.Group{}

	err := r.repo.Preload(ctx, clause.Associations).FindByID(ctx, &group, id)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (r *GroupRepo) FindMany(ctx context.Context, conditions map[string]interface{}) ([]models.Group, error) {
	var groups []models.Group

	err := r.repo.Preload(ctx, clause.Associations).FindMany(ctx, &groups, conditions)
	if err != nil {
		return nil, err
	}

	return groups, nil
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
		return errors.New("already active")
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
		provider.Logger(ctx).Error("group deactivation failed. Already inactive")
		return errors.New("already inactive")
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
