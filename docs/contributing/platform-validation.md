# Platform validation and releases

The [supported-platform policy](../operator-guides/supported-platforms.md) is the
user-facing support contract. This page describes how CI and release preparation
provide evidence for it.

## Development coverage

The broad functional matrix runs on AMD64. Relevant pull requests additionally
run one centralized native ARM64 image build and a representative standalone
installation/pipeline smoke. This test executes the driver and launcher, disables
pipeline caching, and checks an ARM execution marker. It does not require every
functional permutation to run twice.

After master publication, the same ARM smoke consumes the verified immutable
shared indexes from that publication attempt. Architecture-qualified build
artifacts and Kind caches keep AMD64 and ARM64 inputs separate.

## Release qualification from 3.0 onward

For each backend release, including patch releases:

1. Resolve one immutable source commit and build the normal image set for both
   Linux architectures.
2. Validate the complete platform inventory before promoting shared release
   tags. Retain source-matched index records for that workflow attempt.
3. On native AMD64 and ARM64 runners, pull the shared release tags without a
   platform override. Check that the tags still reference the recorded indexes
   and that the locally selected images match the expected architecture and
   image configuration from those indexes.
4. Run the native ARM standalone installation/pipeline smoke using the exact
   published index records and manifests from the resolved source commit.
5. Require a successful complete release workflow before creating/announcing
   the GitHub release. Retain links to its native validation and smoke evidence.

The release workflow performs these checks after publishing images; it is not an
atomic publish-or-rollback transaction. A validation failure blocks release
qualification even though image tags may already exist. Resolve the failure
before continuing the release checkpoints. Follow the publication workflow's
full-run retry requirements so digest artifacts remain from the same attempt.

A workflow dry run builds without publishing and skips checks of published
release tags. It is useful preparation, but not release-qualification evidence.
Adding this machinery does not require cutting an RC now. Manual prerelease tags
can use the publishing workflow; the `kfpr` CLI retains its existing
`MAJOR.MINOR.PATCH` version format.

## The remaining 2.x release

Use the current [release CLI](https://github.com/kubeflow/pipelines/tree/master/release)
with the 2.x version and its `release-2.x` branch. The CLI dispatches the publishing
workflow **from that release branch**, not from master with an older source
checkout. This preserves the legacy image inventory, including MLMD, and the
existing workflow inputs. There is no new ARM64 support or ARM validation
requirement for 2.x, and an existing ARM image variant is not a support claim.

Do not dispatch master's post-MLMD publishing workflow for a 2.x release or
backport the 3.x validation/profile wholesale. Release images and documentation
must describe the selected release line. SDK-only releases do not acquire
backend-image validation requirements.
