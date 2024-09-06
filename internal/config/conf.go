package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v11"
)

const (
	addressDefault     = "localhost:8080"
	databaseURIDefault = "postgresql://postgres:12345@localhost:5432/postgres?sslmode=disable"
)

var (
	runAddress  = flag.String("a", addressDefault, "server adress")
	databaseURI = flag.String("d", databaseURIDefault, "database uri")
)

type Config struct {
	RunAddress  string `envDefault:""`
	DatabaseURI string `envDefault:""`
}

func InitConfig() Config {
	cfg := Config{}
	opts := env.Options{UseFieldNameByDefault: true}
	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		log.Fatalf("Error parsing env: %v", err)
	}
	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatal("Too many arguments")
	}
	if *runAddress != addressDefault {
		cfg.RunAddress = *runAddress
	} else if cfg.RunAddress == "" {
		cfg.RunAddress = addressDefault
	}
	if *databaseURI != databaseURIDefault {
		cfg.DatabaseURI = *databaseURI
	} else if cfg.DatabaseURI == "" {
		cfg.DatabaseURI = databaseURIDefault
	}

	return cfg
}
