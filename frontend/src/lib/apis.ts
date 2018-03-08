import { FileDescriptor } from '../components/file-browser/file-browser';
import { Job } from '../lib/job';
import { Pipeline } from '../lib/pipeline';
import { PipelinePackage } from './pipeline_package';

const apisPrefix = '/_api';

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
export async function getJobs(pipelineId: number): Promise<Job[]> {
  const path = `/pipelines/${pipelineId}/jobs`;
  const response = await fetch(apisPrefix + path);
  const jobs: Job[] = await response.json();
  return jobs;
}

/**
 * Gets the details of a certain pipeline pipeline job given its id.
 */
export async function getJob(id: string): Promise<Job> {
  const response = await fetch(apisPrefix + `/jobs/${id}`);
  return await response.json();
}

/**
 * Submits a new job for the given pipeline id.
 */
export async function newJob(id: number): Promise<Job> {
  const response = await fetch(apisPrefix + `/${id}/jobs`, {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  return await response.json();
}

/**
 * List files at a given path from content service.
 */
export async function listFiles(path: string): Promise<FileDescriptor[]> {
  const response = await fetch(path);
  return await response.json();
}

/**
 * Read file from storage using backend.
 */
export async function readFile(path: string): Promise<string> {
  const response = await fetch(path);
  return await response.text();
}
