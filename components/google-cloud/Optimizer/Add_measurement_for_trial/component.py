from typing import NamedTuple

from kfp.components import create_component_from_func

def add_measurement_for_trial_in_gcp_ai_platform_optimizer(
    trial_name: str,
    metric_value: float,
    complete_trial: bool = True,
    step_count: float = None,
    gcp_project_id: str = None,
    gcp_region: str = "us-central1",
) -> NamedTuple('Outputs', [
    ("trial_name", list),
    ("trial", dict),
    ("stop_trial", bool),
]):
    """Add measurement for a trial and check whether to continue.
    See https://cloud.google.com/ai-platform/optimizer/docs

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>

    Args:
        trial_name: Full trial resource name.
        metric_value: Result of the trial evaluation.
        step_count: Optional. The number of training steps performed with the model. Can be used when checking early stopping.
        complete_trial: Whether the trial should be completed. Only completed trials are used to suggest new trials. Default is True.
    """

    import logging
    import time

    import google.auth
    from googleapiclient import discovery

    logging.getLogger().setLevel(logging.INFO)

    client_id = 'client1'
    metric_name = 'metric'

    credentials, default_project_id = google.auth.default()

    # Validating and inferring the arguments
    if not gcp_project_id:
        gcp_project_id = default_project_id

    # Building the API client.
    # The main API does not work, so we need to build from the published discovery document.
    def create_caip_optimizer_client(project_id):
        from google.cloud import storage
        _OPTIMIZER_API_DOCUMENT_BUCKET = 'caip-optimizer-public'
        _OPTIMIZER_API_DOCUMENT_FILE = 'api/ml_public_google_rest_v1.json'
        client = storage.Client(project_id)
        bucket = client.get_bucket(_OPTIMIZER_API_DOCUMENT_BUCKET)
        blob = bucket.get_blob(_OPTIMIZER_API_DOCUMENT_FILE)
        discovery_document = blob.download_as_string()
        return discovery.build_from_document(service=discovery_document)

    # Workaround for the Optimizer bug: Optimizer returns resource names that use project number, but only supports resource names with project IDs when making requests
    def get_project_number(project_id):
        service = discovery.build('cloudresourcemanager', 'v1', credentials=credentials)
        response = service.projects().get(projectId=project_id).execute()
        return response['projectNumber']

    gcp_project_number = get_project_number(gcp_project_id)

    def fix_resource_name(name):
        return name.replace(gcp_project_number, gcp_project_id)

    ml_api = create_caip_optimizer_client(gcp_project_id)
    trials_api = ml_api.projects().locations().studies().trials()
    operations_api = ml_api.projects().locations().operations()

    measurement = {
        'measurement': {
            'stepCount': step_count,
            'metrics': [{
                'metric': metric_name,
                'value': metric_value,
            }],
        },
    }
    add_measurement_response = trials_api.addMeasurement(
        name=fix_resource_name(trial_name),
        body=measurement,
    ).execute()
    
    if complete_trial:
        should_stop_trial = True
        complete_response = trials_api.complete(
            name=fix_resource_name(trial_name),
        ).execute()
        return (trial_name, complete_response, should_stop_trial)
    else:
        check_early_stopping_response = trials_api.checkEarlyStoppingState(
            name=fix_resource_name(trial_name),
        ).execute()
        operation_name = check_early_stopping_response['name']
        while True:
            get_operation_response = operations_api.get(
                name=fix_resource_name(operation_name),
            ).execute()
            if get_operation_response.get('done'):
                break
            logging.info('Not finished yet: ' + str(get_operation_response))
            time.sleep(10)
        operation_response = get_operation_response['response']
        should_stop_trial = operation_response['shouldStop']
        return (trial_name, add_measurement_response, should_stop_trial)


if __name__ == '__main__':
    add_measurement_for_trial_in_gcp_ai_platform_optimizer_op = create_component_from_func(
        add_measurement_for_trial_in_gcp_ai_platform_optimizer,
        base_image='python:3.8',
        packages_to_install=['google-api-python-client==1.12.3', 'google-cloud-storage==1.31.2', 'google-auth==1.21.3'],
        output_component_file='component.yaml',
    )
