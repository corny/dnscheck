package store

import "fmt"

// PurgeChecks deletes check results older than maxAgeDays and returns the
// number of deleted rows.
func (s *Store) PurgeChecks(maxAgeDays uint) (int64, error) {
	res, err := s.db.Exec(fmt.Sprintf("DELETE FROM nameserver_checks WHERE created_at < NOW() - INTERVAL '%d day'", maxAgeDays))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PurgeNameservers deletes invalid nameservers checked within the last 7
// days whose status has not changed for maxAgeDays and returns the number
// of deleted rows. Requiring a recent check keeps the nameservers if the
// checker has not been running.
func (s *Store) PurgeNameservers(maxAgeDays uint) (int64, error) {
	res, err := s.db.Exec(fmt.Sprintf("DELETE FROM nameservers WHERE status = FALSE AND checked_at > current_date - 7 AND status_changed_at < NOW() - INTERVAL '%d day'", maxAgeDays))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
