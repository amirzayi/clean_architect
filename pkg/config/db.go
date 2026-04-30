package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type DBConfig interface {
	Driver() string
	ConnectionString() string
}

type db struct {
	DriverName string `default:"sqlite" json:"driver" yaml:"driver" toml:"driver"`
	IP         string `default:"127.0.0.1" json:"ip" yaml:"ip" toml:"ip"`
	Port       uint   `default:"3306" json:"port" yaml:"port" toml:"port"`
	UserName   string `default:"amir" json:"userName" yaml:"userName" toml:"userName"`
	Password   string `default:"mirzaei" json:"password" yaml:"password" toml:"password"`
	Name       string `default:"clean-architect" json:"name" yaml:"name" toml:"name"`
	Path       string `default:"." json:"path" yaml:"path" toml:"path"`
}

func (db db) Driver() string {
	return strings.ToLower(strings.TrimSpace(db.DriverName))
}

func (db db) ConnectionString() string {
	// todo: add mongodb connection string
	switch db.Driver() {
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
			db.UserName,
			db.Password,
			db.IP,
			db.Port,
			db.Name,
		)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			db.UserName,
			db.Password,
			db.IP,
			db.Port,
			db.Name,
		)
	case "sqlite":
		return fmt.Sprintf("%s.sqlite", filepath.Join(db.Path, db.Name))
	default:
		return ""
	}
}
