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

## Output location

| Request type | Where to write |
|---|---|
| New repo sample | `kfp-generator_workspace/iteration_1/<name>.py` |
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

Search the repository (for example, using grep/ripgrep under `samples/core/`) to find reference examples for the specific KFP feature or pattern you need (such as conditions, loops, retries, caching, PVC, or resource specs).

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
- Pass **artifacts** with typed paths: `output: dsl.OutputPath(Dataset)`, `input_artifact: dsl.Input[Dataset]`.
- Use `.after()` when you need explicit ordering without a data dependency (see `samples/core/kubernetes_pvc/kubernetes_pvc.py` and `samples/core/execution_order/execution_order.py`).

## Compile and verify

After generating a `.py` pipeline, run the skill evaluation script to validate compilation and execution:

### 1. Static Compilation Verification
```bash
python kfp-generator/scripts/eval_output.py path/to/pipeline.py
```

The script:
- Uses the repo `.venv` Python when available
- Executes the pipeline file to trigger KFP v2 YAML compilation
- Confirms the companion `.yaml` pipeline spec was written
- Prints compiler tracebacks to stderr on failure so the agent can fix and re-run

### 2. Eval-Driven Suite (agentskills.io standard)
To evaluate the overall output quality of this skill across the standardized test cases:

```bash
# Run the full benchmark evaluation suite for iteration 1:
python .agents/kfp-generator/main.py --suite --iteration 1
```

This runs the eval suite defined in `evals/evals.json`, creates the workspace under `workspaces/iteration-1/`, executes assertion grading across `with_skill` vs `without_skill` configurations, and generates `benchmark.json` with aggregate metrics (pass rate, duration, tokens, delta) and `feedback.json` for human review.

## Generation checklist

- [ ] Uses KFP v2 DSL (`dsl.component` / `dsl.container_component` / `dsl.pipeline`)
- [ ] Pipeline and components have docstrings
- [ ] Pipeline parameters have sensible defaults where appropriate
- [ ] `if __name__ == '__main__'` compiles to YAML
- [ ] `scripts/eval_output.py` passes static YAML compilation check for the generated `.py`
- [ ] (When backend is available) `scripts/eval_output.py --run` succeeds and `get_run` confirms `SUCCEEDED` state
- [ ] Matches an existing sample pattern when the feature already exists in `samples/core/`
- [ ] No SDK-only imports inside component function bodies (runtime installs `kfp` with `--no-deps`)

## Anti-patterns

- Do **not** generate v1 `ContainerOp`, `dsl.get_pipeline_conf()`, or `@dsl.pipeline` with `pipeline_func=` keyword.
- Do **not** import heavy SDK modules inside `@dsl.component` bodies unless needed at runtime.

## Related docs

- `AGENTS.md` — monorepo architecture, testing, local execution
- `samples/README.md` — sample conventions and run instructions
- `sdk/python/kfp/dsl/` — DSL implementation and tests
