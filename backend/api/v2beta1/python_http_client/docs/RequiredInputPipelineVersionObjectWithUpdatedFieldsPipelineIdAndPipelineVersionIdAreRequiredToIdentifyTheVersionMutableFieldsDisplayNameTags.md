# RequiredInputPipelineVersionObjectWithUpdatedFieldsPipelineIdAndPipelineVersionIdAreRequiredToIdentifyTheVersionMutableFieldsDisplayNameTags

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**display_name** | **str** | Required if name is not provided. Pipeline version display name provided by user. This is ignored in CreatePipelineAndVersion API. | [optional] 
**name** | **str** | Required if display_name is not provided. Pipeline version name provided by user. This is ignored in CreatePipelineAndVersion API. | [optional] 
**description** | **str** | Optional input field. Short description of the pipeline version. This is ignored in CreatePipelineAndVersion API. | [optional] 
**created_at** | **datetime** | Output. Creation time of the pipeline version. | [optional] 
**package_url** | [**V2beta1Url**](V2beta1Url.md) |  | [optional] 
**code_source_url** | **str** | Input. Optional. The URL to the code source of the pipeline version. The code is usually the Python definition of the pipeline and potentially related the component definitions. This allows users to trace back to how the pipeline YAML was created. | [optional] 
**pipeline_spec** | [**object**](.md) | Output. The pipeline spec for the pipeline version. | [optional] 
**error** | [**GooglerpcStatus**](GooglerpcStatus.md) |  | [optional] 
**tags** | **dict(str, str)** | Optional input field. User-defined tags (key-value pairs) for the pipeline version. Constraints:   - Maximum 20 tags per pipeline version.   - Both keys and values are limited to 63 characters.   - Keys cannot be empty and must not contain the &#39;.&#39; character. Tags are stored in a separate table and can be retrieved via Get and List APIs. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


