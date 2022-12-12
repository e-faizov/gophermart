package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/lib/pq"

	"github.com/e-faizov/gophermart/internal/models"
	"github.com/e-faizov/gophermart/internal/utils"
)

const (
	OtNew        = "NEW"
	OtProcessing = "PROCESSING"
	OtInvalid    = "INVALID"
	OtProcessed  = "PROCESSED"
)

func NewPgStore(conn, secret string) (*PgStore, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, utils.ErrorHelper(fmt.Errorf("error open db: %w", err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = initTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("error init tables: %w", err)
	}

	return &PgStore{
		db:     db,
		secret: secret,
	}, nil
}

type PgStore struct {
	db     *sql.DB
	secret string
}

func (p *PgStore) Register(ctx context.Context, login, password string) (bool, string, error) {
	hash := calcHash(password, p.secret)
	uid := uuid.New()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, "", utils.ErrorHelper(err)
	}

	rollback := func(err error) error {
		errRoll := tx.Rollback()
		if errRoll != nil {
			err = multierror.Append(err, fmt.Errorf("error on rollback %w", errRoll))
		}
		return err
	}

	sqlString := `insert into users (uuid, login, hash) values ($1, $2, $3)`
	_, err = tx.ExecContext(ctx, sqlString, uid.String(), login, hash)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "users_login_uindex" {
			return false, "", nil
		}
		return false, "", rollback(utils.ErrorHelper(err))
	}

	sqlString = `insert into balances (user_id, balance) values ((select id from users where uuid=$1), 0)`
	_, err = tx.ExecContext(ctx, sqlString, uid.String())
	if err != nil {
		return false, "", rollback(utils.ErrorHelper(err))
	}

	err = tx.Commit()
	if err != nil {
		return false, "", utils.ErrorHelper(err)
	}
	return true, uid.String(), nil
}
func (p *PgStore) Login(ctx context.Context, login, password string) (string, bool, error) {
	hash := calcHash(password, p.secret)
	sqlString := `select uuid from users where login=$1 and hash=$2`

	rows, err := p.db.QueryContext(ctx, sqlString, login, hash)
	if err != nil {
		return "", false, utils.ErrorHelper(err)
	}
	defer rows.Close()
	count := 0
	resUudi := ""
	for rows.Next() {
		count++
		err = rows.Scan(&resUudi)
		if err != nil {
			return "", false, utils.ErrorHelper(err)
		}
	}
	if err = rows.Err(); err != nil {
		return "", false, utils.ErrorHelper(err)
	}
	if count != 1 {
		return "", false, nil
	}
	return resUudi, true, nil
}

func (p *PgStore) SaveOrder(ctx context.Context, user, order string) (bool, bool, error) {
	script := `insert into orders (order_id, user_id, uploaded, status)
				values ($1, (select id from users where uuid=$2), $3, (select id from order_types where type=$4))`
	_, err := p.db.ExecContext(ctx, script, order, user, time.Now(), OtNew)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "orders_order_id_uindex" {
			script = `select uuid from users where id=(select user_id from orders where order_id=$1)`
			row := p.db.QueryRowContext(ctx, script, order)
			insertedUserID := ""
			err = row.Scan(&insertedUserID)
			if err != nil {
				return false, false, utils.ErrorHelper(err)
			}
			if insertedUserID == user {
				return false, true, nil
			} else {
				return false, false, nil
			}
		}
		return false, false, utils.ErrorHelper(err)
	}
	return true, true, nil
}

func (p *PgStore) GetOrders(ctx context.Context, user string) ([]models.Order, error) {
	script := `select t1.order_id, t1.uploaded, t2.type, t1.accrual from orders t1
				join order_types t2
				on t1.status=t2.id
				where t1.user_id=(select id from users where uuid=$1)`
	rows, err := p.db.QueryContext(ctx, script, user)
	if err != nil {
		return nil, utils.ErrorHelper(err)
	}
	defer rows.Close()

	var res []models.Order
	for rows.Next() {
		var order models.Order
		err = rows.Scan(&order.Number, &order.Uploaded, &order.Status, &order.Accrual)
		if err != nil {
			return nil, utils.ErrorHelper(err)
		}
		res = append(res, order)
	}
	if err = rows.Err(); err != nil {
		return nil, utils.ErrorHelper(err)
	}
	return res, nil
}

