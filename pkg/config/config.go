// Package config provides configuration management for various services
// such as database, grpc server, http server, etc.
// It reads configuration from file and make available to other parts of application.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

var ErrorEmptyConfigFilePath = errors.New("empty config file path")

// AppConfig holds the configurations for the entire application, including
// db, web and server, logger, auth mechanism, caching and event driven configurations.
type AppConfig struct {
	DB        DBConfig
	Web       WebConfig
	GRPC      GRPCConfig
	Logger    LoggerConfig
	Auth      AuthConfig
	Cache     CacheConfig
	Event     EventConfig
	Scheduler SchedulerConfig
}

// appConfig holds the configurations for the entire application, including
// db, web server, and grpc server configurations.
// It should have Exported fields to work with tags.
type appConfig struct {
	DB        db        `json:"db" yaml:"db" toml:"db"`
	Web       web       `json:"web" yaml:"web" toml:"web"`
	GRPC      grpc      `json:"grpc" yaml:"grpc" toml:"grpc"`
	Logger    logger    `json:"logger" yaml:"logger" toml:"logger"`
	Auth      auth      `json:"auth" yaml:"auth" toml:"auth"`
	Cache     cache     `json:"cache" yaml:"cache" toml:"cache"`
	Event     event     `json:"event" yaml:"event" toml:"event"`
	Scheduler scheduler `json:"scheduler" yaml:"scheduler" toml:"scheduler"`
}

// LoadConfig will return AppConfig which that values are filled by given config file's address.
func LoadConfig(configFilePath string) (AppConfig, error) {
	cfg, err := loadConfig(configFilePath)
	if err != nil {
		return AppConfig{}, err
	}
	return cfg.toAppConfig(), nil
}

func loadConfig(configFilePath string) (appConfig, error) {
	var cfg appConfig

	if strings.TrimSpace(configFilePath) == "" {
		return cfg, ErrorEmptyConfigFilePath
	}

	bytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return cfg, err
	}

	switch filepath.Ext(configFilePath) {
	case ".json":
		err = json.Unmarshal(bytes, &cfg)
	case ".yml", ".yaml":
		err = yaml.Unmarshal(bytes, &cfg)
	case ".toml":
		err = toml.Unmarshal(bytes, &cfg)
	default:
		err = errors.New("unsupported config's file type")
	}

	return cfg, err
}

func (cfg appConfig) toAppConfig() AppConfig {
	return AppConfig{
		DB:        cfg.DB,
		Web:       cfg.Web,
		GRPC:      cfg.GRPC,
		Logger:    cfg.Logger,
		Auth:      cfg.Auth,
		Cache:     cfg.Cache,
		Event:     cfg.Event,
		Scheduler: cfg.Scheduler,
	}
}

// LoadConfigOrDefault will do LoadConfig. if loading had problem, then returns default values config.
func LoadConfigOrDefault(fileAddress string) (AppConfig, error) {
	cfg, err := loadConfig(fileAddress)
	if err != nil {
		err = defaults.Set(&cfg)
	}
	return cfg.toAppConfig(), err
}
