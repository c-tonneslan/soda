# Install

## From source

```
go install github.com/c-tonneslan/soda/cmd/soda@latest
```

Requires Go 1.22+.

## Pre-built binaries

Each release publishes binaries for linux/amd64, linux/arm64, darwin/amd64,
darwin/arm64, and windows/amd64. Grab one from
[the releases page](https://github.com/c-tonneslan/soda/releases) and drop
it in your `PATH`.

## Homebrew

```
brew install c-tonneslan/tap/soda
```

The tap lives at [c-tonneslan/homebrew-tap](https://github.com/c-tonneslan/homebrew-tap)
and the formula is updated on every soda release.

## App Token

Socrata rate-limits unauthenticated requests fairly aggressively. For any
real workload, register a free App Token at the portal you're hitting and
export it:

```
$ export SODA_APP_TOKEN=<your-token>
```

soda picks it up automatically on every request.
