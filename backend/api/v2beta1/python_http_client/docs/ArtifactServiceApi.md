# kfp_server_api.ArtifactServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**artifact_service_get_artifact**](ArtifactServiceApi.md#artifact_service_get_artifact) | **GET** /apis/v2beta1/artifacts/{artifact_id} | Finds a specific Artifact by ID.
[**artifact_service_list_artifacts**](ArtifactServiceApi.md#artifact_service_list_artifacts) | **GET** /apis/v2beta1/artifacts | Finds all artifacts within the specified namespace. Namespace field is required. In multi-user mode, the caller is required to have RBAC verb \&quot;list\&quot; on the \&quot;artifacts\&quot; resource for the specified namespace.


# **artifact_service_get_artifact**
> V2beta1Artifact artifact_service_get_artifact(artifact_id, view=view)

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
view = 'ARTIFACT_VIEW_UNSPECIFIED' # str | Optional. Set to \"DOWNLOAD\" to included a signed URL with an expiry (default 15 seconds, unless configured other wise). This URL can be used to download the Artifact directly from the Artifact's storage provider. Set to \"BASIC\" to exclude the download_url from server responses, thus preventing the creation of any signed url. Defaults to BASIC.   - ARTIFACT_VIEW_UNSPECIFIED: Not specified, equivalent to BASIC.  - BASIC: Server responses excludes download_url  - DOWNLOAD: Server responses include download_url  - RENDER: Server response includes a signed URL, allowing in-browser rendering or preview of the artifact. (optional) (default to 'ARTIFACT_VIEW_UNSPECIFIED')

    try:
        # Finds a specific Artifact by ID.
        api_response = api_instance.artifact_service_get_artifact(artifact_id, view=view)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->artifact_service_get_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **artifact_id** | **str**| Required. The ID of the artifact to be retrieved. | 
 **view** | **str**| Optional. Set to \&quot;DOWNLOAD\&quot; to included a signed URL with an expiry (default 15 seconds, unless configured other wise). This URL can be used to download the Artifact directly from the Artifact&#39;s storage provider. Set to \&quot;BASIC\&quot; to exclude the download_url from server responses, thus preventing the creation of any signed url. Defaults to BASIC.   - ARTIFACT_VIEW_UNSPECIFIED: Not specified, equivalent to BASIC.  - BASIC: Server responses excludes download_url  - DOWNLOAD: Server responses include download_url  - RENDER: Server response includes a signed URL, allowing in-browser rendering or preview of the artifact. | [optional] [default to &#39;ARTIFACT_VIEW_UNSPECIFIED&#39;]

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

# **artifact_service_list_artifacts**
> V2beta1ListArtifactResponse artifact_service_list_artifacts(max_result_size=max_result_size, order_by_field=order_by_field, order_by=order_by, next_page_token=next_page_token, namespace=namespace)

Finds all artifacts within the specified namespace. Namespace field is required. In multi-user mode, the caller is required to have RBAC verb \"list\" on the \"artifacts\" resource for the specified namespace.

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
    max_result_size = 56 # int | Optional. Max number of resources to return in the result. A value of zero or less will result in the default (20). The API implementation also enforces an upper-bound of 100, and picks the minimum between this value and the one specified here. [default = 20] (optional)
order_by_field = 'FIELD_UNSPECIFIED' # str | Optional. Ordering field. [default = ID] (optional) (default to 'FIELD_UNSPECIFIED')
order_by = 'order_by_example' # str | Optional. Can be either \"asc\" (ascending) or \"desc\" (descending). [default = asc] (optional)
next_page_token = 'next_page_token_example' # str | Optional. The next_page_token value returned from a previous List request, if any. (optional)
namespace = 'namespace_example' # str | Required. Namespace of the Artifact's context. (optional)

    try:
        # Finds all artifacts within the specified namespace. Namespace field is required. In multi-user mode, the caller is required to have RBAC verb \"list\" on the \"artifacts\" resource for the specified namespace.
        api_response = api_instance.artifact_service_list_artifacts(max_result_size=max_result_size, order_by_field=order_by_field, order_by=order_by, next_page_token=next_page_token, namespace=namespace)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ArtifactServiceApi->artifact_service_list_artifacts: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **max_result_size** | **int**| Optional. Max number of resources to return in the result. A value of zero or less will result in the default (20). The API implementation also enforces an upper-bound of 100, and picks the minimum between this value and the one specified here. [default &#x3D; 20] | [optional] 
 **order_by_field** | **str**| Optional. Ordering field. [default &#x3D; ID] | [optional] [default to &#39;FIELD_UNSPECIFIED&#39;]
 **order_by** | **str**| Optional. Can be either \&quot;asc\&quot; (ascending) or \&quot;desc\&quot; (descending). [default &#x3D; asc] | [optional] 
 **next_page_token** | **str**| Optional. The next_page_token value returned from a previous List request, if any. | [optional] 
 **namespace** | **str**| Required. Namespace of the Artifact&#39;s context. | [optional] 

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

