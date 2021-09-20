/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Metadata, Workflow } from 'third_party/argo-ui/argo_template';

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

export function parseTaskDisplayNameByNodeId(nodeId: string, workflow?: Workflow): string {
  const node = workflow?.status.nodes[nodeId];
  if (!node) {
    return nodeId;
  }
  const workflowName = workflow?.metadata?.name || '';
  let displayName = node.displayName || node.id;
  if (node.name === `${workflowName}.onExit`) {
    displayName = `onExit - ${node.templateName}`;
  }
  if (workflow?.spec && workflow?.spec.templates) {
    const tmpl = workflow.spec.templates.find(t => t?.name === node.templateName);
    displayName = parseTaskDisplayName(tmpl?.metadata) || displayName;
  }
  return displayName;
}
