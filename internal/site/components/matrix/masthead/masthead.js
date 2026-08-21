// masthead: whose measurement this page is — the corpus, the newest run
// among the lanes, and the kit that graded them. The keys line advertises
// only the affordances the page actually has, and whether a column head opens
// a client's own guide is a fact about the registry, so it is read from the
// data here rather than reported by the grid.
(() => {
  const root = document.querySelector('[data-c="masthead"]');
  root.querySelector(".corpus").textContent = DATA.corpus.version;
  root.querySelector(".date").textContent = DATA.generated;
  root.querySelector(".kit").textContent = DATA.kit;
  if (DATA.clients.some(c => c.docs)) root.querySelector(".docs").hidden = false;
})();
