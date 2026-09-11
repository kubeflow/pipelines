# V2beta1PipelineTask

Runtime information of a task execution used by task APIs and Run.tasks.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 
**display_name** | **str** | User specified name of a task that is defined in [Pipeline.spec][]. | [optional] 
**task_id** | **str** | System-generated ID of a task. | [optional] 
**run_id** | **str** | ID of the parent run. | [optional] 
**pods** | [**list[PipelineTaskTaskPod]**](PipelineTaskTaskPod.md) |  | [optional] 
**cache_fingerprint** | **str** |  | [optional] 
**create_time** | **datetime** | Creation time of a task. | [optional] 
**end_time** | **datetime** | Completion time of a task. | [optional] 
**state** | [**PipelineTaskTaskState**](PipelineTaskTaskState.md) |  | [optional] 
**status_metadata** | [**PipelineTaskStatusMetadata**](PipelineTaskStatusMetadata.md) |  | [optional] 
**state_history** | [**list[PipelineTaskTaskStatus]**](PipelineTaskTaskStatus.md) | A sequence of task statuses. This field keeps a record of state transitions. | [optional] 
**type** | [**PipelineTaskTaskType**](PipelineTaskTaskType.md) |  | [optional] 
**type_attributes** | [**PipelineTaskTypeAttributes**](PipelineTaskTypeAttributes.md) |  | [optional] 
**parent_task_id** | **str** | ID of the parent task if the task is within a component scope. Empty if the task is at the root level. | [optional] 
**child_tasks** | [**list[PipelineTaskChildTask]**](PipelineTaskChildTask.md) | Sequence of dependent tasks. | [optional] 
**inputs** | [**PipelineTaskInputOutputs**](PipelineTaskInputOutputs.md) |  | [optional] 
**outputs** | [**PipelineTaskInputOutputs**](PipelineTaskInputOutputs.md) |  | [optional] 
**scope_path** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


