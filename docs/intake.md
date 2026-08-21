# Intake and CI

**Status: designed, not enabled.** There are no workflows in this repository.
Every build and page render currently runs locally, by hand. This document
describes a mechanism that does not run yet.

## What a submission is

One file: `reports/report-<driver-id>.json`, produced by the kit at its
`main`. Nothing else belongs in the PR — the generated pages are rebuilt from
the report, not submitted with it. A resubmission overwrites the last; git
history is the archive.

The loader refuses a file whose name disagrees with the driver id embedded in
the report, and one from a lane with no registry entry. Those refusals hold
today; CI only makes them happen on a pull request instead of on a laptop.

## PR workflow

Triggered by a pull request touching `reports/**` or `clients.toml`.

1. **Linux runner.** The corpus deliberately contains probes whose line
   endings, trailing bytes and BOMs are the test material — the kit marks
   `corpus/** -text` to keep git out of them — and this repository's
   `.gitattributes` sets `* -text` for the same reason on its own side: the
   reproducibility check below is a byte diff, and a normalized checkout fails
   it with no input having changed. Linux runners check out bytes as
   committed; Windows runners do not.
2. **Check out the kit at `main`**, and the spec at the commit the kit's
   `corpus/SPEC_PIN` names.
3. `go test ./...`
4. `go run ./cmd/wwwvet build --httpvet <kit> --spec <spec>` — derive every
   state from each submitted report's embedded evidence with the kit's current
   check scripts, then render the matrix and the root page.
5. **Commit the regenerated HTML back to the PR branch**, so the PR carries
   both the report and the pages it produces, and the merge is atomic.
6. **Comment an errored summary** per submitted report: counts per state, and
   the probe ids whose reason is `no reject pattern matched`. Information for
   the submitter and for triage, never a gate.

Step 4 is the substance. `report.Regrade` reads each report's evidence through
the same derivation the kit ran, with whatever scripts are on disk; the states
a report carries are never published, only the reading. The site does not
re-run any client, in CI or anywhere else.

## Main-branch rebuild

On every push to `main`: check out the kit at `main`, re-render, and
`git diff --exit-code`. A clean diff asserts that the published pages are
exactly what the committed reports and the current scripts render to. That
assertion is available because rendering is deterministic — `generated` is
stamped from the newest run date among the reports, never the clock, and
nothing else is read from the environment.

Because the page regrades with the kit's current scripts, a kit change can
legitimately move the committed page. The job then opens a rebuild PR rather
than failing.

## Accepting a submission

A gate merges a submission when all four hold: the diff is exactly one file,
`reports/report-<id>.json`, added or replaced; the report carries no
`selection` field — a whole-corpus run; `<id>` is in `clients.toml` `lanes`;
the author is listed in `submitters.toml`, which names the project's own
worker identity and the maintainers. Otherwise the PR is labelled
`needs-review` and waits for a person.

## What a reviewer asserts

The build derives every state, so a reviewer is not asserting the numbers and
should not be reading them. What merging asserts is everything the regrade
cannot see:

- **The lane is admitted.** A driver manifest for this id exists in the kit.
  Admission is decided in the kit repository, on the gate that the tool reads
  the `.http` format.
- **The report is the client it claims to be.** Regrading proves the states
  follow from the embedded evidence; it cannot prove the evidence came from an
  unmodified build of the named client at the recorded version. That is a
  judgement about the submitter and the run, and it is the reviewer's.
- **The registry entry is accurate** — the client's own name, and a link to its
  guide to the `.http` language rather than to an install page.
- **The submission is the whole lane.** A report covering a subset of the
  corpus, or one produced with driver flags that change document semantics, is
  not a conformance run.

## Automated and manual results

Every report published here is regraded by the same scripts, so all published
results stand on the same footing. There is no second-class column today and
no label distinguishing one.

Reports carry a `mode` field, `automated` or `manual`. The kit writes
`automated` for every run it produces; `manual` is reserved for a future
submission route for clients with no headless file interface, where a human
drives the client through the corpus and records what came back. Such a result
would be evidence of a different kind — a person's transcription rather than a
captured process — and the matrix would label the column as such rather than
mixing it silently into a page whose other columns were derived from captured
wire bytes. Nothing produces `manual` yet.
