// Package diff computes a row-level diff between two snapshots of a Socrata
// dataset.
//
// "Row" is a map[string]any. Rows are identified by a key column (defaults to
// `:id`, Socrata's internal row identifier). A row that appears in one
// snapshot but not the other is added/removed; a row that appears in both
// with any field different is changed.
package diff

import (
	"fmt"
	"reflect"
	"sort"
)

// Row is one record from a Socrata snapshot.
type Row = map[string]any

// FieldChange is one field that differs between two versions of the same row.
type FieldChange struct {
	Field string `json:"field"`
	Old   any    `json:"old"`
	New   any    `json:"new"`
}

// ChangedRow is a row that exists in both snapshots but has at least one
// differing field.
type ChangedRow struct {
	Key     string        `json:"key"`
	Changes []FieldChange `json:"changes"`
}

// Result is the full output of a comparison.
type Result struct {
	Key     string       `json:"key"`         // the key column used
	Added   []Row        `json:"added"`
	Removed []Row        `json:"removed"`
	Changed []ChangedRow `json:"changed"`
}

// Compute returns the diff of two row slices identified by keyCol. If keyCol
// is empty, ":id" is used.
func Compute(a, b []Row, keyCol string) (*Result, error) {
	if keyCol == "" {
		keyCol = ":id"
	}
	aMap, err := indexBy(a, keyCol)
	if err != nil {
		return nil, fmt.Errorf("snapshot A: %w", err)
	}
	bMap, err := indexBy(b, keyCol)
	if err != nil {
		return nil, fmt.Errorf("snapshot B: %w", err)
	}
	res := &Result{Key: keyCol}

	// Added: in B but not A.
	for k, row := range bMap {
		if _, ok := aMap[k]; !ok {
			res.Added = append(res.Added, row)
		}
	}
	// Removed: in A but not B.
	for k, row := range aMap {
		if _, ok := bMap[k]; !ok {
			res.Removed = append(res.Removed, row)
		}
	}
	// Changed: in both, with any field different.
	for k, oldRow := range aMap {
		newRow, ok := bMap[k]
		if !ok {
			continue
		}
		changes := compareRows(oldRow, newRow)
		if len(changes) > 0 {
			res.Changed = append(res.Changed, ChangedRow{Key: k, Changes: changes})
		}
	}
	// Deterministic ordering so output is reproducible.
	sort.Slice(res.Changed, func(i, j int) bool { return res.Changed[i].Key < res.Changed[j].Key })
	return res, nil
}

func indexBy(rows []Row, keyCol string) (map[string]Row, error) {
	out := make(map[string]Row, len(rows))
	for i, r := range rows {
		v, ok := r[keyCol]
		if !ok {
			return nil, fmt.Errorf("row %d missing key column %q", i, keyCol)
		}
		key := fmt.Sprint(v)
		out[key] = r
	}
	return out, nil
}

func compareRows(a, b Row) []FieldChange {
	keys := map[string]struct{}{}
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}
	var changes []FieldChange
	for k := range keys {
		if !reflect.DeepEqual(a[k], b[k]) {
			changes = append(changes, FieldChange{Field: k, Old: a[k], New: b[k]})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Field < changes[j].Field })
	return changes
}
