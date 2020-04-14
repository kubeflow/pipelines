import { Metadata } from 'third_party/argo-ui/argo_template';

export function parseTaskDisplayName(metadata?: Metadata): string | undefined {
  if (!metadata?.annotations) {
    return undefined;
  }
  const taskDisplayName = metadata.annotations['pipelines.kubeflow.org/task_display_name'];
  let componentDisplayName: string | undefined;
  try {
    componentDisplayName = JSON.parse(metadata.annotations['pipelines.kubeflow.org/component_spec'])
      .name;
  } catch (err) {
    // Expected error: metadata is missing or malformed
  }
  return taskDisplayName || componentDisplayName;
}
