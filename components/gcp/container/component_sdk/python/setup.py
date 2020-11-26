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

PACKAGE_NAME = 'kfp-component'
VERSION = '1.1.0-alpha.1'

setup(
    name=PACKAGE_NAME,
    version=VERSION,
    description='KubeFlow Pipelines Component SDK',
    author='google',
    install_requires=[
        'kubernetes >= 8.0.1', 'urllib3>=1.15,<1.25', 'fire == 0.1.3',
        'google-api-python-client == 1.7.8', 'google-cloud-storage == 1.14.0',
        'google-cloud-bigquery == 1.9.0'
    ],
    packages=[
        'kfp_component',
    ],
    classifiers=[
        'Intended Audience :: Developers',
        'Intended Audience :: Education',
        'Intended Audience :: Science/Research',
        'License :: OSI Approved :: Apache Software License',
        'Programming Language :: Python',
        'Programming Language :: Python :: 2',
        'Programming Language :: Python :: 2.7',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.5',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Topic :: Scientific/Engineering',
        'Topic :: Scientific/Engineering :: Artificial Intelligence',
        'Topic :: Software Development',
        'Topic :: Software Development :: Libraries',
        'Topic :: Software Development :: Libraries :: Python Modules',
    ],
    include_package_data=True,
)
