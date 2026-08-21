package site

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/http-files/httpvet/report"
)

// ErrNoReports distinguishes "nothing submitted yet" from a real failure.
// Building a matrix out of nothing is an error; rendering the root page with
// no lanes on it is not.
var ErrNoReports = errors.New("no reports submitted")

// Submitted reports live at reports/report-<driver-id>.json, one per lane. A
// resubmission overwrites the last; git history is the archive. The filename
// is not decoration: it must agree with the driver id embedded in the report,
// so a submission cannot land in another client's column by being named as
// one.

const reportPrefix = "report-"

// ReportFile is one submitted report with the path it came from.
type ReportFile struct {
	Path   string
	ID     string
	Report *report.Report
}

// ReportsDir is the intake directory.
func ReportsDir(root string) string {
	return filepath.Join(root, "reports")
}

// LoadReports reads every submitted report, in registry lane order. A file
// whose name disagrees with its embedded driver id, or whose lane is
// unregistered, is refused rather than rendered.
func LoadReports(root string, reg *Registry) ([]ReportFile, error) {
	dir := ReportsDir(root)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", dir, ErrNoReports)
	}
	if err != nil {
		return nil, err
	}
	byID := map[string]ReportFile{}
	have := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, reportPrefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(strings.TrimPrefix(name, reportPrefix), ".json")
		path := filepath.Join(dir, name)
		rep, err := report.Load(path)
		if err != nil {
			return nil, err
		}
		if rep.Driver.ID != id {
			return nil, fmt.Errorf("%s: report is for driver %q — a submission must be named report-<driver-id>.json", path, rep.Driver.ID)
		}
		byID[id] = ReportFile{Path: path, ID: id, Report: rep}
		have[id] = true
	}
	if len(byID) == 0 {
		return nil, fmt.Errorf("%s holds no report-<id>.json: %w", dir, ErrNoReports)
	}
	order, err := reg.Ordered(have)
	if err != nil {
		return nil, err
	}
	out := make([]ReportFile, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

// Reports strips the file wrapper for the renderers, preserving order.
func Reports(files []ReportFile) []*report.Report {
	out := make([]*report.Report, len(files))
	for i, f := range files {
		out[i] = f.Report
	}
	return out
}
