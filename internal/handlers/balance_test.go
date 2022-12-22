package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"

	"github.com/e-faizov/gophermart/internal/middlewares"
	"github.com/e-faizov/gophermart/internal/models"
)

func newBalanceRouter(h *Balances) *chi.Mux {
	r := chi.NewRouter()
	ra := r.With(middlewares.Auth)
	ra.Get("/api/user/balance", h.Balance)
	ra.Post("/api/user/balance/withdraw", h.Withdraw)
	ra.Get("/api/user/withdrawals", h.Withdrawals)

	return r
}

func contextWithJwt(ctx context.Context, user string) context.Context {
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil)
	token, _, _ := tokenAuth.Encode(map[string]interface{}{models.UserUUID: user})
	return context.WithValue(ctx, jwtauth.TokenCtxKey, token)
}

func TestWithdrawalsHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/user/withdrawals", nil)
	if err != nil {
		t.Fatal(err)
	}

	tStore := &testBalanceStore{}

	bl := Balances{
		Store: tStore,
	}

	testRouter := newBalanceRouter(&bl)

	t.Run("OK", func(t *testing.T) {
		tm := time.Now()
		data := []models.Withdraw{
			{
				Order:     "1233143",
				Sum:       123.01,
				Processed: tm,
			},
			{
				Order:     "12331",
				Sum:       1.01,
				Processed: tm,
			},
		}

		tStore.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		tStore.withdrawalsByUserFunc = func(ctx context.Context, uuid string) ([]models.Withdraw, error) {
			return data, nil
		}
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusOK {
			t.Fatal("error, code not 200, code:", wr.Code)
		}

		var res []models.Withdraw
		body, err := io.ReadAll(wr.Body)
		if err != nil {
			t.Error("response error read body", err)
			return
		}

		err = json.Unmarshal(body, &res)
		if err != nil {
			t.Error("response body not json", err)
			return
		}

		if len(res) != 2 {
			t.Error("wrong response len", data, res)
			return
		}

		if res[0].Order != data[0].Order && res[0].Processed != data[0].Processed && res[0].Sum != data[0].Sum &&
			res[1].Order != data[1].Order && res[1].Processed != data[1].Processed && res[1].Sum != data[1].Sum {
			t.Error("wrong response", data, res)
			return
		}
	})

	t.Run("EmptyResult", func(t *testing.T) {
		tStore.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		tStore.withdrawalsByUserFunc = func(ctx context.Context, uuid string) ([]models.Withdraw, error) {
			return []models.Withdraw{}, nil
		}
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusNoContent {
			t.Fatal("error, code not 204, code:", wr.Code)
		}
	})

	t.Run("UserNotFound", func(t *testing.T) {
		tStore.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		tStore.withdrawalsByUserFunc = func(ctx context.Context, uuid string) ([]models.Withdraw, error) {
			return nil, errors.New("user not found")
		}
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusInternalServerError {
			t.Fatal("error, code not 500, code:", wr.Code)
		}
	})

	t.Run("WithoutJwt", withoutJwtTestFunc(req, testRouter))
}

func TestWithdrawHandler(t *testing.T) {
	method := "POST"
	path := "/api/user/balance/withdraw"

	tStore := &testBalanceStore{}

	bl := Balances{
		Store: tStore,
	}

	testRouter := newBalanceRouter(&bl)

	t.Run("OK", func(t *testing.T) {
		tStore.Clear()
		req, err := http.NewRequest(method, path, strings.NewReader("{\"order\":\"176081\", \"sum\":1.0}"))
		if err != nil {
			t.Fatal(err)
		}

		tStore.withdrawFunc = func(ctx context.Context, withdraw models.Withdraw, uuid string) (notEnough bool, err error) {
			return false, nil
		}

		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusOK {
			t.Fatal("error, code not 200, code:", wr.Code)
		}
	})

	t.Run("DBError", func(t *testing.T) {
		tStore.Clear()
		req, err := http.NewRequest(method, path, strings.NewReader("{\"order\":\"176081\", \"sum\":1.0}"))
		if err != nil {
			t.Fatal(err)
		}

		tStore.withdrawFunc = func(ctx context.Context, withdraw models.Withdraw, uuid string) (notEnough bool, err error) {
			return false, errors.New("error")
		}

		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusInternalServerError {
			t.Fatal("error, code not 500, code:", wr.Code)
		}
	})

	t.Run("notEnough", func(t *testing.T) {
		tStore.Clear()
		req, err := http.NewRequest(method, path, strings.NewReader("{\"order\":\"176081\", \"sum\":1.0}"))
		if err != nil {
			t.Fatal(err)
		}

		tStore.withdrawFunc = func(ctx context.Context, withdraw models.Withdraw, uuid string) (notEnough bool, err error) {
			return true, nil
		}

		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusPaymentRequired {
			t.Fatal("error, code not 402, code:", wr.Code)
		}
	})

	t.Run("notLuhn", func(t *testing.T) {
		tStore.Clear()
		req, err := http.NewRequest(method, path, strings.NewReader("{\"order\":\"123\"}"))
		if err != nil {
			t.Fatal(err)
		}
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusUnprocessableEntity {
			t.Fatal("error, code not 422, code:", wr.Code)
		}
	})

	t.Run("notJSON", func(t *testing.T) {
		tStore.Clear()
		req, err := http.NewRequest(method, path, strings.NewReader("{)"))
		if err != nil {
			t.Fatal(err)
		}
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusInternalServerError {
			t.Fatal("error, code not 500, code:", wr.Code)
		}
	})

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("WithoutJwt", withoutJwtTestFunc(req, testRouter))
}

func TestBalanceHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/user/balance", nil)
	if err != nil {
		t.Fatal(err)
	}

	tStore := &testBalanceStore{}

	bl := Balances{
		Store: tStore,
	}

	testRouter := newBalanceRouter(&bl)

	t.Run("OK", func(t *testing.T) {
		tStore.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))

		data := models.Balance{
			Current:   100.0,
			Withdrawn: 50.5,
		}
		tStore.balanceByUserFunc = func(ctx context.Context, uuid string) (models.Balance, error) {
			return data, nil
		}
		wr := serveHTTP(testRouter, req)

		if wr.Code != http.StatusOK {
			t.Error("error, code not 200, code:", wr.Code)
			return
		}

		var res models.Balance
		body, err := io.ReadAll(wr.Body)
		if err != nil {
			t.Error("response error read body", err)
			return
		}

		err = json.Unmarshal(body, &res)
		if err != nil {
			t.Error("response body not json", err)
			return
		}

		if res.Current != data.Current && res.Withdrawn != data.Withdrawn {
			t.Error("wrong response", data, res)
			return
		}
	})

	t.Run("UserNotFound", userNotFoundTestFunc(tStore, req, testRouter))

	t.Run("WithoutJwt", withoutJwtTestFunc(req, testRouter))
}

func userNotFoundTestFunc(store *testBalanceStore, req *http.Request, rt *chi.Mux) func(t *testing.T) {
	return func(t *testing.T) {
		store.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		store.balanceByUserFunc = func(ctx context.Context, uuid string) (models.Balance, error) {
			return models.Balance{}, errors.New("user not found")
		}
		wr := serveHTTP(rt, req)

		if wr.Code != http.StatusInternalServerError {
			t.Fatal("error, code not 500, code:", wr.Code)
		}
	}
}

func withoutJwtTestFunc(req *http.Request, rt *chi.Mux) func(t *testing.T) {
	return func(t *testing.T) {
		req = req.WithContext(context.Background())
		wr := serveHTTP(rt, req)

		if wr.Code != http.StatusUnauthorized {
			t.Error("error, code not 401, code:", wr.Code)
		}
	}
}

func serveHTTP(handler *chi.Mux, req *http.Request) *httptest.ResponseRecorder {
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, req)
	return wr
}

type testBalanceStore struct {
	withdrawFunc          func(ctx context.Context, withdraw models.Withdraw, uuid string) (notEnough bool, err error)
	withdrawalsByUserFunc func(ctx context.Context, uuid string) ([]models.Withdraw, error)
	balanceByUserFunc     func(ctx context.Context, uuid string) (models.Balance, error)
}

func (t *testBalanceStore) Clear() {
	t.withdrawFunc = nil
	t.withdrawalsByUserFunc = nil
	t.balanceByUserFunc = nil
}

func (t *testBalanceStore) Withdraw(ctx context.Context, withdraw models.Withdraw, uuid string) (notEnough bool, err error) {
	if t.withdrawFunc != nil {
		return t.withdrawFunc(ctx, withdraw, uuid)
	}
	return false, nil
}
func (t *testBalanceStore) WithdrawalsByUser(ctx context.Context, uuid string) ([]models.Withdraw, error) {
	if t.withdrawalsByUserFunc != nil {
		return t.withdrawalsByUserFunc(ctx, uuid)
	}
	return nil, nil
}
func (t *testBalanceStore) BalanceByUser(ctx context.Context, uuid string) (models.Balance, error) {
	if t.balanceByUserFunc != nil {
		return t.balanceByUserFunc(ctx, uuid)
	}
	return models.Balance{}, nil
}
