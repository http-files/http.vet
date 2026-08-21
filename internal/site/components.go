package site

import (
	"embed"
	"io/fs"
)

// The site's page-local component sets. A page in this ecosystem is a template
// with `{{component "name"}}` holes, filled by report.Page from a component's
// `<name>/<name>.{html,css,js}` — the shared set in the kit, and the page's own
// `Local` set searched before it. What both pages need is the kit's: the
// evidence renderers, the verdict fold, the panel frame, the release head.
// What one page needs is here.
//
// Each page gets its own root, so `components/<page>/` is exactly that page's
// set. Both pages carry a masthead and neither could ever share one — the
// matrix's says which corpus is being read, the index's says how current the
// site is — so the name is free to be the same on both. Promoting a component
// to both pages would mean moving it to the kit's shared set, which is where a
// thing two pages agree on belongs.

//go:embed components
var componentFS embed.FS

// pageComponents is one page's Local set, rooted so a component's path under
// it is the same as under the shared set.
func pageComponents(page string) fs.FS {
	sub, err := fs.Sub(componentFS, "components/"+page)
	if err != nil {
		panic(err) // the directory is embedded above; this cannot fail at run time
	}
	return sub
}
