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

import os
import subprocess
import sys
import setuptools
try:
  from distutils.spawn import find_executable
except ImportError:
  from shutil import which as find_executable

NAME = "kfp-pipeline-spec"
VERSION = "0.1.4"

PROTO_DIR = os.path.realpath(
    os.path.join(os.path.dirname(__file__), os.pardir))

PKG_DIR = os.path.realpath(
    os.path.join(os.path.dirname(__file__), "kfp", "pipeline_spec"))

# Find the Protocol Compiler. (Taken from protobuf/python/setup.py)
if "PROTOC" in os.environ and os.path.exists(os.environ["PROTOC"]):
  PROTOC = os.environ["PROTOC"]
else:
  PROTOC = find_executable("protoc")


def GenerateProto(source):
  """Generate a _pb2.py from a .proto file.

  Invokes the Protocol Compiler to generate a _pb2.py from the given
  .proto file.  Does nothing if the output already exists and is newer than
  the input.

  Args:
    source: The source proto file that needs to be compiled.
  """

  output = source.replace(".proto", "_pb2.py")

  if not os.path.exists(output) or (
      os.path.exists(source) and
      os.path.getmtime(source) > os.path.getmtime(output)):
    print("Generating %s..." % output)

    if not os.path.exists(source):
      sys.stderr.write("Can't find required file: %s\n" % source)
      sys.exit(-1)

    if PROTOC is None:
      sys.stderr.write("protoc is not found.  Please compile it "
                       "or install the binary package.\n")
      sys.exit(-1)

    protoc_command = [
        PROTOC, "-I%s" % PROTO_DIR,
        "--python_out=%s" % PKG_DIR, source
    ]
    if subprocess.call(protoc_command) != 0:
      sys.exit(-1)


# Generate the protobuf files that we depend on.
GenerateProto(os.path.join(PROTO_DIR, "pipeline_spec.proto"))

setuptools.setup(
    name=NAME,
    version=VERSION,
    description="Kubeflow Pipelines pipeline spec",
    author="google",
    author_email="kubeflow-pipelines@google.com",
    url="https://github.com/kubeflow/pipelines",
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    python_requires=">=3.5.3",
    include_package_data=True,
    license="Apache 2.0",
)
