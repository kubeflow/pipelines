import os
import json
import utils


def compile_and_run_pipeline(client, experiment_id, test_file_dir, pipeline_name):

    env_value = os.environ.copy()
    env_value["PYTHONPATH"] = f"{os.getcwd()}:" + os.environ.get("PYTHONPATH", "")
    pipeline_path = os.path.join(test_file_dir, pipeline_name)
    utils.run_command(
        f"dsl-compile --py {pipeline_path}.py --output {pipeline_path}.yaml",
        env=env_value,
    )
    run = client.run_pipeline(experiment_id, pipeline_name, f"{pipeline_path}.yaml")
    return run.id


def wait_for_job_completion(client, run_id, timeout):
    response = client.wait_for_run_completion(run_id, timeout)
    status = response.run.status.lower() == "succeeded"
    return status


def compile_run_monitor_pipeline(
    client, experiment_id, test_file_dir, pipeline_name, timeout
):
    run_id = compile_and_run_pipeline(
        client, experiment_id, test_file_dir, pipeline_name
    )
    status = wait_for_job_completion(client, run_id, timeout)
    return run_id, status


def get_workflow_json(client, run_id):
    # https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/_client.py#L663
    return client._get_workflow_json(run_id)


def get_workflow_name(client, run_id):
    response = client.get_run(run_id)
    return json.loads(response.pipeline_runtime.workflow_manifest)["metadata"]["name"]
