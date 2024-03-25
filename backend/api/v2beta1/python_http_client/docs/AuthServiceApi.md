# kfp_server_api.AuthServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**auth_service_authorize**](AuthServiceApi.md#auth_service_authorize) | **GET** /apis/v2beta1/auth | 


# **auth_service_authorize**
> object auth_service_authorize(namespace=namespace, resources=resources, verb=verb)



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
    api_instance = kfp_server_api.AuthServiceApi(api_client)
    namespace = 'namespace_example' # str |  (optional)
resources = 'UNASSIGNED_RESOURCES' # str |  (optional) (default to 'UNASSIGNED_RESOURCES')
verb = 'UNASSIGNED_VERB' # str |  (optional) (default to 'UNASSIGNED_VERB')

    try:
        api_response = api_instance.auth_service_authorize(namespace=namespace, resources=resources, verb=verb)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling AuthServiceApi->auth_service_authorize: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | [optional] 
 **resources** | **str**|  | [optional] [default to &#39;UNASSIGNED_RESOURCES&#39;]
 **verb** | **str**|  | [optional] [default to &#39;UNASSIGNED_VERB&#39;]

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

