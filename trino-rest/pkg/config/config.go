package config

import (
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

const (
	DefaultConfigType = "toml"
	DefaultConfigDir  = "./config"
	DefaultConfigFile = "default"
	WorkDirEnv        = "WORKDIR"
)

type Options struct {
	configType        string
	configPath        string
	deafultConfigFile string
}

type Config struct {
	opts  Options
	viper *viper.Viper
}

func NewDefaultOptions() Options {
	var configPath string
	workDir := os.Getenv(WorkDirEnv)
	if workDir != "" {
		configPath = path.Join(workDir, DefaultConfigDir)
	} else {
		_, thisFile, _, _ := runtime.Caller(1)
		configPath = path.Join(path.Dir(thisFile), "../.."+DefaultConfigDir)
	}

	return NewOptions(DefaultConfigType, configPath, DefaultConfigFile)
}

func NewOptions(configType string, configPath string, defaultConfigFile string) Options {
	return Options{configType: configType, configPath: configPath, deafultConfigFile: defaultConfigFile}
}

func NewDefaultConfig() *Config {
	return NewConfig(NewDefaultOptions())
}

func NewConfig(opts Options) *Config {
	return &Config{opts, viper.New()}
}

func (c *Config) LoadEnv(env string, config interface{}) error {
	if err := c.loadConfigByEnv(c.opts.deafultConfigFile, config); err != nil {
		return err
	}
	return c.loadConfigByEnv(env, config)
}

func (c *Config) loadConfigByEnv(configName string, config interface{}) error {
	c.viper.SetEnvPrefix(strings.ToUpper("trino_client"))
	c.viper.SetConfigName(configName)
	c.viper.SetConfigType(c.opts.configType)
	c.viper.AddConfigPath(c.opts.configPath)
	c.viper.AutomaticEnv()
	c.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := c.viper.ReadInConfig(); err != nil {
		return err
	}
	return c.viper.Unmarshal(config)
}
