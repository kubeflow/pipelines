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

"""Module that contains a set of commands to call ML Engine APIs

The commands are aware of KFP execution context and can work under 
retry and cancellation context. The currently supported commands
are: train, batch_prediction, create_model, create_version and 
delete_version.

TODO(hongyes): Provides full ML Engine API support.
"""

from ._create_job import create_job
from ._create_model import create_model
from ._create_version import create_version
from ._delete_version import delete_version
from ._train import train
from ._batch_predict import batch_predict
from ._deploy import deploy
from ._set_default_version import set_default_version
from ._wait_job import wait_job