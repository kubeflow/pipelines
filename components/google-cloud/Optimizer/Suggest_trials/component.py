from typing import NamedTuple

from kfp.components import create_component_from_func

def suggest_trials_in_gcp_ai_platform_optimizer(
    study_name: str,
    suggestion_count: int,
    gcp_project_id: str = None,
    gcp_region: str = "us-central1",
) -> NamedTuple('Outputs', [
    ("suggested_trials", list),
]):
    """Suggests trials (parameter sets) to evaluate.
    See https://cloud.google.com/ai-platform/optimizer/docs

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>

    Args:
        study_name: Full resource name of the study.
        suggestion_count: Number of suggestions to request.
    """

    import logging
    import time

    import google.auth
    from googleapiclient import discovery

    logging.getLogger().setLevel(logging.INFO)

    client_id = 'client1'

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

    suggest_trials_request = trials_api.suggest(
        parent=fix_resource_name(study_name),
        body=dict(
            suggestionCount=suggestion_count,
            clientId=client_id,
        ),
    )
    suggest_trials_response = suggest_trials_request.execute()
    operation_name = suggest_trials_response['name']
    while True:
        get_operation_response = operations_api.get(
            name=fix_resource_name(operation_name),
        ).execute()
        # Knowledge: The "done" key is just missing until the result is available
        if get_operation_response.get('done'):
            break
        logging.info('Not finished yet: ' + str(get_operation_response))
        time.sleep(10)
    operation_response = get_operation_response['response']
    suggested_trials = operation_response['trials']
    return (suggested_trials,)


if __name__ == '__main__':
    suggest_trials_in_gcp_ai_platform_optimizer_op = create_component_from_func(
        suggest_trials_in_gcp_ai_platform_optimizer,
        base_image='python:3.8',
        packages_to_install=['google-api-python-client==1.12.3', 'google-cloud-storage==1.31.2', 'google-auth==1.21.3'],
        output_component_file='component.yaml',
    )
