import Template from "./template";
import { Instance } from "./instance";
import Run from "src/lib/run";
import * as config from './config';

const backendUrl = config.default.api;

/**
 * Gets a list of the pipeline templates defined on the backend.
 */
export async function getTemplates(): Promise<Template[]> {
  const response = await fetch(backendUrl + '/templates');
  const templates: Template[] = await response.json();
  return templates;
}

/**
 * Gets the details of a certain template given its id.
 */
export async function getTemplate(id: number): Promise<Template> {
  const response = await fetch(backendUrl + `/templates/${id}`);
  return await response.json();
}

/**
 * Gets a list of the pipeline template instances defined on the backend.
 */
export async function getInstances(): Promise<Instance[]> {
  const response = await fetch(backendUrl + '/instances');
  const instances: Instance[] = await response.json();
  return instances;
}

/**
 * Gets the details of a certain template instance given its id.
 */
export async function getInstance(id: number): Promise<Instance> {
  const response = await fetch(backendUrl + `/instances/${id}`);
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
export async function getRun(id: number): Promise<Run[]> {
  const response = await fetch(backendUrl + `/runs/${id}`);
  const runs: Run[] = await response.json();
  return runs;
}
