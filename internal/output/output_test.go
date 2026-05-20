package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTableAlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	err := Table(&buf, [][]string{
		{"id", "name", "rows"},
		{"abcd-1234", "311 calls", "12000000"},
		{"x", "y", "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d:\n%s", len(lines), got)
	}
	// tabwriter pads to the longest entry per column plus the configured
	// padding of 2 spaces; if alignment ever breaks, this catches it.
	if !strings.HasPrefix(lines[1], "abcd-1234  311 calls") {
		t.Errorf("alignment off: %q", lines[1])
	}
}

func TestJSONPrettyPrints(t *testing.T) {
	var buf bytes.Buffer
	err := JSON(&buf, map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": 1\n}\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGeoJSONBuildsFeatureCollection(t *testing.T) {
	rows := []map[string]any{
		{
			"unique_key": "1",
			"complaint":  "Noise",
			"location": map[string]any{
				"type":        "Point",
				"coordinates": []any{-73.9, 40.7},
			},
		},
		// A row with no geometry still becomes a feature, with null geometry.
		{"unique_key": "2", "complaint": "Pothole"},
	}

	var buf bytes.Buffer
	if err := GeoJSON(&buf, rows); err != nil {
		t.Fatal(err)
	}

	var fc struct {
		Type     string `json:"type"`
		Features []struct {
			Type       string         `json:"type"`
			Geometry   map[string]any `json:"geometry"`
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(buf.Bytes(), &fc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if fc.Type != "FeatureCollection" {
		t.Errorf("type = %q, want FeatureCollection", fc.Type)
	}
	if len(fc.Features) != 2 {
		t.Fatalf("want 2 features, got %d", len(fc.Features))
	}

	f0 := fc.Features[0]
	if f0.Type != "Feature" {
		t.Errorf("feature type = %q, want Feature", f0.Type)
	}
	if f0.Geometry["type"] != "Point" {
		t.Errorf("geometry type = %v, want Point", f0.Geometry["type"])
	}
	// The geometry field must not leak back into properties.
	if _, ok := f0.Properties["location"]; ok {
		t.Error("location should have moved to geometry, not stayed in properties")
	}
	if f0.Properties["complaint"] != "Noise" {
		t.Errorf("properties.complaint = %v, want Noise", f0.Properties["complaint"])
	}

	if fc.Features[1].Geometry != nil {
		t.Errorf("row without a geometry should get null geometry, got %v", fc.Features[1].Geometry)
	}
}

func TestGeoJSONIgnoresNonGeometryObjects(t *testing.T) {
	// A nested object that merely has a "type" key is not a geometry.
	rows := []map[string]any{{
		"id":   "1",
		"meta": map[string]any{"type": "residential", "units": 4},
	}}
	var buf bytes.Buffer
	if err := GeoJSON(&buf, rows); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"geometry": null`) {
		t.Errorf("expected null geometry, got:\n%s", buf.String())
	}
}
