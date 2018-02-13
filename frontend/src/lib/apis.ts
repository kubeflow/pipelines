import { config } from './config';

import { FileDescriptor } from 'src/components/file-browser/file-browser';
import { Run } from 'src/lib/run';
import { Template } from './template';

const backendUrl = config.api;

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
 * Gets a list of all the pipeline template runs from the backend.
 * If an template id is specified, only the runs defined with this
 * template id are returned.
 */
export async function getRuns(templateId?: number): Promise<Run[]> {
  const path = '/runs' + (templateId !== undefined ? '?templateId=' + templateId : '');
  const response = await fetch(backendUrl + path);
  const runs: Run[] = await response.json();
  return runs;
}

/**
 * Gets the details of a certain pipeline template run fiven its id.
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
