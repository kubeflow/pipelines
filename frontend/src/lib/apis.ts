import { Job } from '../api/job';
import { ListJobsRequest } from '../api/list_jobs_request';
import { ListJobsResponse } from '../api/list_jobs_response';
import { ListPipelinesRequest } from '../api/list_pipelines_request';
import { ListPipelinesResponse } from '../api/list_pipelines_response';
import { ListRunsRequest } from '../api/list_runs_request';
import { ListRunsResponse } from '../api/list_runs_response';
import { Pipeline, PipelineTemplate } from '../api/pipeline';
import { Run } from '../api/run';

const v1alpha2Prefix = '/apis/v1alpha2';

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
export async function listPipelines(request: ListPipelinesRequest): Promise<ListPipelinesResponse> {
  return JSON.parse(await _fetch('/pipelines', v1alpha2Prefix));
}

/**
 * Gets the details of a certain pipeline given its id.
 */
export async function getPipeline(id: string): Promise<Pipeline> {
  return JSON.parse(await _fetch(`/pipelines/${id}`, v1alpha2Prefix));
}

/**
 * Gets the Argo template of a certain pipeline given its id.
 */
export async function getPipelineTemplate(id: number): Promise<PipelineTemplate> {
  return JSON.parse(await _fetch(`/pipelines/${id}/templates`, v1alpha2Prefix));
}

/**
 * Uploads the given pipeline file to the backend, and gets back a Pipeline
 * object with its metadata parsed.
 */
export async function uploadPipeline(pipelineData: any): Promise<Pipeline> {
  const fd = new FormData();
  fd.append('uploadfile', pipelineData, pipelineData.name);
  const response = await _fetch('/pipelines/upload', v1alpha2Prefix, '', {
    body: fd,
    cache: 'no-cache',
    method: 'POST',
  });
  return JSON.parse(response);
}

/**
 * Gets a list of the pipeline jobs defined on the backend.
 */
export async function listJobs(request: ListJobsRequest): Promise<ListJobsResponse> {
  return ListJobsResponse.buildFromObject(
      JSON.parse(await _fetch('/jobs', v1alpha2Prefix, request.toQueryParams())));
}

/**
 * Gets the details of a certain pipeline job given its id.
 */
export async function getJob(id: string): Promise<Job> {
  return Job.buildFromObject(JSON.parse(await _fetch(`/jobs/${id}`, v1alpha2Prefix)));
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
export async function newJob(job: Job): Promise<Job> {
  const response = await _fetch('/jobs', v1alpha2Prefix, '', {
    body: JSON.stringify(job),
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return Job.buildFromObject(JSON.parse(response));
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
export async function listRuns(request: ListRunsRequest): Promise<ListRunsResponse> {
  return ListRunsResponse.buildFromObject(
      JSON.parse(await _fetch(
          `/jobs/${request.jobId}/runs`, v1alpha2Prefix, request.toQueryParams())));
}

/**
 * Gets a job run given its id.
 */
export async function getRun(jobId: string, runId: string): Promise<Run> {
  return Run.buildFromObject(JSON.parse(
      await _fetch(`/jobs/${jobId}/runs/${runId}`, v1alpha2Prefix)));
}

/**
 * Lists files at a given path from content service using server.
 */
export async function listFiles(path: string): Promise<string[]> {
  return JSON.parse(await _fetch(`/artifacts/list/${encodeURIComponent(path)}`));
}

/**
 * Reads file from storage using server.
 */
export function readFile(path: string): Promise<string> {
  return _fetch(`/artifacts/get/${encodeURIComponent(path)}`);
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
