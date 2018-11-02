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

from setuptools import setup, find_packages


setup(
  name='XGBoostPipeline',
  version='0.1.0',
  packages=find_packages(),
  description='XGBoost Pipeline with DataProc and Spark',
  author='Google',
  keywords=[
  ],
  license="Apache Software License",
  long_description="""
  """,
  install_requires=[
    'tensorflow==1.4.1',
  ],
  package_data={
  },
  data_files=[],
)
