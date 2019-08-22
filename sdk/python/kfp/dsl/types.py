# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

class BaseType:
	'''MetaType is a base type for all scalar and artifact types.
	'''
	pass

# Primitive Types
class Integer(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "integer"
		}

class String(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "string"
		}

class Float(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "number"
		}

class Bool(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "boolean"
		}

class List(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "array"
		}

class Dict(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "object",
		}

# GCP Types
class GCSPath(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "string",
			"pattern": "^gs://.*$"
		}

class GCRPath(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "string",
			"pattern": "^.*gcr\\.io/.*$"
		}

class GCPRegion(BaseType):
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "string"
		}

class GCPProjectID(BaseType):
	'''MetaGCPProjectID: GCP project id'''
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "string"
		}

# General Types
class LocalPath(BaseType):
	#TODO: add restriction to path
	def __init__(self):
		self.openapi_schema_validator = {
			"type": "string"
		}

class InconsistentTypeException(Exception):
	'''InconsistencyTypeException is raised when two types are not consistent'''
	pass

def check_types(checked_type, expected_type):
	'''check_types checks the type consistency.
	For each of the attribute in checked_type, there is the same attribute in expected_type with the same value.
	However, expected_type could contain more attributes that checked_type does not contain.
	Args:
		checked_type (BaseType/str/dict): it describes a type from the upstream component output
		expected_type (BaseType/str/dict): it describes a type from the downstream component input
		'''
	if isinstance(checked_type, BaseType):
		checked_type = _instance_to_dict(checked_type)
	elif isinstance(checked_type, str):
		checked_type = {checked_type: {}}
	if isinstance(expected_type, BaseType):
		expected_type = _instance_to_dict(expected_type)
	elif isinstance(expected_type, str):
		expected_type = {expected_type: {}}
	return _check_dict_types(checked_type, expected_type)

def _check_valid_type_dict(payload):
	'''_check_valid_type_dict checks whether a dict is a correct serialization of a type
	Args:
		payload(dict)
	'''
	if not isinstance(payload, dict) or len(payload) != 1:
		return False
	for type_name in payload:
		if not isinstance(payload[type_name], dict):
			return False
		property_types = (int, str, float, bool)
		property_value_types = (int, str, float, bool, dict)
		for property_name in payload[type_name]:
			if not isinstance(property_name, property_types) or not isinstance(payload[type_name][property_name], property_value_types):
				return False
	return True

def _instance_to_dict(instance):
	'''_instance_to_dict serializes the type instance into a python dictionary
	Args:
		instance(BaseType): An instance that describes a type

	Return:
		dict
	'''
	return {type(instance).__name__: instance.__dict__}

def _check_dict_types(checked_type, expected_type):
	'''_check_dict_types checks the type consistency.
	Args:
  	checked_type (dict): A dict that describes a type from the upstream component output
  	expected_type (dict): A dict that describes a type from the downstream component input
	'''
	if not checked_type or not expected_type:
		# If the type is empty, it matches any types
		return True
	checked_type_name,_ = list(checked_type.items())[0]
	expected_type_name,_ = list(expected_type.items())[0]
	if checked_type_name == '' or expected_type_name == '':
		# If the type name is empty, it matches any types
		return True
	if checked_type_name != expected_type_name:
		print('type name ' + str(checked_type_name) + ' is different from expected: ' + str(expected_type_name))
		return False
	type_name = checked_type_name
	for type_property in checked_type[type_name]:
		if type_property not in expected_type[type_name]:
			print(type_name + ' has a property ' + str(type_property) + ' that the latter does not.')
			return False
		if checked_type[type_name][type_property] != expected_type[type_name][type_property]:
			print(type_name + ' has a property ' + str(type_property) + ' with value: ' +
						str(checked_type[type_name][type_property]) + ' and ' +
						str(expected_type[type_name][type_property]))
			return False
	return True
