create table createaccountinfo
(
	trx_id varchar(64) not null,
	create_time int unsigned not null,
	creator varchar(64) not null,
	pubkey varchar(64) not null,
	account varchar(64) not null,
	INDEX createaccount_create_time (create_time),
	INDEX createaccount_creator (creator),
	INDEX creatoraccount_account (account),
  constraint createaccount_trx_id_uindex unique (trx_id)
);
