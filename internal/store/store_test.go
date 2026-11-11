package store

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openEmptyStore opens the database given by TEST_DSN and drops all tables.
// All data in that database is lost.
func openEmptyStore(t *testing.T) *Store {
	t.Helper()

	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set")
	}

	s, err := Open(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	_, err = s.db.Exec(`
		DROP TABLE IF EXISTS nameserver_checks, nameservers CASCADE;
		DROP FUNCTION IF EXISTS insert_nameserver_check() CASCADE;`)
	require.NoError(t, err)

	return s
}

// openTestStore opens an empty database given by TEST_DSN and creates the
// schema.
func openTestStore(t *testing.T) *Store {
	t.Helper()

	s := openEmptyStore(t)
	require.NoError(t, s.EnsureSchema())

	return s
}

func TestEachAddressAfterUpdate(t *testing.T) {
	s := openTestStore(t)

	_, err := s.db.Exec(`INSERT INTO nameservers (id, ip_address, status, checked_at, created_at) VALUES
		(1, '192.0.2.1', true, NOW(), NOW()),
		(2, '192.0.2.2', true, NOW(), NOW()),
		(3, '192.0.2.3', true, NOW(), NOW())`)
	require.NoError(t, err)

	// An update stores a new row version behind the others, so without
	// ORDER BY the first batch returns the IDs 2 and 3.
	_, err = s.db.Exec("UPDATE nameservers SET name='updated' WHERE id=1")
	require.NoError(t, err)

	visited := map[int]string{}
	err = s.EachAddress(2, func(id int, address net.IP) bool {
		visited[id] = address.String()
		return true
	})

	require.NoError(t, err)
	assert.Equal(t, map[int]string{
		1: "192.0.2.1",
		2: "192.0.2.2",
		3: "192.0.2.3",
	}, visited)
}
