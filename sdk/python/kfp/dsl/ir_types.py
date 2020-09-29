# Copyright 2020 Google LLC
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
"""Utilities for component I/O type mapping."""

from kfp.v2.proto import pipeline_spec_pb2


# GCSPath is a special case
GCS_PATH_TYPE = 'gcspath'

# TODO: support more artifact types
ARTIFACT_TYPES_MAPPING = {
    GCS_PATH_TYPE: 'mlpipeline.Artifact',
    'model': 'mlpipeline.Model',
}

PARAMETER_TYPES_MAPPING = {
    'integer': pipeline_spec_pb2.PrimitiveType.INT,
    'int': pipeline_spec_pb2.PrimitiveType.INT,
    'double': pipeline_spec_pb2.PrimitiveType.DOUBLE,
    'float': pipeline_spec_pb2.PrimitiveType.DOUBLE,
    'string': pipeline_spec_pb2.PrimitiveType.STRING,
    'str': pipeline_spec_pb2.PrimitiveType.STRING,
}
