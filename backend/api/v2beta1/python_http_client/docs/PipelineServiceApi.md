# kfp_server_api.PipelineServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**pipeline_service_create_pipeline**](PipelineServiceApi.md#pipeline_service_create_pipeline) | **POST** /apis/v2beta1/pipelines | Creates a pipeline.
[**pipeline_service_create_pipeline_and_version**](PipelineServiceApi.md#pipeline_service_create_pipeline_and_version) | **POST** /apis/v2beta1/pipelines/create | Creates a new pipeline and a new pipeline version in a single transaction.
[**pipeline_service_create_pipeline_version**](PipelineServiceApi.md#pipeline_service_create_pipeline_version) | **POST** /apis/v2beta1/pipelines/{pipeline_id}/versions | Adds a pipeline version to the specified pipeline ID.
[**pipeline_service_delete_pipeline**](PipelineServiceApi.md#pipeline_service_delete_pipeline) | **DELETE** /apis/v2beta1/pipelines/{pipeline_id} | Deletes an empty pipeline by ID. Returns error if the pipeline has pipeline versions.
[**pipeline_service_delete_pipeline_version**](PipelineServiceApi.md#pipeline_service_delete_pipeline_version) | **DELETE** /apis/v2beta1/pipelines/{pipeline_id}/versions/{pipeline_version_id} | Deletes a specific pipeline version by pipeline version ID and pipeline ID.
[**pipeline_service_get_pipeline**](PipelineServiceApi.md#pipeline_service_get_pipeline) | **GET** /apis/v2beta1/pipelines/{pipeline_id} | Finds a specific pipeline by ID.
[**pipeline_service_get_pipeline_by_name**](PipelineServiceApi.md#pipeline_service_get_pipeline_by_name) | **GET** /apis/v2beta1/pipelines/names/{name} | Finds a specific pipeline by name and namespace.
[**pipeline_service_get_pipeline_version**](PipelineServiceApi.md#pipeline_service_get_pipeline_version) | **GET** /apis/v2beta1/pipelines/{pipeline_id}/versions/{pipeline_version_id} | Gets a pipeline version by pipeline version ID and pipeline ID.
[**pipeline_service_list_pipeline_versions**](PipelineServiceApi.md#pipeline_service_list_pipeline_versions) | **GET** /apis/v2beta1/pipelines/{pipeline_id}/versions | Lists all pipeline versions of a given pipeline ID.
[**pipeline_service_list_pipelines**](PipelineServiceApi.md#pipeline_service_list_pipelines) | **GET** /apis/v2beta1/pipelines | Finds all pipelines within a namespace.


# **pipeline_service_create_pipeline**
> V2beta1Pipeline pipeline_service_create_pipeline(body)

Creates a pipeline.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    body = kfp_server_api.V2beta1Pipeline() # V2beta1Pipeline | Required input. Pipeline that needs to be created.

    try:
        # Creates a pipeline.
        api_response = api_instance.pipeline_service_create_pipeline(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_create_pipeline: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1Pipeline**](V2beta1Pipeline.md)| Required input. Pipeline that needs to be created. | 

### Return type

[**V2beta1Pipeline**](V2beta1Pipeline.md)

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

# **pipeline_service_create_pipeline_and_version**
> V2beta1Pipeline pipeline_service_create_pipeline_and_version(body)

Creates a new pipeline and a new pipeline version in a single transaction.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    body = kfp_server_api.V2beta1CreatePipelineAndVersionRequest() # V2beta1CreatePipelineAndVersionRequest | 

    try:
        # Creates a new pipeline and a new pipeline version in a single transaction.
        api_response = api_instance.pipeline_service_create_pipeline_and_version(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_create_pipeline_and_version: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1CreatePipelineAndVersionRequest**](V2beta1CreatePipelineAndVersionRequest.md)|  | 

### Return type

[**V2beta1Pipeline**](V2beta1Pipeline.md)

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

# **pipeline_service_create_pipeline_version**
> V2beta1PipelineVersion pipeline_service_create_pipeline_version(pipeline_id, body)

Adds a pipeline version to the specified pipeline ID.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    pipeline_id = 'pipeline_id_example' # str | Required input. ID of the parent pipeline.
body = kfp_server_api.V2beta1PipelineVersion() # V2beta1PipelineVersion | Required input. Pipeline version ID to be created.

    try:
        # Adds a pipeline version to the specified pipeline ID.
        api_response = api_instance.pipeline_service_create_pipeline_version(pipeline_id, body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_create_pipeline_version: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| Required input. ID of the parent pipeline. | 
 **body** | [**V2beta1PipelineVersion**](V2beta1PipelineVersion.md)| Required input. Pipeline version ID to be created. | 

### Return type

[**V2beta1PipelineVersion**](V2beta1PipelineVersion.md)

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

# **pipeline_service_delete_pipeline**
> object pipeline_service_delete_pipeline(pipeline_id)

Deletes an empty pipeline by ID. Returns error if the pipeline has pipeline versions.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    pipeline_id = 'pipeline_id_example' # str | Required input. ID of the pipeline to be deleted.

    try:
        # Deletes an empty pipeline by ID. Returns error if the pipeline has pipeline versions.
        api_response = api_instance.pipeline_service_delete_pipeline(pipeline_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_delete_pipeline: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| Required input. ID of the pipeline to be deleted. | 

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

# **pipeline_service_delete_pipeline_version**
> object pipeline_service_delete_pipeline_version(pipeline_id, pipeline_version_id)

Deletes a specific pipeline version by pipeline version ID and pipeline ID.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    pipeline_id = 'pipeline_id_example' # str | Required input. ID of the parent pipeline.
pipeline_version_id = 'pipeline_version_id_example' # str | Required input. The ID of the pipeline version to be deleted.

    try:
        # Deletes a specific pipeline version by pipeline version ID and pipeline ID.
        api_response = api_instance.pipeline_service_delete_pipeline_version(pipeline_id, pipeline_version_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_delete_pipeline_version: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| Required input. ID of the parent pipeline. | 
 **pipeline_version_id** | **str**| Required input. The ID of the pipeline version to be deleted. | 

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

# **pipeline_service_get_pipeline**
> V2beta1Pipeline pipeline_service_get_pipeline(pipeline_id)

Finds a specific pipeline by ID.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    pipeline_id = 'pipeline_id_example' # str | Required input. The ID of the pipeline to be retrieved.

    try:
        # Finds a specific pipeline by ID.
        api_response = api_instance.pipeline_service_get_pipeline(pipeline_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| Required input. The ID of the pipeline to be retrieved. | 

### Return type

[**V2beta1Pipeline**](V2beta1Pipeline.md)

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

# **pipeline_service_get_pipeline_by_name**
> V2beta1Pipeline pipeline_service_get_pipeline_by_name(name, namespace=namespace)

Finds a specific pipeline by name and namespace.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    name = 'name_example' # str | Required input. Name of the pipeline to be retrieved.
namespace = 'namespace_example' # str | Optional input. Namespace of the pipeline.  It could be empty if default namespaces needs to be used or if multi-user  support is turned off. (optional)

    try:
        # Finds a specific pipeline by name and namespace.
        api_response = api_instance.pipeline_service_get_pipeline_by_name(name, namespace=namespace)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline_by_name: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Required input. Name of the pipeline to be retrieved. | 
 **namespace** | **str**| Optional input. Namespace of the pipeline.  It could be empty if default namespaces needs to be used or if multi-user  support is turned off. | [optional] 

### Return type

[**V2beta1Pipeline**](V2beta1Pipeline.md)

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

# **pipeline_service_get_pipeline_version**
> V2beta1PipelineVersion pipeline_service_get_pipeline_version(pipeline_id, pipeline_version_id)

Gets a pipeline version by pipeline version ID and pipeline ID.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    pipeline_id = 'pipeline_id_example' # str | Required input. ID of the parent pipeline.
pipeline_version_id = 'pipeline_version_id_example' # str | Required input. ID of the pipeline version to be retrieved.

    try:
        # Gets a pipeline version by pipeline version ID and pipeline ID.
        api_response = api_instance.pipeline_service_get_pipeline_version(pipeline_id, pipeline_version_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline_version: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| Required input. ID of the parent pipeline. | 
 **pipeline_version_id** | **str**| Required input. ID of the pipeline version to be retrieved. | 

### Return type

[**V2beta1PipelineVersion**](V2beta1PipelineVersion.md)

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

# **pipeline_service_list_pipeline_versions**
> V2beta1ListPipelineVersionsResponse pipeline_service_list_pipeline_versions(pipeline_id, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Lists all pipeline versions of a given pipeline ID.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    pipeline_id = 'pipeline_id_example' # str | Required input. ID of the parent pipeline.
page_token = 'page_token_example' # str | A page token to request the results page. (optional)
page_size = 56 # int | The number of pipeline versions to be listed per page. If there are more pipeline versions than this number, the response message will contain a valid value in the nextPageToken field. (optional)
sort_by = 'sort_by_example' # str | Sorting order in form of \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). (optional)

    try:
        # Lists all pipeline versions of a given pipeline ID.
        api_response = api_instance.pipeline_service_list_pipeline_versions(pipeline_id, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_list_pipeline_versions: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| Required input. ID of the parent pipeline. | 
 **page_token** | **str**| A page token to request the results page. | [optional] 
 **page_size** | **int**| The number of pipeline versions to be listed per page. If there are more pipeline versions than this number, the response message will contain a valid value in the nextPageToken field. | [optional] 
 **sort_by** | **str**| Sorting order in form of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot;. Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). | [optional] 

### Return type

[**V2beta1ListPipelineVersionsResponse**](V2beta1ListPipelineVersionsResponse.md)

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

# **pipeline_service_list_pipelines**
> V2beta1ListPipelinesResponse pipeline_service_list_pipelines(namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Finds all pipelines within a namespace.

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
    api_instance = kfp_server_api.PipelineServiceApi(api_client)
    namespace = 'namespace_example' # str | Optional input. Namespace for the pipelines. (optional)
page_token = 'page_token_example' # str | A page token to request the results page. (optional)
page_size = 56 # int | The number of pipelines to be listed per page. If there are more pipelines than this number, the response message will contain a valid value in the nextPageToken field. (optional)
sort_by = 'sort_by_example' # str | Sorting order in form of \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). (optional)

    try:
        # Finds all pipelines within a namespace.
        api_response = api_instance.pipeline_service_list_pipelines(namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_list_pipelines: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Optional input. Namespace for the pipelines. | [optional] 
 **page_token** | **str**| A page token to request the results page. | [optional] 
 **page_size** | **int**| The number of pipelines to be listed per page. If there are more pipelines than this number, the response message will contain a valid value in the nextPageToken field. | [optional] 
 **sort_by** | **str**| Sorting order in form of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot;. Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)). | [optional] 

### Return type

[**V2beta1ListPipelinesResponse**](V2beta1ListPipelinesResponse.md)

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

