# Compile a Pipeline

This fork compiles the KFP Python DSL directly to native Argo resources. It captures
tasks, groups and channels without first building PipelineSpec IR. Compile only
trusted Python sources: decorators execute module-level Python while capturing the
graph, but do not run pipeline tasks.

```python
from kfp import compiler, dsl

@dsl.container_component
def echo(message: str):
    return dsl.ContainerSpec(image='busybox:1.36.1', command=['echo', message])

@dsl.pipeline(name='hello')
def pipeline(message: str = 'Hello'):
    echo(message=message).set_caching_options(False)

compiler.Compiler().compile(pipeline, package_path='pipeline.yaml')
```

The output is an inspectable `argoproj.io/v1alpha1` `WorkflowTemplate`. Its name
contains a content hash; labels identify the logical pipeline and version.
Immutability is a convention, not a Kubernetes property. The native UI creates
versions without updating existing ones.

## Resource kinds

| `resource_kind` | Result |
|---|---|
| `WorkflowTemplate` (default) | Reusable pipeline version |
| `Workflow` | Standalone run with `metadata.generateName` |
| `CronWorkflow` | Recurring run, initially suspended; requires `schedule` |

For example:

```python
compiler.Compiler().compile(
    pipeline,
    'schedule.yaml',
    resource_kind='CronWorkflow',
    schedule='0 9 * * *',
    namespace='kfp-argo',
)
```

Argo 4 uses `spec.schedules`; the Python argument remains `schedule` for one cron
expression. The old KFP `Pipeline`/`PipelineVersion` manifest mode and IR output
are not this compiler's contract.

## Type checking and unsupported features

DSL argument type checking remains enabled. Unsupported constructs fail with a
compile error rather than silently losing their settings. The current supported
corpus covers Python/container components, parameters and typed artifacts, nested
DAGs, conditions, bounded loops, scalar collection and failure-aware exit handlers.

The native prototype does not promise full SDK compatibility. Its executable
corpus and limitations are documented in `samples/argo-native/README.md` in the
repository. In particular, non-Python container scalar outputs currently require
`String`; the Python bridge validates other return types.

## Configuration and caching

`Compiler` accepts `artifact_repository`, `service_account_name`,
`cache_config_map`, `image_digests` and `disable_caching`. The prototype uses an
S3-compatible artifact repository. Runtime artifact URIs and Argo storage keys must
refer to the same repository.

Caching requires immutable image identity. Prefer an image pinned by registry
digest. For local builds, `image_digests` maps source image names to verified Docker
IDs; compiled templates use corresponding `kfp-<digest>` local image tags. Create
those immutable-by-convention snapshots before submission. The example's
`compile_demo.py` does this and refuses to overwrite conflicting content.

Cache keys include resolved input values and artifact identities, not just values
known at compile time. Mutable external configuration and runtime package
installation require caching to be disabled. `disable_caching=True` explicitly
overrides all per-task cache settings; offline Python graph previews use this mode.

Python component images must contain this checkout's runtime or install its wheel.
An unmodified upstream runtime does not implement the native Argo bridge.

## CLI

```sh
kfp dsl compile --py pipeline.py --output pipeline.yaml --disable-caching
argo lint --offline pipeline.yaml
kfp visualize pipeline.yaml --format both -o pipeline.html
```

Use `kfp dsl compile --help` for native resource, namespace, repository, image and
schedule options. See [Run a Pipeline](run-a-pipeline.md) for submission.
