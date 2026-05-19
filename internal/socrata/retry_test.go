package socrata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_RetriesOn429ThenSucceeds(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) < 3 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := &Client{
		HTTP:       srv.Client(),
		RetryBase:  1 * time.Millisecond,
		MaxRetries: 5,
	}
	if _, err := c.get(context.Background(), srv.URL+"/api/views.json?limit=1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := hits.Load(); got != 3 {
		t.Errorf("expected 3 attempts (2 retries + success), got %d", got)
	}
}

func TestClient_RetriesOn5xxThenGivesUp(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := &Client{
		HTTP:       srv.Client(),
		RetryBase:  1 * time.Millisecond,
		MaxRetries: 2,
	}
	_, err := c.get(context.Background(), srv.URL+"/api/views.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadGateway {
		t.Errorf("expected APIError with 502, got %T %v", err, err)
	}
	if got := hits.Load(); got != 3 {
		t.Errorf("expected 1 initial + 2 retries = 3 attempts, got %d", got)
	}
}

func TestClient_DoesNotRetry4xxOtherThan429(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), RetryBase: 1 * time.Millisecond, MaxRetries: 3}
	_, err := c.get(context.Background(), srv.URL+"/q")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("400 is not retryable, expected 1 attempt, got %d", got)
	}
}

func TestClient_HonorsRetryAfterSeconds(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), RetryBase: 10 * time.Second, MaxRetries: 1}
	start := time.Now()
	if _, err := c.get(context.Background(), srv.URL+"/q"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	// Retry-After: 1 should beat the 10s exponential default.
	if elapsed > 3*time.Second {
		t.Errorf("Retry-After ignored, slept for %v", elapsed)
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected to sleep ~1s for Retry-After=1, only waited %v", elapsed)
	}
}

func TestClient_CancelledContextStopsRetryLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30") // would be 30s wait
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), RetryBase: 30 * time.Second, MaxRetries: 5}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.get(ctx, srv.URL+"/q")
	if err == nil {
		t.Fatal("expected error after context cancel")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context deadline error, got %T %v", err, err)
	}
}
