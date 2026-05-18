package sqlitesink

import (
	"path/filepath"
	"testing"

	"github.com/c-tonneslan/soda/internal/socrata"
)

func makeSchema() *socrata.Schema {
	return &socrata.Schema{
		ID:   "abcd-1234",
		Name: "Test Dataset",
		Columns: []socrata.Column{
			{FieldName: ":id", Name: ":id", DataType: "text"},
			{FieldName: "borough", Name: "Borough", DataType: "text"},
			{FieldName: "count", Name: "Count", DataType: "number"},
			{FieldName: "created_date", Name: "Created", DataType: "calendar_date"},
		},
	}
}

func TestInsertCreatesRows(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	s, err := Open(db, makeSchema())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	rows := []map[string]any{
		{":id": "row-1", "borough": "BROOKLYN", "count": float64(5)},
		{":id": "row-2", "borough": "MANHATTAN", "count": float64(3)},
	}
	n, err := s.Insert(rows)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if n != 2 {
		t.Errorf("n: got %d want 2", n)
	}
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM " + s.Table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count: got %d want 2", count)
	}
}

func TestInsertIdempotent(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	s, err := Open(db, makeSchema())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	row := []map[string]any{{":id": "row-1", "borough": "BROOKLYN"}}
	_, _ = s.Insert(row)
	_, _ = s.Insert(row)
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM " + s.Table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected idempotent insert; got %d rows", count)
	}
}

func TestInsertAddsUnknownColumns(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	s, err := Open(db, makeSchema())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows := []map[string]any{
		{":id": "row-1", ":@computed_region_f5dn_yrer": "12345"},
	}
	if _, err := s.Insert(rows); err != nil {
		t.Fatalf("Insert with unknown column: %v", err)
	}
	var v string
	if err := s.DB.QueryRow(`SELECT ":@computed_region_f5dn_yrer" FROM ` + s.Table).Scan(&v); err != nil {
		t.Fatalf("read unknown column: %v", err)
	}
	if v != "12345" {
		t.Errorf("computed region value: got %q want %q", v, "12345")
	}
}

func TestInsertScalarifiesObjects(t *testing.T) {
	db := filepath.Join(t.TempDir(), "t.db")
	s, err := Open(db, makeSchema())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	loc := map[string]any{"type": "Point", "coordinates": []any{-73.9, 40.7}}
	if _, err := s.Insert([]map[string]any{{":id": "r1", "borough": loc}}); err != nil {
		t.Fatalf("Insert with object: %v", err)
	}
	var v string
	if err := s.DB.QueryRow(`SELECT borough FROM ` + s.Table).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v == "" || v[0] != '{' {
		t.Errorf("expected JSON-encoded object, got %q", v)
	}
}

func TestSqliteType(t *testing.T) {
	cases := map[string]string{
		"number":              "REAL",
		"checkbox":            "INTEGER",
		"text":                "TEXT",
		"calendar_date":       "TEXT",
		"floating_timestamp":  "TEXT",
		"point":               "TEXT",
		"":                    "TEXT",
	}
	for in, want := range cases {
		if got := sqliteType(in); got != want {
			t.Errorf("sqliteType(%q): got %q want %q", in, got, want)
		}
	}
}

func TestTableNameSafe(t *testing.T) {
	if got := tableName("abc-1234"); got != "d_abc_1234" {
		t.Errorf("tableName: got %q", got)
	}
}
