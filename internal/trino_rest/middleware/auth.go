package middleware

import (
	"fmt"
	"net/http"

	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/router"
	"github.com/razorpay/trino-gateway/internal/trino_rest/utils"
)

type AuthMiddleware struct {
	authService *router.AuthService
	cfg         *config.Config
}

func NewAuthMiddleware(authService *router.AuthService, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{authService: authService, cfg: cfg}
}

func (m *AuthMiddleware) BasicAuthenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "authorization required")
			return
		}
		if m.cfg.TrinoRest.IsAuthDelegated {
			ctx := r.Context()
			isValid, err := m.authService.ValidateFromValidationProvider(&ctx, username, password)
			if err != nil || !isValid {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			m.cfg.TrinoRest.TrinoBackendDB.DSN = fmt.Sprintf("http://%s:@%s:%d?catalog=%s&schema=%s",
				username,
				m.cfg.TrinoRest.TrinoBackendDB.URL,
				m.cfg.TrinoRest.TrinoBackendDB.Port,
				m.cfg.TrinoRest.TrinoBackendDB.Catalog,
				m.cfg.TrinoRest.TrinoBackendDB.Schema,
			)
		} else {
			if username == "" || password == "" {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			m.cfg.TrinoRest.TrinoBackendDB.DSN = fmt.Sprintf("http://%s:%s@%s:%d?catalog=%s&schema=%s",
				username,
				password,
				m.cfg.TrinoRest.TrinoBackendDB.URL,
				m.cfg.TrinoRest.TrinoBackendDB.Port,
				m.cfg.TrinoRest.TrinoBackendDB.Catalog,
				m.cfg.TrinoRest.TrinoBackendDB.Schema,
			)
		}
		next.ServeHTTP(w, r)
	})
}
