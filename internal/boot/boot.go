package boot

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/fatih/structs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/constants/contextkeys"
	config_reader "github.com/razorpay/trino-gateway/pkg/config"
	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/razorpay/trino-gateway/pkg/spine/db"
	"github.com/rs/xid"
)

const (
	requestIDHttpHeaderKey = "X-Request-ID"
	requestIDCtxKey        = "RequestID"
)

var (
	// Config contains application configuration values.
	Config config.Config

	// DB holds the application db connection.
	DB *db.DB
)

func init() {
	// Init config
	err := config_reader.NewDefaultConfig().Load(GetEnv(), &Config)
	if err != nil {
		log.Fatal(err)
	}

	InitLogger(context.Background())

	// Init Db
	DB, err = db.NewDb(&Config.Db)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func TrinoRestInit() {
	// Init config
	err := config_reader.NewDefaultConfig().Load(GetEnv(), &Config)
	if err != nil {
		log.Fatal(err)
	}
	InitLoggerTrinoRest(context.Background())
}

// Fetch env for bootstrapping
func GetEnv() string {
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		log.Print("APP_ENV not set defaulting to dev env.", environment)
		environment = "dev"
	}

	log.Print("Setting app env to ", environment)

	return environment
}

// GetRequestID gets the request id
// if its already set in the given context
// if there is no requestID set then it'll create a new
// request id and returns the same
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(contextkeys.RequestID).(string); ok {
		return val
	}
	return xid.New().String()
}

// WithRequestID adds a request if to the context and gives the updated context back
// if the passed requestID is empty then creates one by itself
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = xid.New().String()
	}

	return context.WithValue(ctx, contextkeys.RequestID, requestID)
}

// initialize all core dependencies for the application
func initialize(ctx context.Context, env string) error {
	log := InitLogger(ctx)

	ctx = context.WithValue(ctx, logger.LoggerCtxKey, log)

	// Puts git commit hash into config.
	// This is not read automatically because env variable is not in expected format.
	if v, found := os.LookupEnv("GIT_COMMIT_HASH"); found {
		Config.App.GitCommitHash = v
	}

	// Register DB stats prometheus collector
	dbInstance, err := DB.Instance(ctx).DB()
	if err != nil {
		return err
	}
	collector := sqlstats.NewStatsCollector(Config.Db.URL+"-"+Config.Db.Name, dbInstance)
	prometheus.MustRegister(collector)

	return nil
}

func InitApi(ctx context.Context, env string) error {
	err := initialize(ctx, env)
	if err != nil {
		return err
	}

	return nil
}

func InitMigration(ctx context.Context, env string) error {
	err := initialize(ctx, env)
	if err != nil {
		return err
	}

	return nil
}

// // InitTracing initialises opentracing exporter
// func InitTracing(ctx context.Context) (io.Closer, error) {
// 	t, closer, err := tracing.Init(Config.Tracing, Logger(ctx))

// 	Tracer = t

// 	return closer, err
// }

// NewContext adds core key-value e.g. service name, git hash etc to
// existing context or to a new background context and returns.
func NewContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	for k, v := range structs.Map(struct {
		GitCommitHash string
		Env           string
		ServiceName   string
	}{
		GitCommitHash: Config.App.GitCommitHash,
		Env:           Config.App.Env,
		ServiceName:   Config.App.ServiceName,
	}) {
		key := strings.ToLower(k)
		ctx = context.WithValue(ctx, key, v)
	}
	return ctx
}

func TrinoRestNewContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func InitLogger(ctx context.Context) *logger.ZapLogger {
	lgrConfig := logger.Config{
		LogLevel:      Config.App.LogLevel,
		ContextString: "trino-gateway",
	}

	Logger, err := logger.NewLogger(lgrConfig)
	if err != nil {
		panic("failed to initialize logger")
	}

	return Logger
}

func InitLoggerTrinoRest(ctx context.Context) *logger.ZapLogger {
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
