// grid: probes × clients, the evidence one click deep. Every result renders
// through the shared evidence layer; what is here is the grid those results
// stand in — release sections, the fold from probe to feature to release, the
// observe partition, and the walk over the whole thing.
//
// Selection is this component's, not the shared one's. `selection` splices a
// panel directly after the row that owns it, which is what a flow of rows
// wants; here a row's cells are `td`s inside it, so a panel placed after a
// cell would open inside a 104px column. The matrix opens a `tr` spanning
// every column instead, and walks in two dimensions because it is a grid and
// not a list. Everything else — the panel frame, the evidence renderers, the
// verdict fold, the addressable fragment — is the shared layer's.
(() => {
  const root = document.querySelector('[data-c="grid"]');
  const table = root.querySelector("table");

  bind({
    close: () => closeDetail(true),
    // a spec cross-reference is a button only where the feature it names has a
    // row to jump to; the matrix has no other landing place
    xref: label => featureRow(label) ? () => jumpToFeature(label) : null,
  });

  // ── the rows: the shared grouping, with probes resolved ────────────────
  // The order — release, then feature, then probe — arrives decided. All this
  // adds is the probe each feature names, so the rest of the page works on
  // probes rather than on ids.

  const PROBE = new Map((DATA.probes || []).map(p => [p.id, p]));
  const GROUPS = (DATA.groups || []).map(g => ({...g, features: g.features.map(f =>
    ({name: f.name, probes: f.probes.map(id => PROBE.get(id)).filter(Boolean)}))}));

  // ── the folds this page adds: probe → feature → release ────────────────

  function featureFold(area, i) {
    return worstFold(area.probes.map(p => p.cells[i] ? cellFold(p, p.cells[i]) : null).filter(Boolean));
  }

  function releaseSummary(features, i) {
    const folds = features.map(f => featureFold(f, i)).filter(Boolean);
    if (!folds.length) return {word: "no data", v: "mut"};
    if (folds.some(f => f === "deviating" || f === "errored" || f === "unsupported"))
      return {word: "does not conform", v: "bad"};
    const quirked = folds.filter(f => f === "quirked").length;
    if (quirked) return {word: "conforms · " + quirked + " quirked", v: "warn"};
    return {word: "conforms", v: "ok"};
  }

  // ── the observe partition ──────────────────────────────────────────────
  // An observe row's subject is how its lanes split. Lanes that read the same
  // share a colour; the value behind a colour is named in the panel, never on
  // the row. A lane with no reading at all (it refused the document, or the
  // run errored) is a case too, and takes no colour rather than a misleading
  // one.

  const obsReading = cell =>
    cell.state === "observed" && cell.observations && cell.observations.length
      ? cell.observations.map(obsWord).join(" · ") : null;

  function observeCases(p) {
    if (p.class !== "observe") return null;
    const cases = [], seen = new Map();
    let coloured = 0;
    p.cells.forEach((cell, i) => {
      if (!cell) return;
      const reading = obsReading(cell);
      const key = reading === null ? "\u0000" + cell.state : reading;
      if (!seen.has(key)) {
        seen.set(key, cases.length);
        cases.push({reading: reading, state: cell.state, lanes: [],
                    g: reading === null ? -1 : coloured++});
      }
      cases[seen.get(key)].lanes.push(i);
    });
    return cases;
  }

  // lane index -> its case, memoised per probe
  const caseCache = new Map();
  function caseOf(p, i) {
    if (!caseCache.has(p.id)) caseCache.set(p.id, observeCases(p));
    const cases = caseCache.get(p.id);
    if (!cases) return null;
    return cases.find(c => c.lanes.includes(i)) || null;
  }

  // the cell word: the state, except an observed cell shows its dynamic values
  function cellWord(p, cell) {
    if (p.class === "observe" && cell.state === "observed" && cell.observations && cell.observations.length)
      return cell.observations.map(obsWord).join(" · ");
    return cell.state;
  }

  // The readings this row split into, and which lanes gave each. It sits in
  // the panel rather than on the matrix: the row shows the partition, this
  // names it.
  function casesBlock(cases, self) {
    const wrap = h("div", "cases", []);
    wrap.append(h("p", "x-label", [cases.length + " readings across " +
      cases.reduce((n, c) => n + c.lanes.length, 0) + " lanes"]));
    for (const c of cases) {
      const mine = c.lanes.includes(self);
      const row = h("div", "crow" + (mine ? " me" : "") + (c.g < 0 ? " none" : ""), [
        h("span", "cdot", []),
        h("span", "cval", [c.reading === null ? c.state + " — no reading" : c.reading]),
        h("span", "clanes", [c.lanes.map(i => DATA.clients[i].id).join(" ")]),
      ]);
      if (c.g >= 0) row.style.setProperty("--g", "var(--g" + (c.g % 8) + ")");
      wrap.append(row);
    }
    return wrap;
  }

  // ── panels ─────────────────────────────────────────────────────────────

  // A cell panel answers one question: why is this cell that colour. The head
  // carries the answer; under it sits only the evidence that produced this
  // result. Nothing hides behind a click; what does not bear on this cell is
  // simply not in the panel.
  function cellPanel(p, i) {
    const cell = p.cells[i], client = DATA.clients[i];
    const v = verdict(p.class, cell);
    const d = probeDoc(p.id);
    const panel = panelShell(v, [p.id, h("span", "at", [" @ "]), client.id, " — ",
      h("span", "st", [cell.state])]);
    if (d && d.title) panel.append(h("p", "p-title", [d.title]));

    const fails = cell.checks.filter(c => !c.pass);
    if (fails.length) panel.append(h("div", "answer", fails.map(failRow)));
    const line = stateLine(p, cell);
    if (line) panel.append(h("p", "p-state", [line]));
    const held = heldLine(cell);
    if (held) panel.append(held);
    const cases = observeCases(p);
    if (cases && cases.length > 1) panel.append(casesBlock(cases, i));

    // what the probe is — its document, its statements, its check — is the
    // same for every column, so it has a panel of its own and this one carries
    // only what this lane did
    panel.append(evidenceSections(p, cell, client));
    return panel;
  }

  // the probe's identity: what it feeds the client, what the spec demands, how
  // the wire is graded — all of it, inline, with no lane in the question
  function probePanel(p) {
    const d = probeDoc(p.id);
    const panel = panelShell("mut", [p.id]);
    if (d && d.title) panel.append(h("p", "p-title", [d.title]));
    const bits = [p.class + "-class", (p.spec || []).join(" · ")];
    if (d && d.channels) bits.push(d.channels.join("+") + " · responder " + d.responder);
    panel.append(h("p", "p-state", [bits.filter(Boolean).join(" · ")]));
    const ev = h("div", "evidence", [sectionDoc(p, null, null), sectionSpec(p, null, null)]);
    if (d) ev.append(sectionCheck(p, null, null));
    panel.append(ev);
    return panel;
  }

  // A fold is worth opening only to see which parts produced it.
  function rollupPanel(v, headParts, rows) {
    const panel = panelShell(v, headParts);
    panel.append(h("table", "rollup", rows));
    return panel;
  }

  function featurePanel(area, i) {
    const client = DATA.clients[i];
    const fold = featureFold(area, i);
    const v = fold ? foldVerdict(fold) : "mut";
    const rows = area.probes.map(p => {
      const cell = p.cells[i];
      const b = h("button", {class: "linkish", type: "button"}, [p.id]);
      b.addEventListener("click", () => openCell(p.id, i));
      const tr = h("tr", "", [h("td", "pid", [b])]);
      if (!cell) { tr.append(h("td", "", ["—"]), h("td", "why", [])); return tr; }
      tr.append(h("td", "", [h("span", "cell v-" + verdict(p.class, cell), [cell.state])]));
      tr.append(h("td", "why", [why(p, cell, area.name)]));
      return tr;
    });
    const panel = rollupPanel(v, [area.name, h("span", "at", [" @ "]), client.id, " — ",
      h("span", "st", [fold || "no data"])], rows);
    if (hasSpec()) {
      const covered = new Set();
      for (const p of area.probes) for (const a of p.spec || []) covered.add(a);
      const uncovered = specOrder().filter(a => a.startsWith(area.name + "#") && !covered.has(a));
      if (uncovered.length) panel.append(h("p", "dim",
        ["no probe: " + uncovered.map(a => shortAnchor(a, area.name)).join(" ")]));
    }
    return panel;
  }

  function releasePanel(g, feats, i) {
    const client = DATA.clients[i];
    const s = releaseSummary(feats, i);
    const rows = feats.map(f => {
      const fold = featureFold(f, i);
      const b = h("button", {class: "linkish", type: "button"}, [f.name]);
      b.addEventListener("click", () => jumpToFeature(f.name));
      return h("tr", "", [
        h("td", "pid", [b]),
        h("td", "", [fold ? h("span", "cell v-" + foldVerdict(fold), [fold]) : "—"]),
        h("td", "why", []),
      ]);
    });
    return rollupPanel(s.v, [g.label, h("span", "at", [" @ "]), client.id, " — ",
      h("span", "st", [s.word])], rows);
  }

  // ── selection: one open detail row, spliced under the row that owns it ──

  let open = null; // {key, btn, row}

  function closeDetail(restoreFocus) {
    if (!open) return;
    const row = root.querySelector("tr.detail");
    if (row) row.remove();
    if (open.btn) open.btn.classList.remove("selected");
    const btn = open.btn;
    open = null;
    setHash("");
    if (restoreFocus && btn) btn.focus();
  }

  function showDetail(anchorRow, btn, panel, key) {
    const wasOpen = open;
    closeDetail(false);
    if (wasOpen && wasOpen.key === key) return; // clicking the open cell closes it
    const row = h("tr", "detail", [h("td", {colspan: DATA.clients.length + 1}, [panel])]);
    anchorRow.after(row);
    btn.classList.add("selected");
    btn.focus(); // WebKit leaves focus on BODY after a button click; the arrows read activeElement
    open = {key, btn, row};
    panelPlaced(panel, btn);
    if (key.startsWith("p:")) setHash(key.slice(2));
    else if (key.startsWith("c:")) {
      const at = key.lastIndexOf(":");
      setHash(key.slice(2, at) + "@" + DATA.clients[Number(key.slice(at + 1))].id);
    }
    // keep the row that owns the panel visible: the point of opening in place
    // is that you can still see which cell you clicked
    const headH = parseFloat(getComputedStyle(root).getPropertyValue("--head-h")) || 68;
    const ar = anchorRow.getBoundingClientRect();
    if (ar.top < headH + 8 || row.getBoundingClientRect().top > window.innerHeight - 140)
      anchorRow.scrollIntoView({block: "start"});
  }

  function openCell(probeId, i) {
    const btn = root.querySelector(`tr.probe[data-probe="${CSS.escape(probeId)}"] .cell[data-col="${i}"]`);
    if (btn) { btn.focus(); btn.click(); }
  }

  function featureRow(name) {
    return root.querySelector(`tr.feature[data-feature="${CSS.escape(name)}"]`);
  }

  function jumpToFeature(name) {
    const row = featureRow(name);
    if (!row) return;
    row.scrollIntoView({block: "center"});
    row.classList.add("flash");
    setTimeout(() => row.classList.remove("flash"), 900);
  }

  // ── the grid ───────────────────────────────────────────────────────────

  function renderMatrix() {
    // The grid's own width, published where the page can read it: the masthead
    // and the footer take the same width, and neither is inside this component.
    // It goes out resolved, in px, because the column widths are this
    // component's and stay declared on its root — a calc() naming them would
    // substitute to nothing on an element outside it.
    const cs = getComputedStyle(root);
    const col = name => parseFloat(cs.getPropertyValue(name)); // px, as grid.css declares them
    document.documentElement.style.setProperty("--matrix-w",
      col("--probe-col") + DATA.clients.length * col("--client-col") + "px");
    const cg = h("colgroup", "", [h("col", "probe", [])]);
    for (const c of DATA.clients) cg.append(h("col", "client", []));
    table.append(cg);

    const hrow = h("tr", "", [h("th", "", ["feature / probe"])]);
    for (const c of DATA.clients) {
      // the id opens the client's own guide to the format it is graded on,
      // when the registry names one; a lane with none — or a page rendered
      // without --clients — stays plain
      const cid = c.docs
        ? h("a", {class: "cid", href: c.docs, target: "_blank", rel: "noopener noreferrer",
                  title: (c.name ? c.name + " — " : "") + ".http guide · " + c.docs}, [c.id])
        : h("span", {class: "cid", title: c.name || null}, [c.id]);
      // version-less clients get a placeholder so run dates align across columns
      hrow.append(h("th", "", [cid, h("span", "cv", [c.version || "—"]), h("span", "cv", [c.runDate])]));
    }
    table.append(h("thead", "", [hrow]));

    const tbody = h("tbody", "", []);

    const featureBlock = (area) => {
      const frow = h("tr", {class: "feature", data: {feature: area.name}}, []);
      // a release selected it and the corpus does not reach it yet: the feature
      // still gets its row, saying that rather than going missing
      if (!area.probes.length) {
        frow.append(h("th", "", [area.name, h("span", "tag", ["no probes yet"])]));
        DATA.clients.forEach(() => frow.append(h("td", "absent", ["—"])));
        tbody.append(frow);
        return;
      }
      const fmeta = specFeature(area.name);
      frow.append(h("th", {title: fmeta && fmeta.title ? fmeta.title : null}, [area.name]));
      DATA.clients.forEach((c, i) => {
        const fold = featureFold(area, i);
        if (!fold) { frow.append(h("td", "", ["—"])); return; }
        const btn = h("button", {class: "cell v-" + foldVerdict(fold), type: "button", data: {col: i},
          title: area.name + " @ " + c.id + " — worst of " + area.probes.length + " probe(s)"}, [fold]);
        btn.addEventListener("click", () =>
          showDetail(frow, btn, featurePanel(area, i), "f:" + area.name + ":" + i));
        frow.append(h("td", "", [btn]));
      });
      tbody.append(frow);
      for (const p of area.probes) {
        const row = h("tr", {class: "probe", data: {probe: p.id}}, []);
        const pd = probeDoc(p.id);
        const idBtn = h("button", {class: "plabel", type: "button",
          title: pd && pd.title ? pd.title : "what this probe is"}, [p.id]);
        idBtn.addEventListener("click", () => showDetail(row, idBtn, probePanel(p), "p:" + p.id));
        const label = h("th", "sub", [idBtn,
          (p.class === "reject" || p.class === "observe") && h("span", "tag", [p.class])]);
        row.append(label);
        p.cells.forEach((cell, i) => {
          if (!cell) { row.append(h("td", "absent", ["—"])); return; }
          // an observe cell with no reading — the client refused the document —
          // is not one of the row's cases, and must not wear a case colour
          const kase = caseOf(p, i);
          const tint = kase && kase.g >= 0 ? " og" + (kase.g % 8) : "";
          const v = kase && kase.g < 0 && cell.state !== "errored" ? "mut" : verdict(p.class, cell);
          // observed values ellipsize hard in 104px, and comparing them across
          // clients is the whole point of an observe row — keep the full text
          const word = cellWord(p, cell);
          const btn = h("button", {class: "cell v-" + v + tint, type: "button", data: {col: i},
            title: p.id + " @ " + DATA.clients[i].id + " — " + cell.state +
              (word === cell.state ? "" : ": " + word)}, [word]);
          btn.addEventListener("click", () => showDetail(row, btn, cellPanel(p, i), "c:" + p.id + ":" + i));
          row.append(h("td", "", [btn]));
        });
        tbody.append(row);
      }
    };

    // A section lists what its release added; conformance to that release is
    // cumulative, since a year's spec includes every prior year's features. So
    // the fold carries features across the sections the page walks, while the
    // rows under a header stay the year's own.
    let carried = [];
    for (const g of GROUPS) {
      if (!g.features.length) continue; // nothing selected, nothing to say
      tbody.append(h("tr", "release", [releaseHead(h("th", {colspan: DATA.clients.length + 1}, []), g)]));
      if (g.year) {
        const feats = carried.concat(g.features);
        const srow = h("tr", "summary", [
          h("th", "", [carried.length ? "release conformance · incl. prior years" : "release conformance"]),
        ]);
        DATA.clients.forEach((c, i) => {
          const s = releaseSummary(feats, i);
          const btn = h("button", {class: "cell v-" + s.v, type: "button", data: {col: i},
            title: g.label + " @ " + c.id + " — " + s.word}, [s.word]); // the word ellipsizes in 104px
          btn.addEventListener("click", () =>
            showDetail(srow, btn, releasePanel(g, feats, i), "R:" + g.year + ":" + i));
          srow.append(h("td", "", [btn]));
        });
        tbody.append(srow);
        carried = feats;
      }
      for (const f of g.features) featureBlock(f);
    }
    table.append(tbody);

    // an id longer than the fixed column (dotnet-httpie) scales down
    // proportionally to fit — smaller font, same line box, never condensed
    for (const cid of table.querySelectorAll("thead .cid")) {
      const avail = cid.parentElement.clientWidth - 20; // th horizontal padding
      let w = cid.getBoundingClientRect().width;
      if (w <= avail) continue;
      const cs = getComputedStyle(cid);
      cid.style.lineHeight = cs.lineHeight; // pin: header baselines stay aligned
      let size = parseFloat(cs.fontSize) * avail / w;
      for (let i = 0; w > avail && i < 6; i++, size -= 0.1) {
        cid.style.fontSize = size.toFixed(2) + "px";
        w = cid.getBoundingClientRect().width;
      }
    }
    root.style.setProperty("--head-h",
      Math.ceil(table.querySelector("thead th").getBoundingClientRect().height) + "px");
  }

  // ── keyboard: the matrix is a grid, so arrows walk it ──────────────────

  function walk(e) {
    const btn = document.activeElement;
    if (!btn || !btn.classList || !btn.classList.contains("cell")) return false;
    const col = Number(btn.dataset.col);
    if (Number.isNaN(col)) return false;
    const rows = [...root.querySelectorAll("tr.probe, tr.feature")];
    const row = btn.closest("tr");
    const r = rows.indexOf(row);
    let target = null;
    if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
      const next = col + (e.key === "ArrowRight" ? 1 : -1);
      target = row.querySelector(`.cell[data-col="${next}"]`);
    } else if (e.key === "ArrowUp" || e.key === "ArrowDown") {
      for (let i = r + (e.key === "ArrowDown" ? 1 : -1); i >= 0 && i < rows.length;
           i += (e.key === "ArrowDown" ? 1 : -1)) {
        const cand = rows[i].querySelector(`.cell[data-col="${col}"]`);
        if (cand) { target = cand; break; }
      }
    } else return false;
    if (!target) return true;
    target.focus();
    target.scrollIntoView({block: "nearest"});
    if (open) target.click(); // a walk with the panel open keeps it in step
    return true;
  }

  document.addEventListener("keydown", e => {
    if (e.key === "Escape") { closeDetail(true); return; }
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (walk(e)) e.preventDefault();
  });

  renderMatrix();

  // deep link: #<probe> opens the probe, #<probe>@<client> opens that cell
  (function openFromHash() {
    const fragment = decodeURIComponent(location.hash.slice(1));
    if (!fragment) return;
    const [probeId, clientId] = fragment.split("@");
    if (clientId) {
      const i = DATA.clients.findIndex(c => c.id === clientId);
      if (i >= 0) { openCell(probeId, i); return; }
    }
    const row = root.querySelector(`tr.probe[data-probe="${CSS.escape(probeId)}"]`);
    if (row) { const b = row.querySelector(".plabel"); if (b) { b.click(); b.scrollIntoView({block: "center"}); } }
  })();
})();
