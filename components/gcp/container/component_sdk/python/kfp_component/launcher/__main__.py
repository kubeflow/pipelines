# Copyright 2018 Google LLC
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

import argparse
import fire
import importlib
import os
import sys
import logging
from .launcher import launch

def main():
    logging.basicConfig(level=logging.INFO)
    parser = argparse.ArgumentParser(
        prog='launcher',
        description='Launch a python module or file.')
    parser.add_argument('file_or_module', type=str,
        help='Either a python file path or a module name.')
    parser.add_argument(
        '--ui_metadata_path',
        type=str,
        default='/mlpipeline-ui-metadata.json',
        help='Path for the file where the mlpipeline-ui-metadata.json data '
             'should be written.')
    parser.add_argument('args', nargs=argparse.REMAINDER)
    args = parser.parse_args()

    if args.ui_metadata_path:
        os.environ['KFP_UI_METADATA_PATH'] = args.ui_metadata_path

    launch(args.file_or_module, args.args)

if __name__ == '__main__':
    main()