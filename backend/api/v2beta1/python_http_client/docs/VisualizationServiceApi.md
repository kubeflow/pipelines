# kfp_server_api.VisualizationServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**visualization_service_create_visualization_v1**](VisualizationServiceApi.md#visualization_service_create_visualization_v1) | **POST** /apis/v2beta1/visualizations/{namespace} | 


# **visualization_service_create_visualization_v1**
> V2beta1Visualization visualization_service_create_visualization_v1(namespace, body)



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
    api_instance = kfp_server_api.VisualizationServiceApi(api_client)
    namespace = 'namespace_example' # str | 
body = kfp_server_api.V2beta1Visualization() # V2beta1Visualization | 

    try:
        api_response = api_instance.visualization_service_create_visualization_v1(namespace, body)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling VisualizationServiceApi->visualization_service_create_visualization_v1: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **body** | [**V2beta1Visualization**](V2beta1Visualization.md)|  | 

### Return type

[**V2beta1Visualization**](V2beta1Visualization.md)

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

