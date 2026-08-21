package site

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/http-files/httpvet/report"
)

// The matrix is the cross-implementation view: probes × clients, with the
// per-cell state (or, for observe-class probes, the observed dynamic values)
// and the embedded byte evidence one click deep.
//
// It is a pure function over report.json files and the kit checkout beside
// them: every state is derived at build time from a report's embedded
// evidence by the kit's current check scripts, and nothing is stamped from
// the environment. `generated` is the latest run date among the inputs, not
// the clock, so the same reports and scripts render the same bytes on any
// machine at any hour.

//go:embed pages/matrix.html
var matrixTemplate string

//go:embed pages/matrix.css
var matrixStyles string

type mxClient struct {
	ID      string            `json:"id"`
	Name    string            `json:"name,omitempty"` // the client's own name for itself
	Docs    string            `json:"docs,omitempty"` // its guide to the format, one click from the column head
	Version string            `json:"version"`
	RunDate string            `json:"runDate"`
	Host    string            `json:"host"`
	Driver  *report.DriverDoc `json:"driver,omitempty"` // the manifest this lane ran under, when given
}

type mxProbe struct {
	ID    string         `json:"id"`
	Class string         `json:"class"`
	Spec  []string       `json:"spec"`
	Cells []*report.Cell `json:"cells"` // aligned to clients; null = not in that report
}

type mxData struct {
	Kit       string                     `json:"kit"`
	Generated string                     `json:"generated"`
	Corpus    report.CorpusInfo          `json:"corpus"`
	Clients   []mxClient                 `json:"clients"`
	Groups    []report.ReleaseGroup      `json:"groups"`         // release, then feature: the order every page reads in
	Probes    []mxProbe                  `json:"probes"`         // by id; a feature group names the ones it covers
	Docs      map[string]report.ProbeDoc `json:"docs,omitempty"` // corpus definitions, by probe id
	Spec      *report.SpecIndex          `json:"spec,omitempty"` // normative prose, by anchor
}

// MatrixOptions carries the optional inputs. Reports alone render the grid;
// a corpus adds what each probe *is* (document, fixture, assertions) and a
// spec directory adds the normative text each anchor cites. Both are
// display-only — nothing here feeds state derivation.
type MatrixOptions struct {
	Releases []report.ReleaseManifest
	Corpus   *report.CorpusDocs
	Spec     *report.SpecIndex
	Drivers  map[string]report.DriverDoc // by driver id
	Clients  map[string]ClientDoc        // by driver id
}

// MatrixHTML renders one or more report.json files as a self-contained
// features × clients evidence page. Column order is argument order, and the
// caller supplies reports in adoption order. Releases (spec manifests) are
// optional: with them, features group under the release that selected them and
// each release carries a conformance fold; without, every feature sits in the
// one section report.GroupByRelease leaves them in.
func MatrixHTML(reps []*report.Report, opts MatrixOptions) ([]byte, error) {
	if len(reps) == 0 {
		return nil, fmt.Errorf("no reports given")
	}
	// the corpus named is the one on disk — the present — falling back to what
	// the reports say when the page is drawn without a checkout
	corpus := reps[0].Corpus
	if opts.Corpus != nil {
		corpus = report.CorpusInfo{Version: opts.Corpus.Version}
	}

	data := mxData{
		Kit:       report.KitVersion,
		Generated: latestRunDate(reps),
		Corpus:    corpus,
		// never null: the page indexes this before it can draw anything, so
		// reports carrying no probes must still render as a page saying so
		Probes: []mxProbe{},
		Spec:   opts.Spec,
	}
	if opts.Corpus != nil {
		data.Docs = opts.Corpus.Probes
	}
	for _, rep := range reps {
		c := mxClient{
			ID:      rep.Driver.ID,
			Version: rep.Driver.ClientVersion,
			RunDate: rep.RunDate,
			Host:    rep.Host.OS + "/" + rep.Host.Arch,
		}
		if d, ok := opts.Drivers[rep.Driver.ID]; ok {
			c.Driver = &d
		}
		if cd, ok := opts.Clients[rep.Driver.ID]; ok {
			c.Name, c.Docs = cd.Name, cd.Docs
		}
		data.Clients = append(data.Clients, c)
	}

	// union of probes across reports; meta from the first report carrying it
	type probeMeta struct {
		class string
		spec  []string
	}
	metas := map[string]probeMeta{}
	var order []string
	for _, rep := range reps {
		for _, pe := range rep.Probes {
			if pe.Class == "" {
				return nil, fmt.Errorf("driver %s: probe %s has no class field — regenerate the report with this kit version", rep.Driver.ID, pe.ID)
			}
			if _, seen := metas[pe.ID]; !seen {
				metas[pe.ID] = probeMeta{class: pe.Class, spec: pe.Spec}
				order = append(order, pe.ID)
			}
		}
	}
	sort.Strings(order)

	byArea := map[string][]string{}
	for _, id := range order {
		meta := metas[id]
		mp := mxProbe{ID: id, Class: meta.class, Spec: meta.spec, Cells: make([]*report.Cell, len(reps))}
		for i, rep := range reps {
			for _, pe := range rep.Probes {
				if pe.ID == id {
					mp.Cells[i] = report.CellFromEntry(pe)
					break
				}
			}
		}
		data.Probes = append(data.Probes, mp)
		area := report.SpecArea(meta.spec)
		byArea[area] = append(byArea[area], id)
	}
	// the sections themselves: release order, manifest order within one, and a
	// last section for whatever no release selected — the same walk the report
	// page and the index take, so the three agree on what the spec contains
	data.Groups = report.GroupByRelease(byArea, opts.Releases)

	return report.Page{
		Title:    "http.vet · conformance matrix",
		Template: matrixTemplate,
		Styles:   matrixStyles,
		Data:     data,
		Local:    pageComponents("matrix"),
	}.Render()
}

// latestRunDate is the page's `generated` stamp: the most recent run among
// the reports being displayed. Deriving it from the inputs rather than the
// clock is what makes a published page reproducible from the reports beside
// it — regenerate, and either the bytes match or a report changed.
func latestRunDate(reps []*report.Report) string {
	latest := ""
	for _, rep := range reps {
		if rep.RunDate > latest { // ISO-8601 dates compare correctly as strings
			latest = rep.RunDate
		}
	}
	return latest
}
