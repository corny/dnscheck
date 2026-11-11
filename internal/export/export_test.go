package export

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/corny/dnscheck/internal/store"
	"github.com/corny/dnscheck/internal/testdb"
)

func setupNameservers(t *testing.T) *store.Store {
	t.Helper()

	dsn := testdb.DSN(t)
	s, err := store.Open(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	require.NoError(t, s.EnsureSchema())

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`INSERT INTO nameservers (id, ip_address, name, as_number, as_org, country_code, city, status, error, version, dnssec, checked_at, created_at) VALUES
		(1, '192.0.2.1', 'ns1.example.', 64496, 'Example AS', 'de', 'Berlin', true, NULL, 'Knot', true, '2026-01-02 03:04:05', '2025-01-01 00:00:00'),
		(2, '2001:db8::2', NULL, NULL, NULL, NULL, NULL, true, NULL, NULL, false, '2026-01-02 03:04:05', '2025-01-01 00:00:00'),
		(3, '192.0.2.3', NULL, NULL, NULL, 'us', NULL, false, 'i/o timeout', NULL, NULL, '2026-01-02 03:04:05', '2025-01-01 00:00:00')`)
	require.NoError(t, err)

	return s
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	return string(data)
}

func TestExport(t *testing.T) {
	s := setupNameservers(t)
	dir := t.TempDir()

	exporter := Exporter{Destination: dir, Store: s, BatchSize: 2}
	require.NoError(t, exporter.Run())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{
		"nameservers-all.txt", "nameservers-all.csv", "nameservers-all.json",
		"nameservers.txt", "nameservers.csv", "nameservers.json",
		"de.txt", "de.csv", "de.json",
	}, names, "invalid nameservers get no country file")

	assert.Equal(t, "192.0.2.1\n2001:db8::2\n192.0.2.3", readFile(t, dir, "nameservers-all.txt"))
	assert.Equal(t, "192.0.2.1\n2001:db8::2", readFile(t, dir, "nameservers.txt"))
	assert.Equal(t, "192.0.2.1", readFile(t, dir, "de.txt"))

	assert.Equal(t, ""+
		"ip_address,name,as_number,as_org,country_code,city,version,error,dnssec,reliability,checked_at,created_at\n"+
		"192.0.2.1,ns1.example.,64496,Example AS,DE,Berlin,Knot,,true,0.00,2026-01-02T03:04:05Z,2025-01-01T00:00:00Z\n"+
		"2001:db8::2,,,,,,,,false,0.00,2026-01-02T03:04:05Z,2025-01-01T00:00:00Z\n"+
		"192.0.2.3,,,,US,,,i/o timeout,,0.00,2026-01-02T03:04:05Z,2025-01-01T00:00:00Z\n",
		readFile(t, dir, "nameservers-all.csv"))

	var records []map[string]any
	require.NoError(t, json.Unmarshal([]byte(readFile(t, dir, "nameservers.json")), &records))
	require.Len(t, records, 2)
	assert.Equal(t, map[string]any{
		"ip":          "192.0.2.1",
		"name":        "ns1.example.",
		"as_number":   float64(64496),
		"as_org":      "Example AS",
		"country_id":  "DE",
		"city":        "Berlin",
		"version":     "Knot",
		"error":       nil,
		"dnssec":      true,
		"reliability": float64(0),
		"checked_at":  "2026-01-02T03:04:05Z",
		"created_at":  "2025-01-01T00:00:00Z",
	}, records[0])
}

func TestExportTwice(t *testing.T) {
	s := setupNameservers(t)
	dir := t.TempDir()

	exporter := Exporter{Destination: dir, Store: s, BatchSize: 100}
	require.NoError(t, exporter.Run())
	require.NoError(t, exporter.Run())

	assert.Equal(t, "192.0.2.1\n2001:db8::2", readFile(t, dir, "nameservers.txt"))
}

func TestExportFailureKeepsFiles(t *testing.T) {
	s := setupNameservers(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nameservers.txt"), []byte("old"), 0o644))

	// the temporary files cannot be created
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	exporter := Exporter{Destination: dir, Store: s, BatchSize: 100}
	require.Error(t, exporter.Run())

	assert.Equal(t, "old", readFile(t, dir, "nameservers.txt"))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary files left")
}
