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
	openapi_schema_validator = '''{
		"type": "integer"
	}'''

class String(BaseType):
	openapi_schema_validator = '''{
		"type": "string"
	}'''

class Float(BaseType):
	openapi_schema_validator = '''{
		"type": "number"
	}'''

class Bool(BaseType):
	openapi_schema_validator = '''{
		"type": "boolean"
	}'''

class List(BaseType):
	openapi_schema_validator = '''{
		"type": "array"
	}'''

class Dict(BaseType):
	openapi_schema_validator = '''{
		"type": "object",
	}'''

# GCP Types
class GCSPath(BaseType):
	openapi_schema_validator = '''{
		"type": "string",
		"pattern": "^gs://$"
	}'''

	def __init__(self, path_type='', file_type=''):
		'''
		Args
		:param path_type: describes the paths, for example, bucket, directory, file, etc
		:param file_type: describes the files, for example, JSON, CSV, etc.
		'''
		self.path_type = path_type
		self.file_type = file_type

class GCRPath(BaseType):
	openapi_schema_validator = '''{
		"type": "string",
		"pattern": "^(us.|eu.|asia.)?gcr\\.io/.*$"
		}
	}'''

class GCPRegion(BaseType):
	openapi_schema_validator = '''{
		"type": "string", 
		"enum": ["asia-east1","asia-east2","asia-northeast1",
					"asia-south1","asia-southeast1","australia-southeast1",
					"europe-north1","europe-west1","europe-west2",
					"europe-west3","europe-west4","northamerica-northeast1",
					"southamerica-east1","us-central1","us-east1",
					"us-east4","us-west1", "us-west4" ]
	}'''

class GCPProjectID(BaseType):
	'''MetaGCPProjectID: GCP project id'''
	openapi_schema_validator = '''{
		"type": "string"
	}'''

# General Types
class LocalPath(BaseType):
	#TODO: add restriction to path
	openapi_schema_validator = '''{
		"type": "string"
	}'''

class InconsistentTypeException(Exception):
	'''InconsistencyTypeException is raised when two types are not consistent'''
	pass

def _check_valid_dict(payload):
	'''_check_valid_dict_type checks whether a dict is a correct serialization of a type
	Args:
		payload(dict)
	'''
	if not isinstance(payload, dict) or len(payload) != 1:
		return False
	for type_name in payload:
		if not isinstance(payload[type_name], dict):
			return False
		property_types = (int, str, float, bool)
		for property_name in payload[type_name]:
			if not isinstance(property_name, property_types) or not isinstance(payload[type_name][property_name], property_types):
				return False
	return True

def _instance_to_dict(instance):
	'''serialize_type serializes the type instance into a json string
	Args:
		instance(BaseType): An instance that describes a type

	Return:
		dict
	'''
	return {type(instance).__name__: instance.__dict__}

def _str_to_dict(payload):
	import json
	json_dict = json.loads(payload)
	if not _check_valid_dict(json_dict):
		raise ValueError(payload + ' is not a valid type string')
	return json_dict

def _check_dict_types(typeA, typeB):
	'''_check_type_types checks the type consistency.
	Args:
  	typeA (dict): A dict that describes a type from the upstream component output
  	typeB (dict): A dict that describes a type from the downstream component input
	'''
	typeA_name,_ = list(typeA.items())[0]
	typeB_name,_ = list(typeB.items())[0]
	if typeA_name != typeB_name:
		return False
	type_name = typeA_name
	for type_property in typeA[type_name]:
		if type_property not in typeB[type_name]:
			print(type_name + ' has a property ' + str(type_property) + ' that the latter does not.')
			return False
		if typeA[type_name][type_property] != typeB[type_name][type_property]:
			print(type_name + ' has a property ' + str(type_property) + ' with value: ' +
						str(typeA[type_name][type_property]) + ' and ' +
						str(typeB[type_name][type_property]))
			return False
	return True

def check_types(typeA, typeB):
	'''check_types checks the type consistency.
	For each of the attribute in typeA, there is the same attribute in typeB with the same value.
	However, typeB could contain more attributes that typeA does not contain.
	Args:
		typeA (BaseType/str/dict): it describes a type from the upstream component output
		typeB (BaseType/str/dict): it describes a type from the downstream component input
		'''
	if isinstance(typeA, BaseType):
		typeA = _instance_to_dict(typeA)
	elif isinstance(typeA, str):
		typeA = _str_to_dict(typeA)
	if isinstance(typeB, BaseType):
		typeB = _instance_to_dict(typeB)
	elif isinstance(typeB, str):
		typeB = _str_to_dict(typeB)
	return _check_dict_types(typeA, typeB)