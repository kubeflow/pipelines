# Inputs are pages from the upstream GitHub Releases API. A release replay must
# upgrade from an older patch even after the target itself has been published.
def patch: capture("^2\\.18\\.(?<patch>0|[1-9][0-9]*)$").patch | tonumber;
# Non-version image tags use the ordinary PR baseline.
(($target | capture("^2\\.18\\.(?<patch>0|[1-9][0-9]*)(?:[-+].+)?$").patch | tonumber) // null) as $target_patch
| [.[][] | select(.draft == false and .prerelease == false)
   | .tag_name | {tag: ., patch: patch}
   | select($target_patch == null or .patch < $target_patch)]
| max_by(.patch).tag // error("No eligible published 2.18 patch release")
