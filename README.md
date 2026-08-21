# http.vet

The results browser for the `.http` conformance corpus, published at
<https://http.vet>.

The kit ([httpvet](https://github.com/http-files/httpvet)) grades one client against the corpus and writes one self-contained
`report.json`. This repo is the other half: it collects the reports client
owners submit, derives every state afresh from each report's embedded evidence
with the kit's current check scripts, and renders the matrix and the root page
the site publishes.

The site never re-runs a client. A report is evidence — the bytes a client put
on the wire, its output, the files it wrote — and what it means is whatever
the check scripts on disk say at build time. When a script is wrong it gets
fixed, and the next build regrades every report.

## One report per lane

Intake is `reports/report-<driver-id>.json`, one file per lane. A resubmission
overwrites the last; git history is the archive. A lane's column exists if,
and only if, someone submitted it. A probe whose document or fixture changed
since a lane ran is the one thing that wants a re-run — the evidence then
answers a different question — and until then its cell is empty.

Conformance fractions count **graded** probes only — behavioral and reject
classes. An observe-class probe records what a client does where the format
says nothing, so it has no norm to conform to; observations are reported
beside the fraction, never folded into it.

## Layout

```
clients.toml            client registry — lane order, names, format-guide links
reports/                intake — report-<driver-id>.json, one per lane
matrix/index.html       the matrix (generated, committed)
index.html              the root page (generated, committed)
cmd/wwwvet/             the build tool
internal/site/          matrix, index, registry, intake
  pages/                per page: <page>.html markup, <page>.css layout
  components/<page>/    that page's blocks — <name>/<name>.{html,css,js}
docs/intake.md          intake and CI design
LICENSE.md LICENSES/    results CC0 1.0, everything else MIT
CONTRIBUTING.md         what a contribution is and the terms it lands under
CNAME .nojekyll         GitHub Pages configuration
.gitattributes          `* -text`: git never normalizes bytes here
```

Generated pages are committed on purpose: a published result can then be
diffed against the reports it claims to display. `.gitattributes` sets
`* -text` so git never normalizes line endings and turns that diff into noise.

## wwwvet

```
wwwvet build  --httpvet <kit-checkout> --spec <spec-repo>
wwwvet matrix --httpvet <kit-checkout> --spec <spec-repo> [-o <path>]
wwwvet index  --httpvet <kit-checkout> --spec <spec-repo> [-o <path>]
```

| Verb | Does |
|---|---|
| `build` | Regrades every submitted report, then renders `matrix/index.html` and `index.html`. |
| `matrix` | The matrix alone. |
| `index` | The root page alone. |

`--root <dir>` is the site repository root, default `.`.

`--httpvet` points at a kit checkout and supplies `corpus/` — the check
scripts every state is derived with — and `drivers/`, the manifests. `--spec`
points at the spec repository and supplies the normative text each anchor
cites plus `releases/`, the year-named selections every page groups by. A
build without it still renders, quoting no normative prose and holding every
feature in one `outside any release` section that says the manifests are
missing — a local render, never a published one.

## Building locally

Go 1.26.5. The module carries
`replace github.com/http-files/httpvet => ../httpvet` and builds against a
side-by-side checkout: one directory then supplies both the code that
regrades a report and the check scripts it is regraded with.

```sh
git clone https://github.com/http-files/http.vet.git
cd http.vet
# beside it: the kit and the spec
go test ./...
go run ./cmd/wwwvet build --httpvet ../httpvet --spec ../spec
```

That writes `matrix/index.html` and `index.html`. Commit both with whatever
change produced them.

Pushing `main` deploys the live site — Pages serves this repository's root.

## Submitting a report

A client owner runs the kit themselves and PRs one file (`docs/intake.md`).

1. Check the kit out at `main`, `go build ./cmd/httpvet`, and put the client
   binary on `PATH`.
2. Run the whole corpus — no `--probe`:

   ```sh
   ./httpvet run <id>
   ```

   The report must carry every probe in the corpus. A restricted run writes
   `report-<id>.partial.json`, which the site does not take; renamed, it would
   publish with cells missing, which is not a conformance run.
3. Read the `errored` probes before submitting. `errored: nonzero exit … no
   reject pattern matched` is not a client failure: it says the lane's
   `[classify] reject_patterns` have never seen this release's refusal
   messages. Refine them in the kit's `drivers/<id>.toml` — the kit README's
   driver-manifest section covers the calibration — and re-run.
4. PR exactly `reports/report-<id>.json`, and nothing else. A maintainer
   runs `wwwvet build` and commits the regenerated pages before the merge
   lands; under CI the workflow does it on the branch instead.

Intake refuses a file whose name disagrees with the driver id embedded in the
report, and one from a lane that is not in the registry. Reports are
regenerated, never hand-edited: the site derives every state from the evidence
a report embeds, so an edited state is simply overwritten at the next build.

## Adding a lane

A lane starts as an issue on the kit —
<https://github.com/http-files/httpvet/issues> — naming the client, the
command that runs an `.http` file headlessly, and the page where it documents
the format.

Admission is decided in the kit, on three gates in order:

1. **The tool must read the `.http` format.** A tool whose document language
   is its own is out of scope however it spells its files: grading it measures
   the distance between two grammars rather than conformance. A low score is
   never itself a reason to refuse or drop a lane.
2. **The client runs a file headlessly.** A client that executes documents only
   through an editor UI is out by that capability criterion, not by name,
   pending a manual submission route.
3. Past those two, admission is uniform: actively maintained, and either in
   wide use or bringing an engine or transport stack no column already covers.

A lane is a driver manifest, `httpvet/drivers/<id>.toml`: `id`, `argv` as an
array (never a shell string; `{probe}`, `{workdir}`, `{probe_rel}` are
substituted), any `env` the client needs past the harness allowlist,
`timeout_ms`, a `[version]` argv and regex, and `[classify] reject_patterns`.
Flags may configure transport and output shape; a flag that changes document
semantics does not belong in a manifest.

Once the manifest is merged into the kit, one PR here carries the rest: a
`[client.<id>]` registry entry, the lane's position in `lanes`, and the first
`reports/report-<id>.json`.

## The registry

`clients.toml` answers what a graded client is, for display only. Nothing in it
reaches the recompute path.

- `lanes` is the canonical column order — adoption order, oldest first. It is
  data because intake is a directory rather than an argument list.
- `lanes` and the `[client.<id>]` entries must be a bijection. A lane in one
  half and not the other would lose its column or arrive unnamed, so loading
  refuses it.
- `name` is required: the client's own name for itself.
- `docs` is that client's guide to the `.http` **language** — how it documents
  the format the column grades it on — not an install page or a CLI reference.
  It must be `https` and should be a terminal URL: fetched, 200, no redirect
  hop. Where a client publishes no such guide the field is absent, which
  records the absence rather than leaving the lane looking overlooked.
- Every lane is listed on the same terms, including our own.

The registry is deliberately not part of the kit's `drivers/*.toml`: a manifest
is what a lane runs under, and what a client is called and where it documents
the format is display, editable here without touching a run.

## Determinism

Pages are a pure function of the committed reports and the kit checkout
beside them. `generated` is stamped from
the newest run date among the inputs, never the clock, and nothing else is read
from the environment. `wwwvet build` run twice produces byte-identical output.

That is what makes a published page checkable: re-render and diff it
against what is committed, and the page provably displays the reports beside
it. Any change that makes output depend on when or where it ran breaks the
check, and is a defect.

## License

Results are [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/), the
license compatibility data uses; everything else is
[MIT](https://opensource.org/licenses/MIT). See [LICENSE.md](LICENSE.md) for
what a generated page carries and [CONTRIBUTING.md](CONTRIBUTING.md) for the
terms a submission lands under.

## Intake and CI

`docs/intake.md` describes the intake and CI design. No CI is enabled — every
build here currently runs locally.
