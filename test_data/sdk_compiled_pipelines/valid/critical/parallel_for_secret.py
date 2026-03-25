"""Pipeline 2: Single component inside ParallelFor using use_secret_as_env.

This is the exact bug scenario from RHOAIENG-54342 / kubeflow/pipelines#13078.
The secret name comes from a pipeline parameter and is used inside a
dsl.ParallelFor loop. Without the fix, the driver fails with:
  "parent DAG does not have input parameter my_secret_name"
"""
from kfp import dsl, compiler
from kfp.kubernetes import use_secret_as_env


@dsl.component(base_image="python:3.11")
def example_component(item: str) -> str:
    import os
    secret_val = os.environ.get("MY_SECRET_KEY", "not-set")
    print(f"Item: {item}, Secret value: {secret_val}")
    return secret_val


@dsl.pipeline(name="pipeline-2-parallel-for-secret")
def pipeline_parallel_for_secret(my_secret_name: str = "test-secret-1"):
    """Pipeline with 1 component inside ParallelFor using use_secret_as_env."""
    with dsl.ParallelFor(items=["a", "b", "c"], parallelism=1) as item:
        task = example_component(item=item)
        use_secret_as_env(
            task,
            secret_name=my_secret_name,
            secret_key_to_env={
                "username": "MY_SECRET_KEY",
            },
        )


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline_parallel_for_secret,
        package_path="parallel_for_secret.yaml",
    )
    print("Compiled parallel_for_secret.yaml")
