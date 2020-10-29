import pytest
import os
import json
import utils
from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils
from utils import argo_utils


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/kmeans-algo-mnist-processing",
            marks=pytest.mark.canary_test,
        ),
        "resources/config/assume-role-processing",
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

    # Generate random prefix for job name to avoid errors if model with same name exists
    test_params["Arguments"]["job_name"] = input_job_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["job_name"]
    )
    print(f"running test with job_name: {input_job_name}")

    for index, output in enumerate(test_params["Arguments"]["output_config"]):
        if "S3Output" in output:
            test_params["Arguments"]["output_config"][index]["S3Output"][
                "S3Uri"
            ] = os.path.join(output["S3Output"]["S3Uri"], input_job_name)

    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    outputs = {"sagemaker-processing-job": ["job_name", "output_artifacts"]}
    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    # Verify Processing job was successful on SageMaker
    processing_job_name = utils.read_from_file_in_tar(
        output_files["sagemaker-processing-job"]["job_name"]
    )
    print(f"processing job name: {processing_job_name}")
    process_response = sagemaker_utils.describe_processing_job(
        sagemaker_client, processing_job_name
    )
    assert process_response["ProcessingJobStatus"] == "Completed"
    assert process_response["ProcessingJobArn"].split("/")[1] == input_job_name

    # Verify processing job produced the correct outputs
    processing_outputs = json.loads(
        utils.read_from_file_in_tar(
            output_files["sagemaker-processing-job"]["output_artifacts"],
        )
    )
    print(f"processing job outputs: {json.dumps(processing_outputs, indent = 2)}")
    assert processing_outputs is not None

    for output in process_response["ProcessingOutputConfig"]["Outputs"]:
        assert processing_outputs[output["OutputName"]] == output["S3Output"]["S3Uri"]

    assert not argo_utils.error_in_cw_logs(
        workflow_json["metadata"]["name"]
    ), "Found the CloudWatch error message in the log output. Check SageMaker to see if the job has failed."

    utils.remove_dir(download_dir)


def test_terminate_processingjob(kfp_client, experiment_id, region, sagemaker_client):
    test_file_dir = "resources/config/kmeans-algo-mnist-processing"
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

    print(f"Terminating run: {run_id} where Processing job_name: {input_job_name}")
    kfp_client_utils.terminate_run(kfp_client, run_id)

    response = sagemaker_utils.describe_processing_job(sagemaker_client, input_job_name)
    assert response["ProcessingJobStatus"] in ["Stopping", "Stopped"]

    utils.remove_dir(download_dir)
