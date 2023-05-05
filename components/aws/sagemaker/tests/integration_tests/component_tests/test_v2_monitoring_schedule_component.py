import time
import pytest
import os
import utils
from utils import kfp_client_utils
from utils import ack_utils


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
def test_v2_monitoring_schedule(
    kfp_client, experiment_id, test_file_dir, deploy_endpoint, sagemaker_client
):
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )
    k8s_client = ack_utils.k8s_client()

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
            k8s_client, job_definition_name, "modelbiasjobdefinitions"
        )

        # Check if the job definition is created
        assert job_definition_describe["status"]["ackResourceMetadata"]["arn"] != None

        # Verify if monitoring schedule is created with correct name and endpoint
        monitoring_schedule_describe = sagemaker_client.describe_monitoring_schedule(
            MonitoringScheduleName=monitoring_schedule_name
        )
        assert (
            monitoring_schedule_name
            in monitoring_schedule_describe["MonitoringScheduleArn"]
        )

        assert monitoring_schedule_describe["MonitoringScheduleStatus"] == "Scheduled"

        assert monitoring_schedule_describe["EndpointName"] == deploy_endpoint

    finally:
        ack_utils._delete_resource(
            k8s_client,
            job_definition_name,
            "modelbiasjobdefinitions",
            wait_periods=10,
            period_length=30,
        )
        ack_utils._delete_resource(
            k8s_client,
            monitoring_schedule_name,
            "monitoringschedules",
            wait_periods=10,
            period_length=30,
        )


# Testing monitoring schedule with model data quality job definition
@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/v2-monitoring-schedule-update",
            marks=[pytest.mark.canary_test, pytest.mark.v2],
        ),
    ],
)
def test_v2_monitoring_schedule_update(
    kfp_client, experiment_id, test_file_dir, deploy_endpoint, sagemaker_client
):
    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )
    k8s_client = ack_utils.k8s_client()

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

        _, _, _ = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
        )

        job_definition_1_describe_ack = ack_utils._get_resource(
            k8s_client, job_definition_name_1, test_params["Plural"]
        )

        # Check if the job definition is created
        assert (
            job_definition_1_describe_ack["status"]["ackResourceMetadata"]["arn"]
            != None
        )

        # Verify if monitoring schedule is created with correct name and endpoint
        monitoring_schedule_describe = sagemaker_client.describe_monitoring_schedule(
            MonitoringScheduleName=monitoring_schedule_name
        )
        assert (
            monitoring_schedule_name
            in monitoring_schedule_describe["MonitoringScheduleArn"]
        )

        assert monitoring_schedule_describe["MonitoringScheduleStatus"] == "Scheduled"

        assert monitoring_schedule_describe["EndpointName"] == deploy_endpoint

        assert (
            monitoring_schedule_describe["MonitoringScheduleConfig"][
                "MonitoringJobDefinitionName"
            ]
            == job_definition_name_1
        )

        # Verify if job definition is created with correct instance type
        job_definition_describe = sagemaker_client.describe_data_quality_job_definition(
            JobDefinitionName=job_definition_name_1
        )
        assert (
            job_definition_describe["JobResources"]["ClusterConfig"]["InstanceType"]
            == "ml.m5.large"
        )

        # Update monitoring schedule using new job definition
        test_params["Arguments"]["job_definition_name"] = job_definition_name_2
        test_params["Arguments"]["job_resources"]["clusterConfig"][
            "instanceType"
        ] = "ml.m5.xlarge"

        _, _, _ = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
        )

        monitoring_schedule_updated_describe = (
            sagemaker_client.describe_monitoring_schedule(
                MonitoringScheduleName=monitoring_schedule_name
            )
        )

        assert (
            monitoring_schedule_updated_describe["MonitoringScheduleConfig"][
                "MonitoringJobDefinitionName"
            ]
            == job_definition_name_2
        )

        # Verify if job definition is created with correct instance type
        job_definition_2_describe = (
            sagemaker_client.describe_data_quality_job_definition(
                JobDefinitionName=job_definition_name_2
            )
        )
        assert (
            job_definition_2_describe["JobResources"]["ClusterConfig"]["InstanceType"]
            == "ml.m5.xlarge"
        )

    finally:
        ack_utils._delete_resource(
            k8s_client,
            job_definition_name_1,
            test_params["Plural"],
            wait_periods=10,
            period_length=30,
        )
        ack_utils._delete_resource(
            k8s_client,
            job_definition_name_2,
            test_params["Plural"],
            wait_periods=10,
            period_length=30,
        )
        ack_utils._delete_resource(
            k8s_client,
            monitoring_schedule_name,
            "monitoringschedules",
            wait_periods=10,
            period_length=30,
        )
