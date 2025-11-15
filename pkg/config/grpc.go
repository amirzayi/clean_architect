package config

import (
	"fmt"
	"time"
)

type GRPCConfig interface {
	Address() string
	MaxReceiveMsgSize() int
	ReadBufferSize() int
	HasReflection() bool
	ShutdownTimeout() time.Duration
}
type grpc struct {
	BindingIPAddress         string `default:"127.0.0.1" json:"bindingIpAddress" yaml:"bindingIpAddress" toml:"bindingIpAddress"`
	Port                     uint   `default:"8070" json:"port" yaml:"port" toml:"port"`
	MaxReceiveMsgSizeInBytes int    `default:"5120" json:"maxReceiveMsgSize" yaml:"maxReceiveMsgSize" toml:"maxReceiveMsgSize"`
	ReadBufferSizeInBytes    int    `default:"5120" json:"readBufferSize" yaml:"readBufferSize" toml:"readBufferSize"`
	HasReflectionFlag        bool   `default:"true" json:"hasReflection" yaml:"hasReflection" toml:"hasReflection"`
	ShutdownTimeoutInSec     uint   `default:"1" json:"shutdownTimeoutInSec" yaml:"shutdownTimeoutInSec" toml:"shutdownTimeoutInSec"`
}

func (g grpc) Address() string {
	return fmt.Sprintf("%s:%d", g.BindingIPAddress, g.Port)
}

func (g grpc) MaxReceiveMsgSize() int {
	return g.MaxReceiveMsgSizeInBytes
}

func (g grpc) ReadBufferSize() int {
	return g.ReadBufferSizeInBytes
}

func (g grpc) HasReflection() bool {
	return g.HasReflectionFlag
}

func (g grpc) ShutdownTimeout() time.Duration {
	return time.Duration(g.ShutdownTimeoutInSec) * time.Second
}
