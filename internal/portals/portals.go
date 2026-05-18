// Package portals is soda's registry of known Socrata-based open data portals.
//
// Each entry is a one-line config. Adding a portal is a registry change, not
// a code change. Find your portal's bare hostname (the part before
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
	// US cities — largest first within each region
	{Slug: "nyc", Name: "New York City, NY", Host: "data.cityofnewyork.us"},
	{Slug: "chicago", Name: "Chicago, IL", Host: "data.cityofchicago.org"},
	{Slug: "la", Name: "Los Angeles, CA", Host: "data.lacity.org"},
	{Slug: "seattle", Name: "Seattle, WA", Host: "data.seattle.gov"},
	{Slug: "sf", Name: "San Francisco, CA", Host: "data.sfgov.org"},
	{Slug: "dc", Name: "Washington, DC", Host: "opendata.dc.gov"},
	{Slug: "boston", Name: "Boston, MA", Host: "data.boston.gov"},
	{Slug: "baltimore", Name: "Baltimore, MD", Host: "data.baltimorecity.gov"},
	{Slug: "philly", Name: "Philadelphia, PA", Host: "data.phila.gov"},
	{Slug: "austin", Name: "Austin, TX", Host: "data.austintexas.gov"},
	{Slug: "dallas", Name: "Dallas, TX", Host: "www.dallasopendata.com"},
	{Slug: "houston", Name: "Houston, TX", Host: "data.houstontx.gov"},
	{Slug: "denver", Name: "Denver, CO", Host: "denvergov.org"},
	{Slug: "honolulu", Name: "Honolulu, HI", Host: "data.honolulu.gov"},
	{Slug: "kc", Name: "Kansas City, MO", Host: "data.kcmo.org"},
	{Slug: "neworleans", Name: "New Orleans, LA", Host: "data.nola.gov"},
	{Slug: "nashville", Name: "Nashville, TN", Host: "data.nashville.gov"},
	{Slug: "stl", Name: "St. Louis, MO", Host: "www.stlouis-mo.gov"},
	{Slug: "albany", Name: "Albany, NY", Host: "data.albanycountyny.gov"},
	{Slug: "buffalo", Name: "Buffalo, NY", Host: "data.buffalony.gov"},
	{Slug: "rochester", Name: "Rochester, NY", Host: "data.cityofrochester.gov"},
	{Slug: "syracuse", Name: "Syracuse, NY", Host: "data.syrgov.net"},
	{Slug: "sandiego", Name: "San Diego, CA", Host: "data.sandiego.gov"},
	{Slug: "sanjose", Name: "San Jose, CA", Host: "data.sanjoseca.gov"},
	{Slug: "longbeach", Name: "Long Beach, CA", Host: "data.longbeach.gov"},
	{Slug: "oakland", Name: "Oakland, CA", Host: "data.oaklandca.gov"},
	{Slug: "miami", Name: "Miami, FL", Host: "datahub.miamigov.com"},
	{Slug: "tampa", Name: "Tampa, FL", Host: "data.tampagov.net"},
	{Slug: "raleigh", Name: "Raleigh, NC", Host: "data-ral.opendata.arcgis.com"},
	{Slug: "charlotte", Name: "Charlotte, NC", Host: "data.charlottenc.gov"},
	// US counties
	{Slug: "cookcounty", Name: "Cook County, IL", Host: "datacatalog.cookcountyil.gov"},
	{Slug: "kingcounty", Name: "King County, WA", Host: "data.kingcounty.gov"},
	{Slug: "montgomerymd", Name: "Montgomery County, MD", Host: "data.montgomerycountymd.gov"},
	{Slug: "fairfaxva", Name: "Fairfax County, VA", Host: "data.fairfaxcountyva.gov"},
	// US states
	{Slug: "ctstate", Name: "Connecticut (state)", Host: "data.ct.gov"},
	{Slug: "nystate", Name: "New York (state)", Host: "data.ny.gov"},
	{Slug: "mdstate", Name: "Maryland (state)", Host: "opendata.maryland.gov"},
	{Slug: "wastate", Name: "Washington (state)", Host: "data.wa.gov"},
	{Slug: "ilstate", Name: "Illinois (state)", Host: "data.illinois.gov"},
	{Slug: "txstate", Name: "Texas (state)", Host: "data.texas.gov"},
	{Slug: "orstate", Name: "Oregon (state)", Host: "data.oregon.gov"},
	{Slug: "hawaiistate", Name: "Hawaii (state)", Host: "data.hawaii.gov"},
	{Slug: "iowastate", Name: "Iowa (state)", Host: "data.iowa.gov"},
	{Slug: "vermontstate", Name: "Vermont (state)", Host: "data.vermont.gov"},
	// Federal
	{Slug: "healthdata", Name: "HealthData.gov", Host: "healthdata.gov"},
	{Slug: "cdc", Name: "CDC", Host: "data.cdc.gov"},
	{Slug: "medicare", Name: "Medicare.gov", Host: "data.medicare.gov"},
	{Slug: "energy", Name: "U.S. Department of Energy", Host: "data.energy.gov"},
	{Slug: "transportation", Name: "U.S. Department of Transportation", Host: "data.transportation.gov"},
	// International
	{Slug: "edmonton", Name: "Edmonton, AB", Host: "data.edmonton.ca"},
	{Slug: "australia", Name: "Australia (federal)", Host: "data.gov.au"},
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
