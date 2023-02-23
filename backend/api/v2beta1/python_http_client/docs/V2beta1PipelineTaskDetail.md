# V2beta1PipelineTaskDetail

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
**executor_detail** | [**V2beta1PipelineTaskExecutorDetail**](V2beta1PipelineTaskExecutorDetail.md) |  | [optional] 
**state** | [**V2beta1RuntimeState**](V2beta1RuntimeState.md) |  | [optional] 
**execution_id** | **str** | Execution id of the corresponding entry in ML metadata store. | [optional] 
**error** | [**GooglerpcStatus**](GooglerpcStatus.md) |  | [optional] 
**inputs** | [**dict(str, V2beta1ArtifactList)**](V2beta1ArtifactList.md) | Input artifacts of the task. | [optional] 
**outputs** | [**dict(str, V2beta1ArtifactList)**](V2beta1ArtifactList.md) | Output artifacts of the task. | [optional] 
**parent_task_id** | **str** | ID of the parent task if the task is within a component scope. Empty if the task is at the root level. | [optional] 
**state_history** | [**list[V2beta1RuntimeStatus]**](V2beta1RuntimeStatus.md) | A sequence of task statuses. This field keeps a record  of state transitions. | [optional] 
**pod_name** | **str** | Name of the corresponding pod assigned by the orchestration engine. Also known as node_id. | [optional] 
**child_tasks** | [**list[PipelineTaskDetailChildTask]**](PipelineTaskDetailChildTask.md) | Sequence of dependen tasks. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


