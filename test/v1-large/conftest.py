from typing import Callable

import kfp


def runner(
    pipeline_func: Callable,
    timeout: int = 600,
    arguments: dict = {},
):
    client = kfp.Client(host="http://localhost:8888")
    run = client.create_run_from_pipeline_func(
        pipeline_func,
        arguments=arguments,
    )
    result = run.wait_for_run_completion(timeout=timeout)
    assert result.run.status.lower() == "succeeded"