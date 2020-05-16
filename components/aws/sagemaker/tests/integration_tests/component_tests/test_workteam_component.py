import pytest
import os
import json
import utils
from utils import kfp_client_utils
from utils import sagemaker_utils

def create_workteamjob(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    # Get the account, region specific user_pool and client_id for the Sagemaker Workforce. 
    test_params["Arguments"]["user_pool"], test_params["Arguments"]["client_id"], test_params["Arguments"]["user_groups"] = sagemaker_utils.get_cognito_member_definitions(sagemaker_client)

    # Generate random prefix for workteam_name to avoid errors if resources with same name exists
    test_params["Arguments"]["team_name"] = workteam_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["team_name"]
    )

    _ = kfp_client_utils.compile_run_monitor_pipeline(
        kfp_client,
        experiment_id,
        test_params["PipelineDefinition"],
        test_params["Arguments"],
        download_dir,
        test_params["TestName"],
        test_params["Timeout"],
    )

    # Delete generated files
    utils.remove_dir(download_dir)
    
    return workteam_name


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/create-workteam", marks=pytest.mark.canary_test
        )
    ],
)
def test_workteamjob(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    workteam_name = create_workteamjob(kfp_client, experiment_id, region, sagemaker_client, test_file_dir)

    try:
        # Verify WorkTeam was created in SageMaker
        response = sagemaker_utils.describe_workteam(sagemaker_client, workteam_name)
        assert response['Workteam']["CreateDate"] is not None
        assert response['Workteam']["WorkteamName"] == workteam_name
        assert response['Workteam']["WorkteamArn"] is not None
    finally:
        # Cleanup the SageMaker Resources
        sagemaker_utils.delete_workteam(sagemaker_client, workteam_name)
