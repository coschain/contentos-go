create table stateloglibinfo
(
  lib int unsigned not null,
  last_check_time int unsigned not null
);

create table statelog
(
  id bigint unsigned primary key auto_increment,
  block_id varchar(64),
  block_height int unsigned,
  trx_id varchar(64),
  action smallint,
  property varchar(64),
  state json,
  INDEX statelog_block_id_index (block_id),
  INDEX statelog_block_height_index (block_height),
  INDEX statelog_trx_id_index (trx_id)
);