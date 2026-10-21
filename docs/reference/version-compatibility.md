# Version Compatibility

Use the KFP v2 SDK with the v2 runtime. The SDK compiles components and pipelines
to [PipelineSpec IR YAML](../concepts/ir-yaml.md). The backend's canonical REST
API is `/apis/v2`, while the Python SDK continues to call `/apis/v2beta1` so
routine SDK upgrades do not require a server prefix upgrade.

The server also accepts `/apis/v2beta1` and the old `v2beta1` gRPC service
names through compatibility adapters to the same v2 handlers. Existing request
and response fields, filters, and pagination tokens are unchanged; HTTP
requests are forwarded internally without redirects. Python `V2beta1…` model
imports remain aliases of `V2…` models, including direct model-module imports.
Legacy Go import paths are also aliases/forwarders to v2: recompiled callers use
canonical endpoints and protobuf message identities, while already-built clients
continue to use the legacy wire routes. Reflection and protobuf `Any` type URLs
use canonical v2 names; the server does not register historical descriptors or
dynamic legacy message types. The frozen schema is a test-only compatibility
baseline. There is only one implementation per client language.
Upgrade the backend before adopting clients that call `/apis/v2`; older servers
do not necessarily expose it. This is not a compatibility promise for removed
v1 APIs or other breaking changes in the release.

PipelineSpec IR (`v2alpha1`) and Kubernetes-native pipeline CRDs
(`pipelines.kubeflow.org/v2beta1`) have independent version lifecycles.
See the [backend API guide](https://github.com/kubeflow/pipelines/blob/master/backend/api/README.md).

## Rollout order

A v2 server accepts both prefixes in parallel. The Python SDK retains
`/apis/v2beta1` until v2 is available across supported server versions; it does
not probe v2 and fall back. New UI and Go runtime clients use v2 and require
the API-server upgrade first. The UI reports the API as not ready when its
`/apis/v2/healthz` probe returns a non-success status.

For a deployment whose storage/runtime generation is already compatible:

1. Pause new submissions and recurring schedules as needed during the transition.
   Keep existing controllers and driver/launcher image settings until all API
   replicas support v2; do not launch new runtime clients against mixed API pods.
2. Update the API-server image first, retaining the old runtime image settings.
   Wait for its rollout to complete and verify both `/apis/v2/healthz` and
   `/apis/v2beta1/healthz` through the installation's normal authenticated route.
3. Update the persistence agent, scheduled-workflow controller, driver/launcher
   image settings, and UI. Resume submissions/schedules after the rollout.
   Python SDK updates do not require this ordering solely for the API prefix;
   new SDK features may still require a matching backend.

A single `kubectl apply` that updates all deployments concurrently does not
establish this ordering. These routing steps are not an MLMD-to-native migration
procedure: older storage generations and in-flight execution transitions still
require the separately tracked work in
[#14029](https://github.com/kubeflow/pipelines/issues/14029).

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
