package config

import (
	"fmt"
	"strings"
)

type EventConfig interface {
	Driver() string
	ConnectionString() string
}

type event struct {
	DriverName string `default:"" json:"driver" yaml:"driver" toml:"driver"`
	IP         string `default:"" json:"ip" yaml:"ip" toml:"ip"`
	Port       uint   `default:"" json:"port" yaml:"port" toml:"port"`
	UserName   string `default:"" json:"userName" yaml:"userName" toml:"userName"`
	Password   string `default:"" json:"password" yaml:"password" toml:"password"`
	DBNumber   uint   `default:"" json:"db" yaml:"db" toml:"db"`
}

func (e event) Driver() string {
	return strings.ToLower(strings.TrimSpace(e.DriverName))
}

func (e event) ConnectionString() string {
	url := fmt.Sprintf("%s:%d", e.IP, e.Port)
	if e.UserName != "" && e.Password != "" {
		url = fmt.Sprintf("%s:%s@%s", e.UserName, e.Password, url)
	}
	switch e.Driver() {
	case "rabbitmq":
		return fmt.Sprintf("amqp://%s", url)
	case "redis":
		return getRedisURL(e.IP, fmt.Sprint(e.Port), e.UserName, e.Password, fmt.Sprint(e.DBNumber))
	case "nats":
		return fmt.Sprintf("nats://%s", url)
	default:
		return ""
	}
}
