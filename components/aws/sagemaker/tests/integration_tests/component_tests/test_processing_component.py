import pytest
import os
import json
import utils
from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/kmeans-algo-mnist-processing",
            marks=pytest.mark.canary_test,
        )
    ],
)
def test_processingjob(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    test_params["Arguments"]["input_config"] = json.dumps(
        test_params["Arguments"]["input_config"]
    )
    test_params["Arguments"]["output_config"] = json.dumps(
        test_params["Arguments"]["output_config"]
    )

    # Generate random prefix for job name to avoid errors if model with same name exists
    test_params["Arguments"]["job_name"] = input_job_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["job_name"]
    )
    print(f"running test with job_name: {input_job_name}")

    for index, output in enumerate(test_params["Arguments"]["output_config"]):
        if "S3Output" in output:
            test_params["Arguments"]["output_config"][index]["S3Output"]["S3Uri"] = os.path.join(
                output["S3Output"]["S3Uri"], input_job_name
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

    outputs = {"sagemaker-processing-job": ["job_name"]}
    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    # Verify Processing job was successful on SageMaker
    processing_job_name = utils.read_from_file_in_tar(
        output_files["sagemaker-processing-job"]["job_name"], "job_name.txt"
    )
    print(f"processing job name: {processing_job_name}")
    process_response = sagemaker_utils.describe_processing_job(
        sagemaker_client, processing_job_name
    )
    assert process_response["ProcessingJobStatus"] == "Completed"
    assert process_response["ProcessingJobArn"].split("/")[1] == input_job_name

    # # Verify model artifacts output was generated from this run
    # model_artifact_url = utils.read_from_file_in_tar(
    #     output_files["sagemaker-training-job"]["model_artifact_url"],
    #     "model_artifact_url.txt",
    # )
    # print(f"model_artifact_url: {model_artifact_url}")
    # assert model_artifact_url == train_response["ModelArtifacts"]["S3ModelArtifacts"]
    # assert training_job_name in model_artifact_url

    # # Verify training image output is an ECR image
    # training_image = utils.read_from_file_in_tar(
    #     output_files["sagemaker-training-job"]["training_image"], "training_image.txt",
    # )
    # print(f"Training image used: {training_image}")
    # if "ExpectedTrainingImage" in test_params.keys():
    #     assert test_params["ExpectedTrainingImage"] == training_image
    # else:
    #     assert f"dkr.ecr.{region}.amazonaws.com" in training_image

    utils.remove_dir(download_dir)
