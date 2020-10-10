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

from .._client import Client
import sys
import subprocess
import pprint
import time
import json
import click

from tabulate import tabulate

@click.group()
def run():
    """manage run resources"""
    pass

@run.command()
@click.option('-e', '--experiment-id', help='Parent experiment ID of listed runs.')
@click.option('-m', '--max-size', default=100, help='Max size of the listed runs.')
@click.pass_context
def list(ctx, experiment_id, max_size):
    """list recent KFP runs"""
    client = ctx.obj['client']
    response = client.list_runs(experiment_id=experiment_id, page_size=max_size, sort_by='created_at desc')
    if response and response.runs:
        _print_runs(response.runs)
    else:
        print('No runs found.')

@run.command()
@click.option('-e', '--experiment-name', required=True, help='Experiment name of the run.')
@click.option('-r', '--run-name', help='Name of the run.')
@click.option('-f', '--package-file', type=click.Path(exists=True, dir_okay=False), help='Path of the pipeline package file.')
@click.option('-p', '--pipeline-id', help='ID of the pipeline template.')
@click.option('-w', '--watch', is_flag=True, default=False, help='Watch the run status until it finishes.')
@click.option('-v', '--version', help='ID of the pipeline version.')
@click.argument('args', nargs=-1)
@click.pass_context
def submit(ctx, experiment_name, run_name, package_file, pipeline_id, watch, version, args):
    """submit a KFP run"""
    client = ctx.obj['client']
    namespace = ctx.obj['namespace']
    if not run_name:
        run_name = experiment_name

    if not package_file and not pipeline_id:
        print('You must provide one of [package_file, pipeline_id].')
        sys.exit(1)

    arg_dict = dict(arg.split('=') for arg in args)
    experiment = client.create_experiment(experiment_name)
    run = client.run_pipeline(experiment.id, run_name, package_file, arg_dict, pipeline_id, version_id=version)
    print('Run {} is submitted'.format(run.id))
    _display_run(client, namespace, run.id, watch)

@run.command()
@click.option('-w', '--watch', is_flag=True, default=False, help='Watch the run status until it finishes.')
@click.argument('run-id')
@click.pass_context
def get(ctx, watch, run_id):
    """display the details of a KFP run"""
    client = ctx.obj['client']
    namespace = ctx.obj['namespace']
    _display_run(client, namespace, run_id, watch)

def _display_run(client, namespace, run_id, watch):
    run = client.get_run(run_id).run
    _print_runs([run])
    if not watch:
        return
    argo_workflow_name = None
    while True:
        time.sleep(1)
        run_detail = client.get_run(run_id)
        run = run_detail.run
        if run_detail.pipeline_runtime and run_detail.pipeline_runtime.workflow_manifest:
            manifest = json.loads(run_detail.pipeline_runtime.workflow_manifest)
            if manifest['metadata'] and manifest['metadata']['name']:
                argo_workflow_name = manifest['metadata']['name']
                break
        if run_detail.run.status in ['Succeeded', 'Skipped', 'Failed', 'Error']:
            print('Run is finished with status {}.'.format(run_detail.run.status))
            return
    if argo_workflow_name:
        subprocess.run(['argo', 'watch', argo_workflow_name, '-n', namespace])
        _print_runs([run])

def _print_runs(runs):
    headers = ['run id', 'name', 'status', 'created at']
    data = [[run.id, run.name, run.status, run.created_at.isoformat()] for run in runs]
    print(tabulate(data, headers=headers, tablefmt='grid'))
