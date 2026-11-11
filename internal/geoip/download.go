package geoip

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	geoip2 "github.com/oschwald/geoip2-golang/v2"
)

// Editions downloaded by the Updater.
const (
	EditionCity = "GeoLite2-City"
	EditionASN  = "GeoLite2-ASN"
)

var editions = []string{EditionCity, EditionASN}

// downloadURL is MaxMind's download endpoint, %s is the edition.
const downloadURL = "https://download.maxmind.com/geoip/databases/%s/download?suffix=tar.gz"

// maxSize limits the size of a downloaded database.
const maxSize = 512 << 20

// Updater downloads the GeoLite2 databases from MaxMind into Dir.
type Updater struct {
	Dir        string
	AccountID  string
	LicenseKey string

	// MaxAge is the age up to which an existing file is not downloaded again.
	MaxAge time.Duration

	// URL is the download URL with %s for the edition. Defaults to MaxMind's
	// download endpoint.
	URL string

	Client *http.Client
}

// Path returns the path of the database file of the given edition.
func (u *Updater) Path(edition string) string {
	return filepath.Join(u.Dir, edition+".mmdb")
}

// Available tells whether the files of all editions exist.
func (u *Updater) Available() bool {
	for _, edition := range editions {
		if _, err := os.Stat(u.Path(edition)); err != nil {
			return false
		}
	}
	return true
}

// Update downloads every edition whose file is missing or older than MaxAge.
// A file is only replaced by a valid database of the same edition.
func (u *Updater) Update(ctx context.Context) error {
	var errs []error
	for _, edition := range editions {
		if info, err := os.Stat(u.Path(edition)); err == nil && time.Since(info.ModTime()) < u.MaxAge {
			continue
		}
		if err := u.download(ctx, edition); err != nil {
			errs = append(errs, fmt.Errorf("downloading %s: %w", edition, err))
		}
	}
	return errors.Join(errs...)
}

func (u *Updater) download(ctx context.Context, edition string) error {
	url := u.URL
	if url == "" {
		url = downloadURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(url, edition), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(u.AccountID, u.LicenseKey)

	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", res.Status)
	}

	target := u.Path(edition)
	tmp := target + ".tmp"
	defer os.Remove(tmp) // no-op after a successful rename

	if err := extract(res.Body, edition+".mmdb", tmp); err != nil {
		return err
	}
	if err := validate(tmp, edition); err != nil {
		return err
	}

	return os.Rename(tmp, target)
}

// extract writes the file with the given name from a tar.gz archive to dst.
func extract(r io.Reader, name, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s not found in archive", name)
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || path.Base(hdr.Name) != name {
			continue
		}

		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		n, err := io.Copy(f, io.LimitReader(tr, maxSize+1))
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
		if err == nil && n > maxSize {
			err = fmt.Errorf("%s exceeds %d bytes", name, maxSize)
		}
		return err
	}
}

// validate checks that the file is a database of the given edition.
func validate(file, edition string) error {
	reader, err := geoip2.Open(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	if got := reader.Metadata().DatabaseType; got != edition {
		return fmt.Errorf("unexpected database type %q", got)
	}
	return nil
}
