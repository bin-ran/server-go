package managers

import (
	"flag"
	"log/slog"
	"os"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

var (
	Config     BaseConfig
	configFile string
)

type BaseConfig struct {
	Version     string    `toml:"version"`
	Environment string    `toml:"environment"`
	Port        int       `toml:"port" default:"80"`
	HTTPSPort   int       `toml:"https_port" default:"443"`
	WebURL      string    `toml:"webURL"`
	ServerURL   string    `toml:"serverURL"`
	Domain      string    `toml:"domain"`
	PG          DBConfig  `toml:"postgresql"`
	Redis       DBConfig  `toml:"redis"`
	MQ          DBConfig  `toml:"mq"`
	OSS         OSSConfig `toml:"oss"`
}

type DBConfig struct {
	URL      string `toml:"url"`
	Port     int    `toml:"port"`
	Role     string `toml:"role, omitempty"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

func init() {
	flag.StringVar(&configFile, "c", "configurations/dev.toml", "config file of binran")
}

func Environment() {
	flag.Parse()

	file, err := os.Open(configFile)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	if err := toml.NewDecoder(file).Decode(&Config); err != nil {
		panic(err)
	}

	slog.Info("config loaded", "environment", Config.Environment)

	if Config.ServerURL == "" {
		Config.ServerURL = "http://localhost:" + strconv.Itoa(Config.Port)
	}
}
