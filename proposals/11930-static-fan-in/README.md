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
`dsl.ParallelFor`, but it cannot directly bind outputs from independently
instantiated tasks to one ordered `List[...]` input.

This KEP proposes allowing a homogeneous Python list of scalar pipeline channels
as the value of a compatible list-typed component input:

```python
evaluate_models(
    models=[task_a.output, task_b.output, task_c.output],
)
```

The Python list and all of its members are known during compilation. The
compiler lowers the list to an ordered set of task-input references; this does
not make the submitted pipeline graph dynamic.

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

`dsl.Collected(task.output)` supports a different topology: one task output
produced repeatedly inside `dsl.ParallelFor`. It does not represent an ordered
list of independently instantiated task outputs. The current workaround is to
define a fixed number of consumer inputs, coupling the component interface to
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
4. Use ordinary Python list syntax for a compile-time-known list of channels.
5. Leave `dsl.Collected` dedicated to `dsl.ParallelFor` collection.
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

The proposed syntax directly expresses the request in issue #11930:

```python
@dsl.pipeline
def pipeline():
    models = []
    for trainer in [train_svc, train_xgb, train_lr]:
        models.append(trainer().output)

    evaluate_models(models=models)
```

The Python loop executes while compiling the pipeline. The submitted graph
contains three concrete producer tasks and one consumer task.

Producers can also be declared explicitly:

```python
@dsl.pipeline
def pipeline():
    svc = train_svc()
    xgb = train_xgb()
    lr = train_lr()

    evaluate_models(
        models=[svc.output, xgb.output, lr.output],
    )
```

The same API supports parameter outputs.

### Behavior

The compiler distinguishes literal lists from static channel lists by their
members and the destination component input:

| Expression | Meaning |
| --- | --- |
| `values=["a", "b"]` | Existing constant list parameter |
| `values=[task_a.output, task_b.output]` | Static parameter fan-in |
| `models=[task_a.output, task_b.output]` | Static artifact fan-in |
| `models=[task.output]` | One-element static artifact fan-in |
| `dsl.Collected(task.output)` | Existing `ParallelFor` collection |

A non-empty list containing only pipeline channels is a static channel list.
A list containing no channels follows existing literal-list behavior. A list
that mixes channels with constants is rejected in the initial implementation.
The destination must be a compatible list-typed input.

Static collection members must:

- represent scalar values rather than list-valued parameter or artifact
  channels;
- be independently valid scalar references at the consumer's control-flow
  scope;
- all be parameters or all be artifacts; and
- have compatible channel types.

An empty list contains no channels and therefore follows existing parameter
literal validation; it does not create a static channel list. The SDK rejects
`dsl.OneOf`, `dsl.Collected`, nested lists, and other list-valued members because
those constructs require flattening or control-flow semantics beyond static
scalar binding.

Existing control-flow restrictions apply to every member independently. A
producer in the consumer's current loop iteration and an outer-to-inner
broadcast are valid. Inner-to-outer and sibling-loop references are not static
fan-in and are rejected, as are existing illegal escapes from `dsl.If` and
`dsl.ExitHandler`. Parameter selectors from `LoopArgumentVariable` are retained
on the corresponding list item.

### Backward Compatibility and Migration

This feature will ship as a coordinated SDK, pipeline specification, backend,
and KFP Local change in KFP v3.

Existing pipelines require no migration:

- `dsl.Collected(task.output)` remains unchanged;
- existing compiled pipeline specifications remain valid;
- scalar inputs and Python lists containing only ordinary values are unchanged;
  and
- component interfaces continue to use `List[T]` and
  `Input[List[ArtifactType]]`.

Static fan-in is additive and only affects a list argument that contains
pipeline channels and targets a compatible list input.

## Design Details

### Pipeline Specification

Each task input currently identifies one source, such as an upstream task output
or a parent component input. Static fan-in requires one task input to retain an
ordered set of those references.

Add list variants to the existing task-input definitions. List items use
non-recursive scalar reference messages so directly submitted pipeline
specifications cannot express nested lists or unrelated input kinds:

```protobuf
message InputArtifactSpec {
  message ArtifactListItemSpec {
    oneof kind {
      TaskOutputArtifactSpec task_output_artifact = 1;
      string component_input_artifact = 2;
    }
  }

  message ArtifactListSpec {
    repeated ArtifactListItemSpec items = 1;
  }

  oneof kind {
    // Existing kinds omitted.
    ArtifactListSpec artifact_list = 6;
  }
}

message InputParameterSpec {
  message ParameterListItemSpec {
    oneof kind {
      TaskOutputParameterSpec task_output_parameter = 1;
      string component_input_parameter = 2;
    }

    string parameter_expression_selector = 3;
  }

  message ParameterListSpec {
    repeated ParameterListItemSpec items = 1;
  }

  oneof kind {
    // Existing kinds omitted.
    ParameterListSpec parameter_list = 6;
  }
}
```

The item messages intentionally exclude runtime constants, task-final-status
inputs, nested list variants, and list-valued source references. A parameter
item selector is evaluated before the selected value is appended to the result.
The outer `InputParameterSpec.parameter_expression_selector` is invalid when
`parameter_list` is selected.

A shared semantic validator runs for every pipeline-spec ingestion path before
driver execution. It verifies that:

