// Package config provides config settings and launch flags
package config

import (
	"errors"
	"flag"

	"github.com/caarlos0/env/v11"
)

const (
	addressDefault        = "localhost:8080"
	accrualAddressDefault = "http://localhost:8090"
	databaseURIDefault    = "postgresql://postgres:12345@localhost:5432/postgres?sslmode=disable"
)

var (
	runAddress  = flag.String("a", addressDefault, "server adress")
	databaseURI = flag.String("d", databaseURIDefault, "database uri")
	accrualAddr = flag.String("r", accrualAddressDefault, "accrual system address")
)

type Config struct {
	RunAddress           string `envDefault:""`
	DatabaseURI          string `envDefault:""`
	AccrualSystemAddress string `envDefault:""`
	Level                string `envDefault:""`
	PollInterval         int    `envDefault:"2"` // in seconds
}

func InitConfig() (Config, error) {
	cfg := Config{}
	opts := env.Options{UseFieldNameByDefault: true}
	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		return cfg, err
	}
	flag.Parse()
	if len(flag.Args()) > 0 {
		return cfg, errors.New("too many arguments")
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

	if *accrualAddr != accrualAddressDefault {
		cfg.AccrualSystemAddress = *accrualAddr
	} else if cfg.AccrualSystemAddress == "" {
		cfg.AccrualSystemAddress = accrualAddressDefault
	}

	return cfg, nil
}
