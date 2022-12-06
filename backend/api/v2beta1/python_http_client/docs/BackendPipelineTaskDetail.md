# BackendPipelineTaskDetail

Runtime information of a task execution.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**run_id** | **str** | ID of the parent run. | [optional] 
**task_id** | **str** | System-generated ID of a task. | [optional] 
**display_name** | **str** | User specified name of a task that is defined in [Pipeline.spec][]. | [optional] 
**create_time** | **datetime** | Creation time of a task. | [optional] 
**start_time** | **datetime** | Starting time of a task. | [optional] 
**end_time** | **datetime** | Completion time of a task. | [optional] 
**executor_detail** | [**BackendPipelineTaskExecutorDetail**](BackendPipelineTaskExecutorDetail.md) |  | [optional] 
**state** | [**BackendRuntimeState**](BackendRuntimeState.md) |  | [optional] 
**execution_id** | **str** | Execution metadata of a task. | [optional] 
**error** | [**BackendError**](BackendError.md) |  | [optional] 
**inputs** | [**dict(str, BackendArtifactList)**](BackendArtifactList.md) | Input artifacts of the task. | [optional] 
**outputs** | [**dict(str, BackendArtifactList)**](BackendArtifactList.md) | Output artifacts of the task. | [optional] 
**parent_task_id** | **str** | ID of the parent task if the task is within a component scope. Empty if the task is at the root level. | [optional] 
**state_history** | [**list[BackendRuntimeStatus]**](BackendRuntimeStatus.md) | A sequence of task statuses. This field keeps a record  of state transitions. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


