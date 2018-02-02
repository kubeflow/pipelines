import Template from "./template";

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
