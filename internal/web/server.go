// Package web serves the public pages, the export files and the API for
// submitting nameservers.
package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/corny/dnscheck/internal/bogon"
	"github.com/corny/dnscheck/internal/check"
	"github.com/corny/dnscheck/internal/store"
)

//go:embed templates/*.html
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

const (
	recentLimit  = 15  // nameservers on the start page
	countryLimit = 100 // nameservers on a country page
)

// Checker checks submitted nameservers, implemented by *check.Checker.
type Checker interface {
	Ready() bool
	ExecuteJob(job *check.Job)
}

// Config configures the Server.
type Config struct {
	Store     *store.Store
	Checker   Checker
	ExportDir string // directory of the export files

	// shown on the testing page
	Domains   []string
	Reference string
	Timeout   time.Duration
	PurgeDays uint
}

// Server serves the web pages.
type Server struct {
	Config
	locales map[string]*locale
	pages   map[string]map[string]*template.Template // language, page
}

var pageNames = []string{"index", "country", "new", "testing", "changelog"}

// New parses the templates and translations.
func New(config Config) (*Server, error) {
	locales, err := loadLocales()
	if err != nil {
		return nil, err
	}

	s := &Server{
		Config:  config,
		locales: locales,
		pages:   make(map[string]map[string]*template.Template),
	}

	for lang, l := range locales {
		s.pages[lang] = make(map[string]*template.Template)
		for _, name := range pageNames {
			tpl, err := template.New("").Funcs(templateFuncs(l)).ParseFS(templateFiles,
				"templates/layout.html",
				"templates/partials.html",
				"templates/"+name+".html",
			)
			if err != nil {
				return nil, err
			}
			s.pages[lang][name] = tpl
		}
	}

	return s, nil
}

func templateFuncs(l *locale) template.FuncMap {
	return template.FuncMap{
		"lang":    func() string { return l.lang },
		"t":       l.t,
		"country": l.country,
		"number":  l.number,
		"percent": func(f float64) string { return l.number(int(f*100+0.5)) + " %" },
		"iso":     func(t time.Time) string { return t.UTC().Format(time.RFC3339) },
		"time":    func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 UTC") },
		"yes":     func(b *bool) bool { return b != nil && *b },
		"rdap":    func(ip string) string { return "https://client.rdap.org/?type=ip&object=" + ip },
	}
}

// Handler returns the handler for all routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /nameservers", s.index)
	mux.HandleFunc("GET /nameservers/new", s.newNameservers)
	mux.HandleFunc("GET /nameserver/{file}", s.country)
	mux.HandleFunc("GET /testing", s.testing)
	mux.HandleFunc("GET /changelog", s.changelog)
	mux.HandleFunc("POST /api/nameservers", s.submit)

	for _, list := range []string{"nameservers", "nameservers-all"} {
		for _, ext := range []string{"csv", "txt", "json"} {
			mux.HandleFunc("GET /"+list+"."+ext, s.serveExport(list+"."+ext))
		}
	}

	for path, target := range map[string]string{
		"/impressum":    "https://www.digineo.de/impressum/",
		"/datenschutz":  "https://www.digineo.de/datenschutz/",
		"/data-privacy": "https://www.digineo.de/data-privacy/",
	} {
		mux.Handle("GET "+path, http.RedirectHandler(target, http.StatusMovedPermanently))
	}

	static, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, static, "robots.txt")
	})

	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// page holds the data for all templates.
