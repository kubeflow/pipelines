# V2beta1PipelineTaskExecutorDetail

Runtime information of a pipeline task executor.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**main_job** | **str** | The name of the job for the main container execution. | [optional] 
**pre_caching_check_job** | **str** | The name of the job for the pre-caching-check container execution. This job will be available if the Run.pipeline_spec specifies the &#x60;pre_caching_check&#x60; hook in the lifecycle events. | [optional] 
**failed_main_jobs** | **list[str]** | The names of the previously failed job for the main container executions. The list includes the all attempts in chronological order. | [optional] 
**failed_pre_caching_check_jobs** | **list[str]** | The names of the previously failed job for the pre-caching-check container executions. This job will be available if the Run.pipeline_spec specifies the &#x60;pre_caching_check&#x60; hook in the lifecycle events. The list includes the all attempts in chronological order. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


