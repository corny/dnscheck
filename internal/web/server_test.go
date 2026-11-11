package web

import (
	"database/sql"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/corny/dnscheck/internal/check"
	"github.com/corny/dnscheck/internal/store"
	"github.com/corny/dnscheck/internal/testdb"
)

type fakeChecker struct {
	ready bool
	err   string // error of every check
}

func (f *fakeChecker) Ready() bool { return f.ready }

func (f *fakeChecker) ExecuteJob(job *check.Job) {
	job.Status = f.err == ""
	job.Error = f.err
}

type testServer struct {
	handler http.Handler
	checker *fakeChecker
	db      *sql.DB
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	dsn := testdb.DSN(t)
	st, err := store.Open(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { st.Close() })
	require.NoError(t, st.EnsureSchema())

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`INSERT INTO nameservers (ip_address, name, as_number, as_org, country_code, city, status, error, version, dnssec, checked_at, created_at) VALUES
		('192.0.2.1', 'ns1.example.', 64496, 'Example AS', 'DE', 'Berlin', true, NULL, 'Knot', true, NOW() AT TIME ZONE 'utc', NOW()),
		('192.0.2.2', NULL, NULL, NULL, 'AT', NULL, true, NULL, NULL, false, NOW() AT TIME ZONE 'utc', NOW()),
		('192.0.2.3', NULL, NULL, NULL, 'DE', NULL, false, 'i/o timeout', NULL, NULL, NOW() AT TIME ZONE 'utc', NOW())`)
	require.NoError(t, err)

	exportDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(exportDir, "nameservers.csv"), []byte("ip_address\n192.0.2.1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(exportDir, "de.json"), []byte(`[{"ip":"192.0.2.1"}]`), 0o644))

	checker := &fakeChecker{ready: true}
	srv, err := New(Config{
		Store:     st,
		Checker:   checker,
		ExportDir: exportDir,
		Domains:   []string{"wikileaks.org", "does-not-exist.invalid"},
		Reference: "8.8.8.8",
		Timeout:   3 * time.Second,
		PurgeDays: 30,
	})
	require.NoError(t, err)

	return &testServer{handler: srv.Handler(), checker: checker, db: db}
}

// get requests the URL, which may contain a host name.
func (ts *testServer) get(t *testing.T, rawURL string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	rec := httptest.NewRecorder()
	ts.handler.ServeHTTP(rec, req)
	return rec.Result()
}

func (ts *testServer) post(t *testing.T, address string) *http.Response {
	t.Helper()

	body := url.Values{"address": {address}}.Encode()
	req := httptest.NewRequest(http.MethodPost, "/api/nameservers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	ts.handler.ServeHTTP(rec, req)
	return rec.Result()
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(body)
}

func TestLocalesHaveSameKeys(t *testing.T) {
	locales, err := loadLocales()
	require.NoError(t, err)

	de, en := locales["de"], locales["en"]
	assert.ElementsMatch(t, slices.Collect(maps.Keys(de.messages)), slices.Collect(maps.Keys(en.messages)))
	assert.ElementsMatch(t, slices.Collect(maps.Keys(de.countries)), slices.Collect(maps.Keys(en.countries)))
}

func TestLanguage(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		url, lang, text string
	}{
		{"http://www.ungefiltert-surfen.de/", "de", "Zuletzt geprüft"},
		{"http://public-dns.info/", "en", "Recently checked"},
		{"http://public-dns.info/?lang=de", "en", "Recently checked"}, // lang only works locally
		{"http://localhost:8000/", "en", "Recently checked"},
		{"http://localhost:8000/?lang=de", "de", "Zuletzt geprüft"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			res := ts.get(t, tt.url)
			require.Equal(t, http.StatusOK, res.StatusCode)
			body := readBody(t, res)
			assert.Contains(t, body, `<html lang="`+tt.lang+`">`)
			assert.Contains(t, body, tt.text)
		})
	}
}

func TestLanguageLinks(t *testing.T) {
	ts := newTestServer(t)

	body := readBody(t, ts.get(t, "http://public-dns.info/testing"))
	assert.Contains(t, body, `<link rel="alternate" hreflang="de" href="https://www.ungefiltert-surfen.de/testing">`)
	assert.Contains(t, body, `<link rel="alternate" hreflang="en" href="https://public-dns.info/testing">`)
	assert.Contains(t, body, `<a href="https://www.ungefiltert-surfen.de/testing" hreflang="de" lang="de">Deutsch</a>`)

	body = readBody(t, ts.get(t, "http://localhost/testing"))
	assert.Contains(t, body, `<a href="/testing?lang=de" hreflang="de" lang="de">Deutsch</a>`)
}

func TestIndex(t *testing.T) {
	ts := newTestServer(t)

	res := ts.get(t, "http://www.ungefiltert-surfen.de/")
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", res.Header.Get("Content-Type"))
	body := readBody(t, res)

	assert.Contains(t, body, "<strong>2</strong> Nameserver aus <strong>2</strong> Ländern")
	assert.Contains(t, body, "<td>192.0.2.1<em>ns1.example.</em></td>")
	assert.Contains(t, body, "Deutschland<em>Berlin</em>")
	assert.Contains(t, body, "64496<em>Example AS</em>")
	assert.Contains(t, body, "gültig<em>DNSSEC</em>")
	assert.Contains(t, body, `href="https://client.rdap.org/?type=ip&amp;object=192.0.2.1"`)
	assert.NotContains(t, body, "192.0.2.3", "invalid nameservers are not listed")

	// countries sorted by their German names: Deutschland before Österreich
	assert.Regexp(t, `(?s)<a href="/nameserver/de.html">Deutschland</a>.*<a href="/nameserver/at.html">Österreich</a>`, body)
}

