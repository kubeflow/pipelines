import os
import utils
import pytest

from utils import argo_utils


def compile_and_run_pipeline(
    client,
    experiment_id,
    pipeline_definition,
    input_params,
    output_file_dir,
    pipeline_name,
):

    env_value = os.environ.copy()
    env_value["PYTHONPATH"] = f"{os.getcwd()}:" + os.environ.get("PYTHONPATH", "")
    pipeline_path = os.path.join(output_file_dir, pipeline_name)
    utils.run_command(
        f"dsl-compile --py {pipeline_definition} --output {pipeline_path}.yaml",
        env=env_value,
    )
    run = client.run_pipeline(
        experiment_id, pipeline_name, f"{pipeline_path}.yaml", input_params
    )
    return run.id


def wait_for_job_completion(client, run_id, timeout):
    response = client.wait_for_run_completion(run_id, timeout)
    status = response.run.status.lower() == "succeeded"
    return status


def get_workflow_json(client, run_id):
    # API not in readthedocs
    # Refer: https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/_client.py#L663
    return client._get_workflow_json(run_id)


def compile_run_monitor_pipeline(
    client,
    experiment_id,
    pipeline_definition,
    input_params,
    output_file_dir,
    pipeline_name,
    timeout,
    check=True,
):
    run_id = compile_and_run_pipeline(
        client,
        experiment_id,
        pipeline_definition,
        input_params,
        output_file_dir,
        pipeline_name,
    )
    status = wait_for_job_completion(client, run_id, timeout)
    workflow_json = get_workflow_json(client, run_id)

    if check and not status:
        argo_utils.print_workflow_logs(workflow_json["metadata"]["name"])
        pytest.fail(f"Test Failed: {pipeline_name}")

    return run_id, status, workflow_json
