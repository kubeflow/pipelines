# V2beta1Artifact

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_id** | **str** |  | [optional] [readonly] 
**name** | **str** | Required. The client provided name of the artifact. Note: in MLMD when name was set, it had to be unique for that type_id this restriction is removed here If this is a \&quot;Metric\&quot; artifact, the name of the metric is treated as the Key in its K/V pair. | [optional] 
**description** | **str** |  | [optional] 
**type** | [**ArtifactArtifactType**](ArtifactArtifactType.md) |  | [optional] 
**uri** | **str** | The uniform resource identifier of the physical artifact. May be empty if there is no physical artifact. | [optional] 
**metadata** | **dict(str, object)** | Optional. User provided custom properties which are not defined by its type. | [optional] 
**number_value** | **float** |  | [optional] 
**created_at** | **datetime** | Output only. Create time of the artifact in millisecond since epoch. Note: The type and name is updated from mlmd artifact to be consistent with other backend apis. | [optional] [readonly] 
**namespace** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


