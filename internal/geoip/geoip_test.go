package geoip

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeCityDB writes a GeoLite2 City database that maps 192.0.2.0/24 to
// the given country and city.
func writeCityDB(t *testing.T, path, country, city string) {
	t.Helper()

	writeDB(t, path, EditionCity, mmdbtype.Map{
		"country": mmdbtype.Map{"iso_code": mmdbtype.String(country)},
		"city":    mmdbtype.Map{"names": mmdbtype.Map{"en": mmdbtype.String(city)}},
	})
}

// writeDB writes a database of the given type that maps 192.0.2.0/24 to value.
func writeDB(t *testing.T, path, databaseType string, value mmdbtype.DataType) {
	t.Helper()

	tree, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            databaseType,
		IncludeReservedNetworks: true,
	})
	require.NoError(t, err)

	_, network, err := net.ParseCIDR("192.0.2.0/24")
	require.NoError(t, err)
	require.NoError(t, tree.Insert(network, value))

	// write to a temporary file and rename, like the downloader does
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	require.NoError(t, err)
	_, err = tree.WriteTo(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Rename(tmp, path))
}

func cityName(t *testing.T, db *Database) string {
	t.Helper()

	record, err := db.City(net.ParseIP("192.0.2.1"))
	require.NoError(t, err)
	return record.City.Names.English
}

func TestCity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")
	writeCityDB(t, path, "DE", "Berlin")

	db, err := New(path)
	require.NoError(t, err)
	defer db.Close()

	record, err := db.City(net.ParseIP("192.0.2.1"))
	require.NoError(t, err)
	assert.Equal(t, "DE", record.Country.ISOCode)
	assert.Equal(t, "Berlin", record.City.Names.English)
}

func TestReloadOnChange(t *testing.T) {
	defer func(d time.Duration) { reloadInterval = d }(reloadInterval)
	reloadInterval = 10 * time.Millisecond

	path := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")
	writeCityDB(t, path, "DE", "Berlin")

	db, err := New(path)
	require.NoError(t, err)
	defer db.Close()

	// replace the file after the first check has run
	time.Sleep(50 * time.Millisecond)
	writeCityDB(t, path, "FR", "Paris")
	later := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(path, later, later))

	assert.Eventually(t, func() bool { return cityName(t, db) == "Paris" }, time.Second, 10*time.Millisecond)
}
