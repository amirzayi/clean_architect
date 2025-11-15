package config

import "time"

type AuthConfig interface {
	Secret() string
	LifeTime() time.Duration
}

type auth struct {
	SecretKey         string `default:"some_secret" json:"secret" yaml:"secret" toml:"secret"`
	LifeTimeInMinutes int    `default:"1" json:"lifeTime" yaml:"lifeTime" toml:"lifeTime"`
}

func (a auth) Secret() string {
	return a.SecretKey
}

func (a auth) LifeTime() time.Duration {
	return time.Duration(a.LifeTimeInMinutes) * time.Minute
}
