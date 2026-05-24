// Package socrata is a thin client for the SODA (Socrata Open Data) API and
// the Discovery / catalog API.
//
// SODA reference: https://dev.socrata.com/docs/endpoints.html
// Discovery API: https://socratadiscovery.docs.apiary.io/
package socrata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout    = 60 * time.Second
	discoveryEndpoint = "https://api.us.socrata.com/api/catalog/v1"
	defaultMaxRetries = 3
	defaultRetryBase  = 500 * time.Millisecond
)

// Client talks to one Socrata portal at a time, plus the global Discovery API.
type Client struct {
	HTTP    *http.Client
	AppToken string // optional Socrata App Token; required for high-volume use, otherwise rate-limited

	// MaxRetries is the number of additional attempts on retryable
	// responses (HTTP 429 and 5xx). The zero value uses the default of 3.
	// Set it negative to disable retries entirely.
	MaxRetries int
	// RetryBase is the base delay for exponential backoff between
	// retries. The zero value uses the default of 500ms. A Retry-After
	// header from the server overrides this for that one attempt.
	RetryBase time.Duration
}

// New returns a client with a 60s timeout.
func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: defaultTimeout}}
}

// ----- catalog list

// Dataset is a row in a portal's catalog listing.
type Dataset struct {
	ID          string `json:"id"`           // four-by-four (e.g. "erm2-nwe9")
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Updated     string `json:"updated_at"`   // ISO 8601
	Rows        int64  `json:"rows"`
	Columns     int    `json:"columns"`
	URL         string `json:"url"`
}

// Catalog returns the first page of datasets in a portal's catalog. Socrata
// caps each request at 100; pass offset to walk further.
func (c *Client) Catalog(ctx context.Context, host string, limit, offset int) ([]Dataset, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	u := fmt.Sprintf("https://%s/api/views.json?limit=%d&offset=%d", host, limit, offset)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ID          string     `json:"id"`
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Category    string     `json:"category"`
		RowsUpdated int64      `json:"rowsUpdatedAt"`
		Columns     []struct{} `json:"columns"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}
	out := make([]Dataset, 0, len(raw))
	for _, r := range raw {
		out = append(out, Dataset{
			ID:          r.ID,
			Name:        r.Name,
			Description: trim(r.Description, 280),
			Category:    r.Category,
			Updated:     time.Unix(r.RowsUpdated, 0).UTC().Format(time.RFC3339),
			Columns:     len(r.Columns),
			URL:         fmt.Sprintf("https://%s/d/%s", host, r.ID),
		})
	}
	return out, nil
}

// ----- search across portals

// SearchHit is one record from the Discovery API.
type SearchHit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	Updated     string `json:"updated_at"`
	Permalink   string `json:"permalink"`
}

// Search runs a Discovery API query. If domains is non-empty, results are
// restricted to those hosts; otherwise the search spans every Socrata portal.
func (c *Client) Search(ctx context.Context, query string, domains []string, limit int) ([]SearchHit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	v := url.Values{}
	v.Set("q", query)
	v.Set("limit", strconv.Itoa(limit))
	if len(domains) > 0 {
		v.Set("domains", strings.Join(domains, ","))
	}
	body, err := c.get(ctx, discoveryEndpoint+"?"+v.Encode())
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results []struct {
			Resource struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				UpdatedAt   string `json:"updatedAt"`
			} `json:"resource"`
			Metadata struct {
				Domain string `json:"domain"`
			} `json:"metadata"`
			Permalink string `json:"permalink"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	out := make([]SearchHit, 0, len(resp.Results))
	for _, r := range resp.Results {
		out = append(out, SearchHit{
			ID:          r.Resource.ID,
			Name:        r.Resource.Name,
			Description: trim(r.Resource.Description, 200),
			Domain:      r.Metadata.Domain,
			Updated:     r.Resource.UpdatedAt,
			Permalink:   r.Permalink,
		})
	}
	return out, nil
}

// ----- dataset metadata (schema)

// Column describes one column of a Socrata dataset.
type Column struct {
	FieldName string `json:"fieldName"` // what you query with in SoQL
	Name      string `json:"name"`      // human-readable label
	DataType  string `json:"dataType"`
}

// Schema is the structural part of a dataset's metadata.
type Schema struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Updated     string   `json:"updated_at"`
	Attribution string   `json:"attribution"`
	Columns     []Column `json:"columns"`
}

