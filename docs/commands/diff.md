# `soda diff`

Compare two snapshots of a Socrata dataset row by row.

```
$ soda pull nyc erm2-nwe9 --where "borough='BROOKLYN'" -o monday.json
$ soda pull nyc erm2-nwe9 --where "borough='BROOKLYN'" -o tuesday.json
$ soda diff monday.json tuesday.json
diff monday.json -> tuesday.json (key=:id)
  added:   42
  removed: 7
  changed: 18

sample changes:
  row-abc: status, resolution_description
  row-def: closed_date, status
  row-ghi: status
```

## How rows are matched

Both files must be JSON arrays of objects (the default output of
`soda pull`). Rows are joined on a key column, which defaults to `:id`
(Socrata's row identifier). Override with `--key` for datasets where you
want a business key instead:

```
$ soda diff a.json b.json --key unique_key
```

## Output formats

| Flag | Output |
|---|---|
| (default) `--format summary` | Human-readable counts + a sample of changes |
| `--format json` | Full diff as a JSON object with `added`, `removed`, `changed` lists |

The JSON output shape is stable and reproducible:

```json
{
  "key": ":id",
  "added": [ ...rows that are in B but not A... ],
  "removed": [ ...rows that are in A but not B... ],
  "changed": [
    {
      "key": "row-abc",
      "changes": [
        {"field": "status", "old": "Open", "new": "Closed"},
        {"field": "closed_date", "old": null, "new": "2026-05-17T12:00:00.000"}
      ]
    }
  ]
}
```

## Use cases

- **Journalism**: did the city quietly change a row's resolution
  description after the original complaint went viral?
- **Civic auditing**: snapshot a dataset weekly, run `diff` to see what
  was added / silently corrected.
- **Pipeline testing**: catch regressions in your ETL by diffing the
  output of two runs.
