package scores

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/e-faizov/gophermart/internal/models"
	"github.com/e-faizov/gophermart/internal/storage"
	"github.com/e-faizov/gophermart/internal/utils"
)

type Scores struct {
	URL string
}

func (s *Scores) GetScore(ctx context.Context, order string) (new models.Order, toManyReq bool, err error) {
	resp, err := http.Get(s.URL + "/api/orders/" + order)
	if err != nil {
		return models.Order{}, false, utils.ErrorHelper(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return models.Order{}, true, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Order{}, false, utils.ErrorHelper(err)
	}

	var scores models.Scores

	fmt.Println(string(body))

	err = json.Unmarshal(body, &scores)
	if err != nil {
		return models.Order{}, false, utils.ErrorHelper(err)
	}

	switch scores.Status {
	case "REGISTERED", "PROCESSING":
		return models.Order{
			Number: order,
			Status: storage.OtProcessing,
		}, false, nil
	case "PROCESSED":
		var acc float64
		if scores.Accrual != nil {
			acc = *scores.Accrual
		}
		return models.Order{
			Number:  order,
			Status:  storage.OtProcessed,
			Accrual: &acc,
		}, false, nil
	case "INVALID":
		return models.Order{
			Number: order,
			Status: storage.OtInvalid,
		}, false, nil
	}

	return models.Order{}, false, errors.New("error")
}
