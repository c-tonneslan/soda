# Daily change detection

Two soda commands cover most change-detection use cases: `watch` for
streaming new rows as they arrive, `diff` for periodic snapshot comparison.

## Pattern 1: tail with `watch` + a notifier

Pipe `watch`'s NDJSON output into a webhook or logger. This works as a
long-running daemon or as a 5-minute cron with `--once`.

```bash
#!/bin/bash
soda watch nyc erm2-nwe9 --once --since-column created_date \
  | jq -r 'select(.complaint_type=="Water Main Break") | .incident_address' \
  | while read addr; do
      curl -s -X POST -d "{\"text\":\"Water main: ${addr}\"}" $SLACK_WEBHOOK
    done
```

## Pattern 2: nightly snapshot + diff

Save a JSON snapshot every day, diff today against yesterday, commit the
diff to git for history.

```bash
#!/bin/bash
set -e

today=$(date -u +%Y-%m-%d)
yesterday=$(date -u -d 'yesterday' +%Y-%m-%d)

mkdir -p snapshots
soda pull nyc erm2-nwe9 --all \
  --where "created_date >= '${yesterday}T00:00:00'" \
  -o snapshots/${today}.json

if [ -f snapshots/${yesterday}.json ]; then
  soda diff snapshots/${yesterday}.json snapshots/${today}.json --format json \
    > snapshots/diff_${today}.json
fi

cd snapshots
git add . && git commit -m "snapshot ${today}" && git push
```

After a week you've got a git-versioned archive of exactly what changed
day to day, browseable in your editor with `git log -p snapshots/diff_*.json`.

## Pattern 3: a daily summary email

Compose stats + diff into a one-shot report:

```bash
#!/bin/bash
soda stats nyc erm2-nwe9 --where "created_date >= '$(date -u +%Y-%m-%d)'" \
  > today_stats.txt

soda pull nyc erm2-nwe9 --all \
  --where "created_date >= '$(date -u +%Y-%m-%d)'" -o today.json

soda diff yesterday.json today.json > daily_diff.txt
mv today.json yesterday.json

mail -s "NYC 311 daily" you@example.com < <(cat today_stats.txt daily_diff.txt)
```
