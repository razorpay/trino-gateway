package repo

import (
	"context"
	"errors"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
)

type IBackendRepo interface {
	Create(ctx context.Context, backend *models.Backend) error
	Update(ctx context.Context, backend *models.Backend) error
	Find(ctx context.Context, id string) (*models.Backend, error)
	FindMany(ctx context.Context, conditions map[string]interface{}) (*[]models.Backend, error)
	// GetAll(ctx context.Context) (*[]models.Backend, error)
	// GetAllActive(ctx context.Context) (*[]models.Backend, error)
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string) error
	Disable(ctx context.Context, id string) error
}

type BackendRepo struct {
	repo dbRepo.IDbRepo
}

// NewBackendRepo returns a new instance of *BackendRepo
func NewBackendRepo(ctx context.Context, repo dbRepo.IDbRepo) *BackendRepo {
	_ = ctx
	return &BackendRepo{repo: repo}
}

func (r *BackendRepo) Create(ctx context.Context, backend *models.Backend) error {
	err := r.repo.Create(ctx, backend)
	if err != nil {
		provider.Logger(ctx).WithError(err).Error("backend create failed")
		return err
	}

	provider.Logger(ctx).Infow("backend created", map[string]interface{}{"user_id": backend.ID})

	return nil
}

func (r *BackendRepo) Update(ctx context.Context, backend *models.Backend) error {
	err := r.repo.Update(ctx, backend)
	if err != nil {
		provider.Logger(ctx).WithError(err).Errorw("user update failed", map[string]interface{}{"user_id": backend.ID})
		return err
	}

	provider.Logger(ctx).Infow("backend updated", map[string]interface{}{"user_id": backend.ID})

	return nil
}

func (r *BackendRepo) Find(ctx context.Context, id string) (*models.Backend, error) {
	backend := models.Backend{}

	err := r.repo.FindByID(ctx, &backend, id)
	if err != nil {
		return nil, err
	}

	return &backend, nil
}

func (r *BackendRepo) FindMany(ctx context.Context, conditions map[string]interface{}) (*[]models.Backend, error) {
	var backends []models.Backend

	err := r.repo.FindMany(ctx, &backends, conditions)
	if err != nil {
		return nil, err
	}

	return &backends, nil
}

// func (r *BackendRepo) GetAll(ctx context.Context) (*[]models.Backend, error) {
// 	var backends []models.Backend

// 	err := r.repo.FindMany(ctx, &backends, make(map[string]interface{}))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &backends, nil
// }

// func (r *BackendRepo) GetAllActive(ctx context.Context) (*[]models.Backend, error) {
// 	var backends []models.Backend

// 	err := r.repo.FindMany(ctx, &backends, map[string]interface{}{"is_enabled": true})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &backends, nil
// }

func (r *BackendRepo) Enable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("backend activation triggered", map[string]interface{}{"backend_id": id})

	backend, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("backend activation failed: " + err.Error())
		return err
	}

	if *backend.IsEnabled {
		provider.Logger(ctx).Error("backend activation failed. Already active")
		return errors.New("Already active")
	}

	*backend.IsEnabled = true

	if err := r.repo.Update(ctx, backend); err != nil {
		return err
	}

	return nil
}

func (r *BackendRepo) Disable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("backend activation triggered", map[string]interface{}{"backend_id": id})

	backend, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("backend activation failed: " + err.Error())
		return err
	}

	if !*backend.IsEnabled {
		provider.Logger(ctx).Error("backend activation failed. Already active")
		return errors.New("Already active")
	}

	*backend.IsEnabled = false

	if err := r.repo.Update(ctx, backend); err != nil {
		return err
	}

	return nil
}

func (r *BackendRepo) Delete(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("backend delete request", map[string]interface{}{"backend_id": id})

	backend, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("backend delete failed: " + err.Error())
		return err
	}

	// _ = backend

	err = r.repo.Delete(ctx, backend)
	if err != nil {
		return err
	}

	return nil
}
