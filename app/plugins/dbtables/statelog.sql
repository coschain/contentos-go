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

create table stateaccount
(
  account varchar(64),
  balance bigint unsigned default 0,
  UNIQUE Key stateaccount_account_index (account)
);

create table statemint
(
  bp varchar(64),
  revenue bigint unsigned default 0,
  unique key statemint_bp_index (bp)
);

create table statecashout
(
  account varchar(64),
  cashout bigint unsigned default 0,
  unique key statecashout_account_index (account)
);
