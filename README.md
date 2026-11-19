# Kubeflow Pipelines on native Argo

This fork keeps KFP's Python authoring model and graph/log widgets while using
Argo Workflows as the execution model and source of truth.

```text
KFP Python DSL → captured tasks/groups/channels → WorkflowTemplate / Workflow / CronWorkflow
                                                       ↓
                                            Argo workflow-controller
                                                       ↓
                                       component containers + Argo executor

Native KFP UI → Kubernetes + Argo Server → resources, archive, logs and artifacts
```

The KFP API server, ScheduledWorkflow/persistence/viewer controllers, driver and
launcher services, run database, REST clients, legacy frontend server, and their
build/deployment/release stack are removed. Component execution has no in-process
runner or fallback to a KFP backend.

## Start here

- [Build, deploy and verify on local Kubernetes](samples/argo-native/README.md)
- [Compile a pipeline](docs/user-guides/core-functions/compile-a-pipeline.md)
- [Run a pipeline](docs/user-guides/core-functions/run-a-pipeline.md)
- [Architecture and cleanup boundary](docs/agents/architecture.md)
- [Contribution instructions](CONTRIBUTING.md)

`make sdk-test` runs the retained SDK and cleanup-boundary tests.
`make frontend-test` builds and tests the native UI and offline viewer.
`bash test/native-deployment.sh` creates a disposable Kind stack and runs actual
workflow and authenticated browser acceptance tests. Use explicit contexts when
working with an existing cluster.

## Retained schemas

The shared `pipeline_spec.proto` and Kubernetes executor-config schema remain for
DSL type checking, portable container definitions and Kubernetes argument models.
They are not REST APIs, deployed controllers, or a pipeline execution IR builder.
Regenerate their Python bindings with `make generate-python`.

The native compiler intentionally has a bounded compatibility contract. Unsupported
constructs fail explicitly; inspect the sample README before migrating a pipeline.
Global artifact indexing, migration of historical KFP runs, and KFP client/server
compatibility are not provided. The sample stack uses local development storage
credentials and must not be exposed as a production multi-tenant installation.
