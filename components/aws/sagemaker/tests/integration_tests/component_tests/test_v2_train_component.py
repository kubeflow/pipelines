import pytest
import os
import utils
from utils import kfp_client_utils
from utils import minio_utils
from utils import ack_utils
import ast


@pytest.mark.parametrize(
    "test_file_dir",
    [pytest.param("resources/config/ack-training-job", marks=pytest.mark.canary_test)],
)
def test_trainingjobV2(kfp_client, experiment_id, test_file_dir):
    k8s_client = ack_utils.k8s_client()
    test_file_dir = "resources/config/ack-training-job"
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "ack-training-job.yaml"),
        )
    )
    input_job_name = utils.generate_random_string(10) + "-trn-job"
    test_params["Arguments"]["training_job_name"] = input_job_name

    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )
    outputs = {
        "sagemaker-trainingjob": [
            "model_artifacts",
        ]
    }

    # Get output data
    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )
    model_artifact = utils.read_from_file_in_tar(
        output_files["sagemaker-trainingjob"]["model_artifacts"]
    )

    # Verify Training job was successful on SageMaker
    print(f"training job name: {input_job_name}")
    train_response = ack_utils.describe_training_job(k8s_client, input_job_name)
    assert train_response["status"]["trainingJobStatus"] == "Completed"

    # Verify model artifacts output was generated from this run
    model_uri = ast.literal_eval(model_artifact)["s3ModelArtifacts"]
    print(f"model_artifact_url: {model_uri}")
    assert model_uri == train_response["status"]["modelArtifacts"]["s3ModelArtifacts"]
    assert input_job_name in model_uri

    utils.remove_dir(download_dir)


def test_terminate_trainingjob(kfp_client, experiment_id):
    k8s_client = ack_utils.k8s_client()
    test_file_dir = "resources/config/ack-training-job"
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated_terminate"))

    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "woof.yaml"),
        )
    )
    input_job_name = utils.generate_random_string(4) + "-terminate-job"
    test_params["Arguments"]["training_job_name"] = input_job_name

    run_id, _, _ = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        60,
        "running",
    )
    print(f"Terminating run: {run_id} where Training job_name: {input_job_name}")
    kfp_client_utils.terminate_run(kfp_client, run_id)
    desiredStatuses = ["Stopping", "Stopped"]
    training_status_reached = ack_utils.wait_for_trainingjob_status(
        k8s_client, input_job_name, desiredStatuses, 10, 6
    )
    assert training_status_reached

    utils.remove_dir(download_dir)
