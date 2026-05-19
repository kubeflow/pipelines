# kfp_server_api.RunServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**batch_update_tasks**](RunServiceApi.md#batch_update_tasks) | **POST** /apis/v2beta1/tasks:batchUpdate | Updates multiple tasks in bulk.
[**create_task**](RunServiceApi.md#create_task) | **POST** /apis/v2beta1/tasks | Creates a new task.
[**get_task**](RunServiceApi.md#get_task) | **GET** /apis/v2beta1/tasks/{task_id} | Gets a specific task by ID.
[**list_tasks**](RunServiceApi.md#list_tasks) | **GET** /apis/v2beta1/tasks | Lists tasks with optional filtering.
[**run_service_archive_run**](RunServiceApi.md#run_service_archive_run) | **POST** /apis/v2beta1/runs/{run_id}:archive | Archives a run in an experiment given by run ID and experiment ID.
[**run_service_create_run**](RunServiceApi.md#run_service_create_run) | **POST** /apis/v2beta1/runs | Creates a new run in an experiment specified by experiment ID. If experiment ID is not specified, the run is created in the default experiment.
[**run_service_delete_run**](RunServiceApi.md#run_service_delete_run) | **DELETE** /apis/v2beta1/runs/{run_id} | Deletes a run in an experiment given by run ID and experiment ID.
[**run_service_get_run**](RunServiceApi.md#run_service_get_run) | **GET** /apis/v2beta1/runs/{run_id} | Finds a specific run by ID.
[**run_service_list_runs**](RunServiceApi.md#run_service_list_runs) | **GET** /apis/v2beta1/runs | Finds all runs in an experiment given by experiment ID. If experiment id is not specified, finds all runs across all experiments.
[**run_service_retry_run**](RunServiceApi.md#run_service_retry_run) | **POST** /apis/v2beta1/runs/{run_id}:retry | Re-initiates a failed or terminated run.
[**run_service_terminate_run**](RunServiceApi.md#run_service_terminate_run) | **POST** /apis/v2beta1/runs/{run_id}:terminate | Terminates an active run.
[**run_service_unarchive_run**](RunServiceApi.md#run_service_unarchive_run) | **POST** /apis/v2beta1/runs/{run_id}:unarchive | Restores an archived run in an experiment given by run ID and experiment ID.
[**update_task**](RunServiceApi.md#update_task) | **PATCH** /apis/v2beta1/tasks/{task_id} | Updates an existing task.


# **batch_update_tasks**
> V2beta1UpdateTasksBulkResponse batch_update_tasks(body)

Updates multiple tasks in bulk.

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
    body = kfp_server_api.V2beta1UpdateTasksBulkRequest() # V2beta1UpdateTasksBulkRequest | 

    try:
        # Updates multiple tasks in bulk.
        api_response = api_instance.batch_update_tasks(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->batch_update_tasks: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1UpdateTasksBulkRequest**](V2beta1UpdateTasksBulkRequest.md)|  | 

### Return type

[**V2beta1UpdateTasksBulkResponse**](V2beta1UpdateTasksBulkResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_task**
> V2beta1PipelineTaskDetail create_task(task)

Creates a new task.

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
    task = kfp_server_api.V2beta1PipelineTaskDetail() # V2beta1PipelineTaskDetail | 

    try:
        # Creates a new task.
        api_response = api_instance.create_task(task)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->create_task: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task** | [**V2beta1PipelineTaskDetail**](V2beta1PipelineTaskDetail.md)|  | 

### Return type

[**V2beta1PipelineTaskDetail**](V2beta1PipelineTaskDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_task**
> V2beta1PipelineTaskDetail get_task(task_id)

Gets a specific task by ID.

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
    task_id = 'task_id_example' # str | 

    try:
        # Gets a specific task by ID.
        api_response = api_instance.get_task(task_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->get_task: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task_id** | **str**|  | 

### Return type

[**V2beta1PipelineTaskDetail**](V2beta1PipelineTaskDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_tasks**
> V2beta1ListTasksResponse list_tasks(parent_id=parent_id, run_id=run_id, namespace=namespace, page_size=page_size, page_token=page_token, filter=filter, order_by=order_by)

Lists tasks with optional filtering.

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
    parent_id = 'parent_id_example' # str | List all tasks with this parent task. (optional)
run_id = 'run_id_example' # str | List all tasks for this run. (optional)
namespace = 'namespace_example' # str | List all tasks in this namespace. The primary use case for this filter is to detect cache hits. (optional)
page_size = 56 # int |  (optional)
page_token = 'page_token_example' # str |  (optional)
filter = 'filter_example' # str |  (optional)
order_by = 'order_by_example' # str |  (optional)

    try:
        # Lists tasks with optional filtering.
        api_response = api_instance.list_tasks(parent_id=parent_id, run_id=run_id, namespace=namespace, page_size=page_size, page_token=page_token, filter=filter, order_by=order_by)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->list_tasks: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **parent_id** | **str**| List all tasks with this parent task. | [optional] 
 **run_id** | **str**| List all tasks for this run. | [optional] 
 **namespace** | **str**| List all tasks in this namespace. The primary use case for this filter is to detect cache hits. | [optional] 
 **page_size** | **int**|  | [optional] 
 **page_token** | **str**|  | [optional] 
 **filter** | **str**|  | [optional] 
 **order_by** | **str**|  | [optional] 

### Return type

[**V2beta1ListTasksResponse**](V2beta1ListTasksResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_archive_run**
> object run_service_archive_run(run_id, experiment_id=experiment_id)

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
    run_id = 'run_id_example' # str | The ID of the run to be archived.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)

    try:
        # Archives a run in an experiment given by run ID and experiment ID.
        api_response = api_instance.run_service_archive_run(run_id, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_archive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be archived. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_create_run**
> V2beta1Run run_service_create_run(run, experiment_id=experiment_id)

Creates a new run in an experiment specified by experiment ID. If experiment ID is not specified, the run is created in the default experiment.

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
    run = kfp_server_api.V2beta1Run() # V2beta1Run | Run to be created.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)

    try:
        # Creates a new run in an experiment specified by experiment ID. If experiment ID is not specified, the run is created in the default experiment.
        api_response = api_instance.run_service_create_run(run, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_create_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run** | [**V2beta1Run**](V2beta1Run.md)| Run to be created. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_delete_run**
> object run_service_delete_run(run_id, experiment_id=experiment_id)

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
    run_id = 'run_id_example' # str | The ID of the run to be deleted.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)

    try:
        # Deletes a run in an experiment given by run ID and experiment ID.
        api_response = api_instance.run_service_delete_run(run_id, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_delete_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be deleted. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_get_run**
> V2beta1Run run_service_get_run(run_id, experiment_id=experiment_id, view=view)

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
    run_id = 'run_id_example' # str | The ID of the run to be retrieved.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)
view = 'DEFAULT' # str | Optional view mode. This field can be used to adjust how detailed the Run object that is returned will be.   - DEFAULT: By default `tasks` field is omitted. This provides a faster and leaner run object.  - FULL: This view mode displays all the tasks for this run with all its fields populated. (optional) (default to 'DEFAULT')

    try:
        # Finds a specific run by ID.
        api_response = api_instance.run_service_get_run(run_id, experiment_id=experiment_id, view=view)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_get_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be retrieved. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 
 **view** | **str**| Optional view mode. This field can be used to adjust how detailed the Run object that is returned will be.   - DEFAULT: By default &#x60;tasks&#x60; field is omitted. This provides a faster and leaner run object.  - FULL: This view mode displays all the tasks for this run with all its fields populated. | [optional] [default to &#39;DEFAULT&#39;]

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_list_runs**
> V2beta1ListRunsResponse run_service_list_runs(namespace=namespace, experiment_id=experiment_id, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, view=view)

Finds all runs in an experiment given by experiment ID. If experiment id is not specified, finds all runs across all experiments.

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
    namespace = 'namespace_example' # str | Optional input field. Filters based on the namespace. (optional)
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. If empty, response includes runs across all experiments. (optional)
page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquired from the nextPageToken field of the response from the previous ListRuns call or can be omitted when fetching the first page. (optional)
page_size = 56 # int | The number of runs to be listed per page. If there are more runs than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" (Example, \"name asc\" or \"id desc\"). Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). (optional)
view = 'DEFAULT' # str | Optional view mode. This field can be used to adjust how detailed the Run object that is returned will be.   - DEFAULT: By default `tasks` field is omitted. This provides a faster and leaner run object.  - FULL: This view mode displays all the tasks for this run with all its fields populated. (optional) (default to 'DEFAULT')

    try:
        # Finds all runs in an experiment given by experiment ID. If experiment id is not specified, finds all runs across all experiments.
        api_response = api_instance.run_service_list_runs(namespace=namespace, experiment_id=experiment_id, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, view=view)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_list_runs: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Optional input field. Filters based on the namespace. | [optional] 
 **experiment_id** | **str**| The ID of the parent experiment. If empty, response includes runs across all experiments. | [optional] 
 **page_token** | **str**| A page token to request the next page of results. The token is acquired from the nextPageToken field of the response from the previous ListRuns call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of runs to be listed per page. If there are more runs than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; (Example, \&quot;name asc\&quot; or \&quot;id desc\&quot;). Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). | [optional] 
 **view** | **str**| Optional view mode. This field can be used to adjust how detailed the Run object that is returned will be.   - DEFAULT: By default &#x60;tasks&#x60; field is omitted. This provides a faster and leaner run object.  - FULL: This view mode displays all the tasks for this run with all its fields populated. | [optional] [default to &#39;DEFAULT&#39;]

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_retry_run**
> object run_service_retry_run(run_id, experiment_id=experiment_id)

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
    run_id = 'run_id_example' # str | The ID of the run to be retried.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)

    try:
        # Re-initiates a failed or terminated run.
        api_response = api_instance.run_service_retry_run(run_id, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_retry_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be retried. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_terminate_run**
> object run_service_terminate_run(run_id, experiment_id=experiment_id)

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
    run_id = 'run_id_example' # str | The ID of the run to be terminated.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)

    try:
        # Terminates an active run.
        api_response = api_instance.run_service_terminate_run(run_id, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_terminate_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be terminated. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_service_unarchive_run**
> object run_service_unarchive_run(run_id, experiment_id=experiment_id)

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
    run_id = 'run_id_example' # str | The ID of the run to be restored.
experiment_id = 'experiment_id_example' # str | The ID of the parent experiment. (optional)

    try:
        # Restores an archived run in an experiment given by run ID and experiment ID.
        api_response = api_instance.run_service_unarchive_run(run_id, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->run_service_unarchive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run to be restored. | 
 **experiment_id** | **str**| The ID of the parent experiment. | [optional] 

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
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_task**
> V2beta1PipelineTaskDetail update_task(task_id, task)

Updates an existing task.

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
    task_id = 'task_id_example' # str | 
task = kfp_server_api.V2beta1PipelineTaskDetail() # V2beta1PipelineTaskDetail | 

    try:
        # Updates an existing task.
        api_response = api_instance.update_task(task_id, task)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RunServiceApi->update_task: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task_id** | **str**|  | 
 **task** | [**V2beta1PipelineTaskDetail**](V2beta1PipelineTaskDetail.md)|  | 

### Return type

[**V2beta1PipelineTaskDetail**](V2beta1PipelineTaskDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

