# kfp_server_api.RecurringRunServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**recurring_run_service_create_recurring_run**](RecurringRunServiceApi.md#recurring_run_service_create_recurring_run) | **POST** /apis/v2beta1/recurringruns | Creates a new recurring run in an experiment, given the experiment ID.
[**recurring_run_service_delete_recurring_run**](RecurringRunServiceApi.md#recurring_run_service_delete_recurring_run) | **DELETE** /apis/v2beta1/recurringruns/{recurring_run_id} | Deletes a recurring run.
[**recurring_run_service_disable_recurring_run**](RecurringRunServiceApi.md#recurring_run_service_disable_recurring_run) | **POST** /apis/v2beta1/recurringruns/{recurring_run_id}:disable | Stops a recurring run and all its associated runs. The recurring run is not deleted.
[**recurring_run_service_enable_recurring_run**](RecurringRunServiceApi.md#recurring_run_service_enable_recurring_run) | **POST** /apis/v2beta1/recurringruns/{recurring_run_id}:enable | Restarts a recurring run that was previously stopped. All runs associated with the  recurring run will continue.
[**recurring_run_service_get_recurring_run**](RecurringRunServiceApi.md#recurring_run_service_get_recurring_run) | **GET** /apis/v2beta1/recurringruns/{recurring_run_id} | Finds a specific recurring run by ID.
[**recurring_run_service_list_recurring_runs**](RecurringRunServiceApi.md#recurring_run_service_list_recurring_runs) | **GET** /apis/v2beta1/recurringruns | Finds all recurring runs given experiment and namespace.  If experiment ID is not specified, find all recurring runs across all experiments.


# **recurring_run_service_create_recurring_run**
> V2beta1RecurringRun recurring_run_service_create_recurring_run(body)

Creates a new recurring run in an experiment, given the experiment ID.

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
    api_instance = kfp_server_api.RecurringRunServiceApi(api_client)
    body = kfp_server_api.V2beta1RecurringRun() # V2beta1RecurringRun | The recurring run to be created.

    try:
        # Creates a new recurring run in an experiment, given the experiment ID.
        api_response = api_instance.recurring_run_service_create_recurring_run(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RecurringRunServiceApi->recurring_run_service_create_recurring_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1RecurringRun**](V2beta1RecurringRun.md)| The recurring run to be created. | 

### Return type

[**V2beta1RecurringRun**](V2beta1RecurringRun.md)

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

# **recurring_run_service_delete_recurring_run**
> object recurring_run_service_delete_recurring_run(recurring_run_id)

Deletes a recurring run.

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
    api_instance = kfp_server_api.RecurringRunServiceApi(api_client)
    recurring_run_id = 'recurring_run_id_example' # str | The ID of the recurring run to be deleted.

    try:
        # Deletes a recurring run.
        api_response = api_instance.recurring_run_service_delete_recurring_run(recurring_run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RecurringRunServiceApi->recurring_run_service_delete_recurring_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recurring_run_id** | **str**| The ID of the recurring run to be deleted. | 

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

# **recurring_run_service_disable_recurring_run**
> object recurring_run_service_disable_recurring_run(recurring_run_id)

Stops a recurring run and all its associated runs. The recurring run is not deleted.

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
    api_instance = kfp_server_api.RecurringRunServiceApi(api_client)
    recurring_run_id = 'recurring_run_id_example' # str | The ID of the recurring runs to be disabled.

    try:
        # Stops a recurring run and all its associated runs. The recurring run is not deleted.
        api_response = api_instance.recurring_run_service_disable_recurring_run(recurring_run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RecurringRunServiceApi->recurring_run_service_disable_recurring_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recurring_run_id** | **str**| The ID of the recurring runs to be disabled. | 

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

# **recurring_run_service_enable_recurring_run**
> object recurring_run_service_enable_recurring_run(recurring_run_id)

Restarts a recurring run that was previously stopped. All runs associated with the  recurring run will continue.

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
    api_instance = kfp_server_api.RecurringRunServiceApi(api_client)
    recurring_run_id = 'recurring_run_id_example' # str | The ID of the recurring runs to be enabled.

    try:
        # Restarts a recurring run that was previously stopped. All runs associated with the  recurring run will continue.
        api_response = api_instance.recurring_run_service_enable_recurring_run(recurring_run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RecurringRunServiceApi->recurring_run_service_enable_recurring_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recurring_run_id** | **str**| The ID of the recurring runs to be enabled. | 

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

# **recurring_run_service_get_recurring_run**
> V2beta1RecurringRun recurring_run_service_get_recurring_run(recurring_run_id)

Finds a specific recurring run by ID.

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
    api_instance = kfp_server_api.RecurringRunServiceApi(api_client)
    recurring_run_id = 'recurring_run_id_example' # str | The ID of the recurring run to be retrieved.

    try:
        # Finds a specific recurring run by ID.
        api_response = api_instance.recurring_run_service_get_recurring_run(recurring_run_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RecurringRunServiceApi->recurring_run_service_get_recurring_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recurring_run_id** | **str**| The ID of the recurring run to be retrieved. | 

### Return type

[**V2beta1RecurringRun**](V2beta1RecurringRun.md)

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

# **recurring_run_service_list_recurring_runs**
> V2beta1ListRecurringRunsResponse recurring_run_service_list_recurring_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, namespace=namespace, filter=filter, experiment_id=experiment_id)

Finds all recurring runs given experiment and namespace.  If experiment ID is not specified, find all recurring runs across all experiments.

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
    api_instance = kfp_server_api.RecurringRunServiceApi(api_client)
    page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquired from the nextPageToken field of the response from the previous ListRecurringRuns call or can be omitted when fetching the first page. (optional)
page_size = 56 # int | The number of recurring runs to be listed per page. If there are more recurring runs  than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
sort_by = 'sort_by_example' # str | Can be formatted as \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default. (optional)
namespace = 'namespace_example' # str | Optional input. The namespace the recurring runs belong to. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). (optional)
experiment_id = 'experiment_id_example' # str | The ID of the experiment to be retrieved. If empty, list recurring runs across all experiments. (optional)

    try:
        # Finds all recurring runs given experiment and namespace.  If experiment ID is not specified, find all recurring runs across all experiments.
        api_response = api_instance.recurring_run_service_list_recurring_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, namespace=namespace, filter=filter, experiment_id=experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling RecurringRunServiceApi->recurring_run_service_list_recurring_runs: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**| A page token to request the next page of results. The token is acquired from the nextPageToken field of the response from the previous ListRecurringRuns call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of recurring runs to be listed per page. If there are more recurring runs  than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be formatted as \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot;. Ascending by default. | [optional] 
 **namespace** | **str**| Optional input. The namespace the recurring runs belong to. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). | [optional] 
 **experiment_id** | **str**| The ID of the experiment to be retrieved. If empty, list recurring runs across all experiments. | [optional] 

### Return type

[**V2beta1ListRecurringRunsResponse**](V2beta1ListRecurringRunsResponse.md)

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

