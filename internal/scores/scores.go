package scores

import (
	"context"
	"errors"
)

type Scores struct {
}

func (s *Scores) GetScore(ctx context.Context, order string) (accrual float64, done bool, fail bool, err error) {
	if order == "12345674" {
		return 500.50, true, false, nil
	}
	return 0, false, false, errors.New("error")
}
