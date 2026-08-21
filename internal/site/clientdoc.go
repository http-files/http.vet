package site

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// The client registry answers what a graded client *is* — its own name for
// itself, and its guide to the language a column grades it on — and in what
// order the columns stand. That is display metadata, not lane configuration,
// and it deliberately does not live in the kit's drivers/*.toml: a manifest is
// hashed whole and that hash is pinned into every report it produced, so a URL
// recorded there would mark every lane changed-since-its-run without changing
// a single graded state. Kept here, the registry is editable at any time and
// every pin survives.

// ClientDoc is one client's display metadata, keyed by driver id.
type ClientDoc struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Docs string `json:"docs,omitempty"`
}

// Registry is the parsed client registry: the canonical lane order plus the
// per-client metadata. Lane order is adoption order, oldest first — the kit
// took it from the argument list, which a directory of submitted reports has
// no equivalent of, so here it is written down.
type Registry struct {
	Lanes   []string
	Clients map[string]ClientDoc
}

// Ordered returns the registered lanes that appear in have, in registry
// order, and reports any id that is not registered. An unregistered report is
// an error rather than an appended column: the order is the point, and a
// column with no name or documentation link is a column a reader cannot
// interpret.
func (r *Registry) Ordered(have map[string]bool) ([]string, error) {
	var unknown []string
	for id := range have {
		if _, ok := r.Clients[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	// every offender at once, in a fixed order: `have` is a map, so naming one
	// of several would name a different one each run
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("not in the registry: %s — add a [client.<id>] entry for each (and place each in `lanes`) before submitting its report",
			strings.Join(unknown, ", "))
	}
	var out []string
	for _, id := range r.Lanes {
		if have[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// LoadRegistry reads the client registry. Every lane listed in `lanes` must
// have an entry and every entry must be listed — the two halves are one
// statement, and a mismatch means a column would silently vanish or arrive
// unnamed.
func LoadRegistry(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file struct {
		Lanes  []string `toml:"lanes"`
		Client map[string]struct {
			Name string `toml:"name"`
			Docs string `toml:"docs"`
		} `toml:"client"`
	}
	if err := toml.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	reg := &Registry{Lanes: file.Lanes, Clients: map[string]ClientDoc{}}
	ids := make([]string, 0, len(file.Client))
	for id := range file.Client {
		ids = append(ids, id)
	}
	sort.Strings(ids) // a file with two mistakes must report the same one twice
	for _, id := range ids {
		e := file.Client[id]
		if e.Name == "" {
			return nil, fmt.Errorf("%s: client %q: name is required", path, id)
		}
		// the page links out to the open web; anything but https is a mistake
		// worth failing on rather than shipping as a bad column head
		if e.Docs != "" && !strings.HasPrefix(e.Docs, "https://") {
			return nil, fmt.Errorf("%s: client %q: docs must be an https URL, got %q", path, id, e.Docs)
		}
		reg.Clients[id] = ClientDoc{ID: id, Name: e.Name, Docs: e.Docs}
	}
	if len(reg.Clients) == 0 {
		return nil, fmt.Errorf("%s: no [client.<id>] entries", path)
	}
	seen := map[string]bool{}
	for _, id := range reg.Lanes {
		if seen[id] {
			return nil, fmt.Errorf("%s: lane %q listed twice", path, id)
		}
		seen[id] = true
		if _, ok := reg.Clients[id]; !ok {
			return nil, fmt.Errorf("%s: lane %q has no [client.%s] entry", path, id, id)
		}
	}
	for _, id := range ids {
		if !seen[id] {
			return nil, fmt.Errorf("%s: client %q is not listed in `lanes`, so it has no column position", path, id)
		}
	}
	return reg, nil
}
