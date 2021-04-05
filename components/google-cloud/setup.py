# Copyright 2021 Google LLC. All Rights Reserved.
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

import importlib
import os
import types
import dependencies

from setuptools import find_packages
from setuptools import setup

relative_directory = os.path.relpath(os.path.dirname(os.path.abspath(__file__)))
loader = importlib.machinery.SourceFileLoader(
    fullname="version",
    path=os.path.join(relative_directory, "google_cloud_components/version.py"),
)
version = types.ModuleType(loader.name)
loader.exec_module(version)

setup(
    name="google-cloud-components",
    version=version.__version__,
    description="This SDK enables a set of First Party (Google owned) pipeline"
    " components that allow users to take their experience from AI Platform"
    " SDK and other Google Cloud services and create a corresponding pipeline"
    " using KFP or Managed Pipelines.",
    url="https://github.com/kubeflow/pipelines/tree/master/components/google-cloud",
    author="The Google Cloud Components authors",
    author_email="TBD",
    license="Apache License 2.0",
    extras_require={"tests": dependencies.make_required_test_packages()},
    include_package_data=True,
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
        "Programming Language :: Python :: 3.6",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Topic :: Scientific/Engineering",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
        "Topic :: Software Development",
        "Topic :: Software Development :: Libraries",
        "Topic :: Software Development :: Libraries :: Python Modules",
    ],
    package_dir={
        "google_cloud_components":
            os.path.join(relative_directory, "google_cloud_components")
    },
    packages=find_packages(where=relative_directory),
)
