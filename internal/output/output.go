// Package output renders results for the CLI in a couple of formats.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

// Table writes a left-aligned, tab-aligned table of rows to w. The header row
// is the first element of rows; subsequent slices must be the same length.
func Table(w io.Writer, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	return tw.Flush()
}

// JSON pretty-prints v to w as a JSON array.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// geometryTypes is the set of GeoJSON geometry type names. Socrata "point",
// "line", and "polygon" columns already serialize as these objects inside a
// row, so detecting one is just a matter of recognising the shape.
var geometryTypes = map[string]bool{
	"Point":      true,
	"MultiPoint": true,
	"LineString": true, "MultiLineString": true,
	"Polygon": true, "MultiPolygon": true,
	"GeometryCollection": true,
}

// GeoJSON writes rows as a GeoJSON FeatureCollection. For each row the first
// field (by name) holding a GeoJSON geometry object becomes the feature's
// geometry and the remaining fields become its properties. A row with no
// geometry field gets a null geometry, which is still valid GeoJSON.
func GeoJSON(w io.Writer, rows []map[string]any) error {
	features := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		geomKey := geometryField(row)
		props := make(map[string]any, len(row))
		for k, v := range row {
			if k != geomKey {
				props[k] = v
			}
		}
		var geom any
		if geomKey != "" {
			geom = row[geomKey]
		}
		features = append(features, map[string]any{
			"type":       "Feature",
			"geometry":   geom,
			"properties": props,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"type":     "FeatureCollection",
		"features": features,
	})
}

// geometryField returns the name of the field holding a GeoJSON geometry,
// or "" if the row has none. When more than one field qualifies the
// lexicographically first name wins, so the output is deterministic.
func geometryField(row map[string]any) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if isGeometry(row[k]) {
			return k
		}
	}
	return ""
}

func isGeometry(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	t, ok := m["type"].(string)
	if !ok || !geometryTypes[t] {
		return false
	}
	if t == "GeometryCollection" {
		_, ok := m["geometries"]
		return ok
	}
	_, ok = m["coordinates"]
	return ok
}
