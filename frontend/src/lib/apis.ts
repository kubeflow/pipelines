import { Job } from '../api/job';
import { ListJobsRequest } from '../api/list_jobs_request';
import { ListJobsResponse } from '../api/list_jobs_response';
import { ListPackagesRequest } from '../api/list_packages_request';
import { ListPackagesResponse } from '../api/list_packages_response';
import { ListPipelinesRequest } from '../api/list_pipelines_request';
import { ListPipelinesResponse } from '../api/list_pipelines_response';
import { Pipeline } from '../api/pipeline';
import { PackageTemplate, PipelinePackage } from '../api/pipeline_package';

const v1alpha2Prefix = '/apis/v1alpha2';

async function _fetch(
    path: string, apisPrefix?: string, query?: string, init?: RequestInit): Promise<string> {
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
 * Gets a list of the pipeline packages defined on the backend.
 */
export async function getPackages(request: ListPackagesRequest): Promise<ListPackagesResponse> {
  return JSON.parse(await _fetch('/packages', v1alpha2Prefix));
}

/**
 * Gets the details of a certain package given its id.
 */
export async function getPackage(id: number): Promise<PipelinePackage> {
  return JSON.parse(await _fetch(`/packages/${id}`, v1alpha2Prefix));
}

/**
 * Gets the Argo template of a certain package given its id.
 */
export async function getPackageTemplate(id: number): Promise<PackageTemplate> {
  return JSON.parse(await _fetch(`/packages/${id}/templates`, v1alpha2Prefix));
}

/**
 * Uploads the given package file to the backend, and gets back a PipelinePackage
 * object with its metadata parsed.
 */
export async function uploadPackage(packageData: any): Promise<PipelinePackage> {
  const fd = new FormData();
  fd.append('uploadfile', packageData, packageData.name);
  const response = await _fetch('/packages/upload', v1alpha2Prefix, '', {
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
  return ListPipelinesResponse.buildFromObject(
      JSON.parse(await _fetch('/pipelines', v1alpha2Prefix, request.toQueryParams())));
}

/**
 * Gets the details of a certain package pipeline given its id.
 */
export async function getPipeline(id: string): Promise<Pipeline> {
  return Pipeline.buildFromObject(JSON.parse(await _fetch(`/pipelines/${id}`, v1alpha2Prefix)));
}

/**
 * Sends a request to the backened to permanently delete a pipeline.
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
 * Sends a new pipeline request to the backend.
 */
export async function newPipeline(pipeline: Pipeline): Promise<Pipeline> {
  const response = await _fetch('/pipelines', v1alpha2Prefix, '', {
    body: JSON.stringify(pipeline),
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return Pipeline.buildFromObject(JSON.parse(response));
}

/**
 * Sends an enable pipeline request to the backend.
 */
export function enablePipeline(id: string): Promise<string> {
  return _fetch(`/pipelines/${id}/enable`, v1alpha2Prefix, '', {
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
export function disablePipeline(id: string): Promise<string> {
  return _fetch(`/pipelines/${id}/disable`, v1alpha2Prefix, '', {
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
  return ListJobsResponse.buildFromObject(
      JSON.parse(await _fetch(
          `/pipelines/${request.pipelineId}/jobs`, v1alpha2Prefix, request.toQueryParams())));
}

/**
 * Gets a pipeline job given its id.
 */
export async function getJob(pipelineId: string, jobId: string): Promise<Job> {
  return Job.buildFromObject(JSON.parse(
      await _fetch(`/pipelines/${pipelineId}/jobs/${jobId}`, v1alpha2Prefix)));
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
