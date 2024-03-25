# kfp_server_api.PipelineServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**pipeline_service_create_pipeline_v1**](PipelineServiceApi.md#pipeline_service_create_pipeline_v1) | **POST** /apis/v1beta1/pipelines | Creates a pipeline.
[**pipeline_service_create_pipeline_version_v1**](PipelineServiceApi.md#pipeline_service_create_pipeline_version_v1) | **POST** /apis/v1beta1/pipeline_versions | Adds a pipeline version to the specified pipeline.
[**pipeline_service_delete_pipeline_v1**](PipelineServiceApi.md#pipeline_service_delete_pipeline_v1) | **DELETE** /apis/v1beta1/pipelines/{id} | Deletes a pipeline and its pipeline versions.
[**pipeline_service_delete_pipeline_version_v1**](PipelineServiceApi.md#pipeline_service_delete_pipeline_version_v1) | **DELETE** /apis/v1beta1/pipeline_versions/{version_id} | Deletes a pipeline version by pipeline version ID. If the deleted pipeline version is the default pipeline version, the pipeline&#39;s default version changes to the pipeline&#39;s most recent pipeline version. If there are no remaining pipeline versions, the pipeline will have no default version. Examines the run_service_api.ipynb notebook to learn more about creating a run using a pipeline version (https://github.com/kubeflow/pipelines/blob/master/tools/benchmarks/run_service_api.ipynb).
[**pipeline_service_get_pipeline_by_name_v1**](PipelineServiceApi.md#pipeline_service_get_pipeline_by_name_v1) | **GET** /apis/v1beta1/namespaces/{namespace}/pipelines/{name} | Finds a pipeline by Name (and namespace)
[**pipeline_service_get_pipeline_v1**](PipelineServiceApi.md#pipeline_service_get_pipeline_v1) | **GET** /apis/v1beta1/pipelines/{id} | Finds a specific pipeline by ID.
[**pipeline_service_get_pipeline_version_template**](PipelineServiceApi.md#pipeline_service_get_pipeline_version_template) | **GET** /apis/v1beta1/pipeline_versions/{version_id}/templates | Returns a YAML template that contains the specified pipeline version&#39;s description, parameters and metadata.
[**pipeline_service_get_pipeline_version_v1**](PipelineServiceApi.md#pipeline_service_get_pipeline_version_v1) | **GET** /apis/v1beta1/pipeline_versions/{version_id} | Gets a pipeline version by pipeline version ID.
[**pipeline_service_get_template**](PipelineServiceApi.md#pipeline_service_get_template) | **GET** /apis/v1beta1/pipelines/{id}/templates | Returns a single YAML template that contains the description, parameters, and metadata associated with the pipeline provided.
[**pipeline_service_list_pipeline_versions_v1**](PipelineServiceApi.md#pipeline_service_list_pipeline_versions_v1) | **GET** /apis/v1beta1/pipeline_versions | Lists all pipeline versions of a given pipeline.
[**pipeline_service_list_pipelines_v1**](PipelineServiceApi.md#pipeline_service_list_pipelines_v1) | **GET** /apis/v1beta1/pipelines | Finds all pipelines.
[**pipeline_service_update_pipeline_default_version_v1**](PipelineServiceApi.md#pipeline_service_update_pipeline_default_version_v1) | **POST** /apis/v1beta1/pipelines/{pipeline_id}/default_version/{version_id} | Update the default pipeline version of a specific pipeline.


# **pipeline_service_create_pipeline_v1**
> ApiPipeline pipeline_service_create_pipeline_v1(body)

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
    body = kfp_server_api.ApiPipeline() # ApiPipeline | 

    try:
        # Creates a pipeline.
        api_response = api_instance.pipeline_service_create_pipeline_v1(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_create_pipeline_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**ApiPipeline**](ApiPipeline.md)|  | 

### Return type

[**ApiPipeline**](ApiPipeline.md)

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

# **pipeline_service_create_pipeline_version_v1**
> ApiPipelineVersion pipeline_service_create_pipeline_version_v1(body)

Adds a pipeline version to the specified pipeline.

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
    body = kfp_server_api.ApiPipelineVersion() # ApiPipelineVersion | ResourceReference inside PipelineVersion specifies the pipeline that this version belongs to.

    try:
        # Adds a pipeline version to the specified pipeline.
        api_response = api_instance.pipeline_service_create_pipeline_version_v1(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_create_pipeline_version_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**ApiPipelineVersion**](ApiPipelineVersion.md)| ResourceReference inside PipelineVersion specifies the pipeline that this version belongs to. | 

### Return type

[**ApiPipelineVersion**](ApiPipelineVersion.md)

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

# **pipeline_service_delete_pipeline_v1**
> object pipeline_service_delete_pipeline_v1(id)

Deletes a pipeline and its pipeline versions.

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
    id = 'id_example' # str | The ID of the pipeline to be deleted.

    try:
        # Deletes a pipeline and its pipeline versions.
        api_response = api_instance.pipeline_service_delete_pipeline_v1(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_delete_pipeline_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the pipeline to be deleted. | 

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

# **pipeline_service_delete_pipeline_version_v1**
> object pipeline_service_delete_pipeline_version_v1(version_id)

Deletes a pipeline version by pipeline version ID. If the deleted pipeline version is the default pipeline version, the pipeline's default version changes to the pipeline's most recent pipeline version. If there are no remaining pipeline versions, the pipeline will have no default version. Examines the run_service_api.ipynb notebook to learn more about creating a run using a pipeline version (https://github.com/kubeflow/pipelines/blob/master/tools/benchmarks/run_service_api.ipynb).

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
    version_id = 'version_id_example' # str | The ID of the pipeline version to be deleted.

    try:
        # Deletes a pipeline version by pipeline version ID. If the deleted pipeline version is the default pipeline version, the pipeline's default version changes to the pipeline's most recent pipeline version. If there are no remaining pipeline versions, the pipeline will have no default version. Examines the run_service_api.ipynb notebook to learn more about creating a run using a pipeline version (https://github.com/kubeflow/pipelines/blob/master/tools/benchmarks/run_service_api.ipynb).
        api_response = api_instance.pipeline_service_delete_pipeline_version_v1(version_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_delete_pipeline_version_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **version_id** | **str**| The ID of the pipeline version to be deleted. | 

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

# **pipeline_service_get_pipeline_by_name_v1**
> ApiPipeline pipeline_service_get_pipeline_by_name_v1(namespace, name)

Finds a pipeline by Name (and namespace)

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
    namespace = 'namespace_example' # str | The Namespace the pipeline belongs to. In the case of shared pipelines and KFPipeline standalone installation, the pipeline name is the only needed field for unique resource lookup (namespace is not required). In those case, please provide hyphen (dash character, \"-\").
name = 'name_example' # str | The Name of the pipeline to be retrieved.

    try:
        # Finds a pipeline by Name (and namespace)
        api_response = api_instance.pipeline_service_get_pipeline_by_name_v1(namespace, name)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline_by_name_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| The Namespace the pipeline belongs to. In the case of shared pipelines and KFPipeline standalone installation, the pipeline name is the only needed field for unique resource lookup (namespace is not required). In those case, please provide hyphen (dash character, \&quot;-\&quot;). | 
 **name** | **str**| The Name of the pipeline to be retrieved. | 

### Return type

[**ApiPipeline**](ApiPipeline.md)

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

# **pipeline_service_get_pipeline_v1**
> ApiPipeline pipeline_service_get_pipeline_v1(id)

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
    id = 'id_example' # str | The ID of the pipeline to be retrieved.

    try:
        # Finds a specific pipeline by ID.
        api_response = api_instance.pipeline_service_get_pipeline_v1(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the pipeline to be retrieved. | 

### Return type

[**ApiPipeline**](ApiPipeline.md)

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

# **pipeline_service_get_pipeline_version_template**
> ApiGetTemplateResponse pipeline_service_get_pipeline_version_template(version_id)

Returns a YAML template that contains the specified pipeline version's description, parameters and metadata.

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
    version_id = 'version_id_example' # str | The ID of the pipeline version whose template is to be retrieved.

    try:
        # Returns a YAML template that contains the specified pipeline version's description, parameters and metadata.
        api_response = api_instance.pipeline_service_get_pipeline_version_template(version_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline_version_template: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **version_id** | **str**| The ID of the pipeline version whose template is to be retrieved. | 

### Return type

[**ApiGetTemplateResponse**](ApiGetTemplateResponse.md)

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

# **pipeline_service_get_pipeline_version_v1**
> ApiPipelineVersion pipeline_service_get_pipeline_version_v1(version_id)

Gets a pipeline version by pipeline version ID.

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
    version_id = 'version_id_example' # str | The ID of the pipeline version to be retrieved.

    try:
        # Gets a pipeline version by pipeline version ID.
        api_response = api_instance.pipeline_service_get_pipeline_version_v1(version_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_pipeline_version_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **version_id** | **str**| The ID of the pipeline version to be retrieved. | 

### Return type

[**ApiPipelineVersion**](ApiPipelineVersion.md)

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

# **pipeline_service_get_template**
> ApiGetTemplateResponse pipeline_service_get_template(id)

Returns a single YAML template that contains the description, parameters, and metadata associated with the pipeline provided.

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
    id = 'id_example' # str | The ID of the pipeline whose template is to be retrieved.

    try:
        # Returns a single YAML template that contains the description, parameters, and metadata associated with the pipeline provided.
        api_response = api_instance.pipeline_service_get_template(id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_get_template: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| The ID of the pipeline whose template is to be retrieved. | 

### Return type

[**ApiGetTemplateResponse**](ApiGetTemplateResponse.md)

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

# **pipeline_service_list_pipeline_versions_v1**
> ApiListPipelineVersionsResponse pipeline_service_list_pipeline_versions_v1(resource_key_type=resource_key_type, resource_key_id=resource_key_id, page_size=page_size, page_token=page_token, sort_by=sort_by, filter=filter)

Lists all pipeline versions of a given pipeline.

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
    resource_key_type = 'UNKNOWN_RESOURCE_TYPE' # str | The type of the resource that referred to. (optional) (default to 'UNKNOWN_RESOURCE_TYPE')
resource_key_id = 'resource_key_id_example' # str | The ID of the resource that referred to. (optional)
page_size = 56 # int | The number of pipeline versions to be listed per page. If there are more pipeline versions than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListPipelineVersions call or can be omitted when fetching the first page. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" Ascending by default. (optional)
filter = 'filter_example' # str | A base-64 encoded, JSON-serialized Filter protocol buffer (see filter.proto). (optional)

    try:
        # Lists all pipeline versions of a given pipeline.
        api_response = api_instance.pipeline_service_list_pipeline_versions_v1(resource_key_type=resource_key_type, resource_key_id=resource_key_id, page_size=page_size, page_token=page_token, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_list_pipeline_versions_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **resource_key_type** | **str**| The type of the resource that referred to. | [optional] [default to &#39;UNKNOWN_RESOURCE_TYPE&#39;]
 **resource_key_id** | **str**| The ID of the resource that referred to. | [optional] 
 **page_size** | **int**| The number of pipeline versions to be listed per page. If there are more pipeline versions than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **page_token** | **str**| A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListPipelineVersions call or can be omitted when fetching the first page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default. | [optional] 
 **filter** | **str**| A base-64 encoded, JSON-serialized Filter protocol buffer (see filter.proto). | [optional] 

### Return type

[**ApiListPipelineVersionsResponse**](ApiListPipelineVersionsResponse.md)

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

# **pipeline_service_list_pipelines_v1**
> ApiListPipelinesResponse pipeline_service_list_pipelines_v1(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)

Finds all pipelines.

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
    page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListPipelines call. (optional)
page_size = 56 # int | The number of pipelines to be listed per page. If there are more pipelines than this number, the response message will contain a valid value in the nextPageToken field. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)). (optional)
resource_reference_key_type = 'UNKNOWN_RESOURCE_TYPE' # str | The type of the resource that referred to. (optional) (default to 'UNKNOWN_RESOURCE_TYPE')
resource_reference_key_id = 'resource_reference_key_id_example' # str | The ID of the resource that referred to. (optional)

    try:
        # Finds all pipelines.
        api_response = api_instance.pipeline_service_list_pipelines_v1(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_list_pipelines_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**| A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListPipelines call. | [optional] 
 **page_size** | **int**| The number of pipelines to be listed per page. If there are more pipelines than this number, the response message will contain a valid value in the nextPageToken field. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)). | [optional] 
 **resource_reference_key_type** | **str**| The type of the resource that referred to. | [optional] [default to &#39;UNKNOWN_RESOURCE_TYPE&#39;]
 **resource_reference_key_id** | **str**| The ID of the resource that referred to. | [optional] 

### Return type

[**ApiListPipelinesResponse**](ApiListPipelinesResponse.md)

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

# **pipeline_service_update_pipeline_default_version_v1**
> object pipeline_service_update_pipeline_default_version_v1(pipeline_id, version_id)

Update the default pipeline version of a specific pipeline.

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
    pipeline_id = 'pipeline_id_example' # str | The ID of the pipeline to be updated.
version_id = 'version_id_example' # str | The ID of the default version.

    try:
        # Update the default pipeline version of a specific pipeline.
        api_response = api_instance.pipeline_service_update_pipeline_default_version_v1(pipeline_id, version_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineServiceApi->pipeline_service_update_pipeline_default_version_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **pipeline_id** | **str**| The ID of the pipeline to be updated. | 
 **version_id** | **str**| The ID of the default version. | 

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

