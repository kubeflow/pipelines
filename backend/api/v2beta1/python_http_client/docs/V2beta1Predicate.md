# V2beta1Predicate

Predicate captures individual conditions that must be true for a resource being filtered.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**operation** | [**V2beta1PredicateOperation**](V2beta1PredicateOperation.md) |  | [optional] 
**key** | **str** | Key for the operation (first argument). | [optional] 
**int_value** | **int** | Integer. | [optional] 
**long_value** | **str** | Long integer. | [optional] 
**string_value** | **str** | String. | [optional] 
**timestamp_value** | **datetime** | Timestamp values will be converted to Unix time (seconds since the epoch) prior to being used in a filtering operation. | [optional] 
**int_values** | [**PredicateIntValues**](PredicateIntValues.md) |  | [optional] 
**long_values** | [**PredicateLongValues**](PredicateLongValues.md) |  | [optional] 
**string_values** | [**PredicateStringValues**](PredicateStringValues.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


