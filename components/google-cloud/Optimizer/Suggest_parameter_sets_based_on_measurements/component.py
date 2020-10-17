from typing import NamedTuple

from kfp.components import create_component_from_func

def suggest_parameter_sets_from_measurements_using_gcp_ai_platform_optimizer(
    parameter_specs: list,
    metrics_for_parameter_sets: list,
    suggestion_count: int,
    maximize: bool = False,
    metric_specs: list = None,
    gcp_project_id: str = None,
    gcp_region: str = "us-central1",
) -> NamedTuple('Outputs', [
    ("suggested_parameter_sets", list),
]):
    """Suggests trials (parameter sets) to evaluate.
    See https://cloud.google.com/ai-platform/optimizer/docs

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>

    Args:
        parameter_specs: List of parameter specs. See https://cloud.google.com/ai-platform/optimizer/docs/reference/rest/v1/projects.locations.studies#parameterspec
        metrics_for_parameter_sets: List of parameter sets and evaluation metrics for them. Each list item contains "parameters" dict and "metrics" dict. Example: {"parameters": {"p1": 1.1, "p2": 2.2}, "metrics": {"metric1": 101, "metric2": 102} }
        maximize: Whether to miaximize or minimize when optimizing a single metric.Default is to minimize. Ignored if metric_specs list is provided.
        metric_specs: List of metric specs. See https://cloud.google.com/ai-platform/optimizer/docs/reference/rest/v1/projects.locations.studies#metricspec
        suggestion_count: Number of suggestions to request.

        suggested_parameter_sets: List of parameter set dictionaries.
    """

    import logging
    import random
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
    studies_api = ml_api.projects().locations().studies()
    trials_api = ml_api.projects().locations().studies().trials()
    operations_api = ml_api.projects().locations().operations()

    random_integer = random.SystemRandom().getrandbits(256)
    study_id = '{:064x}'.format(random_integer)

    if not metric_specs:
        metric_specs=[{
            'metric': 'metric',
            'goal': 'MAXIMIZE' if maximize else 'MINIMIZE',
        }]
    study_config = {
        'algorithm': 'ALGORITHM_UNSPECIFIED',  # Let the service choose the `default` algorithm.
        'parameters': parameter_specs,
        'metrics': metric_specs,
    }
    study = {'study_config': study_config}

    logging.info(f'Creating temporary study {study_id}')
    create_study_request = studies_api.create(
        parent=f'projects/{gcp_project_id}/locations/{gcp_region}',
        studyId=study_id,
        body=study,
    )
    create_study_response = create_study_request.execute()
    study_name = create_study_response['name']

    paremeter_type_names = {parameter_spec['parameter']: parameter_spec['type'] for parameter_spec in parameter_specs}
    def parameter_name_and_value_to_dict(parameter_name: str, parameter_value) -> dict:
        result = {'parameter': parameter_name}
        paremeter_type_name = paremeter_type_names[parameter_name]
        if paremeter_type_name in ['DOUBLE', 'DISCRETE']:
            result['floatValue'] = parameter_value
        elif paremeter_type_name == 'INTEGER':
            result['intValue'] = parameter_value
        elif paremeter_type_name == 'CATEGORICAL':
            result['stringValue'] = parameter_value
        else:
            raise TypeError(f'Unsupported parameter type "{paremeter_type_name}"')
        return result
        
    try:
        logging.info(f'Adding {len(metrics_for_parameter_sets)} measurements to the study.')
        for parameters_and_metrics in metrics_for_parameter_sets:
            parameter_set = parameters_and_metrics['parameters']
            metrics_set = parameters_and_metrics['metrics']
            trial = {
                'parameters': [
                    parameter_name_and_value_to_dict(parameter_name, parameter_value)
                    for parameter_name, parameter_value in parameter_set.items()
                ],
                'finalMeasurement': {
                    'metrics': [
                        {
                            'metric': metric_name,
                            'value': metric_value,
                        }
                        for metric_name, metric_value in metrics_set.items()
                    ],
                },
                'state': 'COMPLETED',
            }
            create_trial_response = trials_api.create(
                parent=fix_resource_name(study_name),
                body=trial,
            ).execute()
            trial_name = create_trial_response["name"]
            logging.info(f'Added trial "{trial_name}" to the study.')

        logging.info(f'Requesting suggestions.')
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
            logging.info('Operation not finished yet: ' + str(get_operation_response))
            time.sleep(10)
        operation_response = get_operation_response['response']
        suggested_trials = operation_response['trials']

        suggested_parameter_sets = [
            {
                parameter['parameter']: parameter.get('floatValue') or parameter.get('intValue') or parameter.get('stringValue') or 0.0
                for parameter in trial['parameters']
            }
            for trial in suggested_trials
        ]
        return (suggested_parameter_sets,)
    finally:
        logging.info(f'Deleting study: "{study_name}"')
        studies_api.delete(name=fix_resource_name(study_name))


if __name__ == '__main__':
    suggest_parameter_sets_from_measurements_using_gcp_ai_platform_optimizer_op = create_component_from_func(
        suggest_parameter_sets_from_measurements_using_gcp_ai_platform_optimizer,
        base_image='python:3.8',
        packages_to_install=['google-api-python-client==1.12.3', 'google-cloud-storage==1.31.2', 'google-auth==1.21.3'],
        output_component_file='component.yaml',
    )
