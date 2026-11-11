package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurgeNameservers(t *testing.T) {
	s := openTestStore(t)

	_, err := s.db.Exec(`INSERT INTO nameservers (id, ip_address, status, status_changed_at, checked_at, created_at) VALUES
		-- valid for a long time: keep
		(1, '192.0.2.1', true,  NOW() - INTERVAL '40 days', NOW(), NOW() - INTERVAL '1 year'),
		-- invalid for a long time and recently checked: purge
		(2, '192.0.2.2', false, NOW() - INTERVAL '40 days', NOW(), NOW() - INTERVAL '1 year'),
		-- invalid for a short time: keep
		(3, '192.0.2.3', false, NOW() - INTERVAL '10 days', NOW(), NOW() - INTERVAL '1 year'),
		-- invalid for a long time, but not checked recently: keep
		(4, '192.0.2.4', false, NOW() - INTERVAL '40 days', NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 year')`)
	require.NoError(t, err)

	purged, err := s.PurgeNameservers(30)
	require.NoError(t, err)
	assert.EqualValues(t, 1, purged)

	var remaining []int
	rows, err := s.db.Query("SELECT id FROM nameservers ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id int
		require.NoError(t, rows.Scan(&id))
		remaining = append(remaining, id)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []int{1, 3, 4}, remaining)
}

func TestPurgeChecks(t *testing.T) {
	s := openTestStore(t)

	// the insert trigger records a check with created_at = checked_at
	_, err := s.db.Exec(`INSERT INTO nameservers (ip_address, status, checked_at, created_at) VALUES
		('192.0.2.1', true, NOW() - INTERVAL '40 days', NOW() - INTERVAL '40 days'),
		('192.0.2.2', true, NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days')`)
	require.NoError(t, err)

	purged, err := s.PurgeChecks(30)
	require.NoError(t, err)
	assert.EqualValues(t, 1, purged)
	assert.Equal(t, 1, countRows(t, s, "nameserver_checks"))
}
