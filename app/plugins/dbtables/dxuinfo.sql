create table dailystatdapp (
  dapp varchar(64) not null,
  prefix varchar(64) not null,
  status smallint default 1
);

create table daustat (
  date varchar(64) not null ,
  dapp varchar(64) not null ,
  count int unsigned not null default 0,
  INDEX daustat_dapp (dapp),
  constraint daustat_date_dapp_uindex
  unique (date, dapp)
);

create table dnustat (
  date varchar(64) not null ,
  dapp varchar(64) not null ,
  count int unsigned not null default 0,
  INDEX dnustat_dapp (dapp),
  constraint dnustat_date_dapp_uindex
  unique (date, dapp)
);

create table dailystatinfo
(
  lib int unsigned not null,
  date varchar(64) not null,
  last_check_time int unsigned not null
);