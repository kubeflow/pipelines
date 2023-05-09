import time
import pytest
import os
import utils
from utils import kfp_client_utils
from utils import ack_utils


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
    k8s_client = ack_utils.k8s_client()
    job_definition_name = (
        utils.generate_random_string(10) + "-v2-" + test_params["TestName"]
    )
    job_input_name = test_params["JobInputName"]
    test_params["Arguments"]["job_definition_name"] = job_definition_name
    test_params["Arguments"][job_input_name]["endpointInput"][
        "endpointName"
    ] = deploy_endpoint

    try:
        _, _, _ = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
        )

        job_definition_describe = ack_utils._get_resource(
            k8s_client, job_definition_name, test_params["Plural"]
        )

        # Check if the job definition is created
        assert (
            job_definition_name
            in job_definition_describe["status"]["ackResourceMetadata"]["arn"]
        )

    finally:
        ack_utils._delete_resource(
            k8s_client, job_definition_name, test_params["Plural"]
        )
