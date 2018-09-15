// Copyright 2018 Google LLC
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

import { apiJob, apiListJobsResponse } from '../api/job';
import { apiGetTemplateResponse , apiListPipelinesResponse, apiPipeline } from '../api/pipeline';
import { apiListRunsResponse, apiRunDetail } from '../api/run';
import { StoragePath } from '../model/storage';

const v1alpha2Prefix = '/apis/v1alpha2';

export interface BaseListRequest {
  filterBy?: string;
  orderAscending?: boolean;
  pageSize?: number;
  pageToken?: string;
  sortBy?: string;
}

// tslint:disable-next-line:no-empty-interface
export interface ListJobsRequest extends BaseListRequest {
}

// tslint:disable-next-line:no-empty-interface
export interface ListPipelinesRequest extends BaseListRequest {
}

export interface ListRunsRequest extends BaseListRequest {
  jobId?: string;
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum RunSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name'
}

// Valid sortKeys as specified by the backend.
export enum PipelineSortKeys {
  AUTHOR = 'id',
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
}

// Valid sortKeys as specified by the backend.
export enum JobSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
  PIPELINE_ID = 'pipeline_id'
}

function requestToQueryParams(
    request: ListRunsRequest | ListJobsRequest | ListPipelinesRequest): string {
  const params = [];
  if (request.pageSize !== undefined) {
    params.push(`pageSize=${request.pageSize}`);
  }
  if (request.pageToken !== undefined) {
    params.push(`pageToken=${request.pageToken}`);
  }
  if (request.sortBy !== undefined) {
    // Sort is ascending by default.
    // Descending order is indicated by ' desc' after the sorted field name.
    let sortByParam = `sortBy=${request.sortBy}`;
    if (request.orderAscending === false) {
      sortByParam += ' desc';
    }
    params.push(sortByParam);
  }
  if (request.filterBy !== undefined) {
    params.push(`filterBy=${request.filterBy}`);
  }
  if ('jobId' in request && request.jobId !== undefined) {
    params.push(`jobId=${request.jobId}`);
  }
  return params.join('&');
}

async function _fetch(
    path: string, apisPrefix?: string, query?: string, init?: RequestInit): Promise<string> {
  init = Object.assign(init || {}, { credentials: 'same-origin' });
  const response = await fetch((apisPrefix || '') + path + (query ? '?' + query : ''), init);
  const responseText = await response.text();
  if (response.ok) {
    return responseText;
  } else {
    throw new Error(responseText);
  }
}

/**
 * Checks if the API server is ready for traffic.
 */
export async function isApiServerReady(): Promise<boolean> {
  try {
    const healthStats = JSON.parse(await _fetch('/healthz', v1alpha2Prefix));
    return healthStats.apiServerReady;
  } catch (_) {
    return false;
  }
}

/**
 * Gets a list of the pipelines defined on the backend.
 */
export async function listPipelines(
    request: ListPipelinesRequest): Promise<apiListPipelinesResponse> {
  const pipelinesResult = JSON.parse(
      await _fetch('/pipelines', v1alpha2Prefix, requestToQueryParams(request)));
  (pipelinesResult.pipelines || []).forEach((p: any) => {
    p.created_at = new Date(p.created_at);
  });
  return pipelinesResult;
}

/**
 * Gets the details of a certain pipeline given its id.
 */
export async function getPipeline(id: string): Promise<apiPipeline> {
  const pipeline = JSON.parse(await _fetch(`/pipelines/${id}`, v1alpha2Prefix));
  pipeline.created_at = new Date(pipeline.created_at);
  return pipeline;
}

/**
 * Sends a request to the backend to delete a pipeline.
 */
export function deletePipeline(id: string): Promise<string> {
  return _fetch(`/pipelines/${id}`, v1alpha2Prefix, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'DELETE',
  });
}

/**
 * Gets the Argo template of a certain pipeline given its id.
 */
export async function getPipelineTemplate(id: string): Promise<apiGetTemplateResponse> {
  return JSON.parse(await _fetch(`/pipelines/${id}/templates`, v1alpha2Prefix));
}

/**
 * Uploads the given pipeline file to the backend, and gets back a Pipeline
 * object with its metadata parsed.
 */
export async function uploadPipeline(
    pipelineName: string, pipelineData: File): Promise<apiPipeline> {
  const fd = new FormData();
  fd.append('uploadfile', pipelineData, pipelineData.name);
  const response =
      await _fetch(
          '/pipelines/upload',
          v1alpha2Prefix,
          `pipelineName=${encodeURIComponent(pipelineName)}`,
        {
          body: fd,
          cache: 'no-cache',
          method: 'POST',
        });
  return JSON.parse(response);
}

/**
 * Gets a list of the pipeline jobs defined on the backend.
 */
