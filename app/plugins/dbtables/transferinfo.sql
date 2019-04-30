create table transferinfo
(
	trx_id varchar(64) not null,
	create_time int unsigned not null,
	sender varchar(64) not null,
	receiver varchar(64) not null,
	amount int unsigned default 0,
	memo TEXT ,
	INDEX transfer_create_time (create_time),
	INDEX transfer_sender (sender),
	INDEX transfer_receiver (receiver),
  constraint transferinfo_trx_id_uindex unique (trx_id)
);