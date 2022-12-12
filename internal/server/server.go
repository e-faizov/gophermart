package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth"

	"github.com/e-faizov/gophermart/internal/config"
	"github.com/e-faizov/gophermart/internal/handlers"
	"github.com/e-faizov/gophermart/internal/middlewares"
	"github.com/e-faizov/gophermart/internal/scores"
	"github.com/e-faizov/gophermart/internal/storage"
	"github.com/e-faizov/gophermart/internal/updater"
)

var tokenAuth *jwtauth.JWTAuth

const secret = "secret"

func StartServer(cfg config.GopherMartCfg) error {
	db, err := storage.NewPgStore(cfg.DatabaseURI, secret)
	if err != nil {
		panic(err)
	}

	tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)

	userHandlers := handlers.User{
		Store:     db,
		TokenAuth: tokenAuth,
	}

	ordersHandler := handlers.Orders{
		Store: db,
	}

	balancesHandler := handlers.Balances{
		Store: db,
	}

	scoresServ := scores.Scores{
		Url: cfg.AccrualSystemAddress,
	}

	orderUpdater := updater.OrderUpdater{
		Store:  db,
		Scores: &scoresServ,
	}

	orderUpdater.Start()

	r := chi.NewRouter()
	r.Use(middleware.Compress(5))

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userHandlers.Register)
		r.Post("/login", userHandlers.Login)
		r.Post("/logout", userHandlers.Logout)

		ar := r.With(jwtauth.Verifier(tokenAuth), middlewares.CheckAuth)

		ar.Post("/orders", ordersHandler.Post)
		ar.Get("/orders", ordersHandler.Get)
		ar.Post("/balance/withdraw", balancesHandler.Withdraw)
		ar.Get("/balance/withdrawals", balancesHandler.Withdrawals)
		ar.Get("/balance", balancesHandler.Balance)
	})

	return http.ListenAndServe(cfg.RunAddress, r)
}
