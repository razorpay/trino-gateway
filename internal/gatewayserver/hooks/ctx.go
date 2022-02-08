package hooks

import (
	"context"

	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/twitchtv/twirp"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/provider"
)

// Ctx returns function which sets context with core service
// information and puts contextual logger into same for later use.
func Ctx() *twirp.ServerHooks {
	hooks := &twirp.ServerHooks{}

	hooks.RequestRouted = func(ctx context.Context) (context.Context, error) {
		ctx = boot.WithRequestID(ctx, "")

		// Adds more contextual info in above logger.
		// Todo: Check why method, service and package names are not known in this hook.
		reqMethod, _ := twirp.MethodName(ctx)
		reqService, _ := twirp.ServiceName(ctx)
		reqPackage, _ := twirp.PackageName(ctx)
		req := map[string]interface{}{
			"reqId": boot.GetRequestID(ctx),
			// TODO: set auth user in auth Hook
			// "reqUser":    ctx.Value("authUserCtxKey"),
			"reqMethod":  reqMethod,
			"reqService": reqService,
			"reqPackage": reqPackage,
		}

		return context.WithValue(ctx, logger.LoggerCtxKey, provider.Logger(ctx).WithFields(req)), nil
	}

	return hooks
}
