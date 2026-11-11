package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed locales/*.json
var localeFiles embed.FS

// languages maps the language codes to their canonical host names (D17).
var languages = map[string]string{
	"de": "www.ungefiltert-surfen.de",
	"en": "public-dns.info",
}

const defaultLanguage = "en"

// locale holds the translations of one language.
type locale struct {
	lang      string
	messages  map[string]string
	countries map[string]string
	printer   *message.Printer
	collator  *collate.Collator
}

func loadLocales() (map[string]*locale, error) {
	locales := make(map[string]*locale)

	for lang := range languages {
		data, err := localeFiles.ReadFile("locales/" + lang + ".json")
		if err != nil {
			return nil, err
		}

		l := &locale{lang: lang}
		var content struct {
			Messages  map[string]string `json:"messages"`
			Countries map[string]string `json:"countries"`
		}
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, fmt.Errorf("locale %s: %w", lang, err)
		}

		tag := language.MustParse(lang)
		l.messages = content.Messages
		l.countries = content.Countries
		l.printer = message.NewPrinter(tag)
		l.collator = collate.New(tag)
		locales[lang] = l
	}

	return locales, nil
}

// t returns the translated message for key, formatted with args.
// Missing keys are returned as they are, a test ensures there are none.
func (l *locale) t(key string, args ...any) string {
	msg, ok := l.messages[key]
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// country returns the name of the country code, or the code if unknown.
func (l *locale) country(code string) string {
	if name, ok := l.countries[strings.ToUpper(code)]; ok {
		return name
	}
	return code
}

// number formats n with the thousands separator of the language.
func (l *locale) number(n int) string {
	return l.printer.Sprintf("%d", n)
}

// sortByName sorts links by their names in the order of the language.
func (l *locale) sortByName(links []link) {
	slices.SortFunc(links, func(a, b link) int {
		return l.collator.CompareString(a.Name, b.Name)
	})
}

// languageFor returns the language of a request: the language of a known
// host name, or for other hosts (local development) the lang parameter.
func languageFor(r *http.Request) string {
	host := hostname(r)
	for lang, h := range languages {
		if h == host {
			return lang
		}
	}

	if lang := r.URL.Query().Get("lang"); languages[lang] != "" {
		return lang
	}
	return defaultLanguage
}

// isKnownHost tells whether the request uses one of the canonical hosts.
func isKnownHost(r *http.Request) bool {
	host := hostname(r)
	for _, h := range languages {
		if h == host {
			return true
		}
	}
	return false
}

func hostname(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		return r.Host
	}
	return host
}

// link is a hyperlink or an alternative version of a page.
type link struct {
	Lang string
	URL  string
	Name string
}

// alternates returns the URLs of the page in all languages, for hreflang.
func alternates(path string) []link {
	var links []link
	for lang, host := range languages {
		links = append(links, link{Lang: lang, URL: "https://" + host + path})
	}
	slices.SortFunc(links, func(a, b link) int { return strings.Compare(a.Lang, b.Lang) })
	return links
}

// languageSwitch returns the link to the page in the other language. On
// unknown hosts it uses the lang parameter instead of the other host.
func languageSwitch(r *http.Request, current *locale) link {
	for lang, host := range languages {
		if lang == current.lang {
			continue
		}
		name := current.t("language." + lang)
		if isKnownHost(r) {
			return link{Lang: lang, URL: "https://" + host + r.URL.Path, Name: name}
		}
		return link{Lang: lang, URL: r.URL.Path + "?lang=" + lang, Name: name}
	}
	return link{}
}
