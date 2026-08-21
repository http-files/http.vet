module github.com/http-files/http.vet

go 1.26.5

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/http-files/httpvet v0.0.0
)

require (
	go.starlark.net v0.0.0-20260708150628-5395d018f003 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

// The site builds against a kit checkout beside it: the same directory
// supplies the code that recomputes a report and the corpus that report is
// pinned to, so they can never drift apart.
replace github.com/http-files/httpvet => ../httpvet
