package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func countRows(t *testing.T, s *Store, table string) int {
	t.Helper()

	var n int
	require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&n))
	return n
}

func TestEnsureSchemaTwice(t *testing.T) {
	s := openTestStore(t)

	_, err := s.db.Exec("INSERT INTO nameservers (ip_address, status, checked_at, created_at) VALUES ('192.0.2.1', true, NOW(), NOW())")
	require.NoError(t, err)

	// the insert trigger records the first check
	assert.Equal(t, 1, countRows(t, s, "nameserver_checks"))

	// a second run keeps the data and does not duplicate the triggers
	require.NoError(t, s.EnsureSchema())

	_, err = s.db.Exec("INSERT INTO nameservers (ip_address, status, checked_at, created_at) VALUES ('192.0.2.2', true, NOW(), NOW())")
	require.NoError(t, err)

	assert.Equal(t, 2, countRows(t, s, "nameservers"))
	assert.Equal(t, 2, countRows(t, s, "nameserver_checks"))
}

func TestEnsureSchemaUpdateTrigger(t *testing.T) {
	s := openTestStore(t)

	_, err := s.db.Exec("INSERT INTO nameservers (id, ip_address, status, checked_at, created_at) VALUES (1, '192.0.2.1', true, NOW() - INTERVAL '1 hour', NOW())")
	require.NoError(t, err)

	_, err = s.db.Exec("UPDATE nameservers SET status = false, error = 'timeout', checked_at = NOW() WHERE id = 1")
	require.NoError(t, err)

	var (
		reliability     float64
		statusChangedAt bool
	)
	err = s.db.QueryRow("SELECT reliability, status_changed_at IS NOT NULL FROM nameservers WHERE id = 1").Scan(&reliability, &statusChangedAt)
	require.NoError(t, err)

	assert.Equal(t, 2, countRows(t, s, "nameserver_checks"))
	assert.InDelta(t, 0.5, reliability, 0.001)
	assert.True(t, statusChangedAt)
}

func TestEnsureSchemaRailsDatabase(t *testing.T) {
	s := openEmptyStore(t)

	// create the schema like the Rails migrations did, with plain CREATE statements
	rails := strings.NewReplacer(
		" IF NOT EXISTS", "",
		"CREATE OR REPLACE TRIGGER", "CREATE TRIGGER",
		"EXECUTE FUNCTION", "EXECUTE PROCEDURE",
	).Replace(schema)
	_, err := s.db.Exec(rails)
	require.NoError(t, err)

	_, err = s.db.Exec("INSERT INTO nameservers (ip_address, status, checked_at, created_at) VALUES ('192.0.2.1', true, NOW(), NOW())")
	require.NoError(t, err)

	require.NoError(t, s.EnsureSchema())

	_, err = s.db.Exec("INSERT INTO nameservers (ip_address, status, checked_at, created_at) VALUES ('192.0.2.2', true, NOW(), NOW())")
	require.NoError(t, err)

	assert.Equal(t, 2, countRows(t, s, "nameservers"))
	assert.Equal(t, 2, countRows(t, s, "nameserver_checks"))
}
