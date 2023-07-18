import pytest
import os
import utils
from utils import kfp_client_utils
from utils import ack_utils
from utils import sagemaker_utils
from utils import minio_utils

import json


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/ack-hosting",
            marks=[pytest.mark.canary_test, pytest.mark.v2, pytest.mark.shallow_canary],
        ),
        pytest.param("resources/config/ack-hosting-update", marks=pytest.mark.v2),
    ],
)
def test_create_v2_endpoint(kfp_client, experiment_id, boto3_session, test_file_dir):
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
            shallow_canary=True,
        )
    )
    input_model_name = utils.generate_random_string(10) + "-v2-model"
    input_endpoint_config_name = (
        utils.generate_random_string(10) + "-v2-endpoint-config"
    )
    input_endpoint_name = utils.generate_random_string(10) + "-v2-endpoint"

    test_params["Arguments"]["model_name"] = input_model_name
    test_params["Arguments"]["endpoint_config_name"] = input_endpoint_config_name
    test_params["Arguments"]["endpoint_name"] = input_endpoint_name
    test_params["Arguments"]["production_variants"][0]["modelName"] = input_model_name

    if "ExpectedEndpointConfig" in test_params.keys():
        input_second_endpoint_config_name = (
            utils.generate_random_string(10) + "-v2-sec-endpoint-config"
        )
        test_params["Arguments"][
            "second_endpoint_config_name"
        ] = input_second_endpoint_config_name
        test_params["Arguments"]["second_production_variants"][0][
            "modelName"
        ] = input_model_name

    try:
        _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
        )

        endpoint_describe = ack_utils._get_resource(
            input_endpoint_name, "endpoints"
        )

        outputs = {
            "sagemaker-endpoint": [
                "endpoint_status",
                "sagemaker_resource_name",
                "ack_resource_metadata",
            ],
            "sagemaker-endpointconfig": [
                "sagemaker_resource_name",
                "ack_resource_metadata",
            ],
            "sagemaker-model": [
                "sagemaker_resource_name",
                "ack_resource_metadata",
            ]
        }

        DESIRED_COMPONENT_STATUS = "InService"

        # Get output data
        output_files = minio_utils.artifact_download_iterator(
            workflow_json, outputs, download_dir
        )

        output_endpoint_status = utils.read_from_file_in_tar(
            output_files["sagemaker-endpoint"]["endpoint_status"]
        )

        output_ack_resource_metadata_endpoint = kfp_client_utils.get_output_ack_resource_metadata(
            output_files, "sagemaker-endpoint"
        )

        output_ack_resource_metadata_endpoint_config = kfp_client_utils.get_output_ack_resource_metadata(
            output_files, "sagemaker-endpointconfig"
        )

        output_ack_resource_metadata_model = kfp_client_utils.get_output_ack_resource_metadata(
            output_files, "sagemaker-model"
        )

        output_endpoint_name = utils.read_from_file_in_tar(
            output_files["sagemaker-endpoint"]["sagemaker_resource_name"]
        )

        output_endpoint_config_name = utils.read_from_file_in_tar(
            output_files["sagemaker-endpointconfig"]["sagemaker_resource_name"]
        )

        output_model_name = utils.read_from_file_in_tar(
            output_files["sagemaker-model"]["sagemaker_resource_name"]
        )

        assert (
            endpoint_describe["status"]["endpointStatus"]
            == output_endpoint_status
            == DESIRED_COMPONENT_STATUS
        )
        assert output_endpoint_name in output_ack_resource_metadata_endpoint["arn"]
        assert output_endpoint_config_name in output_ack_resource_metadata_endpoint_config["arn"]
        assert output_model_name in output_ack_resource_metadata_model["arn"]

        # Verify that the update was successful by checking that the endpoint config name is the same as the second one.
        if "ExpectedEndpointConfig" in test_params.keys():
            endpoint_describe["spec"][
                "endpointConfigName"
            ] == input_second_endpoint_config_name

        # Validate the model for use by running a prediction
        result = sagemaker_utils.run_predict_mnist(
            boto3_session, input_endpoint_name, download_dir
        )
        print(f"prediction result: {result}")
        assert json.dumps(result, sort_keys=True) == json.dumps(
            test_params["ExpectedPrediction"], sort_keys=True
        )
        utils.remove_dir(download_dir)
    finally:
        ack_utils._delete_resource(input_endpoint_name, "endpoints")
        ack_utils._delete_resource(
            input_endpoint_config_name, "endpointconfigs"
        )
        ack_utils._delete_resource(input_model_name, "models")


@pytest.mark.v2
def test_terminate_v2_endpoint(kfp_client, experiment_id):
    test_file_dir = "resources/config/ack-hosting"
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )
    input_model_name = utils.generate_random_string(10) + "-v2-model"
    input_endpoint_config_name = (
        utils.generate_random_string(10) + "-v2-endpoint-config"
    )
    input_endpoint_name = utils.generate_random_string(10) + "-v2-endpoint"
    test_params["Arguments"]["model_name"] = input_model_name
    test_params["Arguments"]["endpoint_config_name"] = input_endpoint_config_name
    test_params["Arguments"]["endpoint_name"] = input_endpoint_name
    test_params["Arguments"]["production_variants"][0]["modelName"] = input_model_name
    try:
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
        assert ack_utils.wait_for_condition(
            input_endpoint_name,
            ack_utils.does_endpoint_exist,
            wait_periods=12,
            period_length=12,
        )
        kfp_client_utils.terminate_run(kfp_client, run_id)
        assert ack_utils.wait_for_condition(
            input_endpoint_name,
            ack_utils.is_endpoint_deleted,
            wait_periods=20,
            period_length=20,
        )
    finally:
        ack_utils._delete_resource(
            input_endpoint_config_name, "endpointconfigs"
        )
        ack_utils._delete_resource(input_model_name, "models")
