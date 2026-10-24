# PipelineTaskLogHistory

Log references from an execution archived before a manual run retry. References can outlive pods or archived objects and do not guarantee availability.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**before_retry_generation** | **str** | Together with task_id, identifies this snapshot. This is the manual retry generation that archived it, not an automatic pod retry attempt number. | [optional] 
**archived_at** | **datetime** |  | [optional] 
**start_time** | **datetime** |  | [optional] 
**end_time** | **datetime** |  | [optional] 
**state** | [**PipelineTaskTaskState**](PipelineTaskTaskState.md) |  | [optional] 
**pods** | [**list[PipelineTaskTaskPod]**](PipelineTaskTaskPod.md) |  | [optional] 
**log_artifacts** | [**list[InputOutputsIOArtifact]**](InputOutputsIOArtifact.md) | Log artifacts only; never includes task outputs used by downstream tasks. | [optional] 
**driver_logs_uri** | **str** | Legacy driver log reference, when not represented by a log artifact. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


