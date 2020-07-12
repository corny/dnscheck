package export

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Nameserver describes a database record.
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

// IsValid tells you whether this Nameserver is valid (i.e. not "new"
// or "failed").
func (ns *Nameserver) IsValid() bool {
	return ns.Status
}

// GetString returns a string representation of the attribute given.
func (ns *Nameserver) GetString(attr string) string {
	nullStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	var res string
	switch attr {
	case "ip_address":
		res = ns.Address
	case "name":
		res = nullStr(ns.Name)
	case "as_number":
		if ns.ASNumber != nil {
			res = strconv.Itoa(int(*ns.ASNumber))
		}
	case "as_org":
		res = nullStr(ns.ASOrg)
	case "country_code":
		res = nullStr(ns.Country)
	case "city":
		res = nullStr(ns.City)
	case "version":
		res = nullStr(ns.Version)
	case "error":
		res = nullStr(ns.Error)
	case "dnssec":
		if ns.DNSSEC == nil {
			res = ""
		} else if *ns.DNSSEC {
			res = "true"
		} else {
			res = "false"
		}
	case "reliability":
		res = strconv.FormatFloat(ns.Reliability, 'f', 2, 64)
	case "checked_at":
		if ns.CheckedAt == nil {
			res = ""
		} else {
			res = ns.CheckedAt.Format(time.RFC3339)
		}
	case "created_at":
		res = ns.CreatedAt.Format(time.RFC3339)
	default:
		log.Panicln("invalid attribute:", attr)
	}
	return res
}

var (
	// these fields are exported
	nameserverFields = []string{
		"ip_address",
		"name",
		"as_number",
		"as_org",
		"country_code",
		"city",
		"version",
		"error",
		"dnssec",
		"reliability",
		"checked_at",
		"created_at",
	}

	// prepared statement (built from queryNameserverSlice and nameserverFields)
	stmtNameserverSlice *sql.Stmt
)

const queryNameserverSlice = `SELECT "id", "status", "%s" FROM nameservers WHERE "id" > $1 ORDER BY "id" ASC LIMIT $2`

// Each iterates over all Nameserver records and calls the callback for
// each. If the callback returns an error, the iteration stops and that
// error is returned (note: other errors can be returned from this as
// well).
func Each(conn *sql.DB, batchSize uint, callback func(*Nameserver) error) (err error) {
	var (
		rows  *sql.Rows
		curr  = 0
		found = batchSize
	)

	if stmtNameserverSlice == nil {
		q := fmt.Sprintf(queryNameserverSlice, strings.Join(nameserverFields, `", "`))
		stmtNameserverSlice, err = conn.Prepare(strings.TrimSpace(q))
		if err != nil {
			return
		}
	}

	for batchSize == found {
		// Read the next batch
		rows, err = stmtNameserverSlice.Query(curr, batchSize)
		if err != nil {
			return
		}

		found = 0
		for rows.Next() {
			var current *Nameserver
			current, err = scanRow(rows)

			if err != nil {
				return
			}
			curr = current.ID
			if err = callback(current); err != nil {
				return
			}
			found++
		}
		rows.Close()
	}

	return nil
}

func scanRow(row *sql.Rows) (*Nameserver, error) {
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
		chkTime, crtTime  mysql.NullTime
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
