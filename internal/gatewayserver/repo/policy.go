package repo

import (
	"context"
	"errors"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/pkg/spine"
)

type IPolicyRepo interface {
	Create(ctx context.Context, policy *models.Policy) error
	Update(ctx context.Context, policy *models.Policy) error
	Find(ctx context.Context, id string) (*models.Policy, error)
	FindMany(ctx context.Context, conditions map[string]interface{}) ([]models.Policy, error)
	// GetAll(ctx context.Context) ([]models.Policy, error)
	// GetAllActive(ctx context.Context) ([]models.Policy, error)
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string) error
	Disable(ctx context.Context, id string) error
}

type PolicyRepo struct {
	repo dbRepo.IDbRepo
}

// NewCore returns a new instance of *Core
func NewPolicyRepo(repo dbRepo.IDbRepo) *PolicyRepo {
	return &PolicyRepo{repo: repo}
}

func (r *PolicyRepo) Create(ctx context.Context, policy *models.Policy) error {
	err := r.repo.Create(ctx, policy)
	if err != nil {
		provider.Logger(ctx).WithError(err).Errorw("policy create failed", map[string]interface{}{"id": policy.ID})
		return err
	}

	provider.Logger(ctx).Infow("policy created", map[string]interface{}{"id": policy.ID})

	return nil
}

func (r *PolicyRepo) Update(ctx context.Context, policy *models.Policy) error {
	err := r.repo.Update(ctx, policy)
	if err != nil {
		if err == spine.NoRowAffected {
			provider.Logger(ctx).Debugw(
				"no row affected by policy update",
				map[string]interface{}{"policy_id": policy.ID},
			)
			return nil
		}
		provider.Logger(ctx).WithError(err).Errorw(
			"policy update failed",
			map[string]interface{}{"policy_id": policy.ID})
		return err
	}

	provider.Logger(ctx).Infow("policy updated", map[string]interface{}{"id": policy.ID})

	return nil
}

func (r *PolicyRepo) Find(ctx context.Context, id string) (*models.Policy, error) {
	policy := models.Policy{}

	err := r.repo.FindByID(ctx, &policy, id)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

func (r *PolicyRepo) FindMany(ctx context.Context, conditions map[string]interface{}) ([]models.Policy, error) {
	var policies []models.Policy

	err := r.repo.FindMany(ctx, &policies, conditions)
	if err != nil {
		return nil, err
	}

	return policies, nil
}

// func (r *PolicyRepo) GetAll(ctx context.Context) ([]models.Policy, error) {
// 	var policies []models.Policy

// 	err := r.repo.FindMany(ctx, &policies, make(map[string]interface{}))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &policies, nil
// }

// func (r *PolicyRepo) GetAllActive(ctx context.Context) ([]models.Policy, error) {
// 	var policies []models.Policy

// 	err := r.repo.FindMany(ctx, &policies, map[string]interface{}{"is_enabled": true})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &policies, nil
// }

func (r *PolicyRepo) Enable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("policy activation triggered", map[string]interface{}{"policy_id": id})

	policy, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("policy activation failed: " + err.Error())
		return err
	}

	if *policy.IsEnabled {
		provider.Logger(ctx).Error("policy activation failed. Already active")
		return errors.New("Already active")
	}

	*policy.IsEnabled = true

	if err := r.repo.Update(ctx, policy); err != nil {
		return err
	}

	return nil
}

func (r *PolicyRepo) Disable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("policy activation triggered", map[string]interface{}{"policy_id": id})

	policy, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("policy activation failed: " + err.Error())
		return err
	}

	if !*policy.IsEnabled {
		provider.Logger(ctx).Error("policy activation failed. Already active")
		return errors.New("Already active")
	}

	*policy.IsEnabled = false

	if err := r.repo.Update(ctx, policy); err != nil {
		return err
	}

	return nil
}

func (r *PolicyRepo) Delete(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("policy delete request", map[string]interface{}{"policy_id": id})

	policy, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("policy delete failed: " + err.Error())
		return err
	}

	// _ = policy

	err = r.repo.Delete(ctx, policy)
	if err != nil {
		return err
	}

	return nil
}
