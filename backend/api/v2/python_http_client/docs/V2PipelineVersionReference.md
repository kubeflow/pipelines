# V2PipelineVersionReference

Reference to an existing pipeline version.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pipeline_id** | **str** | Input. Unique ID of the parent pipeline. Required unless pipeline_name is set. | [optional] 
**pipeline_version_id** | **str** | Input. Optional. Unique ID of an existing pipeline version. If unset, the latest pipeline version is used. | [optional] 
**pipeline_name** | **str** | Input. Optional. Name of the parent pipeline in the run&#39;s namespace, used instead of pipeline_id. Not returned in responses. | [optional] 
**pipeline_version_name** | **str** | Input. Optional. Name of a version of the pipeline named by pipeline_name, used instead of pipeline_version_id. Not returned in responses. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


