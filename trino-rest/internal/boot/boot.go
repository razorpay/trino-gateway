package boot

import (
	"context"
	"log"
	"os"
	"trino-api/internal/config"
	config_reader "trino-api/pkg/config"
	"trino-api/pkg/logger"
)

var Config config.Config

func Init() {
	// Init config
	err := config_reader.NewDefaultConfig().LoadEnv(GetEnv(), &Config)
	if err != nil {
		log.Fatal(err)
	}
	InitLogger(context.Background())
}
func GetEnv() string {
	// Fetch env for bootstrapping
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		log.Print("APP_ENV not set defaulting to dev env. ", environment)
		environment = "dev"
	}

	log.Print("Setting app env to ", environment)
	return environment
}
func NewContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func InitLogger(ctx context.Context) *logger.ZapLogger {
	lgrConfig := logger.Config{
		LogLevel:      logger.Info,
		ContextString: "trino_client",
	}

	Logger, err := logger.NewLogger(lgrConfig)

	if err != nil {
		panic("failed to initialize logger")
	}

	return Logger
}
