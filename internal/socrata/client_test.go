package socrata

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeServer returns a server that maps {method, path} → handler.
func fakeServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, h := range routes {
		mux.HandleFunc(pattern, h)
	}
	return httptest.NewServer(mux)
}

func mustBody(t *testing.T, rc io.ReadCloser) string {
	t.Helper()
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestCatalog(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/api/views.json": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Errorf("limit: got %q want 2", got)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `[
				{"id":"abcd-1234","name":"A","description":"first","category":"x","rowsUpdatedAt":1700000000,"columns":[{},{}]},
				{"id":"efgh-5678","name":"B","description":"second","category":"y","rowsUpdatedAt":1700001000,"columns":[{}]}
			]`)
		},
	})
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}
	host := strings.TrimPrefix(srv.URL, "http://")
	// hack: re-build a client that hits http instead of https for the test
	c.HTTP = &http.Client{Transport: rewriteHTTPS(host, srv.URL)}

	datasets, err := c.Catalog(context.Background(), host, 2, 0)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(datasets) != 2 {
		t.Fatalf("got %d datasets, want 2", len(datasets))
	}
	if datasets[0].ID != "abcd-1234" || datasets[0].Name != "A" {
		t.Errorf("first dataset: %+v", datasets[0])
	}
	if datasets[0].Columns != 2 {
		t.Errorf("first dataset.Columns: got %d want 2", datasets[0].Columns)
	}
}

func TestInfo(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/api/views/abcd-1234.json": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{
				"id":"abcd-1234","name":"Dataset","description":"d","attribution":"NYC","rowsUpdatedAt":1700000000,
				"columns":[
					{"fieldName":"created_date","name":"Created","dataTypeName":"calendar_date"},
					{"fieldName":"complaint","name":"Complaint","dataTypeName":"text"}
				]
			}`)
		},
	})
	defer srv.Close()
	c := &Client{HTTP: &http.Client{Transport: rewriteHTTPS(strings.TrimPrefix(srv.URL, "http://"), srv.URL)}}
	host := strings.TrimPrefix(srv.URL, "http://")
	got, err := c.Info(context.Background(), host, "abcd-1234")
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got.Name != "Dataset" || got.Attribution != "NYC" {
		t.Errorf("schema: %+v", got)
	}
	if len(got.Columns) != 2 || got.Columns[0].FieldName != "created_date" {
		t.Errorf("columns: %+v", got.Columns)
	}
}

func TestSearch(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/api/catalog/v1": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("q"); got != "housing" {
				t.Errorf("q: got %q want 'housing'", got)
			}
			if got := r.URL.Query().Get("domains"); got != "data.cityofnewyork.us" {
				t.Errorf("domains: got %q", got)
			}
			io.WriteString(w, `{
				"results":[
					{"resource":{"id":"aaaa-1111","name":"Housing","description":"x","updatedAt":"2026-01-01"},
					 "metadata":{"domain":"data.cityofnewyork.us"},
					 "permalink":"https://x/y"}
				]
			}`)
		},
	})
	defer srv.Close()

	// The Discovery API is at a constant hostname; we can't override via the rewriter
	// easily, so just call srv directly.
	c := &Client{HTTP: srv.Client()}
	resp, err := c.HTTP.Get(srv.URL + "/api/catalog/v1?q=housing&domains=data.cityofnewyork.us&limit=5")
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	resp.Body.Close()

	// Use the real Search path via a transport that rewrites the public discovery URL.
	c.HTTP = &http.Client{Transport: rewriteAny(map[string]string{
		discoveryEndpoint: srv.URL + "/api/catalog/v1",
	})}
	hits, err := c.Search(context.Background(), "housing", []string{"data.cityofnewyork.us"}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != "aaaa-1111" || hits[0].Domain != "data.cityofnewyork.us" {
		t.Errorf("hits: %+v", hits)
	}
}

func TestRows(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/resource/abcd-1234.json": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("$where"); got != "borough='BROOKLYN'" {
				t.Errorf("$where: got %q", got)
			}
			if got := r.URL.Query().Get("$limit"); got != "5" {
				t.Errorf("$limit: got %q", got)
			}
			io.WriteString(w, `[{"borough":"BROOKLYN"}]`)
		},
	})
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")
	c := &Client{HTTP: &http.Client{Transport: rewriteHTTPS(host, srv.URL)}}

	body, err := c.Rows(context.Background(), host, "abcd-1234", PullOptions{
		Format: FormatJSON, Limit: 5, Where: "borough='BROOKLYN'",
	})
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	got := mustBody(t, body)
	if !strings.Contains(got, "BROOKLYN") {
		t.Errorf("body: %q", got)
	}
}

func TestRows404IsTypedError(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/resource/missing.json": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not found", http.StatusNotFound)
		},
	})
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")
	c := &Client{HTTP: &http.Client{Transport: rewriteHTTPS(host, srv.URL)}}
	_, err := c.Rows(context.Background(), host, "missing", PullOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %T %v", err, err)
	}
}

// ----- transport helpers

// rewriteHTTPS redirects requests for https://<host>/... to base + same path.
// We need this because the client builds https URLs from the configured host
// and httptest gives us http URLs.
func rewriteHTTPS(host, base string) http.RoundTripper {
	baseURL, _ := url.Parse(base)
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == host && req.URL.Scheme == "https" {
			req.URL.Scheme = baseURL.Scheme
			req.URL.Host = baseURL.Host
		}
		return http.DefaultTransport.RoundTrip(req)
	})
}

// rewriteAny maps fully-qualified prefixes to other prefixes.
func rewriteAny(mapping map[string]string) http.RoundTripper {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		u := req.URL.String()
		for src, dst := range mapping {
			if strings.HasPrefix(u, src) {
				replacement, err := url.Parse(dst + strings.TrimPrefix(u, src))
				if err != nil {
					return nil, err
				}
				req.URL = replacement
				break
			}
		}
		return http.DefaultTransport.RoundTrip(req)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
