package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Title string
	Db    TestDbConfig
}

type TestDbConfig struct {
	Dialect               string
	Protocol              string
	Host                  string
	Port                  int
	Username              string
	Password              string
	SslMode               string
	Name                  string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	IsAuthDelegated       bool
	TrinoBackendDB        TrinoBackendDB `mapstructure:"trino-backend-db"`
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

func TestLoadConfig(t *testing.T) {
	var c TestConfig

	key := strings.ToUpper("trino-gateway") + "_DB_PASSWORD"
	os.Setenv(key, "envpass")
	err := NewConfig(NewOptions("toml", "./testdata", "default")).Load("default", &c)
	assert.Nil(t, err)
	// Asserts that default value exists.
	assert.Equal(t, "mysql", c.Db.Dialect)
	assert.Equal(t, "trino", c.Db.TrinoBackendDB.Dialect)
	// Asserts that application environment specific value got overridden.
	assert.Equal(t, 10, c.Db.MaxOpenConnections)
	assert.Equal(t, "http://user@trino_rest:8080?catalog=default&schema=public", c.Db.TrinoBackendDB.DSN)
	// Asserts that environment variable was honored.
	assert.Equal(t, "envpass", c.Db.Password)
	assert.Equal(t, "user", c.Db.TrinoBackendDB.Password)
}
