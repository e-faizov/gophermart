package updater

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/e-faizov/gophermart/internal/interfaces"
	"github.com/e-faizov/gophermart/internal/storage"
)

type OrderUpdater struct {
	Scores interfaces.Scores
	Store  interfaces.OrdersStorage
}

func (s *OrderUpdater) Start() {
	go s.worker()
}

func (s *OrderUpdater) worker() {
	ctx := context.Background()
	for {
		toManyReq, err := s.update(ctx, storage.OtNew)
		if err != nil {
			log.Error().Err(err).Msg("OrderUpdater.worker error update " + storage.OtNew)
			time.Sleep(time.Second)
			continue
		}

		if toManyReq {
			time.Sleep(time.Minute)
			continue
		}

		toManyReq, err = s.update(ctx, storage.OtProcessing)
		if err != nil {
			log.Error().Err(err).Msg("OrderUpdater.worker error update " + storage.OtProcessing)
			time.Sleep(time.Second)
			continue
		}

		if toManyReq {
			time.Sleep(time.Minute)
			continue
		}

		time.Sleep(time.Second)
	}
}

func (s *OrderUpdater) update(ctx context.Context, status string) (bool, error) {
	orders, err := s.Store.GetOrderIdsByStatus(ctx, status)
	if err != nil {
		return false, err
	}

	for _, order := range orders {
		log.Info().Msg("update order " + order + " with status " + status)
		updatedOrder, toManyReq, err := s.Scores.GetScore(ctx, order)
		if err != nil {
			return false, err
		}

		if toManyReq {
			return toManyReq, nil
		}

		if updatedOrder.Status != status {
			err = s.Store.UpdateOrder(ctx, updatedOrder)
			if err != nil {
				return false, err
			}
		}

	}
	return false, nil
}
