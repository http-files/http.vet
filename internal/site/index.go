package site

import (
	_ "embed"

	"github.com/http-files/httpvet/report"
)

// The root page asks how the standard is progressing, which is a different
// question from the two surfaces either side of it: a client owner's report
// page asks how one client is doing, and http-files.org asks what the language
// is. Here the unit is the release. Reading down a release's column says
// whether that year's selection is settled — everyone at or near the top means
// the wording holds; a ragged column means it does not yet. The features no
// release has selected are the frontier, and they get a column too.
//
// Every state here is derived at build time from each report's embedded
// evidence by the kit's current check scripts; the reports are the record,
// the scripts are the present.

//go:embed pages/index.html
var indexTemplate string

//go:embed pages/index.css
var indexStyles string

// ixBucket is one column: a release's own selected features, or everything no
// release has claimed.
type ixBucket struct {
	Year     string `json:"year,omitempty"` // absent for the unselected frontier
	Label    string `json:"label"`
	Status   string `json:"status,omitempty"` // draft | settled
	Note     string `json:"note,omitempty"`   // what the last column collects, or the input it lacks
	Features int    `json:"features"`
	Probes   int    `json:"probes"`  // graded probes in this bucket
	Conform  int    `json:"conform"` // clients conforming on every one of them
}

// ixScore is one client against one bucket.
type ixScore struct {
	Graded     int `json:"graded"`
	Conforming int `json:"conforming"`
}

// The conforming fraction is over graded probes only. An observe-class probe
// records what a client does where the format says nothing, so it carries no
// verdict — folding it into the headline either way would state a conclusion
// the corpus does not reach.
type ixClient struct {
	ID         string    `json:"id"`
	Name       string    `json:"name,omitempty"`
	Docs       string    `json:"docs,omitempty"`
	Version    string    `json:"version,omitempty"`
	RunDate    string    `json:"runDate"`
	Total      int       `json:"total"`
	Graded     int       `json:"graded"`
	Conforming int       `json:"conforming"`
	Scores     []ixScore `json:"scores"` // aligned to the version's buckets
}

type ixVersion struct {
	Version   string     `json:"version"`
	Kit       string     `json:"kit"`
	Generated string     `json:"generated"`
	Probes    int        `json:"probes"`
	Buckets   []ixBucket `json:"buckets"`
	Clients   []ixClient `json:"clients"`
}

type ixData struct {
	Generated string      `json:"generated"`
	Versions  []ixVersion `json:"versions"` // newest first
}

// bucketsFor turns the shared release grouping into columns, one per section:
// a column counts only the features that release selected, not the prior years
// it also includes — the cumulative reading is what the matrix's release rows
// give, and what this page is for is seeing each year's work on its own.
// Features no release has selected are the last column.
//
// The returned map sends a feature to the column that owns it, which is how a
// probe finds its column: the grouping puts a feature under one release, so
// there is nothing here to decide.
func bucketsFor(byArea map[string][]string, releases []report.ReleaseManifest) ([]ixBucket, map[string]int) {
	groups := report.GroupByRelease(byArea, releases)
	buckets := make([]ixBucket, 0, len(groups))
	column := make(map[string]int, len(byArea))
	for i, g := range groups {
		for _, f := range g.Features {
			column[f.Name] = i
		}
		buckets = append(buckets, ixBucket{
			Year: g.Year, Label: g.Label, Status: g.Status, Note: g.Note,
			Features: len(g.Features),
		})
	}
	return buckets, column
}

// IndexOptions carries the optional inputs, matching MatrixOptions so the two
// call sites read alike. Reports alone render the table; a corpus names the
// version the page is built against, and a spec's manifests give it its
// columns.
type IndexOptions struct {
	Releases []report.ReleaseManifest
	Corpus   *report.CorpusDocs
	Clients  map[string]ClientDoc // by driver id
}

// IndexHTML renders the root page over the submitted reports, already regraded
// by the caller. Like every page here it is a pure function over its inputs:
// `generated` is the newest run date on the site, never the clock. Releases
// are optional — without them the grouping leaves every feature unclaimed, so
// the table is one column that says which input is missing rather than a
// different page.
func IndexHTML(files []ReportFile, opts IndexOptions) ([]byte, error) {
	data := ixData{}
	{
		v := ixVersion{
			Kit: report.KitVersion,
			// no submissions must reach the page as an empty list, not null:
			// the page reads .length before it reaches its own empty-state
			// branch, and one nil here blanks the whole page
			Clients: []ixClient{},
			Buckets: []ixBucket{},
		}
		// the corpus named is the one on disk — the present — falling back to
		// what the reports say when the page is drawn without a checkout.
		// Reading it from the first report alone lets one lane left behind by
		// a corpus bump label every other, and moves the label whenever lane
		// order does.
		if len(files) > 0 {
			v.Version = files[0].Report.Corpus.Version
		}
		if opts.Corpus != nil {
			v.Version = opts.Corpus.Version
		}

		// every feature the corpus reaches, from the reports themselves; the
		// lanes share a corpus, so a probe is counted once however many
		// submitted it
		var entries []report.ProbeEntry
		seen := map[string]bool{}
		for _, f := range files {
			for _, pe := range f.Report.Probes {
				if seen[pe.ID] {
					continue
				}
				seen[pe.ID] = true
				entries = append(entries, pe)
			}
		}
		var column map[string]int
		if len(files) > 0 {
			v.Buckets, column = bucketsFor(report.ProbesByArea(entries), opts.Releases)
		}

		for _, f := range files {
			c := ixClient{
				ID:      f.ID,
				Version: f.Report.Driver.ClientVersion,
				RunDate: f.Report.RunDate,
				Total:   len(f.Report.Probes),
				Scores:  make([]ixScore, len(v.Buckets)),
			}
			if cd, ok := opts.Clients[f.ID]; ok {
				c.Name, c.Docs = cd.Name, cd.Docs
			}
			for _, pe := range f.Report.Probes {
				if pe.Class == "observe" {
					continue
				}
				ok := report.Conforming(pe.Class, pe.State)
				c.Graded++
				if ok {
					c.Conforming++
				}
				if bi, in := column[report.SpecArea(pe.Spec)]; in {
					c.Scores[bi].Graded++
					if ok {
						c.Scores[bi].Conforming++
					}
				}
			}
			if c.Total > v.Probes {
				v.Probes = c.Total
			}
			if c.RunDate > v.Generated {
				v.Generated = c.RunDate
			}
			v.Clients = append(v.Clients, c)
		}

		// a bucket's probe count is the corpus's, so take it from any lane;
		// its conform count is how many lanes cleared the whole bucket
		for bi := range v.Buckets {
			for _, c := range v.Clients {
				if c.Scores[bi].Graded > v.Buckets[bi].Probes {
					v.Buckets[bi].Probes = c.Scores[bi].Graded
				}
				if c.Scores[bi].Graded > 0 && c.Scores[bi].Conforming == c.Scores[bi].Graded {
					v.Buckets[bi].Conform++
				}
			}
		}

		if v.Generated > data.Generated {
			data.Generated = v.Generated
		}
		data.Versions = append(data.Versions, v)
	}

	return report.Page{
		Title:    "http.vet · .http standardization",
		Template: indexTemplate,
		Styles:   indexStyles,
		Data:     data,
		Local:    pageComponents("index"),
	}.Render()
}
