// Adds a sidebar-collapse button on wide screens.
// localStorage is wrapped in try/catch because it can throw in private/hardened browsers.
document.addEventListener("DOMContentLoaded", function () {
  var mq = window.matchMedia("(min-width: 67em)");
  if (!mq.matches) return;

  var sidebar = document.querySelector(".sidebar-drawer");
  if (!sidebar) return;

  var collapsed = false;
  try {
    collapsed = localStorage.getItem("sidebar-collapsed") === "true";
  } catch (e) {
    collapsed = false;
  }

  var btn = document.createElement("button");
  btn.className = "sidebar-toggle-btn";
  btn.setAttribute("aria-label", "Toggle sidebar");
  btn.title = "Toggle sidebar";
  btn.innerHTML =
    '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor"' +
    ' stroke-width="2" stroke-linecap="round">' +
    '<line x1="4" y1="6" x2="20" y2="6"/>' +
    '<line x1="4" y1="12" x2="20" y2="12"/>' +
    '<line x1="4" y1="18" x2="20" y2="18"/>' +
    "</svg>";
  document.body.appendChild(btn);

  function apply(val) {
    collapsed = val;
    document.body.classList.toggle("sidebar-collapsed", collapsed);
    try {
      localStorage.setItem("sidebar-collapsed", collapsed);
    } catch (e) {
      return;
    }
  }

  btn.addEventListener("click", function () {
    apply(!collapsed);
  });

  apply(collapsed);

  mq.addEventListener("change", function (e) {
    btn.style.display = e.matches ? "" : "none";
    if (!e.matches) document.body.classList.remove("sidebar-collapsed");
  });
});
