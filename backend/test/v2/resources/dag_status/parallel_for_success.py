import kfp
from kfp import dsl


@dsl.component()
def hello_world(message: str) -> str:
    """Simple component that succeeds."""
    print(f"Hello {message}!")
    return f"Processed: {message}"


@dsl.pipeline(name="parallel-for-success",
              description="Simple ParallelFor loop that succeeds to test DAG status updates")
def parallel_for_success_pipeline():
    """
    Simple ParallelFor pipeline that succeeds.
    """
    # ParallelFor with 3 iterations - all should succeed
    with dsl.ParallelFor(items=['world', 'kubeflow', 'pipelines']) as item:
        hello_task = hello_world(message=item)
