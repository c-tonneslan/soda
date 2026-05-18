package portals

import (
	"strings"
	"testing"
)

func TestGetKnown(t *testing.T) {
	p, err := Get("nyc")
	if err != nil {
		t.Fatalf("Get(nyc): %v", err)
	}
	if p.Host != "data.cityofnewyork.us" {
		t.Errorf("nyc host: got %q", p.Host)
	}
}

func TestGetUnknownListsKnown(t *testing.T) {
	_, err := Get("atlantis")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nyc") || !strings.Contains(err.Error(), "chicago") {
		t.Errorf("error should list known slugs, got: %v", err)
	}
}

func TestAllSorted(t *testing.T) {
	all := All()
	for i := 1; i < len(all); i++ {
		if all[i-1].Slug >= all[i].Slug {
			t.Errorf("portals not sorted at %d: %q before %q",
				i, all[i-1].Slug, all[i].Slug)
		}
	}
}

func TestURL(t *testing.T) {
	p := Portal{Slug: "x", Host: "example.com"}
	if got := p.URL("/api/views.json"); got != "https://example.com/api/views.json" {
		t.Errorf("URL: got %q", got)
	}
}
