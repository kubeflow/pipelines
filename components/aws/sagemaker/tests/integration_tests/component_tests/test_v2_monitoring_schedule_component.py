import pytest
import os
import utils
from utils import kfp_client_utils
from utils import ack_utils
from utils import minio_utils

FINAL_STATUS = "Scheduled"


def verify_monitoring_schedule_component_outputs(
    workflow_json, download_dir, monitoring_schedule_name
):
    # Verify component outputs
    outputs = {
        "sagemaker-monitoringschedule": [
            "ack_resource_metadata",
            "monitoring_schedule_status",
            "sagemaker_resource_name",
        ]
    }

    output_files = minio_utils.artifact_download_iterator(
        workflow_json, outputs, download_dir
    )

    output_ack_resource_metadata = kfp_client_utils.get_output_ack_resource_metadata(
        output_files, "sagemaker-monitoringschedule"
    )
    output_schedule_status = utils.read_from_file_in_tar(
        output_files["sagemaker-monitoringschedule"]["monitoring_schedule_status"]
    )
    output_schedule_name = utils.read_from_file_in_tar(
        output_files["sagemaker-monitoringschedule"]["sagemaker_resource_name"]
    )

    assert monitoring_schedule_name in output_ack_resource_metadata["arn"]
    assert output_schedule_name == monitoring_schedule_name
    assert output_schedule_status == FINAL_STATUS


# Testing monitoring schedule with model bias job definition
@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/v2-monitoring-schedule",
            marks=[pytest.mark.canary_test, pytest.mark.v2],
        ),
    ],
)
def test_create_v2_monitoring_schedule(
    kfp_client, experiment_id, test_file_dir, deploy_endpoint
):
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # parameters for model bias job definition
    job_definition_name = (
        utils.generate_random_string(10) + "-v2-model-bias-job-definition"
    )
    test_params["Arguments"]["job_definition_name"] = job_definition_name
    test_params["Arguments"]["model_bias_job_input"]["endpointInput"][
        "endpointName"
    ] = deploy_endpoint

    # parameters for monitoring schedule
    monitoring_schedule_name = (
        utils.generate_random_string(10) + "-v2-monitoring-schedule"
    )
    test_params["Arguments"]["monitoring_schedule_name"] = monitoring_schedule_name
    test_params["Arguments"]["monitoring_schedule_config"][
        "monitoringJobDefinitionName"
    ] = job_definition_name

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

        # Verify if the job definition CR is created properly
        job_definition_describe = ack_utils._get_resource(
            job_definition_name, "modelbiasjobdefinitions"
        )
        assert (
            job_definition_name
            in job_definition_describe["status"]["ackResourceMetadata"]["arn"]
        )
        assert (
            job_definition_describe["spec"]["modelBiasJobInput"]["endpointInput"][
                "endpointName"
            ]
            == deploy_endpoint
        )

        # Verify if monitoring schedule CR is created properly
        monitoring_schedule_describe = ack_utils._get_resource(
            monitoring_schedule_name, "monitoringschedules"
        )
        assert (
            monitoring_schedule_name
            in monitoring_schedule_describe["status"]["ackResourceMetadata"]["arn"]
        )
        assert (
            monitoring_schedule_describe["status"]["monitoringScheduleStatus"]
            == FINAL_STATUS
        )

        verify_monitoring_schedule_component_outputs(
            workflow_json, download_dir, monitoring_schedule_name
        )

    finally:
        ack_utils._delete_resource(
            job_definition_name,
            "modelbiasjobdefinitions",
            wait_periods=10,
            period_length=30,
        )
        ack_utils._delete_resource(
            monitoring_schedule_name,
            "monitoringschedules",
            wait_periods=10,
            period_length=30,
        )


