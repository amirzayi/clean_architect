package config

import "fmt"

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
}

func (c cache) Prefix() string {
	return c.PrefixName
}

func (c cache) Driver() string {
	return c.DriverName
}

func (c cache) ConnectionString() string {
	url := fmt.Sprintf("%s:%d", c.IP, c.Port)
	if c.UserName != "" && c.Password != "" {
		url = fmt.Sprintf("%s:%s@%s", c.UserName, c.Password, url)
	}
	switch c.DriverName {
	case "redis":
		return fmt.Sprintf("redis://%s", url)
	case "memcached":
		return url
	default:
		return ""
	}
}
