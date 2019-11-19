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

"""Entrypoint module to launch python module or file dynamically.

This module makes it easier to build kfp component with python code 
by defining a dynamic entrypoint and generate command line arg parser
by python-fire module. It can be used as an entrypoint in the 
container spec to run arbitary python module or code in the local 
image.
"""

from .launcher import launch