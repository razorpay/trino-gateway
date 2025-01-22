package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/router/trinoheaders"
	"github.com/razorpay/trino-gateway/internal/utils"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type IAuthService interface {
	Authenticate(ctx *context.Context, username string, password string) (bool, error)
}

type AuthService struct {
	ValidationProviderURL   string
	ValidationProviderToken string
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
func (s *AuthService) ValidateFromValidationProvider(ctx *context.Context, username string, password string) (bool, error) {
	payload := struct {
		Username string `json:"email"`
		Token    string `json:"token"`
	}{
		Username: username,
		Token:    password,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", s.ValidationProviderURL, bytes.NewReader(payloadBytes))
	req.Header.Set("X-Auth-Token", s.ValidationProviderToken)
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

	return data.OK, nil
}

func (s *AuthService) Authenticate(ctx *context.Context, username string, password string) (bool, error) {
	authCache := s.GetInMemoryAuthCache(ctx)

	if entry, exists := authCache.Get(username); exists && entry == password {
		authCache.Update(username, password)
		return true, nil
	}

	isValid, err := s.ValidateFromValidationProvider(ctx, username, password)
	if err != nil {
		return false, err
	}

	if isValid {
		authCache.Update(username, password)
	}

	return isValid, nil
}

func (s *AuthService) GetInMemoryAuthCache(ctx *context.Context) utils.ISimpleCache {
	ctxKeyName := "routerAuthCache"
	authCache, ok := (*ctx).Value(ctxKeyName).(*utils.InMemorySimpleCache)

	if !ok {

		expiryInterval, _ := time.ParseDuration(boot.Config.Auth.Router.DelegatedAuth.CacheTTLMinutes)

		authCache = &utils.InMemorySimpleCache{
			Cache: make(map[string]struct {
				Timestamp time.Time
				Value     string
			}),
			ExpiryInterval: expiryInterval,
		}
		*ctx = context.WithValue(*ctx, ctxKeyName, authCache)
	}
	return authCache
}

func (r *RouterServer) isAuthDelegated(ctx *context.Context) (bool, error) {
	res, err := r.gatewayApiClient.Policy.EvaluateAuthDelegationForClient(*ctx, &gatewayv1.EvaluateAuthDelegationRequest{IncomingPort: int32(r.port)})
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			fmt.Sprint(LOG_TAG, "Failed to evaluate auth delegation policy. Assuming delegation is disabled."),
			map[string]interface{}{
				"port": r.port,
			})
		return false, err
	}
	return res.GetIsAuthDelegated(), nil
}

func (r *RouterServer) AuthHandler(ctx *context.Context, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if isAuth, _ := r.isAuthDelegated(ctx); isAuth {
			// TODO: Refactor auth type handling to a dedicated type

			// BasicAuth
			username, password, isBasicAuth := req.BasicAuth()

			// CustomAuth
			if !isBasicAuth {
				provider.Logger(*ctx).Debug("Custom Auth type")
				username = trinoheaders.Get(trinoheaders.User, req)
				password = trinoheaders.Get(trinoheaders.Password, req)
			} else {
				if u := trinoheaders.Get(trinoheaders.User, req); u != username {
					errorMsg := fmt.Sprintf("Username from basicauth - %s does not match with User principal - %s", username, u)
					provider.Logger(*ctx).Debug(errorMsg)
					http.Error(w, errorMsg, http.StatusUnauthorized)
					return
				}

				// Remove auth details from request
				req.Header.Del("Authorization")
			}

			// NoAuth
			isNoAuth := password == ""
			if isNoAuth {
				provider.Logger(*ctx).Debug("No Auth type detected")
				errorMsg := "Password required"
				http.Error(w, errorMsg, http.StatusUnauthorized)
				return
			}

			isAuthenticated, err := r.authService.Authenticate(ctx, username, password)
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
			h.ServeHTTP(w, req)
		} else {
			// whacky stuff
			username, password, isBasicAuth := req.BasicAuth()

			// CustomAuth
			if isBasicAuth {
				if username == "upi-offering-service-payments.de-apps@razorpay.com" {
					if u := trinoheaders.Get(trinoheaders.User, req); u != username {
						errorMsg := fmt.Sprintf("Username from basicauth - %s does not match with User principal - %s", username, u)
						provider.Logger(*ctx).Debug(errorMsg)
						http.Error(w, errorMsg, http.StatusUnauthorized)
						return
					}

					// Remove auth details from request
					req.Header.Del("Authorization")
					isAuthenticated, err := r.authService.Authenticate(ctx, username, password)
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
				}
			}
			h.ServeHTTP(w, req)
		}
	})
}
