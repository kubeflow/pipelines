# V2beta1CreateArtifactRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact** | [**V2beta1Artifact**](V2beta1Artifact.md) |  | [optional] 
**run_id** | **str** | An artifact is always created in the context of a run. | [optional] 
**task_id** | **str** | The Task that is associated with the creation of this artifact. | [optional] 
**producer_key** | **str** | The outgoing parameter name of this Artifact within this task&#39;s component spec. For example: def preprocess(my_output: dsl.Outputs[dsl.Artifact]):   ... here the producer_key &#x3D;&#x3D; \&quot;my_output\&quot; Note that IOProducer.task_name is the same as task_name. | [optional] 
**iteration_index** | **str** |  | [optional] 
**reuse_if_exists** | **bool** | When true, create the artifact only if no existing artifact in the same namespace matches the stable reuse identity (type, URI, name, description, and metadata). Concurrent callers race-safely share one row and each still receive their own artifact-task link. When false (default), always insert a new artifact row so reimport&#x3D;true and ordinary outputs may intentionally create duplicates. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


