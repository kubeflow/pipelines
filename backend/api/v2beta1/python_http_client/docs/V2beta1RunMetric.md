# V2beta1RunMetric

Metric assiciated with a run.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**display_name** | **str** | Required input. The user defined name of the metric. It must be 1-63 characters long and must conform to the following regular expression: &#x60;[a-z]([-a-z0-9]*[a-z0-9])?&#x60;. | [optional] 
**node_id** | **str** | Required input. The runtime node ID which reports the metric. The node ID  can be found in the RunDetail.workflow.Status. Metric with same  (node_id, name) are considerd as duplicate. Only the first reporting will  be recorded. Max length is 128. | [optional] 
**number_value** | **float** | The number value of the metric. | [optional] 
**format** | [**RunMetricFormat**](RunMetricFormat.md) |  | [optional] 
**error** | [**GooglerpcStatus**](GooglerpcStatus.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


