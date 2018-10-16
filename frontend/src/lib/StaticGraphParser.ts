/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as dagre from 'dagre';
import { Workflow } from '../../third_party/argo-ui/argo_template';

export interface SelectedNodeInfo {
  args: string[];
  inputs: string[][];
  outputs: string[][];
}

export function createGraph(workflow: Workflow) {
  const g = new dagre.graphlib.Graph();
  g.setGraph({});
  g.setDefaultEdgeLabel(() => ({}));

  const NODE_WIDTH = 180;
  const NODE_HEIGHT = 70;

  const workflowTemplates = workflow.spec.templates;

  // TODO: handle steps / conditionals.
  workflowTemplates.forEach((template) => {
    if (template.dag && template.dag.tasks) {
      template.dag.tasks.forEach((task) => {
        g.setNode(task.name, {
          height: NODE_HEIGHT,
          label: task.name,
          width: NODE_WIDTH,
        });
        (task.dependencies || []).forEach((dep) => g.setEdge(dep, task.name));
      });
    }
  });

  return g;
}

export function getNodeInfo(workflow?: Workflow, nodeId?: string): SelectedNodeInfo {
  const info: SelectedNodeInfo = { args: [], inputs: [[]], outputs: [[]] };
  if (!nodeId || !workflow || !workflow.spec || !workflow.spec.templates) {
    return info;
  }

  for (const template of workflow.spec.templates) {
    if (template.container && template.name === nodeId) {
      info.args = template.container && template.container.args || [];
      if (template.inputs) {
        info.inputs = (template.inputs.parameters || []).map(p => [p.name, p.value || '']);
      }
      if (template.outputs) {
        info.outputs = (template.outputs.parameters || []).map(p => {
          let value = '';
          if (p.value) {
            value = p.value;
          } else if (p.valueFrom) {
            value = p.valueFrom.jqFilter || p.valueFrom.jsonPath || p.valueFrom.parameter || p.valueFrom.path || '';
          }
          return [p.name, value];
        });
      }
    }
  }
  return info;
}
