# Copyright 2020-2022 The Kubeflow Authors
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

import dataclasses
import datetime
import enum
import json
from typing import Any, Dict, Union

import click
import kfp_server_api
import tabulate

KFP_TABLE_FORMAT = 'custom-simple'

tabulate._table_formats.update({  # type: ignore
    KFP_TABLE_FORMAT:
        tabulate.TableFormat(
            lineabove=None,
            linebelowheader=None,
            linebetweenrows=None,
            linebelow=None,
            headerrow=tabulate.DataRow('', '  ', ''),
            datarow=tabulate.DataRow('', '  ', ''),
            padding=0,
            with_header_hide=['lineabove', 'linebelow'])
})


@enum.unique
class OutputFormat(enum.Enum):
    """Enumerated class with the allowed output format constants."""
    table = 'table'
    json = 'json'


RUN_STORAGE_STATE_MAP = {
    kfp_server_api.ApiRunStorageState.AVAILABLE: 'Available',
    kfp_server_api.ApiRunStorageState.ARCHIVED: 'Archived',
}
EXPERIMENT_STORAGE_STATE_MAP = {
    kfp_server_api.ApiExperimentStorageState.AVAILABLE: 'Available',
    kfp_server_api.ApiExperimentStorageState.ARCHIVED: 'Archived',
    kfp_server_api.ApiExperimentStorageState.UNSPECIFIED: 'Unspecified',
}


def snake_to_header(string: str) -> str:
    """Converts a snake case string to a table header by replacing underscores
    with spaces and making uppercase.

    Args:
        string (str): The snake case string.

    Returns:
        str: The header.
    """
    return string.replace('_', ' ').upper()


@dataclasses.dataclass
class ExperimentData:
    id: str
    name: str
    created_at: str
    state: str


def transform_experiment(exp: kfp_server_api.ApiExperiment) -> Dict[str, Any]:
    return dataclasses.asdict(
        ExperimentData(
            id=exp.id,
            name=exp.name,
            created_at=exp.created_at.isoformat(),
            state=EXPERIMENT_STORAGE_STATE_MAP.get(
                exp.storage_state, EXPERIMENT_STORAGE_STATE_MAP[
                    kfp_server_api.ApiExperimentStorageState.AVAILABLE])))


@dataclasses.dataclass
class PipelineData:
    id: str
    name: str
    created_at: str
    default_version: str


def transform_pipeline(pipeline: kfp_server_api.ApiPipeline) -> Dict[str, Any]:
    default_version_id = pipeline.default_version.id if hasattr(
        pipeline,
        'default_version') and pipeline.default_version is not None and hasattr(
            pipeline.default_version, 'id') else None
    return dataclasses.asdict(
        PipelineData(
            id=pipeline.id,
            name=pipeline.name,
            created_at=pipeline.created_at.isoformat(),
            default_version=default_version_id))


@dataclasses.dataclass
class PipelineVersionData:
    id: str
    name: str
    created_at: str
    parent_id: str


def transform_pipeline_version(
        pipeline_version: kfp_server_api.ApiPipelineVersion) -> Dict[str, Any]:
    parent_id = next(
        rr for rr in pipeline_version.resource_references
        if rr.relationship == kfp_server_api.ApiRelationship.OWNER).key.id
    return dataclasses.asdict(
        PipelineVersionData(
            id=pipeline_version.id,
            name=pipeline_version.name,
            created_at=pipeline_version.created_at.isoformat(),
            parent_id=parent_id,
        ))


@dataclasses.dataclass
class RunData:
    id: str
    name: str
    created_at: str
    status: str
    state: str


def transform_run(
    run: Union[kfp_server_api.ApiRun, kfp_server_api.ApiRunDetail]
) -> Dict[str, Any]:
    return dataclasses.asdict((RunData(
        id=run.id,
        name=run.name,
        created_at=run.created_at.isoformat(),
        status=run.status,
        state=RUN_STORAGE_STATE_MAP.get(
            run.storage_state,
            RUN_STORAGE_STATE_MAP[kfp_server_api.ApiRunStorageState.AVAILABLE]))
                              ))


