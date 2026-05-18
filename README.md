# soda

A CLI for working with Socrata-based open data portals. Search across NYC,
Chicago, Seattle, LA, the CDC, and 40+ other government portals from one
command, pull datasets locally as JSON / CSV / NDJSON / SQLite, run SoQL
filters without writing curl, watch a dataset for new rows, and diff two
snapshots to see exactly what changed.

Most major US municipal and state governments publish open data through
[Socrata](https://dev.socrata.com/). Socrata's API is solid, but the
developer experience around it is rough: the web UI is slow, the existing
Python clients are dated, and there's no go-to terminal tool. soda is that
tool.

Full docs at [c-tonneslan.github.io/soda](https://c-tonneslan.github.io/soda/).

## Install

```
go install github.com/c-tonneslan/soda/cmd/soda@latest
```

Or `brew install c-tonneslan/tap/soda`. Or grab a release binary from
[Releases](https://github.com/c-tonneslan/soda/releases).

## At a glance

```
$ soda portals                                         # 49 known portals
$ soda search "affordable housing"                     # discovery API across all portals
$ soda ls nyc --limit 25                               # list datasets in NYC's portal
$ soda info nyc erm2-nwe9                              # schema + metadata
$ soda stats nyc erm2-nwe9                             # row count + date range, no download
$ soda pull nyc erm2-nwe9 --limit 1000 --where "borough='BROOKLYN'"
$ soda pull chicago ydr8-5enu --all --to crime.db      # SQLite, auto-paginated
$ soda pull nyc erm2-nwe9 --format ndjson | jq .       # streaming for jq
$ soda watch nyc erm2-nwe9 --interval 5m               # emit new rows as they arrive
$ soda diff yesterday.json today.json                  # row-level snapshot diff
$ soda open nyc erm2-nwe9                              # open in browser
```

## Output formats

| Flag | Output |
|---|---|
| (default) | Pretty JSON array |
| `--format ndjson` | Newline-delimited JSON, one row per line (great for `jq`) |
| `--format csv` / `--csv` | CSV |
| `--to <file.db>` | SQLite database, one table per dataset, upserts on `:id` |

`--all` walks every row across pages. `--cache` keeps GET responses on disk
under `~/.cache/soda/` so iterative work doesn't keep hitting Socrata.

## SoQL filtering

```
$ soda pull nyc erm2-nwe9 \
    --where "borough='BROOKLYN' AND created_date >= '2026-01-01'" \
    --select "unique_key, complaint_type, created_date" \
    --order "created_date DESC" \
    --all --to nyc311.db
```

Reference for the SoQL dialect:
[dev.socrata.com/docs/queries/](https://dev.socrata.com/docs/queries/) and
the [SoQL recipe](docs/recipes/soql.md) in this repo.

## Watch a dataset

```
$ soda watch nyc erm2-nwe9 --interval 5m
[2026-05-17T21:13:08-04:00] initial watermark set to 2026-05-17T01:54:23.171Z
{":id":"row-...","unique_key":"...","complaint_type":"Noise - Vehicle",...}
[2026-05-17T21:18:08-04:00] 42 new rows, watermark now 2026-05-17T21:14:15.901Z
```

`watch` keeps a high-watermark timestamp in a state file so consecutive
runs (in cron, or as a long-running daemon) only emit what's new. Pass
`--once` for cron use.

## Diff two snapshots

```
$ soda diff monday.json tuesday.json
diff monday.json -> tuesday.json (key=:id)
  added:   42
  removed: 7
  changed: 18

sample changes:
  row-abc: status, resolution_description
  row-def: closed_date, status
```

Pass `--format json` for the full structured diff (added rows, removed
rows, and per-field old/new values for changed rows).

## App Token

Socrata throttles unauthenticated requests fairly hard. For real workloads,
register a free App Token at the portal you're hitting and export:

```
$ export SODA_APP_TOKEN=<your-token>
```

soda picks it up automatically.

## Adding a portal

Open [`internal/portals/portals.go`](internal/portals/portals.go) and add a
row:

```go
{Slug: "philly", Name: "Philadelphia, PA", Host: "data.phila.gov"},
```

That's the whole change. See [docs/adding_a_portal.md](docs/adding_a_portal.md)
for finding hostnames and verifying a portal is Socrata-backed.

## Why not `sodapy`?

[sodapy](https://github.com/xmunoz/sodapy) is fine, but it's a Python
library, not a tool. You install it with pip, write a script, run the
script. soda is a single Go binary you install once and use from the
shell. No Python interpreter, no virtual environment, no script to write
for a one-off CSV pull.

## Pairs with [convene](https://github.com/c-tonneslan/convene)

`convene` covers municipal meeting data from Legistar and Granicus
portals. `soda` covers every other dataset cities publish through Socrata.
Together they handle most of the civic-tech ingestion surface area.

## Development

```
git clone https://github.com/c-tonneslan/soda
cd soda
go build ./...
go test ./...
go run ./cmd/soda portals
```

## License

MIT.
