# V2beta1Filter

Filter is used to filter resources returned from a ListXXX request.  Example filters: 1) Filter runs with status = 'Running' filter {   predicate {     key: \"status\"     operation: EQUALS     string_value: \"Running\"   } }  2) Filter runs that succeeded since Dec 1, 2018 filter {   predicate {     key: \"status\"     operation: EQUALS     string_value: \"Succeeded\"   }   predicate {     key: \"created_at\"     operation: GREATER_THAN     timestamp_value {       seconds: 1543651200     }   } }  3) Filter runs with one of labels 'label_1' or 'label_2'  filter {   predicate {     key: \"label\"     operation: IN     string_values {       value: 'label_1'       value: 'label_2'     }   } }
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**predicates** | [**list[V2beta1Predicate]**](V2beta1Predicate.md) | All predicates are AND-ed when this filter is applied. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


