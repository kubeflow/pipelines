# Architecture

`GraphComponent` captures DSL tasks, groups and channels. The native compiler
lowers those objects to WorkflowTemplates, Workflows and CronWorkflows. Argo owns
execution, status, transfer, cache and archive; the Python bridge runs inside each
component container. The static UI/proxy has no application database or backend.

The two protobuf bindings in `kfp.pipeline_spec` and `kfp.kubernetes` remain shared
authoring/configuration schemas. `dsl/component_definition.py` serializes portable
container definitions for the SDK loader/image builder; it does not register tasks,
compile pipeline IR or contact a server. This is distinct from the removed REST
API definitions and old pipeline IR builder.

The browser reads Kubernetes ConfigMaps for experiments and Argo resources for
versions, runs and schedules. Argo Server provides hydrated/offloaded status and
archive/artifact access. UID is run identity. Cache metadata retains the original
producer. Tokens stay in memory and namespace roles enforce permissions.

`test/native_cleanup_test.py` guards the retired-service boundary. The repo has no
Go execution backend, KFP controllers, KFP REST clients, or in-process runners.
