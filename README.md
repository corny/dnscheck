DNS Check
=========

[![CircleCI](https://circleci.com/gh/corny/dnscheck.svg?style=shield)](https://circleci.com/gh/corny/dnscheck)

This code powers the DNS check of the [Public DNS list](http://public-dns.info) service.
It is written in [Go](http://golang.org/) and it scales very well.

## Dependencies

* [Go-MySQL-Driver](https://github.com/go-sql-driver/mysql)
* [Go-YAML v2](https://gopkg.in/yaml.v2)
* [DNS library by Miek Gieben](https://github.com/miekg/dns)
* [MaxMind DB Reader for Go](https://github.com/oschwald/maxminddb-golang)
* GeoLite2 by MaxMind, available from http://www.maxmind.com
* A MariaDB/MySQL database

## Configuration

### Database configuration

The program is intended to be part of a Rails application.
So you need a `database.yml` with the credentials for your database.

#### Using a socket

    development:
      socket: /var/run/mysqld/mysqld.sock
      database: nameservers_development
      username: root
      password:

#### Using a tcp connection

    production:
      host: 127.0.0.1
      database: nameservers
      username: nameservers
      password: topsecret

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

Replace `env` with your environment name (e.g. development or production) and pass the path to your database.yml.
`dnscheck -h` prints a list of all supported arguments.

    RAILS_ENV=env dnscheck -domains path/to/domains -database path/to/database.yml -geodb path/to/GeoLite2-City.mmdb
