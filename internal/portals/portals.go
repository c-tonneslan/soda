// Package portals is convene's registry of known Socrata-based open data portals.
//
// Each entry is a one-line entry. Adding a new portal is a config change, not
// a code change. If your portal isn't here, find its hostname (the part before
// /resource/<id>.json) and add a Portal{} row.
package portals

import (
	"fmt"
	"sort"
	"strings"
)

// Portal is one Socrata-powered open-data site.
type Portal struct {
	Slug string // CLI handle, e.g. "nyc"
	Name string // human-readable
	Host string // bare hostname, no scheme (e.g. "data.cityofnewyork.us")
	Note string // optional caveat
}

// URL builds an https URL for the given path on this portal.
func (p Portal) URL(path string) string {
	return "https://" + p.Host + path
}

var all = []Portal{
	{Slug: "nyc", Name: "New York City, NY", Host: "data.cityofnewyork.us"},
	{Slug: "chicago", Name: "Chicago, IL", Host: "data.cityofchicago.org"},
	{Slug: "la", Name: "Los Angeles, CA", Host: "data.lacity.org"},
	{Slug: "seattle", Name: "Seattle, WA", Host: "data.seattle.gov"},
	{Slug: "sf", Name: "San Francisco, CA", Host: "data.sfgov.org"},
	{Slug: "dc", Name: "Washington, DC", Host: "opendata.dc.gov"},
	{Slug: "cookcounty", Name: "Cook County, IL", Host: "datacatalog.cookcountyil.gov"},
	{Slug: "kingcounty", Name: "King County, WA", Host: "data.kingcounty.gov"},
	{Slug: "boston", Name: "Boston, MA", Host: "data.boston.gov"},
	{Slug: "baltimore", Name: "Baltimore, MD", Host: "data.baltimorecity.gov"},
	{Slug: "neworleans", Name: "New Orleans, LA", Host: "data.nola.gov"},
	{Slug: "austin", Name: "Austin, TX", Host: "data.austintexas.gov"},
	{Slug: "dallas", Name: "Dallas, TX", Host: "www.dallasopendata.com"},
	{Slug: "denver", Name: "Denver, CO", Host: "denvergov.org"},
	{Slug: "honolulu", Name: "Honolulu, HI", Host: "data.honolulu.gov"},
	{Slug: "albany", Name: "Albany, NY", Host: "data.albanycountyny.gov"},
	{Slug: "buffalo", Name: "Buffalo, NY", Host: "data.buffalony.gov"},
	{Slug: "ctstate", Name: "Connecticut (state)", Host: "data.ct.gov"},
	{Slug: "nystate", Name: "New York (state)", Host: "data.ny.gov"},
	{Slug: "mdstate", Name: "Maryland (state)", Host: "opendata.maryland.gov"},
	{Slug: "wastate", Name: "Washington (state)", Host: "data.wa.gov"},
	{Slug: "healthdata", Name: "HealthData.gov", Host: "healthdata.gov"},
	{Slug: "cdc", Name: "CDC", Host: "data.cdc.gov"},
}

// All returns the preconfigured portals, sorted by slug.
func All() []Portal {
	out := make([]Portal, len(all))
	copy(out, all)
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// Get returns the portal with the given slug, or an error listing known slugs.
func Get(slug string) (Portal, error) {
	for _, p := range all {
		if p.Slug == slug {
			return p, nil
		}
	}
	slugs := make([]string, len(all))
	for i, p := range all {
		slugs[i] = p.Slug
	}
	sort.Strings(slugs)
	return Portal{}, fmt.Errorf("unknown portal %q. known: %s",
		slug, strings.Join(slugs, ", "))
}
