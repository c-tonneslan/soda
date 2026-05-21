package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c-tonneslan/soda/internal/portals"
	"github.com/c-tonneslan/soda/internal/socrata"
)

// pull --all -o used to leave an empty file behind: the deferred closer()
// ran before the deferred final write, so the file was closed first and the
// JSON went nowhere. This pins that the document actually lands on disk.
func TestPullAllWritesFileForJSONAndGeoJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if off := r.URL.Query().Get("$offset"); off != "" && off != "0" {
			io.WriteString(w, `[]`)
			return
		}
		io.WriteString(w, `[
			{"id":"1","name":"A","loc":{"type":"Point","coordinates":[1,2]}},
			{"id":"2","name":"B","loc":{"type":"Point","coordinates":[3,4]}}
		]`)
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	client := &socrata.Client{HTTP: &http.Client{Transport: rewriteToTest(host, srv.URL)}}
	portal := portals.Portal{Slug: "test", Host: host}

	cases := []struct {
		format string
		want   string
	}{
		{"json", `"name"`},
		{"geojson", "FeatureCollection"},
	}
	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.json")
			err := pullAll(context.Background(), client, portal, "test-1234", pullParams{
				format: tc.format, outPath: out, pageSize: 1000, stderr: io.Discard,
			})
			if err != nil {
				t.Fatalf("pullAll: %v", err)
			}
			b, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}
			if len(b) == 0 {
				t.Fatal("output file is empty")
			}
			if !json.Valid(b) {
				t.Fatalf("output is not valid JSON: %s", b)
			}
			if !strings.Contains(string(b), tc.want) {
				t.Errorf("expected %q in output, got: %s", tc.want, b)
			}
		})
	}
}

func rewriteToTest(host, base string) http.RoundTripper {
	baseURL, _ := url.Parse(base)
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == host {
			req.URL.Scheme = baseURL.Scheme
			req.URL.Host = baseURL.Host
		}
		return http.DefaultTransport.RoundTrip(req)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
