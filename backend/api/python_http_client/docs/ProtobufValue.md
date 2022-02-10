# ProtobufValue

`Value` represents a dynamically typed value which can be either null, a number, a string, a boolean, a recursive struct value, or a list of values. A producer of value is expected to set one of that variants, absence of any variant indicates an error.  The JSON representation for `Value` is JSON value.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**null_value** | [**ProtobufNullValue**](ProtobufNullValue.md) |  | [optional] 
**number_value** | **float** | Represents a double value. | [optional] 
**string_value** | **str** | Represents a string value. | [optional] 
**bool_value** | **bool** | Represents a boolean value. | [optional] 
**struct_value** | [**ProtobufStruct**](ProtobufStruct.md) |  | [optional] 
**list_value** | [**ProtobufListValue**](ProtobufListValue.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


