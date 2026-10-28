# Supported-browser qualification checkpoint

Broad qualification tooling is maintained in [qualification PR #14757](https://github.com/kubeflow/pipelines/pull/14757). Commands and results in this document refer to that tooling or the explicitly recorded historical source; they are not additional presubmit gates in the application PR. See [the evidence index](evidence-archive.md) for retained records.

Follow-up: the [automated qualification plan](../browser-support.md#automated-qualification) expands CI across engines and desktop operating systems, uses native automation for Apple coverage, and tracks BrowserStack open-source access for future gaps. Manual test sessions are not part of the current delivery plan; the results below retain their original scope.

This checkpoint applies the [approved browser policy](../browser-support.md) to
actual vendor browser builds. Channel selection was resolved on September 28,
2026 (America/New_York); native records retain their September 29 UTC timestamps.
Re-resolve the rolling channels before release. This is partial qualification,
not completion of the browser/device release gate.

## Results and provenance

The full production fixture suite passed **47/47 cases each** on Chrome
154.0.8037.58, Edge 154.0.4258.37 and Edge 153.0.4234.48: **141 branded-browser
cases**. The [matrix](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/branded-browser-matrix.json) retains exact
browser identities, all case names, eight harness hashes and 20 emitted asset
hashes. These are new runs, separate from the earlier 141-case Playwright-engine
matrix. Tests cover production loading, Runs, Run Details, Pipelines, other
workflows, artifacts/viewers, populated comparisons and graph behavior.

Four actual Firefox builds passed the native smoke: **7/7 checks each**, or 28
checks. These cover required platform features, Runs filtering, graph/task
inspection, pipeline selection, pointer/Space switch activation, two-run empty
comparison and a narrow dark command dialog with keyboard containment. All four
retained results have no captured post-readiness page errors. This is scoped
native smoke, not the full 47-case workflow suite or rich-comparison coverage.

| Policy slot                             | Exact tested browser | Evidence and result                                               |
| --------------------------------------- | -------------------- | ----------------------------------------------------------------- |
| Chrome current stable                   | 154.0.8037.58        | 47/47 production cases                                            |
| Edge current stable                     | 154.0.4258.37        | 47/47 production cases                                            |
| Edge previous stable                    | 153.0.4234.48        | 47/47 production cases                                            |
| Firefox current stable                  | 156.0.1              | [7/7 native checks](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox156/result.json)    |
| Firefox previous stable                 | 155.0.1              | [7/7 native checks](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox155/result.json)    |
| Firefox new ESR                         | 153.3.0              | [7/7 native checks](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox153esr/result.json) |
| Firefox outgoing ESR, supported overlap | 140.16.0             | [7/7 native checks](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox140esr/result.json) |

All executed checks above used macOS 26.6.2 (25G83), arm64, with fresh temporary
browser profiles or contexts. This does not qualify Windows or Linux. Firefox's
privacy-reduced user-agent OS/architecture string is not the host identity.

Application source is `8fe4fb4ec548e18268a5ea4c1a059992b24ee976`; all 20 emitted assets
match the [measured e79f8d42 build](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/bundle-inventory.json) byte for
byte. The full-suite harness source is `bbfe20296b9053e5dba9d3532dfb1f7517adda5b`;
the native smoke source is `8fe4fb4e` (the same blob at `bbfe2029`). No application source, dependency or compiler
target changed for this checkpoint. Previous unit, build and performance evidence
retains its original source identity; no new performance comparison is claimed.

Firefox DMGs were checked against Mozilla's versioned SHA256SUMS and their app
signatures verified. Edge packages were checked against vendor enterprise API
SHA-256 values, package signatures and app signatures. The retained distribution
records identify [Edge 154](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/edge-154.0.4258.37-distribution.json),
[Edge 153](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/edge-153.0.4234.48-distribution.json), and each Firefox
archive. Chrome used the installed vendor binary, with its actual version asserted
by the harness. Extraction of portable apps did not run installers.

All 24 Firefox screenshots and hashes are retained alongside their results.
Visual spot checks covered [current Firefox graph inspection](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox156/run-details-inspector.png),
[current Firefox narrow dark dialog](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox156/command-dialog-narrow-dark.png),
[outgoing ESR switch state](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox140esr/feature-switch.png) and
[outgoing ESR narrow dark dialog](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/supported-browsers/firefox140esr/command-dialog-narrow-dark.png).
No visual blocker was found in those four states; this is not an exhaustive visual
or accessibility assessment.

## Remaining matrix

| Policy slot                  | Resolved target or constraint                                                              | Qualification still required                                                                                                                                         |
| ---------------------------- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Chrome previous stable       | Major 153; vendor Mac ARM releases include 153.0.8010.53 and staged .54/.55 variants       | Choose and record an exact vendor distribution/patch; run the full suite.                                                                                            |
| Chrome Extended Stable       | 152.0.7977.140 for Mac/Windows                                                             | Acquire the exact enterprise build and run the full suite.                                                                                                           |
| Edge Extended Stable         | Windows 152.0.4191.100; exact current Mac enterprise patch still unresolved                | Resolve platform-specific package identity and run the full suite.                                                                                                   |
| Firefox stable and both ESRs | The four exact versions above                                                              | Expand native smoke to full workflow/state coverage; do not substitute Playwright's patched Firefox.                                                                 |
| Safari current annual        | Safari 27; select latest patch for the chosen supported macOS                              | Actual Safari qualification, both themes and responsive/keyboard behavior.                                                                                           |
| Safari previous annual       | Installed identity 26.6.2 (21624.5.1.11.3); latest eligible patch still needs confirmation | No completed Safari application run; obtain actual-browser workflow and viewport evidence.                                                                           |
| iOS/iPadOS current annual    | 27.0.1                                                                                     | Actual device/simulator browser identity, touch/zoom/focus/dialog/inspector tests, and automated physical-device coverage where available; otherwise record the gap. |
| iOS/iPadOS previous annual   | 26.7.1                                                                                     | Same device checks, using a supported OS/device pair.                                                                                                                |
| Additional desktop platforms | Vendor-supported Windows and Linux combinations                                            | Record exact OS/browser identities and representative platform coverage.                                                                                             |

Safari application qualification did not start successfully; no pass is claimed.
The exact Safari and mobile rows require a working browser/device test environment.
A desktop WebKit run does not close these rows. Mobile OS release numbers alone do
not establish the actual Safari/WebKit build; record that from the test device.
Neither a Chrome for Testing build with a different patch number nor an Edge
Windows catalog entry establishes the identity of a requested Mac enterprise build.

The dated matrix is a checkpoint, not a completed release matrix: unresolved
patch/OS selections and all untested rows remain open. Deployment authorization,
assistive-technology acceptance, measured budgets and same-backend rollback remain
separate gates.

## Reproduction and harness verification

Use the repository-pinned Node/npm versions and the already built production
bundle. The eight production smoke files accept `KFP_BROWSER=chromium`,
`KFP_BROWSER_EXECUTABLE_PATH` and `KFP_EXPECTED_BROWSER_VERSION`; an explicit path
wins over `PLAYWRIGHT_CHANNEL`. Run them with `node --test --test-concurrency=1`.
The matrix records the complete command and case list for each browser. A deliberate
wrong-version run verified rejection before page checks, using a conflicting
channel to verify explicit-path precedence. All eight scripts passed syntax,
formatting and repository hooks.

The following historical reproduction requires the qualification tooling from [#14757](https://github.com/kubeflow/pipelines/pull/14757) combined with application #14584, run from that checkout’s `frontend` directory. The native Firefox smoke uses the existing loopback API fixtures and Vite
production preview. Set `KFP_FIREFOX_BINARY`, `KFP_BROWSER_FLOOR_VERSION` and
`KFP_BROWSER_FLOOR_OUTPUT`, then run
`node scripts/ui-modernization-browser-floor.mjs` with a matching loopback
WebDriver. Each session closes in `finally`. Retained capabilities omit temporary
profile paths, process IDs and transport addresses; browser, driver, platform,
source, asset and screenshot identities remain explicit.

## Vendor sources

- [Chrome 154 stable release](https://chromereleases.googleblog.com/2026/09/stable-channel-update-for-desktop_0856730748.html), [Chrome September enterprise announcements](https://chromereleases.googleblog.com/2026/09/), and [Chrome Mac ARM stable version history](https://versionhistory.googleapis.com/v1/chrome/platforms/mac_arm64/channels/stable/versions/all/releases?filter=version%3E%3D153,version%3C154).
- [Microsoft Edge enterprise distribution API](https://edgeupdates.microsoft.com/api/products?view=enterprise), [release schedule](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-release-schedule) and [Extended Stable Windows catalog](https://www.catalog.update.microsoft.com/Search.aspx?q=Microsoft+Edge-Extended+Stable+Channel).
- [Firefox 156.0.1](https://www.firefox.com/en-US/firefox/156.0.1/releasenotes/), [155.0.1](https://www.firefox.com/en-US/firefox/155.0.1/releasenotes/), [153.3.0 ESR](https://www.firefox.com/en-US/firefox/153.3.0/releasenotes/), [140.16.0 ESR](https://www.firefox.com/en-US/firefox/140.16.0/releasenotes/) and [live Mozilla channel metadata](https://product-details.mozilla.org/1.0/firefox_versions.json).
- [Apple current releases](https://support.apple.com/en-us/100100) and [Safari 27 availability](https://support.apple.com/en-us/149039). Apple's security index is not a complete standalone patch catalog for every supported macOS version.
