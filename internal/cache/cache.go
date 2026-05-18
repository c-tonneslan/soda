// Package cache is a tiny content-keyed on-disk cache for GET responses.
//
// Civic-data work is iterative. You pull a dataset, look at it, change a
// SoQL filter, pull again. Hitting Socrata for the same bytes ten times in a
// row is slow and rude. This wraps http.RoundTripper with a file cache.
package cache

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
)

// Transport caches successful GET responses under Dir.
type Transport struct {
	Dir   string
	Inner http.RoundTripper
}

// New returns a Transport that uses ~/.cache/soda by default if dir == "".
func New(dir string, inner http.RoundTripper) *Transport {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache", "soda")
	}
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &Transport{Dir: dir, Inner: inner}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return t.Inner.RoundTrip(req)
	}
	path := t.pathFor(req)
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		resp, err := http.ReadResponse(bufio.NewReader(f), req)
		if err == nil {
			return resp, nil
		}
		// Stale/corrupt entry; fall through and refetch.
	}
	resp, err := t.Inner.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.write(path, resp)
	}
	return resp, nil
}

func (t *Transport) pathFor(req *http.Request) string {
	h := sha256.Sum256([]byte(req.URL.String()))
	hexkey := hex.EncodeToString(h[:])
	// One subdirectory level so we don't pile 10k files in one dir.
	return filepath.Join(t.Dir, hexkey[:2], hexkey+".http")
}

func (t *Transport) write(path string, resp *http.Response) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cache-*")
	if err != nil {
		return
	}
	if err := resp.Write(tmp); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return
	}
	tmp.Close()
	// After Write, the resp body is closed. Reopen the cached copy and replace
	// the body so the caller can still read it.
	_ = os.Rename(tmp.Name(), path)
	if f, err := os.Open(path); err == nil {
		rebuilt, err := http.ReadResponse(bufio.NewReader(f), resp.Request)
		if err == nil {
			resp.Body = rebuilt.Body
		} else {
			f.Close()
		}
	}
}
