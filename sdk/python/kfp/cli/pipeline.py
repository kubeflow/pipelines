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
# See the License for the specific language governing permissions and
# limitations under the License.

from .._client import Client
import sys
import subprocess
import pprint
import time
import json
import click

from tabulate import tabulate


@click.group()
def pipeline():
    """Upload pipelines"""
    pass


@pipeline.command()
@click.option('-f', '--package-file', type=click.Path(exists=True, dir_okay=False), help='Path of the pipeline package file.')
@click.option('-n', '--pipeline-name', help='Name of the Pipeline.')
@click.pass_context
def upload(ctx, package_file, pipeline_name):
    """Upload a pipeline package"""
    client = ctx.obj['client']
    response = client.upload_pipeline(pipeline_package_path=package_file, pipeline_name=pipeline_name)

    headers = ['pipeline id', 'name', 'created at']
    data = [[response.id, response.name, response.created_at.isoformat()]]
    print(tabulate(data, headers=headers, tablefmt='grid'))
