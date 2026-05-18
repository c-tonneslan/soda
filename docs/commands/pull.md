# `soda pull`

Download dataset rows.

## Output formats

| Flag | Output |
|---|---|
| (default) | Pretty JSON array |
| `--format ndjson` | Newline-delimited JSON, one row per line (great for `jq`) |
| `--format csv` or `--csv` | CSV |
| `--to <file.db>` | SQLite database |

`--to` is only compatible with the default JSON path; use it alongside
`--all` to bulk-load a dataset.

## SoQL filtering

Pass [SoQL clauses](https://dev.socrata.com/docs/queries/) directly:

```
$ soda pull nyc erm2-nwe9 \
    --where "borough='BROOKLYN' AND created_date >= '2026-01-01'" \
    --select "unique_key, complaint_type, created_date" \
    --order "created_date DESC" \
    --limit 1000
```

## Single page vs. all rows

By default, `pull` makes one request and returns up to 1000 rows (Socrata's
server-side default). Pass `--limit` up to 50000 for one large page.

Pass `--all` to auto-paginate the entire (filtered) dataset:

```
$ soda pull chicago ydr8-5enu --all --where "year=2024" --to chicago_2024.db
```

`--all` requires either NDJSON / JSON output (default) or `--to`. CSV is
incompatible with `--all` because the second page would re-emit the header.

## Writing to a file

`-o` writes the response to a file instead of stdout:

```
$ soda pull nyc erm2-nwe9 --limit 5000 -o nyc311.json
wrote 8420314 bytes to nyc311.json
```

## SQLite schema

The SQLite sink creates one table per dataset, named `d_<four-by-four>` with
hyphens replaced by underscores. Column types come from the dataset's SODA
schema (`number` → REAL, `checkbox` → INTEGER, everything else → TEXT). The
table's primary key is `:id`, Socrata's internal row identifier, so reruns
upsert in place rather than duplicate. Nested values (point types, etc.)
are JSON-encoded into TEXT columns.

If the dataset's response has columns the schema didn't declare (Socrata
sometimes adds `:@computed_region_*` fields), soda adds them on the fly as
TEXT columns.
