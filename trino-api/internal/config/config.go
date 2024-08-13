package config

type Config struct {
	App  App
	Db   DB
	Auth Auth
}

type App struct {
	AppEnv          string
	ServiceName     string
	Host            string
	Port            int
	MetricsPort     int
	ShutDownTimeout int
	ShutDownDelay   int
}

type DB struct {
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

type Auth struct {
	Token map[string]string `toml:"token"`
}
