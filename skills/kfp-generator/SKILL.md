---
name: kfp-generator
description: >-
  Generate Kubeflow Pipelines (KFP) v2 DSL pipelines and components for this
  monorepo. Use when the user asks to create, scaffold, or compile a KFP
  pipeline, component, or sample; generate pipeline Python from a description;
  or add a new entry under samples/.
---

# KFP Generator

Generate KFP v2 pipelines that match this repo's SDK, samples, and conventions.

## Before generating

1. Read the closest existing sample in `samples/core/` for the feature being used.
2. Prefer **v2 DSL** (`@dsl.component`, `@dsl.container_component`, `@dsl.pipeline`) — not v1 `ContainerOp` or `load_component_from_url` unless maintaining legacy code.
3. Reuse existing components from `components/` when possible instead of inlining container logic.

## Output location

| Request type | Where to write |
|---|---|
| New repo sample | `samples/core/<name>/<name>.py` |
| One-off / user pipeline | Path the user specifies, or current working directory |
| Tutorial / notebook | `samples/tutorials/` |

Sample convention: **directory name = file name** (e.g. `sequential/sequential.py`).

## File template

Every generated `.py` pipeline should include:

```python
from kfp import dsl, compiler


@dsl.component
def my_task(input_text: str) -> str:
    """Describe what this task does."""
    return input_text.upper()


@dsl.pipeline(name='my-pipeline', description='One-line description.')
def my_pipeline(input_text: str = 'hello'):
    my_task(input_text=input_text)


if __name__ == '__main__':
    compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
```

## Component types

### Lightweight Python component

Use for pure Python logic with no custom container image:

```python
@dsl.component
def add(a: int, b: int) -> int:
    return a + b
```

### Container component

Use when a specific image and shell command are required:

```python
@dsl.container_component
def gcs_download(url: str, output: dsl.OutputPath(str)):
    return dsl.ContainerSpec(
        image='google/cloud-sdk:279.0.0',
        command=['sh', '-c'],
        args=['gsutil cat $0 | tee $1', url, output],
    )
```

## Pipeline patterns

Map user intent to existing samples:

| Feature | Reference sample |
|---|---|
| Sequential steps | `samples/core/sequential/` |
| Parallel + join | `samples/core/parallel_join/` |
| Conditions | `samples/core/condition/condition_v2.py` |
| Nested conditions | `samples/core/condition/nested_condition.py` |
| For-loops | `samples/core/loop_static/`, `loop_parameter/`, `loop_output/` |
| Parallel loops | `samples/core/loop_parallelism/` |
| Exit handler | `samples/core/exit_handler/` |
| Retry | `samples/core/retry/` |
| Multiple outputs | `samples/core/multiple_outputs/` |
| Directory artifacts | `samples/core/output_a_directory/` |
| Resource requests | `samples/core/resource_spec/` |
| Secrets | `samples/core/secret/` |
| PVC | `samples/core/kubernetes_pvc/` |
| Caching | `samples/core/caching/` |

### Condition

```python
with dsl.Condition(flip.output == 'heads'):
    print_msg(msg='heads branch')
```

### Parallelism

```python
with dsl.ParallelFor(items=[1, 2, 3], parallelism=2) as item:
    my_task(value=item)
```

### Exit handler

```python
with dsl.ExitHandler(cleanup_task):
    main_task()
```

## Data passing rules

- Pass **parameters** via function arguments: `task_b(input=task_a.output)`.
- Pass **artifacts** with typed paths: `output: dsl.OutputPath(str)`, `input_artifact: dsl.InputPath()`.
- Avoid v1 `.apply()`; in v2, use `.after()` when you need explicit ordering without a data dependency (see `samples/core/kubernetes_pvc/kubernetes_pvc.py` and `samples/core/execution_order/execution_order.py`).

## Kubernetes config

- **CPU/memory/GPU requests**: set on component via resource decorators or `resource_spec` sample patterns.
- **All other K8s settings** (volumes, node selectors, tolerations, etc.): use `kfp-kubernetes` platform spec — see `kubernetes_platform/python/kfp/kubernetes/`.

## Compile and verify

After generating a `.py` pipeline, run the skill evaluation script:

```bash
python .cursor/skills/kfp-generator/scripts/eval_output.py path/to/pipeline.py
```

The script:

- Uses the repo `.venv` Python when available
- Executes the pipeline file to trigger KFP v2 YAML compilation
- Confirms the companion `.yaml` pipeline spec was written
- Prints compiler tracebacks to stderr on failure so the agent can fix and re-run

If validation fails, read the traceback, fix the generated pipeline, and re-run until it passes.

For SDK changes (not typical pipeline generation), run targeted tests:

```bash
pytest -v sdk/python/kfp/dsl/<relevant>_test.py
```

## Generation checklist

- [ ] Uses KFP v2 DSL (`dsl.component` / `dsl.container_component` / `dsl.pipeline`)
- [ ] Pipeline and components have docstrings
- [ ] Pipeline parameters have sensible defaults where appropriate
- [ ] `if __name__ == '__main__'` compiles to YAML
- [ ] `scripts/eval_output.py` passes for the generated `.py`
- [ ] Matches an existing sample pattern when the feature already exists in `samples/core/`
- [ ] No SDK-only imports inside component function bodies (runtime installs `kfp` with `--no-deps`)

## Anti-patterns

- Do **not** generate v1 `ContainerOp`, `dsl.get_pipeline_conf()`, or `@dsl.pipeline` with `pipeline_func=` keyword.
- Do **not** import heavy SDK modules inside `@dsl.component` bodies unless needed at runtime.
- Do **not** duplicate logic that already exists in `components/` — load or import shared components instead.

## Related docs

- `AGENTS.md` — monorepo architecture, testing, local execution
- `samples/README.md` — sample conventions and run instructions
- `sdk/python/kfp/dsl/` — DSL implementation and tests
