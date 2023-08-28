package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/router/trinoheaders"
)

type AuthCache struct {
	Cache map[string]struct {
		Timestamp time.Time
		Password  string
	}
	mu sync.Mutex
}
type AuthService interface {
	Authenticate(ctx *context.Context, username, password string) (bool, error)
}
type DefaultAuthService struct{}

func NewDefaultAuthService() *DefaultAuthService {
	return &DefaultAuthService{}
}

/*
Calls an Auth Token Validator Service with following api contract:
With Params-
{
	"email": "abc@xyz.com",
	"token": "token123"
}
api returns-
If Authenticated - {"ok": true}
If not authenticated- {"ok": false}

@returns-
boolean{True or False},error_message
*/
func (s *DefaultAuthService) Authenticate(ctx *context.Context, username string, password string) (bool, error) {
	authCache, ok := (*ctx).Value("routerAuthCache").(*AuthCache)

	if !ok {
		authCache = &AuthCache{
			Cache: make(map[string]struct {
				Timestamp time.Time
				Password  string
			}),
		}
		*ctx = context.WithValue(*ctx, "routerAuthCache", authCache)
	}

	if authCache.Check(username, password) {
		authCache.Update(username, password)
		return true, nil
	}
	payload := struct {
		Username string `json:"email"`
		Token    string `json:"token"`
	}{
		Username: username,
		Token:    password,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", boot.Config.Auth.Router.ValidationURL, bytes.NewReader(payloadBytes))
	req.Header.Set("X-Auth-Token", boot.Config.Auth.Router.ValidationToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	var data struct {
		OK bool `json:"ok"`
	}
	jsonParseError := json.Unmarshal([]byte(respBody), &data)
	if jsonParseError != nil {
		return false, jsonParseError
	}

	if data.OK {
		authCache.Update(username, password)
	}

	return data.OK, nil
}

func (authCache *AuthCache) Check(username, password string) bool {
	authCache.mu.Lock()
	defer authCache.mu.Unlock()

	entry, found := authCache.Cache[username]

	if !found {
		return false
	}
	cachedInterval := boot.Config.Auth.Router.CacheTTLMinutes

	cachedDuration, _ := time.ParseDuration(cachedInterval)

	if time.Since(entry.Timestamp) > cachedDuration {
		// If entry is older than cachedDuration, then delete the record and return false
		delete(authCache.Cache, username)
		return false
	}

	return entry.Password == password
}

func (authCache *AuthCache) Update(username, password string) {
	authCache.mu.Lock()
	defer authCache.mu.Unlock()

	authCache.Cache[username] = struct {
		Timestamp time.Time
		Password  string
	}{
		Timestamp: time.Now(),
		Password:  password,
	}
}

func WithAuth(ctx *context.Context, h http.Handler, authService ...AuthService) http.Handler {

	var auth AuthService
	if len(authService) > 0 {
		auth = authService[0]
	} else {
		auth = NewDefaultAuthService()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username := trinoheaders.Get(trinoheaders.User, r)
		password := trinoheaders.Get(trinoheaders.Password, r)

		//Remove this later after full rollout
		if password == "" {
			h.ServeHTTP(w, r)
			return
		}

		isAuthenticated, err := auth.Authenticate(ctx, username, password)

		if err != nil {
			errorMsg := fmt.Sprintf("Unable to Authenticate users. Getting error - %s", err)
			provider.Logger(*ctx).Error(errorMsg)
			http.Error(w, "Unable to Authenticate the user", http.StatusNotFound)
			return
		}
		if !isAuthenticated {
			provider.Logger(*ctx).Debug(fmt.Sprintf("User - %s not authenticated", username))
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}
