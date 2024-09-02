package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/router"
	"github.com/razorpay/trino-gateway/internal/trino_rest/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Authenticate(ctx *context.Context, username string, password string) (bool, error) {
	args := m.Called(ctx, username, password)
	return args.Bool(0), args.Error(1)
}
func Test_MissingCredentials(t *testing.T) {
	cfg := &config.Config{}
	middleware := NewAuthMiddleware(&router.AuthService{}, cfg)

	handler := middleware.BasicAuthenticator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func Test_InvalidCredentials_Delegated(t *testing.T) {
	authService := MockAuthService{}
	ctx := context.Background()
	authService.On("Authenticate", &ctx, "invalidUser", "").Return(false, nil)
	cfg := &config.Config{}
	cfg.TrinoRest.IsAuthDelegated = true

	middleware := NewAuthMiddleware(&router.AuthService{}, cfg)

	handler := middleware.BasicAuthenticator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"errorCode":401,"message":"invalid credentials"}}`))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)
	req.SetBasicAuth("invalidUser", "invalidUser")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()
	var respData model.RespData
	err := json.NewDecoder(result.Body).Decode(&respData)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, respData.Error.ErrorCode)
	assert.Equal(t, "invalid credentials", respData.Error.Message)
}

func Test_ValidCredentials_Delegated(t *testing.T) {
	authService := MockAuthService{}
	ctx := context.Background()
	authService.On("Authenticate", &ctx, "validUser", "validPass").Return(true, nil)
	cfg := &config.Config{}
	cfg.TrinoRest.IsAuthDelegated = true
	cfg.TrinoRest.TrinoBackendDB.URL = "localhost"
	cfg.TrinoRest.TrinoBackendDB.Port = 8080
	cfg.TrinoRest.TrinoBackendDB.Catalog = "default"
	cfg.TrinoRest.TrinoBackendDB.Schema = "public"
	middleware := NewAuthMiddleware(&router.AuthService{}, cfg)
	handler := middleware.BasicAuthenticator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)
	req.SetBasicAuth("validUser", "validPass")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://validUser:@localhost:8080?catalog=default&schema=public", cfg.TrinoRest.TrinoBackendDB.DSN)
}

func Test_InvalidCredentials_NotDelegated(t *testing.T) {
	authService := MockAuthService{}
	ctx := context.Background()
	authService.On("Authenticate", &ctx, "", "").Return(false, nil)
	cfg := &config.Config{}
	cfg.TrinoRest.IsAuthDelegated = false

	middleware := NewAuthMiddleware(&router.AuthService{}, cfg)

	handler := middleware.BasicAuthenticator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)
	req.SetBasicAuth("", "")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()
	var respData model.RespData
	err := json.NewDecoder(result.Body).Decode(&respData)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, respData.Error.ErrorCode)
	assert.Equal(t, "invalid credentials", respData.Error.Message)
}

func Test_ValidCredentials_NotDelegated(t *testing.T) {
	authService := MockAuthService{}
	ctx := context.Background()
	authService.On("Authenticate", &ctx, "validUser", "validPass").Return(true, nil)
	cfg := &config.Config{}
	cfg.TrinoRest.IsAuthDelegated = false
	cfg.TrinoRest.TrinoBackendDB.URL = "localhost"
	cfg.TrinoRest.TrinoBackendDB.Port = 8080
	cfg.TrinoRest.TrinoBackendDB.Catalog = "default"
	cfg.TrinoRest.TrinoBackendDB.Schema = "public"
	middleware := NewAuthMiddleware(&router.AuthService{}, cfg)
	handler := middleware.BasicAuthenticator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(ctx)
	req.SetBasicAuth("validUser", "validPass")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://validUser:validPass@localhost:8080?catalog=default&schema=public", cfg.TrinoRest.TrinoBackendDB.DSN)
}
