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

from .output import print_output, OutputFormat


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
    output_format = ctx.obj["output"]
    if not pipeline_name:
        pipeline_name = package_file.split(".")[0]

    pipeline = client.upload_pipeline(package_file, pipeline_name, description)
    _display_pipeline(pipeline, output_format)


@pipeline.command()
@click.option(
    "-p",
    "--pipeline-id",
    help="ID of the pipeline",
    required=False
)
@click.option(
    "-n",
    "--pipeline-name",
    help="Name of pipeline",
    required=False
)
@click.option(
    "-v",
    "--pipeline-version",
    help="Name of the pipeline version",
    required=True
)
@click.argument("package-file")
@click.pass_context
def upload_version(ctx, package_file, pipeline_version, pipeline_id=None, pipeline_name=None):
    """Upload a version of the KFP pipeline"""
    client = ctx.obj["client"]
    output_format = ctx.obj["output"]
    if bool(pipeline_id) == bool(pipeline_name):
        raise ValueError("Need to suppy 'pipeline-name' or 'pipeline-id'")
    if pipeline_name is not None:
        pipeline_id = client.get_pipeline_id(name=pipeline_name)
        if pipeline_id is None:
            raise ValueError("Can't find a pipeline with name: %s" % pipeline_name)
    version = client.pipeline_uploads.upload_pipeline_version(
        package_file, name=pipeline_version, pipelineid=pipeline_id)
    _display_pipeline_version(version, output_format)


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
    output_format = ctx.obj["output"]

    response = client.list_pipelines(
        page_size=max_size,
        sort_by="created_at desc"
    )
    if response.pipelines:
        _print_pipelines(response.pipelines, output_format)
    else:
        logging.info("No pipelines found")


@pipeline.command()
@click.argument("pipeline-id")
@click.option(
    "-m",
    "--max-size",
    default=10,
    help="Max size of the listed pipelines."
)
@click.pass_context
def list_versions(ctx, pipeline_id, max_size):
    """List versions of an uploaded KFP pipeline"""
    client = ctx.obj["client"]
    output_format = ctx.obj["output"]

    response = client.list_pipeline_versions(
        pipeline_id=pipeline_id,
        page_size=max_size,
        sort_by="created_at desc"
    )
    if response.versions:
        _print_pipeline_versions(response.versions, output_format)
    else:
        logging.info("No pipeline or version found")


@pipeline.command()
@click.argument("pipeline-id")
@click.pass_context
def get(ctx, pipeline_id):
    """Get detailed information about an uploaded KFP pipeline"""
    client = ctx.obj["client"]
    output_format = ctx.obj["output"]

    pipeline = client.get_pipeline(pipeline_id)
    _display_pipeline(pipeline, output_format)


@pipeline.command()
@click.argument("pipeline-id")
@click.pass_context
def delete(ctx, pipeline_id):
    """Delete an uploaded KFP pipeline"""
    client = ctx.obj["client"]

    client.delete_pipeline(pipeline_id)
    print("{} is deleted".format(pipeline_id))


def _print_pipelines(pipelines, output_format):
    headers = ["Pipeline ID", "Name", "Uploaded at"]
    data = [[
        pipeline.id,
        pipeline.name,
        pipeline.created_at.isoformat()
    ] for pipeline in pipelines]
    print_output(data, headers, output_format, table_format="grid")


def _print_pipeline_versions(versions, output_format):
    headers = ["Version ID", "Version name", "Uploaded at"]
    data = [[
        version.id,
        version.name,
        version.created_at.isoformat()
    ] for version in versions]
    print_output(data, headers, output_format, table_format="grid")


def _display_pipeline(pipeline, output_format):
    # Pipeline information
    table = [
        ["ID", pipeline.id],
        ["Name", pipeline.name],
        ["Description", pipeline.description],
        ["Uploaded at", pipeline.created_at.isoformat()],
    ]

    # Pipeline parameter details
    headers = ["Parameter Name", "Default Value"]
    data = [[param.name, param.value] for param in pipeline.parameters]

    if output_format == OutputFormat.table.name:
        print_output([], ["Pipeline Details"], output_format)
        print_output(table, [], output_format, table_format="plain")
        print_output(data, headers, output_format, table_format="grid")
    elif output_format == OutputFormat.json.name:
        output = dict()
        output["Pipeline Details"] = dict(table)
        params = []
        for item in data:
            params.append(dict(zip(headers, item)))
        output["Pipeline Parameters"] = params
        print_output(output, [], output_format)


def _display_pipeline_version(version, output_format):
    pipeline_id = version.resource_references[0].key.id
    table = [
        ["Pipeline ID", pipeline_id],
        ["Version Name", version.name],
        ["Uploaded at", version.created_at.isoformat()],
    ]

    if output_format == OutputFormat.table.name:
        print_output([], ["Pipeline Version Details"], output_format)
        print_output(table, [], output_format, table_format="plain")
    elif output_format == OutputFormat.json.name:
        print_output(dict(table), [], output_format)
