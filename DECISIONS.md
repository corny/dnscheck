# Decisions

Current architecture and project decisions for merging dnscheck and the Rails application. Each entry is either `decided` or `open`. This file only reflects the current state: when a decision changes, edit the entry in place; when it no longer applies, remove it. History lives in git.

## D1: Retire the Rails application, Go serves the public pages

- Status: decided
- Decision: All features of the Rails application (`../rails`) move into this repository. The Rails application is shut down afterwards.

## D2: Work on the postgres branch

- Status: decided
- Decision: The `postgres` branch runs in production and is the base for all work. `master` (MySQL) is not used and stays untouched.
- Open: whether and how `postgres` becomes the default branch.

## D3: Delivery as Docker container

- Status: decided
- Decision:
  - The application is shipped as a Docker image, built by GitHub Actions and pushed to `ghcr.io/corny/dnscheck`.
  - One long-running process: checker, web server, `export` (every 4 hours) and `purge` run inside it on internal intervals. No cron in or around the container.
  - TLS is terminated by a reverse proxy in front of the container. The container speaks plain HTTP.
  - The container downloads and updates the GeoLite2 databases itself (D16).
  - Base image `gcr.io/distroless/static`, running as non-root user.
  - One volume at `/data` for export files and GeoLite2 databases.

## D4: Repository and license

- Status: decided
- Decision: The repository stays at `github.com/corny/dnscheck`, the module path stays `github.com/corny/dnscheck`. CI runs on GitHub Actions. License is AGPL-3.0.

## D5: Frontend with html/template, Pico CSS and plain JavaScript

- Status: decided
- Decision:
  - Server-side rendering with Go `html/template`, templates and static files embedded via `embed`.
  - Pico CSS as a static file in the repository.
  - No Node.js build. The submission form and relative timestamps use plain JavaScript.

## D6: Schema management with an idempotent schema file

- Status: decided
- Decision:
  - The PostgreSQL schema from the Rails migrations (tables, indexes, trigger `insert_nameserver_check`) lives in `internal/store/schema.sql`, embedded in the binary and executed on every start.
  - Every statement must be idempotent (`IF NOT EXISTS`, `CREATE OR REPLACE`, `ADD COLUMN IF NOT EXISTS`). Existing databases created by Rails are left untouched.
  - No version table, no migration library. Production runs PostgreSQL 16 or newer, CI tests against PostgreSQL 16.
  - If a schema change cannot be expressed idempotently, versioned migrations are reconsidered.

## D7: Translations as embedded JSON and per-language templates

- Status: decided
- Decision: The Rails locale files (de, en, including country names) are converted to JSON and embedded; lookup uses `encoding/json`, no YAML dependency. Language-specific text blocks are separate templates per language.

## D17: Language selection by host name

- Status: decided
- Decision:
  - The language is derived only from the `Host` header: `www.ungefiltert-surfen.de` is German, `public-dns.info` is English, every other host is English. `Accept-Language` is ignored, so a URL always delivers the same content.
  - Host name variants (with or without `www`) are redirected by the web server in front of the application; the application does not handle them.
  - The reverse proxy must pass the original `Host` header to the container.
  - For local development, `?lang=de` or `?lang=en` selects the language, but only on hosts that are not in the mapping above (for example `localhost`). No session, no cookie.

## D8: No statistics graph

- Status: decided
- Decision: The statistics section (`stats.png`, `stats.rrd`) on `/testing` is dropped. Metrics are available via Prometheus at `/metrics`.

## D9: Public URLs

- Status: decided
- Decision:
  - HTML pages: `/`, `/nameservers`, `/nameservers/new`, `/nameserver/<cc>`, `/testing`, `/changelog`, and the redirects `/impressum`, `/datenschutz`, `/data-privacy`.
  - Downloads: `/nameservers.{csv,txt,json}`, `/nameservers-all.{csv,txt,json}`, `/nameserver/<cc>.{csv,txt,json}`, served by the Go application.
  - API: `POST /api/nameservers` for the submission form.

## D10: Revised page contents

- Status: decided
- Decision: Texts are revised, not copied 1:1. Outdated references (Ruby on Rails, Bootstrap, dead links, http links) are updated or removed, and the changelog gets an entry for the rewrite.

## D11: No rate limit for submissions

- Status: decided
- Decision: `POST /api/nameservers` has no rate limit. Revisit if abuse occurs.

## D12: No request logging

- Status: decided
- Decision: The application does not log HTTP requests and does not store IP addresses of visitors or submitters. Errors and operational events of the checker are logged.

## D13: Current Go version and dependencies

- Status: decided
- Decision: Use the current Go version and update or replace all dependencies (D14).

## D14: Replacements for outdated dependencies

- Status: decided
- Decision:
  - `gopkg.in/alecthomas/kingpin.v2` -> `github.com/alecthomas/kingpin/v2`.
  - `github.com/pkg/errors` -> standard library (`fmt.Errorf` with `%w`, `errors`).
  - `github.com/lib/pq` -> `github.com/jackc/pgx/v5` via `database/sql`.
  - `github.com/rs/cors` -> removed, form and API share the same origin.
  - `github.com/go-sql-driver/mysql` -> removed (`sql.NullTime` instead of `mysql.NullTime`).
  - `github.com/erikdubbelboer/gspt` -> removed.

## D15: Purge

- Status: decided
- Decision: `purge` runs as an internal interval in the long-running process (D3). It stays disabled until its bug is fixed: it currently deletes servers with `status = TRUE` instead of invalid ones.
- Open: interval.

## D16: GeoLite2 download inside the container

- Status: decided
- Decision:
  - The container downloads the GeoLite2 City and ASN databases from MaxMind and keeps them up to date.
  - MaxMind account ID and license key are passed as secrets or environment variables, never baked into the image or the repository.
  - Download and extraction in Go (HTTP plus `archive/tar` and `compress/gzip`, no new dependency). Check daily, replace the file atomically; the existing reload in `geoip/` picks up the new file.
  - On start without a database, checks wait until the first download succeeds.
