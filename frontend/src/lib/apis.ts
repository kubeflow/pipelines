import { Job, JobMetadata } from '../model/job';
import { Pipeline } from '../model/pipeline';
import { PipelinePackage } from '../model/pipeline_package';

const apisPrefix = '/apis/v1alpha1';

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
export async function newPipeline(pipeline: Pipeline): Promise<Pipeline> {
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
 * Sends an enable pipeline request to the backend.
 */
export async function enablePipeline(id: number): Promise<string> {
  const response = await fetch(apisPrefix + `/pipelines/${id}/enable`, {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  const responseText = await response.text();
  if (!response.ok) {
    throw new Error(responseText);
  }
  return responseText;
}

/**
 * Sends a disable pipeline request to the backend.
 */
export async function disablePipeline(id: number): Promise<string> {
  const response = await fetch(apisPrefix + `/pipelines/${id}/disable`, {
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
  });
  const responseText = await response.text();
  if (!response.ok) {
    throw new Error(responseText);
  }
  return responseText;
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
 * Gets a pipeline job given its id.
 */
export async function getJob(pipelineId: number, jobId: string): Promise<Job> {
  const response = await fetch(apisPrefix + `/pipelines/${pipelineId}/jobs/${jobId}`);
  return await response.json();
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
 * Gets the address (IP + port) of a Tensorboard service given the logdir
 */
export async function getTensorboardApp(logdir: string): Promise<string> {
  const response = await fetch(apisPrefix +
    `/apps/tensorboard?logdir=${encodeURIComponent(logdir)}`);
  return await response.text();
}

/**
 * Starts a deployment and service for Tensorboard given the logdir
 */
export async function startTensorboardApp(logdir: string): Promise<string> {
  const response = await fetch(apisPrefix +
    `/apps/tensorboard?logdir=${encodeURIComponent(logdir)}`, {
      headers: {
        'content-type': 'application/json',
      },
      method: 'POST',
    }
  );
  return await response.text();
}