- the destination input exists;
- `parameter_list` targets a `LIST` parameter;
- `artifact_list` targets an artifact input with `is_artifact_list = true`;
- every source is scalar and valid at the consumer's control-flow scope;
- all artifact sources are schema-compatible with the destination; and
- all parameter sources are homogeneous where the IR retains enough type
  information to determine that.

The pipeline IR does not retain the element type of `List[T]` parameters, so
backend validation cannot recover or enforce `T`. The SDK continues to perform
the stronger source-language type check.

### SDK and Backend

The SDK compiler inspects task arguments before literal conversion. For a
Python list containing channels, it validates the destination and members,
registers every channel as an input of the consumer, and emits the ordered list.
A list containing only ordinary Python values continues through the existing
constant-list path. Mixed channel and constant lists fail with an actionable
error instead of being partially serialized.

Channel discovery and control-flow validation apply to every member
independently, including channels surfaced through parent DAG boundaries. The
backend compiler adds every referenced producer as an implicit dependency. At
runtime, the driver resolves each item through the existing scalar input paths
into an ordered parameter list or artifact list.

KFP v3 stores artifacts and task relationships in the native KFP task and
artifact APIs rather than ML Metadata. Static artifact fan-in must preserve
authored order and duplicates through those APIs and across nested DAG
boundaries. The native artifact-task input relationship therefore records a
collection position. Hydration orders static collection members by that
position, and relationship uniqueness includes the position so a collection
such as `[a, b, a]` remains valid. Loop collection continues to use its existing
iteration identity.

### KFP Local Considerations

KFP Local resolves the new task-input variants using the same ordering as the
remote driver. Both parameter and artifact static fan-in must work with local
pipeline execution.

### Frontend Considerations

The generated TypeScript pipeline-spec bindings must be regenerated for the new
input variants. Static graph parsing must inspect each parameter-list and
artifact-list item so every producer is connected to the consumer. Artifact
members retain the existing producer-to-artifact-to-consumer visualization.
The graph represents unique dependencies rather than rendering overlapping
edges for repeated members. Artifact edge IDs include the producer task, output
key, consumer task, and consumer input name. Selecting the consumer shows the
ordered input binding, including repeated members and per-item selectors, so a
collection such as `[a, b, a]` remains inspectable without adding a synthetic
runtime node for the Python list.

Static fan-in does not add a pipeline-level runtime input or change the run
creation experience.

### Test Plan

SDK tests will cover unchanged constant lists and `dsl.Collected` behavior,
parameter and artifact channel lists, one-element lists, ordering, duplicates,
nested DAG boundaries, and invalid lists. Invalid cases include mixed constants
and channels, incompatible members, nested lists, and list-valued channels.
Control-flow tests will cover same-iteration, outer-to-inner, inner-to-outer,
sibling-loop, `dsl.If`, and `dsl.ExitHandler` references, including per-item
parameter selectors.

Backend tests will cover direct-IR validation for unsupported item kinds,
selectors, missing or scalar destinations, incompatible artifact schemas,
dependency inference, and ordered resolution. A nested remote artifact
regression will verify exact count, order, and duplicates with `[a, b, a]`
through the native task and artifact APIs. KFP Local tests will execute both
input kinds.

Frontend tests will use multiple producers with the same output key and a
repeated member, verifying every unique dependency edge and the exact ordered
binding in task details.

An integration test will execute the issue #11930 example with three independent
model trainers and one `Input[List[Model]]` evaluator.

## Drawbacks

1. Python list handling becomes dependent on whether its members are ordinary
   values or pipeline channels.
2. Mixed constant and channel lists are rejected until their desired runtime
   semantics are designed.
3. The feature requires coordinated SDK, pipeline specification, backend, and
   KFP Local changes.

## Alternatives

### Extend `dsl.Collected`

```python
evaluate_models(
    models=dsl.Collected(svc.output, xgb.output, lr.output),
)
```

This provides an explicit DSL marker without introducing a new class. However,
`dsl.Collected` currently means runtime aggregation of one output across
`dsl.ParallelFor` iterations. Extending it would make its behavior depend on
constructor shape:

```python
dsl.Collected(channel)    # Loop collection.
dsl.Collected([channel])  # One-element static fan-in.
```

That distinction is subtle for pipeline authors and code-generating agents.
Keeping `dsl.Collected` loop-specific provides a smaller and clearer authoring
contract.

### Introduce a New `FanIn` or `UnionChannel` API

```python
evaluate_models(
    models=dsl.FanIn(svc.output, xgb.output, lr.output),
)
```

A dedicated API is unambiguous and leaves `dsl.Collected` unchanged. However,
it introduces another DSL concept for a value already represented naturally by
a Python list and a `List[T]` component input. The initial feature has no
runtime assembly policy that requires a separate object.

A dedicated construct may be appropriate in the future if collection gains
semantics that Python list syntax cannot express, such as flattening a
list-valued channel, omitting missing conditional members, or applying a
minimum-success policy. Those behaviors are outside this KEP.

### Continue Using Separate Inputs

Separate inputs work today but couple component signatures to a fixed number of
producers and do not support reusable list-consuming components.

## Implementation History

- 2024-12-30: Related iterable channel request discussed in issue #10840.
- 2025-06-26: Static multi-producer fan-in requested in issue #11930.
- 2026-08-15: Initial KEP proposed for KFP v3.
