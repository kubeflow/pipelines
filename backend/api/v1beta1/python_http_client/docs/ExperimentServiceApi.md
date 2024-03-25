# kfp_server_api.ExperimentServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**experiment_service_archive_experiment_v1**](ExperimentServiceApi.md#experiment_service_archive_experiment_v1) | **POST** /apis/v1beta1/experiments/{id}:archive | Archives an experiment and the experiment&#39;s runs and jobs.
[**experiment_service_create_experiment_v1**](ExperimentServiceApi.md#experiment_service_create_experiment_v1) | **POST** /apis/v1beta1/experiments | Creates a new experiment.
[**experiment_service_delete_experiment_v1**](ExperimentServiceApi.md#experiment_service_delete_experiment_v1) | **DELETE** /apis/v1beta1/experiments/{id} | Deletes an experiment without deleting the experiment&#39;s runs and jobs. To avoid unexpected behaviors, delete an experiment&#39;s runs and jobs before deleting the experiment.
[**experiment_service_get_experiment_v1**](ExperimentServiceApi.md#experiment_service_get_experiment_v1) | **GET** /apis/v1beta1/experiments/{id} | Finds a specific experiment by ID.
[**experiment_service_list_experiments_v1**](ExperimentServiceApi.md#experiment_service_list_experiments_v1) | **GET** /apis/v1beta1/experiments | Finds all experiments. Supports pagination, and sorting on certain fields.
[**experiment_service_unarchive_experiment_v1**](ExperimentServiceApi.md#experiment_service_unarchive_experiment_v1) | **POST** /apis/v1beta1/experiments/{id}:unarchive | Restores an archived experiment. The experiment&#39;s archived runs and jobs will stay archived.


# **experiment_service_archive_experiment_v1**
> object experiment_service_archive_experiment_v1(id)

Archives an experiment and the experiment's runs and jobs.

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
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    id = 'id_example' # str | The ID of the experiment to be archived.

    try:
        # Archives an experiment and the experiment's runs and jobs.
        api_response = api_instance.experiment_service_archive_experiment_v1(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->experiment_service_archive_experiment_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the experiment to be archived. | 

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

# **experiment_service_create_experiment_v1**
> ApiExperiment experiment_service_create_experiment_v1(body)

Creates a new experiment.

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
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    body = kfp_server_api.ApiExperiment() # ApiExperiment | The experiment to be created.

    try:
        # Creates a new experiment.
        api_response = api_instance.experiment_service_create_experiment_v1(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->experiment_service_create_experiment_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**ApiExperiment**](ApiExperiment.md)| The experiment to be created. | 

### Return type

[**ApiExperiment**](ApiExperiment.md)

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

# **experiment_service_delete_experiment_v1**
> object experiment_service_delete_experiment_v1(id)

Deletes an experiment without deleting the experiment's runs and jobs. To avoid unexpected behaviors, delete an experiment's runs and jobs before deleting the experiment.

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
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    id = 'id_example' # str | The ID of the experiment to be deleted.

    try:
        # Deletes an experiment without deleting the experiment's runs and jobs. To avoid unexpected behaviors, delete an experiment's runs and jobs before deleting the experiment.
        api_response = api_instance.experiment_service_delete_experiment_v1(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->experiment_service_delete_experiment_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the experiment to be deleted. | 

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

# **experiment_service_get_experiment_v1**
> ApiExperiment experiment_service_get_experiment_v1(id)

Finds a specific experiment by ID.

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
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    id = 'id_example' # str | The ID of the experiment to be retrieved.

    try:
        # Finds a specific experiment by ID.
        api_response = api_instance.experiment_service_get_experiment_v1(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->experiment_service_get_experiment_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the experiment to be retrieved. | 

### Return type

[**ApiExperiment**](ApiExperiment.md)

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

# **experiment_service_list_experiments_v1**
> ApiListExperimentsResponse experiment_service_list_experiments_v1(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)

Finds all experiments. Supports pagination, and sorting on certain fields.

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
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page. (optional)
page_size = 56 # int | The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)). (optional)
resource_reference_key_type = 'UNKNOWN_RESOURCE_TYPE' # str | The type of the resource that referred to. (optional) (default to 'UNKNOWN_RESOURCE_TYPE')
resource_reference_key_id = 'resource_reference_key_id_example' # str | The ID of the resource that referred to. (optional)

    try:
        # Finds all experiments. Supports pagination, and sorting on certain fields.
        api_response = api_instance.experiment_service_list_experiments_v1(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->experiment_service_list_experiments_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**| A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)). | [optional] 
 **resource_reference_key_type** | **str**| The type of the resource that referred to. | [optional] [default to &#39;UNKNOWN_RESOURCE_TYPE&#39;]
 **resource_reference_key_id** | **str**| The ID of the resource that referred to. | [optional] 

### Return type

[**ApiListExperimentsResponse**](ApiListExperimentsResponse.md)

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

# **experiment_service_unarchive_experiment_v1**
> object experiment_service_unarchive_experiment_v1(id)

Restores an archived experiment. The experiment's archived runs and jobs will stay archived.

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
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    id = 'id_example' # str | The ID of the experiment to be restored.

    try:
        # Restores an archived experiment. The experiment's archived runs and jobs will stay archived.
        api_response = api_instance.experiment_service_unarchive_experiment_v1(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->experiment_service_unarchive_experiment_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the experiment to be restored. | 

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

