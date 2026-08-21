// masthead: the newest run on the site. Like every date on every page here it
// comes from the reports, never the clock — a site with no submissions at all
// has no run to name, and says so rather than showing today.
(() => {
  const root = document.querySelector('[data-c="masthead"]');
  root.querySelector(".date").textContent = DATA.generated || "—";
})();
