# V2RuntimeConfig

The runtime config.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**parameters** | **dict(str, object)** | The runtime parameters of the Pipeline. The parameters will be used to replace the placeholders at runtime. | [optional] 
**pipeline_root** | **str** |  | [optional] 
**enable_caching** | **bool** | Overrides caching for every task in the run, including tasks in nested DAGs. When unset, each task keeps the caching setting it was compiled with. When true, caching is still disabled if the deployment disables it. For recurring runs, the setting applies to every triggered run. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


