# Hosted browser qualification

Browser qualification runs in GitHub Actions. It requires no workstation browser
installation, local Safari/Xcode setup, or manual test execution.

The existing [Frontend Tests workflow](../../../.github/workflows/frontend.yml)
passes [432/432 production-fixture cases](https://github.com/kubeflow/pipelines/actions/runs/36594507403)
across Chromium, Firefox and WebKit on Linux, Windows and macOS: 48 cases per lane,
with no skipped cases. At PR head `6bf2852e62d57cc080a30fe846bc870f1b3c54ac`
(tested merge `c37456c9f2190a892ed42d8b864f313a7ca327de`), audited TAP/JUnit and
workflow reports agree on source identity and identical hashes for all 22 shared
build files. This is engine coverage, separate from exact-version qualification.
The same hosted source passes 1,760 UI tests and 1,103 server tests, formatting,
lint, type checks, React peer checks and the production build.

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

All lanes use standard `macos-26` ARM64 runners. Simulator OS initialization has
a separate five-minute deadline. Simulator jobs then build WebDriverAgent in a
separate eight-minute phase before Appium session startup, record its resolved
package version and output hashes, and require the generated test manifest, runner
app and test executable before reuse. Mobile jobs have a 45-minute outer limit;
desktop Safari retains 35 minutes. Native UI checks retain their own bounded
command and suite deadlines.
Apple automation uses Xcode 26.6 and pins Appium 3.8.0/XCUITest 12.13.3 for simulators. It records Safari bundle builds and simulator
runtime builds as well as marketing versions. The current Chrome vendor URL is
mutable; signed-app and exact-version checks reject a changed release. Chrome for
Testing is not substituted for a missing exact end-user Chrome patch.

Downloaded Chrome, Edge and Firefox releases require strict app-signature and
vendor-team verification.
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
controls; simulator checks use native page taps and WebDriver text-entry commands.
iOS WebDriver text entry uses Appium's WebKit typing atom and does not establish
physical-keyboard or IME behavior. Reports identify the interactions exercised on
each platform. The read-only YAML check compares the complete existing Ace
document and read-only setting before and after native keyboard input on desktop
and theme changes on all platforms; virtualized visible lines are not a stable
proxy for document contents.

Desktop Safari CI configures macOS inline predictions and automatic spelling
correction off before browser launch and records each preference readback.
[Apple documents the user settings](https://support.apple.com/guide/mac-help/mchlp2299/mac);
the `NSGlobalDomain` key mapping follows the
[nix-darwin implementation](https://github.com/nix-darwin/nix-darwin/blob/4cff07de74b50e64bdd68cd4e722ab5b6b35ee48/modules/system/defaults/NSGlobalDomain.nix),
not an Apple-published CLI contract. Exact field assertions still apply. Native
IME, predictive-text and autocorrection compatibility remain unqualified; these
checks establish deterministic native keyboard input only. Retained input-event
diagnostics distinguish native composition edits from application value resets.

These read/draft checks use fixed same-origin HTTP fixtures and reject backend
mutations. They do not establish parity with every request/mutation/recovery case
in the 48-case suite, real deployment authorization, physical-device behavior,
performance budgets, or untested assistive technology. Screenshots retain evidence;
they do not by themselves establish visual equivalence.

## Hosted results

[Run 36594507275](https://github.com/kubeflow/pipelines/actions/runs/36594507275)
verified all nine vendor/supplementary lanes and desktop Safari at PR head
`6bf2852e62d57cc080a30fe846bc870f1b3c54ac`, tested merge
`c37456c9f2190a892ed42d8b864f313a7ca327de`:

| Verified lanes                                      | Passing checks           |
| --------------------------------------------------- | ------------------------ |
| Actual Chrome stable and Edge stable/previous       | 144/144 production cases |
| Supplementary Chrome for Testing current/previous   | 96/96 production cases   |
| Firefox stable/previous and both supported ESR rows | 48/48 native checks      |
| Desktop Safari with recorded CI text preferences   | 12/12 native checks      |

The audit verified source identity, all 22 shared build-file hashes, exact browser
and downloaded-distribution identity, 70 native screenshot hashes, and both Edge
post-suite identities and volume cleanup. These 300 passing checks qualify those
ten rows only. Desktop Safari retained exact experiment name and description
values; its bounded input trace recorded no composition deletion or application
value reset. Prediction, autocorrection and IME behavior remain unqualified.

Both mobile simulator lanes remain open. The latest
[run 36604957937](https://github.com/kubeflow/pipelines/actions/runs/36604957937)
at `fd3c43f08bce6ea6d2c8a7c85de92d0917f8af7a` verified all nine vendor lanes
(288 checks) and 35 helper tests. Desktop Safari exposed a brittle comparison of
Ace's virtualized visible lines after focus; the assertion now uses its complete
document model. iPhone reached Run Details, then target scrolling moved a graph
node beneath graph controls. Placement now scrolls the containing canvas while
preserving internal offsets. If a node is cropped at minimum zoom, at most two
native pans start on verified empty canvas space and must move it toward the
visible area. Final viewport and hit-test checks remain required. iPad's cold
WebDriverAgent compilation succeeded but consumed driver startup time; the
separate build phase above addresses that setup boundary. These corrections need
hosted verification; partial or failed lanes do not count as qualified.

Native Safari preparation dismisses the keyboard through its browser-owned form
toolbar, then closes recognized first-launch prompts before page readiness or
taps. Run-link navigation uses a uniquely identified native accessibility link:
XCUITest 12.13.3 can match both a link and its text child and fall back to
[incorrect coordinate conversion under Safari input zoom](https://github.com/appium/appium-xcuitest-driver/blob/v12.13.3/lib/commands/web-native-bridge.ts#L376).
Native activation, route/task assertions and recorded viewport hit tests remain
required. Browser setup and interaction preparation do not relax application
assertions.

Earlier attempts retained evidence of vendor-version drift, native input timing
and simulator startup failures. Exact-version checks caught Edge updating during
a suite; the read-only installation and final identity check now prevent that
from being accepted. A Chromium check also exposed artifact storage access before
experiment namespace resolution. Task, sub-DAG and artifact-node reads/downloads
now wait for that namespace, with unit regressions and a new production-browser
case. The corrected 48-case suite passed on actual Chrome and both supplementary
Chrome for Testing versions in the superseded
[36586016478 attempt](https://github.com/kubeflow/pipelines/actions/runs/36586016478).
That partial attempt does not qualify the complete matrix.

## Remaining policy coverage and BrowserStack follow-up

- Resolve actual previous Chrome Mac and Chrome/Edge Extended Stable distributions;
  the catalog records why the available supplementary builds do not close them.
- Qualify current/previous annual Safari and iOS/iPadOS at their latest policy
  patches. The pinned runner-available builds above are partial Apple coverage.
- Extend native Firefox/Apple mutation and recovery parity beyond the read/draft
  checks, and qualify the remaining exact desktop OS/version combinations.
- Automate native IME, predictive-text and autocorrection scenarios separately
  from the deterministic desktop Safari input configuration.
- Investigate [BrowserStack open-source access](https://www.browserstack.com/open-source)
  for missing exact versions and physical iPhone/iPad coverage. As of September 29,
  its public program lists Automate, five users and five parallel sessions; project
  approval, available inventory and retention still need confirmation. Its
  [GitHub Actions integration](https://www.browserstack.com/docs/automate/selenium/github-actions)
  supports automated execution and a CI tunnel to the fixture server. Use the
  [Automate inventory API](https://www.browserstack.com/docs/automate/api-reference/selenium/browser)
  to establish exact available combinations once credentials exist.
- Validate CI connection to the fixture server, immutable build provenance,
  screenshots/log retention and account access requirements before proposing a
  BrowserStack integration. No account application or purchase is part of this work.

The [browser support policy](../browser-support.md) remains unchanged. Missing
coverage stays open in the issue and is not reclassified as a successful test.
