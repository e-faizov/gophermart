package main

import (
	"github.com/rs/zerolog/log"
	
	"github.com/e-faizov/gophermart/internal/config"
	"github.com/e-faizov/gophermart/internal/server"
)

func main() {
	cfg := config.GetConfig()

	err := server.StartServer(cfg)
	log.Error().Err(err).Msg("fail start server")
}
