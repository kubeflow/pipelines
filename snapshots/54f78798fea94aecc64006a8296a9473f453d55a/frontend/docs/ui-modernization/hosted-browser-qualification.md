# Hosted browser qualification

Browser qualification runs in GitHub Actions. It requires no workstation browser
installation, local Safari/Xcode setup, or manual test execution.

The historical [Frontend Tests workflow](../../../.github/workflows/frontend.yml)
passes [432/432 production-fixture cases](https://github.com/kubeflow/pipelines/actions/runs/36624711864)
across Chromium, Firefox and WebKit on Linux, Windows and macOS: 48 cases per lane,
with no skipped cases. At PR head `70cde25a20f80097a786a4a0f4c68392f4bccb57`
(tested merge `c062298a63a7fe427ac5815784a38a4b96ef787f`), audited TAP/JUnit and
workflow reports agree on source identity and identical hashes for all 22 shared
build files within the engine matrix. This is engine coverage, separate from exact-version qualification.
The same hosted source passes 1,760 UI tests and 1,103 server tests, formatting,
lint, type checks, React peer checks and the production build.

## Workflow and evidence

[Frontend Browser Qualification](../../../.github/workflows/frontend-browser-qualification.yml)
runs weekly and through workflow dispatch against the chosen ref. It includes the
full nine-lane OS/engine matrix, including Linux Chromium, plus the 17
vendor/native-browser lanes. Each qualification run tests its selected source;
it does not reuse a previous PR's Chromium result. Unit/server tests, lint, type checks, the
production build and Linux Chromium remain required in `frontend.yml`. Scheduled
runs begin only after the workflow lands on the default branch. Refresh the dated
vendor pins before a manual release-candidate run and use the intended release
commit.

Browser, performance and deployment/rollback qualification are observational,
not PR or release gates. Manual release-candidate evidence is advisory until
[issue #14754](https://github.com/kubeflow/pipelines/issues/14754) establishes
repeatability and maintainers separately approve promotion. The daily CI health
report includes these workflows. First-attempt failure reporting, deduplicated
alerts, overdue-run detection and triage ownership remain follow-up work. Confirmed product regressions still require triage.

One production build is shared by all lanes. Every selected lane must pass for
the run to count as successful evidence; failures are retained and cannot be
hidden by another passing lane. Nonblocking status does not use
`continue-on-error`, relax supported-browser policy or weaken assertions. Artifacts
include browser distribution/signature provenance where applicable, exact browser
and OS identity, source SHA, asset hashes, checks/screenshots, and lifecycle logs.
Available failure evidence is uploaded even after setup errors. Results and the
version manifest and shared build are retained for seven days. GitHub clamped the
previous 90-day request to the repository maximum of seven days, verified from
artifact creation/expiry metadata on September 30. Selected sanitized release
records must be preserved in the repository before these downloads expire.

The installers and native runners require disposable GitHub-hosted CI. Browser
archives, extracted apps, drivers and simulator tooling live in runner temporary
storage. Apple jobs create and remove their own simulator and driver processes.

## Dated matrix: 2026-09-29

The [machine-readable catalog](../../scripts/qualified-browsers.json) contains
vendor URLs, published checksums, signing identities, sources and unresolved
policy slots. Exact pins are intentional; a mismatched browser or missing runtime
fails the lane instead of silently substituting a different version.

Chrome stable was refreshed to 154.0.8037.93 after its vendor download changed.
[Google Version History](https://versionhistory.googleapis.com/v1/chrome/platforms/mac_arm64/channels/stable/versions/154.0.8037.93/releases)
records full ARM64 stable rollout from September 29 at 18:30 UTC. Earlier Chrome
154.0.8037.58 checkpoints remain historical; they do not qualify the new pin.

| Lane                        | Version            | Suite               | Qualification role                                        |
| --------------------------- | ------------------ | ------------------- | --------------------------------------------------------- |
| Chrome stable               | 154.0.8037.93      | 48 production cases | Current stable policy row                                 |
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
command and suite deadlines. Mobile Appium commands through first-page readiness allow 60 seconds; subsequent
suite interactions keep 40 seconds. Cold Web Inspector and native accessibility
queries have returned correct results just beyond 40 seconds. Appium also waits
for an in-flight native alert probe after JavaScript returns, so the startup
allowance includes initial page readiness. Alert detection remains enabled.
Native XCTest idle waits are set to a positive two seconds, with quiescence
checks retained. The harness reads the live setting from the owned WebDriverAgent
session and requires it to match before checking the application. This bounds the
stacked waits for native input without weakening readiness or exact-value assertions.
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
and exact browser version must also match. Actual Chrome and Edge run from owned
read-only disk images because hosted updaters can change a browser during the
suite. Each version, full signature and executable hash is verified again after
the suite, and the volume is detached before uploading final provenance. Update-policy changes and
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

[Run 36624712051](https://github.com/kubeflow/pipelines/actions/runs/36624712051)
passes **12/12 lanes and 324/324 checks**, all on attempt 1, at PR head
`70cde25a20f80097a786a4a0f4c68392f4bccb57` and tested merge
`c062298a63a7fe427ac5815784a38a4b96ef787f`. Every configured row in the dated matrix
above is verified for its stated scope:

| Verified lanes                                      | Passing checks           |
| --------------------------------------------------- | ------------------------ |
| Actual Chrome stable and Edge stable/previous       | 144/144 production cases |
| Supplementary Chrome for Testing current/previous   | 96/96 production cases   |
| Firefox stable/previous and both supported ESR rows | 48/48 native checks      |
| Desktop Safari with recorded CI text preferences    | 12/12 native checks      |
| iPhone 17 simulator, Safari 26.5                    | 12/12 native checks      |
| iPad (A16) simulator, Safari 26.5                   | 12/12 native checks      |

The independent audit verifies source/merge identity, all 22 shared build-file
hashes within the qualification matrix, exact browser and downloaded-distribution
identity, all 98 native screenshot hashes, Chrome/Edge final signed executable
identity, and owned volume/simulator cleanup. All 57 browser-free qualification
helper checks pass. The separate engine build has identical runtime files; its
`Editor-CiEMRP2l.js.map` differs only in two build-generated Vite placeholders.
The source-map difference is retained explicitly in the build comparison.

Both simulators verify first-page startup completion before the checks, unchanged
filter scale 1→1, native run/artifact link activation with route assertions, and
native Recurring/One-off activation with associated-radio and checked-state
assertions. Live WebDriverAgent settings read back the positive two-second idle
wait. Prebuilt WDA package identity and output hashes are recorded with the exact
owned simulator and derived-data path; those binaries are not uploaded, so their
hashes are reported provenance rather than independently rehashed artifacts.

Native Safari preparation handles its known first-launch prompts, keyboard and
active address editor before interaction. Wrapped inline targets use an actual
client fragment for hit testing. Uniquely typed native link/radio actions avoid
[Appium coordinate-calibration errors](https://github.com/appium/appium-xcuitest-driver/blob/v12.13.3/lib/commands/web-native-bridge.ts#L376)
while retaining viewport, route, selection and draft-value assertions. Native
alert detection remains enabled throughout the bounded startup and test phases.

Qualification also caught two application regressions: artifact reads/downloads
now wait for experiment namespace resolution, and shared Input/TextField text is
16px on coarse-pointer devices to prevent Safari input focus zoom. The earlier
13px field produced scale 1.231343 on iPhone, matching
[WebKit's focus-zoom calculation](https://github.com/WebKit/WebKit/blob/main/Source/WebKit/UIProcess/API/ios/WKWebViewIOS.mm#L1752-L1773).
Current native checks preserve scale and inspector Close reachability. User zoom
remains enabled; arbitrary pinch-zoom modal reachability is still unqualified.

Earlier failed attempts retain evidence of vendor-version drift, simulator
startup/discovery, native calibration and an isolated previous-Edge first-test
timeout. Those partial results are not counted in the passing matrix above.
The narrower native fixture scope, policy-version gaps and release acceptance
criteria remain separate from this completed qualification checkpoint.

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
