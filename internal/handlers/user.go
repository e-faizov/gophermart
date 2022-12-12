package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/jwtauth"
	"github.com/rs/zerolog/log"

	"github.com/e-faizov/gophermart/internal/interfaces"
	"github.com/e-faizov/gophermart/internal/models"
	"github.com/e-faizov/gophermart/internal/utils"
)

const userUUID = "user_uuid"

type User struct {
	Store     interfaces.UserStorage
	TokenAuth *jwtauth.JWTAuth
}

func (u *User) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := unmarshalUser(r)
	if err != nil {
		log.Error().Err(err).Msg("User.Register error unmarshal data")
		http.Error(w, "wrong body", http.StatusBadRequest)
		return
	}
	ok, uid, err := u.Store.Register(ctx, user.Login, user.Password)
	if err != nil {
		log.Error().Err(err).Msg("User.Register sql error")
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, "already created", http.StatusConflict)
		return
	}

	token, err := u.token(uid)
	if err != nil {
		log.Error().Err(err).Msg("User.Register error create token")
		u.Logout(w, r)
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	setCookie(w, token)
}

func (u *User) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := unmarshalUser(r)
	if err != nil {
		log.Error().Err(err).Msg("User.Login error unmarshal data")
		u.Logout(w, r)
		http.Error(w, "wrong body", http.StatusBadRequest)
		return
	}
	uid, ok, err := u.Store.Login(ctx, user.Login, user.Password)
	if err != nil {
		log.Error().Err(err).Msg("User.Login error verify user")
		u.Logout(w, r)
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	if !ok {
		u.Logout(w, r)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := u.token(uid)
	if err != nil {
		log.Error().Err(err).Msg("User.Login error create token")
		u.Logout(w, r)
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	setCookie(w, token)
}

func setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    token,
	})
}

func (u *User) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    "",
	})
}

func (u *User) token(user string) (string, error) {
	_, tokenString, err := u.TokenAuth.Encode(map[string]interface{}{userUUID: user})
	return tokenString, err
}

func unmarshalUser(r *http.Request) (models.User, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return models.User{}, utils.ErrorHelper(err)
	}

	var data models.User
	err = json.Unmarshal(body, &data)
	if err != nil {
		return models.User{}, utils.ErrorHelper(err)
	}
	return data, nil
}

func getUserFromReq(r *http.Request) (string, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		return "", err
	}

	ret, ok := claims[userUUID]
	if !ok {
		return "", errors.New("user not found")
	}
	return ret.(string), nil
}
