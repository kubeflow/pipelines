import pytest
import os
import utils
from utils import kfp_client_utils
from utils import ack_utils
from utils import minio_utils


# Testing data quality job definition component and model explainability job definition component
@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/v2-monitoring-job-data-quality", marks=pytest.mark.v2
        ),
        pytest.param(
            "resources/config/v2-monitoring-job-model-explainability",
            marks=pytest.mark.v2,
        ),
        pytest.param(
            "resources/config/v2-monitoring-job-model-bias", marks=pytest.mark.v2
        ),
        pytest.param(
            "resources/config/v2-monitoring-job-model-quality", marks=pytest.mark.v2
        ),
    ],
)
def test_job_definitions(kfp_client, experiment_id, test_file_dir, deploy_endpoint):
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )
    job_definition_name = (
        utils.generate_random_string(10) + "-v2-" + test_params["TestName"]
    )
    job_input_name = test_params["JobInputName"]
    test_params["Arguments"]["job_definition_name"] = job_definition_name
    test_params["Arguments"][job_input_name]["endpointInput"][
        "endpointName"
    ] = deploy_endpoint

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

        # Verify if the job definition CR is created
        job_definition_describe = ack_utils._get_resource(
            job_definition_name, test_params["Plural"]
        )
        assert (
            job_definition_name
            in job_definition_describe["status"]["ackResourceMetadata"]["arn"]
        )

        # Verify component output
        step_name = "sagemaker-" + test_params["Plural"][:-1]
        outputs = {
            step_name: [
                "ack_resource_metadata",
                "sagemaker_resource_name",
            ]
        }

        output_files = minio_utils.artifact_download_iterator(
            workflow_json, outputs, download_dir
        )

        output_ack_resource_metadata = (
            kfp_client_utils.get_output_ack_resource_metadata(output_files, step_name)
        )
        output_resource_name = utils.read_from_file_in_tar(
            output_files[step_name]["sagemaker_resource_name"]
        )

        assert job_definition_name in output_ack_resource_metadata["arn"]
        assert output_resource_name == job_definition_name

    finally:
        ack_utils._delete_resource(
            job_definition_name, test_params["Plural"]
        )
