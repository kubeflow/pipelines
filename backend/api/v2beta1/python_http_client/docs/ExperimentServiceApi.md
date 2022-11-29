# kfp_server_api.ExperimentServiceApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**archive_experiment**](ExperimentServiceApi.md#archive_experiment) | **POST** /apis/v2beta1/experiments/{experiment_id}:archive | Archives an experiment and the experiment&#39;s runs and recurring runs.
[**create_experiment**](ExperimentServiceApi.md#create_experiment) | **POST** /apis/v2beta1/experiments | Creates a new experiment.
[**delete_experiment**](ExperimentServiceApi.md#delete_experiment) | **DELETE** /apis/v2beta1/experiments/{experiment_id} | Deletes an experiment without deleting the experiment&#39;s runs and recurring  runs. To avoid unexpected behaviors, delete an experiment&#39;s runs and recurring  runs before deleting the experiment.
[**get_experiment**](ExperimentServiceApi.md#get_experiment) | **GET** /apis/v2beta1/experiments/{experiment_id} | Finds a specific experiment by ID.
[**list_experiments**](ExperimentServiceApi.md#list_experiments) | **GET** /apis/v2beta1/experiments | Finds all experiments. Supports pagination, and sorting on certain fields.
[**unarchive_experiment**](ExperimentServiceApi.md#unarchive_experiment) | **POST** /apis/v2beta1/experiments/{experiment_id}:unarchive | Restores an archived experiment. The experiment&#39;s archived runs and recurring runs will stay archived.


# **archive_experiment**
> object archive_experiment(experiment_id)

Archives an experiment and the experiment's runs and recurring runs.

### Example

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


# Enter a context with an instance of the API client
with kfp_server_api.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the experiment to be archived.

    try:
        # Archives an experiment and the experiment's runs and recurring runs.
        api_response = api_instance.archive_experiment(experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->archive_experiment: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the experiment to be archived. | 

### Return type

**object**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_experiment**
> V2beta1Experiment create_experiment(body)

Creates a new experiment.

### Example

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


# Enter a context with an instance of the API client
with kfp_server_api.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    body = kfp_server_api.V2beta1Experiment() # V2beta1Experiment | The experiment to be created.

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
 **body** | [**V2beta1Experiment**](V2beta1Experiment.md)| The experiment to be created. | 

### Return type

[**V2beta1Experiment**](V2beta1Experiment.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_experiment**
> object delete_experiment(experiment_id)

Deletes an experiment without deleting the experiment's runs and recurring  runs. To avoid unexpected behaviors, delete an experiment's runs and recurring  runs before deleting the experiment.

### Example

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


# Enter a context with an instance of the API client
with kfp_server_api.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the experiment to be deleted.

    try:
        # Deletes an experiment without deleting the experiment's runs and recurring  runs. To avoid unexpected behaviors, delete an experiment's runs and recurring  runs before deleting the experiment.
        api_response = api_instance.delete_experiment(experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->delete_experiment: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the experiment to be deleted. | 

### Return type

**object**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_experiment**
> V2beta1Experiment get_experiment(experiment_id)

Finds a specific experiment by ID.

### Example

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


# Enter a context with an instance of the API client
with kfp_server_api.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the experiment to be retrieved.

    try:
        # Finds a specific experiment by ID.
        api_response = api_instance.get_experiment(experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->get_experiment: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the experiment to be retrieved. | 

### Return type

[**V2beta1Experiment**](V2beta1Experiment.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_experiments**
> V2beta1ListExperimentsResponse list_experiments(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, namespace=namespace)

Finds all experiments. Supports pagination, and sorting on certain fields.

### Example

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


# Enter a context with an instance of the API client
with kfp_server_api.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    page_token = 'page_token_example' # str | A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiments call or can be omitted when fetching the first page. (optional)
page_size = 56 # int | The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page. (optional)
sort_by = 'sort_by_example' # str | Can be format of \"field_name\", \"field_name asc\" or \"field_name desc\" Ascending by default. (optional)
filter = 'filter_example' # str | A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v2beta1/api/filter.proto)). (optional)
namespace = 'namespace_example' # str | Which namespace to filter the experiments on. (optional)

    try:
        # Finds all experiments. Supports pagination, and sorting on certain fields.
        api_response = api_instance.list_experiments(page_token=page_token, page_size=page_size, sort_by=sort_by, filter=filter, namespace=namespace)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->list_experiments: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page_token** | **str**| A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiments call or can be omitted when fetching the first page. | [optional] 
 **page_size** | **int**| The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page. | [optional] 
 **sort_by** | **str**| Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default. | [optional] 
 **filter** | **str**| A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v2beta1/api/filter.proto)). | [optional] 
 **namespace** | **str**| Which namespace to filter the experiments on. | [optional] 

### Return type

[**V2beta1ListExperimentsResponse**](V2beta1ListExperimentsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **unarchive_experiment**
> object unarchive_experiment(experiment_id)

Restores an archived experiment. The experiment's archived runs and recurring runs will stay archived.

### Example

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


# Enter a context with an instance of the API client
with kfp_server_api.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = kfp_server_api.ExperimentServiceApi(api_client)
    experiment_id = 'experiment_id_example' # str | The ID of the experiment to be restored.

    try:
        # Restores an archived experiment. The experiment's archived runs and recurring runs will stay archived.
        api_response = api_instance.unarchive_experiment(experiment_id)
        pprint(api_response)
    except ApiException as e:
        print("Exception when calling ExperimentServiceApi->unarchive_experiment: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **experiment_id** | **str**| The ID of the experiment to be restored. | 

### Return type

**object**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

