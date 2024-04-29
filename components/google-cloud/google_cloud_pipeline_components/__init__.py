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
"""Google Cloud Pipeline Components."""
import sys
import warnings

from google_cloud_pipeline_components.version import __version__

if sys.version_info < (3, 9):
  warnings.warn(
      (
          ' Google Cloud Pipeline Components will drop support for Python 3.8'
          ' on Oct 1, 2024. To use new versions of the GCPC SDK after that'
          ' date, you will need to upgrade to Python >= 3.9. See'
          ' https://devguide.python.org/versions/ for more details.'
      ),
      FutureWarning,
      stacklevel=2,
  )
