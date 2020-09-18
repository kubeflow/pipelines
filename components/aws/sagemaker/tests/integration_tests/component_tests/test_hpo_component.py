import pytest
import os
import json
import utils

from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils


@pytest.mark.parametrize(
    "test_file_dir",
    [pytest.param("resources/config/kmeans-mnist-hpo", marks=pytest.mark.canary_test)],
)
def test_hyperparameter_tuning(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )
    if "job_name" in test_params["Arguments"]:
        test_params["Arguments"]["job_name"] = (
            utils.generate_random_string(5) + "-" + test_params["Arguments"]["job_name"]
        )

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
        "sagemaker-hyperparameter-tuning": [
            "best_hyperparameters",
            "best_job_name",
            "hpo_job_name",
            "model_artifact_url",
            "training_image",
        ]
    }
    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    # Verify HPO job was successful on SageMaker
    hpo_job_name = utils.read_from_file_in_tar(
        output_files["sagemaker-hyperparameter-tuning"]["hpo_job_name"]
    )
    print(f"HPO job name: {hpo_job_name}")
    hpo_response = sagemaker_utils.describe_hpo_job(sagemaker_client, hpo_job_name)
    assert hpo_response["HyperParameterTuningJobStatus"] == "Completed"
    if "job_name" in test_params["Arguments"]:
        assert hpo_response["HyperParameterTuningJobName"] == hpo_job_name

    # Verify training image output is an ECR image
    training_image = utils.read_from_file_in_tar(
        output_files["sagemaker-hyperparameter-tuning"]["training_image"]
    )
    print(f"Training image used: {training_image}")
    if "ExpectedTrainingImage" in test_params.keys():
        assert test_params["ExpectedTrainingImage"] == training_image
    else:
        assert f"dkr.ecr.{region}.amazonaws.com" in training_image

    # Verify Training job was part of HPO job, returned as best and was successful
    best_training_job_name = utils.read_from_file_in_tar(
        output_files["sagemaker-hyperparameter-tuning"]["best_job_name"]
    )
    print(f"best training job name: {best_training_job_name}")
    train_response = sagemaker_utils.describe_training_job(
        sagemaker_client, best_training_job_name
    )
    assert train_response["TuningJobArn"] == hpo_response["HyperParameterTuningJobArn"]
    assert (
        train_response["TrainingJobName"]
        == hpo_response["BestTrainingJob"]["TrainingJobName"]
    )
    assert train_response["TrainingJobStatus"] == "Completed"

    # Verify model artifacts output was generated from this run
    model_artifact_url = utils.read_from_file_in_tar(
        output_files["sagemaker-hyperparameter-tuning"]["model_artifact_url"]
    )
    print(f"model_artifact_url: {model_artifact_url}")
    assert model_artifact_url == train_response["ModelArtifacts"]["S3ModelArtifacts"]
    assert best_training_job_name in model_artifact_url

    # Verify hyper_parameters output is not empty
    hyper_parameters = json.loads(
        utils.read_from_file_in_tar(
            output_files["sagemaker-hyperparameter-tuning"]["best_hyperparameters"]
        )
    )
    print(f"HPO best hyperparameters: {json.dumps(hyper_parameters, indent = 2)}")
    assert hyper_parameters is not None

    utils.remove_dir(download_dir)


def test_terminate_hpojob(kfp_client, experiment_id, region, sagemaker_client):
    test_file_dir = "resources/config/kmeans-mnist-hpo"
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
        utils.generate_random_string(4) + "-terminate-job"
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

    print(f"Terminating run: {run_id} where HPO job_name: {input_job_name}")
    kfp_client_utils.terminate_run(kfp_client, run_id)

    response = sagemaker_utils.describe_hpo_job(sagemaker_client, input_job_name)
    assert response["HyperParameterTuningJobStatus"] in ["Stopping", "Stopped"]

    utils.remove_dir(download_dir)
