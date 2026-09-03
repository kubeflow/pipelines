// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as express from 'express';
import { Response } from 'express-serve-static-core';
import * as fs from 'fs';
import * as _path from 'path';
import {
  ApiExperiment,
  ApiExperimentStorageState,
  ApiListExperimentsResponse,
} from '../src/apis/experiment';
import { ApiFilter, PredicateOp } from '../src/apis/filter';
import { ApiJob, ApiListJobsResponse } from '../src/apis/job';
import {
  ApiListPipelinesResponse,
  ApiListPipelineVersionsResponse,
  ApiPipeline,
  ApiPipelineVersion,
} from '../src/apis/pipeline';
import { ApiListRunsResponse, ApiResourceType, ApiRun, ApiRunStorageState } from '../src/apis/run';
import {
  ArtifactArtifactType,
  V2beta1Artifact,
  V2beta1ArtifactTask,
  V2beta1IOType,
} from '../src/apisv2beta1/artifact';
import {
  V2beta1Experiment,
  V2beta1ExperimentStorageState,
  V2beta1ListExperimentsResponse,
} from '../src/apisv2beta1/experiment';
import { V2beta1Filter, V2beta1PredicateOperation } from '../src/apisv2beta1/filter';
import {
  V2beta1ListPipelineVersionsResponse,
  V2beta1ListPipelinesResponse,
  V2beta1Pipeline,
  V2beta1PipelineVersion,
} from '../src/apisv2beta1/pipeline';
import {
  V2beta1ListRecurringRunsResponse,
  V2beta1RecurringRun,
  V2beta1RecurringRunStatus,
  V2beta1Trigger,
} from '../src/apisv2beta1/recurringrun';
import {
  PipelineTaskTaskPodType,
  PipelineTaskTaskState,
  PipelineTaskTaskType,
  V2beta1ListRunsResponse,
  V2beta1PipelineTask,
  V2beta1Run,
  V2beta1RunStorageState,
  V2beta1RuntimeState,
} from '../src/apisv2beta1/run';
import {
  ExperimentSortKeys,
  JobSortKeys,
  PipelineSortKeys,
  PipelineVersionSortKeys,
  RunSortKeys,
} from '../src/lib/Apis';
import RunUtils from '../src/lib/RunUtils';
import {
  data as fixedData,
  namedPipelines,
  PIPELINE_VERSIONS_LIST_FULL,
  PIPELINE_VERSIONS_LIST_MAP,
  v2PipelineSpecMap,
} from './fixed-data';
import helloWorldRuntime from './data/v1/runtime/integration-test-runtime';
import registerTensorboardProxy from './tensorboard-proxy';

const rocMetadataJsonPath = './eval-output/metadata.json';
const rocMetadataJsonPath2 = './eval-output/metadata2.json';
const rocDataPath = './eval-output/roc.csv';
const rocDataPath2 = './eval-output/roc2.csv';
const tableDataPath = './eval-output/table.csv';

const confusionMatrixMetadataJsonPath = './model-output/metadata.json';
const confusionMatrixPath = './model-output/confusion_matrix.csv';
const helloWorldHtmlPath = './model-output/hello-world.html';
const helloWorldBigHtmlPath = './model-output/hello-world-big.html';

const v1beta1Prefix = '/apis/v1beta1';
const v2beta1Prefix = '/apis/v2beta1';
const mockNativeRunId = 'e0115ac1-0479-4194-a22d-01e65e09a32b';

const mockV2Artifacts: V2beta1Artifact[] = [
  {
    artifact_id: 'mock-artifact-1',
    created_at: new Date('2026-01-01T00:00:00.000Z'),
    description: 'Representative native artifact for local frontend development.',
    name: 'mock-dataset',
    namespace: 'kubeflow-user-example-com',
    type: ArtifactArtifactType.Dataset,
    uri: 's3://mlpipeline/private-artifacts/kubeflow-user-example-com/mock-run/mock-dataset',
  },
];
const mockV2Tasks: V2beta1PipelineTask[] = [
  {
    child_tasks: [
      { name: 'chicago-taxi-trips-dataset', task_id: 'mock-task-producer' },
      { name: 'convert-csv-to-apache-parquet', task_id: 'mock-task-consumer' },
    ],
    create_time: new Date('2026-01-01T00:00:00.000Z'),
    display_name: 'xgboost-sample-pipeline',
    end_time: new Date('2026-01-01T00:03:00.000Z'),
    name: 'root',
    run_id: mockNativeRunId,
    scope_path: 'root',
    state: PipelineTaskTaskState.SUCCEEDED,
    task_id: 'mock-task-root',
    type: PipelineTaskTaskType.ROOT,
  },
  {
    create_time: new Date('2026-01-01T00:00:10.000Z'),
    display_name: 'Chicago taxi trips dataset',
    end_time: new Date('2026-01-01T00:01:00.000Z'),
    name: 'chicago-taxi-trips-dataset',
    outputs: {
      artifacts: [
        {
          artifact_key: 'table',
          artifacts: mockV2Artifacts,
          type: V2beta1IOType.OUTPUT,
        },
      ],
    },
    parent_task_id: 'mock-task-root',
    pods: [
      {
        name: 'mock-chicago-taxi-trips-dataset-executor',
        type: PipelineTaskTaskPodType.EXECUTOR,
        uid: 'mock-producer-pod-uid',
      },
    ],
    run_id: mockNativeRunId,
    scope_path: 'root.chicago-taxi-trips-dataset',
    state: PipelineTaskTaskState.SUCCEEDED,
    task_id: 'mock-task-producer',
    type: PipelineTaskTaskType.RUNTIME,
  },
  {
    create_time: new Date('2026-01-01T00:01:05.000Z'),
    display_name: 'Convert CSV to Apache Parquet',
    end_time: new Date('2026-01-01T00:02:00.000Z'),
    inputs: {
      artifacts: [
        {
          artifact_key: 'data',
          artifacts: mockV2Artifacts,
          producer: { task_name: 'chicago-taxi-trips-dataset' },
          type: V2beta1IOType.TASK_OUTPUT_INPUT,
        },
      ],
    },
    name: 'convert-csv-to-apache-parquet',
    parent_task_id: 'mock-task-root',
    pods: [
      {
        name: 'mock-convert-csv-to-apache-parquet-driver',
        type: PipelineTaskTaskPodType.DRIVER,
        uid: 'mock-consumer-driver-pod-uid',
      },
      {
        name: 'mock-convert-csv-to-apache-parquet-executor',
        type: PipelineTaskTaskPodType.EXECUTOR,
        uid: 'mock-consumer-executor-pod-uid',
      },
    ],
    run_id: mockNativeRunId,
    scope_path: 'root.convert-csv-to-apache-parquet',
    state: PipelineTaskTaskState.SUCCEEDED,
    task_id: 'mock-task-consumer',
    type: PipelineTaskTaskType.RUNTIME,
  },
];
const mockV2ArtifactTasks: V2beta1ArtifactTask[] = [
  {
    artifact_id: 'mock-artifact-1',
    id: 'mock-artifact-task-output',
    key: 'table',
    run_id: mockNativeRunId,
    task_id: 'mock-task-producer',
    type: V2beta1IOType.OUTPUT,
  },
  {
    artifact_id: 'mock-artifact-1',
    id: 'mock-artifact-task-input',
    key: 'data',
    producer: { task_name: 'chicago-taxi-trips-dataset' },
    run_id: mockNativeRunId,
    task_id: 'mock-task-consumer',
    type: V2beta1IOType.TASK_OUTPUT_INPUT,
  },
];

