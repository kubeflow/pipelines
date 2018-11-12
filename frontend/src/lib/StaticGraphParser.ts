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

interface ConditionalInfo {
  condition: string;
  taskName: string;
}

export interface SelectedNodeInfo {
  nodeType: 'container' | 'dag' | 'unknown';
  containerInfo?: {
    args: string[];
    command: string[];
    image: string;
    inputs: string[][];
    outputs: string[][];
  };
  dagInfo?: {
    conditionalTasks: ConditionalInfo[];
  };
}

export function createGraph(workflow: Workflow): dagre.graphlib.Graph {
  const g = new dagre.graphlib.Graph();
  g.setGraph({});
  g.setDefaultEdgeLabel(() => ({}));

  const NODE_WIDTH = 180;
  const NODE_HEIGHT = 70;

  if (!workflow.spec || !workflow.spec.templates) {
    throw new Error('Could not generate graph. Provided Pipeline had no components.');
  }

  const workflowTemplates = workflow.spec.templates;

  // Iterate through the workflow's templates to construct the graph
  for (const template of workflowTemplates) {
    // Argo allows specifying a single global exit handler. We also highlight that node.
    if (template.name === workflow.spec.onExit) {
      g.setNode(template.name, {
        bgColor: '#eee',
        height: NODE_HEIGHT,
        label: 'onExit - ' + template.name,
        width: NODE_WIDTH,
      });
    }

    // TODO remove
    // g.setNode(template.name, {
    //   height: NODE_HEIGHT,
    //   label: template.name,
    //   width: NODE_WIDTH,
    // });
    // DAGs are the main component making up a Pipeline, and each DAG is composed of tasks.
    if (template.dag && template.dag.tasks) {

      template.dag.tasks.forEach((task) => {
        // tslint:disable-next-line:no-console
        console.log('task', task);
        if (!task.name.startsWith('exit-handler')) {
          g.setNode(task.name, {
            bgColor: task.when ? 'cornsilk' : undefined,
            height: NODE_HEIGHT,
            label: task.name,
            width: NODE_WIDTH,
          });
        }

        // TODO remove
        // g.setEdge(task.name, task.template);

        if (template.name !== workflow.spec.entrypoint && !template.name.startsWith('exit-handler')) {
          // g.setEdge(template.name, task.name);
        }

        // DAG tasks can indicate dependencies which are graphically shown as parents with edges
        // pointing to their children (the task(s)).
        // (task.dependencies || []).forEach((dep) => g.setEdge(dep, task.name));
      });
    }
  }

  // tslint:disable-next-line:no-console
  console.log(g);

  // DSL-compiled Pipelines will always include a DAG, so they should never reach this point.
  // It is, however, possible for users to upload manually constructed Pipelines, and extremely
  // simple ones may have no steps or DAGs, just an entry point container.
  if (g.nodeCount() === 0) {
    const entryPointTemplate = workflowTemplates.find((t) => t.name === workflow.spec.entrypoint);
    if (entryPointTemplate) {
      g.setNode(entryPointTemplate.name, {
        height: NODE_HEIGHT,
        label: entryPointTemplate.name,
        width: NODE_WIDTH,
      });
    }
  }

  return g;
}

export function getNodeInfo(workflow?: Workflow, nodeId?: string): SelectedNodeInfo | null {
  if (!nodeId || !workflow || !workflow.spec || !workflow.spec.templates) {
    return null;
  }

  const info: SelectedNodeInfo = { nodeType: 'unknown' };
  const template = workflow.spec.templates.find((t) => t.name === nodeId);
  if (template) {
    if (template.container) {
      info.nodeType = 'container';
      info.containerInfo = {
        args: template.container.args || [],
        command: template.container.command || [],
        image: template.container.image || '',
        inputs: [[]],
        outputs: [[]]
      };
      if (template.inputs) {
        info.containerInfo.inputs =
          (template.inputs.parameters || []).map(p => [p.name, p.value || '']);
      }
      if (template.outputs) {
        info.containerInfo.outputs = (template.outputs.parameters || []).map(p => {
          let value = '';
          if (p.value) {
            value = p.value;
          } else if (p.valueFrom) {
            value = p.valueFrom.jqFilter || p.valueFrom.jsonPath || p.valueFrom.parameter || p.valueFrom.path || '';
          }
          return [p.name, value];
        });
      }
    } else if (template.dag && template.dag.tasks) {
      // Conditionals are represented by DAG tasks with 'when' properties.
      info.nodeType = 'dag';
      const conditionalTasks: ConditionalInfo[] = [];
      template.dag.tasks.forEach((task) => {
        if (task.when) {
          conditionalTasks.push({ condition: task.when, taskName: task.name });
        }
      });
      info.dagInfo = { conditionalTasks };
    }
  }
  return info;
}