func (p *PgStore) GetOrderIdsByStatus(ctx context.Context, tp string) ([]string, error) {
	script := `select t1.order_id from orders t1
				join order_types t2
				on t1.status=t2.id
				where t2.type=$1`
	rows, err := p.db.QueryContext(ctx, script, tp)
	if err != nil {
		return nil, utils.ErrorHelper(err)
	}
	defer rows.Close()

	var res []string
	for rows.Next() {
		var order string
		err = rows.Scan(&order)
		if err != nil {
			return nil, utils.ErrorHelper(err)
		}
		res = append(res, order)
	}
	if err = rows.Err(); err != nil {
		return nil, utils.ErrorHelper(err)
	}
	return res, nil
}

func (p *PgStore) UpdateOrder(ctx context.Context, order models.Order) error {
	switch order.Status {
	case OtInvalid, OtNew, OtProcessing:
		script := "update orders set status=(select id from order_types where type=$1) where order_id=$2"
		_, err := p.db.ExecContext(ctx, script, order.Status, order.Number)
		return utils.ErrorHelper(err)
	case OtProcessed:
		script :=
			`with order_update as (update orders set status=(select id from order_types where type=$1), accrual=$2 where order_id=$3 returning user_id)
		update balances set balance=balance+$2 where user_id=(select user_id from order_update)`
		_, err := p.db.ExecContext(ctx, script, order.Status, order.Accrual, order.Number)
		return utils.ErrorHelper(err)
	}

	return utils.ErrorHelper(errors.New("unknown order status: " + order.Status))
}

func (p *PgStore) WithdrawalsByUser(ctx context.Context, uuid string) ([]models.Withdraw, error) {
	sqlString := "select order_id, sum, processed from withdrawals where user_id=(select id from users where uuid=$1)"

	rows, err := p.db.QueryContext(ctx, sqlString, uuid)
	if err != nil {
		return nil, utils.ErrorHelper(err)
	}
	defer rows.Close()
	var res []models.Withdraw

	for rows.Next() {
		var tmp models.Withdraw
		err = rows.Scan(&tmp.Order, &tmp.Sum, &tmp.Processed)
		if err != nil {
			return nil, utils.ErrorHelper(err)
		}
		res = append(res, tmp)
	}
	if err = rows.Err(); err != nil {
		return nil, utils.ErrorHelper(err)
	}

	return res, nil
}

func (p *PgStore) BalanceByUser(ctx context.Context, uuid string) (models.Balance, error) {
	sqlString := `select balance from balances where user_id=(select id from users where uuid=$1)`
	row := p.db.QueryRowContext(ctx, sqlString, uuid)
	var res models.Balance
	err := row.Scan(&res.Current)
	if err != nil {
		return models.Balance{}, utils.ErrorHelper(err)
	}

	sqlString = `select sum(sum) from withdrawals where user_id=(select id from users where uuid=$1)`
	row = p.db.QueryRowContext(ctx, sqlString, uuid)

	var tmp *float64
	err = row.Scan(&tmp)
	if err != nil {
		return models.Balance{}, utils.ErrorHelper(err)
	}

	if tmp != nil {
		res.Withdrawn = *tmp
	}
	return res, nil
}

func (p *PgStore) Withdraw(ctx context.Context, withdraw models.Withdraw, uuid string) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, utils.ErrorHelper(err)
	}
	rollback := func(err error) error {
		errRoll := tx.Rollback()
		if errRoll != nil {
			err = multierror.Append(err, fmt.Errorf("error on rollback %w", errRoll))
		}
		return err
	}

	sqlString := `update balances set balance=balance-$1 where user_id=(select id from users where uuid=$2)`

	_, err = tx.ExecContext(ctx, sqlString, withdraw.Sum, uuid)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "balances_nonnegative" {
			return true, nil
		}
		return false, rollback(utils.ErrorHelper(err))
	}

	sqlString = `insert into withdrawals (user_id, order_id, sum, processed) values ((select id from users where uuid=$1), $2, $3, $4)`
	_, err = tx.ExecContext(ctx, sqlString, uuid, withdraw.Order, withdraw.Sum, time.Now())
	if err != nil {
		return false, rollback(utils.ErrorHelper(err))
	}

	err = tx.Commit()
	return false, utils.ErrorHelper(err)
}

func calcHash(s string, k string) string {
	h := hmac.New(sha256.New, []byte(k))
	h.Write([]byte(s))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
}
