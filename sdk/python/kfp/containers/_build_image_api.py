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
    'build_image_from_env',
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


_default_package_url_template = 'https://pypi.org/pypi/{}/json/'


_package_info_cache = {}


def _python_package_exists(name: str, version : str = None) -> str:
    package_subdir = name + ('/' + version if version else '')
    package_url = _default_package_url_template.format(package_subdir)
    if package_url in _package_info_cache:
        return True
    
    response = requests.get(package_url)
    if response.ok:
        _package_info_cache[package_url] = response.json()
        return True

    return False


def generate_requirements_file(requirements_path: str):
    logging.info('Generating the requirements.txt file')
    import pkg_resources
    with open(requirements_path, 'w') as f:
        for dist in pkg_resources.working_set:
            req = dist.as_requirement()
            logging.info('Checking package: {}.'.format(req))

            # Checking the existense of the particular package version and not just the package project as some system-installed packages exist on PyPI, but the version is old.
            # ERROR: Could not find a version that satisfies the requirement python-apt===1.8.3- (from -r requirements.txt (line 195)) (from versions: 0.0.0, 0.7.8)
            # ERROR: No matching distribution found for python-apt===1.8.3- (from -r requirements.txt (line 195))                        
            if _python_package_exists(dist.project_name, dist.version):
                f.write(str(req) + '\n')
            else:
                if _python_package_exists(dist.project_name, version=None):
                    logging.warning('Package "{}" exists on PyPI, but version "{}" does not exist.'.format(dist.project_name, dist.version))
                else:
                    logging.warning('Package version "{}" does not exist on PyPI.'.format(req))


_container_work_dir = '/python_env'


def generate_dockerfile_text(context_dir: str, dockerfile_path: str):
    # Generating the Dockerfile
    logging.info('Generating the Dockerfile')

    requirements_rel_path = 'requirements.txt'
    requirements_path = os.path.join(context_dir, requirements_rel_path)
    requirements_file_exists = os.path.exists(requirements_path)

    # Creating requirements.txt from the list of installed packages if requirements.txt does not already exist.
    if not requirements_file_exists:
        logging.error('''requirements.txt file was not found''')
        generate_requirements_file(requirements_path)
        requirements_file_exists = True

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


def build_image_from_env(image_name : str = None, source_dir : str = None, file_filter_re : str = '.*\.py',  timeout : int = 1000, builder : ContainerBuilder = None):
    '''build_image_from_env builds and pushes a new container image that reconstructs the current python environment.
    Args:
        image_name: Optional. The image repo name where the new container image will be pushed. The name will be generated if not not set.
        source_dir: Optional. The directory that will be used as the environment source. The requirements.txt and python files inside the directory will be captured.
        timeout: Optional. The image building timeout in seconds.
        builder: Optional. An instance of ContainerBuilder or compatible class that will be used to build the image.
    '''
    current_dir = source_dir or os.getcwd()
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
