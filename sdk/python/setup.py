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


NAME = 'kfp'
VERSION = '0.1'

REQUIRES = ['urllib3 >= 1.15', 'six >= 1.10', 'certifi', 'python-dateutil', 'PyYAML',
            'google-cloud-storage >= 1.13.0', 'kubernetes == 8.0.0']

setup(
    name=NAME,
    version=VERSION,
    description='KubeFlow Pipelines SDK',
    author='google',
    install_requires=REQUIRES,
    packages=[
      'kfp',
      'kfp.compiler',
      'kfp.components',
      'kfp.dsl',
      'kfp.notebook',
      'kfp_experiment',
      'kfp_experiment.api',
      'kfp_experiment.models',
      'kfp_run',
      'kfp_run.api',
      'kfp_run.models',
    ],
    include_package_data=True,
    entry_points = {
      'console_scripts': [ 
        'dsl-compile = kfp.compiler.main:main',
      ]
    }
)
