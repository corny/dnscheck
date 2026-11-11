// Package store contains all access to the PostgreSQL database.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net"

	// registers the "pgx" driver for database/sql
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/corny/dnscheck/internal/check"
)

// Store wraps the database connection.
type Store struct {
	db *sql.DB
}

// Open opens the database given by a PostgreSQL data source name.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// EachAddress calls fn for the ID and address of every nameserver, reading
// them in batches of batchSize. The iteration stops when fn returns false.
func (s *Store) EachAddress(batchSize int, fn func(id int, address net.IP) bool) error {
	lastID := 0

	for {
		batch, err := s.addressBatch(lastID, batchSize)
		if err != nil {
			return err
		}

		for _, a := range batch {
			if !fn(a.id, a.ip) {
				return nil
			}
		}

		if len(batch) < batchSize {
			return nil
		}
		lastID = batch[len(batch)-1].id
	}
}

type idAddress struct {
	id int
	ip net.IP
}

// addressBatch reads up to batchSize nameservers with an ID greater than afterID.
func (s *Store) addressBatch(afterID, batchSize int) ([]idAddress, error) {
	rows, err := s.db.Query("SELECT id, ip_address FROM nameservers WHERE id > $1 LIMIT $2", afterID, batchSize)
	if err != nil {
		return nil, fmt.Errorf("select batch failed: %w", err)
	}
	defer rows.Close()

	var batch []idAddress
	for rows.Next() {
		var (
			id      int
			address string
		)
		if err := rows.Scan(&id, &address); err != nil {
			return nil, fmt.Errorf("scanning DB values failed: %w", err)
		}
		batch = append(batch, idAddress{id: id, ip: net.ParseIP(address)})
	}

	return batch, rows.Err()
}

// FindID returns the ID of the nameserver with the given address.
func (s *Store) FindID(address net.IP) (id int, found bool, err error) {
	err = s.db.QueryRow("SELECT id FROM nameservers WHERE ip_address=$1", address.String()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// SaveResult updates the nameserver of a check job, or inserts it if the
// job has no ID yet.
func (s *Store) SaveResult(job *check.Job) error {
	args := []any{job.Name, job.Status, job.Error, job.Version, job.Dnssec, nil, nil, nil, nil, nil}

	if job.City != nil {
		args[5] = job.City.Country.ISOCode
		args[6] = job.City.City.Names.English
	}

	if job.ASN != nil {
		args[7] = job.ASN.AutonomousSystemNumber
		args[8] = job.ASN.AutonomousSystemOrganization
	}

	var err error
	if job.ID > 0 {
		args[len(args)-1] = job.ID
		_, err = s.db.Exec("UPDATE nameservers SET name=$1, status=$2, error=$3, version=$4, dnssec=$5, country_code=$6, city=$7, as_number=$8, as_org=$9, checked_at = NOW() at time zone 'utc' WHERE id=$10", args...)
	} else {
		args[len(args)-1] = job.Address.String()
		_, err = s.db.Exec("INSERT INTO nameservers (name, status, error, version, dnssec, country_code, city, as_number, as_org, ip_address, reliability, checked_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1, NOW() at time zone 'utc', NOW() at time zone 'utc')", args...)
	}

	return err
}