// Info fetches the full metadata document for one dataset.
func (c *Client) Info(ctx context.Context, host, fourByFour string) (*Schema, error) {
	u := fmt.Sprintf("https://%s/api/views/%s.json", host, fourByFour)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Attribution string `json:"attribution"`
		RowsUpdated int64  `json:"rowsUpdatedAt"`
		Columns     []struct {
			FieldName    string `json:"fieldName"`
			Name         string `json:"name"`
			DataTypeName string `json:"dataTypeName"`
		} `json:"columns"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("info: %w", err)
	}
	cols := make([]Column, len(raw.Columns))
	for i, c := range raw.Columns {
		cols[i] = Column{FieldName: c.FieldName, Name: c.Name, DataType: c.DataTypeName}
	}
	return &Schema{
		ID:          raw.ID,
		Name:        raw.Name,
		Description: raw.Description,
		Updated:     time.Unix(raw.RowsUpdated, 0).UTC().Format(time.RFC3339),
		Attribution: raw.Attribution,
		Columns:     cols,
	}, nil
}

// ----- rows (the actual data)

// Format is how Socrata renders a /resource/<id> response.
type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// PullOptions controls a Rows() call.
type PullOptions struct {
	Format Format
	Limit  int    // 0 means default (Socrata's default is 1000)
	Offset int    // for paging
	Where  string // SoQL $where clause
	Order  string // SoQL $order clause
	Select string // SoQL $select clause
}

// Pages returns one page of rows as a []map[string]any (only meaningful for
// JSON output). Used by auto-pagination to walk the whole dataset.
func (c *Client) Pages(ctx context.Context, host, fourByFour string, opts PullOptions) ([]map[string]any, error) {
	opts.Format = FormatJSON
	body, err := c.Rows(ctx, host, fourByFour, opts)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var rows []map[string]any
	if err := json.NewDecoder(body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode page: %w", err)
	}
	return rows, nil
}

// Count returns the total row count for a dataset.
func (c *Client) Count(ctx context.Context, host, fourByFour string, where string) (int64, error) {
	opts := PullOptions{Format: FormatJSON, Select: "count(*)"}
	if where != "" {
		opts.Where = where
	}
	body, err := c.Rows(ctx, host, fourByFour, opts)
	if err != nil {
		return 0, err
	}
	defer body.Close()
	var out []struct {
		Count string `json:"count"`
	}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return 0, fmt.Errorf("decode count: %w", err)
	}
	if len(out) == 0 {
		return 0, nil
	}
	n, _ := strconv.ParseInt(out[0].Count, 10, 64)
	return n, nil
}

// Rows streams the response body for /resource/<id>.<format> with the given
// query options. The caller is responsible for closing the body.
func (c *Client) Rows(ctx context.Context, host, fourByFour string, opts PullOptions) (io.ReadCloser, error) {
	if opts.Format == "" {
		opts.Format = FormatJSON
	}
	v := url.Values{}
	if opts.Limit > 0 {
		v.Set("$limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		v.Set("$offset", strconv.Itoa(opts.Offset))
	}
	if opts.Where != "" {
		v.Set("$where", opts.Where)
	}
	if opts.Order != "" {
		v.Set("$order", opts.Order)
	}
	if opts.Select != "" {
		v.Set("$select", opts.Select)
	}
	u := fmt.Sprintf("https://%s/resource/%s.%s", host, fourByFour, opts.Format)
	if q := v.Encode(); q != "" {
		u += "?" + q
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if c.AppToken != "" {
		req.Header.Set("X-App-Token", c.AppToken)
	}
	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &APIError{Status: resp.StatusCode, URL: u, Body: string(body)}
	}
	return resp.Body, nil
}

// ----- internals

// APIError represents a non-2xx response from a Socrata endpoint.
type APIError struct {
	Status int
	URL    string
	Body   string
}

func (e *APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	return fmt.Sprintf("HTTP %d from %s: %s", e.Status, e.URL, body)
}

// Is lets callers check errors.Is(err, ErrNotFound) etc.
var ErrNotFound = errors.New("not found")

func (e *APIError) Is(target error) bool {
	return target == ErrNotFound && e.Status == http.StatusNotFound
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AppToken != "" {
		req.Header.Set("X-App-Token", c.AppToken)
	}
	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Status: resp.StatusCode, URL: url, Body: string(body)}
	}
	return body, nil
}

// doWithRetry runs req with backoff on retryable responses. Socrata
// rate-limits any anonymous client aggressively, so a quick exponential
// retry on 429 saves most one-off invocations that don't ship an App
// Token from failing on the first hit. Honors Retry-After when the
// server sets it.
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	maxRetries := c.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	} else if maxRetries < 0 {
		maxRetries = 0
	}
	base := c.RetryBase
	if base == 0 {
		base = defaultRetryBase
	}
	for attempt := 0; ; attempt++ {
		resp, err := c.HTTP.Do(req)
		if err != nil {
			// Network errors aren't retried — the caller's context is
			// the canonical place to bound the whole operation.
			return nil, err
		}
		if !isRetryable(resp.StatusCode) || attempt >= maxRetries {
			return resp, nil
		}
		// Drain and close so the connection can be reused, then sleep.
		wait := retryAfter(resp, base, attempt)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}
}

func isRetryable(status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500 && status <= 599
}

// retryAfter returns the delay before the next attempt. Prefer the
// server's Retry-After hint when present; otherwise back off exponentially
// from base. The hint can be either an integer number of seconds or an
// HTTP-date, per RFC 7231 §7.1.3.
func retryAfter(resp *http.Response, base time.Duration, attempt int) time.Duration {
	if h := resp.Header.Get("Retry-After"); h != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
		if t, err := http.ParseTime(h); err == nil {
			if d := time.Until(t); d > 0 {
				return d
			}
		}
	}
	return base << attempt
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
