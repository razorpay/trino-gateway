package router_test

import (
	"context"
	"errors"
	r "github.com/razorpay/trino-gateway/internal/router"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type MockAuthenticator struct {
	mock.Mock
}

// A mock struct that implements the AuthService interface
type AuthServiceMock struct {
	authMock *MockAuthenticator
}
type AuthCache struct {
	cache map[string]struct {
		Timestamp time.Time
		Password  string
	}
	mu sync.Mutex
}

func createContextWithHeaders() context.Context {
	// Create a context with a value for routerAuthCache
	ctx := context.WithValue(context.Background(), "routerAuthCache", &AuthCache{
		cache: make(map[string]struct {
			Timestamp time.Time
			Password  string
		}),
	})
	return ctx
}

func createTestRequestWithHeaders(username, password string) *http.Request {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trino-User", username)
	req.Header.Set("X-Trino-Password", password)
	return req
}

func (m *AuthServiceMock) Authenticate(ctx *context.Context, username string, password string) (bool, error) {
	args := m.authMock.Called(ctx, username, password)
	return args.Bool(0), args.Error(1)
}

func Test_WithAuth_Authorised(t *testing.T) {
	authMock := &MockAuthenticator{}
	ctx := createContextWithHeaders()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	authMock.On("Authenticate", &ctx, "testuser", "testpassword").Return(true, nil)

	authService := &AuthServiceMock{authMock}

	wrappedHandler := r.WithAuth(&ctx, http.HandlerFunc(handler), authService)

	req := createTestRequestWithHeaders("testuser", "testpassword")

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	authMock.AssertExpectations(t)
}

func Test_WithAuth_UnAuthorised(t *testing.T) {
	authMock := &MockAuthenticator{}
	ctx := createContextWithHeaders()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	authMock.On("Authenticate", &ctx, "testuser", "testpassword").Return(false, nil)

	authService := &AuthServiceMock{authMock}

	wrappedHandler := r.WithAuth(&ctx, http.HandlerFunc(handler), authService)

	req := createTestRequestWithHeaders("testuser", "testpassword")

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	authMock.AssertExpectations(t)
}

func Test_WithAuth_Error(t *testing.T) {
	authMock := &MockAuthenticator{}
	ctx := createContextWithHeaders()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	authMock.On("Authenticate", &ctx, "testuser", "testpassword").Return(false, errors.New("authentication failed"))

	authService := &AuthServiceMock{authMock}

	wrappedHandler := r.WithAuth(&ctx, http.HandlerFunc(handler), authService)

	req := createTestRequestWithHeaders("testuser", "testpassword")

	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rr.Code)
	}

	authMock.AssertExpectations(t)
}

func Test_Check_ValidCredentials_WithinCache(t *testing.T) {
	// Create an instance of AuthCache

	authCache := &r.AuthCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Password  string
		}),
	}
	//r.AuthCache{authCache}
	// Define test credentials
	username := "testuser"
	password := "testpassword"

	// Add a test entry to the cache
	entry := struct {
		Timestamp time.Time
		Password  string
	}{
		Timestamp: time.Now(),
		Password:  password,
	}
	authCache.Cache[username] = entry
	valid := authCache.Check(username, password)

	// Assert that the check method returns true for valid credentials
	if !valid {
		t.Error("Expected valid credentials, but got invalid credentials")
	}
}

func Test_Check_ValidCredentials_NotCached(t *testing.T) {
	// Create an instance of AuthCache

	authCache := &r.AuthCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Password  string
		}),
	}
	//r.AuthCache{authCache}
	// Define test credentials
	username := "testuser"
	password := "testpassword"

	// Add a test entry to the cache
	entry := struct {
		Timestamp time.Time
		Password  string
	}{
		Timestamp: time.Now().Add(-15 * time.Minute),
		Password:  password,
	}
	authCache.Cache[username] = entry
	valid := authCache.Check(username, password)

	// Assert that the check method returns true for valid credentials
	if valid {
		t.Error("Expected expired credentials, but got valid credentials")
	}
}

func Test_Check_InValidCredentials_WithinCache(t *testing.T) {
	// Create an instance of AuthCache

	authCache := &r.AuthCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Password  string
		}),
	}
	//r.AuthCache{authCache}
	// Define test credentials
	username := "testuser"
	password := "testpassword"

	// Add a test entry to the cache
	entry := struct {
		Timestamp time.Time
		Password  string
	}{
		Timestamp: time.Now(),
		Password:  password,
	}
	authCache.Cache[username] = entry

	wrongpassword := "falsepassword"
	valid := authCache.Check(username, wrongpassword)

	// Assert that the check method returns true for valid credentials
	if valid {
		t.Error("Expected Invalid credentials, but got valid credentials")
	}
}

func Test_Update(t *testing.T) {
	authCache := &r.AuthCache{
		Cache: make(map[string]struct {
			Timestamp time.Time
			Password  string
		}),
	}
	authCache.Update("testUser", "testPassword")

	expectedPassword := "testPassword"
	if entry, exists := authCache.Cache["testUser"]; exists {
		if entry.Password != expectedPassword {
			t.Errorf("Password for 'testUser' is not as expected. Got: %s, Expected: %s", entry.Password, expectedPassword)
		}
	} else {
		t.Errorf("Entry for 'testUser' not found in the cache")
	}
}
