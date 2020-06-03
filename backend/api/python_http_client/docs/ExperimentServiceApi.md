# kfp_server_api.ExperimentServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**archive_experiment**](ExperimentServiceApi.md#archive_experiment) | **POST** /apis/v1beta1/experiments/{id}:archive | Archives an experiment and the experiment&#39;s runs and jobs.
[**create_experiment**](ExperimentServiceApi.md#create_experiment) | **POST** /apis/v1beta1/experiments | Creates a new experiment.
[**delete_experiment**](ExperimentServiceApi.md#delete_experiment) | **DELETE** /apis/v1beta1/experiments/{id} | Deletes an experiment without deleting the experiment&#39;s runs and jobs. To avoid unexpected behaviors, delete an experiment&#39;s runs and jobs before deleting the experiment.
[**get_experiment**](ExperimentServiceApi.md#get_experiment) | **GET** /apis/v1beta1/experiments/{id} | Finds a specific experiment by ID.
[**list_experiment**](ExperimentServiceApi.md#list_experiment) | **GET** /apis/v1beta1/experiments | Finds all experiments. Supports pagination, and sorting on certain fields.
[**unarchive_experiment**](ExperimentServiceApi.md#unarchive_experiment) | **POST** /apis/v1beta1/experiments/{id}:unarchive | Restores an archived experiment. The experiment&#39;s archived runs and jobs will stay archived.


# **archive_experiment**
> object archive_experiment(id)

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
        api_response = api_instance.archive_experiment(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->archive_experiment: %s\n" % e)
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
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_experiment**
> ApiExperiment create_experiment(body)

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
        api_response = api_instance.create_experiment(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->create_experiment: %s\n" % e)
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
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_experiment**
> object delete_experiment(id)

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
        api_response = api_instance.delete_experiment(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->delete_experiment: %s\n" % e)
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
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_experiment**
> ApiExperiment get_experiment(id)

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
        api_response = api_instance.get_experiment(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->get_experiment: %s\n" % e)
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
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_experiment**
> ApiListExperimentsResponse list_experiment(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)

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
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/ blob/master/backend/api/filter.proto)). (optional)
resource_reference_key_type = 'UNKNOWN_RESOURCE_TYPE' # str | The type of the resource that referred to. (optional) (default to 'UNKNOWN_RESOURCE_TYPE')
resource_reference_key_id = 'resource_reference_key_id_example' # str | The ID of the resource that referred to. (optional)

    try:
        # Finds all experiments. Supports pagination, and sorting on certain fields.
        api_response = api_instance.list_experiment(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->list_experiment: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**| A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/ blob/master/backend/api/filter.proto)). | [optional] 
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
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **unarchive_experiment**
> object unarchive_experiment(id)

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
        api_response = api_instance.unarchive_experiment(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->unarchive_experiment: %s\n" % e)
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
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

