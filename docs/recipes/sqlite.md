# Building a local SQLite database

`soda pull --to <file.db>` writes records into a SQLite database, one table
per dataset. Combine with `--all` to bulk-load entire datasets.

```
$ soda pull nyc erm2-nwe9 --all --where "created_date >= '2026-01-01'" --to nyc311.db
fetched 50000 rows (offset=50000)
fetched 100000 rows (offset=100000)
fetched 150000 rows (offset=150000)
...
wrote 412580 rows into nyc311.db (table d_erm2_nwe9)
```

## Querying

```
$ sqlite3 nyc311.db
sqlite> SELECT borough, COUNT(*) FROM d_erm2_nwe9 GROUP BY 1 ORDER BY 2 DESC;
BROOKLYN|118390
QUEENS|107712
MANHATTAN|97844
BRONX|65213
STATEN ISLAND|23421
```

## Combining multiple datasets in one file

A single `.db` can hold many dataset tables. Just keep using the same path:

```
$ soda pull nyc erm2-nwe9 --all --to civic.db
$ soda pull nyc 64uk-42ks --all --to civic.db
$ soda pull nyc rt7n-d92f --all --to civic.db

$ sqlite3 civic.db '.tables'
d_64uk_42ks  d_erm2_nwe9  d_rt7n_d92f
```

## Incremental refresh

Reruns upsert on `:id`, so re-pulling the same query updates existing rows
in place. Pair with `soda watch` (or a cron job) to keep a local mirror
fresh:

```
# Sync every 10 minutes (in cron)
*/10 * * * * soda pull nyc erm2-nwe9 \
  --where ":updated_at >= '$(date -u -d '15 minutes ago' +%Y-%m-%dT%H:%M:%S)'" \
  --to /var/lib/soda/nyc311.db
```

## Schema notes

- Column types come from the dataset's SODA schema:
  - `number` → REAL
  - `checkbox` → INTEGER
  - all date/time variants → TEXT (ISO 8601 strings — easier to filter than
    UNIX timestamps)
  - everything else → TEXT
- Nested values (point types, computed regions) are JSON-encoded into TEXT
  columns. Use SQLite's `json_extract()` to query them.
- If the API returns a column the schema didn't declare, soda adds it as a
  TEXT column on the fly.
