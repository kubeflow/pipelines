from __future__ import print_function

import time
import kfp_server_api
import os
import requests
from kfp_server_api.rest import ApiException
from pprint import pprint
from kfp_login import get_istio_auth_session
from kfp_namespace import retrieve_namespaces
import json

host = os.getenv("KUBEFLOW_HOST")
username = os.getenv("KUBEFLOW_USERNAME")
password = os.getenv("KUBEFLOW_PASSWORD")

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

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration, cookie=auth_session["session_cookie"]) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    run_id = '<change yours>' # str | The ID of the run to be retrieved.

    try:
        # Finds a specific run by ID.
        api_response = api_instance.get_run(run_id)
        output = api_response.pipeline_runtime.workflow_manifest
        output = json.loads(output)
        nodes = output['status']['nodes']
        conditions = output['status']['conditions'] # Comfirm completion.
        output_value = None

        for node_id, node in nodes.items():
            if 'inputs' in node and 'parameters' in node['inputs']:
                for parameter in node['inputs']['parameters']:
                    if parameter['name'] == 'decision-tree-classifier-Accuracy':
                        output_value = parameter['value']
                        break
        
        if output_value is not None:
            print(f"Decision Tree Classifier Accuracy: {output_value}")
        else:
            print("Parameter not found.")
    except ApiException as e:
        print("Exception when calling RunServiceApi->get_run: %s\n" % e)
