# Discovery commands

## `soda portals`

Lists all preconfigured portals.

```
$ soda portals
```

## `soda ls <portal>`

Lists datasets in a portal. Socrata caps results at 100 per request; paginate
with `--offset`.

```
$ soda ls chicago --limit 50
$ soda ls chicago --limit 50 --offset 50
$ soda ls chicago --json
```

## `soda info <portal> <id>`

Shows the dataset's metadata: name, attribution, last update, description,
and the full column list with SODA types.

```
$ soda info nyc erm2-nwe9
311 Service Requests from 2020 to Present
by 311
updated 2026-05-17

311 responds to thousands of inquiries, comments and requests...

FIELD                TYPE           LABEL
unique_key           text           Unique Key
created_date         calendar_date  Created Date
...
```

`--json` outputs the same data as a JSON object for programmatic use.