export async function listJobs(request: ListJobsRequest): Promise<apiListJobsResponse> {
  const jobsResult = JSON.parse(
      await _fetch('/jobs', v1alpha2Prefix, requestToQueryParams(request)));
  (jobsResult.jobs || []).forEach((j: any) => {
    if (j.created_at) {
      j.created_at = new Date(j.created_at);
    }
    if (j.updated_at) {
      j.updated_at = new Date(j.updated_at);
    }
    if (j.trigger) {
      if (j.trigger.cron_schedule && j.trigger.cron_schedule.start_time) {
        j.trigger.cron_schedule.start_time = new Date(j.trigger.cron_schedule.start_time);
      } if (j.trigger.cron_schedule && j.trigger.cron_schedule.end_time) {
        j.trigger.cron_schedule.end_time = new Date(j.trigger.cron_schedule.end_time);
      }
      if (j.trigger.periodic_schedule && j.trigger.periodic_schedule.start_time) {
        j.trigger.periodic_schedule.start_time = new Date(j.trigger.periodic_schedule.start_time);
      } if (j.trigger.periodic_schedule && j.trigger.periodic_schedule.end_time) {
        j.trigger.periodic_schedule.end_time = new Date(j.trigger.periodic_schedule.end_time);
      }
    }
  });
  return jobsResult;
}

/**
 * Gets the details of a certain pipeline job given its id.
 */
export async function getJob(id: string): Promise<apiJob> {
  const job = JSON.parse(await _fetch(`/jobs/${id}`, v1alpha2Prefix));
  if (job.created_at) {
    job.created_at = new Date(job.created_at);
  }
  if (job.updated_at) {
    job.updated_at = new Date(job.updated_at);
  }
  if (job.trigger) {
    if (job.trigger.cron_schedule && job.trigger.cron_schedule.start_time) {
      job.trigger.cron_schedule.start_time = new Date(job.trigger.cron_schedule.start_time);
    } if (job.trigger.cron_schedule && job.trigger.cron_schedule.end_time) {
      job.trigger.cron_schedule.end_time = new Date(job.trigger.cron_schedule.end_time);
    }
    if (job.trigger.periodic_schedule && job.trigger.periodic_schedule.start_time) {
      job.trigger.periodic_schedule.start_time = new Date(job.trigger.periodic_schedule.start_time);
    } if (job.trigger.periodic_schedule && job.trigger.periodic_schedule.end_time) {
      job.trigger.periodic_schedule.end_time = new Date(job.trigger.periodic_schedule.end_time);
    }
  }
  return job;
}

/**
 * Sends a request to the backened to permanently delete a job.
 */
export function deleteJob(id: string): Promise<string> {
  return _fetch(`/jobs/${id}`, v1alpha2Prefix, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'DELETE',
  });
}

/**
 * Sends a new job request to the backend.
 */
export async function newJob(job: apiJob): Promise<apiJob> {
  const response = await _fetch('/jobs', v1alpha2Prefix, '', {
    body: JSON.stringify(job),
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return JSON.parse(response);
}

/**
 * Sends an enable job request to the backend.
 */
export function enableJob(id: string): Promise<string> {
  return _fetch(`/jobs/${id}/enable`, v1alpha2Prefix, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
}

/**
 * Sends a disable job request to the backend.
 */
export function disableJob(id: string): Promise<string> {
  return _fetch(`/jobs/${id}/disable`, v1alpha2Prefix, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
}

/**
 * Gets a list of all the job runs from the backend.
 */
export async function listRuns(request: ListRunsRequest): Promise<apiListRunsResponse> {
  let response;
  if (request.jobId) {
    response = JSON.parse(await _fetch(
        `/jobs/${request.jobId}/runs`, v1alpha2Prefix, requestToQueryParams(request)));
  } else {
    response = JSON.parse(await _fetch(`/runs`, v1alpha2Prefix, requestToQueryParams(request)));
  }
  (response.runs || []).forEach((r: any) => {
    if (r.created_at) {
      r.created_at = new Date(r.created_at);
    }
    if (r.updated_at) {
      r.updated_at = new Date(r.updated_at);
    }
  });
  return response;
}

/**
 * Gets a job run given its id.
 */
export async function getRun(runId: string): Promise<apiRunDetail> {
  const run = JSON.parse(await _fetch(`/runs/${runId}`, v1alpha2Prefix));
  if (run.created_at) {
    run.created_at = new Date(run.created_at);
  }
  if (run.updated_at) {
    run.updated_at = new Date(run.updated_at);
  }
  return run;
}

/**
 * Reads file from storage using server.
 */
export function readFile(path: StoragePath): Promise<string> {
  return _fetch('/artifacts/get' +
      `?source=${path.source}&bucket=${path.bucket}&key=${encodeURIComponent(path.key)}`);
}

/**
 * Gets the address (IP + port) of a Tensorboard service given the logdir
 */
export function getTensorboardApp(logdir: string): Promise<string> {
  return _fetch(`/apps/tensorboard?logdir=${encodeURIComponent(logdir)}`);
}

/**
 * Starts a deployment and service for Tensorboard given the logdir
 */
export function startTensorboardApp(logdir: string): Promise<string> {
  return _fetch(
      `/apps/tensorboard?logdir=${encodeURIComponent(logdir)}`,
      undefined,
      undefined,
      { headers: { 'content-type': 'application/json', }, method: 'POST', }
  );
}

/**
 * Get pod logs
 */
export function getPodLogs(podName: string): Promise<string> {
  return _fetch(`/k8s/pod/logs?podname=${encodeURIComponent(podName)}`);
}
