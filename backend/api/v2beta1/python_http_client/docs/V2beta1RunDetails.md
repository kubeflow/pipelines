# V2beta1RunDetails

Runtime details of a run.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pipeline_context_id** | **str** | Pipeline context ID of a run. | [optional] 
**pipeline_run_context_id** | **str** | Pipeline run context ID of a run. | [optional] 
**task_details** | [**list[V2beta1PipelineTaskDetail]**](V2beta1PipelineTaskDetail.md) | Deprecated: use top-level task APIs and Run.tasks instead. This legacy field remains in the schema for backward wire compatibility only. As part of the next major release, the backend intentionally stops populating this field and returns task data only through the top-level task APIs and Run.tasks. It will be removed in the next major API version. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


