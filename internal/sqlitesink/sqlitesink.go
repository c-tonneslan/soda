// Package sqlitesink writes Socrata rows into a SQLite database.
//
// Socrata returns each row as an arbitrary JSON object; column names and
// types come from the dataset metadata. We declare a single table per
// dataset, named after the four-by-four (so multiple datasets coexist in
// one .db file), with column types inferred from the dataset schema.
//
// Reruns are idempotent: each row is upserted on its `:id` (Socrata's
// internal row identifier), so re-pulling refreshes rather than duplicates.
package sqlitesink

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/c-tonneslan/soda/internal/socrata"
)

// Sink writes rows for one dataset into one SQLite database.
type Sink struct {
	DB     *sql.DB
	Table  string
	cols   []socrata.Column
	colSet map[string]struct{}
}

// Open creates or opens path and prepares a table named after the dataset.
// Columns are derived from schema; any extra fields that show up at write
// time get added on the fly so dataset evolution doesn't break old DBs.
func Open(path string, schema *socrata.Schema) (*Sink, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	table := tableName(schema.ID)
	s := &Sink{DB: db, Table: table, cols: schema.Columns, colSet: map[string]struct{}{}}
	if err := s.ensureTable(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Insert writes rows to the table. Existing rows with the same `:id` get
// replaced.
func (s *Sink) Insert(rows []map[string]any) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, row := range rows {
		// If the row carries unknown columns (e.g. `:@computed_region_*`), add them.
		for k := range row {
			if _, ok := s.colSet[k]; !ok {
				if err := s.addColumn(tx, k); err != nil {
					return 0, err
				}
			}
		}
		cols := make([]string, 0, len(s.cols))
		placeholders := make([]string, 0, len(s.cols))
		values := make([]any, 0, len(s.cols))
		for name := range s.colSet {
			cols = append(cols, quote(name))
			placeholders = append(placeholders, "?")
			values = append(values, scalarify(row[name]))
		}
		stmt := fmt.Sprintf(
			"INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
			s.Table, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
		)
		if _, err := tx.Exec(stmt, values...); err != nil {
			return 0, fmt.Errorf("insert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(rows), nil
}

// Close releases the underlying database handle.
func (s *Sink) Close() error { return s.DB.Close() }

// ----- internals

func (s *Sink) ensureTable() error {
	parts := make([]string, 0, len(s.cols)+1)
	hasID := false
	for _, c := range s.cols {
		s.colSet[c.FieldName] = struct{}{}
		decl := quote(c.FieldName) + " " + sqliteType(c.DataType)
		if c.FieldName == ":id" {
			decl += " PRIMARY KEY"
			hasID = true
		}
		parts = append(parts, decl)
	}
	if !hasID {
		// SODA always exposes :id, but schemas from /api/views.json sometimes
		// omit it. Add it manually so upserts work.
		parts = append([]string{`":id" TEXT PRIMARY KEY`}, parts...)
		s.colSet[":id"] = struct{}{}
	}
	stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", s.Table, strings.Join(parts, ", "))
	_, err := s.DB.Exec(stmt)
	return err
}

func (s *Sink) addColumn(tx *sql.Tx, name string) error {
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s TEXT", s.Table, quote(name))); err != nil {
		return fmt.Errorf("add column %q: %w", name, err)
	}
	s.colSet[name] = struct{}{}
	return nil
}

// scalarify flattens nested JSON values to a TEXT representation. SQLite
// accepts strings, numbers, and nils natively; everything else (objects,
// arrays) gets JSON-encoded so the row still lands.
func scalarify(v any) any {
	switch x := v.(type) {
	case nil, string, float64, int, int64, bool:
		return x
	default:
		// JSON-encode anything Socrata gave us as an object (e.g. point types)
		b, err := jsonMarshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(b)
	}
}

// sqliteType maps a SODA type to a SQLite affinity.
func sqliteType(t string) string {
	switch strings.ToLower(t) {
	case "number":
		return "REAL"
	case "checkbox":
		return "INTEGER"
	case "calendar_date", "floating_timestamp", "fixed_timestamp", "date":
		return "TEXT" // ISO 8601 strings are easier to filter than UNIX timestamps
	default:
		return "TEXT"
	}
}

// tableName converts a four-by-four into a valid SQLite identifier.
func tableName(id string) string {
	return "d_" + strings.ReplaceAll(id, "-", "_")
}

// quote double-quotes a column name; SODA's `:id` etc. need it.
func quote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
