package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Database struct {
	Host           string        `env:"DB_HOST" envDefault:"localhost"`
	Port           uint16        `env:"DB_PORT" envDefault:"5432"`
	User           string        `env:"DB_USER" envDefault:"notifications"`
	Password       string        `env:"DB_PASSWORD" envDefault:"notifications"`
	Name           string        `env:"DB_NAME" envDefault:"notifications"`
	SSLMode        string        `env:"DB_SSLMODE" envDefault:"disable"`
	ConnectTimeout time.Duration `env:"DB_CONNECT_TIMEOUT" envDefault:"10s"`
}

type MessageBird struct {
	BaseURL    string        `env:"MESSAGEBIRD_BASE_URL,notEmpty"`
	AccessKey  string        `env:"MESSAGEBIRD_ACCESS_KEY,notEmpty"`
	Originator string        `env:"MESSAGEBIRD_ORIGINATOR" envDefault:"ACME"`
	Timeout    time.Duration `env:"MESSAGEBIRD_TIMEOUT" envDefault:"10s"`
}

type Config struct {
	HTTPAddr    string `env:"HTTP_ADDR" envDefault:":8080"`
	Database    Database
	MessageBird MessageBird
}

func Load() (Config, error) {
	return load[Config](nil)
}

func LoadDatabase() (Database, error) {
	return load[Database](nil)
}

func load[T any](environment map[string]string) (T, error) {
	return env.ParseAsWithOptions[T](env.Options{Environment: environment})
}
