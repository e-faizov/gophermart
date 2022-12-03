package main

import (
	"github.com/e-faizov/gophermart/internal/config"
	"github.com/e-faizov/gophermart/internal/server"
)

func main() {
	cfg := config.GetConfig()

	server.StartServer(cfg)
}
