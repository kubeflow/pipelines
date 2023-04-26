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
            "resources/config/ack-model-explainability-job-definition",
            marks=pytest.mark.canary_test,
        ),
        pytest.param(
            "resources/config/ack-data-quality-job-definition",
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

    print(test_params)

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

        print("Describe job definition " + job_definition_name)
        print(job_definition_describe)

        # Check if the job definition is created

        assert (
            job_definition_describe["status"]["conditions"][0]["type"]
            == "ACK.ResourceSynced"
        )
        assert (
            job_definition_name
            in job_definition_describe["status"]["ackResourceMetadata"]["arn"]
        )

    finally:
        ack_utils._delete_resource(
            k8s_client, job_definition_name, test_params["Plural"]
        )


# Testing monitoring schedule with model bias job definition
@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/ack-monitoring-schedule",
            marks=pytest.mark.canary_test,
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

    print(test_params)

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
        assert (
            job_definition_describe["status"]["conditions"][0]["type"]
            == "ACK.ResourceSynced"
        )

        assert job_definition_describe["status"]["ackResourceMetadata"]["arn"] != None

        # Check if the monitoring schedule is created
        print("Describe monitoring Schedule Name: " + monitoring_schedule_name)

        # wait for 5 minutes for monitoring schedule to be scheduled
        for i in range(10):
            monitoring_schedule_describe = (
                sagemaker_client.describe_monitoring_schedule(
                    MonitoringScheduleName=monitoring_schedule_name
                )
            )

            if monitoring_schedule_describe["MonitoringScheduleStatus"] == "Scheduled":
                break

        print(f"Describe monitoring Schedule \n {monitoring_schedule_describe}")

        # Verify if monitoring schedule is created with correct name and endpoint

        assert (
            monitoring_schedule_name
            in monitoring_schedule_describe["MonitoringScheduleArn"]
        )

        assert monitoring_schedule_describe["MonitoringScheduleStatus"] == "Scheduled"

        assert monitoring_schedule_describe["EndpointName"] == deploy_endpoint

    finally:
        ack_utils._delete_resource(
            k8s_client, job_definition_name, "modelbiasjobdefinitions"
        )
        try:
            # wait for 10 minutes for monitoring schedule to be deleted
            for i in range(20):
                res = ack_utils._delete_resource(
                    k8s_client, monitoring_schedule_name, "monitoringschedules"
                )

                if res == None:
                    print("MonitoringSchedule resource deleted")
                    break

                time.sleep(30)
        except:
            print("MonitoringSchedule resource failed")
            raise
