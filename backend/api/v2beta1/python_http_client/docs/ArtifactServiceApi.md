# kfp_server_api.ArtifactServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**batch_create_artifact_tasks**](ArtifactServiceApi.md#batch_create_artifact_tasks) | **POST** /apis/v2beta1/artifact_tasks:batchCreate | Creates multiple artifact-task relationships in bulk.
[**batch_create_artifacts**](ArtifactServiceApi.md#batch_create_artifacts) | **POST** /apis/v2beta1/artifacts:batchCreate | Creates multiple artifacts in bulk.
[**create_artifact**](ArtifactServiceApi.md#create_artifact) | **POST** /apis/v2beta1/artifacts | Creates a new artifact.
[**create_artifact_task**](ArtifactServiceApi.md#create_artifact_task) | **POST** /apis/v2beta1/artifact_tasks | Creates an artifact-task relationship.
[**get_artifact**](ArtifactServiceApi.md#get_artifact) | **GET** /apis/v2beta1/artifacts/{artifact_id} | Finds a specific Artifact by ID.
[**list_artifact_tasks**](ArtifactServiceApi.md#list_artifact_tasks) | **GET** /apis/v2beta1/artifact_tasks | Lists artifact-task relationships.
[**list_artifacts**](ArtifactServiceApi.md#list_artifacts) | **GET** /apis/v2beta1/artifacts | Finds all artifacts within the specified namespace.


# **batch_create_artifact_tasks**
> V2beta1CreateArtifactTasksBulkResponse batch_create_artifact_tasks(body)

Creates multiple artifact-task relationships in bulk.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    body = kfp_server_api.V2beta1CreateArtifactTasksBulkRequest() # V2beta1CreateArtifactTasksBulkRequest | 

    try:
        # Creates multiple artifact-task relationships in bulk.
        api_response = api_instance.batch_create_artifact_tasks(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->batch_create_artifact_tasks: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1CreateArtifactTasksBulkRequest**](V2beta1CreateArtifactTasksBulkRequest.md)|  | 

### Return type

[**V2beta1CreateArtifactTasksBulkResponse**](V2beta1CreateArtifactTasksBulkResponse.md)

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

# **batch_create_artifacts**
> V2beta1CreateArtifactsBulkResponse batch_create_artifacts(body)

Creates multiple artifacts in bulk.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    body = kfp_server_api.V2beta1CreateArtifactsBulkRequest() # V2beta1CreateArtifactsBulkRequest | 

    try:
        # Creates multiple artifacts in bulk.
        api_response = api_instance.batch_create_artifacts(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->batch_create_artifacts: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1CreateArtifactsBulkRequest**](V2beta1CreateArtifactsBulkRequest.md)|  | 

### Return type

[**V2beta1CreateArtifactsBulkResponse**](V2beta1CreateArtifactsBulkResponse.md)

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

# **create_artifact**
> V2beta1Artifact create_artifact(body)

Creates a new artifact.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    body = kfp_server_api.V2beta1CreateArtifactRequest() # V2beta1CreateArtifactRequest | 

    try:
        # Creates a new artifact.
        api_response = api_instance.create_artifact(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->create_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1CreateArtifactRequest**](V2beta1CreateArtifactRequest.md)|  | 

### Return type

[**V2beta1Artifact**](V2beta1Artifact.md)

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

# **create_artifact_task**
> V2beta1ArtifactTask create_artifact_task(body)

Creates an artifact-task relationship.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    body = kfp_server_api.V2beta1CreateArtifactTaskRequest() # V2beta1CreateArtifactTaskRequest | 

    try:
        # Creates an artifact-task relationship.
        api_response = api_instance.create_artifact_task(body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->create_artifact_task: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**V2beta1CreateArtifactTaskRequest**](V2beta1CreateArtifactTaskRequest.md)|  | 

### Return type

[**V2beta1ArtifactTask**](V2beta1ArtifactTask.md)

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

# **get_artifact**
> V2beta1Artifact get_artifact(artifact_id)

Finds a specific Artifact by ID.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    artifact_id = 'artifact_id_example' # str | Required. The ID of the artifact to be retrieved.

    try:
        # Finds a specific Artifact by ID.
        api_response = api_instance.get_artifact(artifact_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->get_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **artifact_id** | **str**| Required. The ID of the artifact to be retrieved. | 

### Return type

[**V2beta1Artifact**](V2beta1Artifact.md)

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

# **list_artifact_tasks**
> V2beta1ListArtifactTasksResponse list_artifact_tasks(task_ids=task_ids, run_ids=run_ids, artifact_ids=artifact_ids, type=type, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Lists artifact-task relationships.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    task_ids = ['task_ids_example'] # list[str] | Optional, filter artifact task by a set of task_ids (optional)
run_ids = ['run_ids_example'] # list[str] | Optional, filter artifact task by a set of run_ids (optional)
artifact_ids = ['artifact_ids_example'] # list[str] | Optional, filter artifact task by a set of artifact_ids (optional)
type = 'UNSPECIFIED' # str | Optional. Only list artifact tasks that have artifacts of this type.   - UNSPECIFIED: For validation  - COMPONENT_DEFAULT_INPUT: This is used for inputs that are provided via default parameters in the component input definitions  - TASK_OUTPUT_INPUT: This is used for inputs that are provided via upstream tasks. In the sdk this appears as: TaskInputsSpec.kind.task_output_parameter & TaskInputsSpec.kind.task_output_artifact  - COMPONENT_INPUT: Used for inputs that are passed from parent tasks.  - RUNTIME_VALUE_INPUT: Hardcoded values passed as arguments to the task.  - COLLECTED_INPUTS: Used for dsl.Collected Usage of this type indicates that all Artifacts within the IOArtifact.artifacts are inputs collected from sub tasks with ITERATOR_OUTPUT outputs.  - ITERATOR_INPUT: In a for loop task, introduced via ParallelFor, this type is used to indicate whether this resolved input belongs to a parameterIterator or artifactIterator. In such a case the \"artifacts\" field for IOArtifact.artifacts is the list of resolved items for this parallelFor.  - ITERATOR_INPUT_RAW: Hardcoded iterator parameters. Raw Iterator inputs have no producer  - ITERATOR_OUTPUT: When an output is produced by a Runtime Iteration Task This value is use to differentiate between standard inputs  - OUTPUT: All other output types fall under this type.  - ONE_OF_OUTPUT: An output of a Conditions branch. (optional) (default to 'UNSPECIFIED')
page_token = 'page_token_example' # str |  (optional)
page_size = 56 # int |  (optional)
sort_by = 'sort_by_example' # str |  (optional)
filter = 'filter_example' # str |  (optional)

    try:
        # Lists artifact-task relationships.
        api_response = api_instance.list_artifact_tasks(task_ids=task_ids, run_ids=run_ids, artifact_ids=artifact_ids, type=type, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->list_artifact_tasks: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **task_ids** | [**list[str]**](str.md)| Optional, filter artifact task by a set of task_ids | [optional] 
 **run_ids** | [**list[str]**](str.md)| Optional, filter artifact task by a set of run_ids | [optional] 
 **artifact_ids** | [**list[str]**](str.md)| Optional, filter artifact task by a set of artifact_ids | [optional] 
 **type** | **str**| Optional. Only list artifact tasks that have artifacts of this type.   - UNSPECIFIED: For validation  - COMPONENT_DEFAULT_INPUT: This is used for inputs that are provided via default parameters in the component input definitions  - TASK_OUTPUT_INPUT: This is used for inputs that are provided via upstream tasks. In the sdk this appears as: TaskInputsSpec.kind.task_output_parameter &amp; TaskInputsSpec.kind.task_output_artifact  - COMPONENT_INPUT: Used for inputs that are passed from parent tasks.  - RUNTIME_VALUE_INPUT: Hardcoded values passed as arguments to the task.  - COLLECTED_INPUTS: Used for dsl.Collected Usage of this type indicates that all Artifacts within the IOArtifact.artifacts are inputs collected from sub tasks with ITERATOR_OUTPUT outputs.  - ITERATOR_INPUT: In a for loop task, introduced via ParallelFor, this type is used to indicate whether this resolved input belongs to a parameterIterator or artifactIterator. In such a case the \&quot;artifacts\&quot; field for IOArtifact.artifacts is the list of resolved items for this parallelFor.  - ITERATOR_INPUT_RAW: Hardcoded iterator parameters. Raw Iterator inputs have no producer  - ITERATOR_OUTPUT: When an output is produced by a Runtime Iteration Task This value is use to differentiate between standard inputs  - OUTPUT: All other output types fall under this type.  - ONE_OF_OUTPUT: An output of a Conditions branch. | [optional] [default to &#39;UNSPECIFIED&#39;]
 **page_token** | **str**|  | [optional] 
 **page_size** | **int**|  | [optional] 
 **sort_by** | **str**|  | [optional] 
 **filter** | **str**|  | [optional] 

### Return type

[**V2beta1ListArtifactTasksResponse**](V2beta1ListArtifactTasksResponse.md)

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

# **list_artifacts**
> V2beta1ListArtifactResponse list_artifacts(namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)

Finds all artifacts within the specified namespace.

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
    api_instance = kfp_server_api.ArtifactServiceApi(api_client)
    namespace = 'namespace_example' # str | Optional input. Namespace for the artifacts. (optional)
page_token = 'page_token_example' # str | A page token to request the results page. (optional)
page_size = 56 # int | The number of artifacts to be listed per page. If there are more artifacts than this number, the response message will contain a valid value in the nextPageToken field. (optional)
sort_by = 'sort_by_example' # str | Sorting order in form of \"field_name\", \"field_name asc\" or \"field_name desc\". Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/artifacts/blob/master/backend/api/filter.proto)). (optional)

    try:
        # Finds all artifacts within the specified namespace.
        api_response = api_instance.list_artifacts(namespace=namespace, page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->list_artifacts: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Optional input. Namespace for the artifacts. | [optional] 
 **page_token** | **str**| A page token to request the results page. | [optional] 
 **page_size** | **int**| The number of artifacts to be listed per page. If there are more artifacts than this number, the response message will contain a valid value in the nextPageToken field. | [optional] 
 **sort_by** | **str**| Sorting order in form of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot;. Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized filter protocol buffer (see [filter.proto](https://github.com/kubeflow/artifacts/blob/master/backend/api/filter.proto)). | [optional] 

### Return type

[**V2beta1ListArtifactResponse**](V2beta1ListArtifactResponse.md)

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

