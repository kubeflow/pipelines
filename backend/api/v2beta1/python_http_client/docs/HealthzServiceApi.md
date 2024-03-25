# kfp_server_api.HealthzServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**healthz_service_get_healthz**](HealthzServiceApi.md#healthz_service_get_healthz) | **GET** /apis/v2beta1/healthz | Get healthz data.


# **healthz_service_get_healthz**
> V2beta1GetHealthzResponse healthz_service_get_healthz()

Get healthz data.

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
    api_instance = kfp_server_api.HealthzServiceApi(api_client)
    
    try:
        # Get healthz data.
        api_response = api_instance.healthz_service_get_healthz()
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling HealthzServiceApi->healthz_service_get_healthz: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**V2beta1GetHealthzResponse**](V2beta1GetHealthzResponse.md)

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

