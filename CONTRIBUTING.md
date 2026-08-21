# Contributing

http.vet publishes conformance results for the `.http` file format. Every
implementation is measured the same way and listed on the same terms, including
the maintainers' own (`req`). Results describe what a client did with a
document; they do not rank, recommend, or market. If you maintain a client, you
are the authority on how it is named and where its format guide points.

## What a contribution is

| Kind | What it touches |
|---|---|
| A result | one `reports/report-<driver-id>.json`, produced by the kit |
| A registry entry | `clients.toml` — your client's name, its format-guide link, its lane position |
| A new lane | a driver manifest in the kit first, then the two above |
| Site code or docs | `cmd/`, `internal/`, `README.md`, `docs/` |

Reports are regenerated, never edited. The site derives every state from the
evidence a report embeds, with the kit's current check scripts, so a
hand-edited state is simply overwritten at the next build. The generated pages
are rebuilt from the reports; do not hand-edit those either.

## Licensing

Results — `reports/` and `clients.toml` — are **CC0 1.0**; everything
else is **MIT**. See [LICENSE.md](LICENSE.md).

By opening a pull request you agree that your contribution is provided under
the matching license and that you have the right to contribute the material.

**On submitting a report specifically.** A report embeds what your client
printed and what it put on the wire, captured verbatim. Placing it under CC0 is
a dedication of the measurement — the states, counts, and captured bytes — so
that a conformance result can be quoted, mirrored, and recomputed by anyone
without permission. Submit a report for a client you maintain, or one you are
otherwise free to publish a recording of.

## Neutrality

- Technical, factual, no marketing language, no rankings.
- A low score is never a reason to remove a lane, and a high one earns no
  placement. Column order is adoption order, recorded in `clients.toml`.
- A lane must read the `.http` format. A tool whose document language is its
  own is out of scope however it spells its files — grading it would measure
  the distance between two grammars rather than conformance.
