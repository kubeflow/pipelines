# Browser support

The UI modernization adopts the following browser support policy, approved by a
maintainer on 2026-09-28. Release qualification against this policy is still
pending. The [supported-browser checkpoint](ui-modernization/supported-browser-qualification.md)
records actual Chrome/Edge workflow runs and native Firefox stable/ESR smoke;
it does not establish support for every channel or OS below.

| Browser                            | Supported releases                                                                                                         |
| ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Chrome and Microsoft Edge, desktop | Latest two stable major releases, plus the current Extended Stable release for each browser.                               |
| Firefox, desktop                   | Latest two stable major releases, plus the current ESR and the outgoing ESR during Mozilla's supported transition overlap. |
| Safari on macOS                    | Current and previous annual Safari releases, each at its latest available patch on a vendor-supported macOS version.       |
| Safari on iOS and iPadOS           | Current and previous annual OS releases, each at its latest available patch.                                               |

Safari's window means annual releases, not the latest two minor updates. Use
vendor-supported operating systems. Beta, nightly, embedded webviews and other
unlisted browsers are outside the qualification commitment. Older browsers may
continue to work, but KFP does not qualify them or block access by user-agent
version. Browser support does not change Python `kfp.local` behavior or require a
local SDK or artifact migration.

## Release and build rules

For each KFP minor release, resolve the channel policy above into a dated matrix
of exact browser and OS versions. Retain browser/OS identities, application
commit, emitted asset hashes, test results and known limitations. Recheck vendor
channels before release rather than treating this document as a permanent version
list. For patch releases, test current supported channels while preserving the
minor release's minimum required browser capabilities and compiler targets.
Intentional increases in required browser capabilities belong in a KFP minor
release and must be announced in its release notes.

The [production Browserslist](../package.json) and [Vite targets](../vite.config.mts)
currently retain conservative output targets: Chrome/Edge 111, Firefox 128 and
Safari/iOS 16.4. These are compiler compatibility settings, **not KFP's supported
browser matrix**. They accommodate the stack's technical minimums without
requiring qualification of those retired versions. Keep them explicit and aligned;
raising them is not necessary to adopt this policy. Any future target change must
remain compatible with the oldest supported enterprise/Safari release, pass
production build and browser qualification, and refresh asset-specific evidence.
JavaScript syntax transforms do not supply missing runtime APIs or CSS features.

## Automated qualification

Qualification uses automated browser workflows, keyboard/focus assertions,
accessibility audits and screenshot regression checks. Manual test sessions are
not part of the delivery plan. Automated coverage does not establish blanket WCAG
conformance or untested screen-reader/speech-control behavior; retain these limits
explicitly instead of treating them as passes.

The production fixture suite runs in CI across Chromium, Firefox and WebKit on
Linux, Windows and macOS. CI engine coverage is separate from the exact branded
stable/enterprise/Safari release matrix below. Each lane must pass and retain its
actual browser version, OS/source identity, bundle hashes and test outcomes. A
failed lane must not be hidden by a successful lane or `continue-on-error`.

Browser qualification runs on disposable hosted CI runners; workstation browser
installation and local Safari/Xcode setup are not prerequisites. The separate
[hosted qualification workflow](../../.github/workflows/frontend-browser-qualification.yml)
runs weekly and on demand for releases, with PR validation when its tooling changes.
Its [dated catalog and coverage report](ui-modernization/hosted-browser-qualification.md)
separate supported-version evidence from supplementary builds and unavailable slots.
Refresh exact channel pins before release; a successful dated run does not prove
that later vendor releases have been tested.

Actual Safari and iOS/iPadOS qualification uses automated native-browser/device
sessions. Unavailable environments remain coverage gaps. Investigate
[BrowserStack's open-source program](https://www.browserstack.com/open-source) for
future automated device and version coverage; eligibility and exact available
browser/device versions must be confirmed before relying on it.

## Qualification before release

- [x] Adopt the channel policy and retire Chrome/Edge 111, Firefox 128 and Safari/iOS 16.4 as release qualification targets.
- [x] Record a [dated qualification checkpoint](ui-modernization/supported-browser-qualification.md), with exact tested builds and remaining channel/OS gaps.
- [ ] Complete the dated exact-version/OS matrix for the release, including Chrome/Edge Extended Stable and all Firefox ESR releases still in their supported transition.
- [ ] Run the production workflow suite on current and previous stable Chrome/Edge/Firefox and the enterprise releases above. Retain failures and route/action/state results for each actual browser identity.
- [ ] Qualify current and previous annual macOS Safari releases and iOS/iPadOS devices or simulators, recording which was used. Automate touch, zoom, keyboard/focus, dialogs, inspector scrolling and visual-viewport checks in both themes; retain actual device identities and record any unavailable physical-device automation.
- [ ] Verify the final production bundle against the oldest supported versions as well as current releases. Resolve supported-browser failures that block loading, core workflows, data integrity, keyboard access or reachable content before release; document other limitations explicitly.
- [ ] Investigate BrowserStack open-source eligibility and automated Safari/iOS/iPadOS, enterprise-version and desktop-OS coverage. Confirm exact versions, CI integration, result retention and access requirements before proposing account setup.
- [ ] Publish the tested matrix and changed requirements in release notes. Keep deployment, automated accessibility, performance-budget and rollback gates tracked separately, with untested assistive-technology limitations explicit.

Playwright Chromium/Firefox/WebKit provide useful engine coverage; they do not
substitute for every branded browser version, Safari device or OS combination.
The [retained reports](ui-modernization/browser-accessibility-qualification.md)
identify their original versions and source assets. The Firefox 128 run remains
historical regression evidence, not qualification of the newly supported ESRs.
Keep the feature-detected input activation fallback and its regression tests;
retiring a version alone does not prove that behavior is absent in every supported
browser.

The opt-in [native WebDriver smoke](../scripts/ui-modernization-browser-floor.mjs)
requires `KFP_BROWSER_FLOOR_VERSION` to equal the actual browser's full version.
Set `KFP_WEBDRIVER_BROWSER` to `firefox`, `chrome`, `MicrosoftEdge` or `safari` and
use an isolated matching driver with the existing loopback fixture API and
production preview. The smoke fails if the reported version differs. Its seven
scenarios supplement the full workflow and device checks; they do not replace them.

## References

- [Chrome release channels](https://support.google.com/chrome/a/answer/16942104)
- [Microsoft Edge release schedule](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-release-schedule)
- [Firefox ESR release cycle and overlap](https://support.mozilla.org/en-US/kb/firefox-esr-release-cycle)
- [Safari release notes](https://developer.apple.com/documentation/safari-release-notes)
- [Tailwind technical browser requirements](https://tailwindcss.com/docs/compatibility)
