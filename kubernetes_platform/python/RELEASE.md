## Kubernetes SDK releases

Kubernetes configuration is now part of the single `kfp` distribution. Use the
[SDK-only recipe](../../release/README.md#sdk-only-release) in the canonical maintainer runbook; do not bump a separate Kubernetes
version or publish `kfp-kubernetes` from master.

`kfp.kubernetes.__version__` comes from `kfp.version`. Its API reference is
included in the SDK documentation. New releases no longer create separate
`kfp-kubernetes-*` documentation branches. Existing documentation remains
available until its Read the Docs project redirects users to the SDK reference.

For maintenance of a split-package release, use the scripts and instructions
from that release's existing branch, not master.

Inspect the [release checkpoint](../../release/README.md#recovery-helpers) before retrying a partial release. Review the [Markdown preview](../../release/README.md#review-release-notes-before-publishing) before publishing SDK notes.
