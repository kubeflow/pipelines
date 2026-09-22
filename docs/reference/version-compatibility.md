# Version Compatibility

Use the KFP v2 SDK with the v2 runtime. The SDK compiles components and pipelines
to [PipelineSpec IR YAML](../concepts/ir-yaml.md), and clients use the
`/apis/v2beta1` REST API.

Use matching SDK and runtime releases when adopting new features: support may
arrive in the compiler before it is available in your deployed runtime. Consult
the [release notes](https://github.com/kubeflow/pipelines/releases) for the release
you deploy.

Argo Workflow YAML is an internal runtime output, not a supported pipeline upload
format. Component loaders accept compiled IR, not handwritten component YAML.
See [upgrading pipeline definitions](../user-guides/migration.md) before upgrading
an existing installation.
