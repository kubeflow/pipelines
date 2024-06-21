from __future__ import print_function

import time
import kfp_server_api
import os
import requests
import string
import random
import json
from kfp_server_api.rest import ApiException
from pprint import pprint
from kfp_login import get_istio_auth_session
from kfp_namespace import retrieve_namespaces

host = "http://140.128.102.163:31740"
username = "kubeflow02@gmail.com"
password = "tkiizd"

auth_session = get_istio_auth_session(
        url=host,
        username=username,
        password=password
    )

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = os.path.join(host, "pipeline"),
)
configuration.debug = True

namespaces = retrieve_namespaces(host, auth_session)
#print("available namespace: {}".format(namespaces))

def random_suffix() -> string:
    return ''.join(random.choices(string.ascii_lowercase + string.digits, k=10))

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration, cookie=auth_session["session_cookie"]) as api_client:
    # Create an instance of the  Experiment API class
    experiment_api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    name="experiment-" + random_suffix()
    description="This is a experiment for only_logistic_regression."
    resource_reference_key_id = namespaces[0]
    resource_references=[kfp_server_api.models.ApiResourceReference(
        key=kfp_server_api.models.ApiResourceKey(
            type=kfp_server_api.models.ApiResourceType.NAMESPACE,
            id=resource_reference_key_id
        ),
        relationship=kfp_server_api.models.ApiRelationship.OWNER
    )]
    body = kfp_server_api.ApiExperiment(name=name, description=description, resource_references=resource_references) # ApiExperiment | The experiment to be created.
    try:
        # Creates a new experiment.
        experiment_api_response = experiment_api_instance.create_experiment(body)
        experiment_id = experiment_api_response.id # str | The ID of the run to be retrieved.
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->create_experiment: %s\n" % e)
    
    # Create an instance of the pipeline API class
    api_instance = kfp_server_api.PipelineUploadServiceApi(api_client) 
    uploadfile='../pipelines_yamls/only_logistic_regression.yaml'
    name='pipeline-' + random_suffix()
    description="This is a only_logistic_regression pipline."
    try:
        pipeline_api_response = api_instance.upload_pipeline(uploadfile, name=name, description=description)
        pipeline_id = pipeline_api_response.id # str | The ID of the run to be retrieved.
    except ApiException as e:
        print("Exception when calling PipelineUploadServiceApi->upload_pipeline: %s\n" % e)

    # Create an instance of the run API class
    run_api_instance = kfp_server_api.RunServiceApi(api_client)
    display_name = 'run_only_logistic_regression' + random_suffix()
    description = "This is a only_logistic_regression run."
    pipeline_spec = kfp_server_api.ApiPipelineSpec(pipeline_id=pipeline_id)
    resource_reference_key_id = namespaces[0]
    resource_references=[kfp_server_api.models.ApiResourceReference(
    key=kfp_server_api.models.ApiResourceKey(id=experiment_id, type=kfp_server_api.models.ApiResourceType.EXPERIMENT),
    relationship=kfp_server_api.models.ApiRelationship.OWNER )]
    body = kfp_server_api.ApiRun(name=display_name, description=description, pipeline_spec=pipeline_spec, resource_references=resource_references) # ApiRun | 
    try:
        # Creates a new run.
        run_api_response = run_api_instance.create_run(body)
        run_id = run_api_response.run.id # str | The ID of the run to be retrieved.
    except ApiException as e:
        print("Exception when calling RunServiceApi->create_run: %s\n" % e)

    Completed_flag = False
    polling_interval = 10  # Time in seconds between polls

    while not Completed_flag:
        try:
            time.sleep(1)
            # Finds a specific run by ID.
            api_instance = run_api_instance.get_run(run_id)
            output = api_instance.pipeline_runtime.workflow_manifest
            output = json.loads(output)

            try:
                nodes = output['status']['nodes']
                conditions = output['status']['conditions'] # Comfirm completion.
                
            except KeyError:
                nodes = {}
                conditions = []

            output_value = None
            Completed_flag = conditions[1]['status'] if len(conditions) > 1 else False

        except ApiException as e:
            print("Exception when calling RunServiceApi->get_run: %s\n" % e)
            break

        if not Completed_flag:
            print("Pipeline is still running. Waiting...")
            time.sleep(polling_interval-1)

        for node_id, node in nodes.items():
                if 'inputs' in node and 'parameters' in node['inputs']:
                    for parameter in node['inputs']['parameters']:
                        if parameter['name'] == 'logistic-regression-classifier-Accuracy':
                            output_value = parameter['value']
                            
    if output_value is not None:
        print(f"Logistic Regression Classifier Accuracy: {output_value}")
    else:
        print("Parameter not found.")
        print(nodes)
