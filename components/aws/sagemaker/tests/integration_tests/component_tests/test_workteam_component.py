import pytest
import os
import utils
from utils import kfp_client_utils
from utils import sagemaker_utils
from utils import minio_utils


def create_workteamjob(
    kfp_client, test_params, experiment_id, region, sagemaker_client, download_dir
):
    # Get the account, region specific user_pool and client_id for the SageMaker Workforce.
    (
        test_params["Arguments"]["user_pool"],
        test_params["Arguments"]["client_id"],
        test_params["Arguments"]["user_groups"],
    ) = sagemaker_utils.get_cognito_member_definitions(sagemaker_client)

    _, _, workflow_json = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    return workflow_json


@pytest.mark.parametrize(
    "test_file_dir",
    [pytest.param("resources/config/create-workteam", marks=pytest.mark.canary_test)],
)
def test_workteamjob(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))

    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Generate random prefix for workteam_name to avoid errors if resources with same name exists
    test_params["Arguments"]["team_name"] = workteam_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["team_name"]
    )

    try:
        workflow_json = create_workteamjob(
            kfp_client,
            test_params,
            experiment_id,
            region,
            sagemaker_client,
            download_dir,
        )

        outputs = {"sagemaker-private-workforce": ["workteam_arn"]}

        output_files = minio_utils.artifact_download_iterator(
            workflow_json, outputs, download_dir
        )

        response = sagemaker_utils.describe_workteam(sagemaker_client, workteam_name)

        # Verify WorkTeam was created in SageMaker
        assert response["Workteam"]["CreateDate"] is not None
        assert response["Workteam"]["WorkteamName"] == workteam_name

        # Verify WorkTeam arn artifact was created in Minio and matches the one in SageMaker
        workteam_arn = utils.read_from_file_in_tar(
            output_files["sagemaker-private-workforce"]["workteam_arn"]
        )
        assert response["Workteam"]["WorkteamArn"] == workteam_arn

    finally:
        workteams = sagemaker_utils.list_workteams(sagemaker_client)["Workteams"]
        workteam_names = list(map((lambda x: x["WorkteamName"]), workteams))
        # Check workteam was successfully created
        if workteam_name in workteam_names:
            sagemaker_utils.delete_workteam(sagemaker_client, workteam_name)

    # Delete generated files only if the test is successful
    utils.remove_dir(download_dir)
