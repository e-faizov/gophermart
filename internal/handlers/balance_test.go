package handlers

import (
	"context"
	"github.com/e-faizov/gophermart/internal/interfaces"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRouter(h *Orders) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/balance/withdraw", h.Withdraw)
	r.Get("/balance/withdrawals", h.Withdrawals)
	r.Get("/balance", h.Balance)

	return r
}

func testRequest(request *http.Request, store interfaces.OrdersStorage) *http.Response {
	h := Orders{
		Store: store,
	}
	request = request.WithContext(context.Background())
	w := httptest.NewRecorder()
	testHandle := newRouter(&h)
	testHandle.ServeHTTP(w, request)
	return w.Result()
}

func TestBalanceHandlers(t *testing.T) {

	type want struct {
		statusCode int
	}
	var emptyStore storeTest

	gaugeTestData :=
		`{
"id": "testg",
"type": "gauge"
		}
`

	counterTestData :=
		`{
"id": "testc",
"type": "counter"
		}
`

	tests := []struct {
		name    string
		request string
		body    io.Reader
		method  string
		want    want
	}{
		{
			name:    "unknown type",
			request: "/api/user/balance",
			method:  http.MethodPost,
			want: want{
				statusCode: http.StatusNotImplemented,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			result := testRequest(request, &emptyStore)

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			err := result.Body.Close()
			require.NoError(t, err)
		})
	}

}
