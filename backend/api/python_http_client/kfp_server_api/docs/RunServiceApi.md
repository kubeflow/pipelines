# kfp_server_api.RunServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**archive_run**](RunServiceApi.md#archive_run) | **POST** /apis/v1beta1/runs/{id}:archive | Archive a run.
[**create_run**](RunServiceApi.md#create_run) | **POST** /apis/v1beta1/runs | Create a new run.
[**delete_run**](RunServiceApi.md#delete_run) | **DELETE** /apis/v1beta1/runs/{id} | Delete a run.
[**get_run**](RunServiceApi.md#get_run) | **GET** /apis/v1beta1/runs/{run_id} | Find a specific run by ID.
[**list_runs**](RunServiceApi.md#list_runs) | **GET** /apis/v1beta1/runs | Find all runs.
[**read_artifact**](RunServiceApi.md#read_artifact) | **GET** /apis/v1beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read | Find a run&#39;s artifact data.
[**report_run_metrics**](RunServiceApi.md#report_run_metrics) | **POST** /apis/v1beta1/runs/{run_id}:reportMetrics | ReportRunMetrics reports metrics of a run. Each metric is reported in its own transaction, so this API accepts partial failures. Metric can be uniquely identified by (run_id, node_id, name). Duplicate reporting will be ignored by the API. First reporting wins.
[**retry_run**](RunServiceApi.md#retry_run) | **POST** /apis/v1beta1/runs/{run_id}/retry | Re-initiate a failed or terminated run.
[**terminate_run**](RunServiceApi.md#terminate_run) | **POST** /apis/v1beta1/runs/{run_id}/terminate | Terminate an active run.
[**unarchive_run**](RunServiceApi.md#unarchive_run) | **POST** /apis/v1beta1/runs/{id}:unarchive | Restore an archived run.


# **archive_run**
> object archive_run(id)

Archive a run.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
id = 'id_example' # str | 

try:
    # Archive a run.
    api_response = api_instance.archive_run(id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->archive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**|  | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_run**
> ApiRunDetail create_run(body)

Create a new run.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
body = kfp_server_api.ApiRun() # ApiRun | 

try:
    # Create a new run.
    api_response = api_instance.create_run(body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->create_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**ApiRun**](ApiRun.md)|  | 

### Return type

[**ApiRunDetail**](ApiRunDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_run**
> object delete_run(id)

Delete a run.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
id = 'id_example' # str | 

try:
    # Delete a run.
    api_response = api_instance.delete_run(id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->delete_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**|  | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_run**
> ApiRunDetail get_run(run_id)

Find a specific run by ID.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
run_id = 'run_id_example' # str | 

try:
    # Find a specific run by ID.
    api_response = api_instance.get_run(run_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->get_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**|  | 

### Return type

[**ApiRunDetail**](ApiRunDetail.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_runs**
> ApiListRunsResponse list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id, filter=filter)

Find all runs.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
page_token = 'page_token_example' # str |  (optional)
page_size = 56 # int |  (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name des\" (Example, \"name asc\" or \"id des\"). Ascending by default. (optional)
resource_reference_key_type = 'UNKNOWN_RESOURCE_TYPE' # str | The type of the resource that referred to. (optional) (default to UNKNOWN_RESOURCE_TYPE)
resource_reference_key_id = 'resource_reference_key_id_example' # str | The ID of the resource that referred to. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/ blob/master/backend/api/filter.proto)). (optional)

try:
    # Find all runs.
    api_response = api_instance.list_runs(page_token=page_token, page_size=page_size, sort_by=sort_by, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id, filter=filter)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->list_runs: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**|  | [optional] 
 **page_size** | **int**|  | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name des\&quot; (Example, \&quot;name asc\&quot; or \&quot;id des\&quot;). Ascending by default. | [optional] 
 **resource_reference_key_type** | **str**| The type of the resource that referred to. | [optional] [default to UNKNOWN_RESOURCE_TYPE]
 **resource_reference_key_id** | **str**| The ID of the resource that referred to. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/ blob/master/backend/api/filter.proto)). | [optional] 

### Return type

[**ApiListRunsResponse**](ApiListRunsResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **read_artifact**
> ApiReadArtifactResponse read_artifact(run_id, node_id, artifact_name)

Find a run's artifact data.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
run_id = 'run_id_example' # str | The ID of the run.
node_id = 'node_id_example' # str | The ID of the running node.
artifact_name = 'artifact_name_example' # str | The name of the artifact.

try:
    # Find a run's artifact data.
    api_response = api_instance.read_artifact(run_id, node_id, artifact_name)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->read_artifact: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| The ID of the run. | 
 **node_id** | **str**| The ID of the running node. | 
 **artifact_name** | **str**| The name of the artifact. | 

### Return type

[**ApiReadArtifactResponse**](ApiReadArtifactResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **report_run_metrics**
> ApiReportRunMetricsResponse report_run_metrics(run_id, body)

ReportRunMetrics reports metrics of a run. Each metric is reported in its own transaction, so this API accepts partial failures. Metric can be uniquely identified by (run_id, node_id, name). Duplicate reporting will be ignored by the API. First reporting wins.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
run_id = 'run_id_example' # str | Required. The parent run ID of the metric.
body = kfp_server_api.ApiReportRunMetricsRequest() # ApiReportRunMetricsRequest | 

try:
    # ReportRunMetrics reports metrics of a run. Each metric is reported in its own transaction, so this API accepts partial failures. Metric can be uniquely identified by (run_id, node_id, name). Duplicate reporting will be ignored by the API. First reporting wins.
    api_response = api_instance.report_run_metrics(run_id, body)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->report_run_metrics: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**| Required. The parent run ID of the metric. | 
 **body** | [**ApiReportRunMetricsRequest**](ApiReportRunMetricsRequest.md)|  | 

### Return type

[**ApiReportRunMetricsResponse**](ApiReportRunMetricsResponse.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **retry_run**
> object retry_run(run_id)

Re-initiate a failed or terminated run.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
run_id = 'run_id_example' # str | 

try:
    # Re-initiate a failed or terminated run.
    api_response = api_instance.retry_run(run_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->retry_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**|  | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **terminate_run**
> object terminate_run(run_id)

Terminate an active run.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
run_id = 'run_id_example' # str | 

try:
    # Terminate an active run.
    api_response = api_instance.terminate_run(run_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->terminate_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **run_id** | **str**|  | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **unarchive_run**
> object unarchive_run(id)

Restore an archived run.

### Example
```python
from __future__ import print_function
import time
import kfp_server_api
from kfp_server_api.rest import ApiException
from pprint import pprint

# Configure API key authorization: Bearer
configuration = kfp_server_api.Configuration()
configuration.api_key['authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['authorization'] = 'Bearer'

# create an instance of the API class
api_instance = kfp_server_api.RunServiceApi(kfp_server_api.ApiClient(configuration))
id = 'id_example' # str | 

try:
    # Restore an archived run.
    api_response = api_instance.unarchive_run(id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling RunServiceApi->unarchive_run: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**|  | 

### Return type

**object**

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

