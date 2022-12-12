package interfaces

import (
	"context"

	"github.com/e-faizov/gophermart/internal/models"
)

type Scores interface {
	GetScore(ctx context.Context, order string) (new models.Order, toManyReq bool, err error)
}
