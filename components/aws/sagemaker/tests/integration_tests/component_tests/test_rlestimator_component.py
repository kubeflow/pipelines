import random
import string

import pytest
import os
import utils
from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils
from utils import argo_utils


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/rlestimator-training", marks=pytest.mark.canary_test
        ),
    ],
)
def test_trainingjob(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    test_params["Arguments"]["job_name"] = input_job_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["job_name"]
    )
    print(f"running test with job_name: {input_job_name}")

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
        "sagemaker-rlestimator-training-job": [
            "job_name",
            "model_artifact_url",
            "training_image",
        ]
    }
    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    # Verify Training job was successful on SageMaker
    training_job_name = utils.read_from_file_in_tar(
        output_files["sagemaker-rlestimator-training-job"]["job_name"]
    )
    print(f"training job name: {training_job_name}")
    train_response = sagemaker_utils.describe_training_job(
        sagemaker_client, training_job_name
    )
    assert train_response["TrainingJobStatus"] == "Stopped"

    # Verify model artifacts output was generated from this run
    model_artifact_url = utils.read_from_file_in_tar(
        output_files["sagemaker-rlestimator-training-job"]["model_artifact_url"]
    )
    print(f"model_artifact_url: {model_artifact_url}")
    assert model_artifact_url == train_response["ModelArtifacts"]["S3ModelArtifacts"]
    assert training_job_name in model_artifact_url

    # Verify training image output is an ECR image
    training_image = utils.read_from_file_in_tar(
        output_files["sagemaker-rlestimator-training-job"]["training_image"]
    )
    print(f"Training image used: {training_image}")
    if "ExpectedTrainingImage" in test_params.keys():
        assert test_params["ExpectedTrainingImage"] == training_image
    else:
        assert f"dkr.ecr.{region}.amazonaws.com" in training_image

    assert not argo_utils.error_in_cw_logs(
        workflow_json["metadata"]["name"]
    ), "Found the CloudWatch error message in the log output. Check SageMaker to see if the job has failed."

    utils.remove_dir(download_dir)


def test_terminate_trainingjob(kfp_client, experiment_id, sagemaker_client):
    test_file_dir = "resources/config/rlestimator-training"
    download_dir = utils.mkdir(
        os.path.join(test_file_dir + "/generated_test_terminate")
    )
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    input_job_name = test_params["Arguments"]["job_name"] = (
        "".join(random.choice(string.ascii_lowercase) for i in range(10))
        + "-terminate-job"
    )

    run_id, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
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

    response = sagemaker_utils.describe_training_job(sagemaker_client, input_job_name)
    assert response["TrainingJobStatus"] in ["Stopping", "Stopped"]

    utils.remove_dir(download_dir)
