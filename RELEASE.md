# Kubeflow Pipelines releases

Use the notes for the exact version you are installing to understand changes,
required upgrade actions, security fixes, and known limitations.

| I want to… | Start here |
| --- | --- |
| See what shipped | [GitHub releases](https://github.com/kubeflow/pipelines/releases): backend tags are `MAJOR.MINOR.PATCH`; Python SDK tags are `sdk-MAJOR.MINOR.PATCH` |
| Review SDK changes and migration instructions | [SDK release notes](sdk/RELEASE.md) (the development section describes unreleased work) |
| Browse the detailed change history | [Changelog](CHANGELOG.md) |
| Install or upgrade a deployment | [Installation guide](https://www.kubeflow.org/docs/components/pipelines/operator-guides/installation/), plus the target version's release notes |
| Check compatibility | [Dependency compatibility matrix](README.md#dependencies-compatibility-matrix) and [supported platforms](docs/operator-guides/supported-platforms.md); check the target release's notes for version-specific limits |
| Use the Python SDK or Kubernetes integration | [SDK reference](https://kubeflow-pipelines.readthedocs.io/en/stable/), including Kubernetes APIs; [legacy Kubernetes integration reference](https://kfp-kubernetes.readthedocs.io/en/stable/) for split-package releases |
| Prepare or publish a release | [Maintainer release runbook](release/README.md) |
| Contribute a release-note entry | [Release-note policy](CONTRIBUTING.md#release-note-policy) |

Backend and SDK versions can advance independently. A similarly numbered tag is
not a compatibility guarantee: consult the selected SDK and backend release notes
before upgrading. The maintainer runbook covers backend-only, SDK-only, and
combined releases, including unified SDK and split-package maintenance releases.
