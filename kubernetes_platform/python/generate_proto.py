# Copyright 2023 The Kubeflow Authors
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

import os
import re
import subprocess

try:
    from distutils.spawn import find_executable
except ImportError:
    from shutil import which as find_executable

PLATFORM_DIR = os.path.realpath(os.path.dirname(os.path.dirname(__file__)))
PIPELINE_API_PROTO_DIR = os.path.join(
    os.path.dirname(PLATFORM_DIR), 'api', 'v2alpha1')
PROTO_DIR = os.path.join(PLATFORM_DIR, 'proto')

PKG_DIR = os.path.realpath(
    os.path.join(PLATFORM_DIR, '../sdk/python/kfp/kubernetes'))

# Find the Protocol Compiler. (Taken from protobuf/python/setup.py)
if 'PROTOC' in os.environ and os.path.exists(os.environ['PROTOC']):
    PROTOC = os.environ['PROTOC']
else:
    PROTOC = find_executable('protoc')


def replace_import(file_path: str) -> None:
    """Point the generated executor configuration at the SDK pipeline spec."""
    with open(file_path, 'r') as f:
        content = f.read()

    # Replace the specific import line
    new_content = re.sub(
        r'^import\s+pipeline_spec_pb2\s+as\s+pipeline__spec__pb2\s*$',
        'from kfp.pipeline_spec import pipeline_spec_pb2 as pipeline__spec__pb2\n',
        content,
        flags=re.MULTILINE)

    # Only write back if something changed
    if content != new_content:
        with open(file_path, 'w') as f:
            f.write(new_content)
        print(f"Updated imports in {file_path}")
    else:
        print(f"No changes needed in {file_path}")


def generate_proto(source: str) -> None:
    """Generate a _pb2.py from a .proto file.

    Invokes the Protocol Compiler to generate a _pb2.py from the given
    .proto file. Always regenerate to account for imported pipeline schemas.

    Args:
      source: The source proto file that needs to be compiled.
    """
    if not os.path.isfile(source):
        raise FileNotFoundError(f"Can't find required file: {source}")
    if PROTOC is None:
        raise RuntimeError(
            'protoc is not found. Install the protobuf compiler.')
    os.makedirs(PKG_DIR, exist_ok=True)
    subprocess.run([
        PROTOC,
        f'-I={PIPELINE_API_PROTO_DIR}',
        f'-I={PROTO_DIR}',
        '--experimental_allow_proto3_optional',
        f'--python_out={PKG_DIR}',
        source,
    ],
                   check=True)
    replace_import(f'{PKG_DIR}/kubernetes_executor_config_pb2.py')


if __name__ == '__main__':
    generate_proto(os.path.join(PROTO_DIR, 'kubernetes_executor_config.proto'))
