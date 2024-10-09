from __future__ import print_function
import string
import random
import time
import kfp_server_api
import os
import requests
from kfp_server_api.rest import ApiException

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

def random_suffix() -> string:
    return ''.join(random.choices(string.ascii_lowercase + string.digits, k=10))

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration, cookie=auth_session["session_cookie"]) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.PipelineUploadServiceApi(api_client) 
    uploadfile='<change yours>' # The yaml file in your local path.
    name='pipeline-' + random_suffix()
    description='<change yours>' # str | The description of pipeline.
    try:
        api_response = api_instance.upload_pipeline(uploadfile, name=name, description=description)
        print(api_response)
    except ApiException as e:
        print("Exception when calling PipelineUploadServiceApi->upload_pipeline: %s\n" % e)