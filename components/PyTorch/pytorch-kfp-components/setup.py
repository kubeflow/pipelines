#!/usr/bin/env/python3
#
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
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

from setuptools import setup, find_packages


def make_required_install_packages():
    return [
      "pytorch-lightning>=1.4.0",
      "torch>=1.7.1",
      "torch-model-archiver",
    ]


def make_required_test_packages():
    return make_required_install_packages() + [
      "mock>=4.0.0",
      "flake8>=3.0.0",
      "pylint",
      "pytest>=6.0.0",
      "wget",
      "pandas",
      "minio"
    ]


def make_dependency_links():
    return []


def detect_version(base_path):
    loader = importlib.machinery.SourceFileLoader(
        fullname="version",
        path=os.path.join(base_path,
                          "pytorch_kfp_components/__init__.py"),
    )
    version = types.ModuleType(loader.name)
    loader.exec_module(version)
    return version.__version__


if __name__ == "__main__":
    relative_directory = os.path.relpath(
        os.path.dirname(os.path.abspath(__file__)))

    version = detect_version(relative_directory)

    setup(
        name="pytorch-kfp-components",
        version=version,
        description="PyTorch Kubeflow Pipeline",
        url="https://github.com/kubeflow/pipelines/tree/master/components/PyTorch/pytorch-kfp-components/",
        author="The PyTorch Kubeflow Pipeline Components authors",
        author_email="pytorch-kfp-components@fb.com",
        license="Apache License 2.0",
        extra_requires={"tests": make_required_test_packages()},
        include_package_data=True,
        python_requires=">=3.6",
        install_requires=make_required_install_packages(),
        dependency_links=make_dependency_links(),
        keywords=[
            "Kubeflow Pipelines",
            "KFP",
            "ML workflow",
            "PyTorch",
        ],
        classifiers=[
            "Development Status :: 4 - Beta",
            "Operating System :: Unix",
            "Operating System :: MacOS",
            "Intended Audience :: Developers",
            "Intended Audience :: Education",
            "Intended Audience :: Science/Research",
            "License :: OSI Approved :: Apache Software License",
            "Programming Language :: Python :: 3 :: Only",
            "Topic :: Scientific/Engineering",
            "Topic :: Scientific/Engineering :: Artificial Intelligence",
            "Topic :: Software Development",
            "Topic :: Software Development :: Libraries",
            "Topic :: Software Development :: Libraries :: Python Modules",
        ],
        package_dir={
            "pytorch_kfp_components":
                os.path.join(relative_directory, "pytorch_kfp_components")
        },
        packages=find_packages(where=relative_directory),
    )
