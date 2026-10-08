# Qualification evidence archive

The large generated JSON reports have moved out of the source tree. Written
results, performance budgets, smaller audit records, source identities, hash
manifests and screenshots remain here. No application, test harness or acceptance
budget changed as part of this cleanup.

[Download the complete original evidence archive](https://github.com/jeffspahr/jeffspahr-pipelines/releases/download/ui-modernization-evidence-20261006/kfp-ui-modernization-evidence-a058b109.tar.gz)
or [browse the archive release](https://github.com/jeffspahr/jeffspahr-pipelines/releases/tag/ui-modernization-evidence-20261006). This fork prerelease is an
evidence archive, not a Kubeflow Pipelines product release. It contains the entire
`frontend/docs/ui-modernization` directory from commit `a058b10903912eb8bd29fc15a24b63b8e2722987`.
The original files are also retained in that pinned Git commit, protected by the
archive release tag.

Archive SHA-256: `7cac429d045e7b2870c3199a76642f391667664abfcfb7f29fbc40352ca95bde`.

[File inventory and SHA-256 hashes](archived-evidence.json) identify every removed
report and link directly to its original contents. Extract the archive into a
separate directory to reproduce the original relative paths and validate the
existing dated `record-hashes.json` and other manifests. Those manifests describe
the original evidence snapshots; their archived entries are intentionally no
longer present in this checkout. Do not overlay historical evidence onto a working
tree used for new qualification.

Each dated report retains its own tested revision and limitations. Archiving an
older result does not qualify a newer head. The [tracking issue](https://github.com/kubeflow/pipelines/issues/14572)
and PR qualification links identify the final implementation's CI results.

## Archived reports

| Report | Original lines |
| --- | ---: |
| [layout-stability/performance-evidence.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/layout-stability/performance-evidence.json) | 14,742 |
| [qualification/lighthouse-accessibility.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/qualification/lighthouse-accessibility.json) | 4,376 |
| [qualification/performance-evidence.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/qualification/performance-evidence.json) | 10,908 |
| [release-qualification-2026-09-30/performance.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/performance.json) | 82,113 |
| [release-qualification-2026-09-30/vendor-records/artifact-file-hashes.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/artifact-file-hashes.json) | 1,030 |
| [release-qualification-2026-09-30/vendor-records/qualification-apple-desktop-26/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-apple-desktop-26/checks/result.json) | 9,409 |
| [release-qualification-2026-09-30/vendor-records/qualification-apple-desktop-27/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-apple-desktop-27/checks/result.json) | 9,409 |
| [release-qualification-2026-09-30/vendor-records/qualification-apple-ipad-26/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-apple-ipad-26/checks/result.json) | 11,284 |
| [release-qualification-2026-09-30/vendor-records/qualification-apple-iphone-26/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-apple-iphone-26/checks/result.json) | 11,501 |
| [release-qualification-2026-09-30/vendor-records/qualification-apple-iphone-27/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-apple-iphone-27/checks/result.json) | 11,553 |
| [release-qualification-2026-09-30/vendor-records/qualification-firefox-esr/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-firefox-esr/native/result.json) | 9,262 |
| [release-qualification-2026-09-30/vendor-records/qualification-firefox-esr-linux/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-firefox-esr-linux/native/result.json) | 9,262 |
| [release-qualification-2026-09-30/vendor-records/qualification-firefox-esr-overlap/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-firefox-esr-overlap/native/result.json) | 9,262 |
| [release-qualification-2026-09-30/vendor-records/qualification-firefox-previous/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-firefox-previous/native/result.json) | 9,262 |
| [release-qualification-2026-09-30/vendor-records/qualification-firefox-stable/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-firefox-stable/native/result.json) | 9,262 |
| [release-qualification-2026-09-30/vendor-records/qualification-firefox-stable-linux/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-09-30/vendor-records/qualification-firefox-stable-linux/native/result.json) | 9,262 |
| [release-qualification-2026-10-02/deployment-multiuser/audit.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-02/deployment-multiuser/audit.json) | 2,560 |
| [release-qualification-2026-10-02/deployment-multiuser/profile-authorization-state.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-02/deployment-multiuser/profile-authorization-state.json) | 2,737 |
| [release-qualification-2026-10-02/deployment-multiuser/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-02/deployment-multiuser/result.json) | 2,871 |
| [release-qualification-2026-10-02/deployment-standalone/audit.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-02/deployment-standalone/audit.json) | 2,446 |
| [release-qualification-2026-10-02/deployment-standalone/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-02/deployment-standalone/result.json) | 2,707 |
| [release-qualification-2026-10-02/performance.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-02/performance.json) | 82,343 |
| [release-qualification-2026-10-05/deployment-multiuser/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05/deployment-multiuser/result.json) | 2,871 |
| [release-qualification-2026-10-05/deployment-standalone/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05/deployment-standalone/result.json) | 2,743 |
| [release-qualification-2026-10-05/performance.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05/performance.json) | 119,208 |
| [release-qualification-2026-10-05-rebased/deployment-multiuser/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/deployment-multiuser/result.json) | 2,925 |
| [release-qualification-2026-10-05-rebased/deployment-standalone/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/deployment-standalone/result.json) | 2,665 |
| [release-qualification-2026-10-05-rebased/mobile/ipad26/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/mobile/ipad26/result.json) | 11,284 |
| [release-qualification-2026-10-05-rebased/mobile/ipad27/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/mobile/ipad27/result.json) | 11,286 |
| [release-qualification-2026-10-05-rebased/mobile/iphone26/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/mobile/iphone26/result.json) | 11,501 |
| [release-qualification-2026-10-05-rebased/mobile/iphone27/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/mobile/iphone27/result.json) | 11,553 |
| [release-qualification-2026-10-05-rebased/performance.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/performance.json) | 119,241 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-apple-desktop-26/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-apple-desktop-26/checks/result.json) | 9,409 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-apple-desktop-27/checks/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-apple-desktop-27/checks/result.json) | 9,409 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-esr/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-esr/native/result.json) | 9,262 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-esr-linux/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-esr-linux/native/result.json) | 9,262 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-esr-overlap/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-esr-overlap/native/result.json) | 9,262 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-previous/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-previous/native/result.json) | 9,262 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-stable/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-stable/native/result.json) | 9,262 |
| [release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-stable-linux/native/result.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/release-qualification-2026-10-05-rebased/vendor-desktop/qualification-firefox-stable-linux/native/result.json) | 9,262 |
| [workflows/performance-evidence.json](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/workflows/performance-evidence.json) | 3,285 |
