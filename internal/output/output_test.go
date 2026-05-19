package output

import (
	"bytes"
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
