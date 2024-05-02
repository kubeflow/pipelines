# Copyright 2023 The Kubeflow Authors
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

import setuptools

NAME = 'kfp-kubernetes'
REQUIREMENTS = [
    'protobuf>=4.21.1,<5',
    'kfp>=2.6.0,<3',
]
DEV_REQUIREMENTS = [
    'docformatter==1.4',
    'isort==5.10.1',
    'mypy==0.941',
    'pre-commit==2.19.0',
    'pycln==2.1.1',
    'pytest==7.1.2',
    'pytest-xdist==2.5.0',
    'yapf==0.32.0',
]


def find_version(*file_path_parts: str) -> str:
    """Get version from kfp.__init__.__version__."""

    file_path = os.path.join(os.path.dirname(__file__), *file_path_parts)
    with open(file_path, 'r') as f:
        version_file_text = f.read()

    version_match = re.search(
        r"^__version__ = ['\"]([^'\"]*)['\"]",
        version_file_text,
        re.M,
    )
    if version_match:
        return version_match[1]
    else:
        raise ValueError('Could not find version.')


def read_readme() -> str:
    readme_path = os.path.join(os.path.dirname(__file__), 'README.md')
    with open(readme_path) as f:
        return f.read()


setuptools.setup(
    name=NAME,
    version=find_version('kfp', 'kubernetes', '__init__.py'),
    description='Kubernetes platform configuration library and generated protos.',
    long_description=read_readme(),
    long_description_content_type='text/markdown',
    author='google',
    author_email='kubeflow-pipelines@google.com',
    url='https://github.com/kubeflow/pipelines',
    project_urls={
        'Documentation':
            'https://kfp-kubernetes.readthedocs.io/',
        'Bug Tracker':
            'https://github.com/kubeflow/pipelines/issues',
        'Source':
            'https://github.com/kubeflow/pipelines/tree/master/kubernetes_platform/python',
    },
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    python_requires='>=3.8.0,<3.13.0',
    install_requires=REQUIREMENTS,
    include_package_data=True,
    extras_require={
        'dev': DEV_REQUIREMENTS,
    },
    license='Apache 2.0',
)
