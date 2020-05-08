import utils
import os
import pytest

from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils
from utils import s3_utils


@pytest.mark.parametrize(
    "test_file_dir", ["resources/config/kmeans-mnist-batch-transform"]
)
def test_transform_job(
    kfp_client,
    experiment_id,
    s3_client,
    sagemaker_client,
    s3_data_bucket,
    test_file_dir,
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Generate random prefix for model, job name to avoid errors if resources with same name exists
    test_params["Arguments"]["model_name"] = test_params["Arguments"][
        "job_name"
    ] = input_job_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["model_name"]
    )

    # Generate unique location for output since output filename is generated according to the content_type
    test_params["Arguments"]["output_location"] = os.path.join(
        test_params["Arguments"]["output_location"], input_job_name
    )

    run_id, status, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    outputs = {"sagemaker-batch-transformation": ["output_location"]}

    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    # Verify Job was successful on SageMaker
    response = sagemaker_utils.describe_transform_job(sagemaker_client, input_job_name)
    assert response["TransformJobStatus"] == "Completed"
    assert response["TransformJobName"] == input_job_name

    # Verify output location from pipeline matches job output and that the transformed file exists
    output_location = utils.extract_information(
        output_files["sagemaker-batch-transformation"]["output_location"], "data",
    )
    print(f"output location: {output_location}")
    assert output_location == response["TransformOutput"]["S3OutputPath"]
    # Get relative path of file in S3 bucket
    file_key = os.path.join(
        "/".join(output_location.split("/")[3:]), test_params["ExpectedOutputFile"]
    )
    assert s3_utils.check_object_exists(s3_client, s3_data_bucket, file_key)
