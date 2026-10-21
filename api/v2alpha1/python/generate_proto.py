# Copyright 2022 The Kubeflow Authors
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

try:
    from distutils.spawn import find_executable
except ImportError:
    from shutil import which as find_executable

PROTO_DIR = os.path.realpath(os.path.join(os.path.dirname(__file__), os.pardir))

PKG_DIR = os.path.realpath(
    os.path.join(
        os.path.dirname(__file__), "../../../sdk/python/kfp/pipeline_spec"))

# Find the Protocol Compiler. (Taken from protobuf/python/setup.py)
if "PROTOC" in os.environ and os.path.exists(os.environ["PROTOC"]):
    PROTOC = os.environ["PROTOC"]
else:
    PROTOC = find_executable("protoc")


def generate_proto(source: str) -> None:
    """Generate a _pb2.py from a .proto file.

    Invokes the Protocol Compiler to generate a _pb2.py from the given
    .proto file. Always regenerate so imported schemas cannot leave stale code.

    Args:
      source: The source proto file that needs to be compiled.
    """

    if not os.path.isfile(source):
        raise FileNotFoundError(f"Can't find required file: {source}")
    if PROTOC is None:
        raise RuntimeError(
            'protoc is not found. Install the protobuf compiler.')
    os.makedirs(PKG_DIR, exist_ok=True)
    subprocess.run(
        [PROTOC, f'-I{PROTO_DIR}', f'--python_out={PKG_DIR}', source],
        check=True)


if __name__ == '__main__':
    # Generate the protobuf files that we depend on.
    generate_proto(os.path.join(PROTO_DIR, "pipeline_spec.proto"))
