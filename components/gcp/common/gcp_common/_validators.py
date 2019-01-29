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
import re

def validate_required(value, field):
    if value == None or value == '':
        raise ValueError('{} is required.'.format(field))
    return value

def validate_type(value, value_type, field):
    if type(value) is not value_type:
        raise ValueError('{} is not in {} type'.format(
            field, str(value_type)))
    return value

def validate_project_id(project_id):
    return validate_regex(r'^[a-zA-Z0-9-:]+$', 
        project_id, 'project_id')

def validate_gcs_path(gcs_path, field):
    return validate_regex(r'^gs://[a-z0-9-_\./]+$', 
        gcs_path, field)

def validate_regex(pattern, value, field):
    if not re.match(pattern, value):
        raise ValueError('{} does not match with pattern {}.'.format(
            field, pattern
        ))
    return value

def validate_list(value, field):
    return validate_type(value, list, field)

def validate_dict(value, field):
    return validate_type(value, dict, field)