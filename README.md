DNS Check
=========

[![Build Status](https://travis-ci.org/corny/dnscheck.svg?branch=master)](https://travis-ci.org/corny/dnscheck)

This code powers the DNS check of the [Public DNS list](http://public-dns.tk) service.
It is written in [Go](http://golang.org/) and it scales very well.

## Dependencies

* [Go-MySQL-Driver](https://github.com/go-sql-driver/mysql)
* [Go-YAML v2](https://gopkg.in/yaml.v2)
* [DNS library by Miek Gieben](https://github.com/miekg/dns)
* [MaxMind DB Reader for Go](https://github.com/oschwald/maxminddb-golang)
* GeoLite2 by MaxMind, available from http://www.maxmind.com
* A MySQL database

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

Create a database with the following table:

    CREATE TABLE `nameservers` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `ip` varchar(255) NOT NULL,
      `name` varchar(255) DEFAULT NULL,
      `state` varchar(255) NOT NULL DEFAULT 'new',
      `state_changed_at` datetime DEFAULT NULL,
      `error` varchar(255) DEFAULT NULL,
      `country_id` char(2) DEFAULT NULL,
      `city` varchar(255) DEFAULT NULL,
      `checked_at` datetime DEFAULT NULL,
      `created_at` datetime NOT NULL,
      `updated_at` datetime NOT NULL,
      `version` varchar(255) DEFAULT NULL,
      `dnssec` boolean DEFAULT NULL,
      PRIMARY KEY (`id`),
      UNIQUE KEY `index_nameservers_on_ip` (`ip`),
      KEY `index_nameservers_on_state` (`state`),
      KEY `country_state_checked` (`country_id`,`state`,`checked_at`),
      KEY `index_nameservers_on_version` (`version`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8

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