let tensorboardPod = '';

// This is a copy of the BaseResource defined within src/pages/ResourceSelector
interface BaseResource {
  id?: string;
  created_at?: Date;
  description?: string;
  name?: string;
  error?: string;
}

interface V2FilterableResource {
  created_at?: Date;
  display_name?: string;
  name?: string;
  storage_state?: string;
}

function getQueryString(queryParam: unknown): string | undefined {
  if (typeof queryParam === 'string') {
    return queryParam;
  }
  if (Array.isArray(queryParam)) {
    return queryParam.find((value): value is string => typeof value === 'string');
  }
  return undefined;
}

function getQueryNumber(queryParam: unknown): number | undefined {
  const queryString = getQueryString(queryParam);
  if (queryString === undefined || queryString === '') {
    return undefined;
  }
  const queryNumber = Number(queryString);
  return Number.isNaN(queryNumber) ? undefined : queryNumber;
}

function getDecodedQueryString(queryParam: unknown): string | undefined {
  const queryString = getQueryString(queryParam);
  if (queryString === undefined) {
    return undefined;
  }
  try {
    return decodeURIComponent(queryString);
  } catch {
    return undefined;
  }
}

function getRequiredDecodedQueryString(
  res: Response,
  queryParam: unknown,
  queryParamName: string,
): string | undefined {
  const queryString = getQueryString(queryParam);
  if (queryString === undefined || queryString === '') {
    res.status(400).send(`${queryParamName} argument is required`);
    return undefined;
  }

  let decodedQueryString: string;
  try {
    decodedQueryString = decodeURIComponent(queryString);
  } catch {
    res.status(400).send(`${queryParamName} argument is invalid`);
    return undefined;
  }

  if (decodedQueryString === '') {
    res.status(400).send(`${queryParamName} argument is required`);
    return undefined;
  }

  return decodedQueryString;
}

function getMockBackendFilePath(relativePath: string): string {
  return _path.resolve(process.cwd(), 'mock-backend', relativePath);
}

function sendMockBackendFile(res: Response, relativePath: string): void {
  res.send(fs.readFileSync(getMockBackendFilePath(relativePath), 'utf-8'));
}

function getSortKeyAndOrder(
  defaultSortKey: string,
  queryParam?: string,
): { desc: boolean; key: string } {
  let key = defaultSortKey;
  let desc = false;

  if (queryParam) {
    const keyParts = queryParam.split(' ');
    key = keyParts[0];

    // Check that the key is properly formatted.
    if (
      keyParts.length > 2 ||
      (keyParts.length === 2 && keyParts[1] !== 'asc' && keyParts[1] !== 'desc')
    ) {
      throw new Error(`Invalid sort string: ${queryParam}`);
    }

    desc = keyParts.length === 2 && keyParts[1] === 'desc';
  }
  return { desc, key };
}

function getExperimentId(runOrJob: ApiRun | ApiJob): string | undefined {
  return RunUtils.getAllExperimentReferences(runOrJob).find((ref) => ref.key?.id)?.key?.id;
}

function getV2StorageState(storageState?: string): V2beta1RunStorageState {
  return storageState === ApiRunStorageState.STORAGESTATE_ARCHIVED
    ? V2beta1RunStorageState.ARCHIVED
    : V2beta1RunStorageState.AVAILABLE;
}

function getV2ExperimentStorageState(storageState?: string): V2beta1ExperimentStorageState {
  return storageState === ApiExperimentStorageState.STORAGESTATE_ARCHIVED
    ? V2beta1ExperimentStorageState.ARCHIVED
    : V2beta1ExperimentStorageState.AVAILABLE;
}

function getV2RuntimeState(status?: string): V2beta1RuntimeState {
  switch ((status || '').toLowerCase()) {
    case 'running':
      return V2beta1RuntimeState.RUNNING;
    case 'failed':
      return V2beta1RuntimeState.FAILED;
    case 'pending':
      return V2beta1RuntimeState.PENDING;
    case 'skipped':
      return V2beta1RuntimeState.SKIPPED;
    case 'canceled':
      return V2beta1RuntimeState.CANCELED;
    case 'canceling':
      return V2beta1RuntimeState.CANCELING;
    case 'paused':
      return V2beta1RuntimeState.PAUSED;
    case 'succeeded':
      return V2beta1RuntimeState.SUCCEEDED;
    default:
      return V2beta1RuntimeState.RUNTIME_STATE_UNSPECIFIED;
  }
}

function getV2ResourceValue(resource: V2FilterableResource, key: string): unknown {
  if (key === 'name') {
    return resource.name || resource.display_name;
  }
  if (key === 'display_name') {
    return resource.display_name || resource.name;
  }
  return (resource as Record<string, unknown>)[key];
}

