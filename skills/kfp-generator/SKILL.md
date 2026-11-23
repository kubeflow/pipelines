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

## Before generating

1. **Bootstrap & locate the KFP reference checkout first:** Run `<skill-dir>/main.py` (for a project-local install, `.agents/skills/kfp-generator/main.py`) or ensure `KFP_REFERENCE_DIR` / `KFP_SAMPLES_DIR` is set so the upstream KFP reference checkout exists in a dedicated directory (default to `<skill-dir>/.kfp-ref`).
2. **Read reference samples from the KFP reference checkout:** Read the closest existing sample under `$KFP_SAMPLES_DIR/core` (or `<skill-dir>/.kfp-ref/samples/core` / `samples/core/` if already inside a `kubeflow/pipelines` repo) for the feature being used.
3. **Prefer v2 DSL:** Use `@dsl.component`, `@dsl.container_component`, and `@dsl.pipeline` — not v1 `ContainerOp` or `load_component_from_url` unless maintaining legacy code.

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

Resolve bundled scripts relative to the skill directory containing this `SKILL.md` (for a project-local install, `.agents/skills/kfp-generator/`):

### 1. Static Compilation Verification
```bash
python3 .agents/skills/kfp-generator/scripts/eval_output.py path/to/pipeline.py
```

### 2. Evaluation Procedure
To evaluate skill output quality:

1. Review the test prompts and expected checks in `evals/evals.json`.
2. Generate `.py` pipelines with and without the skill (for example, in `workspaces/iteration-1/`).
3. Run `eval_output.py` on each generated `.py` pipeline to verify compilation.
4. Compare outputs manually against `evals/evals.json` and the **Generation checklist**, recording findings in your own notes or review summary.


## Generation checklist

- [ ] Uses KFP v2 DSL (`dsl.component` / `dsl.container_component` / `dsl.pipeline`)
- [ ] Pipeline and components have docstrings
- [ ] Pipeline parameters have sensible defaults where appropriate
- [ ] `if __name__ == '__main__'` compiles to YAML
- [ ] `scripts/eval_output.py` passes static YAML compilation check for the generated `.py`
- [ ] Matches an existing sample pattern when the feature already exists in `samples/core/`
- [ ] No SDK-only imports inside component function bodies (runtime installs `kfp` with `--no-deps`)

## Anti-patterns

- Do **not** generate v1 `ContainerOp`, `dsl.get_pipeline_conf()`, or `@dsl.pipeline` with `pipeline_func=` keyword.
- Do **not** import heavy SDK modules inside `@dsl.component` bodies unless needed at runtime.

## Related docs

- `AGENTS.md` — monorepo architecture, testing, local execution
- `samples/README.md` — sample conventions and run instructions
- `sdk/python/kfp/dsl/` — DSL implementation and tests
