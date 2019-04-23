create table trxinfo
(
	trx_id varchar(64) not null,
	block_height int unsigned not null,
	block_time int unsigned not null,
	block_id varchar(64) null,
	invoice json null,
	operations json null
);