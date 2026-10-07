# Version Compatibility

Use the KFP v2 SDK with the v2 runtime. The SDK compiles components and pipelines
to [PipelineSpec IR YAML](../concepts/ir-yaml.md), and clients use the
`/apis/v2` REST API.

The server also accepts `/apis/v2beta1` and the old `v2beta1` gRPC service
names through compatibility adapters to the same v2 handlers. Existing request
and response fields, filters, and pagination tokens are unchanged; HTTP
requests are forwarded internally without redirects. Python `V2beta1…` model
imports remain aliases of `V2…` models, including direct model-module imports.
Upgrade the backend before adopting clients that call `/apis/v2`; older servers
do not necessarily expose it. This is not a compatibility promise for removed
v1 APIs or other breaking changes in the release.

This backend API promotion does not change PipelineSpec IR (`v2alpha1`) or
Kubernetes-native pipeline CRDs (`pipelines.kubeflow.org/v2beta1`).
See the [backend API guide](https://github.com/kubeflow/pipelines/blob/master/backend/api/README.md).

Use matching SDK and runtime releases when adopting new features: support may
arrive in the compiler before it is available in your deployed runtime. Consult
the [release notes](https://github.com/kubeflow/pipelines/releases) for the release
you deploy.

Argo Workflow YAML is an internal runtime output, not a supported pipeline upload
format. Component loaders accept only compiled PipelineSpec IR YAML. Loading
legacy v1 container component YAML is no longer supported, including from v2
pipeline sources. Convert those files with an older compatible KFP v2 SDK before
upgrading, or rewrite the components with v2 decorators. See
[upgrading pipeline definitions](../user-guides/migration.md).
