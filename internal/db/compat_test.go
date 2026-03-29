package db

import (
	"database/sql"
	"testing"
)

// TestMigrationIdempotent verifies migrate() is safe to run multiple times
// against the same database (simulates upgrading an existing installation).
func TestMigrationIdempotent(t *testing.T) {
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	defer s.Close()

	if err := s.migrate(); err != nil {
		t.Errorf("second migrate() call failed: %v", err)
	}
	if err := s.migrate(); err != nil {
		t.Errorf("third migrate() call failed: %v", err)
	}
}

// TestSchemaRequiredColumns ensures the entries table always has all required
// columns. Removing or renaming a column is a breaking change.
func TestSchemaRequiredColumns(t *testing.T) {
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	defer s.Close()

	rows, err := s.db.Query("PRAGMA table_info(entries)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	cols := map[string]struct{}{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		cols[name] = struct{}{}
	}

	required := []string{"id", "category", "key", "value", "created_at", "updated_at"}
	for _, col := range required {
		if _, ok := cols[col]; !ok {
			t.Errorf("required column %q is missing from entries table — this is a breaking change", col)
		}
	}
}

// TestMigrationPreservesExistingRows verifies that running migrate() against a
// database that already has data does not delete or alter any existing rows.
func TestMigrationPreservesExistingRows(t *testing.T) {
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	defer s.Close()

	_, err = s.db.Exec(
		`INSERT INTO entries (category, key, value) VALUES ('test', 'mykey', 'myvalue')`,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Simulate an upgrade by running migrate again.
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate after existing data: %v", err)
	}

	var value string
	err = s.db.QueryRow("SELECT value FROM entries WHERE key = 'mykey'").Scan(&value)
	if err != nil {
		t.Fatalf("row missing after migrate: %v", err)
	}
	if value != "myvalue" {
		t.Errorf("value = %q after migrate, want %q — migrate must not alter existing data", value, "myvalue")
	}
}

// TestValuesStoredAsPlainText verifies the DB layer stores values verbatim and
// never transforms them. This is the contract that the API presentation layer
// relies on when wrapping values for responses.
func TestValuesStoredAsPlainText(t *testing.T) {
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	defer s.Close()

	cases := []struct {
		key   string
		value string
	}{
		{"plain", "San Jose"},
		{"json_obj", `{"lat": 37.3, "lon": -121.9}`},
		{"json_arr", `["a","b","c"]`},
		{"number_str", "72"},
		{"bool_str", "true"},
		{"empty", ""},
		{"whitespace", "  hello  "},
	}

	for _, tc := range cases {
		s.db.Exec(`INSERT INTO entries (category, key, value) VALUES ('test', ?, ?)`, tc.key, tc.value)

		var got string
		s.db.QueryRow("SELECT value FROM entries WHERE key = ?", tc.key).Scan(&got)
		if got != tc.value {
			t.Errorf("key %q: stored %q, want %q — DB must not transform values", tc.key, got, tc.value)
		}
	}
}

// TestUniqueConstraintCategoryKey verifies the uniqueness constraint on
// (category, key) still exists. Removing it would be a breaking behavioral change.
func TestUniqueConstraintCategoryKey(t *testing.T) {
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	defer s.Close()

	s.db.Exec(`INSERT INTO entries (category, key, value) VALUES ('cat', 'k', 'v1')`)
	_, err = s.db.Exec(`INSERT INTO entries (category, key, value) VALUES ('cat', 'k', 'v2')`)
	if err == nil {
		t.Error("duplicate (category, key) insert should fail — UNIQUE constraint must remain")
	}
}

// TestCategoryDefaultsToDefault verifies that omitting category stores "default",
// which is the documented contract for clients that don't specify a category.
func TestCategoryDefaultsToDefault(t *testing.T) {
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	defer s.Close()

	s.db.Exec(`INSERT INTO entries (key, value) VALUES ('k', 'v')`)

	var cat string
	s.db.QueryRow("SELECT category FROM entries WHERE key = 'k'").Scan(&cat)
	if cat != "default" {
		t.Errorf("category = %q, want %q — default category must be 'default'", cat, "default")
	}
}
