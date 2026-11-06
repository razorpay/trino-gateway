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
	Router  Router
}

// Router configures the smart router sidecar. Mode "shadow" logs the
// router's decision and routes as before; "enforce" follows it. Tiers maps a
// routing group to its S/M/L backends, e.g. tiers.adhoc.S = ["trino-srm"].
type Router struct {
	Enabled   bool
	URL       string
	TimeoutMs int
	Mode      string
	Tiers     map[string]map[string][]string
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
