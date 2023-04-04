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
"""Setup script."""

import glob
import importlib
import os
import types

import dependencies
from setuptools import find_packages
from setuptools import setup

relative_directory = os.path.relpath(os.path.dirname(os.path.abspath(__file__)))
GCPC_DIR_NAME = "google_cloud_pipeline_components"
relative_data_path = os.path.join(relative_directory, GCPC_DIR_NAME)

loader = importlib.machinery.SourceFileLoader(
    fullname="version",
    path=os.path.join(relative_directory, GCPC_DIR_NAME, "version.py"),
)
version = types.ModuleType(loader.name)
loader.exec_module(version)

# Get the long descriptions including link to RELEASE notes from README files.
with open("README.md") as fp:
  _GCPC_LONG_DESCRIPTION = fp.read()

setup(
    name="google-cloud-pipeline-components",
    version=version.__version__,
    description="This SDK enables a set of First Party (Google owned) pipeline"
    " components that allow users to take their experience from Vertex AI"
    " SDK and other Google Cloud services and create a corresponding pipeline"
    " using KFP or Managed Pipelines.",
    long_description=_GCPC_LONG_DESCRIPTION,
    long_description_content_type="text/markdown",
    url="https://github.com/kubeflow/pipelines/tree/master/components/google-cloud",
    author="The Google Cloud Pipeline Components authors",
    author_email="google-cloud-pipeline-components@google.com",
    license="Apache License 2.0",
    extras_require={"tests": dependencies.make_required_test_packages()},
    include_package_data=True,
    python_requires=">=3.7.0,<3.12.0",
    install_requires=dependencies.make_required_install_packages(),
    dependency_links=dependencies.make_dependency_links(),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Operating System :: Unix",
        "Operating System :: MacOS",
        "Intended Audience :: Developers",
        "Intended Audience :: Education",
        "Intended Audience :: Science/Research",
        "License :: OSI Approved :: Apache Software License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Topic :: Scientific/Engineering",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
        "Topic :: Software Development",
        "Topic :: Software Development :: Libraries",
        "Topic :: Software Development :: Libraries :: Python Modules",
    ],
    package_dir={
        GCPC_DIR_NAME: os.path.join(relative_directory, GCPC_DIR_NAME)
    },
    packages=find_packages(where=relative_directory, include="*"),
    package_data={
        GCPC_DIR_NAME: [
            x.replace(relative_data_path + "/", "") for x in
            glob.glob(relative_data_path + "/**/*.yaml", recursive=True) +
            glob.glob(
                relative_data_path + "/**/automl/**/*.json",
                recursive=True)
        ]
    },
)
