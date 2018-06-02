import { Job } from '../model/job';
import { ListJobsRequest } from '../model/list_jobs_request';
import { ListJobsResponse } from '../model/list_jobs_response';
import { ListPipelinesRequest } from '../model/list_pipelines_request';
import { ListPipelinesResponse } from '../model/list_pipelines_response';
import { Pipeline } from '../model/pipeline';
import { PipelinePackage } from '../model/pipeline_package';

const apisPrefix = '/apis/v1alpha1';

async function _fetch(
    path: string, query?: string, init?: RequestInit): Promise<string> {
  const response = await fetch(apisPrefix + path + (query ? '?' + query : ''), init);
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
    const healthStats = JSON.parse(await _fetch('/healthz'));
    return healthStats.apiServerReady;
  } catch (_) {
    return false;
  }
}

/**
 * Gets a list of the pipeline packages defined on the backend.
 */
export async function getPackages(): Promise<PipelinePackage[]> {
  return JSON.parse(await _fetch('/packages'));
}

/**
 * Gets the details of a certain package given its id.
 */
export async function getPackage(id: number): Promise<PipelinePackage> {
  return JSON.parse(await _fetch(`/packages/${id}`));
}

/**
 * Gets the Argo template of a certain package given its id.
 */
export async function getPackageTemplate(id: number): Promise<string> {
  return await _fetch(`/packages/${id}/templates`);
}

/**
 * Uploads the given package file to the backend, and gets back a PipelinePackage
 * object with its metadata parsed.
 */
export async function uploadPackage(packageData: any): Promise<PipelinePackage> {
  const fd = new FormData();
  fd.append('uploadfile', packageData, packageData.name);
  const response = await _fetch('/packages/upload', '', {
    body: fd,
    cache: 'no-cache',
    method: 'POST',
  });
  return JSON.parse(response);
}

/**
 * Gets a list of the pipeline package pipelines defined on the backend.
 */
export async function getPipelines(request: ListPipelinesRequest): Promise<ListPipelinesResponse> {
  const queryParams: string[] = [];
  queryParams.push('pageToken=' + request.pageToken);
  queryParams.push('pageSize=' + request.pageSize);
  queryParams.push('sortBy=' + encodeURIComponent(request.sortBy));
  queryParams.push('ascending=' + request.orderAscending);
  if (request.filterBy) {
    // TODO: this isn't actually supported by the backend yet (and won't be for a while.) (5/23)
    queryParams.push('filterBy=' + encodeURIComponent(request.filterBy));
  }
  return JSON.parse(await _fetch('/pipelines', queryParams.join('&')));
}

/**
 * Gets the details of a certain package pipeline given its id.
 */
export async function getPipeline(id: number): Promise<Pipeline> {
  return JSON.parse(await _fetch(`/pipelines/${id}`));
}

/**
 * Sends a request to the backened to permanently delete a pipeline.
 */
export async function deletePipeline(id: number): Promise<string> {
  return await _fetch(`/pipelines/${id}`, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'DELETE',
  });
}

/**
 * Sends a new pipeline request to the backend.
 */
export async function newPipeline(pipeline: Pipeline): Promise<Pipeline> {
  const response = await _fetch('/pipelines', '', {
    body: JSON.stringify(pipeline),
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return JSON.parse(response);
}

/**
 * Sends an enable pipeline request to the backend.
 */
export async function enablePipeline(id: number): Promise<string> {
  return await _fetch(`/pipelines/${id}/enable`, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
}

/**
 * Sends a disable pipeline request to the backend.
 */
export async function disablePipeline(id: number): Promise<string> {
  return await _fetch(`/pipelines/${id}/disable`, '', {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
}

/**
 * Gets a list of all the pipeline jobs belonging to the specified pipelined
 * from the backend.
 */
export async function getJobs(request: ListJobsRequest): Promise<ListJobsResponse> {
  const queryParams: string[] = [];
  queryParams.push('pageToken=' + request.pageToken);
  queryParams.push('pageSize=' + request.pageSize);
  queryParams.push('pipelineId=' + request.pipelineId);
  queryParams.push('sortBy=' + encodeURIComponent(request.sortBy));
  queryParams.push('ascending=' + request.orderAscending);
  if (request.filterBy) {
    // TODO: this isn't actually supported by the backend yet (and won't be for a while.) (5/23)
    queryParams.push('filterBy=' + encodeURIComponent(request.filterBy));
  }
  return JSON.parse(await _fetch(`/pipelines/${request.pipelineId}/jobs`, queryParams.join('&')));
}

/**
 * Gets a pipeline job given its id.
 */
export async function getJob(pipelineId: number, jobId: string): Promise<Job> {
  return JSON.parse(await _fetch(`/pipelines/${pipelineId}/jobs/${jobId}`));
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
export async function readFile(path: string): Promise<string> {
  return await _fetch(`/artifacts/get/${encodeURIComponent(path)}`);
}

/**
 * Gets the address (IP + port) of a Tensorboard service given the logdir
 */
export async function getTensorboardApp(logdir: string): Promise<string> {
  return await _fetch(`/apps/tensorboard?logdir=${encodeURIComponent(logdir)}`);
}

/**
 * Starts a deployment and service for Tensorboard given the logdir
 */
export async function startTensorboardApp(logdir: string): Promise<string> {
  return await _fetch(
      `/apps/tensorboard?logdir=${encodeURIComponent(logdir)}`,
      '',
      { headers: { 'content-type': 'application/json', }, method: 'POST', }
  );
}

/**
 * Get pod logs
 */
export async function getPodLogs(podName: string): Promise<string> {
  return await _fetch(`/k8s/pod/logs?podname=${encodeURIComponent(podName)}`);
}
