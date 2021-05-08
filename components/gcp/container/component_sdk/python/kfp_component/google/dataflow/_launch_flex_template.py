import logging

from google.cloud import storage

from kfp_component.core import KfpExecutionContext
from ._client import DataflowClient
from ._common_ops import (
    wait_and_dump_job, get_staging_location, read_job_id_and_location,
    upload_job_id_and_location
)


def launch_flex_template(
    project_id,
    location,
    launch_parameters,
    validate_only=False,
    staging_dir=None,
    wait_interval=30,
    job_id_output_path='/tmp/kfp/output/dataflow/job_id.txt',
    job_object_output_path='/tmp/kfp/output/dataflow/job.json',
):
    """Launches a dataflow job from a flex template.

    Args:
        project_id (str): Required. The ID of the Cloud Platform project that the job belongs to.
        location (str): The regional endpoint to which to direct the request.
        launch_parameters (dict): Parameters to provide to the template
            being launched. Schema defined in
            https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.locations.flexTemplates/launch#LaunchFlexTemplateParameter.
            `jobName` will be replaced by generated name.
        validate_only (boolean): If true, the request is validated but
            not actually executed. Defaults to false.
        staging_dir (str): Optional. The GCS directory for keeping staging files.
            A random subdirectory will be created under the directory to keep job info
            for resuming the job in case of failure.
        wait_interval (int): The wait seconds between polling.
        job_id_output_path (str): Optional. Output file to save job_id of execution
        job_object_output_path (str): Optional. Output file to save job details of execution

    Returns:
        The completed job.
    """
    storage_client = storage.Client()
    df_client = DataflowClient()
    job_id = None

    def cancel():
        if job_id:
            df_client.cancel_job(project_id, job_id, location)

    with KfpExecutionContext(on_cancel=cancel) as ctx:
        staging_location = get_staging_location(staging_dir, ctx.context_id())
        job_id, _ = read_job_id_and_location(storage_client, staging_location)
        # Continue waiting for the job if it's has been uploaded to staging location.
        if job_id:
            job = df_client.get_job(project_id, job_id, location)
            job = wait_and_dump_job(
                df_client,
                project_id,
                location,
                job,
                wait_interval,
                job_id_output_path=job_id_output_path,
                job_object_output_path=job_object_output_path,
            )
            logging.info(f'Skipping, existing job: {job}')
            return job

        if launch_parameters is None:
            launch_parameters = {}

        request_body = {
            'launchParameter': launch_parameters,
            'validateOnly': validate_only
        }

        request_body['launchParameter']['jobName'] = 'job-' + ctx.context_id()

        response = df_client.launch_flex_template(
            project_id, request_body, location
        )

        job = response.get('job', None)
        if not job:
            # Validate only mode
            return job

        job_id = job.get('id')
        upload_job_id_and_location(
            storage_client, staging_location, job_id, location
        )
        job = wait_and_dump_job(
            df_client,
            project_id,
            location,
            job,
            wait_interval,
            job_id_output_path=job_id_output_path,
            job_object_output_path=job_object_output_path,
        )
        logging.info(f'Completed job: {job}')
        return job
