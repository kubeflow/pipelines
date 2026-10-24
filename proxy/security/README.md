# Inverse-proxy Debian triage

This is an additive review view, not a scanner replacement or release-policy input.
It never rewrites OSV/SARIF evidence, dismisses alerts, filters findings, or changes
qualification decisions. The release gates continue to consume their original
reports and acceptance rules.

Download and retain the complete qualification manifest, image scan envelope,
and build log from the exact CI run. Save a Debian tracker snapshot alongside
those artifacts, then generate a separate view:

```sh
curl -fsSL https://security-tracker.debian.org/tracker/data/json -o debian-tracker.json
python3 proxy/security/triage_debian.py \
  --scan raw-osv.json --tracker debian-tracker.json --output triage.json
```

The view records hashes of both input files and the original image binding. Verify
that binding and scan checksum against the qualification manifest separately.
Keep the snapshots: tracker status may change after a scan. The view refuses to
overwrite either input, including through symlinks or hard links.

Grouping uses ecosystem, source-package name, **reported version**, and advisory
ID. It retains every original binary package and source location. Versions are
not normalized or merged: binary packages may carry different epochs or rebuild
suffixes. Distinct-CVE counts therefore differ from grouped findings. For example,
the Debian 13 baseline from run 40057661555 has 394 occurrences, 180 version-aware
source groups, and 168 advisory IDs.

Tracker status is context, not proof that an installed version is fixed. Only
Debian 13/trixie is interpreted; other releases remain unsupported. An unimportant
rating, missing fix, ignored issue, or absence of a security advisory does not
mean not affected. Missing tracker data remains unknown.

## Applicability review

`applicability-review.json` records a bounded review of the Debian 13 AMD64 image
from the baseline run. Each entry lists the observed package versions and its
supporting source. Revalidate these conclusions when packages, build options,
architecture, or registration behavior change. No qualification code reads this
file. Even reviewed findings remain in the raw reports and release gates.

Eight advisory IDs have vendor-build, vendor-configuration, or architecture
exclusions. A ninth, the jq test-runner assertion, is outside the registration
execution path; that narrower conclusion is not a blanket package exemption.
Everything not listed remains unassessed for application reachability. In
particular, curl/TLS and Python execute during registration; do not dismiss those
families just because Debian defers a fix. Source-package matches for systemd or
nscd also require checking the affected executable, not merely its library name.

## Runtime minimization

Registration uses curl for both metadata requests and endpoint-config download,
so wget is unnecessary. The runtime explicitly keeps Python and the installed
requests environment. Pip and Debian's venv-creation/seed-wheel packages are
removed after installation; no package metadata is deleted to hide findings.
The Cloud SDK and its bundled interpreter are retained. Image builds run upstream agent tests in the builder stage, then rerun
registration, real endpoint discovery, binary smoke checks, and tests of this
review tool after runtime cleanup. Test sources and this tool
are not retained in the final runtime filesystem.
