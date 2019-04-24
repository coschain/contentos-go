create table trxinfo
(
	trx_id varchar(64) not null,
	block_height int unsigned not null,
	block_time int unsigned not null,
	invoice json null,
	operations json null,
	block_id varchar(64) not null,
	creator varchar(64) not null,
	INDEX trxinfo_block_height_index (block_height),
	INDEX trxinfo_block_time_index (block_time),
	INDEX trxinfo_block_id (block_id),
	INDEX trxinfo_block_creator (creator),
	constraint trxinfo_trx_id_uindex
		unique (trx_id)
);
