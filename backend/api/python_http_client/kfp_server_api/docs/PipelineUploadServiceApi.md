# kfp_server_api.PipelineUploadServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**upload_pipeline**](PipelineUploadServiceApi.md#upload_pipeline) | **POST** /apis/v1beta1/pipelines/upload | 
[**upload_pipeline_version**](PipelineUploadServiceApi.md#upload_pipeline_version) | **POST** /apis/v1beta1/pipelines/upload_version | 


# **upload_pipeline**
> ApiPipeline upload_pipeline(uploadfile, name=name, description=description)



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
    api_instance = kfp_server_api.PipelineUploadServiceApi(api_client)
    uploadfile = '/path/to/file' # file | The pipeline to upload. Maximum size of 32MB is supported.
name = 'name_example' # str |  (optional)
description = 'description_example' # str |  (optional)

    try:
        api_response = api_instance.upload_pipeline(uploadfile, name=name, description=description)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineUploadServiceApi->upload_pipeline: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uploadfile** | **file**| The pipeline to upload. Maximum size of 32MB is supported. | 
 **name** | **str**|  | [optional] 
 **description** | **str**|  | [optional] 

### Return type

[**ApiPipeline**](ApiPipeline.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** |  |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **upload_pipeline_version**
> ApiPipelineVersion upload_pipeline_version(uploadfile, name=name, pipelineid=pipelineid)



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
    api_instance = kfp_server_api.PipelineUploadServiceApi(api_client)
    uploadfile = '/path/to/file' # file | The pipeline to upload. Maximum size of 32MB is supported.
name = 'name_example' # str |  (optional)
pipelineid = 'pipelineid_example' # str |  (optional)

    try:
        api_response = api_instance.upload_pipeline_version(uploadfile, name=name, pipelineid=pipelineid)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling PipelineUploadServiceApi->upload_pipeline_version: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **uploadfile** | **file**| The pipeline to upload. Maximum size of 32MB is supported. | 
 **name** | **str**|  | [optional] 
 **pipelineid** | **str**|  | [optional] 

### Return type

[**ApiPipelineVersion**](ApiPipelineVersion.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: multipart/form-data
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** |  |  -  |
**0** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

