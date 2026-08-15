# KEP-11930: Static Fan-In of Pipeline Task Outputs

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [SDK User Experience](#sdk-user-experience)
  - [Behavior](#behavior)
  - [Backward Compatibility and Migration](#backward-compatibility-and-migration)
- [Design Details](#design-details)
  - [Pipeline Specification](#pipeline-specification)
  - [SDK and Backend](#sdk-and-backend)
  - [KFP Local Considerations](#kfp-local-considerations)
  - [Frontend Considerations](#frontend-considerations)
  - [Test Plan](#test-plan)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Kubeflow Pipelines (KFP) can collect outputs from iterations of one task inside
`dsl.ParallelFor`, but it cannot directly combine outputs from independently
instantiated tasks into one ordered `List[...]` input.

This KEP proposes extending `dsl.Collected` to support static fan-in:

```python
# Existing ParallelFor behavior.
dsl.Collected(task.output)

# Proposed static fan-in behavior.
dsl.Collected(task_a.output, task_b.output, task_c.output)
dsl.Collected([task_a.output, task_b.output, task_c.output])
```

Static fan-in combines pipeline channels known during compilation. It does not
make the submitted pipeline graph dynamic.

## Motivation

Ensemble evaluation, model selection, and similar machine-learning workflows
often produce several models using different components and then pass those
models to one evaluator:

```python
@dsl.component
def evaluate_models(models: dsl.Input[List[dsl.Model]]):
    ...


@dsl.pipeline
def pipeline():
    svc = train_svc()
    xgb = train_xgb()
    lr = train_lr()

    evaluate_models(
        models=...,  # No direct way to construct this List[Model].
    )
```

`dsl.Collected(task.output)` currently supports a different topology: one task
output produced repeatedly inside `dsl.ParallelFor`. The current workaround is
to define a fixed number of consumer inputs, coupling the component interface to
one pipeline topology and requiring it to change whenever a producer is added
or removed.

Issue [#10840](https://github.com/kubeflow/pipelines/issues/10840) discussed
discovering pipeline channels inside ordinary Python containers. Maintainer
feedback noted that Python control flow in a pipeline function runs during
compilation and should not be confused with runtime pipeline dynamism. Issue
[#11930](https://github.com/kubeflow/pipelines/issues/11930) narrows the need to
combining a compile-time-known set of outputs.

### Goals

1. Allow independent parameter or artifact outputs to feed one compatible
   `List[...]` component input.
2. Preserve the order provided by the pipeline author.
3. Infer a dependency from the consumer to every producer.
4. Provide explicit DSL syntax for static fan-in.
5. Preserve existing `dsl.Collected(task.output)` behavior.
6. Support remote execution and KFP Local.

### Non-Goals

1. Constructing or changing the pipeline graph at runtime.
2. Selecting an unknown number of producers from runtime data.
3. Replacing `dsl.ParallelFor`.
4. Combining incompatible parameter or artifact types.
5. Supporting arbitrary nested Python containers.
6. Supporting static collections as pipeline outputs in the initial
   implementation.

## Proposal

### SDK User Experience

The sequence form supports the expression requested in issue #11930:

```python
@dsl.pipeline
def pipeline():
    models = []
    for trainer in [train_svc, train_xgb, train_lr]:
        models.append(trainer().output)

    evaluate_models(models=dsl.Collected(models))
```

The Python loop executes while compiling the pipeline. The submitted graph
contains three concrete producer tasks and one consumer task.

The variadic form supports explicitly declared producers:

```python
@dsl.pipeline
def pipeline():
    svc = train_svc()
    xgb = train_xgb()
    lr = train_lr()

    evaluate_models(
        models=dsl.Collected(svc.output, xgb.output, lr.output),
    )
```

The same API supports parameter outputs.

### Behavior

`dsl.Collected` selects its behavior from the constructor call:

| Expression | Meaning |
| --- | --- |
| `dsl.Collected(channel)` | Existing `ParallelFor` collection |
| `dsl.Collected(channel_a, channel_b)` | Static fan-in |
| `dsl.Collected([channel_a, channel_b])` | Static fan-in |
| `dsl.Collected([channel])` | One-element static fan-in |

The single-channel form must retain its existing meaning for backward
compatibility. Wrapping one channel in a sequence explicitly requests static
fan-in.

Static collection members must:

- be pipeline channels;
- all be parameters or all be artifacts; and
- have compatible channel types.

The SDK rejects empty collections because their type cannot be inferred. It
also rejects `dsl.OneOf` and nested `dsl.Collected` values because those
constructs have different control-flow semantics.

### Backward Compatibility and Migration

This feature will ship as a coordinated SDK, pipeline specification, backend,
and KFP Local change in KFP v3.

Existing pipelines require no migration:

- `dsl.Collected(task.output)` retains its current behavior;
- existing compiled pipeline specifications remain valid;
- scalar inputs and constant list parameters are unchanged; and
- component interfaces continue to use `List[T]` and
  `Input[List[ArtifactType]]`.

Static fan-in is additive and only affects pipelines that opt into the new
constructor forms.

## Design Details

### Pipeline Specification

Each task input currently identifies one source, such as an upstream task output
or a parent component input. Static fan-in requires one task input to retain an
ordered set of those references.

Add list variants to the existing task-input definitions. Each item reuses the
existing scalar input reference:

```protobuf
message InputArtifactSpec {
  message ArtifactListSpec {
    repeated InputArtifactSpec items = 1;
  }

  oneof kind {
    // Existing kinds omitted.
    ArtifactListSpec artifact_list = 6;
  }
}

message InputParameterSpec {
  message ParameterListSpec {
    repeated InputParameterSpec items = 1;
  }

  oneof kind {
    // Existing kinds omitted.
    ParameterListSpec parameter_list = 6;
  }
}
```

Reusing the existing input reference types preserves order and supports channels
surfaced through nested DAG boundaries.

### SDK and Backend

The SDK compiler validates and normalizes the collection, registers each member
as an input of the consumer, and emits the ordered list. The backend compiler
adds every referenced producer as an implicit dependency. At runtime, the
driver resolves each item through the existing scalar input paths into an
ordered parameter list or artifact list.

### KFP Local Considerations

KFP Local resolves the new task-input variants using the same ordering as the
remote driver. Both parameter and artifact static fan-in must work with local
pipeline execution.

### Frontend Considerations

No frontend changes are required. Static fan-in only changes task-to-task
wiring in the compiled pipeline. It does not add a pipeline-level runtime input
or change the run creation experience.

### Test Plan

SDK tests will cover existing single-channel behavior, each new constructor
form, ordering, nested DAG boundaries, and invalid collections. Backend tests
will cover dependency inference and ordered resolution for parameter and
artifact lists. KFP Local tests will execute both input kinds.

An integration test will execute the issue #11930 example with three independent
model trainers and one `Input[List[Model]]` evaluator.

## Drawbacks

1. `dsl.Collected` gains a second, related meaning.
2. A one-element static collection requires `dsl.Collected([channel])` because
   `dsl.Collected(channel)` must retain its existing loop behavior.
3. The feature requires coordinated SDK, pipeline specification, backend, and
   KFP Local changes.

## Alternatives

### Accept Raw Python Lists of Channels

```python
evaluate_models(models=[svc.output, xgb.output, lr.output])
```

This is shorter, but makes ordinary Python containers context-sensitive and
blurs the distinction between Python executed during compilation and explicit
KFP DSL constructs. It also raises broader questions about tuples,
dictionaries, nested containers, and mixed constants and channels.

An explicit `dsl.Collected` expression communicates that the values participate
in pipeline graph construction.

### Introduce a New `FanIn` or `UnionChannel` API

```python
evaluate_models(
    models=dsl.FanIn(svc.output, xgb.output, lr.output),
)
```

A dedicated API is unambiguous and avoids extending a loop-oriented class.
However, it introduces another DSL concept that produces the same ordered list
contract as `dsl.Collected`. `UnionChannel` may also imply set-like semantics,
while this feature preserves order and duplicates.

This is the preferred alternative if maintainers do not want
`dsl.Collected` to support both loop collection and static fan-in.

### Continue Using Separate Inputs

Separate inputs work today but couple component signatures to a fixed number of
producers and do not support reusable list-consuming components.

## Implementation History

- 2024-12-30: Related iterable channel request discussed in issue #10840.
- 2025-06-26: Static multi-producer fan-in requested in issue #11930.
- 2026-08-15: Initial KEP proposed for KFP v3.
