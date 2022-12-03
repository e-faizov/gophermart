create table users
(
	id serial,
	uuid text,
	login text,
	hash text
)
create unique index users_uuid_uindex
	on users (uuid)
create unique index users_login_uindex
	on users (login)
alter table users
	add constraint users_pk
		primary key (login)
