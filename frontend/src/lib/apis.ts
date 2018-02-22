import { FileDescriptor } from '../components/file-browser/file-browser';
import { Instance } from '../lib/instance';
import { Run } from '../lib/run';
import { config } from './config';
import { PipelinePackage } from './pipeline_package';

const backendUrl = config.api;

/**
 * Gets a list of the pipeline packages defined on the backend.
 */
export async function getPackages(): Promise<PipelinePackage[]> {
  const response = await fetch(backendUrl + '/packages');
  const packages: PipelinePackage[] = await response.json();
  return packages;
}

/**
 * Gets the details of a certain package given its id.
 */
export async function getPackage(id: number): Promise<PipelinePackage> {
  const response = await fetch(backendUrl + `/packages/${id}`);
  return await response.json();
}

/**
 * Gets a list of the pipeline package instances defined on the backend.
 */
export async function getInstances(): Promise<Instance[]> {
  const response = await fetch(backendUrl + '/instances');
  const instances: Instance[] = await response.json();
  return instances;
}

/**
 * Gets the details of a certain package instance given its id.
 */
export async function getInstance(id: number): Promise<Instance> {
  const response = await fetch(backendUrl + `/instances/${id}`);
  return await response.json();
}

/**
 * Sends a new instance request to the backend.
 */
export async function newInstance(instance: Instance) {
  const response = await fetch(backendUrl + '/instances', {
    body: JSON.stringify(instance),
    cache: 'no-cache',
    headers: {
      'content-type': 'application/json'
    },
    method: 'POST'
  });
  return await response.json();
}

/**
 * Gets a list of all the pipeline instance runs from the backend.
 * If an instance id is specified, only the runs defined with this
 * instance id are returned.
 */
export async function getRuns(instanceId?: number): Promise<Run[]> {
  const path = '/runs' + (instanceId !== undefined ? '?instanceId=' + instanceId : '');
  const response = await fetch(backendUrl + path);
  const runs: Run[] = await response.json();
  return runs;
}

/**
 * Gets the details of a certain pipeline instance run fiven its id.
 */
export async function getRun(id: number): Promise<Run> {
  const response = await fetch(backendUrl + `/runs/${id}`);
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
