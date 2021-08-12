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

import argparse
import os
import sys
from . import custom_job_remote_runner


def _make_parent_dirs_and_return_path(file_path: str):
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    return file_path


def _parse_args(args):
    """Parse command line arguments.

    Args:
        args: A list of arguments.

    Returns:
        An argparse.Namespace class instance holding parsed args.
    """
    parser = argparse.ArgumentParser(
        prog='Vertex Pipelines service launcher', description=''
    )
    parser.add_argument(
        "--type",
        dest="type",
        type=str,
        required=True,
        default=argparse.SUPPRESS
    )
    parser.add_argument(
        "--gcp_project",
        dest="gcp_project",
        type=str,
        required=True,
        default=argparse.SUPPRESS
    )
    parser.add_argument(
        "--gcp_region",
        dest="gcp_region",
        type=str,
        required=True,
        default=argparse.SUPPRESS
    )
    parser.add_argument(
        "--payload",
        dest="payload",
        type=str,
        required=True,
        default=argparse.SUPPRESS
    )
    parser.add_argument(
        "--gcp_resources",
        dest="gcp_resources",
        type=_make_parent_dirs_and_return_path,
        required=True,
        default=argparse.SUPPRESS
    )
    parsed_args, _ = parser.parse_known_args(args)
    return vars(parsed_args)


def main(argv):
    """Main entry.

    expected input args are as follows:
    Project - Required. The project of which the resource will be launched.
    Region - Required. The region of which the resource will be launched.
    Type - Required. GCP launcher is a single container. This Enum will
        specify which resource to be launched.
    Request payload - Required. The full serialized json of the resource spec.
        Note this can contain the Pipeline Placeholders.
    gcp_resources placeholder output for returning job_id.

    Args:
        argv: A list of system arguments.
    """

    parsed_args = _parse_args(argv)

    if parsed_args['type'] == 'CustomJob':
        custom_job_remote_runner.create_custom_job(**parsed_args)


if __name__ == '__main__':
    main(sys.argv[1:])