func TestNameserversIsIndex(t *testing.T) {
	ts := newTestServer(t)

	res := ts.get(t, "http://public-dns.info/nameservers")
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, readBody(t, res), "Recently checked")
}

func TestCountry(t *testing.T) {
	ts := newTestServer(t)

	for _, path := range []string{"/nameserver/de.html", "/nameserver/de", "/nameserver/DE.html"} {
		t.Run(path, func(t *testing.T) {
			res := ts.get(t, "http://public-dns.info"+path)
			require.Equal(t, http.StatusOK, res.StatusCode)
			body := readBody(t, res)
			assert.Contains(t, body, "<h1>DNS servers in Germany</h1>")
			assert.Contains(t, body, "Download all 1 valid nameservers:")
			assert.Contains(t, body, `<a href="/nameserver/de.csv">CSV</a>`)
			assert.Contains(t, body, "<td>Berlin</td>", "no country name on a country page")
			assert.NotContains(t, body, "192.0.2.3")
		})
	}

	for _, path := range []string{"/nameserver/xx.html", "/nameserver/deu", "/nameserver/de.php", "/nameserver/..%2Fnameservers.csv"} {
		t.Run(path, func(t *testing.T) {
			assert.Equal(t, http.StatusNotFound, ts.get(t, "http://public-dns.info"+path).StatusCode)
		})
	}
}

func TestDownloads(t *testing.T) {
	ts := newTestServer(t)

	res := ts.get(t, "/nameservers.csv")
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "ip_address\n192.0.2.1\n", readBody(t, res))
	assert.NotEmpty(t, res.Header.Get("Last-Modified"))

	res = ts.get(t, "/nameserver/de.json")
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, `[{"ip":"192.0.2.1"}]`, readBody(t, res))

	assert.Equal(t, http.StatusNotFound, ts.get(t, "/nameservers-all.txt").StatusCode, "not exported yet")
}

func TestStaticPages(t *testing.T) {
	ts := newTestServer(t)

	res := ts.get(t, "http://www.ungefiltert-surfen.de/testing")
	require.Equal(t, http.StatusOK, res.StatusCode)
	body := readBody(t, res)
	assert.Contains(t, body, "innerhalb von 3 Sekunden")
	assert.Contains(t, body, "<code>8.8.8.8</code>")
	assert.Contains(t, body, "<li><code>does-not-exist.invalid</code></li>")
	assert.Contains(t, body, "der letzten 30 Tage")

	for _, path := range []string{"/changelog", "/nameservers/new", "/static/pico.min.css", "/static/app.js", "/static/site.css", "/robots.txt"} {
		assert.Equal(t, http.StatusOK, ts.get(t, path).StatusCode, path)
	}
	assert.Equal(t, http.StatusNotFound, ts.get(t, "/unknown").StatusCode)
}

func TestRedirects(t *testing.T) {
	ts := newTestServer(t)

	res := ts.get(t, "/impressum")
	assert.Equal(t, http.StatusMovedPermanently, res.StatusCode)
	assert.Equal(t, "https://www.digineo.de/impressum/", res.Header.Get("Location"))
}

func TestSecurityHeaders(t *testing.T) {
	ts := newTestServer(t)

	res := ts.get(t, "/")
	assert.Contains(t, res.Header.Get("Content-Security-Policy"), "default-src 'self'")
	assert.Equal(t, "nosniff", res.Header.Get("X-Content-Type-Options"))
}

func TestSubmit(t *testing.T) {
	ts := newTestServer(t)

	res := ts.post(t, "198.51.100.1") // TEST-NET-2 is a bogon
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
	assert.Contains(t, readBody(t, res), "bogon")

	res = ts.post(t, "not an address")
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
	assert.Contains(t, readBody(t, res), "invalid IP address")

	res = ts.post(t, "")
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)

	res = ts.post(t, "192.0.2.1")
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode, "192.0.2.1 is a bogon as well")

	res = ts.post(t, "9.9.9.9")
	require.Equal(t, http.StatusOK, res.StatusCode)
	var job check.Job
	require.NoError(t, json.NewDecoder(res.Body).Decode(&job))
	assert.True(t, job.Status)

	var count int
	require.NoError(t, ts.db.QueryRow("SELECT COUNT(*) FROM nameservers WHERE ip_address = '9.9.9.9'").Scan(&count))
	assert.Equal(t, 1, count)

	res = ts.post(t, "9.9.9.9")
	assert.Equal(t, http.StatusConflict, res.StatusCode)
}

func TestSubmitFailedCheck(t *testing.T) {
	ts := newTestServer(t)
	ts.checker.err = "i/o timeout"

	res := ts.post(t, "9.9.9.9")
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
	assert.Equal(t, "i/o timeout\n", readBody(t, res))

	var count int
	require.NoError(t, ts.db.QueryRow("SELECT COUNT(*) FROM nameservers WHERE ip_address = '9.9.9.9'").Scan(&count))
	assert.Zero(t, count, "failed nameservers are not saved")
}

func TestSubmitBeforeCheckerIsReady(t *testing.T) {
	ts := newTestServer(t)
	ts.checker.ready = false

	res := ts.post(t, "9.9.9.9")
	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}
