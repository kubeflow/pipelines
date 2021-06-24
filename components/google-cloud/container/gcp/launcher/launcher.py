# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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

import argparse
import custom_job

def _make_parent_dirs_and_return_path(file_path: str):
    import os
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    return file_path

_parser = argparse.ArgumentParser(prog='Vertex Pipelines service launcher', description='')
_parser.add_argument("--type", dest="type", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--gcp_project", dest="gcp_project", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--gcp_region", dest="gcp_region", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--payload", dest="payload", type=str, required=True, default=argparse.SUPPRESS)
_parser.add_argument("--gcp_resources", dest="gcp_resources", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)
_parsed_args = vars(_parser.parse_args())

if _parsed_args['type']=='CustomJob':
    _outputs = custom_job.create_custom_job(**_parsed_args)
