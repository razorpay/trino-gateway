package config

import (
	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

type Config struct {
	App     App
	Auth    Auth
	Db      db.Config
	Gateway Gateway
	Monitor Monitor
}

// App contains application-specific config values
type App struct {
	Env                     string
	GitCommitHash           string
	LogLevel                string
	MetricsPort             int
	Port                    int
	ServiceExternalHostname string
	ServiceHostname         string
	ServiceName             string
	ShutdownDelay           int
	ShutdownTimeout         int
}

type Auth struct {
	Token          string
	TokenHeaderKey string
	Router         struct {
		ValidationURL   string
		ValidationToken string
		CacheTTLMinutes string
		Authenticate    string
	}
}

type Gateway struct {
	DefaultRoutingGroup string
	Ports               []int
	Network             string
}

type Monitor struct {
	Interval          string
	StatsValiditySecs int
	Trino             struct {
		User     string
		Password string
	}
	HealthCheckSql string
}
