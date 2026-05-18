# Adding a portal

Adding a Socrata-based portal to soda is a one-line entry in
[`internal/portals/portals.go`](https://github.com/c-tonneslan/soda/blob/main/internal/portals/portals.go).

## Step 1: Find the host

Open the portal's website and look at any dataset URL. The part between
`https://` and the first slash is the host.

| URL | Host |
|---|---|
| `https://data.cityofnewyork.us/d/erm2-nwe9` | `data.cityofnewyork.us` |
| `https://data.cdc.gov/Public-Health-Surveillance/...` | `data.cdc.gov` |
| `https://data.boston.gov/dataset/311-service-requests` | `data.boston.gov` |

## Step 2: Verify it speaks SODA

```
curl -s "https://<host>/api/views.json?limit=1" | jq '.[0].id'
```

A four-by-four ID means yes.

## Step 3: Add the entry

```go
{Slug: "philly", Name: "Philadelphia, PA", Host: "data.phila.gov"},
```

The slug is whatever you'd want to type at the CLI. Keep it short.

## Step 4: Run it

```
soda ls philly --limit 3
```

If datasets show up, ship a PR.

## Portals that aren't Socrata

Some government portals look similar but use CKAN, ArcGIS, or custom
backends. soda only supports Socrata. To tell them apart:

| Backend | Looks like |
|---|---|
| Socrata | `data.<city>.gov` with `/resource/<four-by-four>.json` endpoints |
| CKAN | URLs containing `/api/3/action/` |
| ArcGIS Hub | `data-<city>.opendata.arcgis.com` |

You can usually find the Socrata mirror if a city has both; check the
dataset's "Export → JSON" link.
