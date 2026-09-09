// Resolve the docs root relative to the current page so the brand link works
// under any base path. Sphinx exposes it via data-content_root or DOCUMENTATION_OPTIONS.URL_ROOT.
document.addEventListener("DOMContentLoaded", function () {
  var brand = document.querySelector(".top-nav-brand");
  if (!brand) return;

  var root = null;
  var el = document.querySelector("[data-content_root]");
  if (el) {
    root = el.getAttribute("data-content_root");
  } else if (
    typeof DOCUMENTATION_OPTIONS !== "undefined" &&
    DOCUMENTATION_OPTIONS.URL_ROOT != null
  ) {
    root = DOCUMENTATION_OPTIONS.URL_ROOT;
  }

  if (!root) root = "./";
  if (root.charAt(root.length - 1) !== "/") root += "/";

  brand.setAttribute("href", root + "index.html");

  // Keep internal navigation in the same tab.
  brand.removeAttribute("target");
  brand.removeAttribute("rel");

  // Resolve the logo src relative to the current page (see href above).
  var logo = document.querySelector(".top-nav-logo");
  if (logo && logo.tagName === "IMG") {
    logo.setAttribute("src", root + "_static/pipelines-icon.svg");
  }
});

// Light/dark-only toggle; cloneNode strips Furo's handler so only ours runs.
document.addEventListener("DOMContentLoaded", function () {
  var toggle = document.querySelector(".top-nav-theme-toggle");
  if (!toggle) return;

  var fresh = toggle.cloneNode(true);
  fresh.classList.remove("theme-toggle");
  toggle.parentNode.replaceChild(fresh, toggle);

  function resolvedTheme() {
    var t = document.body.dataset.theme;
    if (t === "light" || t === "dark") return t;
    // Stored value is "auto" or unset — fall back to the OS preference.
    return window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
  }

  fresh.addEventListener("click", function () {
    var next = resolvedTheme() === "dark" ? "light" : "dark";
    document.body.dataset.theme = next;
    try {
      localStorage.setItem("theme", next);
    } catch (e) {
      return;
    }
  });
});
