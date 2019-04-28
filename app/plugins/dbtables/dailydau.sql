create table dailydau
(
	date varchar(16) not null,
  hour smallint not null,
  pg int unsigned default 0,
  ct int unsigned default 0,
  g2 int unsigned default 0,
  ec int unsigned default 0,
	constraint dailydau_date_hour_uindex
		unique (date, hour)
);