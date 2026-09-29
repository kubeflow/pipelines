# Hosted browser qualification

Browser qualification runs in GitHub Actions. It requires no workstation browser
installation, local Safari/Xcode setup, or manual test execution.

The existing [Frontend Tests workflow](../../../.github/workflows/frontend.yml)
runs 48 production-fixture cases across Chromium, Firefox and WebKit on Linux,
Windows and macOS. Its retained [423/423 checkpoint](https://github.com/kubeflow/pipelines/actions/runs/36520796567)
used 47 cases per lane and is engine coverage, separate from the exact-version
qualification below.

## Workflow and evidence

[Frontend Browser Qualification](../../../.github/workflows/frontend-browser-qualification.yml)
runs weekly and through workflow dispatch against the chosen release ref. Changes
to its scripts, fixtures, manifest and toolchain also run on PRs. Scheduled runs
begin after the workflow lands on the default branch. Release qualification must
refresh the dated vendor pins and run against the intended release commit.

One production build is shared by all lanes. Every selected lane is required;
failures are retained and cannot be hidden by another passing lane. Artifacts
include browser distribution/signature provenance where applicable, exact browser
and OS identity, source SHA, asset hashes, checks/screenshots, and lifecycle logs.
Available failure evidence is uploaded even after setup errors. Results and the
version manifest are retained for 90 days; the shared build is retained for seven.

The installers and native runners require disposable GitHub-hosted CI. Browser
archives, extracted apps, drivers and simulator tooling live in runner temporary
storage. Apple jobs create and remove their own simulator and driver processes.

## Dated matrix: 2026-09-29

The [machine-readable catalog](../../scripts/qualified-browsers.json) contains
vendor URLs, published checksums, signing identities, sources and unresolved
policy slots. Exact pins are intentional; a mismatched browser or missing runtime
fails the lane instead of silently substituting a different version.

| Lane                        | Version            | Suite               | Qualification role                                        |
| --------------------------- | ------------------ | ------------------- | --------------------------------------------------------- |
| Chrome stable               | 154.0.8037.58      | 48 production cases | Current stable policy row                                 |
| Chrome for Testing current  | 154.0.8037.57      | 48 production cases | Supplementary; different Mac patch                        |
| Chrome for Testing previous | 153.0.8010.52      | 48 production cases | Supplementary; different Mac patch                        |
| Edge stable                 | 154.0.4258.37      | 48 production cases | Current stable policy row                                 |
| Edge previous               | 153.0.4234.48      | 48 production cases | Previous stable policy row                                |
| Firefox stable              | 157.0              | 12 native checks    | Current stable policy row                                 |
| Firefox previous            | 156.0.1            | 12 native checks    | Previous stable policy row                                |
| Firefox ESR                 | 153.4.0            | 12 native checks    | Current ESR policy row                                    |
| Firefox outgoing ESR        | 140.17.0           | 12 native checks    | Supported ESR overlap                                     |
| macOS Safari                | 26.6.2             | 12 native checks    | Runner-available build; latest policy patches remain open |
| iPhone 17 simulator         | iOS/Safari 26.5    | 12 native checks    | Simulator coverage; not latest policy patch               |
| iPad (A16) simulator        | iPadOS/Safari 26.5 | 12 native checks    | Simulator coverage; not latest policy patch               |

The runner is `macos-26` ARM64. Apple automation uses Xcode 26.6 and pins Appium
3.8.0/XCUITest 12.13.3 for simulators. It records Safari bundle builds and simulator
runtime builds as well as marketing versions. The current Chrome vendor URL is
mutable; signed-app and exact-version checks reject a changed release. Chrome for
Testing is not substituted for a missing exact end-user Chrome patch.

Actual vendor releases require strict app-signature and vendor-team verification.
The two supplementary Chrome for Testing archives have observed linker ad-hoc
signatures without vendor resource seals. They instead require reviewed SHA-256
pins matching two hosted downloads, a pinned official Google Storage object
generation, and matching vendor MD5 and size. Their recorded packaging identity
and exact browser version must also match. Edge runs from an owned read-only disk
image because a hosted updater can ignore a newly installed policy. Its version,
full signature and executable hash are verified again after the suite, and the
volume is detached before uploading final provenance. Update-policy changes and
mounts are restricted to the disposable qualification runner.

## Native workflow coverage

The native suite extends the earlier seven-smoke checkpoint to 12 checks: platform
capabilities; Runs filtering; graph/task inspection; pipeline selection; production
YAML editor rendering; import validation; experiment draft values; run/recurring-run
drafts; artifacts/lineage; settings controls; comparison selection; and command
palette, focus, themes and responsive behavior. Desktop checks exercise keyboard
controls; simulator checks use touch-compatible interactions. Reports identify
which interactions were exercised on each platform.

These read/draft checks use fixed same-origin HTTP fixtures and reject backend
mutations. They do not establish parity with every request/mutation/recovery case
in the 48-case suite, real deployment authorization, physical-device behavior,
performance budgets, or untested assistive technology. Screenshots retain evidence;
they do not by themselves establish visual equivalence.

## Hosted results

First hosted execution is pending. Configured lanes are not counted as passes.

## Remaining policy coverage and BrowserStack follow-up

- Resolve actual previous Chrome Mac and Chrome/Edge Extended Stable distributions;
  the catalog records why the available supplementary builds do not close them.
- Qualify current/previous annual Safari and iOS/iPadOS at their latest policy
  patches. The pinned runner-available builds above are partial Apple coverage.
- Extend native Firefox/Apple mutation and recovery parity beyond the read/draft
  checks, and qualify the remaining exact desktop OS/version combinations.
- Investigate [BrowserStack open-source access](https://www.browserstack.com/open-source)
  for missing exact versions and physical iPhone/iPad coverage. Its program includes
  Automate, but access, available inventory, concurrency and retention must be
  confirmed before relying on it. Use the [Automate inventory API](https://www.browserstack.com/docs/automate/api-reference/selenium/browser)
  to establish exact available combinations once credentials exist.
- Validate CI connection to the fixture server, immutable build provenance,
  screenshots/log retention and account access requirements before proposing a
  BrowserStack integration. No account application or purchase is part of this work.

The [browser support policy](../browser-support.md) remains unchanged. Missing
coverage stays open in the issue and is not reclassified as a successful test.
