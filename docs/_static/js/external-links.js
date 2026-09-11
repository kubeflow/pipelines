// Open links to other sites in a new tab. Compare parsed hostnames rather than
// substrings so that a URL merely containing the current host somewhere in its
// path or query is not misread as internal.
document.addEventListener("DOMContentLoaded", function () {
  var currentHost = window.location.hostname;

  document.querySelectorAll("a[href]").forEach(function (link) {
    if (link.classList.contains("top-nav-brand")) return;

    var href = link.getAttribute("href");
    if (!href || !(href.startsWith("http://") || href.startsWith("https://"))) {
      return;
    }

    try {
      var url = new URL(href);
      if (url.hostname && url.hostname !== currentHost) {
        link.setAttribute("target", "_blank");
        link.setAttribute("rel", "noopener noreferrer");
      }
    } catch (e) {
      return;
    }
  });
});
