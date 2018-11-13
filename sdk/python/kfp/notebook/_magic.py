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


try:
  import IPython
  import IPython.core.magic
except ImportError:
  raise Exception('This module can only be loaded in Jupyter.')


import os
import tempfile
from ..compiler import build_docker_image


@IPython.core.magic.register_cell_magic
def docker(line, cell):
  """cell magic for %%docker"""

  if len(line.split()) != 2:
    raise ValueError("usage: %%docker [gcr.io/project/image:tag] [gs://staging-bucket]")
  if not cell.strip():
    raise ValueError("Please fill in a dockerfile content in the cell.")

  target, staging = line.split()
  

  with tempfile.NamedTemporaryFile(mode='wt', delete=False) as f:
    f.write(cell)

  build_docker_image(staging, target, f.name)
  os.remove(f.name)
