package server

import (
	"github.com/e-faizov/gophermart/internal/config"
	"github.com/e-faizov/gophermart/internal/handlers"
	"github.com/e-faizov/gophermart/internal/middlewares"
	"github.com/e-faizov/gophermart/internal/storage"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth"
)

var tokenAuth *jwtauth.JWTAuth

const secret = "secret"

func StartServer(cfg config.GopherMartCfg) error {
	db, err := storage.NewPgStore(cfg.DatabaseUri, secret)
	if err != nil {
		panic(err)
	}

	tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)

	r := chi.NewRouter()
	r.Use(middleware.Compress(5))

	userHandlers := handlers.User{
		Store:     db,
		TokenAuth: tokenAuth,
	}

	ordersHandler := handlers.Orders{
		Store: db,
	}

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userHandlers.Register)
		r.Post("/login", userHandlers.Login)
		r.Post("/logout", userHandlers.Logout)

		ar := r.With(jwtauth.Verifier(tokenAuth), middlewares.CheckAuth)

		ar.Post("/orders", ordersHandler.Post)
		ar.Get("/orders", ordersHandler.Get)
		ar.Post("/balance/withdraw", ordersHandler.Withdraw)
		ar.Get("/balance/withdrawals", ordersHandler.Withdrawals)
		ar.Get("/balance", ordersHandler.Balance)
	})

	return http.ListenAndServe(cfg.RunAddress, r)
}
