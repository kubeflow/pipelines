import os
import utils
import pytest
import time

from utils import argo_utils


def compile_and_run_pipeline(
    client,
    experiment_id,
    pipeline_definition,
    input_params,
    output_file_dir,
    pipeline_name,
):
    pipeline_path = os.path.join(output_file_dir, pipeline_name)
    utils.run_command(
        f"dsl-compile --py {pipeline_definition} --output {pipeline_path}.yaml"
    )
    run = client.run_pipeline(
        experiment_id, pipeline_name, f"{pipeline_path}.yaml", input_params
    )
    return run.id


def wait_for_job_completion(client, run_id, timeout):
    response = client.wait_for_run_completion(run_id, timeout)
    status = response.run.status.lower() == "succeeded"
    return status


def wait_for_job_to_start(client, run_id):
    time.sleep(10)
    response = client.get_run(run_id)
    status = response.run.status.lower() == "running"
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
        pytest.fail(f"Test Failed: {pipeline_name}. Run-id: {run_id}")

    return run_id, status, workflow_json


def compile_run_pipeline(
    client,
    experiment_id,
    pipeline_definition,
    input_params,
    output_file_dir,
    pipeline_name,
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
    status = wait_for_job_to_start(client, run_id)
    workflow_json = get_workflow_json(client, run_id)

    if check and not status:
        argo_utils.print_workflow_logs(workflow_json["metadata"]["name"])
        pytest.fail(f"Test Failed: {pipeline_name}. Run-id: {run_id}")
        
    return run_id, status, workflow_json
