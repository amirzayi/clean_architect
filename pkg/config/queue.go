package config

import (
	"fmt"
	"strings"
	"time"
)

type QueueConfig interface {
	Driver() string
	ConnectionString() string
	RunEvery() time.Duration
}

type queue struct {
	DriverName           string `default:"" json:"driver" yaml:"driver" toml:"driver"`
	IP                   string `default:"" json:"ip" yaml:"ip" toml:"ip"`
	Port                 uint   `default:"" json:"port" yaml:"port" toml:"port"`
	UserName             string `default:"" json:"userName" yaml:"userName" toml:"userName"`
	Password             string `default:"" json:"password" yaml:"password" toml:"password"`
	DBNumber             uint   `default:"" json:"db" yaml:"db" toml:"db"`
	RunEveryMilliSeconds int    `default:"100" json:"run_every_milliseconds" yaml:"run_every_milliseconds" toml:"run_every_milliseconds"`
}

func (q queue) Driver() string {
	return strings.ToLower(strings.TrimSpace(q.DriverName))
}

func (q queue) ConnectionString() string {
	switch q.Driver() {
	case "redis":
		return getRedisURL(q.IP, fmt.Sprint(q.Port), q.UserName, q.Password, fmt.Sprint(q.DBNumber))
	default:
		return ""
	}
}

func (q queue) RunEvery() time.Duration {
	return time.Millisecond * time.Duration(q.RunEveryMilliSeconds)
}
