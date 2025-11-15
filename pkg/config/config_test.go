package config_test

import (
	"testing"

	"github.com/amirzayi/clean_architect/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := config.LoadConfig(" ")
	require.NotNil(t, cfg)
	require.Nil(t, cfg.Auth)
	require.Nil(t, cfg.Cache)
	require.Nil(t, cfg.DB)
	require.Nil(t, cfg.Event)
	require.Nil(t, cfg.GRPC)
	require.Nil(t, cfg.Logger)
	require.Nil(t, cfg.Web)
	require.ErrorIs(t, err, config.ErrorEmptyConfigFilePath)
}

func TestLoadConfigOrDefault(t *testing.T) {
	cfg, err := config.LoadConfigOrDefault(" ")
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Auth)
	require.NotNil(t, cfg.Cache)
	require.NotNil(t, cfg.DB)
	require.NotNil(t, cfg.Event)
	require.NotNil(t, cfg.GRPC)
	require.NotNil(t, cfg.Logger)
	require.NotNil(t, cfg.Web)
	require.Equal(t, cfg.Auth.Secret(), "some_secret")
	require.NoError(t, err)
}
