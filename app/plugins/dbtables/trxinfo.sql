create table trxinfo
(
	trx_id varchar(64) not null,
	block_height int unsigned not null,
	block_time int unsigned not null,
	invoice json null,
	operations json null,
	block_id varchar(64) null,
	constraint trxinfo_trx_id_uindex
		unique (trx_id)
);
