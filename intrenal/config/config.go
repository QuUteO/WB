package config

import (
	"WB_Service/intrenal/db"
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

type Config struct {
	Env        string            `yaml:"env" env-required:"true"`
	TTl        time.Duration     `yaml:"ttl" env-default:"10s"`
	HTTPConfig HTTP              `yaml:"http" env-required:"true"`
	Postgres   db.PostgresConfig `yaml:"postgres" env-required:"true"`
	Kafka      Kafka             `yaml:"kafka" env-default:"kafka"`
}

type Kafka struct {
	Brokers []string `yaml:"brokers" env-required:"true"`
	Topic   string   `yaml:"topic" env-required:"true"`
	GroupID string   `yaml:"group_id" env-required:"true"`
}

type HTTP struct {
	Address     string        `yaml:"address"`
	Timeout     time.Duration `yaml:"timeout" env-default:"10s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"120s"`
}

func MustLoad() *Config {
	cfgPath := fetchPath()
	if cfgPath == "" {
		panic("config file path is empty")
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		panic("config file not found")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
		panic(err)
	}

	return &cfg
}

func fetchPath() string {
	var res string

	flag.StringVar(&res, "config", "", "config file path is nil")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG")
	}

	return res
}
