# RequiredInputPipelineObjectWithUpdatedFieldsPipelineIdIsRequiredToIdentifyThePipelineMutableFieldsDisplayNameTags

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**display_name** | **str** | Required if name is not provided. Pipeline display name provided by user. | [optional] 
**name** | **str** | Required if display_name is not provided. Pipeline name provided by user. | [optional] 
**description** | **str** | Optional input field. A short description of the pipeline. | [optional] 
**created_at** | **datetime** | Output. Creation time of the pipeline. | [optional] 
**namespace** | **str** | Input. A namespace this pipeline belongs to. Causes error if user is not authorized to access the specified namespace. If not specified in CreatePipeline, default namespace is used. | [optional] 
**error** | [**GooglerpcStatus**](GooglerpcStatus.md) |  | [optional] 
**tags** | **dict(str, str)** | Optional. User-defined tags as key-value pairs associated with the pipeline. Tags can be used to label and categorize pipelines. Constraints:   - Maximum 20 tags per pipeline.   - Both key and value are limited to 63 characters.   - Keys cannot be empty and must not contain the &#39;.&#39; character. Tags can be filtered on when listing pipelines using the filter parameter with keys prefixed by \&quot;tags.\&quot; (e.g., filter predicate key \&quot;tags.team\&quot; with string_value \&quot;ml-ops\&quot;). | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


