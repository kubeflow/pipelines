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
import subprocess
import sys

try:
    from distutils.spawn import find_executable
except ImportError:
    from shutil import which as find_executable

PLATFORM_DIR = os.path.realpath(os.path.dirname(os.path.dirname(__file__)))

PROTO_DIR = os.path.join(PLATFORM_DIR, 'proto')

PKG_DIR = os.path.realpath(
    os.path.join(PLATFORM_DIR, 'python', 'kfp', 'kubernetes'))

# Find the Protocol Compiler. (Taken from protobuf/python/setup.py)
if 'PROTOC' in os.environ and os.path.exists(os.environ['PROTOC']):
    PROTOC = os.environ['PROTOC']
else:
    PROTOC = find_executable('protoc')


def generate_proto(source: str) -> None:
    """Generate a _pb2.py from a .proto file.

    Invokes the Protocol Compiler to generate a _pb2.py from the given
    .proto file.  Does nothing if the output already exists and is newer than
    the input.

    Args:
      source: The source proto file that needs to be compiled.
    """
    output = source.replace('.proto', '_pb2.py')

    if not os.path.exists(output) or (
            os.path.exists(source) and
            os.path.getmtime(source) > os.path.getmtime(output)):
        print(f'Generating {output}...')

        if not os.path.exists(source):
            sys.stderr.write(f"Can't find required file: {source}\n")
            sys.exit(-1)

        if PROTOC is None:
            sys.stderr.write(
                'protoc is not found. Please compile it or install the binary package.\n'
            )
            sys.exit(-1)

        protoc_command = [
            PROTOC,
            f'-I={PROTO_DIR}',
            f'--experimental_allow_proto3_optional',
            f'--python_out={PKG_DIR}',
            source,
        ]

        if subprocess.call(protoc_command) != 0:
            sys.exit(-1)


if __name__ == '__main__':
    generate_proto(os.path.join(PROTO_DIR, 'kubernetes_executor_config.proto'))
