DNS Check
=========

[![Build Status](https://github.com/corny/dnscheck/workflows/build/badge.svg?branch=master)](https://github.com/corny/dnscheck/actions)

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

    non-existent.example.com
    wikileaks.org
    rotten.com

## Usage

Build the binary:

    go build ./cmd/dnscheck

`dnscheck -h` prints a list of all supported arguments. Example:

    dnscheck check --domains path/to/domains.txt --database "host=/var/run/postgresql dbname=publicdns" --geodb-city path/to/GeoLite2-City.mmdb --geodb-asn path/to/GeoLite2-ASN.mmdb
