import pytest
import os
import utils
from utils import kfp_client_utils
from utils import argo_utils
from utils import minio_utils
from utils import sagemaker_utils


def test_simple_mnist_training(
    kfp_client, experiment_id, s3_client, sagemaker_client, region
):

    pipeline_name = "simple_mnist_training"
    test_file_dir = os.path.join("resources/", pipeline_name)
    run_id, status = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client, experiment_id, test_file_dir, pipeline_name, 1200
    )

    workflow_json = kfp_client_utils.get_workflow_json(kfp_client, run_id)

    if not status:
        argo_utils.print_workflow_logs(workflow_json["metadata"]["name"])
        pytest.fail(f"Test Failed: {pipeline_name}")
    else:
        print("run suceeded")
        outputs = {"sagemaker-training-job": ["job_name", "model_artifact_url"]}
        output_files = minio_utils.artifact_download_iterator(
            workflow_json, outputs, test_file_dir
        )

        # Verify Training job was successful on Sagemaker
        training_job_name = utils.extract_information(
            output_files["sagemaker-training-job"]["job_name"], "job_name.txt"
        )
        print("training job name: ", training_job_name)
        train_response = sagemaker_utils.describe_training_job(
            sagemaker_client, training_job_name.decode()
        )
        assert train_response["TrainingJobStatus"] == "Completed"

        # Verify model artifacts output
        model_artifact_url = utils.extract_information(
            output_files["sagemaker-training-job"]["model_artifact_url"],
            "model_artifact_url.txt",
        )
        print("model_artifact_url: ", model_artifact_url)
        assert (
            model_artifact_url.decode()
            == train_response["ModelArtifacts"]["S3ModelArtifacts"]
        )

    # assert 0
