# Static artifact fan-in

## Summary

This proposal adds an ordered artifact fan-in primitive to Kubeflow Pipelines.
It lets a task that declares an artifact-list input consume compatible outputs
from multiple independently instantiated upstream tasks.

```python
@dsl.pipeline
def ensemble_pipeline():
    svc = train_svc()
    xgb = train_xgb()
    lr = train_lr()

    evaluate_models(models=[svc.output, xgb.output, lr.output])
```

The SDK also exposes the equivalent explicit form:

```python
evaluate_models(
    models=dsl.UnionChannel(svc.output, xgb.output, lr.output))
```

## Motivation

`dsl.Collected` collects repeated outputs from one task inside a
`dsl.ParallelFor`. `dsl.OneOf` selects one available output from mutually
exclusive branches. Neither represents the common ensemble shape where
different components independently produce artifacts that one downstream task
must receive as a list.

Today authors must define one input per producer. That makes reusable evaluator,
selection, and ensemble components depend on a fixed number of models and adds
manual wiring whenever the producer set changes.

## Goals

- Accept a static Python list of individual artifact channels for an
  `Input[List[T]]` component input.
- Preserve the order declared by the pipeline author.
- Require all sources to have the same artifact type as each other and the
  downstream input.
- Make every producer a normal DAG dependency.
- Support graph-component boundaries, the backend driver, and KFP Local.
- Provide an explicit `dsl.UnionChannel` form for libraries that build inputs
  programmatically.

## Non-goals

- Changing the number of tasks or sources at runtime.
- Combining parameters and artifacts in one list.
- Flattening nested fan-in, `dsl.Collected`, or `dsl.OneOf` channels.
- Defining partial-success behavior when an upstream producer fails.

Existing task failure semantics remain unchanged: a normal downstream task is
not scheduled unless all of its required upstream tasks succeed.

## Pipeline IR

`TaskInputsSpec.InputArtifactSpec` gains `artifact_sources`, an ordered list of
ordinary `InputArtifactSpec` values:

```proto
message ArtifactSourcesSpec {
  repeated InputArtifactSpec artifacts = 1;
}

oneof kind {
  TaskOutputArtifactSpec task_output_artifact = 3;
  string component_input_artifact = 4;
  ArtifactSourcesSpec artifact_sources = 6;
}
```

Reusing `InputArtifactSpec` keeps parent-component input forwarding and task
output resolution identical to existing single-artifact inputs. The driver
resolves entries from left to right and concatenates their artifact lists.
This also handles a graph input that already contains more than one artifact.

The representation is additive. Existing compiled pipelines do not change, and
old input kinds retain their field numbers.

## SDK behavior

When a component input is an artifact list and its argument is a Python list
containing only individual artifact channels, `PipelineTask` normalizes it to a
`UnionArtifact`. A mixed list of artifact channels and literal values is
rejected with a type error.

`UnionArtifact` stores its source channels for compilation and dependency
analysis. It is itself typed as an artifact-list channel, so existing component
input compatibility checks still validate the downstream artifact type.

The following are rejected at compile time:

- an empty `dsl.UnionChannel`;
- different artifact types in one union;
- artifact-list sources, including `dsl.Collected`;
- nested `dsl.UnionChannel` or `dsl.OneOf` sources.

These restrictions make ordering and flattening unambiguous. Additional
compositions can be proposed independently if their runtime semantics are
defined.

## Runtime behavior

The backend driver recursively resolves every `artifact_sources` entry, appends
the resulting artifacts in declaration order, and passes one `ArtifactList` to
the executor. KFP Local performs the same ordered resolution in its in-memory IO
store.

No new Kubernetes resource or ML Metadata schema is required.

## Testing

The implementation includes tests for:

- SDK validation and ordering;
- raw-list normalization and explicit `dsl.UnionChannel` compilation;
- dependency creation for every producer;
- backend resolution and flattening of parent artifact lists;
- KFP Local resolution order.

## Alternatives considered

### Use `dsl.ParallelFor` and `dsl.Collected`

This works only when one component is repeated. It does not model independent
components with different implementations or signatures.

### Add a synthetic collector task

A collector container could copy artifact metadata, but it adds a runtime pod,
requires storage-specific behavior, and obscures the true DAG dependencies.
Fan-in is a graph/data-binding operation and belongs in the IR.

### Encode artifact references as a parameter list

Parameters cannot preserve artifact typing, lineage, metadata, or executor
artifact materialization semantics.

## Compatibility and rollout

The feature requires an SDK/compiler and runtime that understand
`artifact_sources`. Pipelines using the new field should fail clearly on older
runtimes rather than silently dropping inputs. The API addition should be
documented in the release notes when shipped.

This proposal addresses [kubeflow/pipelines#11930](https://github.com/kubeflow/pipelines/issues/11930)
and the related iterable-channel discussion in
[kubeflow/pipelines#10840](https://github.com/kubeflow/pipelines/issues/10840).
