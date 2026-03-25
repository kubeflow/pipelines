"""Pipeline 4: Two components with nested DAG using ParallelFor and use_secret_as_env.

Tests the most complex scenario: an outer task produces a secret name,
and an inner ParallelFor loop uses that secret name via use_secret_as_env.
This tests cross-DAG taskOutputParameter rewriting.
"""
from kfp import dsl, compiler
from kfp.kubernetes import use_secret_as_env


@dsl.component(base_image="python:3.11")
def emit_secret_name() -> str:
    """Emits the secret name dynamically."""
    return "test-secret-1"


@dsl.component(base_image="python:3.11")
def worker_component(item: str) -> str:
    import os
    secret_val = os.environ.get("MY_SECRET_KEY", "not-set")
    print(f"Item: {item}, Secret value: {secret_val}")
    return secret_val


@dsl.pipeline(name="pipeline-4-nested-parallel-for-secret")
def pipeline_nested_parallel_for_secret():
    """Pipeline with outer task output used as secret name inside ParallelFor."""
    secret_name_task = emit_secret_name()
    with dsl.ParallelFor(items=["x", "y"], parallelism=1) as item:
        task = worker_component(item=item)
        use_secret_as_env(
            task,
            secret_name=secret_name_task.output,
            secret_key_to_env={
                "username": "MY_SECRET_KEY",
            },
        )


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_nested_parallel_for_secret,
        package_path="nested_parallel_for_secret.yaml",
    )
    print("Compiled nested_parallel_for_secret.yaml")
