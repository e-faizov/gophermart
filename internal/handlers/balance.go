package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/render"
	"github.com/joeljunstrom/go-luhn"
	"github.com/rs/zerolog/log"

	"github.com/e-faizov/gophermart/internal/interfaces"
	"github.com/e-faizov/gophermart/internal/models"
)

type Balances struct {
	Store interfaces.BalanceStorage
}

func (b *Balances) Balance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := ctx.Value(models.UUIDKey).(string)

	res, err := b.Store.BalanceByUser(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Balance error get balance by user")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, res)
}

func (b *Balances) Withdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := ctx.Value(models.UUIDKey).(string)

	withdrawals, err := b.Store.WithdrawalsByUser(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Withdrawals error WithdrawalsByUser")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		http.Error(w, "", http.StatusNoContent)
	}

	render.JSON(w, r, withdrawals)
}

func (b *Balances) Withdraw(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := ctx.Value(models.UUIDKey).(string)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Withdraw error read body")
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	var withdraw models.Withdraw
	err = json.Unmarshal(body, &withdraw)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Withdraw error unmarshal body")
		http.Error(w, "wrong body", http.StatusInternalServerError)
		return
	}

	if !luhn.Valid(withdraw.Order) {
		log.Error().Err(err).Msg("Orders.Withdraw error not luhn")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	notEnough, err := b.Store.Withdraw(ctx, withdraw, userID)
	if err != nil {
		log.Error().Err(err).Msg("Orders.Withdraw error withdraw")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	if notEnough {
		http.Error(w, "", http.StatusPaymentRequired)
		return
	}
}