function getComparableValue(value: unknown): number | string {
  if (value instanceof Date) {
    return value.getTime();
  }
  if (typeof value === 'number') {
    return value;
  }
  return String(value || '');
}

type V1SortableResource = ApiJob | ApiExperiment | ApiRun | ApiPipeline;

/**
 * The *SortKeys enums advertise the sort keys the real backend accepts, and a
 * few of those are not stored on the v1 models. Derive those here so sorting by
 * them behaves like the backend instead of silently comparing undefined.
 *
 * `lastRunCreatedAt` is the lookup built by buildLastRunCreatedAtMap, threaded
 * in so it is built once per request rather than per comparison.
 */
function getV1ResourceValue(
  resource: V1SortableResource,
  key: string,
  lastRunCreatedAt: () => Map<string, Date>,
): unknown {
  const record = resource as Record<string, unknown>;
  if (key === 'display_name') {
    // PipelineSortKeys/PipelineVersionSortKeys advertise display_name, but the
    // v1 models only carry name.
    return record.display_name ?? record.name;
  }
  if (key === 'last_run_created_at') {
    // ExperimentSortKeys advertises last_run_created_at; it is a property of the
    // experiment's runs rather than of the experiment itself.
    const id = record.id;
    return typeof id === 'string' ? lastRunCreatedAt().get(id) : undefined;
  }
  return record[key];
}

/** Latest run creation time per experiment, in one pass over fixedData.runs. */
function buildLastRunCreatedAtMap(): Map<string, Date> {
  const latest = new Map<string, Date>();
  for (const runDetail of fixedData.runs) {
    const run = runDetail.run;
    if (!run || !run.created_at) {
      continue;
    }
    for (const reference of RunUtils.getAllExperimentReferences(run)) {
      const experimentId = reference.key?.id;
      if (!experimentId) {
        continue;
      }
      const current = latest.get(experimentId);
      if (!current || run.created_at > current) {
        latest.set(experimentId, run.created_at);
      }
    }
  }
  return latest;
}

/**
 * Sorts a copy of `resources` so repeated requests do not reorder the shared
 * fixture data in place. The value getter is the only part that varies between
 * the v1 and v2 models, so the comparison logic lives here only.
 */
function sortResources<T>(
  resources: T[],
  defaultSortKey: string,
  queryParam: string | undefined,
  getValue: (resource: T, key: string) => unknown,
): T[] {
  const { desc, key } = getSortKeyAndOrder(defaultSortKey, queryParam);
  return resources.slice().sort((a, b) => {
    const aValue = getComparableValue(getValue(a, key));
    const bValue = getComparableValue(getValue(b, key));
    let result = 0;
    if (aValue < bValue) {
      result = -1;
    }
    if (aValue > bValue) {
      result = 1;
    }
    return result * (desc ? -1 : 1);
  });
}

function sortV1Resources<T extends V1SortableResource>(
  resources: T[],
  defaultSortKey: string,
  queryParam?: string,
): T[] {
  // Built at most once per call, and only when the sort key actually needs it.
  let lastRunCreatedAt: Map<string, Date> | undefined;
  const getLastRunCreatedAt = () => {
    if (!lastRunCreatedAt) {
      lastRunCreatedAt = buildLastRunCreatedAtMap();
    }
    return lastRunCreatedAt;
  };
  return sortResources(resources, defaultSortKey, queryParam, (resource, key) =>
    getV1ResourceValue(resource, key, getLastRunCreatedAt),
  );
}

function sortV2Resources<T extends V2FilterableResource>(
  resources: T[],
  defaultSortKey: string,
  queryParam?: string,
): T[] {
  return sortResources(resources, defaultSortKey, queryParam, getV2ResourceValue);
}

function filterV2Resources<T extends V2FilterableResource>(
  resources: T[],
  filterString?: string,
): T[] {
  if (!filterString) {
    return resources;
  }
  const filter: V2beta1Filter = JSON.parse(decodeURIComponent(filterString));
  return ((filter && filter.predicates) || []).reduce((filteredResources, predicate) => {
    const key = predicate.key || '';
    const stringValue = predicate.string_value || '';
    switch (predicate.operation) {
      case V2beta1PredicateOperation.EQUALS:
        return filteredResources.filter(
          (resource) => String(getV2ResourceValue(resource, key) || '') === stringValue,
        );
      case V2beta1PredicateOperation.NOT_EQUALS:
        return filteredResources.filter(
          (resource) => String(getV2ResourceValue(resource, key) || '') !== stringValue,
        );
      case V2beta1PredicateOperation.IS_SUBSTRING:
        return filteredResources.filter((resource) =>
          String(getV2ResourceValue(resource, key) || '')
            .toLocaleLowerCase()
            .includes(stringValue.toLocaleLowerCase()),
        );
      default:
        throw new Error(`Operation: ${predicate.operation} is not yet supported by the mock API`);
    }
  }, resources);
}

function getPage<T>(
  resources: T[],
  pageTokenQueryParam: unknown,
  pageSizeQueryParam: unknown,
): { nextPageToken: string; page: T[] } {
  const start = getQueryNumber(pageTokenQueryParam) || 0;
  const end = start + (getQueryNumber(pageSizeQueryParam) || 20);
  return {
    nextPageToken: end < resources.length ? end + '' : '',
    page: resources.slice(start, end),
  };
}

function toV2Experiment(experiment: ApiExperiment): V2beta1Experiment {
  return {
    created_at: experiment.created_at,
    description: experiment.description,
    display_name: experiment.name,
    experiment_id: experiment.id,
    storage_state: getV2ExperimentStorageState(experiment.storage_state),
  };
}

function toV2Pipeline(pipeline: ApiPipeline): V2beta1Pipeline {
  return {
    created_at: pipeline.created_at,
    description: pipeline.description,
    display_name: (pipeline as { display_name?: string }).display_name || pipeline.name,
    name: pipeline.name,
    pipeline_id: pipeline.id,
  };
}

