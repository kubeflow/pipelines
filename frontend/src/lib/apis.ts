import Template from "./template";
import { Instance } from "./instance";

/**
 * Gets a list of the pipeline templates defined on the backend.
 */
export async function getTemplates(): Promise<Template[]> {
  const response = await fetch('/_templates');
  const templates: Template[] = await response.json();
  return templates;
}

/**
 * Gets the details of a certain template given its id.
 */
export async function getTemplate(id: number): Promise<Template> {
  const response = await fetch(`/_templates?id=${id}`);
  const templates: Template[] = await response.json();
  return templates[0];
}

/**
 * Gets a list of the pipeline template instances defined on the backend.
 */
export async function getInstances(): Promise<Instance[]> {
  const response = await fetch('/_instances');
  const instances: Instance[] = await response.json();
  return instances;
}

/**
 * Gets the details of a certain template instance given its id.
 */
export async function getInstance(id: number): Promise<Instance> {
  const response = await fetch(`/_instances?id=${id}`);
  const instances: Instance[] = await response.json();
  return instances[0];
}
