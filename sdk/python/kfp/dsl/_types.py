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

class MetaType:
	'''MetaType is a base type for all scalar and artifact types.
	'''
	pass

class MetaGCSPath(MetaType):
	openapi_schema_validator = '''{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"pattern": "^gs://$"
			}
		}

	}'''
	# path_type describes the paths, for example, bucket, directory, file, etc.
	path_type = ''
	# file_type describes the files, for example, JSON, CSV, etc.
	file_type = ''

	def __init__(self, path):
		self.path = path

def GCSPath(attr={}):
	return type('GCSPath', (MetaGCSPath, ), attr)


class MetaGCPRegion(MetaType):
	openapi_schema_validator = '''{
		"type": "object",
			"properties": {
				"region": {
					"type": "string", 
					"enum": ["asia-east1","asia-east2","asia-northeast1",
					"asia-south1","asia-southeast1","australia-southeast1",
					"europe-north1","europe-west1","europe-west2",
					"europe-west3","europe-west4","northamerica-northeast1",
					"southamerica-east1","us-central1","us-east1",
					"us-east4","us-west1", "us-west4" ]
				}
	}'''

	def __init__(self, region):
		self.region = region

def GCPRegion(attr={}):
	return type('GCPRegion', (GCPRegion, ), attr)