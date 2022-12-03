package interfaces

import "context"

type Scores interface {
	GetScore(ctx context.Context, order string) (accrual float64, done bool, fail bool, err error)
}
