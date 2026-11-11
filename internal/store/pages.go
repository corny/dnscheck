package store

import "strings"

// CountValid returns the number of valid nameservers.
func (s *Store) CountValid() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT COUNT(*) FROM nameservers WHERE status").Scan(&n)
	return n, err
}

// ValidCountries returns the codes of all countries with valid nameservers,
// in upper case.
func (s *Store) ValidCountries() ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT UPPER(country_code) FROM nameservers WHERE status AND country_code <> ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

// RecentValid returns the most recently checked valid nameservers.
func (s *Store) RecentValid(limit int) ([]*Nameserver, error) {
	return s.queryNameservers("SELECT "+nameserverColumns+" FROM nameservers WHERE status ORDER BY checked_at DESC LIMIT $1", limit)
}

// CountryValid returns the most recently checked valid nameservers of a
// country and the total number of valid nameservers in that country.
func (s *Store) CountryValid(country string, limit int) ([]*Nameserver, int, error) {
	country = strings.ToUpper(country)

	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM nameservers WHERE status AND country_code = $1", country).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	list, err := s.queryNameservers("SELECT "+nameserverColumns+" FROM nameservers WHERE status AND country_code = $1 ORDER BY checked_at DESC LIMIT $2", country, limit)
	return list, total, err
}

func (s *Store) queryNameservers(query string, args ...any) ([]*Nameserver, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Nameserver
	for rows.Next() {
		ns, err := scanNameserver(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, ns)
	}
	return list, rows.Err()
}
