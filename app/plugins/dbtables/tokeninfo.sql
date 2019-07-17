create table tokenlibinfo
(
    lib int unsigned not null,
    last_check_time int unsigned not null
);

create table markedtoken
(
    symbol varchar(64),
    owner varchar(64)
);

create table tokenbalance
(
    symbol varchar(64),
    owner varchar(64),
    account varchar(64),
    balance bigint unsigned default 0
);
