package repo

import (
	"context"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/database/dbRepo"
	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/pkg/spine"
)

type IBackendRepo interface {
	Create(ctx context.Context, backend *models.Backend) error
	Update(ctx context.Context, backend *models.Backend) error
	Find(ctx context.Context, id string) (*models.Backend, error)
	FindMany(ctx context.Context, conditions map[string]interface{}) ([]models.Backend, error)
	// GetAll(ctx context.Context) ([]models.Backend, error)
	// GetAllActive(ctx context.Context) ([]models.Backend, error)
	GetAllActiveByIDs(ctx context.Context, ids []string) ([]models.Backend, error)
	Delete(ctx context.Context, id string) error
	Enable(ctx context.Context, id string) error
	Disable(ctx context.Context, id string) error
	MarkHealthy(ctx context.Context, id string) error
	MarkUnhealthy(ctx context.Context, id string) error
}

type BackendRepo struct {
	repo dbRepo.IDbRepo
}

// NewBackendRepo returns a new instance of *BackendRepo
func NewBackendRepo(repo dbRepo.IDbRepo) *BackendRepo {
	return &BackendRepo{repo: repo}
}

func (r *BackendRepo) Create(ctx context.Context, backend *models.Backend) error {
	err := r.repo.Create(ctx, backend)
	if err != nil {
		provider.Logger(ctx).WithError(err).Error("backend create failed")
		return err
	}

	provider.Logger(ctx).Infow("backend created", map[string]interface{}{"backend_id": backend.ID})

	return nil
}

func (r *BackendRepo) Update(ctx context.Context, backend *models.Backend) error {
	err := r.repo.Update(ctx, backend)
	if err != nil {
		if err == spine.NoRowAffected {
			provider.Logger(ctx).Debugw(
				"no row affected by backend update",
				map[string]interface{}{"backend_id": backend.ID},
			)
			return nil
		}
		provider.Logger(ctx).WithError(err).Errorw(
			"backend update failed",
			map[string]interface{}{"backend_id": backend.ID},
		)
		return err
	}

	provider.Logger(ctx).Infow("backend updated", map[string]interface{}{"backend_id": backend.ID})

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

// Returns list of backend ids which are ready to serve traffic
func (r *BackendRepo) GetAllActiveByIDs(ctx context.Context, ids []string) ([]models.Backend, error) {
	var backends []models.Backend

	err := r.repo.FindWithConditionByIDs(
		ctx,
		&backends,
		map[string]interface{}{"is_enabled": true, "is_healthy": true},
		ids,
	)
	if err != nil {
		return nil, err
	}

	return backends, nil
}

func (r *BackendRepo) FindMany(ctx context.Context, conditions map[string]interface{}) ([]models.Backend, error) {
	var backends []models.Backend

	err := r.repo.FindMany(ctx, &backends, conditions)
	if err != nil {
		return nil, err
	}

	return backends, nil
}

// func (r *BackendRepo) GetAll(ctx context.Context) ([]models.Backend, error) {
// 	var backends []models.Backend

// 	err := r.repo.FindMany(ctx, &backends, make(map[string]interface{}))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &backends, nil
// }

// func (r *BackendRepo) GetAllActive(ctx context.Context) ([]models.Backend, error) {
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
		provider.Logger(ctx).Info("backend activation failed. Already active")
		return nil
	}

	*backend.IsEnabled = true

	if err := r.repo.Update(ctx, backend); err != nil {
		return err
	}

	return nil
}

func (r *BackendRepo) Disable(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("backend deactivation triggered", map[string]interface{}{"backend_id": id})

	backend, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("backend deactivation failed: " + err.Error())
		return err
	}

	if !*backend.IsEnabled {
		provider.Logger(ctx).Info("backend deactivation failed. Already inactive")
		return nil
	}

	*backend.IsEnabled = false

	if err := r.repo.Update(ctx, backend); err != nil {
		return err
	}

	return nil
}

func (r *BackendRepo) MarkHealthy(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("backend mark as healthy triggered", map[string]interface{}{"backend_id": id})

	backend, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("backend mark as healthy failed: " + err.Error())
		return err
	}

	if *backend.IsHealthy {
		provider.Logger(ctx).Info("backend mark as healthy failed. Already healthy")
		return nil
	}

	*backend.IsHealthy = true

	if err := r.repo.Update(ctx, backend); err != nil {
		return err
	}

	return nil
}

func (r *BackendRepo) MarkUnhealthy(ctx context.Context, id string) error {
	provider.Logger(ctx).Infow("backend mark as unhealthy triggered", map[string]interface{}{"backend_id": id})

	backend, err := r.Find(ctx, id)
	if err != nil {
		provider.Logger(ctx).Error("backend mark as unhealthy failed: " + err.Error())
		return err
	}

	if !*backend.IsHealthy {
		provider.Logger(ctx).Info("backend mark as unhealthy failed. Already unhealthy")
		return nil
	}

	*backend.IsHealthy = false

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
