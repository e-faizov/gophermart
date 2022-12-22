package middlewares

import (
	"context"
	"github.com/e-faizov/gophermart/internal/models"
	"github.com/rs/zerolog/log"
	"net/http"

	"github.com/go-chi/jwtauth"
	"github.com/lestrrat-go/jwx/jwt"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token, claims, err := jwtauth.FromContext(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error get jwt from context")
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		if token == nil || jwt.Validate(token) != nil {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		ret, ok := claims[models.UserUUID]
		if !ok {
			log.Error().Err(err).Msg("error can't find user uuid in jwt")
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		r = r.WithContext(context.WithValue(ctx, models.UUIDKey, ret))

		next.ServeHTTP(w, r)
	})
}