# Test updating monitoring schedule using the same pipeline
# Steps:
# Prepare pipeline inputs for job_definition_1 and monitoring schedule
# Run the pipeline
# Update pipeline input (instanceType) for job_definition_2
# Rerun the same pipeline
@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/v2-monitoring-schedule-update",
            marks=[pytest.mark.v2],
        ),
    ],
)
def test_update_v2_monitoring_schedule(
    kfp_client, experiment_id, test_file_dir, deploy_endpoint
):
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # parameters for job definition
    test_params["Arguments"][test_params["JobInputName"]]["endpointInput"][
        "endpointName"
    ] = deploy_endpoint

    job_definition_name_1 = (
        utils.generate_random_string(10) + "-v2-model-data-quality-defi"
    )
    job_definition_name_2 = (
        utils.generate_random_string(10) + "-v2-model-data-quality-defi"
    )

    # parameter for monitoring schedule
    monitoring_schedule_name = (
        utils.generate_random_string(10) + "-v2-monitoring-schedule"
    )

    try:
        test_params["Arguments"]["job_definition_name"] = job_definition_name_1
        test_params["Arguments"]["job_resources"]["clusterConfig"][
            "instanceType"
        ] = "ml.m5.large"
        test_params["Arguments"]["monitoring_schedule_name"] = monitoring_schedule_name

        _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
        )

        # Verify if monitoring schedule CR is created properly
        monitoring_schedule_describe = ack_utils._get_resource(
            monitoring_schedule_name, "monitoringschedules"
        )
        assert (
            monitoring_schedule_name
            in monitoring_schedule_describe["status"]["ackResourceMetadata"]["arn"]
        )
        assert (
            monitoring_schedule_describe["status"]["monitoringScheduleStatus"]
            == FINAL_STATUS
        )
        assert (
            monitoring_schedule_describe["spec"]["monitoringScheduleConfig"][
                "monitoringJobDefinitionName"
            ]
            == job_definition_name_1
        )

        # Verify if the job definition CR is created properly
        job_definition_1_describe = ack_utils._get_resource(
            job_definition_name_1, "dataqualityjobdefinitions"
        )
        assert (
            job_definition_name_1
            in job_definition_1_describe["status"]["ackResourceMetadata"]["arn"]
        )
        assert (
            job_definition_1_describe["spec"]["dataQualityJobInput"]["endpointInput"][
                "endpointName"
            ]
            == deploy_endpoint
        )
        assert (
            job_definition_1_describe["spec"]["jobResources"]["clusterConfig"][
                "instanceType"
            ]
            == "ml.m5.large"
        )

        verify_monitoring_schedule_component_outputs(
            workflow_json, download_dir, monitoring_schedule_name
        )

        # Update monitoring schedule using new job definition
        test_params["Arguments"]["job_definition_name"] = job_definition_name_2
        test_params["Arguments"]["job_resources"]["clusterConfig"][
            "instanceType"
        ] = "ml.m5.xlarge"

        _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
        )

        # Verify if monitoring schedule is updated with correct job definition
        monitoring_schedule_updated_describe = ack_utils._get_resource(
            monitoring_schedule_name, "monitoringschedules"
        )
        assert (
            monitoring_schedule_updated_describe["status"]["monitoringScheduleStatus"]
            == FINAL_STATUS
        )
        assert (
            monitoring_schedule_updated_describe["spec"]["monitoringScheduleConfig"][
                "monitoringJobDefinitionName"
            ]
            == job_definition_name_2
        )

        # Verify if the new job definition CR is created properly
        job_definition_2_describe = ack_utils._get_resource(
            job_definition_name_2, "dataqualityjobdefinitions"
        )
        assert (
            job_definition_name_2
            in job_definition_2_describe["status"]["ackResourceMetadata"]["arn"]
        )
        assert (
            job_definition_2_describe["spec"]["jobResources"]["clusterConfig"][
                "instanceType"
            ]
            == "ml.m5.xlarge"
        )

        verify_monitoring_schedule_component_outputs(
            workflow_json, download_dir, monitoring_schedule_name
        )

    finally:
        ack_utils._delete_resource(
            job_definition_name_1,
            test_params["Plural"],
            wait_periods=10,
            period_length=30,
        )
        ack_utils._delete_resource(
            job_definition_name_2,
            test_params["Plural"],
            wait_periods=10,
            period_length=30,
        )
        ack_utils._delete_resource(
            monitoring_schedule_name,
            "monitoringschedules",
            wait_periods=10,
            period_length=30,
        )
