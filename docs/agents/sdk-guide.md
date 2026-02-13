# KFP SDK Guide (Python)

SDK classes, compilation flow, executor, client, local execution, and artifact types for working on `sdk/python/` and `kubernetes_platform/python/`.

**See also:** [architecture.md](architecture.md) (E2E flow), [protobuf-build.md](protobuf-build.md) (proto generation for `kfp-pipeline-spec`), [testing-ci.md](testing-ci.md) (SDK tests)

---

## SDK compilation flow

```
@dsl.pipeline decorator
  -> Pipeline context manager (pipeline_context.py) registers tasks
    -> Component factory (component_factory.py) creates ComponentSpec
      -> PipelineTask (pipeline_task.py) instantiates with PipelineChannels
        -> Compiler (compiler.py) traverses task graph
          -> pipeline_spec_builder.py builds PipelineSpec proto messages
            -> Serialized to IR YAML via protobuf JSON format
```

## Key classes and their roles

| Class | File | Purpose |
|-------|------|---------|
| `Pipeline` | `sdk/python/kfp/dsl/pipeline_context.py` | Context manager; registers tasks during pipeline definition |
| `PipelineTask` | `sdk/python/kfp/dsl/pipeline_task.py` | Represents an instantiated component; manages inputs/outputs/config |
| `PipelineChannel` | `sdk/python/kfp/dsl/pipeline_channel.py` | Abstract base for data flow between tasks (parameters and artifacts) |
| `ComponentSpec` | `sdk/python/kfp/dsl/structures.py` | Full component specification with inputs/outputs/implementation |
| `Compiler` | `sdk/python/kfp/compiler/compiler.py` | Entry point; calls `pipeline_spec_builder` to produce IR YAML |
| `Executor` | `sdk/python/kfp/dsl/executor.py` | Handles runtime artifact/parameter mapping and function invocation |
| `TasksGroup` | `sdk/python/kfp/dsl/tasks_group.py` | Base for control flow: `ParallelFor`, `Condition`, `ExitHandler` |
| `Client` | `sdk/python/kfp/client/client.py` | KFP API client for remote pipeline upload and run management |

## Key SDK files

| Component | File | Key symbols |
|-----------|------|-------------|
| Pipeline decorator | `sdk/python/kfp/dsl/pipeline_context.py` | `@pipeline`, `Pipeline` class |
| Component decorator | `sdk/python/kfp/dsl/component_decorator.py` | `@component` |
| Pipeline Task | `sdk/python/kfp/dsl/pipeline_task.py` | `PipelineTask` |
| Pipeline Channel | `sdk/python/kfp/dsl/pipeline_channel.py` | `PipelineChannel`, `PipelineParameterChannel`, `PipelineArtifactChannel` |
| Component Factory | `sdk/python/kfp/dsl/component_factory.py` | `create_component_from_func()`, `ComponentInfo` |
| Structures | `sdk/python/kfp/dsl/structures.py` | `InputSpec`, `OutputSpec`, `ComponentSpec`, `ContainerSpec` |
| Tasks Groups | `sdk/python/kfp/dsl/tasks_group.py` | `ParallelFor`, `Condition`, `ExitHandler` |
| For Loop | `sdk/python/kfp/dsl/for_loop.py` | `LoopParameterArgument`, `Collected` |
| Compiler | `sdk/python/kfp/compiler/compiler.py` | `Compiler.compile()` |
| Pipeline Spec Builder | `sdk/python/kfp/compiler/pipeline_spec_builder.py` | `build_task_spec_for_task()`, `to_protobuf_value()` |
| Compiler Utils | `sdk/python/kfp/compiler/compiler_utils.py` | `get_all_groups()`, `get_inputs_for_all_groups()` |
| Executor | `sdk/python/kfp/dsl/executor.py` | `Executor` class |
| Executor Main | `sdk/python/kfp/dsl/executor_main.py` | `executor_main()` entry point |
| Artifact Types | `sdk/python/kfp/dsl/types/artifact_types.py` | `Artifact`, `Model`, `Dataset`, `Metrics` |
| Client | `sdk/python/kfp/client/client.py` | `Client` class |
| Local Pipeline | `sdk/python/kfp/local/pipeline_orchestrator.py` | `run_local_pipeline()` |
| Subprocess Handler | `sdk/python/kfp/local/subprocess_task_handler.py` | `SubprocessTaskHandler` |
| Docker Handler | `sdk/python/kfp/local/docker_task_handler.py` | `DockerTaskHandler` |

## Local execution

### Subprocess Runner (no Docker required)

```python
from kfp import local
local.init(runner=local.SubprocessRunner())

# Run components directly
task = my_component(param="value")
print(task.output)
```

### Docker Runner (requires Docker)

```python
from kfp import local
local.init(runner=local.DockerRunner())

# Runs components in containers
task = my_component(param="value")
```

### Pipeline execution

```python
# Pipelines can be executed like regular functions
run = my_pipeline(input_param="test")

# If the pipeline has a single output:
print(run.output)

# Or, for named outputs:
print(run.outputs['<output_name>'])
```

Note: Local execution outputs are stored in `./local_outputs` by default.

Notes:

- SubprocessRunner supports only Lightweight Python Components (executes the KFP Python executor directly).
- Use DockerRunner for Container Components or when task images require containerized execution.

## Artifact types

Inheritance hierarchy:

```
Artifact (sdk/python/kfp/dsl/types/artifact_types.py)
  |- ClassificationMetrics, Dataset, HTML, Markdown, Metrics, Model, SlicedClassificationMetrics
```

## Documentation

SDK reference docs are auto-generated with Sphinx using autodoc from Python docstrings. Keep SDK docstrings user-facing and accurate, as they appear in published documentation.
