// versions: a row per client against a column per release. Nothing here
// opens, filters or folds — the whole page is visible at once, and a reader
// who wants a probe follows the link to the matrix.
(() => {
  const root = document.querySelector('[data-c="versions"]');

  // the bar itself is the shared layer's; the column it stands in is this page's
  const scoreCell = (s, rest) =>
    h("td", rest ? "col-rest" : "", [score(s ? s.conforming : 0, s ? s.graded : 0)]);

  function clientRow(c, buckets) {
    const name = c.name || c.id;
    const label = c.docs
      ? h("a", {href: c.docs, text: name, rel: "noopener", target: "_blank"})
      : textNode(name);
    const cells = [h("td", null, [label, c.version
      ? h("span", {class: "cv", text: "  " + c.version})
      : h("span", {class: "nover", text: "  —"})])];
    buckets.forEach((b, i) => cells.push(scoreCell(c.scores && c.scores[i], !b.year)));
    return h("tr", null, cells);
  }

  // A column is a release section stood on its side, so its head is the shared
  // one — the same words, in the same order, as the section it links to. What
  // this page adds under it is the column's own arithmetic.
  function bucketHead(b, lanes) {
    const th = releaseHead(h("th", b.year ? "" : "col-rest", []), b);
    th.append(h("span", {class: "b-sub",
      text: b.features + (b.features === 1 ? " feature · " : " features · ") + b.probes + " probes"}));
    if (b.probes) {
      th.append(h("span", {class: "b-conform",
        text: b.conform + " of " + lanes + (b.year ? " conform" : " clear it")}));
    }
    return th;
  }

  function versionBlock(v) {
    const clients = v.clients || [];
    const buckets = v.buckets || [];

    const head = h("div", "vhead", [
      h("a", {class: "name", href: "matrix/", text: v.version || "corpus"}),
      h("span", {class: "grow pin", text:
        v.probes + " probes · " + clients.length + " clients · kit " + v.kit}),
      h("a", {href: "matrix/", text: "full matrix →"}),
    ]);

    let body;
    if (!clients.length) {
      body = h("div", {class: "empty", text: "no reports submitted yet"});
    } else {
      body = h("div", "tablewrap", [h("table", null, [
        h("thead", null, [h("tr", null,
          [h("th", {class: "who", text: "client"})].concat(
            buckets.map(b => bucketHead(b, clients.length))))]),
        h("tbody", null, clients.map(c => clientRow(c, buckets))),
      ])]);
    }
    return h("section", "v", [head, body]);
  }

  for (const v of DATA.versions || []) root.append(versionBlock(v));
})();