function toV2PipelineVersion(
  version: ApiPipelineVersion,
  pipelineId?: string,
): V2beta1PipelineVersion {
  return {
    code_source_url: version.code_source_url,
    created_at: version.created_at,
    description: version.description,
    display_name: (version as { display_name?: string }).display_name || version.name,
    name: version.name,
    pipeline_id: pipelineId || version.id,
    pipeline_version_id: version.id,
  };
}

function toV2Run(run: ApiRun): V2beta1Run {
  const pipelineId = RunUtils.getPipelineId(run);
  const pipelineVersionId = RunUtils.getPipelineVersionId(run);
  return {
    created_at: run.created_at,
    description: run.description,
    display_name: run.name,
    experiment_id: getExperimentId(run),
    finished_at: run.finished_at,
    pipeline_spec: run.pipeline_spec,
    pipeline_version_reference:
      pipelineId && pipelineVersionId
        ? {
            pipeline_id: pipelineId,
            pipeline_version_id: pipelineVersionId,
          }
        : undefined,
    run_id: run.id,
    scheduled_at: run.scheduled_at,
    state: getV2RuntimeState(run.status),
    storage_state: getV2StorageState(run.storage_state),
  };
}

function toV2RecurringRun(job: ApiJob): V2beta1RecurringRun {
  return {
    created_at: job.created_at,
    description: job.description,
    display_name: job.name,
    experiment_id: getExperimentId(job),
    max_concurrency: job.max_concurrency,
    pipeline_spec: job.pipeline_spec,
    recurring_run_id: job.id,
    status: job.enabled ? V2beta1RecurringRunStatus.ENABLED : V2beta1RecurringRunStatus.DISABLED,
    trigger: job.trigger as unknown as V2beta1Trigger,
    updated_at: job.updated_at,
  };
}

function getV2PipelineVersions(pipelineId: string): V2beta1PipelineVersion[] {
  const versions = PIPELINE_VERSIONS_LIST_MAP.get(pipelineId);
  if (versions && versions.length > 0) {
    return versions.map((version) => toV2PipelineVersion(version, pipelineId));
  }

  const pipeline = fixedData.pipelines.find((candidate) => candidate.id === pipelineId);
  if (!pipeline) {
    return [];
  }

  return [toV2PipelineVersion(pipeline.default_version || pipeline, pipelineId)];
}

function getV2PipelineVersion(
  pipelineId: string,
  pipelineVersionId: string,
): V2beta1PipelineVersion | undefined {
  return (
    getV2PipelineVersions(pipelineId).find(
      (version) => version.pipeline_version_id === pipelineVersionId,
    ) ||
    PIPELINE_VERSIONS_LIST_FULL.map((version) => toV2PipelineVersion(version, pipelineId)).find(
      (version) => version.pipeline_version_id === pipelineVersionId,
    )
  );
}

