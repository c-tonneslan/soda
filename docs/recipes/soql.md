# SoQL filter reference

[SoQL](https://dev.socrata.com/docs/queries/) is SQL-flavored and soda passes
the clauses straight through. The full list of operators is on Socrata's
docs site; this page covers the patterns you'll actually use.

## `--where`

| Pattern | Example |
|---|---|
| Equality | `borough='BROOKLYN'` |
| Numeric comparison | `latitude > 40.7` |
| Date comparison | `created_date >= '2026-01-01T00:00:00'` |
| Date range | `created_date between '2026-01-01' and '2026-04-01'` |
| Multiple conditions | `borough='BROOKLYN' AND complaint_type='Noise - Residential'` |
| In a list | `borough in ('BROOKLYN', 'QUEENS')` |
| Substring | `lower(descriptor) like '%pothole%'` |
| Not null | `closed_date IS NOT NULL` |
| Within bounding box | `within_box(location, 40.7, -74.0, 40.6, -73.9)` |
| Within radius (meters) | `within_circle(location, 40.7128, -74.0060, 1000)` |

```
$ soda pull nyc erm2-nwe9 --limit 100 \
    --where "borough='BROOKLYN' AND created_date >= '2026-01-01'"
```

## `--order`

```
$ soda pull nyc erm2-nwe9 --order "created_date DESC"
$ soda pull nyc erm2-nwe9 --order "borough ASC, created_date DESC"
```

When using `--all`, pass an `--order` clause whose column is indexed (most
date columns are). Without one, Socrata's pagination order is not
guaranteed to be stable.

## `--select`

Project specific fields, run aggregates, or rename columns:

```
$ soda pull nyc erm2-nwe9 --select "borough, count(*) as n" --group "borough"
$ soda pull nyc erm2-nwe9 --select ":id, created_date, complaint_type"
$ soda pull nyc erm2-nwe9 --select "min(created_date), max(created_date)"
```

## Hidden metadata fields

Every Socrata row carries a few `:`-prefixed fields that aren't in the
visible schema. Select them explicitly:

| Field | Notes |
|---|---|
| `:id` | Stable row identifier. Used by `soda diff` and SQLite primary keys. |
| `:created_at` | When the row was first ingested. |
| `:updated_at` | When the row was last modified. Used by `soda watch`. |

```
$ soda pull nyc erm2-nwe9 --select "*, :updated_at" --limit 5
```

## Quoting

Single-quote string literals. Double-quote nothing. Backtick nothing.
SoQL is its own dialect, not SQL.
