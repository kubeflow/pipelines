# Native Kubeflow Pipelines

This fork keeps KFP authoring and visualization and executes native Argo resources.
The KFP API server, drivers, launchers, ScheduledWorkflow/persistence/viewer
controllers, REST clients and their deployment stack have been retired.

## Changes and evidence

- Start bug fixes with a repeatable reproducer and add a regression test. Verify
  the expected failure before the fix and the passing result afterward.
- Record exact validation commands and results, including anything unverified.
- PR descriptions start with `## Problem` when the change needs explanation.
  Include a before/after diagram or screenshot when applicable.
- Sign commits with `git commit -s`; do not add agent co-authors.
- Preserve authoring type checks and native cluster execution. Use Kubernetes and
  Argo clients for management; do not restore retired KFP endpoints or controllers.

## Guides

Read the relevant guide before changing its area:

- [Conventions](docs/agents/conventions.md)
- [Architecture](docs/agents/architecture.md)
- [Development/deployment](docs/agents/development.md)
- [Tests/formatting](docs/agents/testing.md)
- [Schema generation](docs/agents/generation.md) for generated bindings
- [Frontend](docs/agents/frontend.md)
- [CI](docs/agents/ci.md)
- [Dependencies](docs/agents/dependabot.md)
- [Domain](docs/agents/domain.md) when exploring domain behavior
- [Issue tracking](docs/agents/issue-tracker.md) for issue/PR work
- [Triage labels](docs/agents/triage-labels.md) for triage

The executable deployment and compatibility contract are documented in
[samples/argo-native/README.md](samples/argo-native/README.md). Use explicit
Kubernetes contexts; preserve unrelated workloads and existing run/artifact data.
