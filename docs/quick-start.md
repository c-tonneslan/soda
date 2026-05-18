# Quick start

List the portals soda knows about:

```
$ soda portals
SLUG       NAME                 HOST
albany     Albany, NY           data.albanycountyny.gov
austin     Austin, TX           data.austintexas.gov
...
nyc        New York City, NY    data.cityofnewyork.us
seattle    Seattle, WA          data.seattle.gov
```

Search every portal:

```
$ soda search "affordable housing"
ID         DOMAIN                 UPDATED     NAME
hg8x-zxpr  data.cityofnewyork.us  2026-03-20  Affordable Housing Production by Building
rckt-8prm  data.lacity.org        2025-09-11  Total and Affordable Housing Units
hq68-rnsi  data.cityofnewyork.us  2026-02-13  Affordable Housing Production by Project
```

Inspect a dataset before downloading:

```
$ soda stats nyc erm2-nwe9
311 Service Requests from 2020 to Present
rows:     21153127
columns:  48
updated:  2026-05-17
earliest created_date: 2020-01-01T00:00:00.000
latest created_date:   2026-05-16T01:51:14.000
```

Pull rows with a SoQL filter:

```
$ soda pull nyc erm2-nwe9 --limit 1000 \
    --where "borough='BROOKLYN' AND complaint_type='Noise - Residential'" \
    -o brooklyn_noise.json
```

Or stream the entire filtered subset as NDJSON to jq:

```
$ soda pull nyc erm2-nwe9 --all \
    --where "created_date >= '2026-01-01'" \
    --select ":id,borough,complaint_type" --format ndjson \
    | jq -c 'select(.borough=="MANHATTAN")'
```

Build a queryable local SQLite database:

```
$ soda pull chicago ydr8-5enu --all --to chicago_crime.db
fetched 50000 rows (offset=50000)
fetched 100000 rows (offset=100000)
...
wrote 8451293 rows into chicago_crime.db (table d_ydr8_5enu)

$ sqlite3 chicago_crime.db \
    'SELECT primary_type, COUNT(*) AS n FROM d_ydr8_5enu GROUP BY 1 ORDER BY n DESC LIMIT 5'
```

Watch a dataset for new rows (great as a cron):

```
$ soda watch nyc erm2-nwe9 --once
[2026-05-17T21:13:08-04:00] initial watermark set to 2026-05-17T01:54:23.171Z

$ soda watch nyc erm2-nwe9 --once   # an hour later
{":id":"row-...","unique_key":"...","complaint_type":"Noise - Vehicle",...}
[2026-05-17T22:13:08-04:00] 42 new rows, watermark now 2026-05-17T22:11:02.412Z
```

Diff two snapshots:

```
$ soda diff yesterday.json today.json
diff yesterday.json -> today.json (key=:id)
  added:   18
  removed: 2
  changed: 7

sample changes:
  row-abc: status, resolution_description
  row-def: closed_date, status
```
