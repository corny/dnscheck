DNS Check
=========

[![Build Status](https://github.com/corny/dnscheck/workflows/build/badge.svg?branch=master)](https://github.com/corny/dnscheck/actions)
[![Codecov](https://codecov.io/gh/corny/dnscheck/branch/master/graph/badge.svg)](https://codecov.io/gh/corny/dnscheck)

This code powers the DNS check of the [Public DNS list](http://public-dns.info) service.
It is written in [Go](http://golang.org/) and it scales very well.

## Dependencies

* [Go postgres driver](https://github.com/lib/pq)
* [DNS library by Miek Gieben](https://github.com/miekg/dns)
* [MaxMind DB Reader for Go](https://github.com/oschwald/maxminddb-golang)
* GeoLite2 by MaxMind, available from http://www.maxmind.com
* A PostgreSQL database

## Configuration

### Database configuration

The schema of the database connection string is documented at the [pq package documentation](https://pkg.go.dev/github.com/lib/pq).

### Database scheme

After creating the database apply the [structure.sql](structure.sql):

    mysql $database < structure.sql

## Domain list

Create a list of domains to query the nameservers for.
You should include at least one domain that does not exist.
All of the domains should resolve to the same IP addresses (not location based / GeoIP).

    non-existent.example.com
    wikileaks.org
    rotten.com

## Usage

`dnscheck -h` prints a list of all supported arguments. Example:

    dnscheck check --domains path/to/domains.txt --database "host=/var/run/postgresql dbname=publicdns" --geodb path/to/GeoLite2-City.mmdb
