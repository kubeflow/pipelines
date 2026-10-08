# Rebased release-candidate qualification — October 5, 2026

Implementation head `8de88ada3ca3748fbdfbce6f2c6c69d60766b8be` was tested as merge
`f06cc3dcfdbf0a0d6380a55f5fe5b60b72c1a507` with master
`e3c93651a44fa718ab079010bb332318854e0491`. This snapshot predates the subsequent
native iPad keyboard fallback and release-note publication. Those changes do not
change application assets; their subsequent CI status is tracked in issue #14572.
No PR merge or release publication is represented by this report.

## Required qualification

- [Frontend CI](https://github.com/kubeflow/pipelines/actions/runs/37268437985):
  1,763 UI tests, 1,183 server tests and 432 browser checks across nine
  Linux/Windows/macOS Chromium/Firefox/WebKit lanes pass. The audited shared
  bundle contains 28 files including six valid gzip sidecars.
- [Performance](https://github.com/kubeflow/pipelines/actions/runs/37268437794):
  all 29 budgets pass. Independent audit verifies 126 matched observations and
  trace hashes, parsed trace contents, actual transfer encodings/lengths,
  28 larger-workload observations and the unchanged 596,806-byte entry gzip size.
- [Deployment](https://github.com/kubeflow/pipelines/actions/runs/37268438678):
  all 18 standalone and 21 authenticated multi-user upgrade/rollback checks pass.
  Image/config/runtime-manifest and asset identities, preserved data/preferences,
  unchanged backend and authentication/signing invariants are independently audited.
- [Pre-commit](https://github.com/kubeflow/pipelines/actions/runs/37268437696)
  passes after inheriting master's isort 5.10.1 pin and formatting the remaining
  long import in the Go-version maintenance helper.

Seven-sample medians in the matched throttled experiment:

| Endpoint | Previous modern | Candidate |
| --- | ---: | ---: |
| Cold editor display | 3,339.5 ms | 1,373.3 ms |
| Initialized YAML worker | 3,690.1 ms | 1,454.7 ms |
| Runs readiness | 13,319.6 ms | 5,385.2 ms |
| Run Details readiness | 13,405.8 ms | 5,431.6 ms |
| Compare readiness | 13,520.1 ms | 5,552.2 ms |

The protocol remains 4× CPU, 150 ms latency, 1.6 Mbps download, 0.75 Mbps upload,
1280×720/DPR1, fresh contexts and warm browser/runner caches. Larger workloads
are unthrottled. These are fixture measurements, not field percentiles. All 35
candidate non-editor observations request no editor/worker assets. Original
JS/CSS bytes match the previous qualified modern build; no editor preload was added.
Trace/report timestamp comparison allows 0.2 ms rounding (maximum observed
0.144 ms in legacy evidence); this audit tolerance changes no acceptance budget.

Recreate recovery upper bounds were 40.085 s upgrade / 38.052 s rollback for
standalone and 43.377 s / 40.725 s for multi-user. These are observations, not a
zero-downtime claim or a maximum guarantee. Deployment records verify decoded
asset bodies; wire gzip encoding is separately verified in performance and HTTP tests.

## Dated configured browser matrix

[Browser qualification](https://github.com/kubeflow/pipelines/actions/runs/37268437745)
passes **17/17 configured lanes and 420/420 checks**, with no failed or skipped
checks. This is configured coverage, not a claim that every supported distribution,
patch, physical device or accessibility environment has been exercised.

| Browser/environment | Observed versions | Checks per lane |
| --- | --- | ---: |
| Chrome, macOS | 154.0.8037.98 | 48 |
| Chrome for Testing, supplementary | 154.0.8037.57; 153.0.8010.52 | 48 |
| Edge, macOS | 154.0.4258.37; 153.0.4234.48 | 48 |
| Firefox, macOS | 157.0; 156.0.1; ESR 153.4.0; ESR 140.17.0 | 15 |
| Firefox, Linux | 157.0; ESR 153.4.0 | 15 |
| Desktop Safari/macOS | 26.6.2/26.6.2; 27.0/27.0 | 15 |
| iPhone simulators, MobileSafari/iOS | 26.5/26.5; 27.0/27.0 | 15 |
| iPad simulators, MobileSafari/iPadOS | 26.5/26.5; 27.0/27.0 | 15 |

The vendor/desktop audit covers 360 checks and 136 screenshot hashes. Five
Playwright vendor lanes record all 28 shared assets; eight native desktop lanes
record their two loaded entry assets, matching the shared bundle. The four mobile
lanes add 60 checks and 176 successful native preparation operations, with matching
source/entry identities, no browser errors and successful cleanup. Their complete
raw records retain simulator/WDA/Safari identities and native action evidence.
Mobile screenshot references remain in those records; this report does not claim
independent validation of every mobile screenshot's bytes.

Earlier iPhone debugger and iPad Appium startup failures recovered in the normal
rerun without widening timeouts. iPad26's generic keyboard dismissal also passed
this run. A subsequent narrow native Hide keyboard fallback handles the exact
control captured in the prior failed run; it preserves ambiguity and keyboard-gone
assertions and has 47 browser-free regression checks. This snapshot does not claim
that fallback executed here.

## Disposition and retention

The [release and migration notes](../release-notes.md) preserve the approved
browser policy and explain the accepted scope: status-selector/global-statistic
deferrals, automated-only acceptance, and explicit provider/device/IME/pinch/AT
coverage follow-ups. Available configured-lane failures remain blockers. Final
published-image verification, merge and release execution remain release-owner
steps; they are not performed by this preparation task.

Raw JSON records and independent audits are retained alongside this report.
`record-hashes.json` verifies retained bytes after formatting, and raw OCI manifest
filenames verify their exact digest identities. Complete traces, screenshots,
images and logs are available from Actions for seven days; retained hashes do not
retrieve expired artifacts. Historical reports retain their own source provenance.
