create table stateloglibinfo
(
  lib int unsigned not null,
  last_check_time int unsigned not null
);

create table statelog
(
  block_height int unsigned,
  block_log json,
  UNIQUE KEY statelog_block_height (block_height)
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
