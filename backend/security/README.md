# Python runtime image Debian review

This review covers the release-2.18 visualization-server and metadata-writer
images. It does not suppress alerts or change release security gates. Keep the
complete OSV scan envelopes, raw SARIF and qualification lane manifests; the
CSV here is an additional review index, not replacement evidence.

## Baseline and provenance

[CI run 40225213135](https://github.com/kubeflow/pipelines/actions/runs/40225213135)
scanned source `b2ee75287f09640db6a668537632ae141b43af89` with OSV Scanner 2.5.0.
The downloaded visualization-server AMD64 and ARM64 and metadata-writer AMD64
artifacts all contain the same Debian findings. This review does not imply
metadata-writer ARM64 was scanned in that run.

Each image/architecture has **164 raw Debian result occurrences**, **81 distinct
source-package/version/advisory groups**, and **69 distinct advisory IDs** across
18 source packages. Repeated occurrences are not 164 independent vulnerabilities;
the same advisory can also affect several source packages. The SARIF does not
retain all binary-package identities, so do not infer binary-package removability
from a source-package match alone.

The [baseline CSV](python-image-debian-baseline.csv) records all 81 groups,
reported versions, occurrence counts and vendor status. Status is from the
[Debian tracker JSON](https://security-tracker.debian.org/tracker/data/json)
downloaded on 2026-10-25, SHA-256 `0a96c0576126a41e77ec38cafba672cd7708dd93f8e253e22ce7a3a5f897b564`.

| Scanned artifact | Image ID | Scan envelope SHA-256 |
| --- | --- | --- |
| kfp-metadata-writer linux/amd64 | `sha256:f1c77b459b7e579fa4299dc047a3707f9a8cb7f65806bb0531ab79aafe44d974` | `e323d098ca7f54ded03f8a5292ae902bf0536c7b46695d5db9568ba13c001e15` |
| kfp-visualization-server linux/amd64 | `sha256:1f4003b8fc886f7baee5c4b8edf27d1da0e18ccbee876c24556ecae4e2455003` | `c6c34de881525c0966366c202c7a7e8a27daf552357db8836865339fcb931ef2` |
| kfp-visualization-server linux/arm64 | `sha256:d642623c1aa597b04cb83753a96d432e8a3ef5eff7ed560039172be96c2760a1` | `44eba8deef5638c94a29a736c7ecf5487ddcde62054ba7b39084bd11e01d6b69` |

## Fixability and applicability

- **79 groups are open with no fixed trixie version listed**, including two
  marked postponed. This means the tracker provides no Debian upgrade target
  for these matches at this snapshot. It does not establish that they are
  harmless, permanently unfixable, or unreachable in the application.
- **One group is undetermined:** PAM 1.7.0-5,
  [CVE-2025-8941](https://security-tracker.debian.org/tracker/CVE-2025-8941),
  with four raw occurrences. This concerns pam_namespace local privilege
  escalation. Retain it pending validation of installed modules and PAM
  configuration; source-package presence alone is insufficient for dismissal.
- **One group has a vendor not-affected determination:** SQLite
  3.46.1-7+deb13u2,
  [CVE-2026-39113](https://security-tracker.debian.org/tracker/CVE-2026-39113).
  Debian explicitly states that the vulnerable code was never present in a
  Debian released version and records fixed_version `0`. The reported installed
  version matches the trixie version in that tracker entry. This is a bounded
  vendor applicability conclusion, not an installed-version upgrade fix. Keep
  the raw scanner finding and release-gate treatment unchanged.

No remaining group has a positive fixed-version target in the trixie tracker
snapshot. Prioritize verification of libc, SQLite, compression and terminal
libraries used by Python, then inspect individual executables and modules in
utility families before proposing runtime removal. A low urgency or postponed
fix is not a not-affected determination. Do not remove dpkg metadata to hide
source-package matches.

## Source-package breakdown

Counts apply separately to each of the three scanned image/architecture pairs.
Distinct groups include the reported version; do not sum advisory counts across
packages to infer distinct vulnerabilities.

| Source package | Reported version(s) | Raw occurrences | Distinct groups |
| --- | --- | ---: | ---: |
| acl | `2.3.2-2+b1` | 2 | 2 |
| apt | `3.0.3` | 2 | 1 |
| attr | `1:2.5.2-3` | 1 | 1 |
| bzip2 | `1.0.8-6` | 1 | 1 |
| coreutils | `9.7-3` | 4 | 4 |
| dash | `0.5.12-12` | 2 | 2 |
| diffutils | `1:3.10-4` | 1 | 1 |
| gcc-14 | `14.2.0-19` | 6 | 2 |
| glibc | `2.41-12+deb13u4` | 42 | 21 |
| ncurses | `6.5+20250216-2` | 8 | 2 |
| pam | `1.7.0-5` | 8 | 2 |
| perl | `5.40.1-6+deb13u1` | 4 | 4 |
| shadow | `1:4.17.4-2` | 4 | 2 |
| sqlite3 | `3.46.1-7+deb13u2` | 5 | 5 |
| systemd | `257.13-1~deb13u1` | 14 | 7 |
| tar | `1.35+dfsg-3.1` | 4 | 4 |
| util-linux | `1:2.41.5-0+deb13u1`, `1:4.16.0-2+really2.41.5-0+deb13u1`, `2.41.5-0+deb13u1` | 54 | 18 |
| zlib | `1:1.3.dfsg+really1.3.1-1+b1` | 2 | 2 |

Revalidate this baseline against the rebuilt images and a retained tracker
snapshot after dependency changes. New package versions, architectures or Debian
status changes can invalidate this review. No application reachability exemption
is claimed for the 80 open or undetermined groups.
