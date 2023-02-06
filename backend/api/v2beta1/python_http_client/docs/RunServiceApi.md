# kfp_server_api.RunServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**archive_run**](RunServiceApi.md#archive_run) | **POST** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}:archive | Archives a run in an experiment given by run ID and experiment ID.
[**create_run**](RunServiceApi.md#create_run) | **POST** /apis/v2beta1/experiments/{experiment_id}/runs | Creates a new run in an experiment specified by experiment ID.  If experiment ID is not specified, the run is created in the default experiment.
[**delete_run**](RunServiceApi.md#delete_run) | **DELETE** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id} | Deletes a run in an experiment given by run ID and experiment ID.
[**get_run**](RunServiceApi.md#get_run) | **GET** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id} | Finds a specific run by ID.
[**list_runs**](RunServiceApi.md#list_runs) | **GET** /apis/v2beta1/experiments/{experiment_id}/runs | Finds all runs in an experiment given by experiment ID.  If experiment id is not specified, finds all runs across all experiments.
[**read_artifact**](RunServiceApi.md#read_artifact) | **GET** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read | Finds artifact data in a run.
[**retry_run**](RunServiceApi.md#retry_run) | **POST** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}:retry | Re-initiates a failed or terminated run.
[**terminate_run**](RunServiceApi.md#terminate_run) | **POST** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}:terminate | Terminates an active run.
[**unarchive_run**](RunServiceApi.md#unarchive_run) | **POST** /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}:unarchive | Restores an archived run in an experiment given by run ID and experiment ID.


# **archive_run**
> object archive_run(experiment_id, run_id)

Archives a run in an experiment given by run ID and experiment ID.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | The ID of the run to be archived.

    try:
        # Archives a run in an experiment given by run ID and experiment ID.
        api_response = api_instance.archive_run(experiment_id, run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->archive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| The ID of the run to be archived. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_run**
> V2beta1Run create_run(experiment_id, body)

Creates a new run in an experiment specified by experiment ID.  If experiment ID is not specified, the run is created in the default experiment.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
body = kfp_server_api.V2beta1Run() # V2beta1Run | Run to be created.

    try:
        # Creates a new run in an experiment specified by experiment ID.  If experiment ID is not specified, the run is created in the default experiment.
        api_response = api_instance.create_run(experiment_id, body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->create_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **body** | [**V2beta1Run**](V2beta1Run.md)| Run to be created. | 

### Return type

[**V2beta1Run**](V2beta1Run.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_run**
> object delete_run(experiment_id, run_id)

Deletes a run in an experiment given by run ID and experiment ID.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | The ID of the run to be deleted.

    try:
        # Deletes a run in an experiment given by run ID and experiment ID.
        api_response = api_instance.delete_run(experiment_id, run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->delete_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| The ID of the run to be deleted. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_run**
> V2beta1Run get_run(experiment_id, run_id)

Finds a specific run by ID.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | The ID of the run to be retrieved.

    try:
        # Finds a specific run by ID.
        api_response = api_instance.get_run(experiment_id, run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->get_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| The ID of the run to be retrieved. | 

### Return type

[**V2beta1Run**](V2beta1Run.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_runs**
> V2beta1ListRunsResponse list_runs(experiment_id, namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Finds all runs in an experiment given by experiment ID.  If experiment id is not specified, finds all runs across all experiments.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. If empty, response includes runs across all experiments.
namespace = 'namespace_example' # str | Optional input field. Filters based on the namespace. (optional)
page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquired from the nextPageToken field of the response from the previous ListRuns call or can be omitted when fetching the first page. (optional)
page_size = 56 # int | The number of runs to be listed per page. If there are more runs than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" (Example, \"name asc\" or \"id desc\"). Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). (optional)

    try:
        # Finds all runs in an experiment given by experiment ID.  If experiment id is not specified, finds all runs across all experiments.
        api_response = api_instance.list_runs(experiment_id, namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->list_runs: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. If empty, response includes runs across all experiments. | 
 **namespace** | **str**| Optional input field. Filters based on the namespace. | [optional] 
 **page_token** | **str**| A page token to request the next page of results. The token is acquired from the nextPageToken field of the response from the previous ListRuns call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of runs to be listed per page. If there are more runs than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; (Example, \&quot;name asc\&quot; or \&quot;id desc\&quot;). Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). | [optional] 

### Return type

[**V2beta1ListRunsResponse**](V2beta1ListRunsResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **read_artifact**
> V2beta1ReadArtifactResponse read_artifact(experiment_id, run_id, node_id, artifact_name)

Finds artifact data in a run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | ID of the run.
node_id = 'node_id_example' # str | ID of the running node.
artifact_name = 'artifact_name_example' # str | Name of the artifact.

    try:
        # Finds artifact data in a run.
        api_response = api_instance.read_artifact(experiment_id, run_id, node_id, artifact_name)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->read_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| ID of the run. | 
 **node_id** | **str**| ID of the running node. | 
 **artifact_name** | **str**| Name of the artifact. | 

### Return type

[**V2beta1ReadArtifactResponse**](V2beta1ReadArtifactResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **retry_run**
> object retry_run(experiment_id, run_id)

Re-initiates a failed or terminated run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | The ID of the run to be retried.

    try:
        # Re-initiates a failed or terminated run.
        api_response = api_instance.retry_run(experiment_id, run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->retry_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| The ID of the run to be retried. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **terminate_run**
> object terminate_run(experiment_id, run_id)

Terminates an active run.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | The ID of the run to be terminated.

    try:
        # Terminates an active run.
        api_response = api_instance.terminate_run(experiment_id, run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->terminate_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| The ID of the run to be terminated. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **unarchive_run**
> object unarchive_run(experiment_id, run_id)

Restores an archived run in an experiment given by run ID and experiment ID.

### Example

* Api Key Authentication (Bearer):
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = kfp_server_api.Configuration(
    host = "http://localhost"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration(
    host = "http://localhost",
    api_key = {
        'authorization': 'YOUR_API_KEY'
    }
)
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# Enter a context with an instance of the API client
with kfp_server_api.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.RunServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the parent experiment.
run_id = 'run_id_example' # str | The ID of the run to be restored.

    try:
        # Restores an archived run in an experiment given by run ID and experiment ID.
        api_response = api_instance.unarchive_run(experiment_id, run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->unarchive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the parent experiment. | 
 **run_id** | **str**| The ID of the run to be restored. | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

