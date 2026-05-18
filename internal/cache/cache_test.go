package cache

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestCacheServesFromDisk(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		io.WriteString(w, "hello")
	}))
	defer srv.Close()
	dir := t.TempDir()
	transport := New(dir, http.DefaultTransport)
	c := &http.Client{Transport: transport}

	for i := 0; i < 3; i++ {
		resp, err := c.Get(srv.URL + "/x")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "hello" {
			t.Errorf("body: %q", body)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("upstream hits: got %d want 1", got)
	}
}

func TestCacheDifferentURLs(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		io.WriteString(w, "ok")
	}))
	defer srv.Close()
	c := &http.Client{Transport: New(t.TempDir(), http.DefaultTransport)}
	for _, path := range []string{"/a", "/b", "/a"} {
		resp, err := c.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	// /a hit once, /b hit once, /a served from cache the second time
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("hits: got %d want 2", got)
	}
}

func TestCacheSkipsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	dir := t.TempDir()
	c := &http.Client{Transport: New(dir, http.DefaultTransport)}
	for i := 0; i < 2; i++ {
		resp, err := c.Get(srv.URL + "/x")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	// Cache shouldn't have persisted the 500
	matches, _ := globAll(dir)
	if len(matches) != 0 {
		t.Errorf("cache should be empty after 500s, got %v", matches)
	}
}

func TestCacheSkipsPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "ok")
	}))
	defer srv.Close()
	dir := t.TempDir()
	c := &http.Client{Transport: New(dir, http.DefaultTransport)}
	resp, err := c.Post(srv.URL+"/x", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	matches, _ := globAll(dir)
	if len(matches) != 0 {
		t.Errorf("cache should not persist POSTs, got %v", matches)
	}
}

func globAll(dir string) ([]string, error) {
	matches := []string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		matches = append(matches, path)
		return nil
	})
	return matches, err
}
