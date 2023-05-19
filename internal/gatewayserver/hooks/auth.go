package hooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/twitchtv/twirp"

	"github.com/razorpay/trino-gateway/internal/boot"
)

type contextkey int

const (
	authTokenCtxKey contextkey = iota
	authUrlPathCtxKey
)

func Auth() *twirp.ServerHooks {
	hooks := &twirp.ServerHooks{}

	hooks.RequestReceived = func(ctx context.Context) (context.Context, error) {
		m, _ := ctx.Value(authUrlPathCtxKey).(string)
		if strings.Contains(m, "/Get") || strings.Contains(m, "/List") {
			return ctx, nil
		}

		token, _ := ctx.Value(authTokenCtxKey).(string)

		if token == "" {
			return ctx, twirp.NewError(
				twirp.Unauthenticated,
				fmt.Sprint(
					"empty/undefined apiToken in header: ",
					boot.Config.Auth.TokenHeaderKey),
			)
		}

		if boot.Config.Auth.Token == token {
			return ctx, nil
		}

		return ctx, twirp.NewError(twirp.Unauthenticated, "invalid apiToken for authentication")
	}

	return hooks
}

func WithAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := r.Header.Get(boot.Config.Auth.TokenHeaderKey)
		urlPath := r.URL.Path

		ctx = context.WithValue(ctx, authTokenCtxKey, token)
		ctx = context.WithValue(ctx, authUrlPathCtxKey, urlPath)

		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}
