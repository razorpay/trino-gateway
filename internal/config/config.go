package config

import (
	"time"

	"github.com/razorpay/trino-gateway/pkg/spine/db"
)

type Config struct {
	App       App
	Auth      Auth
	Db        db.Config
	Gateway   Gateway
	Monitor   Monitor
	TrinoRest TrinoRest
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
		DelegatedAuth struct {
			ValidationProviderURL   string
			ValidationProviderToken string
			CacheTTLMinutes         string
		}
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

type TrinoRest struct {
	AppEnv          string
	ServiceName     string
	Hostname        string
	Port            int
	MetricsPort     int
	ShutdownTimeout time.Duration
	ShutdownDelay   time.Duration
	MaxRecords      int
	IsAuthDelegated bool
	TrinoBackendDB  TrinoBackendDB
}

type TrinoBackendDB struct {
	Dialect  string
	Protocol string
	URL      string
	Port     int
	Username string
	Password string
	Catalog  string
	Schema   string
	DSN      string
}
