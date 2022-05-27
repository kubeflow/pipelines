import asyncio
from typing import Callable, Dict, List
import pytest
import os
import yaml
from kfp import client
import kfp_server_api
from sdk.python.kfp.client.client import RunPipelineResult

HOST = 'https://1c0b05d19abcd1c1-dot-us-central1.pipelines.googleusercontent.com/'
MINUTE = 60


def load_config() -> List[Dict[str, str]]:
    with open(os.path.join('samples', 'test', 'config.yaml'), 'r') as f:
        return yaml.safe_load(f)


def import_pipeline_func(module_name: str, function_name: str) -> Callable:
    """Import a pipeline function from a module.

    Args:
        module_name: The name of the module.
        function_name: The name of the function.

    Returns:
        Callable: The pipeline function.
    """
    module = __import__(module_name, fromlist=[function_name])
    return getattr(module, function_name)


def run(module_name: str, function_name: str) -> RunPipelineResult:
    """Run the pipeline

    Args:
        module_name: The name of the module.
        function_name: The name of the function.

    Returns:
        RunPipelineResult: The run result.
    """
    pipeline_func = import_pipeline_func(module_name, function_name)
    return client.Client(host=HOST).create_run_from_pipeline_func(pipeline_func)


def wait(run_result: RunPipelineResult) -> kfp_server_api.ApiRun:
    timeout_mins = 10

    run_response = run_result.wait_for_run_completion(timeout_mins * MINUTE)
    return run_response.run


config = load_config()

config = [config[0]]
sample = config[0]

name = config[0]['name']
config[0]['path'] = '.'.join(
    config[0]['path'].split('.')[:-1]) + '.condition_v2'
path = config[0]['path']

keys = list(config[0].keys())
config = [tuple(dictionary.values()) for dictionary in config]


@pytest.mark.asyncio_cooperative
@pytest.mark.parametrize(keys, config)
async def test(path: str, name: str):
    """Runs all samples and test that they succeed.

    Args:
        module_name: The name of the module.
        function_name: The name of the function.
    """
    loop = asyncio.get_running_loop()

    run_result = await loop.run_in_executor(None, run, path, name)
    url = f"{HOST}/#/runs/details/{run_result.run_id}"
    print(f'Running pipeline {name} from {path}:  {url}.')

    api_run = await loop.run_in_executor(None, wait, run_result)
    assert api_run.status == 'Succeeded', f'Pipeline {path}.{name} failed. More info: {url}.'
