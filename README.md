# soda

A CLI for working with Socrata-based open data portals. Search across NYC,
Chicago, Seattle, LA, the CDC, and dozens of other government portals from one
command, pull datasets locally as JSON or CSV, run SoQL filters without
writing curl, and inspect schemas without opening a browser.

Most major US municipal and state governments publish open data through
[Socrata](https://dev.socrata.com/). Socrata's API is solid but the developer
experience around it is rough: the web UI is slow, the existing Python
clients are dated, and there's no go-to terminal tool. `soda` is that tool.

## Install

```
go install github.com/c-tonneslan/soda/cmd/soda@latest
```

Or download a release binary from [Releases](https://github.com/c-tonneslan/soda/releases).

## At a glance

```
$ soda portals
SLUG        NAME                 HOST
albany      Albany, NY           data.albanycountyny.gov
austin      Austin, TX           data.austintexas.gov
boston      Boston, MA           data.boston.gov
chicago     Chicago, IL          data.cityofchicago.org
nyc         New York City, NY    data.cityofnewyork.us
seattle     Seattle, WA          data.seattle.gov
...

$ soda ls nyc --limit 5
ID         UPDATED     NAME
35j5-n34v  2026-04-10  ZIP Code Tabulation Areas
qhkz-4dqm  2026-03-19  Citywide Mobility Survey - Vehicle 2024
...

$ soda info nyc erm2-nwe9
311 Service Requests from 2020 to Present
by 311
updated 2026-05-17

311 responds to thousands of inquiries, comments and requests...

FIELD                TYPE           LABEL
unique_key           text           Unique Key
created_date         calendar_date  Created Date
...

$ soda pull nyc erm2-nwe9 --limit 1000 --where "borough='BROOKLYN'" -o brooklyn311.json
wrote 421032 bytes to brooklyn311.json

$ soda search "affordable housing" --limit 5
ID         DOMAIN                 UPDATED     NAME
hg8x-zxpr  data.cityofnewyork.us  2026-03-20  Affordable Housing Production by Building
rckt-8prm  data.lacity.org        2025-09-11  Total and Affordable Housing Units
hq68-rnsi  data.cityofnewyork.us  2026-02-13  Affordable Housing Production by Project
```

## Commands

| Command | What it does |
|---|---|
| `soda portals` | List the open-data portals soda knows about |
| `soda ls <portal>` | List datasets in a portal (paginate with `--limit` / `--offset`) |
| `soda info <portal> <id>` | Show metadata + column schema for a dataset |
| `soda pull <portal> <id>` | Download rows. `--csv` for CSV; `--where`, `--order`, `--select` for SoQL |
| `soda search <query>` | Search across every Socrata portal (or one with `--portal`) |

All commands accept `--json` (where applicable) to emit machine-readable
output instead of the human-friendly table.

## SoQL examples

Filter:

```
$ soda pull nyc erm2-nwe9 --where "borough='BROOKLYN' AND complaint_type='Noise - Residential'"
```

Project + sort:

```
$ soda pull chicago ydr8-5enu --select "incident_id, primary_type, date" --order "date DESC"
```

Reference: [SoQL clauses on dev.socrata.com](https://dev.socrata.com/docs/queries/).

## App tokens

Socrata throttles unauthenticated requests fairly aggressively. For any real
workload, register a free App Token at the portal you're hitting and export
it:

```
$ export SODA_APP_TOKEN=<your-token>
```

soda picks it up automatically on every request.

## Adding a portal

Open
[`internal/portals/portals.go`](internal/portals/portals.go)
and add a row:

```go
{Slug: "philly", Name: "Philadelphia, PA", Host: "data.phila.gov"},
```

(The slug is whatever you want to type at the CLI. The host is the bare
hostname, no scheme.)

## Why not `sodapy`?

[`sodapy`](https://github.com/xmunoz/sodapy) is fine, but it's a Python
library, not a tool. You install it with pip, write a script, run the
script. soda is a single Go binary you install once and use from the
shell. No Python interpreter, no virtual environment, no script to write
for a one-off CSV pull.

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
