package config

import (
	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

type Config struct {
	App     App
	Auth    Auth
	Db      db.Config
	Gateway Gateway
}

// App contains application-specific config values
type App struct {
	Env                     string
	GitCommitHash           string
	LogLevel                string
	MetricsPort             int
	GuiPort                 int
	Port                    int
	ServiceExternalHostname string
	ServiceHostname         string
	ServiceName             string
	ShutdownDelay           int
	ShutdownTimeout         int
}

type Auth struct {
	Password string
	Username string
}

type Gateway struct {
	DefaultRoutingGroup string
	Ports               []int
}

type Monitor struct {
	IntervalSecs int
	Threshold    int
}
