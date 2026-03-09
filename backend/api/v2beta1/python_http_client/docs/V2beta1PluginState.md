# V2beta1PluginState

Describes the state of a plugin's operations. Unlike RuntimeState (which covers pipeline/task lifecycle including CANCELING, PAUSED, SKIPPED), PluginState only reflects whether the plugin's own work succeeded or failed, independent of the pipeline run outcome.   - PLUGIN_STATE_UNSPECIFIED: Default value. The plugin state is unknown or not yet set.  - PLUGIN_RUNNING: Plugin operations are in progress.  - PLUGIN_SUCCEEDED: Plugin operations completed successfully.  - PLUGIN_FAILED: Plugin operations failed.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