// tslint:disable-next-line:no-default-export
export default (app: express.Application) => {
  app.use((req, _, next) => {
    // tslint:disable-next-line:no-console
    console.info(req.method + ' ' + req.originalUrl);
    next();
  });

  registerTensorboardProxy(app as any);

  app.set('json spaces', 2);
  app.use(express.json());

  app.get(v1beta1Prefix + '/healthz', (_, res) => {
    res.header('Content-Type', 'application/json');
    res.send({
      apiServerCommitHash: 'd3c4add0a95e930c70a330466d0923827784eb9a',
      apiServerReady: true,
      buildDate: 'Wed Jan 9 19:40:24 UTC 2019',
      frontendCommitHash: '8efb2fcff9f666ba5b101647e909dc9c6889cecb',
    });
  });

  app.get(v2beta1Prefix + '/healthz', (_, res) => {
    res.header('Content-Type', 'application/json');
    res.send({
      apiServerCommitHash: 'd3c4add0a95e930c70a330466d0923827784eb9a',
      apiServerMultiUser: false,
      apiServerReady: true,
      buildDate: 'Wed Jan 9 19:40:24 UTC 2019',
      frontendCommitHash: '8efb2fcff9f666ba5b101647e909dc9c6889cecb',
      pipelineStore: 'database',
    });
  });

  app.get(v2beta1Prefix + '/experiments', (req, res) => {
    res.header('Content-Type', 'application/json');
    const experiments = sortV2Resources(
      filterV2Resources(
        fixedData.experiments.map(toV2Experiment),
        getQueryString(req.query.filter),
      ),
      ExperimentSortKeys.NAME,
      getQueryString(req.query.sort_by),
    );
    const page = getPage(experiments, req.query.page_token, req.query.page_size);
    const response: V2beta1ListExperimentsResponse = {
      experiments: page.page,
      next_page_token: page.nextPageToken,
      total_size: experiments.length,
    };

    res.json(response);
  });

  app.get(v2beta1Prefix + '/experiments/:eid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const experiment = fixedData.experiments.find((exp) => exp.id === req.params.eid);
    if (!experiment) {
      res.status(404).send(`No experiment was found with ID: ${req.params.eid}`);
      return;
    }
    res.json(toV2Experiment(experiment));
  });

  app.get(v2beta1Prefix + '/pipelines', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pipelines = sortV2Resources(
      filterV2Resources(fixedData.pipelines.map(toV2Pipeline), getQueryString(req.query.filter)),
      PipelineSortKeys.CREATED_AT,
      getQueryString(req.query.sort_by),
    );
    const page = getPage(pipelines, req.query.page_token, req.query.page_size);
    const response: V2beta1ListPipelinesResponse = {
      next_page_token: page.nextPageToken,
      pipelines: page.page,
      total_size: pipelines.length,
    };

    res.json(response);
  });

  app.get(v2beta1Prefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pipeline = fixedData.pipelines.find((candidate) => candidate.id === req.params.pid);
    if (!pipeline) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    res.json(toV2Pipeline(pipeline));
  });

  app.get(v2beta1Prefix + '/pipelines/:pid/versions', (req, res) => {
    res.header('Content-Type', 'application/json');
    const versions = sortV2Resources(
      filterV2Resources(getV2PipelineVersions(req.params.pid), getQueryString(req.query.filter)),
      PipelineVersionSortKeys.CREATED_AT,
      getQueryString(req.query.sort_by),
    );
    const page = getPage(versions, req.query.page_token, req.query.page_size);
    const response: V2beta1ListPipelineVersionsResponse = {
      next_page_token: page.nextPageToken,
      pipeline_versions: page.page,
      total_size: versions.length,
    };

    res.json(response);
  });

  app.get(v2beta1Prefix + '/pipelines/:pid/versions/:pvid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const version = getV2PipelineVersion(req.params.pid, req.params.pvid);
    if (!version) {
      res.status(404).send(`No pipeline version was found with ID: ${req.params.pvid}`);
      return;
    }
    res.json(version);
  });

  app.get(v2beta1Prefix + '/runs', (req, res) => {
    res.header('Content-Type', 'application/json');
    let runs = fixedData.runs.map((runDetail) => toV2Run(runDetail.run!));
    const experimentId = getQueryString(req.query.experiment_id);
    if (experimentId) {
      runs = runs.filter((run) => run.experiment_id === experimentId);
    }
    runs = sortV2Resources(
      filterV2Resources(runs, getQueryString(req.query.filter)),
      RunSortKeys.CREATED_AT,
      getQueryString(req.query.sort_by),
    );
    const page = getPage(runs, req.query.page_token, req.query.page_size);
    const response: V2beta1ListRunsResponse = {
      next_page_token: page.nextPageToken,
      runs: page.page,
      total_size: runs.length,
    };

    res.json(response);
  });

  app.get(v2beta1Prefix + '/runs/:rid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const run = fixedData.runs.find((runDetail) => runDetail.run!.id === req.params.rid);
    if (!run) {
      res.status(404).send('Cannot find a run with id: ' + req.params.rid);
      return;
    }
    res.json(toV2Run(run.run!));
  });

  app.get(v2beta1Prefix + '/runs/:rid/tasks', (req, res) => {
    res.json({ tasks: req.params.rid === mockNativeRunId ? mockV2Tasks : [] });
  });

  app.get(v2beta1Prefix + '/artifacts', (_req, res) => {
    res.json({ artifacts: mockV2Artifacts, total_size: mockV2Artifacts.length });
  });

  app.get(v2beta1Prefix + '/artifacts/:artifactId', (req, res) => {
    const artifact = mockV2Artifacts.find(
      (candidate) => candidate.artifact_id === req.params.artifactId,
    );
    if (!artifact) {
      res.status(404).send(`No artifact was found with ID: ${req.params.artifactId}`);
      return;
    }
    res.json(artifact);
  });

  app.get(v2beta1Prefix + '/artifact_tasks', (_req, res) => {
    res.json({ artifact_tasks: mockV2ArtifactTasks });
  });

  app.get(v2beta1Prefix + '/recurringruns', (req, res) => {
    res.header('Content-Type', 'application/json');
    let recurringRuns = fixedData.jobs.map(toV2RecurringRun);
    const experimentId = getQueryString(req.query.experiment_id);
    if (experimentId) {
      recurringRuns = recurringRuns.filter(
        (recurringRun) => recurringRun.experiment_id === experimentId,
      );
    }
    recurringRuns = sortV2Resources(
      filterV2Resources(recurringRuns, getQueryString(req.query.filter)),
      JobSortKeys.CREATED_AT,
      getQueryString(req.query.sort_by),
    );
    const page = getPage(recurringRuns, req.query.page_token, req.query.page_size);
    const response: V2beta1ListRecurringRunsResponse = {
      next_page_token: page.nextPageToken,
      recurringRuns: page.page,
      total_size: recurringRuns.length,
    };

    res.json(response);
  });

  app.get(v2beta1Prefix + '/recurringruns/:rid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const recurringRun = fixedData.jobs.find((job) => job.id === req.params.rid);
    if (!recurringRun) {
      res.status(404).send(`No recurring run was found with ID: ${req.params.rid}`);
      return;
    }
    res.json(toV2RecurringRun(recurringRun));
  });

  app.get('/hub/', (_, res) => {
    res.sendStatus(200);
  });

  app.get(v1beta1Prefix + '/jobs', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ApiListJobsResponse = {
      jobs: [],
      next_page_token: '',
    };

    let jobs: ApiJob[] = fixedData.jobs;
    const filterQuery = getQueryString(req.query.filter);
    if (filterQuery) {
      jobs = filterResources(fixedData.jobs, filterQuery);
    }

    jobs = sortV1Resources(jobs, ExperimentSortKeys.CREATED_AT, getQueryString(req.query.sort_by));

    const start = getQueryNumber(req.query.page_token) || 0;
    const end = start + (getQueryNumber(req.query.page_size) || 20);
    response.jobs = jobs.slice(start, end);

    if (end < jobs.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.get(v1beta1Prefix + '/experiments', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ApiListExperimentsResponse = {
      experiments: [],
      next_page_token: '',
    };

    let experiments: ApiExperiment[] = fixedData.experiments;
    const filterQuery = getQueryString(req.query.filter);
    if (filterQuery) {
      experiments = filterResources(fixedData.experiments, filterQuery);
    }

    experiments = sortV1Resources(
      experiments,
      ExperimentSortKeys.NAME,
      getQueryString(req.query.sortBy),
    );

    const start = getQueryNumber(req.query.pageToken) || 0;
    const end = start + (getQueryNumber(req.query.pageSize) || 20);
    response.experiments = experiments.slice(start, end);

    if (end < experiments.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.post(v1beta1Prefix + '/experiments', (req, res) => {
    const experiment: ApiExperiment = req.body;
    if (
      fixedData.experiments.find((e) => e.name!.toLowerCase() === experiment.name!.toLowerCase())
    ) {
      res.status(404).send('An experiment with the same name already exists');
      return;
    }
    experiment.id = 'new-experiment-' + (fixedData.experiments.length + 1);
    fixedData.experiments.push(experiment);

    setTimeout(() => {
      res.send(fixedData.experiments[fixedData.experiments.length - 1]);
    }, 1000);
  });

  app.get(v1beta1Prefix + '/experiments/:eid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const experiment = fixedData.experiments.find((exp) => exp.id === req.params.eid);
    if (!experiment) {
      res.status(404).send(`No experiment was found with ID: ${req.params.eid}`);
      return;
    }
    res.json(experiment);
  });

  app.post(v1beta1Prefix + '/jobs', (req, res) => {
    const job: ApiJob = req.body;
    job.id = 'new-job-' + (fixedData.jobs.length + 1);
    job.created_at = new Date();
    job.updated_at = new Date();
    job.enabled = !!job.trigger;
    fixedData.jobs.push(job);

    const date = new Date().toISOString();
    fixedData.runs.push({
      pipeline_runtime: {
        workflow_manifest: JSON.stringify(helloWorldRuntime),
      },
      run: {
        created_at: new Date(),
        id: 'job-at-' + date,
        name: 'job-' + job.name + date,
        scheduled_at: new Date(),
        status: 'Running',
      },
    });
    setTimeout(() => {
      res.send(fixedData.jobs[fixedData.jobs.length - 1]);
    }, 1000);
  });

  app.all(v1beta1Prefix + '/jobs/:jid', (req, res) => {
    res.header('Content-Type', 'application/json');
    switch (req.method) {
      case 'DELETE':
        const i = fixedData.jobs.findIndex((j) => j.id === req.params.jid);
        if (fixedData.jobs[i].name!.startsWith('Cannot be deleted')) {
          res.status(502).send(`Deletion failed for job: '${fixedData.jobs[i].name}'`);
        } else {
          // Delete the job from fixedData.
          fixedData.jobs.splice(i, 1);
          res.json({});
        }
        break;
      case 'GET':
        const job = fixedData.jobs.find((j) => j.id === req.params.jid);
        if (job) {
          res.json(job);
        } else {
          res.status(404).send(`No job was found with ID: ${req.params.jid}`);
        }
        break;
      default:
        res.status(405).send('Unsupported request type: ' + req.method);
    }
  });

  app.get(v1beta1Prefix + '/runs', (req, res) => {
    res.header('Content-Type', 'application/json');
    // Note: the way that we use the next_page_token here may not reflect the way the backend works.
    const response: ApiListRunsResponse = {
      next_page_token: '',
      runs: [],
    };

    let runs: ApiRun[] = fixedData.runs.map((r) => r.run!);

    const filterQuery = getQueryString(req.query.filter);
    if (filterQuery) {
      runs = filterResources(runs, filterQuery);
    }

    const resourceReferenceType = getQueryString(req.query['resource_reference_key.type']);
    const resourceReferenceId = getQueryString(req.query['resource_reference_key.id']);
    if (resourceReferenceType === ApiResourceType.EXPERIMENT && resourceReferenceId) {
      runs = runs.filter((r) =>
        RunUtils.getAllExperimentReferences(r).some(
          (ref) => (ref.key && ref.key.id && ref.key.id === resourceReferenceId) || false,
        ),
      );
    }

    runs = sortV1Resources(runs, RunSortKeys.CREATED_AT, getQueryString(req.query.sort_by));

    const start = getQueryNumber(req.query.page_token) || 0;
    const end = start + (getQueryNumber(req.query.page_size) || 20);
    response.runs = runs.slice(start, end);

    if (end < runs.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.get(v1beta1Prefix + '/runs/:rid', (req, res) => {
    const rid = req.params.rid;
    const run = fixedData.runs.find((r) => r.run!.id === rid);
    if (!run) {
      res.status(404).send('Cannot find a run with id: ' + rid);
      return;
    }
    res.json(run);
  });

  app.post(v1beta1Prefix + '/runs', (req, res) => {
    const date = new Date();
    const run: ApiRun = req.body;
    run.id = 'new-run-' + (fixedData.runs.length + 1);
    run.created_at = date;
    run.scheduled_at = date;
    run.status = 'Running';

    fixedData.runs.push({
      pipeline_runtime: {
        workflow_manifest: JSON.stringify(helloWorldRuntime),
      },
      run,
    });
    setTimeout(() => {
      res.send(fixedData.jobs[fixedData.jobs.length - 1]);
    }, 1000);
  });

  app.post(v1beta1Prefix + '/runs/:rid::method', (req, res) => {
    if (req.params.method !== 'archive' && req.params.method !== 'unarchive') {
      res.status(500).send('Bad method');
    }
    const runDetail = fixedData.runs.find((r) => r.run!.id === req.params.rid);
    if (runDetail) {
      runDetail.run!.storage_state =
        req.params.method === 'archive'
          ? ApiRunStorageState.STORAGESTATE_ARCHIVED
          : ApiRunStorageState.STORAGESTATE_AVAILABLE;
      res.json({});
    } else {
      res.status(500).send('Cannot find a run with id ' + req.params.rid);
    }
  });

  app.post(v1beta1Prefix + '/jobs/:jid/enable', (req, res) => {
    setTimeout(() => {
      const job = fixedData.jobs.find((j) => j.id === req.params.jid);
      if (job) {
        job.enabled = true;
        res.json({});
      } else {
        res.status(500).send('Cannot find a job with id ' + req.params.jid);
      }
    }, 1000);
  });

  app.post(v1beta1Prefix + '/jobs/:jid/disable', (req, res) => {
    setTimeout(() => {
      const job = fixedData.jobs.find((j) => j.id === req.params.jid);
      if (job) {
        job.enabled = false;
        res.json({});
      } else {
        res.status(500).send('Cannot find a job with id ' + req.params.jid);
      }
    }, 1000);
  });

  function filterResources(resources: BaseResource[], filterString?: string): BaseResource[] {
    if (!filterString) {
      return resources;
    }
    const filter: ApiFilter = JSON.parse(decodeURIComponent(filterString));
    ((filter && filter.predicates) || []).forEach((p) => {
      resources = resources.filter((r) => {
        switch (p.op) {
          case PredicateOp.EQUALS:
            if (p.key === 'name') {
              return (
                r.name && r.name.toLocaleLowerCase() === (p.string_value || '').toLocaleLowerCase()
              );
            } else if (p.key === 'storage_state') {
              return (
                (r as ApiRun).storage_state &&
                (r as ApiRun).storage_state!.toString() === p.string_value
              );
            } else {
              throw new Error(`Key: ${p.key} is not yet supported by the mock API server`);
            }
          case PredicateOp.NOT_EQUALS:
            if (p.key === 'name') {
              return (
                r.name && r.name.toLocaleLowerCase() !== (p.string_value || '').toLocaleLowerCase()
              );
            } else if (p.key === 'storage_state') {
              return ((r as ApiRun).storage_state || {}).toString() !== p.string_value;
            } else {
              throw new Error(`Key: ${p.key} is not yet supported by the mock API server`);
            }
          case PredicateOp.IS_SUBSTRING:
            if (p.key !== 'name') {
              throw new Error(`Key: ${p.key} is not yet supported by the mock API server`);
            }
            return (
              r.name &&
              r.name.toLocaleLowerCase().includes((p.string_value || '').toLocaleLowerCase())
            );
          case PredicateOp.GREATER_THAN:
          // Fall through
          case PredicateOp.GREATER_THAN_EQUALS:
          // Fall through
          case PredicateOp.LESS_THAN:
          // Fall through
          case PredicateOp.LESS_THAN_EQUALS:
            // Fall through
            throw new Error(`Op: ${p.op} is not yet supported by the mock API server`);
          default:
            throw new Error(`Unknown Predicate op: ${p.op}`);
        }
      });
    });
    return resources;
  }

  app.get(v1beta1Prefix + '/pipelines', (req, res) => {
    res.header('Content-Type', 'application/json');
    const response: ApiListPipelinesResponse = {
      next_page_token: '',
      pipelines: [],
    };

    let pipelines: ApiPipeline[] = fixedData.pipelines;
    const filterQuery = getQueryString(req.query.filter);
    if (filterQuery) {
      pipelines = filterResources(fixedData.pipelines, filterQuery);
    }

    pipelines = sortV1Resources(
      pipelines,
      PipelineSortKeys.CREATED_AT,
      getQueryString(req.query.sort_by),
    );

    const start = getQueryNumber(req.query.page_token) || 0;
    const end = start + (getQueryNumber(req.query.page_size) || 20);
    response.pipelines = pipelines.slice(start, end);

    if (end < pipelines.length) {
      response.next_page_token = end + '';
    }

    res.json(response);
  });

  app.delete(v1beta1Prefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const i = fixedData.pipelines.findIndex((p) => p.id === req.params.pid);

    if (i === -1) {
      res.status(404).send(`No pipelines was found with ID: ${req.params.pid}`);
      return;
    }

    if (fixedData.pipelines[i].name!.startsWith('Cannot be deleted')) {
      res.status(502).send(`Deletion failed for pipeline: '${fixedData.pipelines[i].name}'`);
      return;
    }
    // Delete the pipelines from fixedData.
    fixedData.pipelines.splice(i, 1);
    res.json({});
  });

  app.get(v1beta1Prefix + '/pipelines/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pipeline = fixedData.pipelines.find((p) => p.id === req.params.pid);
    if (!pipeline) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    res.json(pipeline);
  });

  app.get(v1beta1Prefix + '/pipelines/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');
    const pipeline = fixedData.pipelines.find((p) => p.id === req.params.pid);
    if (!pipeline) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    let filePath = '';
    if (req.params.pid === namedPipelines.noParams.id) {
      filePath = './mock-backend/data/v1/template/mock-conditional-template.yaml';
    } else if (req.params.pid === namedPipelines.unstructuredText.id) {
      filePath = './mock-backend/data/v1/template/mock-recursive-template.yaml';
    } else {
      filePath = './mock-backend/data/v1/template/mock-template.yaml';
    }
    if (v2PipelineSpecMap.has(req.params.pid)) {
      const specPath = v2PipelineSpecMap.get(req.params.pid);
      if (specPath) {
        filePath = specPath;
      }
      console.log(filePath);
    }
    res.send(JSON.stringify({ template: fs.readFileSync(filePath, 'utf-8') }));
  });

  app.get(v1beta1Prefix + '/pipeline_versions/:pid/templates', (req, res) => {
    res.header('Content-Type', 'text/x-yaml');

    // Find v2 pipeline template
    const templatePath = v2PipelineSpecMap.get(req.params.pid);
    if (templatePath != null) {
      console.log(templatePath);
      res.send(JSON.stringify({ template: fs.readFileSync(templatePath, 'utf-8') }));
      return;
    }

    // Default and v1 version list. Return mock template consistently.
    const version = fixedData.versions.find((p) => p.id === req.params.pid);
    if (!version) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    const filePath = './mock-backend/mock-recursive-template.yaml';

    res.send(JSON.stringify({ template: fs.readFileSync(filePath, 'utf-8') }));
  });

  app.get(v1beta1Prefix + '/pipeline_versions/:pid', (req, res) => {
    res.header('Content-Type', 'application/json');
    const pipeline = PIPELINE_VERSIONS_LIST_FULL.find((p) => p.id === req.params.pid);
    if (!pipeline) {
      res.status(404).send(`No pipeline was found with ID: ${req.params.pid}`);
      return;
    }
    res.json(pipeline);
  });

  app.get(v1beta1Prefix + '/pipeline_versions', (req, res) => {
    // Sample query format:
    // query: {
    //   'resource_key.type': 'PIPELINE',
    //   'resource_key.id': '8fbe3bd6-a01f-11e8-98d0-529269fb1459',
    //   page_size: '50',
    //   sort_by: 'created_at desc'
    // },
    const resourceKeyId = getQueryString(req.query['resource_key.id']);
    const resourceKeyType = getQueryString(req.query['resource_key.type']);
    const pageSize = getQueryNumber(req.query.page_size);
    if (resourceKeyId && resourceKeyType === 'PIPELINE' && (pageSize || 0) > 0) {
      const response: ApiListPipelineVersionsResponse = {
        next_page_token: '',
        versions: [],
      };

      let versions: ApiPipelineVersion[] = PIPELINE_VERSIONS_LIST_MAP.get(resourceKeyId) || [];

      if (versions.length === 0) {
        const pipeline = fixedData.pipelines.find((p) => p.id === resourceKeyId);

        if (pipeline == null || !pipeline.default_version) {
          return;
        }

        // Default version list is pipeline with single default version.
        const pipeline_versions_list_response: ApiListPipelineVersionsResponse = {
          total_size: 1,
          versions: [pipeline.default_version],
        };
        res.json(pipeline_versions_list_response);
        return;
      }

      const start = getQueryNumber(req.query.page_token) || 0;
      const end = start + (pageSize || 20);
      response.versions = versions.slice(start, end);

      if (end < versions.length) {
        response.next_page_token = end + '';
      }

      res.json(response);

      return;
    }
    return;
  });

  app.get(v1beta1Prefix + '/pipeline_versions/:pid', (req, res) => {
    // TODO: Temporary returning default version only. It requires
    // keeping a record of all pipeline id in order to search non-default version.
    res.header('Content-Type', 'application/json');
    const pipeline = fixedData.pipelines.find((p) => p.id === req.params.pid);
    if (!pipeline) {
      res
        .status(404)
        .send(
          `No pipeline found with ID: ${req.params.pid}, non-default version can't be found yet.`,
        );
      return;
    }
    if (pipeline.default_version) {
      res.json(pipeline.default_version);
    }
  });

  function mockCreatePipeline(res: Response, name: string, body?: any): void {
    res.header('Content-Type', 'application/json');
    // Don't allow uploading multiple pipelines with the same name
    if (fixedData.pipelines.find((p) => p.name === name)) {
      res
        .status(502)
        .send(`A Pipeline named: "${name}" already exists. Please choose a different name.`);
    } else {
      const pipeline = body || {};
      pipeline.id = 'new-pipeline-' + (fixedData.pipelines.length + 1);
      pipeline.name = name;
      pipeline.created_at = new Date();
      pipeline.description =
        'TODO: the mock middleware does not actually use the uploaded pipeline';
      pipeline.parameters = [
        {
          name: 'output',
        },
        {
          name: 'param-1',
        },
        {
          name: 'param-2',
        },
      ];
      fixedData.pipelines.push(pipeline);
      setTimeout(() => {
        res.send(fixedData.pipelines[fixedData.pipelines.length - 1]);
      }, 1000);
    }
  }

  app.post(v1beta1Prefix + '/pipelines', (req, res) => {
    mockCreatePipeline(res, req.body.name);
  });

  app.post(v1beta1Prefix + '/pipelines/upload', (req, res) => {
    const pipelineName = getRequiredDecodedQueryString(res, req.query.name, 'name');
    if (!pipelineName) {
      return;
    }
    mockCreatePipeline(res, pipelineName, req.body);
  });

  app.get('/artifacts/get', (req, res) => {
    const key = getRequiredDecodedQueryString(res, req.query.key, 'key');
    if (!key) {
      return;
    }
    res.header('Content-Type', 'application/json');
    if (key.endsWith('roc.csv')) {
      sendMockBackendFile(res, rocDataPath);
    } else if (key.endsWith('roc2.csv')) {
      sendMockBackendFile(res, rocDataPath2);
    } else if (key.endsWith('confusion_matrix.csv')) {
      sendMockBackendFile(res, confusionMatrixPath);
    } else if (key.endsWith('table.csv')) {
      sendMockBackendFile(res, tableDataPath);
    } else if (key.endsWith('hello-world.html')) {
      sendMockBackendFile(res, helloWorldHtmlPath);
    } else if (key.endsWith('hello-world-big.html')) {
      sendMockBackendFile(res, helloWorldBigHtmlPath);
    } else if (key === 'analysis') {
      sendMockBackendFile(res, confusionMatrixMetadataJsonPath);
    } else if (key === 'analysis2') {
      sendMockBackendFile(res, confusionMatrixMetadataJsonPath);
    } else if (key === 'model') {
      sendMockBackendFile(res, rocMetadataJsonPath);
    } else if (key === 'model2') {
      sendMockBackendFile(res, rocMetadataJsonPath2);
    } else {
      // TODO: what does production return here?
      res.send('dummy file for key: ' + key);
    }
  });

  app.get('/apps/tensorboard', (req, res) => {
    res.send({ proxyPath: tensorboardPod, tfVersion: '', image: '' });
  });

  app.post('/apps/tensorboard', (req, res) => {
    tensorboardPod = 'apps/tensorboard/proxy/mock-token/';
    setTimeout(() => {
      res.send(tensorboardPod);
    }, 1000);
  });

  app.all(v1beta1Prefix + '/_proxy/*', (_req, res) => {
    res.status(410).send('The generic /_proxy/ endpoint is deprecated and no longer supported.');
  });

  app.get('/k8s/pod/logs', (req, res) => {
    const podName = getRequiredDecodedQueryString(res, req.query.podname, 'podname');
    if (!podName) {
      return;
    }
    if (podName === 'json-12abc') {
      res.status(404).send('pod not found');
      return;
    }
    if (podName === 'coinflip-recursive-q7dqb-3721646052') {
      res.status(500).send('Failed to retrieve log');
      return;
    }
    const shortLog = fs.readFileSync('./mock-backend/shortlog.txt', 'utf-8');
    const longLog = fs.readFileSync('./mock-backend/longlog.txt', 'utf-8');
    const log = podName === 'coinflip-recursive-q7dqb-3466727817' ? longLog : shortLog;
    setTimeout(() => {
      res.send(log);
    }, 300);
  });

  app.get('/visualizations/allowed', (req, res) => {
    res.send(true);
  });

  // Uncomment this instead to test 404 endpoints.
  // app.get('/system/cluster-name', (_, res) => {
  //   res.status(404).send('404 Not Found');
  // });
  // app.get('/system/project-id', (_, res) => {
  //   res.status(404).send('404 Not Found');
  // });
  app.get('/system/cluster-name', (_, res) => {
    res.send('mock-cluster-name');
  });
  app.get('/system/project-id', (_, res) => {
    res.send('mock-project-id');
  });

  app.all(v1beta1Prefix + '*', (req, res) => {
    res.status(404).send('Bad request endpoint.');
  });
  app.all(v2beta1Prefix + '*', (req, res) => {
    res.status(404).send('Bad request endpoint.');
  });
};
