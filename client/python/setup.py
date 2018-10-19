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


from setuptools import setup


NAME = 'seira_client'
VERSION = '0.1'


REQUIRES = ['urllib3 >= 1.15', 'six >= 1.10', 'certifi', 'python-dateutil']

setup(
    name=NAME,
    version=VERSION,
    description='seira_client',
    author='google',
    install_requires=REQUIRES,
    packages=[
      'seira_job',
      'seira_job.api',
      'seira_job.models',
      'seira_upload',
      'seira_upload.api',
      'seira_upload.models',
      'seira_pipeline',
      'seira_pipeline.api',
      'seira_pipeline.models',
      'seira_run',
      'seira_run.api',
      'seira_run.models',
      'seira_client',
    ],
    include_package_data=True,
)
