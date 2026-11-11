package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertPageData(t *testing.T, s *Store) {
	t.Helper()

	_, err := s.db.Exec(`INSERT INTO nameservers (id, ip_address, country_code, status, checked_at, created_at) VALUES
		(1, '192.0.2.1', 'DE', true,  NOW() - INTERVAL '3 hours', NOW()),
		(2, '192.0.2.2', 'DE', true,  NOW() - INTERVAL '1 hour', NOW()),
		(3, '192.0.2.3', 'DE', false, NOW(), NOW()),
		(4, '192.0.2.4', 'FR', true,  NOW() - INTERVAL '2 hours', NOW()),
		(5, '192.0.2.5', '',   true,  NOW(), NOW()),
		(6, '192.0.2.6', NULL, true,  NOW() - INTERVAL '5 hours', NOW()),
		(7, '192.0.2.7', 'US', false, NOW(), NOW())`)
	require.NoError(t, err)
}

func addresses(list []*Nameserver) []string {
	var result []string
	for _, ns := range list {
		result = append(result, ns.Address)
	}
	return result
}

func TestCountValid(t *testing.T) {
	s := openTestStore(t)
	insertPageData(t, s)

	n, err := s.CountValid()
	require.NoError(t, err)
	assert.Equal(t, 5, n)
}

func TestValidCountries(t *testing.T) {
	s := openTestStore(t)
	insertPageData(t, s)

	codes, err := s.ValidCountries()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"DE", "FR"}, codes, "no empty codes, no countries with only invalid nameservers")
}

func TestRecentValid(t *testing.T) {
	s := openTestStore(t)
	insertPageData(t, s)

	list, err := s.RecentValid(3)
	require.NoError(t, err)
	assert.Equal(t, []string{"192.0.2.5", "192.0.2.2", "192.0.2.4"}, addresses(list))
}

func TestCountryValid(t *testing.T) {
	s := openTestStore(t)
	insertPageData(t, s)

	list, total, err := s.CountryValid("de", 1)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Equal(t, []string{"192.0.2.2"}, addresses(list))
	assert.Equal(t, "DE", *list[0].Country)

	list, total, err = s.CountryValid("xx", 10)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
}
