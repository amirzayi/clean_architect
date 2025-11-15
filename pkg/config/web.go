package config

import (
	"fmt"
	"time"
)

type WebConfig interface {
	Address() string
	ReadTimeOut() time.Duration
	IdleTimeout() time.Duration
	WriteTimeout() time.Duration
	ReadHeaderTimeout() time.Duration
	ShutdownTimeout() time.Duration
}

type web struct {
	BindingIPAddress       string `default:"0.0.0.0" json:"bindingIpAddress" yaml:"bindingIpAddress" toml:"bindingIpAddress"`
	Port                   uint   `default:"8071" json:"port" yaml:"port" toml:"port"`
	ReadTimeOutInSec       uint   `default:"7" json:"readTimeOutInSec" yaml:"readTimeOutInSec" toml:"readTimeOutInSec"`
	IdleTimeoutInSec       uint   `default:"10" json:"idleTimeoutInSec" yaml:"idleTimeoutInSec" toml:"idleTimeoutInSec"`
	WriteTimeoutInSec      uint   `default:"20" json:"writeTimeoutInSec" yaml:"writeTimeoutInSec" toml:"writeTimeoutInSec"`
	ReadHeaderTimeoutInSec uint   `default:"1" json:"readHeaderTimeoutInSec" yaml:"readHeaderTimeoutInSec" toml:"readHeaderTimeoutInSec"`
	ShutdownTimeoutInSec   uint   `default:"1" json:"shutdownTimeoutInSec" yaml:"shutdownTimeoutInSec" toml:"shutdownTimeoutInSec"`
}

func (w web) Address() string {
	return fmt.Sprintf("%s:%d", w.BindingIPAddress, w.Port)
}

func (w web) ReadTimeOut() time.Duration {
	return time.Duration(w.ReadTimeOutInSec) * time.Second
}

func (w web) IdleTimeout() time.Duration {
	return time.Duration(w.IdleTimeoutInSec) * time.Second
}

func (w web) WriteTimeout() time.Duration {
	return time.Duration(w.WriteTimeoutInSec) * time.Second
}

func (w web) ReadHeaderTimeout() time.Duration {
	return time.Duration(w.ReadHeaderTimeoutInSec) * time.Second
}

func (w web) ShutdownTimeout() time.Duration {
	return time.Duration(w.ShutdownTimeoutInSec) * time.Second
}
