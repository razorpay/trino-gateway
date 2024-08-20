package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Title string
	Db    TestDbConfig
}

type TestDbConfig struct {
	Dialect  string
	Protocol string
	Host     string
	Port     int
	UserName string
	PassWord string
	Catalog  string
	Schema   string
	DSN      string
}

func TestConfigLoader(t *testing.T) {
	var c TestConfig

	key := strings.ToUpper("trino_client") + "_DB_PASSWORD"
	os.Setenv(key, "envpass")
	err := NewConfig(NewOptions("toml", "./testdata", "default")).LoadEnv("drone", &c)
	assert.Nil(t, err)
	assert.Equal(t, "trino", c.Db.Dialect)
	assert.Equal(t, "localhost", c.Db.Host)
	assert.Equal(t, "envpass", c.Db.PassWord)
	assert.Equal(t, "mysql://trino-client:trino-client@127.0.0.1:8090/trino-client-catalog/trino-client-schema", c.Db.DSN)
}
