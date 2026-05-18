# soda

A CLI for working with Socrata-based open data portals. Search across NYC,
Chicago, Seattle, LA, the CDC, and 40+ other government portals from one
command, pull datasets locally as JSON / CSV / NDJSON / SQLite, run SoQL
filters without writing curl, watch a dataset for changes on an interval,
and diff two snapshots to see exactly what changed.

Most major US municipal and state governments publish open data through
[Socrata](https://dev.socrata.com/). Socrata's API is solid, but the
developer experience around it is rough: the web UI is slow, the existing
Python clients are dated, and there's no go-to terminal tool. `soda` is that
tool.

## Why use it

- **Two-command install**: `go install` or grab the release binary. No
  Python interpreter, no virtualenv, no SDK.
- **49 portals preconfigured**: NYC, Chicago, LA, Seattle, SF, DC, Boston,
  Baltimore, Philly, Austin, Dallas, Houston, Denver, Phoenix, every major
  state portal, the CDC, plus international.
- **Full SoQL support**: pass `--where`, `--order`, `--select` straight
  through.
- **Sinks that match your tools**: pretty JSON for humans, CSV for
  spreadsheets, NDJSON for `jq`, SQLite for everything else.
- **Auto-pagination**: `--all` walks every row, no manual `--offset`
  bookkeeping.
- **Change detection**: `soda watch` polls a dataset and emits only the
  rows that arrived since you last checked. `soda diff` compares two
  snapshots row-by-row.

## Next steps

- [Install](install.md)
- [Quick start](quick-start.md)
- [Adding a portal](adding_a_portal.md)
- [SoQL filter reference](recipes/soql.md)
