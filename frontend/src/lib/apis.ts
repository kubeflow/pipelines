import { Workflow } from '../model/argo_template';
import { Pipeline } from '../model/pipeline';
import { PipelinePackage } from '../model/pipeline_package';

const apisPrefix = '/apis/v1alpha1';

export interface JobMetadata {
  name: string;
  scheduledAt: number;
}

/**
 * Gets a list of the pipeline packages defined on the backend.
 */
export async function getPackages(): Promise<PipelinePackage[]> {
  const response = await fetch(apisPrefix + '/packages');
  const packages: PipelinePackage[] = await response.json();
  return packages;
}

/**
 * Gets the details of a certain package given its id.
 */
export async function getPackage(id: number): Promise<PipelinePackage> {
  const response = await fetch(apisPrefix + `/packages/${id}`);
  return await response.json();
}

/**
 * Gets the Argo template of a certain package given its id.
 */
export async function getPackageTemplate(id: number): Promise<string> {
  const response = await fetch(apisPrefix + `/packages/${id}/templates`);
  return await response.text();
}

/**
 * Uploads the given package file to the backend, and gets back a PipelinePackage
 * object with its metadata parsed.
 */
export async function uploadPackage(packageData: any): Promise<PipelinePackage> {
  const fd = new FormData();
  fd.append('uploadfile', packageData, packageData.name);
  const response = await fetch(apisPrefix + '/packages/upload', {
    body: fd,
    cache: 'no-cache',
    method: 'POST',
  });
  return await response.json();
}

/**
 * Gets a list of the pipeline package pipelines defined on the backend.
 */
export async function getPipelines(): Promise<Pipeline[]> {
  const response = await fetch(apisPrefix + '/pipelines');
  const pipelines: Pipeline[] = await response.json();
  return pipelines;
}

/**
 * Gets the details of a certain package pipeline given its id.
 */
export async function getPipeline(id: number): Promise<Pipeline> {
  const response = await fetch(apisPrefix + `/pipelines/${id}`);
  return await response.json();
}

/**
 * Sends a new pipeline request to the backend.
 */
export async function newPipeline(pipeline: Pipeline) {
  const response = await fetch(apisPrefix + '/pipelines', {
    body: JSON.stringify(pipeline),
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return await response.json();
}

/**
 * Gets a list of all the pipeline jobs belonging to the specified pipelined
 * from the backend.
 */
export async function getJobs(pipelineId: number): Promise<JobMetadata[]> {
  const path = `/pipelines/${pipelineId}/jobs`;
  const response = await fetch(apisPrefix + path);
  const jobs: JobMetadata[] = await response.json();
  return jobs;
}

/**
 * Gets the details of a certain pipeline pipeline job given its id.
 */
export async function getJob(pipelineId: number, jobId: string): Promise<Workflow> {
  const response = await fetch(apisPrefix + `/pipelines/${pipelineId}/jobs/${jobId}`);
  return (await response.json()).job;
}

/**
 * Lists files at a given path from content service using server.
 */
export async function listFiles(path: string): Promise<string[]> {
  const response = await fetch(apisPrefix + `/artifacts/list/${encodeURIComponent(path)}`);
  return await response.json();
}

/**
 * Reads file from storage using server.
 */
export async function readFile(path: string): Promise<string> {
  const response = await fetch(apisPrefix + `/artifacts/get/${encodeURIComponent(path)}`);
  return await response.text();
}

/**
 * Gets the address (IP + port) of a pod running a certain process
 */
export async function getPod(processName: string, args: string): Promise<string> {
  const response = await fetch(apisPrefix + `/pods/${processName}?${encodeURIComponent(args)}`);
  return await response.text();
}

/**
 * Starts a pod given a process name and arguments
 */
export async function startPod(processName: string, args: string) {
  const response = await fetch(apisPrefix + `/pods/${processName}?${encodeURIComponent(args)}`, {
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return await response.text();
}
