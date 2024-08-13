package boot

import (
	"context"
	"log"
	"os"
	"trino-api/internal/config"
	config_reader "trino-api/pkg/config"
)

var Config config.Config

func Init() {
	// Init config
	err := config_reader.NewDefaultConfig().LoadEnv(GetEnv(), &Config)
	if err != nil {
		log.Fatal(err)
	}
}
func GetEnv() string {
	// Fetch env for bootstrapping
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "dev"
	}

	return environment
}
func NewContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

// func initialize(ctx context.Context, env string) error {
// 	// log := InitLogger(ctx)

// 	context.WithValue(ctx, logger.LoggerCtxKey)

// 	return nil
// }

// func InitLogger(ctx context.Context) *logger.ZapLogger {
// 	lgrConfig := logger.Config{
// 		LogLevel:      logger.Info,
// 		ContextString: "trino_client",
// 	}

// 	Logger, err := logger.NewLogger(lgrConfig)

// 	if err != nil {
// 		panic("failed to initialize logger")
// 	}

// 	return Logger
// }

// func Logger(ctx context.Context) *logger.Entry {
// 	ctxLogger, err := logger.Ctx(ctx)

// 	if err == nil {
// 		return ctxLogger
// 	}

// 	return nil
// }

// func InitApi(ctx context.Context, env string) error {
// 	err := initialize(ctx, env)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
