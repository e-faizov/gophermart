create table balances
(
    user_id int
        constraint balances_pk
            primary key,
    balance float8
        constraint balances_nonnegative check(balance >= 0)
)