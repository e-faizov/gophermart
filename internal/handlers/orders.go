package handlers

import (
	"github.com/e-faizov/gophermart/internal/interfaces"
	"github.com/go-chi/render"
	"github.com/joeljunstrom/go-luhn"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
)

type Orders struct {
	Store interfaces.OrdersStorage
}

func (o *Orders) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Post error read body")
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	userId, err := getUserFromReq(r)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Post error get user from request")
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	id := string(b)

	if !luhn.Valid(id) {
		log.Error().Err(err).Msg("Orders.Post error not luhn")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	inserted, thisUser, err := o.Store.SaveOrder(ctx, userId, id)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Post error save order number")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if !inserted && !thisUser {
		log.Error().Err(err).Msg("Orders.Post error already inserted")
		http.Error(w, "", http.StatusConflict)
		return
	}

	if inserted {
		http.Error(w, "", http.StatusAccepted)
	}
}

func (o *Orders) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userId, err := getUserFromReq(r)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Get error get user from request")
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	orders, err := o.Store.GetOrders(ctx, userId)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Get error get orders")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	render.JSON(w, r, orders)
}
