package config

type LoggerConfig interface {
	Level() int
	Directory() string
	FileCreationMode() int
	RemoteURL() string
	Console() bool
}

type logger struct {
	LogLevel             int    `default:"0" json:"level" yaml:"level" toml:"level"`
	DirectoryPath        string `default:"log" json:"directory" yaml:"directory" toml:"directory"`
	FileCreationModeType int    `default:"0" json:"fileCreationMode" yaml:"fileCreationMode" toml:"fileCreationMode"`
	RemoteAdrressURL     string `default:"" json:"remoteURL" yaml:"remoteURL" toml:"remoteURL"`
	ConsolePrinter       bool   `default:"true" json:"console" yaml:"console" toml:"console"`
}

func (l logger) Level() int {
	return l.LogLevel * 4
}

func (l logger) Directory() string {
	return l.DirectoryPath
}

func (l logger) FileCreationMode() int {
	return l.FileCreationModeType
}

func (l logger) RemoteURL() string {
	return l.RemoteAdrressURL
}

func (l logger) Console() bool {
	return l.ConsolePrinter
}
