package updater

import (
	"context"

	"github.com/e-faizov/gophermart/internal/interfaces"
	"github.com/e-faizov/gophermart/internal/storage"
)

type StatusUpdater struct {
	Scores interfaces.Scores
	Store  interfaces.OrdersStorage
}

func (s *StatusUpdater) Start() {
	go s.worker()
}

func (s *StatusUpdater) worker() {
	ctx := context.Background()
	for {
		s.update(ctx)
	}
}

func (s *StatusUpdater) update(ctx context.Context) error {
	newOrders, err := s.Store.GetOrderIdsByStatus(ctx, storage.OtNew)
	if err != nil {
		return err
	}

	for _, order := range newOrders {
		_, done, fail, err := s.Scores.GetScore(ctx, order)
		if err != nil {
			return err
		}

		if !done && !fail {
			err = s.Store.UpdateOrderStatus(ctx, order, storage.OtProcessing)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
