package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

type GopherMartCfg struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	DatabaseUri          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

var (
	cfg    GopherMartCfg
	inited bool
)

func GetConfig() GopherMartCfg {
	if !inited {
		flag.StringVar(&(cfg.RunAddress), "a", "localhost:8081", "RUN_ADDRESS")
		flag.StringVar(&(cfg.DatabaseUri), "d", "", "DATABASE_URI")
		flag.StringVar(&(cfg.AccrualSystemAddress), "r", "", "ACCRUAL_SYSTEM_ADDRESS")

		flag.Parse()
		if err := env.Parse(&cfg); err != nil {
			panic(err)
		}
		inited = true
	}
	return cfg
}
