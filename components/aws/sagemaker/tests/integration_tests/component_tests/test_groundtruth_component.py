import pytest
import os
import utils
from utils import kfp_client_utils
from utils import sagemaker_utils
from test_workteam_component import create_workteamjob


def create_initial_workteam(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir, download_dir
):
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    test_params["Arguments"]["team_name"] = workteam_name = (
        utils.generate_random_string(5) + "-" + test_params["Arguments"]["team_name"]
    )

    # First create a workteam using a separate pipeline and get the name, arn of the workteam created.
    create_workteamjob(
        kfp_client, test_params, experiment_id, region, sagemaker_client, download_dir,
    )

    workteam_arn = sagemaker_utils.get_workteam_arn(sagemaker_client, workteam_name)

    return workteam_name, workteam_arn


@pytest.mark.parametrize(
    "test_file_dir",
    [
        pytest.param(
            "resources/config/image-classification-groundtruth",
            marks=pytest.mark.canary_test,
        )
    ],
)
def test_groundtruth_labeling_job(
    kfp_client, experiment_id, region, sagemaker_client, test_file_dir
):

    download_dir = utils.mkdir(os.path.join(test_file_dir + "/generated"))
    test_params = utils.load_params(
        utils.replace_placeholders(
            os.path.join(test_file_dir, "config.yaml"),
            os.path.join(download_dir, "config.yaml"),
        )
    )

    workteam_arn = ground_truth_train_job_name = None
    # Verify the GroundTruthJob was created in SageMaker and is InProgress.
    # TODO: Add a bot to complete the labeling job and check for completion instead.
    try:
        workteam_name, workteam_arn = create_initial_workteam(
            kfp_client,
            experiment_id,
            region,
            sagemaker_client,
            "resources/config/create-workteam",
            download_dir,
        )

        test_params["Arguments"]["workteam_arn"] = workteam_arn

        # Generate the ground_truth_train_job_name based on the workteam which will be used for labeling.
        test_params["Arguments"][
            "ground_truth_train_job_name"
        ] = ground_truth_train_job_name = (
            test_params["Arguments"]["ground_truth_train_job_name"]
            + "-by-"
            + workteam_name
        )

        run_id, _, _ = kfp_client_utils.compile_run_monitor_pipeline(
            kfp_client,
            experiment_id,
            test_params["PipelineDefinition"],
            test_params["Arguments"],
            download_dir,
            test_params["TestName"],
            test_params["Timeout"],
            test_params["StatusToCheck"],
        )

        response = sagemaker_utils.describe_labeling_job(
            sagemaker_client, ground_truth_train_job_name
        )
        assert response["LabelingJobStatus"] == "InProgress"

        # Verify that the workteam has the specified labeling job
        labeling_jobs = sagemaker_utils.list_labeling_jobs_for_workteam(
            sagemaker_client, workteam_arn
        )
        assert len(labeling_jobs["LabelingJobSummaryList"]) == 1
        assert (
            labeling_jobs["LabelingJobSummaryList"][0]["LabelingJobName"]
            == ground_truth_train_job_name
        )

        # Test terminate functionality
        print(
            f"Terminating run: {run_id} where GT job_name: {ground_truth_train_job_name}"
        )
        kfp_client_utils.terminate_run(kfp_client, run_id)
        response = sagemaker_utils.describe_labeling_job(
            sagemaker_client, ground_truth_train_job_name
        )
        assert response["LabelingJobStatus"] in ["Completed", "Stopping", "Stopped"]
    finally:
        print(
            f"Clean up workteam: {workteam_arn} and GT job: {ground_truth_train_job_name}"
        )
        if workteam_arn:
            try:
                if ground_truth_train_job_name:
                    # Check if terminate failed, and stop the labeling job
                    labeling_jobs = sagemaker_utils.list_labeling_jobs_for_workteam(
                        sagemaker_client, workteam_arn
                    )
                    # Check for status before terminating because list can return stopping jobs
                    if (
                        len(labeling_jobs["LabelingJobSummaryList"]) > 0
                        and sagemaker_utils.describe_labeling_job(
                            sagemaker_client, ground_truth_train_job_name
                        )["LabelingJobStatus"]
                        == "InProgress"
                    ):
                        sagemaker_utils.stop_labeling_job(
                            sagemaker_client, ground_truth_train_job_name
                        )
            finally:
                # Cleanup the workteam
                workteams = sagemaker_utils.list_workteams(sagemaker_client)[
                    "Workteams"
                ]
                workteam_names = list(map((lambda x: x["WorkteamName"]), workteams))
                if workteam_name in workteam_names:
                    sagemaker_utils.delete_workteam(sagemaker_client, workteam_name)

    # Delete generated files
    utils.remove_dir(download_dir)
