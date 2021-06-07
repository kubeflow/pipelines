def create_custom_job(
  type,
  gcp_project,
  gcp_region,
  payload,
  gcp_resources,
  ):
  import logging
  import subprocess
  import time
  from google.cloud import aiplatform
  from google.protobuf import json_format
  from google.protobuf.struct_pb2 import Value
  from google.cloud.aiplatform.compat.types import job_state as gca_job_state

  _POLLING_INTERVAL_IN_SECONDS = 20
  _CONNECTION_ERROR_RETRY_LIMIT = 5

  _JOB_COMPLETE_STATES = (
      gca_job_state.JobState.JOB_STATE_SUCCEEDED,
      gca_job_state.JobState.JOB_STATE_FAILED,
      gca_job_state.JobState.JOB_STATE_CANCELLED,
      gca_job_state.JobState.JOB_STATE_PAUSED,
  )

  _JOB_ERROR_STATES = (
      gca_job_state.JobState.JOB_STATE_FAILED,
      gca_job_state.JobState.JOB_STATE_CANCELLED,
  )

  client_options = {"api_endpoint": gcp_region+'-aiplatform.googleapis.com'}
  # Initialize client that will be used to create and send requests.
  job_client = aiplatform.gapic.JobServiceClient(client_options=client_options)

  parent = f"projects/{gcp_project}/locations/{gcp_region}"
  job_spec = {
      "display_name": 'yang-customjob-fpc-test-1',
      'job_spec': {
          "worker_pool_specs": [
              {
                  "replica_count": 1,
                  "machine_spec": {
                      "machine_type": "n1-standard-4",
                  },
                  "container_spec": {
                      "image_uri": "busybox",
                      "command": [
                          "sleep",
                          "60"
                      ],
                  },
              }
          ]
      },
      'labels': {'vertex-pipeline-job-id': '123'},
  }
  create_custom_job_response = job_client.create_custom_job(parent=parent, custom_job=job_spec)
  custom_job_name = create_custom_job_response.name

  # Write the job id to output
  with open(gcp_resources, 'w') as f:
    f.write(custom_job_name)

  # Poll the job status
  get_custom_job_response = job_client.get_custom_job(name=custom_job_name)
  retry_count = 0

  while get_custom_job_response.state not in _JOB_COMPLETE_STATES:
    time.sleep(_POLLING_INTERVAL_IN_SECONDS)

    try:
      get_custom_job_response = job_client.get_custom_job(name=custom_job_name)
      logging.info('GetCustomJob response state =%s', get_custom_job_response.state)
      retry_count = 0
    # Handle transient connection error.
    except ConnectionError as err:
      if retry_count < _CONNECTION_ERROR_RETRY_LIMIT:
        retry_count += 1
        logging.warning(
            'ConnectionError (%s) encountered when polling job: %s. Trying to '
            'recreate the API client.', err, custom_job_name)
        # Recreate the Python API client.
        job_client = aiplatform.gapic.JobServiceClient(client_options=client_options)
        get_custom_job_response = job_client.get_custom_job(name=custom_job_name)
      else:
        logging.error('Request failed after %s retries.',
                      _CONNECTION_ERROR_RETRY_LIMIT)
        raise

  if get_custom_job_response.state in _JOB_ERROR_STATES:
    raise RuntimeError("Job failed with:\n%s" % get_custom_job_response.state)
  else:
    logging.info('CustomJob %s completed with response state =%s', custom_job_name, get_custom_job_response.state)
