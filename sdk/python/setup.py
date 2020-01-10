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

import os
import re
from setuptools import setup

NAME = 'kfp'
#VERSION = .... Change the version in kfp/__init__.py

REQUIRES = [
    'urllib3>=1.15,<1.25',  #Fixing the version conflict with the "requests" package
    'six >= 1.10',
    'certifi',
    'python-dateutil',
    'PyYAML',
    'google-cloud-storage>=1.13.0',
    'kubernetes>=8.0.0, <=10.0.0',
    'PyJWT>=1.6.4',
    'cryptography>=2.4.2',
    'google-auth>=1.6.1',
    'requests_toolbelt>=0.8.0',
    'cloudpickle==1.1.1',
    'kfp-server-api >= 0.1.18, <= 0.1.40',  #Update the upper version whenever a new version of the kfp-server-api package is released. Update the lower version when there is a breaking change in kfp-server-api.
    'argo-models == 2.2.1a',  #2.2.1a is equivalent to argo 2.2.1
    'jsonschema >= 3.0.1',
    'tabulate == 0.8.3',
    'click == 7.0',
    'Deprecated',
]

def find_version(*file_path_parts):
    here = os.path.abspath(os.path.dirname(__file__))
    with open(os.path.join(here, *file_path_parts), 'r') as fp:
        version_file_text = fp.read()

    version_match = re.search(
        r"^__version__ = ['\"]([^'\"]*)['\"]",
        version_file_text,
        re.M,
    )
    if version_match:
        return version_match.group(1)

    raise RuntimeError("Unable to find version string.")

setup(
    name=NAME,
    version=find_version("kfp", "__init__.py"),
    description='KubeFlow Pipelines SDK',
    author='google',
    install_requires=REQUIRES,
    packages=[
        'kfp',
        'kfp.cli',
        'kfp.cli.diagnose_me',
        'kfp.compiler',
        'kfp.components',
        'kfp.components.structures',
        'kfp.components.structures.kubernetes',
        'kfp.containers',
        'kfp.dsl',
        'kfp.dsl.extensions',
        'kfp.notebook',
    ],
    classifiers=[
        'Intended Audience :: Developers',
        'Intended Audience :: Education',
        'Intended Audience :: Science/Research',
        'License :: OSI Approved :: Apache Software License',
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
    python_requires='>=3.5.3',
    include_package_data=True,
    entry_points={
        'console_scripts': [
            'dsl-compile = kfp.compiler.main:main', 'kfp=kfp.__main__:main'
        ]
    })
