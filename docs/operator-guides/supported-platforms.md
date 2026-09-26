# Supported platforms

## Architecture support by release

| KFP release line | Supported Kubernetes node architectures |
| --- | --- |
| 2.x, including 2.18 | Linux AMD64 (`linux/amd64`) |
| 3.0 and later | Linux AMD64 and ARM64 (`linux/amd64`, `linux/arm64`) |

ARM64 is a supported architecture beginning with KFP 3.0. Architecture-specific
defects are handled through the normal bug-reporting and maintenance process.
Features retain their existing support status and prerequisites on either
architecture; architecture support does not make an experimental feature stable.

This is the support policy for the upcoming 3.0 release. Current master already
has native ARM64 installation and pipeline-execution validation. Release images
must also pass the [release validation requirements](../contributing/platform-validation.md)
before a release is declared ready. A successful master build is not a released
version.

Some 2.x images include ARM64 variants, but the 2.x installation retains
architecture-specific ML Metadata dependencies. Those image variants do not
constitute ARM64 support for 2.x, and the final 2.x release retains its existing
support boundary.

## Installation and image selection

Use the same standard KFP manifests and shared release image tags on either
architecture. The container runtime selects the appropriate image from the
multi-architecture index; architecture-specific manifest forks or tag edits are
not needed.

The ordinary KFP image set consists of the API server, frontend, persistence
agent, scheduled-workflow controller, viewer controller, driver, and launcher.
Starting with 3.0, their shared release tags must contain both supported Linux
architectures. Optional `-amd64` and `-arm64` aliases do not replace the shared tag.

Distribution-specific add-ons and external services retain their own platform
requirements. The legacy GCP inverse-proxy image remains AMD64-only and is not
part of the standard multi-architecture image set. This policy does not change
that integration. Upgrade and migration requirements likewise follow the
release's upgrade guidance, independently of architecture.

See the [installation guide](installation.md) for deployment options. The policy
describes deployed KFP services and runtime images, not the architecture of
maintainer build-tool containers or the machine running the Python SDK.

## Pipeline component images

Pipeline authors are responsible for supplying component images that support the
nodes where their tasks run. A multi-architecture KFP installation cannot make an
AMD64-only user image execute natively on ARM64. Publish multi-architecture
component images or use appropriate Kubernetes scheduling constraints when a
component requires a particular architecture or accelerator.

## Reporting problems

Report architecture-specific problems through the usual
[KFP issue tracker](https://github.com/kubeflow/pipelines/issues). Include the KFP
version, node architecture, installation profile, affected image references, and
pod events or logs. Known incompatibilities should be documented specifically
rather than inferred from differences in test coverage.
