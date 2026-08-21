package site

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/http-files/httpvet/report"
)

// fakeReport builds a minimal report for the renderers. The kit has its own
// copy of this helper; duplicating five lines is cheaper than exporting test
// scaffolding across a module boundary.
func fakeReport(driverID string, entries ...report.ProbeEntry) *report.Report {
	return &report.Report{
		Schema:  1,
		Kit:     report.KitInfo{Version: report.KitVersion},
		Corpus:  report.CorpusInfo{Version: "2026-draft.1"},
		Driver:  report.DriverInfo{ID: driverID, ClientVersion: "1.0"},
		Host:    report.HostInfo{OS: "darwin", Arch: "arm64"},
		Mode:    "automated",
		RunDate: "2026-08-16",
		Probes:  entries,
	}
}

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, err := LoadRegistry("../../clients.toml")
	if err != nil {
		t.Fatalf("clients.toml: %v", err)
	}
	return reg
}

func writeTOML(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clients.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeReport files a report where intake expects it and returns the root.
func writeReport(t *testing.T, root, name string, rep *report.Report) {
	t.Helper()
	if err := os.MkdirAll(ReportsDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := report.Write(filepath.Join(ReportsDir(root), name), rep); err != nil {
		t.Fatal(err)
	}
}

// The registry carries both halves of one statement: which lanes exist and
// what order their columns stand in. A lane in one half and not the other
// would either lose its column or arrive unnamed, so loading refuses it.
func TestRegistryHalvesAgree(t *testing.T) {
	reg := testRegistry(t)
	if len(reg.Lanes) != len(reg.Clients) {
		t.Errorf("%d lanes but %d client entries", len(reg.Lanes), len(reg.Clients))
	}
	for _, id := range reg.Lanes {
		if c, ok := reg.Clients[id]; !ok || c.Name == "" {
			t.Errorf("%s: lane without a usable registry entry", id)
		}
	}
}

func TestRegistryRejectsUnlistedLane(t *testing.T) {
	path := writeTOML(t, "lanes = [\"a\"]\n[client.a]\nname = \"A\"\n[client.b]\nname = \"B\"\n")
	if _, err := LoadRegistry(path); err == nil {
		t.Error("a client with no position in `lanes` must be refused")
	}
}

func TestRegistryRejectsMissingEntry(t *testing.T) {
	path := writeTOML(t, "lanes = [\"a\", \"b\"]\n[client.a]\nname = \"A\"\n")
	if _, err := LoadRegistry(path); err == nil {
		t.Error("a lane with no [client.<id>] entry must be refused")
	}
}

func TestRegistryRejectsNonHTTPS(t *testing.T) {
	path := writeTOML(t, "lanes = [\"x\"]\n[client.x]\nname = \"X\"\ndocs = \"http://example.invalid/guide\"\n")
	if _, err := LoadRegistry(path); err == nil {
		t.Error("plain http should be refused")
	}
}

func TestRegistryRejectsEmpty(t *testing.T) {
	path := writeTOML(t, "# nothing here\n")
	if _, err := LoadRegistry(path); err == nil {
		t.Error("a registry with no entries is not a registry")
	}
}

// Column order is the registry's, not the filesystem's — and a report from an
// unregistered lane is refused rather than appended.
func TestOrderedFollowsRegistry(t *testing.T) {
	reg := &Registry{
		Lanes:   []string{"first", "second", "third"},
		Clients: map[string]ClientDoc{"first": {ID: "first"}, "second": {ID: "second"}, "third": {ID: "third"}},
	}
	got, err := reg.Ordered(map[string]bool{"third": true, "first": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "third" {
		t.Errorf("registry order not honoured: %v", got)
	}
	if _, err := reg.Ordered(map[string]bool{"stranger": true}); err == nil {
		t.Error("an unregistered lane must be refused")
	}
}

// generated is derived from the reports, never the clock: the same inputs must
// render the same bytes.
func TestMatrixGeneratedIsLatestRunDate(t *testing.T) {
	entry := report.ProbeEntry{ID: "p", Class: "behavioral", Spec: []string{"request-line#method"}, State: "supported"}
	older := fakeReport("httpyac", entry)
	older.RunDate = "2026-08-01"
	newer := fakeReport("kulala", entry)
	newer.RunDate = "2026-08-16"

	first, err := MatrixHTML([]*report.Report{older, newer}, MatrixOptions{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := MatrixHTML([]*report.Report{older, newer}, MatrixOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("the same reports must render identical bytes")
	}
	if got := latestRunDate([]*report.Report{older, newer}); got != "2026-08-16" {
		t.Errorf("generated should be the newest run date, got %q", got)
	}
}

func TestMatrixRenders(t *testing.T) {
	entry := report.ProbeEntry{ID: "request-get-basic", Class: "behavioral",
		Spec: []string{"request-line#method"}, State: "supported"}
	b, err := MatrixHTML([]*report.Report{fakeReport("httpyac", entry)}, MatrixOptions{})
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{"httpyac", "request-get-basic", "<!doctype html>", `"generated":"2026-08-16"`} {
		if !strings.Contains(page, want) {
			t.Errorf("page missing %q", want)
		}
	}
	if strings.Contains(page, "__HTTPVET_DATA__") {
		t.Error("data placeholder not substituted")
	}
	if _, err := MatrixHTML(nil, MatrixOptions{}); err == nil {
		t.Error("a matrix over no reports is an error")
	}
}

// The published pages are generated from the committed reports, so the real
// intake directory must render — this is the check that a submitted report
// which no longer loads is caught by `go test`, not by a broken site.
func TestSiteRendersFromCommittedReports(t *testing.T) {
	reg := testRegistry(t)
	files, err := LoadReports("../..", reg)
	if err != nil {
		t.Fatalf("reports/: %v", err)
	}
	t.Logf("rendering %d lanes", len(files))
	for i, f := range files {
		if f.Report.Driver.ID != f.ID {
			t.Errorf("%s: driver id %q disagrees with filename", f.Path, f.Report.Driver.ID)
		}
		if i > 0 && slices.Index(reg.Lanes, f.ID) < slices.Index(reg.Lanes, files[i-1].ID) {
			t.Errorf("%s: reports out of registry order", f.ID)
		}
	}
	if _, err := MatrixHTML(Reports(files), MatrixOptions{Clients: reg.Clients}); err != nil {
		t.Errorf("committed reports do not render: %v", err)
	}
	if _, err := IndexHTML(files, IndexOptions{Clients: reg.Clients}); err != nil {
		t.Errorf("index does not render: %v", err)
	}
}

// A submission must be named for the lane it grades; otherwise a report could
// land in another client's column by filename alone.
func TestReportFilenameMustMatchDriverID(t *testing.T) {
	root := t.TempDir()
	writeReport(t, root, "report-httpyac.json", fakeReport("kulala", report.ProbeEntry{ID: "p", Class: "behavioral", State: "supported"}))
	reg := &Registry{Lanes: []string{"httpyac", "kulala"},
		Clients: map[string]ClientDoc{"httpyac": {ID: "httpyac"}, "kulala": {ID: "kulala"}}}
	if _, err := LoadReports(root, reg); err == nil {
		t.Error("a report whose filename disagrees with its driver id must be refused")
	}
}

// A site with nothing submitted still has a root page; only a matrix over
// nothing is an error.
func TestNoReportsIsNotAnError(t *testing.T) {
	root := t.TempDir()
	reg := &Registry{Lanes: []string{"httpyac"}, Clients: map[string]ClientDoc{"httpyac": {ID: "httpyac", Name: "httpyac"}}}
	if _, err := LoadReports(root, reg); !errors.Is(err, ErrNoReports) {
		t.Fatalf("a missing intake folder must report ErrNoReports, got %v", err)
	}
	if err := os.MkdirAll(ReportsDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReports(root, reg); !errors.Is(err, ErrNoReports) {
		t.Fatalf("an empty intake folder must report ErrNoReports, got %v", err)
	}
	b, err := IndexHTML(nil, IndexOptions{Clients: reg.Clients})
	if err != nil {
		t.Fatalf("the index must render with no submissions: %v", err)
	}
	page := string(b)
	// null here is not cosmetic: the template reads .length before reaching its
	// own empty-state branch, so one null blanks the entire page while the
	// build reports success
	if strings.Contains(page, `"clients":null`) {
		t.Error("no submissions must render as an empty client list, not null")
	}
	if !strings.Contains(page, `"clients":[]`) {
		t.Error("expected an empty client list")
	}
}

// The root page's question is whether a release's selection holds, so a probe
// counts toward the release that selected its feature and toward no other. A
// feature no release has claimed lands in the frontier column instead of being
// dropped — divergence out there is the expected finding, not a missing one.
func TestIndexBucketsByRelease(t *testing.T) {
	releases := []report.ReleaseManifest{
		{Year: "2026", Status: "draft", Features: []string{"request-line", "comments"}},
		{Year: "2027", Status: "draft", Features: []string{"method-verbatim"}},
	}
	byArea := map[string][]string{
		"request-line": {"get"}, "comments": {"comment-hash"},
		"method-verbatim": {"method-lowercase"}, "message-body": {"body-json"},
	}
	buckets, column := bucketsFor(byArea, releases)

	if len(buckets) != 3 {
		t.Fatalf("expected a column per release plus the frontier, got %d", len(buckets))
	}
	if buckets[0].Year != "2026" || buckets[0].Features != 2 {
		t.Errorf("2026 column wrong: %+v", buckets[0])
	}
	if buckets[1].Year != "2027" || buckets[1].Features != 1 {
		t.Errorf("2027 column counts only what that year added: %+v", buckets[1])
	}
	if buckets[2].Year != "" || buckets[2].Label != report.UnreleasedLabel || buckets[2].Features != 1 {
		t.Errorf("unclaimed features belong to the frontier column: %+v", buckets[2])
	}
	for feature, want := range map[string]int{
		"request-line": 0, "comments": 0, "method-verbatim": 1, "message-body": 2,
	} {
		if got, in := column[feature]; !in || got != want {
			t.Errorf("%s belongs to column %d, got %d (present: %v)", feature, want, got, in)
		}
	}
}

// Without manifests the page keeps its shape: one column holding every feature,
// every client still standing in it, and a head saying which input is missing.
func TestIndexWithoutReleasesKeepsEveryClient(t *testing.T) {
	root := t.TempDir()
	writeReport(t, root, "report-httpyac.json", fakeReport("httpyac",
		report.ProbeEntry{ID: "get", Class: "behavioral", State: "supported", Spec: []string{"request-line#target"}},
		report.ProbeEntry{ID: "body", Class: "behavioral", State: "deviating", Spec: []string{"message-body#blank-line"}}))
	reg := &Registry{Lanes: []string{"httpyac"}, Clients: map[string]ClientDoc{"httpyac": {ID: "httpyac", Name: "httpyac"}}}
	files, err := LoadReports(root, reg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := IndexHTML(files, IndexOptions{Clients: reg.Clients})
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	if !strings.Contains(page, `"label":"`+report.UnreleasedLabel+`","note":"`+report.NoReleasesNote+`"`) {
		t.Error("the one column must carry the unclaimed label and say the manifests are missing")
	}
	if !strings.Contains(page, `"features":2,"probes":2`) {
		t.Errorf("the column must hold every feature and probe: %s", page[strings.Index(page, `"buckets"`):])
	}
	if !strings.Contains(page, `"scores":[{"graded":2,"conforming":1}]`) {
		t.Error("the client must keep its row and its score in that column")
	}
}

// The version a page names is the corpus in the checkout — the present — and
// not whichever lane stands first. A single report left behind by a corpus
// bump must not label the lanes that are current, and no label may move
// because column order did.
func TestIndexNamesTheCorpusOnDisk(t *testing.T) {
	root := t.TempDir()
	entry := report.ProbeEntry{ID: "get", Class: "behavioral", State: "supported", Spec: []string{"request-line#target"}}
	stale := fakeReport("ijhttp", entry)
	stale.Corpus.Version = "2026-draft.1"
	current := fakeReport("httpyac", entry)
	current.Corpus.Version = "2026-draft.3"
	writeReport(t, root, "report-ijhttp.json", stale)
	writeReport(t, root, "report-httpyac.json", current)
	reg := &Registry{Lanes: []string{"ijhttp", "httpyac"}, Clients: map[string]ClientDoc{
		"ijhttp": {ID: "ijhttp", Name: "ijhttp"}, "httpyac": {ID: "httpyac", Name: "httpyac"}}}
	files, err := LoadReports(root, reg)
	if err != nil {
		t.Fatal(err)
	}

	opts := IndexOptions{Clients: reg.Clients, Corpus: &report.CorpusDocs{Version: "2026-draft.3"}}
	b, err := IndexHTML(files, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"version":"2026-draft.3"`) {
		t.Error("the page must name the corpus in the checkout, not the first lane's")
	}

	// drawn without a checkout there is nothing else to read, so the reports
	// answer for themselves
	opts.Corpus = nil
	b, err = IndexHTML(files, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"version":"2026-draft.1"`) {
		t.Error("without a corpus the reports are the only source for the version")
	}
}
