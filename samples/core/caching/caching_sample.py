kfp_endpoint = None

import datetime
import time

import kfp as kfp
from kfp import dsl


@dsl.component
def do_work_op(seconds: float = 60) -> str:
    import datetime
    import time
    print(f"Working for {seconds} seconds.")
    for i in range(int(seconds)):
        print(f"Working: {i}.")
        time.sleep(1)
    print("Done.")
    return datetime.datetime.now().isoformat()


@kfp.dsl.pipeline(name='caching-pipeline')
def caching_pipeline(seconds: float = 60):
    # All outputs of successful executions are cached
    work_task = do_work_op(seconds=seconds)


# Test 1
# Running the pipeline for the first time.
# The pipeline performs work and the results are cached.
# The pipeline run time should be ~60 seconds.
print("Starting test 1")
start_time = datetime.datetime.now()
kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
    caching_pipeline,
    arguments=dict(seconds=60),
).wait_for_run_completion(timeout=999)
elapsed_time = datetime.datetime.now() - start_time
print(f"Total run time: {int(elapsed_time.total_seconds())} seconds")

# Test 2
# Running the pipeline the second time.
# The pipeline should reuse the cached results and complete faster.
# The pipeline run time should be <60 seconds.
print("Starting test 2")
start_time = datetime.datetime.now()
kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
    caching_pipeline,
    arguments=dict(seconds=60),
).wait_for_run_completion(timeout=999)
elapsed_time = datetime.datetime.now() - start_time
print(f"Total run time: {int(elapsed_time.total_seconds())} seconds")

if elapsed_time.total_seconds() > 60:
    raise RuntimeError("The cached execution was not re-used or pipeline run took to long to complete.")


# Test 3
# In this case, the pipeline should not reuse the cached result, since they are
# disabled.
@kfp.dsl.pipeline(name='caching-pipeline3')
def caching_pipeline3(seconds: float = 60):
    work_task = do_work_op(seconds=seconds)
    work_task.set_caching_options(enable_caching=False)


print("Starting test 3")
start_time = datetime.datetime.now()
kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(
    caching_pipeline3,
    arguments=dict(seconds=60),
).wait_for_run_completion(timeout=999)
elapsed_time = datetime.datetime.now() - start_time
print(f"Total run time: {int(elapsed_time.total_seconds())} seconds")

if elapsed_time.total_seconds() < 60:
    raise RuntimeError("The cached execution was apparently re-used, but that should not happen.")
