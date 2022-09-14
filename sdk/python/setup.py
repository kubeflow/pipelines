# Copyright 2018-2022 The Kubeflow Authors
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
from typing import List

import setuptools


def get_requirements(requirements_file: str) -> List[str]:
    """Read requirements from requirements.in."""

    file_path = os.path.join(os.path.dirname(__file__), requirements_file)
    with open(file_path, 'r') as f:
        lines = f.readlines()
    lines = [line.strip() for line in lines]
    lines = [line for line in lines if not line.startswith('#') and line]
    return lines


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
        return version_match.group(1)

    raise RuntimeError(f'Unable to find version string in file: {file_path}.')


def read_readme() -> str:
    readme_path = os.path.join(os.path.dirname(__file__), 'README.md')
    with open(readme_path) as f:
        return f.read()


setuptools.setup(
    name='kfp',
    version=find_version('kfp', '__init__.py'),
    description='Kubeflow Pipelines SDK',
    long_description=read_readme(),
    long_description_content_type='text/markdown',
    author='The Kubeflow Authors',
    url='https://github.com/kubeflow/pipelines',
    project_urls={
        'Documentation':
            'https://kubeflow-pipelines.readthedocs.io/en/stable/',
        'Bug Tracker':
            'https://github.com/kubeflow/pipelines/issues',
        'Source':
            'https://github.com/kubeflow/pipelines/tree/master/sdk',
        'Changelog':
            'https://github.com/kubeflow/pipelines/blob/master/sdk/RELEASE.md',
    },
    install_requires=get_requirements('requirements.in'),
    extras_require={
        'all': ['docker'],
    },
    packages=setuptools.find_packages(exclude=['*test*']),
    classifiers=[
        'Intended Audience :: Developers',
        'Intended Audience :: Education',
        'Intended Audience :: Science/Research',
        'License :: OSI Approved :: Apache Software License',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Topic :: Scientific/Engineering',
        'Topic :: Scientific/Engineering :: Artificial Intelligence',
        'Topic :: Software Development',
        'Topic :: Software Development :: Libraries',
        'Topic :: Software Development :: Libraries :: Python Modules',
    ],
    python_requires='>=3.7.0',
    include_package_data=True,
    entry_points={
        'console_scripts': [
            'dsl-compile = kfp.cli.compile_:main',
            'dsl-compile-deprecated = kfp.deprecated.compiler.main:main',
            'kfp=kfp.cli.__main__:main',
        ]
    })
