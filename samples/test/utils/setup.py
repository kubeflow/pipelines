# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import setuptools

setuptools.setup(
    name="kfp-samples-test-utils",
    version="0.0.1",
    author="Kubeflow Pipelines Team",
    author_email="kubeflow-pipelines@google.com",
    description="Kubeflow Pipelines sample test utils",
    url="https://github.com/kubeflow/pipelines/blob/master/samples/test/utils",
    project_urls={
        "Bug Tracker": "https://github.com/kubeflow/pipelines/issues",
    },
    install_requires=[  # The exact versions should be determined elsewhere.
        'kfp',
        'ml-metadata',
        'nbconvert~=6.0',  # >=6.0 <7
    ],
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: Apache Software License",
        "Operating System :: OS Independent",
    ],
    package_dir={"": "."},
    packages=setuptools.find_packages(where="."),
    python_requires=">=3.6",
)
