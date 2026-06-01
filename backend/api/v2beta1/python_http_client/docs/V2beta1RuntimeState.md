# V2beta1RuntimeState

Describes the runtime state of an entity.   - RUNTIME_STATE_UNSPECIFIED: Default value. This value is not used.  - PENDING: Service is preparing to execute an entity.  - RUNNING: Entity execution is in progress.  - SUCCEEDED: Entity completed successfully.  - SKIPPED: Entity has been skipped. For example, due to caching.  - FAILED: Entity execution has failed.  - CANCELING: Entity is being canceled. From this state, an entity may only change its state to SUCCEEDED, FAILED or CANCELED.  - CANCELED: Entity has been canceled.  - PAUSED: Entity has been paused. It can be resumed.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


