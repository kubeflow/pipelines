# Copyright 2018 The Kubeflow Authors
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

# NOTICE, after any updates to the following, ./requirements.in should be updated
# accordingly.
REQUIRES = [
    'absl-py>=0.9,<=0.11',
    'PyYAML>=5.3,<6',
    # `Blob.from_string` was introduced in google-cloud-storage 1.20.0
    # https://github.com/googleapis/python-storage/blob/master/CHANGELOG.md#1200
    'google-cloud-storage>=1.20.0,<2',
    'kubernetes>=8.0.0,<19',
    # NOTE: Maintainers, please do not require google-auth>=2.x.x
    # Until this issue is closed
    # https://github.com/googleapis/google-cloud-python/issues/10566
    'google-auth>=1.6.1,<3',
    'requests-toolbelt>=0.8.0,<1',
    'cloudpickle>=2.0.0,<3',
    # Update the upper version whenever a new major version of the
    # kfp-server-api package is released.
    # Update the lower version when kfp sdk depends on new apis/fields in
    # kfp-server-api.
    # Note, please also update ./requirements.in
    'kfp-server-api>=1.1.2,<2.0.0',
    'jsonschema>=3.0.1,<4',
    'tabulate>=0.8.6,<1',
    'click>=7.1.2,<9',
    'Deprecated>=1.2.7,<2',
    'strip-hints>=0.1.8,<1',
    'docstring-parser>=0.7.3,<1',
    'kfp-pipeline-spec>=0.1.13,<0.2.0',
    'fire>=0.3.1,<1',
    'protobuf>=3.13.0,<4',
    'uritemplate>=3.0.1,<4',
    'pydantic>=1.8.2,<2',
    'typer>=0.3.2,<1.0',
    # Standard library backports
    'dataclasses;python_version<"3.7"',
    'typing-extensions>=3.7.4,<4;python_version<"3.9"',
]

EXTRAS_REQUIRE = {
    'all': ['docker'],
}


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

    raise RuntimeError('Unable to find version string.')


setup(
    name=NAME,
    version=find_version('kfp', '__init__.py'),
    description='KubeFlow Pipelines SDK',
    author='The Kubeflow Authors',
    url="https://github.com/kubeflow/pipelines",
    project_urls={
        "Documentation":
            "https://kubeflow-pipelines.readthedocs.io/en/stable/",
        "Bug Tracker":
            "https://github.com/kubeflow/pipelines/issues",
        "Source":
            "https://github.com/kubeflow/pipelines/tree/master/sdk",
        "Changelog":
            "https://github.com/kubeflow/pipelines/blob/master/sdk/RELEASE.md",
    },
    install_requires=REQUIRES,
    extras_require=EXTRAS_REQUIRE,
    packages=[
        'kfp',
        'kfp.auth',
        'kfp.cli',
        'kfp.cli.diagnose_me',
        'kfp.compiler',
        'kfp.components',
        'kfp.components.structures',
        'kfp.containers',
        'kfp.dsl',
        'kfp.dsl.extensions',
        'kfp.notebook',
        'kfp.v2',
        'kfp.v2.compiler',
        'kfp.v2.components',
        'kfp.v2.components.types',
        'kfp.v2.dsl',
    ],
    classifiers=[
        'Intended Audience :: Developers',
        'Intended Audience :: Education',
        'Intended Audience :: Science/Research',
        'License :: OSI Approved :: Apache Software License',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Topic :: Scientific/Engineering',
        'Topic :: Scientific/Engineering :: Artificial Intelligence',
        'Topic :: Software Development',
        'Topic :: Software Development :: Libraries',
        'Topic :: Software Development :: Libraries :: Python Modules',
    ],
    python_requires='>=3.6.1',
    include_package_data=True,
    entry_points={
        'console_scripts': [
            'dsl-compile = kfp.compiler.main:main',
            'dsl-compile-v2 = kfp.v2.compiler.main:main',
            'kfp=kfp.__main__:main'
        ]
    })
