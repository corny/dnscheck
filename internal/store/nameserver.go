package store

import (
	"database/sql"
	"strings"
	"time"
)

// Nameserver describes a nameserver record.
type Nameserver struct {
	ID          int        `json:"-"`
	Status      bool       `json:"-"`
	Address     string     `json:"ip"`
	Name        *string    `json:"name"`
	ASNumber    *uint      `json:"as_number"`
	ASOrg       *string    `json:"as_org"`
	Country     *string    `json:"country_id"`
	City        *string    `json:"city"`
	Version     *string    `json:"version"`
	Error       *string    `json:"error"`
	DNSSEC      *bool      `json:"dnssec"`
	Reliability float64    `json:"reliability"`
	CheckedAt   *time.Time `json:"checked_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// IsValid tells whether the nameserver passed its last check.
func (ns *Nameserver) IsValid() bool {
	return ns.Status
}

const queryNameserverBatch = `SELECT "id", "status", "ip_address", "name", "as_number", "as_org", "country_code", "city", "version", "error", "dnssec", "reliability", "checked_at", "created_at" FROM nameservers WHERE "id" > $1 ORDER BY "id" ASC LIMIT $2`

// EachNameserver iterates over all nameservers, reading them in batches of
// batchSize, and calls fn for each. If fn returns an error, the iteration
// stops and that error is returned.
func (s *Store) EachNameserver(batchSize uint, fn func(*Nameserver) error) error {
	stmt, err := s.db.Prepare(queryNameserverBatch)
	if err != nil {
		return err
	}
	defer stmt.Close()

	lastID := 0
	for {
		found, err := nameserverBatch(stmt, &lastID, batchSize, fn)
		if err != nil {
			return err
		}
		if found < batchSize {
			return nil
		}
	}
}

// nameserverBatch reads one batch after *lastID, calls fn for each record
// and advances *lastID.
func nameserverBatch(stmt *sql.Stmt, lastID *int, batchSize uint, fn func(*Nameserver) error) (uint, error) {
	rows, err := stmt.Query(*lastID, batchSize)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var found uint
	for rows.Next() {
		ns, err := scanNameserver(rows)
		if err != nil {
			return found, err
		}
		*lastID = ns.ID
		if err := fn(ns); err != nil {
			return found, err
		}
		found++
	}

	return found, rows.Err()
}

func scanNameserver(row *sql.Rows) (*Nameserver, error) {
	var (
		id                int
		status            bool
		address           string
		asNumber          sql.NullInt32
		asOrg             sql.NullString
		name, ver, errStr sql.NullString
		ccode, city       sql.NullString
		dnssec            sql.NullBool
		rel               sql.NullFloat64
		chkTime, crtTime  sql.NullTime
	)

	err := row.Scan(&id, &status, &address, &name, &asNumber, &asOrg, &ccode, &city, &ver, &errStr, &dnssec, &rel, &chkTime, &crtTime)
	if err != nil {
		return nil, err
	}

	r := &Nameserver{
		ID:      id,
		Address: address,
		Status:  status,
	}

	if name.Valid {
		r.Name = &name.String
	}
	if asNumber.Valid {
		v := uint(asNumber.Int32)
		r.ASNumber = &v
	}
	if asOrg.Valid {
		r.ASOrg = &asOrg.String
	}
	if ccode.Valid {
		s := strings.ToUpper(ccode.String)
		r.Country = &s
	}
	if city.Valid {
		r.City = &city.String
	}
	if ver.Valid {
		r.Version = &ver.String
	}
	if errStr.Valid {
		r.Error = &errStr.String
	}
	if dnssec.Valid {
		r.DNSSEC = &dnssec.Bool
	}
	if rel.Valid {
		r.Reliability = rel.Float64
	}
	if chkTime.Valid {
		t := chkTime.Time.UTC()
		r.CheckedAt = &t
	}
	if crtTime.Valid {
		r.CreatedAt = crtTime.Time.UTC()
	}
	return r, nil
}
