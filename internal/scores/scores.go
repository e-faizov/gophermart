package scores

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/e-faizov/gophermart/internal/models"
	"github.com/e-faizov/gophermart/internal/utils"
	"io"
	"net/http"
)

type Scores struct {
}

func (s *Scores) GetScore(ctx context.Context, order string) (accrual float64, done bool, fail bool, err error) {
	resp, err := http.Get("localhost:3000/api/order/" + order)
	if err != nil {
		return 0, false, false, utils.ErrorHelper(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, false, utils.ErrorHelper(err)
	}

	var scores models.Scores

	err = json.Unmarshal(body, &scores)
	if err != nil {
		return 0, false, false, utils.ErrorHelper(err)
	}

	switch scores.Status {
	case "REGISTERED", "PROCESSING":
		return 0, false, false, nil
	case "PROCESSED":
		var acc float64
		if scores.Accrual != nil {
			acc = *scores.Accrual
		}
		return acc, true, false, nil
	case "INVALID":
		return 0, false, true, nil
	}

	return 0, false, false, errors.New("error")
}
