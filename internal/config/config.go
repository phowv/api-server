package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Host    string        `env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port    int           `env:"SERVER_PORT" env-default:"8080"`
	AppEnv  string        `env:"APP_ENV" env-required:"true"`
	Timeout time.Duration `env:"SERVER_TIMEOUT" env-required:"true"`

	PhotosBucketName string `env:"PHOTOS_BUCKET_NAME" env-default:"photos"`

	DatabaseHost     string `env:"POSTGRES_HOST" env-required:"true"`
	DatabasePort     int    `env:"POSTGRES_PORT" env-required:"true"`
	DatabaseUser     string `env:"POSTGRES_USER" env-required:"true"`
	DatabasePassword string `env:"POSTGRES_PASSWORD" env-required:"true"`
	DatabaseName     string `env:"POSTGRES_DB" env-required:"true"`

	StorageHost     string `env:"MINIO_HOST" env-required:"true"`
	StoragePort     int    `env:"MINIO_DATA_PORT" env-required:"true"`
	StorageUser     string `env:"MINIO_USER" env-required:"true"`
	StoragePassword string `env:"MINIO_PASSWORD" env-required:"true"`
}

func MustLoad() *Config {
	var config Config

	if err := cleanenv.ReadEnv(&config); err != nil {
		log.Fatalf("error reading config file: %s", err)
	}

	return &config
}
