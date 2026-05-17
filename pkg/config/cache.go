package config

import (
	"fmt"
	"strings"
)

type CacheConfig interface {
	Prefix() string
	Driver() string
	ConnectionString() string
}

type cache struct {
	DriverName string `default:"" json:"driver" yaml:"driver" toml:"driver"`
	IP         string `default:"" json:"ip" yaml:"ip" toml:"ip"`
	Port       uint   `default:"" json:"port" yaml:"port" toml:"port"`
	PrefixName string `default:"" json:"prefix" yaml:"prefix" toml:"prefix"`
	UserName   string `default:"" json:"userName" yaml:"userName" toml:"userName"`
	Password   string `default:"" json:"password" yaml:"password" toml:"password"`
	DBNumber   uint   `default:"" json:"db" yaml:"db" toml:"db"`
}

func (c cache) Prefix() string {
	return c.PrefixName
}

func (c cache) Driver() string {
	return strings.ToLower(strings.TrimSpace(c.DriverName))
}

func (c cache) ConnectionString() string {
	url := fmt.Sprintf("%s:%d", c.IP, c.Port)
	if c.UserName != "" && c.Password != "" {
		url = fmt.Sprintf("%s:%s@%s", c.UserName, c.Password, url)
	}
	switch c.Driver() {
	case "redis":
		return getRedisURL(c.IP, fmt.Sprint(c.Port), c.UserName, c.Password, fmt.Sprint(c.DBNumber))
	case "memcached":
		return url
	default:
		return ""
	}
}
