kfp_endpoint = None

import datetime
import time

import kfp
from kfp.components import create_component_from_func


@create_component_from_func
def do_work_op(seconds: int = 30) -> str:
    import datetime
    import time
    print(f"Working for {seconds} seconds.")
    for i in range(seconds):
        print(f"Working: {i}.")
        time.sleep(1)
    print("Done.")
    return datetime.datetime.now().isoformat()


def caching_pipeline(seconds: int = 30):
    # All outputs of successfull executions are cached
    work_task = do_work_op(seconds)


# Test 1
# Running the pipeline for the first time.
# The pipeline performs work and the results are cached.
# The pipeline run time should be ~30 seconds.
print("Starting test 1")
start_time = datetime.datetime.now()
kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
    caching_pipeline,
    arguments=dict(seconds=30),
).wait_for_run_completion(timeout=999)
elapsed_time = datetime.datetime.now() - start_time
print(f"Total run time: {int(elapsed_time.total_seconds())} seconds")


# Test 2
# Running the pipeline the second time.
# The pipeline should reuse the cached results and complete faster.
# The pipeline run time should be <30 seconds.
print("Starting test 2")
start_time = datetime.datetime.now()
kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
    caching_pipeline,
    arguments=dict(seconds=30),
).wait_for_run_completion(timeout=999)
elapsed_time = datetime.datetime.now() - start_time
print(f"Total run time: {int(elapsed_time.total_seconds())} seconds")

if elapsed_time.total_seconds() > 30:
    raise RuntimeError("The cached execution was not re-used or pipeline run took to long to complete.")


# Test 3
# For each task we can specify the maximum cached data staleness.
# For eample: task.execution_options.caching_strategy.max_cache_staleness = "P5s" or = "P14d"
# Cached results that are older than the specified time span, are not reused.
# In this case, the pipeline should not reuse the cached result, since they will be stale.

def caching_pipeline3(seconds: int = 30):
    # All outputs of successfull executions are cached
    work_task = do_work_op(seconds)
    work_task.execution_options.caching_strategy.max_cache_staleness = 'P5s'  # = 5 seconds

# Waiting for some time for the cached data to become stale:
time.sleep(10)
print("Starting test 3")
start_time = datetime.datetime.now()
kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
    caching_pipeline3,
    arguments=dict(seconds=30),
).wait_for_run_completion(timeout=999)
elapsed_time = datetime.datetime.now() - start_time
print(f"Total run time: {int(elapsed_time.total_seconds())} seconds")

if elapsed_time.total_seconds() < 30:
    raise RuntimeError("The cached execution was apparently re-used, but that should not happen.")
