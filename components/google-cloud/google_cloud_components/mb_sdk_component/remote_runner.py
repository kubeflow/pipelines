# Copyright 2020 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Module for remote execution of Model Builder SDK as a pipeline component"""

import inspect
import argparse
import os
from google.cloud import aiplatform
from google.cloud import storage

INIT_KEY = 'init'
METHOD_KEY = 'method'

def split_args(kwargs):
    init_args = {}
    method_args = {}
    config_args = {}
    
    for key, arg in kwargs.items():
        if key.startswith(INIT_KEY):
            init_args[key.split(".")[-1]] = arg
        elif key.startswith(METHOD_KEY):
            method_args[key.split(".")[-1]] = arg
        else:
            config_args[key.split(".")[-1]] = arg
    
    return init_args, method_args, config_args

def write_to_gcs(project, gcs_uri, text):
    
    gcs_uri = gcs_uri[5:]
    gcs_bucket, gcs_blob = gcs_uri.split('/', 1)
    
    with open('resource_name.txt', 'w') as f:
        f.write(text)

    client = storage.Client(project=project)
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.blob(os.path.join(gcs_blob, 'resource_name.txt'))
    blob.upload_from_filename('resource_name.txt')

def read_from_gcs(project, gcs_uri):
    gcs_uri = gcs_uri[5:]
    gcs_bucket, gcs_blob = gcs_uri.split('/', 1)
    client = storage.Client(project=project)
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.get_blob(os.path.join(gcs_blob, 'resource_name.txt'))
    resource_name = blob.download_as_string().decode('utf-8')
    return resource_name

def resolve_project(serialized_args):
    return serialized_args['init'].get('project', serialized_args['method'].get('project'))

def resolve_input_args(value, _type, project):
    """If this is an input from Pipelines, read it directly from gcs."""
    if issubclass(_type, aiplatform.base.AiPlatformResourceNoun):
        if value.startswith('gs://'): # not a resource noun:
            value = read_from_gcs(project, value)
    return value

def resolve_init_args(key, value, project):
    if key.endswith('_name'):
        if value.startswith('gs://'): # not a resource noun
            value = read_from_gcs(project, value)
    return value



def runner(cls_name, method_name, resource_name_output_uri, kwargs):
    cls = getattr(aiplatform, cls_name)

    init_args, method_args, config_args = split_args(kwargs)

    serialized_args = {
        'init': init_args,
        'method': method_args
    }

    project = resolve_project(serialized_args)

    for key, param in inspect.signature(cls.__init__).parameters.items():
        if key in serialized_args['init']:
            serialized_args['init'][key] = resolve_init_args(
                key,
                serialized_args['init'][key],
                project)
            
    
    print(serialized_args['init'])
    obj = cls(**serialized_args['init']) if serialized_args['init'] else cls

    
    method = getattr(obj, method_name)

    for key, param in inspect.signature(method).parameters.items():
        if key in serialized_args['method']:
            type_args = getattr(param.annotation, '__args__', param.annotation)
            cast = type_args[0] if isinstance(type_args, tuple) else type_args
            serialized_args['method'][key] = resolve_input_args(serialized_args['method'][key], cast, project)
            serialized_args['method'][key] = cast(serialized_args['method'][key])
    
    print(serialized_args['method'])
    output = method(**serialized_args['method'])
    
    if output:
        write_to_gcs(
            project,
            resource_name_output_uri,
            output.resource_name)
        return output


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cls_name", type=str)
    parser.add_argument("--method_name", type=str)
    parser.add_argument("--resource_name_output_uri", type=str)

    args, unknown_args = parser.parse_known_args()
    kwargs = {}

    key_value = None
    for arg in unknown_args:
        print(arg)
        if "=" in arg:
            key, value = arg[2:].split("=")
            kwargs[key] = value
        else:
            if not key_value:
                key_value = arg[2:]
            else:
                kwargs[key_value] = arg
                key_value = None

        
    print(runner(
        args.cls_name,
        args.method_name,
        args.resource_name_output_uri,    
        kwargs))



if __name__ == "__main__":
    main()