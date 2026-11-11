DNS Check
=========

[![Build Status](https://github.com/corny/dnscheck/workflows/build/badge.svg?branch=main)](https://github.com/corny/dnscheck/actions)

This code powers the DNS check of the [Public DNS list](https://public-dns.info/) service.
It is written in [Go](https://go.dev/) and it scales very well.

## Dependencies

* [pgx PostgreSQL driver](https://github.com/jackc/pgx)
* [DNS library by Miek Gieben](https://github.com/miekg/dns)
* [GeoIP2 Reader for Go](https://github.com/oschwald/geoip2-golang)
* GeoLite2 City and ASN databases by MaxMind, available from https://www.maxmind.com/
* A PostgreSQL database

## Configuration

### Database configuration

The database is given as a PostgreSQL connection string via `--database` or the environment variable `DSN`.
The format is documented at the [pgx package documentation](https://pkg.go.dev/github.com/jackc/pgx/v5#hdr-Establishing_a_Connection).

### Database scheme

The schema is created automatically on start, see [internal/store/schema.sql](internal/store/schema.sql).

## Domain list

Create a list of domains to query the nameservers for.
You should include at least one domain that does not exist.
All of the domains should resolve to the same IP addresses (not location based / GeoIP).

    wikileaks.org
    www.dnssec-tools.org
    does-not-exist.invalid

See [domains.txt](domains.txt) for the list in use.

## Usage

Build the binary:

    go build ./cmd/dnscheck

The `check` command runs continuously: it serves the web pages of public-dns.info and ungefiltert-surfen.de, checks all nameservers, exports them every 4 hours and purges old data once a day.
The language of the pages depends on the host name; on other hosts, `?lang=de` selects German.
If `MAXMIND_ACCOUNT_ID` and `MAXMIND_LICENSE_KEY` are set, it downloads the GeoLite2 databases into `--geoip.dir` and updates them daily; otherwise it uses the files `GeoLite2-City.mmdb` and `GeoLite2-ASN.mmdb` found there.

`dnscheck -h` prints a list of all supported arguments. Example:

    dnscheck check --domains path/to/domains.txt --database "host=/var/run/postgresql dbname=publicdns" --geoip.dir path/to/geoip --export.output path/to/export

## Docker

Every push to `main` builds an image and pushes it to `ghcr.io/corny/dnscheck` with the tags `main` and `sha-<commit>`.
The image runs `dnscheck check` as a non-root user and keeps the export files and GeoLite2 databases in the volume `/data`.

    docker run -d -p 8000:8000 -v dnscheck:/data \
      -e DSN="host=db user=nameservers password=... dbname=nameservers" \
      -e MAXMIND_ACCOUNT_ID=... -e MAXMIND_LICENSE_KEY=... \
      ghcr.io/corny/dnscheck:main

TLS is terminated by a reverse proxy in front of the container, which must pass the original `Host` header so the pages are served in the right language.
