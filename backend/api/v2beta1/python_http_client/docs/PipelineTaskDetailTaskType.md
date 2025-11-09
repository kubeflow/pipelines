# PipelineTaskDetailTaskType

 - ROOT: Root task is the top ancestor task to all tasks in the pipeline run It is the only task with no parent task in a Pipeline Run.  - RUNTIME: All child tasks in the Run DAG are Runtime tasks. With the exception of K8S driver pods. These tasks are the only tasks that have Executor Pods.  - CONDITION_BRANCH: Condition Branch is the wrapper task of an If block  - CONDITION: Condition is an individual \"if\" branch, and is a child to a CONDITION_BRANCH task.  - LOOP: Task Group for CONDITION_BRANCH Task Group for RUNTIME Loop Iterations  - DAG: Generic DAG task type for types like Nested Pipelines where there is no declarative way to detect this within a driver.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


