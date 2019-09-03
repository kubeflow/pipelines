# Copyright 2019 Google LLC
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
# See the License for the speci

__all__ = [
    'build_image_from_working_dir',
]


import logging
import os
import re
import shutil
import sys
import tempfile

import requests

from . import get_default_image_builder
from ..compiler._container_builder import ContainerBuilder


def get_python_image() -> str:
    return 'python' + ':{}.{}.{}'.format(sys.version_info[0], sys.version_info[1], sys.version_info[2])


default_base_image = get_python_image


_container_work_dir = '/python_env'


def generate_dockerfile_text(context_dir: str, dockerfile_path: str):
    # Generating the Dockerfile
    logging.info('Generating the Dockerfile')

    requirements_rel_path = 'requirements.txt'
    requirements_path = os.path.join(context_dir, requirements_rel_path)
    requirements_file_exists = os.path.exists(requirements_path)

    base_image = default_base_image
    if callable(base_image):
        base_image = base_image()

    dockerfile_lines = []
    dockerfile_lines.append('FROM {}'.format(base_image))
    dockerfile_lines.append('COPY . {}'.format(_container_work_dir))
    dockerfile_lines.append('WORKDIR {}'.format(_container_work_dir))
    if requirements_file_exists:
        dockerfile_lines.append('RUN python3 -m pip install -r {}'.format(requirements_rel_path))

    return '\n'.join(dockerfile_lines)


def build_image_from_working_dir(image_name : str = None, working_dir : str = None, file_filter_re : str = '.*\.py',  timeout : int = 1000, builder : ContainerBuilder = None) -> str:
    '''build_image_from_working_dir builds and pushes a new container image that captures the current python environment.
    Args:
        image_name: Optional. The image repo name where the new container image will be pushed. The name will be generated if not not set.
        working_dir: Optional. The directory that will be captured. The current directory will be used if omitted. The requirements.txt and python files inside the directory will be captured.
        timeout: Optional. The image building timeout in seconds.
        builder: Optional. An instance of ContainerBuilder or compatible class that will be used to build the image.
    Returns:
        The full name of the container image including the hash digest. E.g. gcr.io/my-org/my-image@sha256:86c1...793c.
    '''
    current_dir = working_dir or os.getcwd()
    with tempfile.TemporaryDirectory() as context_dir:
        logging.info('Creating the build context directory: {}'.format(context_dir))

        # Copying all *.py and requirements.txt files
        for dirpath, dirnames, filenames in os.walk(current_dir):
            dst_dirpath = os.path.join(context_dir, os.path.relpath(dirpath, current_dir))
            os.makedirs(dst_dirpath, exist_ok=True)
            for file_name in filenames:
                if re.match(file_filter_re, file_name) or file_name == 'requirements.txt':
                    src_path = os.path.join(dirpath, file_name)
                    dst_path = os.path.join(dst_dirpath, file_name)
                    shutil.copy(src_path, dst_path)
            

        src_dockerfile_path = os.path.join(current_dir, 'Dockerfile')
        dst_dockerfile_path = os.path.join(context_dir, 'Dockerfile')
        if os.path.exists(src_dockerfile_path):
            shutil.copy(src_dockerfile_path, dst_dockerfile_path)
        else:
            dockerfile_text = generate_dockerfile_text(context_dir, dst_dockerfile_path)
            with open(dst_dockerfile_path, 'w') as f:
                f.write(dockerfile_text)

        if builder is None:
            builder = get_default_image_builder()
        return builder.build(
            local_dir=context_dir,
            target_image=image_name,
            timeout=timeout,
        )
