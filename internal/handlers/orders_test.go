package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/e-faizov/gophermart/internal/models"
	"github.com/e-faizov/gophermart/internal/storage"
)

func newOrderRouter(h *Orders) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/user/orders", h.Get)
	r.Post("/api/user/orders", h.Post)

	return r
}

func TestOrdersGetHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/user/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	tStore := &testOrdersStore{}

	handlers := &Orders{
		Store: tStore,
	}

	testRouter := newOrderRouter(handlers)

	t.Run("OK", func(t *testing.T) {
		acc := float64(103.2)
		tm := time.Now()
		data := []models.Order{
			{
				Number:   "1",
				Status:   storage.OtNew,
				Uploaded: tm,
			},
			{
				Number:   "2",
				Status:   storage.OtProcessing,
				Uploaded: tm,
			},
			{
				Number:   "3",
				Status:   storage.OtInvalid,
				Uploaded: tm,
			},
			{
				Number:   "3",
				Status:   storage.OtProcessed,
				Accrual:  &acc,
				Uploaded: tm,
			},
		}
		tStore.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		tStore.getOrders = func(ctx context.Context, user string) ([]models.Order, error) {
			return data, nil
		}
		wr := serveHttp(testRouter, req)

		if wr.Code != http.StatusOK {
			t.Error("error, code not 200, code:", wr.Code)
			return
		}

		var res []models.Order
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

		if len(res) != 4 {
			t.Error("wrong response len", data, res)
			return
		}

		if res[0].Number != data[0].Number && res[0].Status != data[0].Status && res[0].Uploaded != data[0].Uploaded {
			t.Error("wrong response", data, res)
			return
		}

		if res[1].Number != data[1].Number && res[1].Status != data[1].Status && res[1].Uploaded != data[1].Uploaded {
			t.Error("wrong response", data, res)
			return
		}

		if res[2].Number != data[2].Number && res[2].Status != data[2].Status && res[2].Uploaded != data[2].Uploaded {
			t.Error("wrong response", data, res)
			return
		}

		if res[3].Number != data[3].Number && res[3].Status != data[3].Status && res[3].Uploaded != data[3].Uploaded && *res[3].Accrual != *data[3].Accrual {
			t.Error("wrong response", data, res)
			return
		}
	})

	t.Run("DBError", func(t *testing.T) {
		tStore.Clear()
		req = req.WithContext(contextWithJwt(context.Background(), "test user"))
		tStore.getOrders = func(ctx context.Context, user string) ([]models.Order, error) {
			return nil, errors.New("db error")
		}
		wr := serveHttp(testRouter, req)

		if wr.Code != http.StatusInternalServerError {
			t.Error("error, code not 500, code:", wr.Code)
			return
		}
	})

	t.Run("WithoutJwt", withoutJwtTestFunc(req, testRouter))
}

type testOrdersStore struct {
	saveOrder           func(ctx context.Context, user, order string) (inserted bool, thisUser bool, err error)
	getOrders           func(ctx context.Context, user string) ([]models.Order, error)
	getOrderIdsByStatus func(ctx context.Context, status string) ([]string, error)
	updateOrder         func(ctx context.Context, order models.Order) error
}

func (t *testOrdersStore) Clear() {
	t.saveOrder = nil
	t.getOrders = nil
	t.getOrderIdsByStatus = nil
	t.updateOrder = nil
}

func (t *testOrdersStore) SaveOrder(ctx context.Context, user, order string) (inserted bool, thisUser bool, err error) {
	if t.saveOrder != nil {
		return t.saveOrder(ctx, user, order)
	}
	return false, false, nil
}
func (t *testOrdersStore) GetOrders(ctx context.Context, user string) ([]models.Order, error) {
	if t.getOrders != nil {
		return t.getOrders(ctx, user)
	}
	return nil, nil
}
func (t *testOrdersStore) GetOrderIdsByStatus(ctx context.Context, status string) ([]string, error) {
	if t.getOrderIdsByStatus != nil {
		return t.getOrderIdsByStatus(ctx, status)
	}
	return nil, nil
}
func (t *testOrdersStore) UpdateOrder(ctx context.Context, order models.Order) error {
	if t.updateOrder != nil {
		return t.updateOrder(ctx, order)
	}
	return nil
}
