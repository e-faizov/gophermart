package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"

	"github.com/e-faizov/gophermart/internal/utils"
)

func tableExist(ctx context.Context, db *sql.DB, tb string) bool {
	sql := `select exists (
	   select from information_schema.tables
	   where  table_schema = 'public'
	   and    table_name   = $1
	   )
`

	var exists bool
	row := db.QueryRowContext(ctx, sql, tb)
	err := row.Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func createTable(ctx context.Context, db *sql.DB, sqls ...string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return utils.ErrorHelper(fmt.Errorf("error open Tx: %w", err))
	}
	rollback := func(err error) error {
		errRoll := tx.Rollback()
		if errRoll != nil {
			err = multierror.Append(err, fmt.Errorf("error on rollback %w", errRoll))
		}
		return err
	}
	for _, s := range sqls {
		_, err = tx.ExecContext(ctx, s)
		if err != nil {
			return rollback(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		return utils.ErrorHelper(err)
	}
	return nil
}

func createOrderTypesTable(ctx context.Context, db *sql.DB) error {
	err := createTable(ctx, db,
		`create table order_types
(
	id int,
	type text
)`,
		`insert into order_types (id, type) values (0, 'NEW')`,
		`insert into order_types (id, type) values (1, 'PROCESSING')`,
		`insert into order_types (id, type) values (2, 'INVALID')`,
		`insert into order_types (id, type) values (3, 'PROCESSED')`)
	return utils.ErrorHelper(err)
}

func createOrdersTable(ctx context.Context, db *sql.DB) error {
	err := createTable(ctx, db,
		`create table orders
(
	id serial,
	order_id int not null,
	user_id int not null,
	uploaded timestamp not null,
	status int not null,
	accrual float8
)`,
		`create unique index orders_order_id_uindex
	on orders (order_id)`,
		`alter table orders
	add constraint orders_pk
		primary key (order_id)`)
	return utils.ErrorHelper(err)
}

func createUsersTable(ctx context.Context, db *sql.DB) error {

	err := createTable(ctx, db,
		`create table users
(
	id serial,
	uuid text,
	login text,
	hash text
)`,
		`create unique index users_uuid_uindex
	on users (uuid)`,
		`create unique index users_login_uindex
	on users (login)`,
		`alter table users
	add constraint users_pk
		primary key (login)`)
	return utils.ErrorHelper(err)
}

func createWithdrawalsTable(ctx context.Context, db *sql.DB) error {
	err := createTable(ctx, db,
		`create table withdrawals
(
    id        serial,
    user_id   int       not null,
    order_id  text      not null,
    sum       float8    not null,
    processed timestamp not null
)`,
		`create unique index withdrawals_order_id_uindex
    on withdrawals (order_id)`)
	return utils.ErrorHelper(err)
}

func createBalancesTable(ctx context.Context, db *sql.DB) error {
	err := createTable(ctx, db,
		`create table balances
(
	user_id int
		constraint balances_pk
			primary key,
	balance float8
        constraint balances_nonnegative check(balance >= 0)
)`)
	return utils.ErrorHelper(err)
}

func initTables(ctx context.Context, db *sql.DB) error {
	var err error
	exist := tableExist(ctx, db, "users")
	if !exist {
		err = createUsersTable(ctx, db)
		if err != nil {
			return fmt.Errorf("error create users: %v", err)
		}
		log.Info().Msg("users table created")
	}

	exist = tableExist(ctx, db, "orders")
	if !exist {
		err = createOrdersTable(ctx, db)
		if err != nil {
			return fmt.Errorf("error create orders: %v", err)
		}
		log.Info().Msg("users orders created")
	}

	exist = tableExist(ctx, db, "order_types")
	if !exist {
		err = createOrderTypesTable(ctx, db)
		if err != nil {
			return fmt.Errorf("error create order_types: %v", err)
		}
		log.Info().Msg("users order_types created")
	}

	exist = tableExist(ctx, db, "balances")
	if !exist {
		err = createBalancesTable(ctx, db)
		if err != nil {
			return fmt.Errorf("error create order_types: %v", err)
		}
		log.Info().Msg("users balances created")
	}

	exist = tableExist(ctx, db, "withdrawals")
	if !exist {
		err = createWithdrawalsTable(ctx, db)
		if err != nil {
			return fmt.Errorf("error create order_types: %v", err)
		}
		log.Info().Msg("users withdrawals created")
	}
	return nil
}

func clearTable(db *sql.DB) {
	db.Exec("drop table users")
	db.Exec("drop table orders")
	db.Exec("drop table order_types")
	db.Exec("drop table balances")
	db.Exec("drop table withdrawals")
}
