# `soda search`

Search Socrata datasets via the Discovery API.

```
$ soda search "affordable housing"
ID         DOMAIN                 UPDATED     NAME
hg8x-zxpr  data.cityofnewyork.us  2026-03-20  Affordable Housing Production by Building
rckt-8prm  data.lacity.org        2025-09-11  Total and Affordable Housing Units
hq68-rnsi  data.cityofnewyork.us  2026-02-13  Affordable Housing Production by Project
```

By default soda hits the global Discovery endpoint, which spans every
Socrata-hosted portal in the world (not just the ones soda knows about by
slug). Restrict to one portal with `--portal`:

```
$ soda search "permits" --portal nyc --limit 5
```

`--json` emits the raw hit list for piping into other tools:

```
$ soda search "snow plow" --json | jq -r '.[] | .domain + " " + .name'
```

## Caveats

- The Discovery API doesn't index every portal's full catalog; some smaller
  portals only show their top datasets. For exhaustive listings on a single
  portal, prefer `soda ls <portal>` paginated with `--offset`.
- Search hits are deduplicated by ID across portals, so a dataset
  republished by multiple agencies may show up only once.
