package store

import _ "embed"

//go:embed schema.sql
var schema string

// EnsureSchema creates missing tables, indexes, functions and triggers.
// Existing objects and data are left untouched, except that the trigger
// function and triggers are replaced by their current definition.
func (s *Store) EnsureSchema() error {
	// a query with several statements runs in a single implicit transaction
	_, err := s.db.Exec(schema)
	return err
}
