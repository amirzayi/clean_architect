package config

import (
	"fmt"
	"strings"
)

type SchedulerConfig interface {
	Driver() string
	ConnectionString() string
}

type scheduler struct {
	DriverName string `default:"" json:"driver" yaml:"driver" toml:"driver"`
	IP         string `default:"" json:"ip" yaml:"ip" toml:"ip"`
	Port       uint   `default:"" json:"port" yaml:"port" toml:"port"`
	PrefixName string `default:"" json:"prefix" yaml:"prefix" toml:"prefix"`
	UserName   string `default:"" json:"userName" yaml:"userName" toml:"userName"`
	Password   string `default:"" json:"password" yaml:"password" toml:"password"`
}

func (s scheduler) Driver() string {
	return strings.ToLower(strings.TrimSpace(s.DriverName))
}

func (s scheduler) ConnectionString() string {
	url := fmt.Sprintf("%s:%d", s.IP, s.Port)
	if s.UserName != "" && s.Password != "" {
		url = fmt.Sprintf("%s:%s@%s", s.UserName, s.Password, url)
	}
	switch s.Driver() {
	case "redis":
		return fmt.Sprintf("redis://%s", url)
	default:
		return ""
	}
}
