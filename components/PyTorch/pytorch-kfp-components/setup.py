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

import os
import re
from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

relative_directory = os.path.relpath(os.path.dirname(os.path.abspath(__file__)))
# get version string from module
with open(
        os.path.join(os.path.dirname(__file__),
                     "pytorch_kfp_components/__init__.py"),
        "r",
) as f:
    version = re.search(r"__version__ = ['\"]([^'\"]*)['\"]", f.read(),
                        re.M).group(1)

setup(
    name="pytorch-kfp-components",
    version=version,
    description="PyTorch Kubeflow Pipeline",
    url="https://github.com/kubeflow/pipelines/tree/master/components",
    author="The PyTorch Pipeline Components authors",
    author_email="PyTorch-pipeline-components@fb.com",
    license="BSD",
    long_description=long_description,
    long_description_content_type="text/markdown",
    include_package_data=True,
    python_requires=">=3.6",
    install_requires=["torch", "kfp", "torchserve", "pytorch-lightning==1.3.2"],
    keywords=[
        "Kubeflow",
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
        "License :: OSI Approved :: BSD License",
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
    packages=find_packages(),
)
