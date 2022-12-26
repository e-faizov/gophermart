package updater

import (
	"context"
	"fmt"
	"github.com/e-faizov/gophermart/internal/storage"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"time"

	"github.com/e-faizov/gophermart/internal/interfaces"
)

type OrderUpdater struct {
	Scores interfaces.Scores
	Store  interfaces.OrdersStorage
	cancel context.CancelFunc
	done   chan struct{}
}

func (s *OrderUpdater) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.done = make(chan struct{})
	s.cancel = cancel
	go s.worker(ctx)
}

func (s *OrderUpdater) Stop() {
	s.cancel()
	<-s.done
}

func (s *OrderUpdater) worker(ctx context.Context) {
	var sleep time.Duration
	for {
		select {
		case <-ctx.Done():
			s.done <- struct{}{}
			return
		default:
			time.Sleep(sleep)
			toManyReq, err := s.update(ctx, storage.OtNew)
			if err != nil {
				log.Error().Err(err).Msg("OrderUpdater.worker error update " + storage.OtNew)
				sleep = time.Second
				continue
			}

			if toManyReq {
				sleep = time.Minute
				continue
			}

			toManyReq, err = s.update(ctx, storage.OtProcessing)
			if err != nil {
				log.Error().Err(err).Msg("OrderUpdater.worker error update " + storage.OtProcessing)
				sleep = time.Second
				continue
			}

			if toManyReq {
				sleep = time.Minute
				continue
			}
			sleep = time.Second
		}
	}
}
func (s *OrderUpdater) update(ctx context.Context, status string) (bool, error) {

	for {
		tx, err := s.Store.NewUpdaterTx(ctx)
		if err != nil {
			return false, err
		}

		rollback := func(err error) error {
			errRoll := tx.Rollback()
			if errRoll != nil {
				err = multierror.Append(err, fmt.Errorf("error on rollback %w", errRoll))
			}
			return err
		}

		order, notFound, err := tx.GetOrderIdsByStatus(ctx, status)
		if err != nil {
			return false, rollback(err)
		}

		if notFound {
			return false, nil
		}

		log.Info().Msg("update order " + order + " with status " + status)
		updatedOrder, toManyReq, err := s.Scores.GetScore(ctx, order)
		if err != nil {
			return false, rollback(err)
		}

		if toManyReq {
			return toManyReq, rollback(nil)
		}

		if updatedOrder.Status != status {
			err = tx.UpdateOrder(ctx, updatedOrder)
			if err != nil {
				return false, rollback(err)
			}
		}

		err = tx.Commit()
		if err != nil {
			return false, err
		}
	}
}
