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

import os
import re
import sys
import tempfile
import unittest
from pathlib import Path
from typing import Callable

import mock

from kfp.containers import build_image_from_working_dir


class MockImageBuilder:
    def __init__(self, dockerfile_text_check : Callable[[str], None] = None, requirements_text_check : Callable[[str], None] = None, file_paths_check : Callable[[str], None] = None):
        self.dockerfile_text_check = dockerfile_text_check
        self.requirements_text_check = requirements_text_check
        self.file_paths_check = file_paths_check

    def build(self, local_dir = None, target_image = None, timeout = 1000):
        if self.dockerfile_text_check:
            actual_dockerfile_text = (Path(local_dir) / 'Dockerfile').read_text()
            self.dockerfile_text_check(actual_dockerfile_text)
        if self.requirements_text_check:
            actual_requirements_text = (Path(local_dir) / 'requirements.txt').read_text()
            self.requirements_text_check(actual_requirements_text)
        if self.file_paths_check:
            file_paths = set(os.path.relpath(os.path.join(dirpath, file_name), local_dir) for dirpath, dirnames, filenames in os.walk(local_dir) for file_name in filenames)
            self.file_paths_check(file_paths)
        return target_image


class BuildImageApiTests(unittest.TestCase):
    def test_build_image_from_working_dir(self):
        expected_dockerfile_text_re = '''
FROM python:3.6.5
WORKDIR /.*
COPY requirements.txt .
RUN python3 -m pip install -r requirements.txt
COPY . .
'''
        #mock_builder = 
        with tempfile.TemporaryDirectory() as context_dir:
            requirements_text = 'pandas==1.24'
            requirements_txt_relpath = Path('.') / 'requirements.txt'
            file1_py_relpath = Path('.') / 'lib' / 'file1.py'
            file1_sh_relpath = Path('.') / 'lib' / 'file1.sh'

            context_path = Path(context_dir)
            (context_path / requirements_txt_relpath).write_text(requirements_text)
            (context_path / file1_py_relpath).parent.mkdir(parents=True, exist_ok=True)
            (context_path / file1_py_relpath).write_text('#py file')
            (context_path / file1_sh_relpath).parent.mkdir(parents=True, exist_ok=True)
            (context_path / file1_sh_relpath).write_text('#sh file')
            expected_file_paths = {
                'Dockerfile',
                str(requirements_txt_relpath),
                str(file1_py_relpath),
            }
            def dockerfile_text_check(actual_dockerfile_text):
                self.assertRegex(actual_dockerfile_text.strip(), expected_dockerfile_text_re.strip())
            def requirements_text_check(actual_requirements_text):
                self.assertEqual(actual_requirements_text.strip(), requirements_text.strip())
            def file_paths_check(file_paths):
                self.assertEqual(file_paths, expected_file_paths)

            builder = MockImageBuilder(dockerfile_text_check, requirements_text_check, file_paths_check)
            result = build_image_from_working_dir(working_dir=context_dir, base_image='python:3.6.5', builder=builder)

if __name__ == '__main__':
    unittest.main()
