package geoip

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// archive packs content as <edition>_20260101/<edition>.mmdb into a tar.gz.
func archive(t *testing.T, edition string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: edition + "_20260101/", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: edition + "_20260101/COPYRIGHT.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 2}))
	_, err := tw.Write([]byte("(c"))
	require.NoError(t, err)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: edition + "_20260101/" + edition + ".mmdb", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))}))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// maxmind serves the given archives per edition and counts the requests.
func maxmind(t *testing.T, archives map[string][]byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		user, pass, ok := r.BasicAuth()
		if !ok || user != "42" || pass != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		edition := strings.Split(r.URL.Path, "/")[1]
		data, found := archives[edition]
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

func validArchives(t *testing.T) map[string][]byte {
	t.Helper()
	dir := t.TempDir()

	city := filepath.Join(dir, "city.mmdb")
	writeCityDB(t, city, "DE", "Berlin")
	asn := filepath.Join(dir, "asn.mmdb")
	writeDB(t, asn, EditionASN, mmdbtype.Map{
		"autonomous_system_number":       mmdbtype.Uint32(64496),
		"autonomous_system_organization": mmdbtype.String("Example AS"),
	})

	read := func(path string) []byte {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		return data
	}
	return map[string][]byte{
		EditionCity: archive(t, EditionCity, read(city)),
		EditionASN:  archive(t, EditionASN, read(asn)),
	}
}

func newUpdater(t *testing.T, srv *httptest.Server) *Updater {
	return &Updater{
		Dir:        t.TempDir(),
		AccountID:  "42",
		LicenseKey: "secret",
		MaxAge:     24 * time.Hour,
		URL:        srv.URL + "/%s/download",
	}
}

func TestUpdate(t *testing.T) {
	srv, requests := maxmind(t, validArchives(t))
	u := newUpdater(t, srv)

	assert.False(t, u.Available())
	require.NoError(t, u.Update(context.Background()))
	assert.True(t, u.Available())
	assert.EqualValues(t, 2, requests.Load())

	city, err := New(u.Path(EditionCity))
	require.NoError(t, err)
	defer city.Close()
	assert.Equal(t, "Berlin", cityName(t, city))

	asn, err := New(u.Path(EditionASN))
	require.NoError(t, err)
	defer asn.Close()
	record, err := asn.ASN(net.ParseIP("192.0.2.1"))
	require.NoError(t, err)
	assert.EqualValues(t, 64496, record.AutonomousSystemNumber)

	// fresh files are not downloaded again
	require.NoError(t, u.Update(context.Background()))
	assert.EqualValues(t, 2, requests.Load())
}

func TestUpdateUnauthorized(t *testing.T) {
	srv, _ := maxmind(t, validArchives(t))
	u := newUpdater(t, srv)
	u.LicenseKey = "wrong"

	err := u.Update(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.False(t, u.Available())
}

func TestUpdateKeepsFileOnInvalidDownload(t *testing.T) {
	archives := validArchives(t)
	archives[EditionCity] = archive(t, EditionCity, []byte("not a database"))
	srv, _ := maxmind(t, archives)
	u := newUpdater(t, srv)
	u.MaxAge = 0

	writeCityDB(t, u.Path(EditionCity), "FR", "Paris")

	require.Error(t, u.Update(context.Background()))

	city, err := New(u.Path(EditionCity))
	require.NoError(t, err)
	defer city.Close()
	assert.Equal(t, "Paris", cityName(t, city))

	entries, err := os.ReadDir(u.Dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{"GeoLite2-City.mmdb", "GeoLite2-ASN.mmdb"}, names, "no temporary files left")
}

func TestUpdateWrongEdition(t *testing.T) {
	// an ASN database named like the City database
	asn := filepath.Join(t.TempDir(), "asn.mmdb")
	writeDB(t, asn, EditionASN, mmdbtype.Map{"autonomous_system_number": mmdbtype.Uint32(64496)})
	content, err := os.ReadFile(asn)
	require.NoError(t, err)

	archives := validArchives(t)
	archives[EditionCity] = archive(t, EditionCity, content)
	srv, _ := maxmind(t, archives)
	u := newUpdater(t, srv)

	err = u.Update(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unexpected database type "GeoLite2-ASN"`)
	assert.NoFileExists(t, u.Path(EditionCity))
}

func TestUpdateMissingFile(t *testing.T) {
	archives := validArchives(t)
	archives[EditionCity] = archives[EditionASN]
	srv, _ := maxmind(t, archives)
	u := newUpdater(t, srv)

	err := u.Update(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GeoLite2-City.mmdb not found in archive")
}
