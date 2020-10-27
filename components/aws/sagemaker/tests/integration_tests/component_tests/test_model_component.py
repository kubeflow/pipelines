import pytest
import os
import utils

from utils import kfp_client_utils
from utils import minio_utils
from utils import sagemaker_utils


@pytest.mark.parametrize(
    "test_file_dir", ["resources/config/kmeans-mnist-model"],
)
def test_createmodel(kfp_client, experiment_id, sagemaker_client, test_file_dir):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Generate random prefix for model name to avoid errors if model with same name exists
    test_params["Arguments"]["model_name"] = input_model_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["model_name"]
    )
    print(f"running test with model_name: {input_model_name}")

    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    outputs = {"sagemaker-create-model": ["model_name"]}

    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    output_model_name = utils.read_from_file_in_tar(
        output_files["sagemaker-create-model"]["model_name"]
    )
    print(f"model_name: {output_model_name}")
    assert output_model_name == input_model_name
    assert (
        sagemaker_utils.describe_model(sagemaker_client, input_model_name) is not None
    )

    utils.remove_dir(download_dir)
