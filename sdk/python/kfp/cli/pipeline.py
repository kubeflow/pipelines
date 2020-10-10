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

import click
import logging
from tabulate import tabulate


@click.group()
def pipeline():
    """Manage pipeline resources"""
    pass


@pipeline.command()
@click.option(
    "-p",
    "--pipeline-name",
    help="Name of the pipeline."
)
@click.option(
    "-d",
    "--description",
    help="Description for the pipeline."
)
@click.argument("package-file")
@click.pass_context
def upload(ctx, pipeline_name, package_file, description=None):
    """Upload a KFP pipeline"""
    client = ctx.obj["client"]
    if not pipeline_name:
        pipeline_name = package_file.split(".")[0]

    pipeline = client.upload_pipeline(package_file, pipeline_name, description)
    logging.info("Pipeline {} has been submitted\n".format(pipeline.id))
    _display_pipeline(pipeline)


@pipeline.command()
@click.option(
    "-p",
    "--pipeline-id",
    help="ID of the pipeline",
    required=True
)
@click.option(
    "-v",
    "--pipeline-version",
    help="Name of the pipeline version",
    required=True
)
@click.argument("package-file")
@click.pass_context
def upload_version(ctx, package_file, pipeline_version, pipeline_id):
    """Upload a version of the KFP pipeline"""
    client = ctx.obj["client"]

    version = client.pipeline_uploads.upload_pipeline_version(
        package_file, name=pipeline_version, pipelineid=pipeline_id)
    logging.info(
        "The {} version of the pipeline {} has been submitted\n".format(
            pipeline_version, pipeline_id))
    _display_pipeline_version(version)


@pipeline.command()
@click.option(
    "-m",
    "--max-size",
    default=100,
    help="Max size of the listed pipelines."
)
@click.pass_context
def list(ctx, max_size):
    """List uploaded KFP pipelines"""
    client = ctx.obj["client"]

    response = client.list_pipelines(
        page_size=max_size,
        sort_by="created_at desc"
    )
    if response.pipelines:
        _print_pipelines(response.pipelines)
    else:
        logging.info("No pipelines found")


@pipeline.command()
@click.argument("pipeline-id")
@click.pass_context
def get(ctx, pipeline_id):
    """Get detailed information about an uploaded KFP pipeline"""
    client = ctx.obj["client"]

    pipeline = client.get_pipeline(pipeline_id)
    _display_pipeline(pipeline)


@pipeline.command()
@click.argument("pipeline-id")
@click.pass_context
def delete(ctx, pipeline_id):
    """Delete an uploaded KFP pipeline"""
    client = ctx.obj["client"]

    client.delete_pipeline(pipeline_id)
    print("{} is deleted".format(pipeline_id))


def _print_pipelines(pipelines):
    headers = ["Pipeline ID", "Name", "Uploaded at"]
    data = [[
        pipeline.id,
        pipeline.name,
        pipeline.created_at.isoformat()
    ] for pipeline in pipelines]
    print(tabulate(data, headers=headers, tablefmt="grid"))


def _display_pipeline(pipeline):
    # Print pipeline information
    print(tabulate([], headers=["Pipeline Details"]))
    table = [
        ["ID", pipeline.id],
        ["Name", pipeline.name],
        ["Description", pipeline.description],
        ["Uploaded at", pipeline.created_at.isoformat()],
    ]
    print(tabulate(table, tablefmt="plain"))

    # Print pipeline parameter details
    headers = ["Parameter Name", "Default Value"]
    data = [[param.name, param.value] for param in pipeline.parameters]
    print(tabulate(data, headers=headers, tablefmt="grid"))


def _display_pipeline_version(version):
    print(tabulate([], headers=["Pipeline Version Details"]))
    pipeline_id = version.resource_references[0].key.id
    table = [
        ["Pipeline ID", pipeline_id],
        ["Version Name", version.name],
        ["Uploaded at", version.created_at.isoformat()],
    ]
    print(tabulate(table, tablefmt="plain"))
