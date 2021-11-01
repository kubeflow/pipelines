# ApiPipelineSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pipeline_id** | **str** | Optional input field. The ID of the pipeline user uploaded before. | [optional] 
**pipeline_name** | **str** | Optional output field. The name of the pipeline. Not empty if the pipeline id is not empty. | [optional] 
**workflow_manifest** | **str** | Optional input field. The marshalled raw argo JSON workflow. This will be deprecated when pipeline_manifest is in use. | [optional] 
**pipeline_manifest** | **str** | Optional input field. The raw pipeline JSON spec. | [optional] 
**parameters** | [**list[ApiParameter]**](ApiParameter.md) |  | [optional] 
**runtime_config** | [**PipelineSpecRuntimeConfig**](PipelineSpecRuntimeConfig.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


