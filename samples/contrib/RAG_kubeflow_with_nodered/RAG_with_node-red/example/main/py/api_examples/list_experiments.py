from __future__ import print_function

import time
import kfp_server_api
import os
import requests
from kfp_server_api.rest import ApiException
from pprint import pprint
from kfp_login import get_istio_auth_session
from kfp_namespace import retrieve_namespaces

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
print("available namespace: {}".format(namespaces))

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration, cookie=auth_session["session_cookie"]) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    resource_reference_key_type = "NAMESPACE"
    resource_reference_key_id = namespaces[0]
    list_experiment_response = api_instance.list_experiment(resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)
    for experiment in list_experiment_response.experiments:
        pprint(experiment)