type page struct {
	Title      string
	Alternates []link
	Switch     link

	Count       int
	Countries   []link
	Nameservers []*store.Nameserver
	Country     string // country code on a country page

	Domains   []string
	Reference string
	Timeout   time.Duration
	PurgeDays uint
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, p *page) {
	l := s.locales[languageFor(r)]
	p.Alternates = alternates(r.URL.Path)
	p.Switch = languageSwitch(r, l)

	var buf bytes.Buffer
	if err := s.pages[l.lang][name].ExecuteTemplate(&buf, "layout", p); err != nil {
		s.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	l := s.locales[languageFor(r)]

	count, err := s.Store.CountValid()
	if err != nil {
		s.serverError(w, err)
		return
	}
	codes, err := s.Store.ValidCountries()
	if err != nil {
		s.serverError(w, err)
		return
	}
	recent, err := s.Store.RecentValid(recentLimit)
	if err != nil {
		s.serverError(w, err)
		return
	}

	countries := make([]link, len(codes))
	for i, code := range codes {
		countries[i] = link{URL: "/nameserver/" + strings.ToLower(code) + ".html", Name: l.country(code)}
	}
	l.sortByName(countries)

	s.render(w, r, "index", &page{
		Title:       l.t("index.title"),
		Count:       count,
		Countries:   countries,
		Nameservers: recent,
	})
}

var (
	countryPage = regexp.MustCompile(`^([a-z]{2})(\.html)?$`)
	countryFile = regexp.MustCompile(`^[a-z]{2}\.(csv|txt|json)$`)
)

// country serves /nameserver/<cc>, /nameserver/<cc>.html and the export
// files /nameserver/<cc>.{csv,txt,json}.
func (s *Server) country(w http.ResponseWriter, r *http.Request) {
	l := s.locales[languageFor(r)]
	file := strings.ToLower(r.PathValue("file"))

	if countryFile.MatchString(file) {
		s.serveExport(file)(w, r)
		return
	}

	m := countryPage.FindStringSubmatch(file)
	if m == nil || l.country(m[1]) == m[1] {
		http.NotFound(w, r)
		return
	}
	code := strings.ToUpper(m[1])

	list, total, err := s.Store.CountryValid(code, countryLimit)
	if err != nil {
		s.serverError(w, err)
		return
	}

	s.render(w, r, "country", &page{
		Title:       l.t("country.title", l.country(code)),
		Count:       total,
		Nameservers: list,
		Country:     strings.ToLower(code),
	})
}

// serveExport serves an export file. name must be validated by the caller.
func (s *Server) serveExport(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(s.ExportDir, name))
	}
}

func (s *Server) newNameservers(w http.ResponseWriter, r *http.Request) {
	l := s.locales[languageFor(r)]
	s.render(w, r, "new", &page{Title: l.t("new.title")})
}

func (s *Server) testing(w http.ResponseWriter, r *http.Request) {
	l := s.locales[languageFor(r)]
	s.render(w, r, "testing", &page{
		Title:     l.t("testing.title"),
		Domains:   s.Domains,
		Reference: s.Reference,
		Timeout:   s.Timeout,
		PurgeDays: s.PurgeDays,
	})
}

func (s *Server) changelog(w http.ResponseWriter, r *http.Request) {
	l := s.locales[languageFor(r)]
	s.render(w, r, "changelog", &page{Title: l.t("changelog.title")})
}

var errAlreadyExists = errors.New("already exists")

// submit checks a submitted nameserver and saves it if it is valid.
func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	addrStr := r.FormValue("address")
	if addrStr == "" {
		http.Error(w, "address parameter is missing or empty", http.StatusUnprocessableEntity)
		return
	}

	address := net.ParseIP(strings.TrimSpace(addrStr))
	if address == nil {
		http.Error(w, "invalid IP address", http.StatusUnprocessableEntity)
		return
	}

	if bogon.IsBogon(address) {
		http.Error(w, "bogon IP address", http.StatusUnprocessableEntity)
		return
	}

	if !s.Checker.Ready() {
		http.Error(w, "checker is starting, please try again later", http.StatusServiceUnavailable)
		return
	}

	job, err := s.addNameserver(address)
	var failed checkError
	switch {
	case errors.Is(err, errAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.As(err, &failed):
		http.Error(w, failed.Error(), http.StatusUnprocessableEntity)
	case err != nil:
		s.serverError(w, err)
	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(&job)
	}
}

// checkError is the result of a failed check.
type checkError string

func (e checkError) Error() string { return string(e) }

func (s *Server) addNameserver(address net.IP) (job check.Job, err error) {
	job.Address = address

	_, found, err := s.Store.FindID(address)
	if err != nil {
		return job, err
	}
	if found {
		return job, errAlreadyExists
	}

	s.Checker.ExecuteJob(&job)
	if job.Error != "" {
		return job, checkError(job.Error)
	}

	return job, s.Store.SaveResult(&job)
}
