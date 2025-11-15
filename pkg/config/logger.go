package config

type LoggerConfig interface {
	Level() int
	Directory() string
	FileCreationMode() int
	RemoteURL() string
	Console() bool
}

type logger struct {
	Level1            int    `default:"0" json:"level" yaml:"level" toml:"level"`
	DirectoryPath     string `default:"log" json:"directory" yaml:"directory" toml:"directory"`
	FileCreationMode1 int    `default:"0" json:"fileCreationMode" yaml:"fileCreationMode" toml:"fileCreationMode"`
	RemoteAdrressURL  string `default:"" json:"remoteURL" yaml:"remoteURL" toml:"remoteURL"`
	ConsolePrinter    bool   `default:"true" json:"console" yaml:"console" toml:"console"`
}

func (l logger) Level() int {
	return l.Level1
}

func (l logger) Directory() string {
	return l.DirectoryPath
}

func (l logger) FileCreationMode() int {
	return l.FileCreationMode1
}

func (l logger) RemoteURL() string {
	return l.RemoteAdrressURL
}

func (l logger) Console() bool {
	return l.ConsolePrinter
}
