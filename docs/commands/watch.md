# `soda watch`

Poll a dataset on an interval and emit only the rows that have arrived since
last poll.

```
$ soda watch nyc erm2-nwe9 --interval 5m
[2026-05-17T21:13:08-04:00] initial watermark set to 2026-05-17T01:54:23.171Z
{":id":"row-xyz", ..., ":updated_at":"2026-05-17T21:14:11.022Z"}
{":id":"row-abc", ..., ":updated_at":"2026-05-17T21:14:15.901Z"}
[2026-05-17T21:18:08-04:00] 2 new rows, watermark now 2026-05-17T21:14:15.901Z
```

## How it works

soda keeps a high-watermark timestamp (defaults to `:updated_at`, Socrata's
internal last-modified field) in a state file. On each poll, it asks for
rows whose value is strictly greater than the watermark, emits them, and
updates the file.

The first poll sets the watermark to the dataset's current max so you don't
get a flood of historical rows. Subsequent polls emit only what's actually
new.

## Flags

| Flag | Default | Notes |
|---|---|---|
| `--interval` | `1m` | How often to poll (e.g. `30s`, `5m`, `1h`) |
| `--once` | false | Poll a single time and exit. Useful in cron jobs. |
| `--since-column` | `:updated_at` | Watermark column. Override for datasets where `:updated_at` isn't meaningful. |
| `--state-file` | `~/.cache/soda/watch_<portal>_<id>.state` | Path to persist the watermark |

## In cron

```
*/5 * * * * cd /var/lib/soda && /usr/local/bin/soda watch nyc erm2-nwe9 --once >> /var/log/nyc311.log 2>&1
```

The state file makes consecutive runs pick up where they left off, so a
5-minute cron is equivalent to running with `--interval 5m`.

## Why use `:updated_at`?

It's the only timestamp every Socrata row carries automatically. Even
datasets without an explicit date column get one. If your dataset has a
business-meaningful timestamp you'd rather watch (e.g. `created_date`),
pass `--since-column created_date`.
