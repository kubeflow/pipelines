# V2beta1Visualization

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type** | [**V2beta1VisualizationType**](V2beta1VisualizationType.md) |  | [optional] 
**source** | **str** | Path pattern of input data to be used during generation of visualizations. This is required when creating the pipeline through CreateVisualization API. | [optional] 
**arguments** | **str** | Variables to be used during generation of a visualization. This should be provided as a JSON string. This is required when creating the pipeline through CreateVisualization API. | [optional] 
**html** | **str** | Output. Generated visualization html. | [optional] 
**error** | **str** | In case any error happens when generating visualizations, only visualization ID and the error message are returned. Client has the flexibility of choosing how to handle the error. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


