package interfaces

import (
	"context"
	"github.com/e-faizov/gophermart/internal/models"
)

type UserStorage interface {
	Register(ctx context.Context, login, password string) (bool, error)
	Login(ctx context.Context, login, password string) (uuid string, ok bool, err error)
}

type OrdersStorage interface {
	SaveOrder(ctx context.Context, user, order string) (inserted bool, thisUser bool, err error)
	GetOrders(ctx context.Context, user string) ([]models.Order, error)
	GetOrderIdsByStatus(ctx context.Context, status string) ([]string, error)
	UpdateOrderStatus(ctx context.Context, order, status string) error
	Withdraw(ctx context.Context, withdraw models.Withdraw, uuid string) (notEnough bool, err error)
	WithdrawalsByUser(ctx context.Context, uuid string) ([]models.Withdraw, error)
	BalanceByUser(ctx context.Context, uuid string) (models.Balance, error)
}
