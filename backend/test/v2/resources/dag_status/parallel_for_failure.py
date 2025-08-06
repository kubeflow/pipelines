import kfp
from kfp import dsl


@dsl.component()
def fail_task(item: str):
    """Component that always fails."""
    import sys
    print(f"Processing {item}")
    print("This task is designed to fail for testing purposes")
    sys.exit(1)


@dsl.pipeline(name="parallel-for-failure", description="Simple ParallelFor loop that fails to test DAG status updates")
def parallel_for_failure_pipeline():
    """
    Simple ParallelFor pipeline that fails.
    """
    # ParallelFor with 3 iterations - all should fail
    with dsl.ParallelFor(items=['item1', 'item2', 'item3']) as item:
        fail_task_instance = fail_task(item=item)
