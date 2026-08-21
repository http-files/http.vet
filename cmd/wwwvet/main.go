// wwwvet builds http.vet — the results browser for the .http conformance
// corpus.
//
// The kit (httpvet) grades one client and writes one report.json. This tool is
// the other half: it takes the reports client owners submit, derives every
// state afresh from each report's own embedded evidence with the kit's current
// check scripts, and renders the matrix and the root page the site publishes.
//
// Verbs:
//
//	wwwvet build  --httpvet ../httpvet --spec ../spec   (matrix + index)
//	wwwvet matrix --httpvet ../httpvet --spec ../spec [-o matrix/index.html]
//	wwwvet index  --httpvet ../httpvet --spec ../spec [-o index.html]
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/http-files/http.vet/internal/site"
	"github.com/http-files/httpvet/report"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "matrix":
		err = cmdMatrix(os.Args[2:])
	case "index":
		err = cmdIndex(os.Args[2:])
	case "build":
		err = cmdBuild(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "wwwvet:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(`
usage:
  wwwvet build  --httpvet <kit-checkout> --spec <spec-repo>
  wwwvet matrix --httpvet <kit-checkout> --spec <spec-repo> [-o <path>]
  wwwvet index  --httpvet <kit-checkout> --spec <spec-repo> [-o <path>]

common flags:
  --root <dir>      site repository root (default ".")
  --httpvet <dir>   kit checkout: corpus/ supplies the check scripts every
                    state is derived with, drivers/ the manifests (required)
  --spec <dir>      spec repository: normative text, and releases/ for the
                    release sections. Without it every feature renders under
                    one section saying the manifests are missing.
`))
}

// ctx is the resolved working set every verb starts from.
type ctx struct {
	root string
	kit  string
	spec string
	reg  *site.Registry
}

func setup(fs *flag.FlagSet, args []string) (*ctx, error) {
	root := fs.String("root", ".", "site repository root")
	kit := fs.String("httpvet", "", "httpvet checkout supplying corpus/ and drivers/ (required)")
	spec := fs.String("spec", "", "spec repository — normative text and release manifests")
	fs.Parse(args)
	if *kit == "" {
		return nil, fmt.Errorf("%s requires --httpvet", fs.Name())
	}
	reg, err := site.LoadRegistry(filepath.Join(*root, "clients.toml"))
	if err != nil {
		return nil, err
	}
	return &ctx{root: *root, kit: *kit, spec: *spec, reg: reg}, nil
}

// regraded loads every submitted report and derives its states afresh from
// its embedded evidence, with the corpus and driver manifest in the kit
// checkout. The site never re-runs a client and never takes a submitted state
// on trust; what it publishes is what the current scripts read in the
// evidence. A lane must have a manifest in the kit — admission is decided
// there. A report that covers less than the corpus is published with the
// cells it has and named on stderr.
func regraded(c *ctx) ([]site.ReportFile, error) {
	files, err := site.LoadReports(c.root, c.reg)
	if err != nil {
		return nil, err
	}
	corpusDir := filepath.Join(c.kit, "corpus")
	corpus, err := report.LoadCorpusDocs(corpusDir)
	if err != nil {
		return nil, err
	}
	for i := range files {
		manifest := filepath.Join(c.kit, "drivers", files[i].ID+".toml")
		if _, err := os.Stat(manifest); err != nil {
			return nil, fmt.Errorf("%s: no driver manifest %s in the kit checkout — a lane must be admitted to the kit before its reports are published here", files[i].Path, manifest)
		}
		rep, err := report.RegradeFile(files[i].Report, corpusDir, manifest)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", files[i].Path, err)
		}
		if note := missingProbes(corpus, rep); note != "" {
			fmt.Fprintf(os.Stderr, "· %s: %s\n", files[i].ID, note)
		}
		files[i].Report = rep
	}
	return files, nil
}

// missingProbes names the corpus probes a report does not answer.
func missingProbes(corpus *report.CorpusDocs, rep *report.Report) string {
	have := make(map[string]bool, len(rep.Probes))
	for _, pe := range rep.Probes {
		have[pe.ID] = true
	}
	var missing []string
	for id := range corpus.Probes {
		if !have[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	sort.Strings(missing)
	shown := missing
	if len(shown) > 6 {
		shown = shown[:6]
	}
	more := ""
	if len(missing) > len(shown) {
		more = ", …"
	}
	return fmt.Sprintf("%d of %d corpus probes not in the report (%s%s) — those cells are empty until the lane re-runs",
		len(missing), len(corpus.Probes), strings.Join(shown, ", "), more)
}

// loadReleases reads the release manifests the spec repository carries. A
// checkout without releases/ is not an error: the pages render every feature
// outside any release, which is what no manifest read means.
func loadReleases(spec string) ([]report.ReleaseManifest, error) {
	if spec == "" {
		return nil, nil
	}
	dir := filepath.Join(spec, "releases")
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return report.LoadReleases(dir)
}

func writeFile(out string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, b, 0o644)
}

func buildMatrix(c *ctx, files []site.ReportFile, out string) error {
	opts := site.MatrixOptions{Clients: c.reg.Clients}
	var err error
	if opts.Corpus, err = report.LoadCorpusDocs(filepath.Join(c.kit, "corpus")); err != nil {
		return err
	}
	if opts.Drivers, err = report.LoadDriverDocs(filepath.Join(c.kit, "drivers")); err != nil {
		return err
	}
	if c.spec != "" {
		if opts.Spec, err = report.LoadSpec(c.spec); err != nil {
			return err
		}
		if opts.Releases, err = loadReleases(c.spec); err != nil {
			return err
		}
	}
	b, err := site.MatrixHTML(site.Reports(files), opts)
	if err != nil {
		return err
	}
	if err := writeFile(out, b); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "matrix written to %s (%d lanes)\n", out, len(files))
	return nil
}

func buildIndex(c *ctx, files []site.ReportFile, out string) error {
	opts := site.IndexOptions{Clients: c.reg.Clients}
	var err error
	if opts.Corpus, err = report.LoadCorpusDocs(filepath.Join(c.kit, "corpus")); err != nil {
		return err
	}
	if opts.Releases, err = loadReleases(c.spec); err != nil {
		return err
	}
	b, err := site.IndexHTML(files, opts)
	if err != nil {
		return err
	}
	if err := writeFile(out, b); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "index written to %s (%d lanes)\n", out, len(files))
	return nil
}

func cmdMatrix(args []string) error {
	fs := flag.NewFlagSet("matrix", flag.ExitOnError)
	out := fs.String("o", "", "output path (default <root>/matrix/index.html)")
	c, err := setup(fs, args)
	if err != nil {
		return err
	}
	files, err := regraded(c)
	if err != nil {
		return err
	}
	path := *out
	if path == "" {
		path = filepath.Join(c.root, "matrix", "index.html")
	}
	return buildMatrix(c, files, path)
}

func cmdIndex(args []string) error {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	out := fs.String("o", "", "output path (default <root>/index.html)")
	c, err := setup(fs, args)
	if err != nil {
		return err
	}
	files, err := regraded(c)
	if err != nil && !errors.Is(err, site.ErrNoReports) {
		return err
	}
	path := *out
	if path == "" {
		path = filepath.Join(c.root, "index.html")
	}
	return buildIndex(c, files, path)
}

// cmdBuild is the whole site step: regrade every submitted report, render the
// matrix and the root page. No submissions is not a failure — the root page
// still renders and says so; only a matrix over nothing is skipped.
func cmdBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	c, err := setup(fs, args)
	if err != nil {
		return err
	}
	files, err := regraded(c)
	if err != nil && !errors.Is(err, site.ErrNoReports) {
		return err
	}
	if len(files) > 0 {
		if err := buildMatrix(c, files, filepath.Join(c.root, "matrix", "index.html")); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(os.Stderr, "no reports submitted yet — no matrix to build")
	}
	return buildIndex(c, files, filepath.Join(c.root, "index.html"))
}
