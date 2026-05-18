# Utility commands

## `soda stats <portal> <id>`

Quick summary of a dataset without downloading it. Reports row count,
column count, last update, and the min/max of the first date column.

```
$ soda stats nyc erm2-nwe9
311 Service Requests from 2020 to Present
rows:     21153127
columns:  48
updated:  2026-05-17
earliest created_date: 2020-01-01T00:00:00.000
latest created_date:   2026-05-16T01:51:14.000
```

`--where` restricts the row count + date range to a SoQL filter, useful for
scoping how much you're about to pull:

```
$ soda stats nyc erm2-nwe9 --where "borough='BROOKLYN' AND complaint_type='Noise - Residential'"
```

## `soda open <portal> <id>`

Opens the dataset's web page in your default browser. Pure convenience.

```
$ soda open nyc erm2-nwe9
https://data.cityofnewyork.us/d/erm2-nwe9
```

It also prints the URL so you can copy/paste it if the open command fails.
