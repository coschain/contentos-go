create table dailystatdapp (
  dapp varchar(64) not null,
  prefix varchar(64) not null,
  status smallint default 1
);

create table dailystat (
  date varchar(64) not null ,
  dapp varchar(64) not null ,
  dau int unsigned not null default 0,
  dnu int unsigned not null default 0,
  trxs int unsigned not null default 0,
  amount bigint unsigned not null default 0,
  tusr int unsigned not null  default 0,
  INDEX dailystat_dapp (dapp),
  constraint dailystat_date_dapp_uindex
  unique (date, dapp)
);

create table dailystatinfo
(
  lib int unsigned not null,
  date varchar(64) not null,
  last_check_time int unsigned not null
);