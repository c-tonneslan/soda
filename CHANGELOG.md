# Changelog

## Unreleased

- HTTP client retries on 429 and 5xx with exponential backoff (3 attempts
  by default, honors `Retry-After`). Most anonymous-token Socrata workflows
  now ride out a brief rate-limit hiccup instead of failing on first hit.
  `Client.MaxRetries` and `Client.RetryBase` knobs for library users.

## 0.4.0 — 2026-05-17

Docs and distribution.

- mkdocs-material docs site at [c-tonneslan.github.io/soda](https://c-tonneslan.github.io/soda/).
  Pages for install, quick-start, every command, SoQL reference, SQLite recipe,
  and change-detection recipe.
- Release workflow now bundles each binary as a tar.gz (or zip for Windows),
  computes SHA256SUMS, and emits a generated Homebrew formula. If a
  `SODA_TAP_TOKEN` secret is set, the formula is committed straight to
  `c-tonneslan/homebrew-tap`.

## 0.3.0 — 2026-05-17

Change detection.

- `soda watch <portal> <id>` polls a dataset on an interval and emits only
  rows whose timestamp column (default `:updated_at`) exceeds a stored
  watermark. State persists in `~/.cache/soda/`. The first poll seeds the
  watermark at the current max instead of dumping history. `--once` for
  cron use.
- `soda diff <before.json> <after.json>` row-level compares two snapshots
  by `:id` (override with `--key`). Output as `summary` (default) or full
  `json` with added / removed / changed lists; changed rows include
  field-level old/new values.

## 0.2.0 — 2026-05-17

Bigger data, more sinks, a couple of utility commands.

- `soda pull --all` auto-paginates the entire (filtered) dataset.
- `soda pull --to <file.db>` writes into a SQLite database. One table per
  dataset, named `d_<four-by-four>`, columns typed from the SODA schema,
  upserts on `:id`. Unknown columns get added as TEXT on the fly.
- `soda pull --format ndjson` for streaming-friendly output.
- `--cache` (global flag) caches GET responses under `~/.cache/soda/` for
  iterative work.
- `--verbose` (global flag) logs every URL hit to stderr.
- New `soda stats <portal> <id>` command. Row count + column count +
  earliest/latest date without downloading the dataset.
- New `soda open <portal> <id>` command. Opens the dataset's web page in
  the default browser.
- Better error messages: 429 hints at `SODA_APP_TOKEN`, 403 explains the
  token requirement, 404 suggests `soda search`.
- 49 portals preconfigured (up from 23). Adds Houston, Phoenix, Nashville,
  Miami, Tampa, Raleigh, Charlotte, San Diego, Long Beach, Oakland, Saint
  Louis, plus IL/TX/OR/HI/IA/VT state portals, Medicare/Energy/DOT
  federal portals, Edmonton, and Australia.

## 0.1.0 — 2026-05-17

First release.

- `soda portals` lists 23 preconfigured Socrata-based portals.
- `soda ls <portal>` lists datasets with `--limit` / `--offset`.
- `soda info <portal> <id>` shows metadata + column schema.
- `soda pull <portal> <id>` downloads rows. Supports `--csv`, `--limit`,
  `--offset`, and SoQL clauses (`--where`, `--order`, `--select`).
- `soda search <query>` runs the Discovery API across every portal,
  optionally restricted to one with `--portal`.
- Every command supports `--json` for machine-readable output.
- `SODA_APP_TOKEN` env var for authenticated requests.
- Single static Go binary; works on Linux, macOS, Windows.
- GitHub Actions builds release binaries for 5 platform/arch combos on
  tag push.