@dataclasses.dataclass
class JobData:
    id: str
    name: str
    created_at: str
    experiment_id: str
    status: str


def transform_job(recurring_run: kfp_server_api.ApiJob) -> Dict[str, Any]:
    experiment_id = next(
        rr for rr in recurring_run.resource_references
        if rr.key.type == kfp_server_api.ApiResourceType.EXPERIMENT).key.id
    return dataclasses.asdict(
        JobData(
            id=recurring_run.id,
            name=recurring_run.name,
            created_at=recurring_run.created_at.isoformat(),
            experiment_id=experiment_id,
            status=recurring_run.status))


@enum.unique
class ModelType(enum.Enum):
    """Enumerated class with the allowed output format constants."""
    EXPERIMENT = 'EXPERIMENT'
    PIPELINE = 'PIPELINE'
    PIPELINE_VERSION = 'PIPELINE_VERSION'
    RUN = 'RUN'
    JOB = 'JOB'


transformer_map = {
    ModelType.EXPERIMENT: transform_experiment,
    ModelType.PIPELINE: transform_pipeline,
    ModelType.PIPELINE_VERSION: transform_pipeline_version,
    ModelType.RUN: transform_run,
    ModelType.JOB: transform_job,
}

dataclass_map = {
    ModelType.EXPERIMENT: ExperimentData,
    ModelType.PIPELINE: PipelineData,
    ModelType.PIPELINE_VERSION: PipelineVersionData,
    ModelType.RUN: RunData,
    ModelType.JOB: JobData,
}


class DatetimeEncoder(json.JSONEncoder):
    """JSON encoder for serializing datetime objects."""

    def default(self, obj: Any) -> Any:
        if isinstance(obj, datetime.datetime):
            return obj.isoformat()
        return json.JSONEncoder.default(self, obj)


def print_output(resources: list, model_type: ModelType,
                 output_format: str) -> None:
    """Prints output in tabular or JSON format, using click.echo.

    Args:
        resources (list): List of same-type resources to print.
        output_format (str): One of 'table' or 'json'.

    Raises:
        NotImplementedError: If the output format is not one of 'table' or 'json'.
    """
    if isinstance(resources, list):
        single_resource = False
    else:
        resources = [resources]
        single_resource = True

    if output_format == OutputFormat.table.name:
        transformer = transformer_map[model_type]
        output_headers = dataclass_map[  # type: ignore
            model_type].__dataclass_fields__.keys()
        resources = [transformer(r) for r in resources]

        data = [list(resource.values()) for resource in resources]
        headers = [snake_to_header(header) for header in output_headers]
        click.echo(
            tabulate.tabulate(data, headers=headers, tablefmt='custom-simple'))

    elif output_format == OutputFormat.json.name:
        data = resources[0].to_dict() if single_resource else [
            resources.to_dict() for resources in resources
        ]
        click.echo(json.dumps(data, indent=2, cls=DatetimeEncoder), nl=False)
    else:
        raise NotImplementedError(f'Unknown output format: {output_format}.')


def print_deleted_text(resource_type: str, resource_id: str,
                       output_format: str) -> None:
    """Prints a standardized output for deletion actions, using click.echo.

    Args:
        resource_type (str): The type of resource (e.g. 'experiment') deleted.
        resource_id (str): The ID of the resource deleted.
        output_format (str): The format for the output (one of 'table' or 'json').

    Raises:
        NotImplementedError: If the output format is not one of 'table' or 'json'.
    """
    if output_format == OutputFormat.table.name:
        click.echo(f'{resource_type.capitalize()} {resource_id} deleted.')

    elif output_format == OutputFormat.json.name:
        click.echo(json.dumps(resource_id, indent=2), nl=False)

    else:
        raise NotImplementedError(f'Unknown output format: {output_format}.')
