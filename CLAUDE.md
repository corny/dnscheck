# CLAUDE.md

This repository contains the DNS checker behind https://public-dns.info/ (English) and https://www.ungefiltert-surfen.de/ (German). It periodically checks a list of public DNS servers for reachability and correct answers and exports the results as CSV, text and JSON.

## Goal of the ongoing migration

The public pages are currently served by a separate Rails application (Rails 6.0, Ruby 2.7, Webpacker, Vue 2, Bootstrap 4; source in `../rails`, remote `git.digineo.de:julian/nameservers.git`). Its features move into this Go application, after which Rails is retired.

Fundamental decisions and open questions are recorded in [DECISIONS.md](DECISIONS.md). Check it before changing architecture, database, URLs or dependencies, and add new decisions there.

## Branches

All work happens on `main` (DECISIONS.md D2). Commit messages follow Conventional Commits.

- `main`: builds on `postgres` and contains all further work.
- `postgres`: the state currently running in production (PostgreSQL, endpoint `POST /api/nameservers`, called by the submission form of the Rails site).
- `master`: outdated MySQL state, not used and left untouched.

Known issue on `postgres`: `purge` deletes valid instead of invalid servers. Fixed on `main`; `purge` has not been run in production for years and is not scheduled yet (DECISIONS.md D15).

## Structure

- `cmd/dnscheck/`: the binary. `main.go` parses flags (kingpin) and wires everything together; subcommands `check`, `export`, `purge`. `http.go` serves `POST /api/nameservers` (CORS for the Rails origin) and metrics, `prometheus.go` exposes the metrics.
- `internal/check/`: DNS checks. Each server is compared with a reference nameserver (default `8.8.8.8`) for the domains in `domains.txt`, plus NXDOMAIN, DNSSEC (AD bit), PTR and `version.bind`.
- `internal/checkrun/`: check loop: reads all nameservers from the store in an interval, feeds them to the checker and writes the results back.
- `internal/store/`: all SQL lives here (schema, nameserver lookups, check results, export iteration, purge). No SQL outside this package. `schema.sql` is applied on every start (D6), including the trigger `insert_nameserver_check`, which fills `nameserver_checks` and computes `reliability`.
- `internal/export/`: writes `nameservers`, `nameservers-all` and one `<cc>` per country as `.txt`, `.csv`, `.json`, atomically (temporary file plus rename).
- `internal/geoip/`: GeoLite2 wrapper that reloads on file changes.
- `internal/bogon/`: rejects private and reserved address ranges.

## Commands

```sh
go build ./cmd/dnscheck
go vet ./...
go test ./...
```

Some tests in `check/` query public DNS servers and need network access. `go test -short ./...` skips them; CI runs the full suite.

Database tests in `internal/store` run against the PostgreSQL database given by `TEST_DSN` and are skipped if it is unset. They drop and recreate all tables, so never point `TEST_DSN` at a database with real data. Locally:

```sh
createdb dnscheck_test
TEST_DSN="host=/tmp dbname=dnscheck_test" go test -race ./...
```

CI starts a PostgreSQL service container for these tests. Always say so when a test is red or was skipped, never count it as a success.

Local run:

```sh
DSN="host=/var/run/postgresql dbname=publicdns" go run ./cmd/dnscheck check --domains domains.txt --geodb-city GeoLite2-City.mmdb --geodb-asn GeoLite2-ASN.mmdb
```

The GeoLite2 databases (City, ASN) come from MaxMind and are not part of the repository.

## Rules for this project

- Delivery is a Docker image with one long-running process: checker, web server, export and purge on internal intervals; TLS by a reverse proxy; GeoLite2 downloaded by the process itself with account ID and license key from secrets (D3, D16).
- Download URLs and `POST /api/nameservers` stay unchanged (D9). Repository and module path stay `github.com/corny/dnscheck`, license AGPL-3.0 (D4).
- The URLs of the HTML pages stay unchanged (D9): `/`, `/nameservers`, `/nameservers/new`, `/nameserver/<cc>`, `/testing`, `/changelog`, `/impressum`, `/datenschutz`, `/data-privacy`.
- Frontend: `html/template`, Pico CSS, plain JavaScript, no Node.js build (D5). Templates and static files are embedded via `embed`. Translations as embedded JSON plus per-language templates (D7).
- Schema changes only in `internal/store/schema.sql`, and every statement must be idempotent (D6).
- The language is derived from the host name: `www.ungefiltert-surfen.de` is German, `public-dns.info` is English. Both sites reference each other via `hreflang`.
- Form input is untrusted: parse IP addresses, reject bogons.
- No request logging, no storage of visitor or submitter IP addresses (D12, GDPR).
- No secrets (DSN with password, MaxMind license key) in the repository.
- New dependencies only with a rationale in DECISIONS.md.
