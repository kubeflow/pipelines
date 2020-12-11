import pytest
import os
import utils

from utils import kfp_client_utils
from utils import minio_utils
from utils import robomaker_utils
from utils import get_s3_data_bucket


def create_simulation_app(kfp_client, experiment_id, creat_app_dir, app_name):
    download_dir = utils.mkdir(os.path.join(creat_app_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(creat_app_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Generate random prefix for sim app name
    sim_app_name = test_params["Arguments"]["app_name"] = (
        utils.generate_random_string(5) + "-" + app_name
    )

    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    return workflow_json, sim_app_name


def create_robot_app(client):
    robomaker_sources = [
        {
            "s3Bucket": get_s3_data_bucket(),
            "s3Key": "robomaker/robot_ws.tar",
            "architecture": "X86_64",
        }
    ]
    robomaker_suite = {"name": "ROS", "version": "Melodic"}
    app_name = utils.generate_random_string(5) + "-test-robot-app"

    response = robomaker_utils.create_robot_application(
        client, app_name, robomaker_sources, robomaker_suite
    )
    return response["arn"]


@pytest.mark.parametrize(
    "test_file_dir", ["resources/config/robomaker-create-simulation-app"],
)
def test_create_simulation_app(
    kfp_client, experiment_id, robomaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Create simulation app with random name
    workflow_json, sim_app_name = create_simulation_app(
        kfp_client, experiment_id, test_file_dir, test_params["Arguments"]["app_name"]
    )

    try:
        print(f"running test with simulation application name: {sim_app_name}")

        outputs = {"robomaker-create-simulation-application": ["arn"]}

        output_files = minio_utils.artifact_download_iterator(
            workflow_json, outputs, download_dir
        )

        sim_app_arn = utils.read_from_file_in_tar(
            output_files["robomaker-create-simulation-application"]["arn"]
        )
        print(f"Simulation Application arn: {sim_app_arn}")

        # Verify simulation application exists
        assert (
            robomaker_utils.describe_simulation_application(
                robomaker_client, sim_app_arn
            )["name"]
            == sim_app_name
        )

    finally:
        robomaker_utils.delete_simulation_application(robomaker_client, sim_app_arn)


@pytest.mark.parametrize(
    "test_file_dir", ["resources/config/robomaker-delete-simulation-app"],
)
def test_delete_simulation_app(
    kfp_client, experiment_id, robomaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Create simulation app with random name
    workflow_json, sim_app_name = create_simulation_app(
        kfp_client,
        experiment_id,
        "resources/config/robomaker-create-simulation-app",
        "fake-app-name",
    )

    print(f"running test with simulation application name: {sim_app_name}")

    create_outputs = {"robomaker-create-simulation-application": ["arn"]}

    create_output_files = minio_utils.artifact_download_iterator(
        workflow_json, create_outputs, download_dir
    )

    sim_app_arn = utils.read_from_file_in_tar(
        create_output_files["robomaker-create-simulation-application"]["arn"]
    )
    print(f"Simulation Application arn: {sim_app_arn}")

    # Here we perform the delete
    test_params["Arguments"]["arn"] = sim_app_arn
    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    # Verify simulation application does not exist
    simulation_applications = robomaker_utils.list_simulation_applications(
        robomaker_client, sim_app_name
    )
    assert len(simulation_applications["simulationApplicationSummaries"]) == 0


@pytest.mark.parametrize(
    "test_file_dir", ["resources/config/robomaker-simulation-job"],
)
def test_run_simulation_job(kfp_client, experiment_id, robomaker_client, test_file_dir):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Create simulation app with random name
    sim_app_workflow_json, sim_app_name = create_simulation_app(
        kfp_client,
        experiment_id,
        "resources/config/robomaker-create-simulation-app",
        "random-app-name",
    )

    print(f"running test with simulation application name: {sim_app_name}")

    sim_app_outputs = {"robomaker-create-simulation-application": ["arn"]}

    sim_app_output_files = minio_utils.artifact_download_iterator(
        sim_app_workflow_json, sim_app_outputs, download_dir
    )

    sim_app_arn = utils.read_from_file_in_tar(
        sim_app_output_files["robomaker-create-simulation-application"]["arn"]
    )
    print(f"Simulation Application arn: {sim_app_arn}")

    # Create Robot App by invoking api directly
    robot_app_arn = create_robot_app(robomaker_client)

    # Here we run the simulation job
    test_params["Arguments"]["sim_app_arn"] = sim_app_arn
    test_params["Arguments"]["robot_app_arn"] = robot_app_arn

    _, _, sim_job_workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    sim_job_outputs = {"robomaker-create-simulation-job": ["arn"]}
    sim_job_output_files = minio_utils.artifact_download_iterator(
        sim_job_workflow_json, sim_job_outputs, download_dir
    )
    sim_job_arn = utils.read_from_file_in_tar(
        sim_job_output_files["robomaker-create-simulation-job"]["arn"]
    )
    print(f"Simulation Job arn: {sim_job_arn}")

    # Verify simulation job ran successfully
    assert robomaker_utils.describe_simulation_job(robomaker_client, sim_job_arn)[
        "status"
    ] not in ["Failed", "RunningFailed"]
