# CLAUDE.md

This repository contains the DNS checker behind https://public-dns.info/ (English) and https://www.ungefiltert-surfen.de/ (German). It periodically checks a list of public DNS servers for reachability and correct answers and exports the results as CSV, text and JSON.

## Goal of the ongoing migration

The public pages are currently served by a separate Rails application (Rails 6.0, Ruby 2.7, Webpacker, Vue 2, Bootstrap 4; source in `../rails`, remote `git.digineo.de:julian/nameservers.git`). Its features move into this Go application, after which Rails is retired.

Fundamental decisions and open questions are recorded in [DECISIONS.md](DECISIONS.md). Check it before changing architecture, database, URLs or dependencies, and add new decisions there.

## Current state (starting point)

All work happens on the `postgres` branch; `master` is not used and stays untouched (DECISIONS.md D2). Commit messages follow Conventional Commits.

- `postgres` (July 2020): runs in production and is the base for all further work. PostgreSQL schema matching Rails (`ip_address`, `status` as boolean, `country_code`, `as_number`, `as_org`), DSN via flag or environment variable `DSN`, endpoint `POST /api/nameservers` with bogon check, GeoLite2 ASN lookup, check loop with interval.
- `master`: outdated MySQL state (`ip`, `state` as string, `country_id`), configured via a Rails `database.yml` and `RAILS_ENV`. Only dependency updates since 2020; not continued.

The submission form of the Rails site calls `/api/nameservers` on the Go application.

Known issue on `postgres`: `purge` deletes servers with `status = TRUE` instead of invalid ones. `purge` has not been run in production for years and must not be run before the fix (DECISIONS.md D2, D15).

## Structure

- `cmd/dnscheck/`: the binary. `main.go` parses flags (kingpin) and wires everything together; subcommands `check`, `export`, `purge`. `http.go` serves `POST /api/nameservers` (CORS for the Rails origin) and metrics, `prometheus.go` exposes the metrics.
- `internal/check/`: DNS checks. Each server is compared with a reference nameserver (default `8.8.8.8`) for the domains in `domains.txt`, plus NXDOMAIN, DNSSEC (AD bit), PTR and `version.bind`.
- `internal/checkrun/`: check loop: reads all nameservers from the store in an interval, feeds them to the checker and writes the results back.
- `internal/store/`: all SQL lives here (nameserver lookups, check results, export iteration, purge). No SQL outside this package.
- `internal/export/`: writes `nameservers`, `nameservers-all` and one `<cc>` per country as `.txt`, `.csv`, `.json`, atomically (temporary file plus rename).
- `internal/geoip/`: GeoLite2 wrapper that reloads on file changes.
- `internal/bogon/`: rejects private and reserved address ranges.
- `structure.sql`: still the old MySQL schema, not valid for production. The PostgreSQL schema currently lives in the Rails migrations (`../rails/db/migrate`), including the trigger `insert_nameserver_check`, which fills `nameserver_checks` and computes `reliability`.

## Commands

```sh
go build ./cmd/dnscheck
go vet ./...
go test ./...
```

Some tests in `check/` query public DNS servers and need network access. `go test -short ./...` skips them; CI runs the full suite. Always say so when a test is red or was skipped, never count it as a success.

Local run (`postgres` branch):

```sh
DSN="host=/var/run/postgresql dbname=publicdns" go run ./cmd/dnscheck check --domains domains.txt --geodb-city GeoLite2-City.mmdb --geodb-asn GeoLite2-ASN.mmdb
```

The GeoLite2 databases (City, ASN) come from MaxMind and are not part of the repository.

## Rules for this project

- Delivery is a Docker image with one long-running process: checker, web server, export and purge on internal intervals; TLS by a reverse proxy; GeoLite2 downloaded by the process itself with account ID and license key from secrets (D3, D16).
- Download URLs and `POST /api/nameservers` stay unchanged (D9). Repository and module path stay `github.com/corny/dnscheck`, license AGPL-3.0 (D4).
- The URLs of the HTML pages stay unchanged (D9): `/`, `/nameservers`, `/nameservers/new`, `/nameserver/<cc>`, `/testing`, `/changelog`, `/impressum`, `/datenschutz`, `/data-privacy`.
- Frontend: `html/template`, Pico CSS, plain JavaScript, no Node.js build (D5). Templates and static files are embedded via `embed`. Translations as embedded JSON plus per-language templates (D7).
- Schema changes only as embedded SQL migration files (D6).
- The language is derived from the host name: `www.ungefiltert-surfen.de` is German, `public-dns.info` is English. Both sites reference each other via `hreflang`.
- Form input is untrusted: parse IP addresses, reject bogons.
- No request logging, no storage of visitor or submitter IP addresses (D12, GDPR).
- No secrets (DSN with password, MaxMind license key) in the repository.
- New dependencies only with a rationale in DECISIONS.md.
