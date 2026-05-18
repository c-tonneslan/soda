# Changelog

## 0.1.0 — 2026-05-17

First release.

- `soda portals` lists 23 preconfigured Socrata-based portals (NYC, Chicago,
  Seattle, LA, SF, Boston, Baltimore, Austin, Dallas, Denver, Honolulu,
  Albany, Buffalo, CT, NY, MD, WA, HealthData.gov, CDC, more).
- `soda ls <portal>` lists datasets in a portal with `--limit` / `--offset`.
- `soda info <portal> <id>` shows metadata + column schema.
- `soda pull <portal> <id>` downloads rows. Supports `--csv`,
  `--limit`, `--offset`, `--where`, `--order`, `--select` SoQL clauses,
  and `-o` to write to file.
- `soda search <query>` runs the Socrata Discovery API across every portal,
  optionally restricted to one with `--portal`.
- Every command accepts `--json` for machine-readable output.
- Honors `SODA_APP_TOKEN` for authenticated requests (higher rate limits).
- Single static Go binary; works on Linux, macOS, Windows.
- GitHub Actions builds release binaries for 5 platform/arch combos on tag
  push.
