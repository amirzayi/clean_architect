package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type SchedulerConfig interface {
	Driver() string
	ConnectionString() string
	Concurrency() int
	RunEvery() time.Duration
}

type scheduler struct {
	DriverName       string `default:"" json:"driver" yaml:"driver" toml:"driver"`
	IP               string `default:"" json:"ip" yaml:"ip" toml:"ip"`
	Port             uint   `default:"" json:"port" yaml:"port" toml:"port"`
	PrefixName       string `default:"" json:"prefix" yaml:"prefix" toml:"prefix"`
	UserName         string `default:"" json:"userName" yaml:"userName" toml:"userName"`
	Password         string `default:"" json:"password" yaml:"password" toml:"password"`
	Name             string `default:"clean-architect" json:"name" yaml:"name" toml:"name"`
	Path             string `default:"." json:"path" yaml:"path" toml:"path"`
	ConcurrencyLevel int    `default:"." json:"concurrency" yaml:"concurrency" toml:"concurrency"`
	RunEverySeconds  int    `default:"." json:"run_every_seconds" yaml:"run_every_seconds" toml:"run_every_seconds"`
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
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
			s.UserName,
			s.Password,
			s.IP,
			s.Port,
			s.Name,
		)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			s.UserName,
			s.Password,
			s.IP,
			s.Port,
			s.Name,
		)
	case "sqlite":
		return fmt.Sprintf("%s.sqlite", filepath.Join(s.Path, s.Name))
	default:
		return ""
	}
}

func (s scheduler) Concurrency() int {
	if s.ConcurrencyLevel < 1 {
		return 1
	}
	return s.ConcurrencyLevel
}
func (s scheduler) RunEvery() time.Duration {
	if s.RunEverySeconds < 1 {
		return time.Second
	}
	return time.Second * time.Duration(s.RunEverySeconds)
}
