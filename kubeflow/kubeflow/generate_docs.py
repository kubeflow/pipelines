#!/usr/bin/env python

# Copyright 2018 The Kubeflow Authors All rights reserved.
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

import glob
import os
import subprocess

if __name__ == "__main__":
  this_dir = os.path.dirname(__file__)

  GOPATH = os.getenv("GOPATH")
  doc_gen = os.path.join(GOPATH, "bin/doc-gen")
  for f in os.listdir(this_dir):
    full_dir = os.path.join(this_dir, f)
    if not os.path.isdir(f):
      continue
    prototypes = glob.glob(os.path.join(full_dir, "prototypes/*.jsonnet"))


    command = [doc_gen, os.path.join(full_dir, "parts.yaml")]
    command.extend(prototypes)
    with open(os.path.join(full_dir, "README.md"), "w") as hout:
      subprocess.check_call(command, stdout=hout)
