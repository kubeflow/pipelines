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
import { Workflow, Template } from '../../third_party/argo-ui/argo_template';
import { logger } from './Utils';

export type nodeType = 'container' | 'dag' | 'unknown';

export class SelectedNodeInfo {
  public args: string[];
  public command: string[];
  public condition: string;
  public image: string;
  public inputs: string[][];
  public nodeType: nodeType;
  public outputs: string[][];

  constructor() {
    this.args = [];
    this.command = [];
    this.condition = '';
    this.image = '';
    this.inputs = [[]];
    this.nodeType = 'unknown';
    this.outputs = [[]];
  }
}


const NODE_WIDTH = 172;
const NODE_HEIGHT = 64;

export function _populateInfoFromTemplate(info: SelectedNodeInfo, template?: Template): SelectedNodeInfo {
  if (!template || !template.container) {
    return info;
  }

  info.nodeType = 'container';
  info.args = template.container.args || [],
  info.command = template.container.command || [],
  info.image = template.container.image || '';

  if (template.inputs) {
    info.inputs =
      (template.inputs.parameters || []).map(p => [p.name, p.value || '']);
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
  return info;
}

/**
 * Recursively construct the static graph of the Pipeline.
 *
 * @param graph The dagre graph object
 * @param rootTemplateId The current template being explored at this level of recursion
 * @param templates An unchanging map of template name to template and type
 * @param parent The task that was being examined when the function recursed
 */
function buildDag(
    graph: dagre.graphlib.Graph,
    rootTemplateId: string,
    templates: Map<string, { nodeType: nodeType, template: Template }>,
    parent?: string): void {

  const root = templates.get(rootTemplateId);

  if (root && root.nodeType === 'dag') {
    const template = root.template;

    (template.dag.tasks || []).forEach((task) => {

      const nodeId = (parent || '') + '/' + task.name;

      // If the user specifies an exit handler, then the compiler will wrap the entire Pipeline
      // within an additional DAG template prefixed with "exit-handler".
      // If this is the case, we simply treat it as the root of the graph and work from there
      if (task.name.startsWith('exit-handler')) {
        buildDag(graph, task.template, templates);
      } else {

        // Parent here will be the task that pointed to this DAG template.
        // We do not set an edge if the task has dependencies because in that case it makes sense
        // for incoming edges to only be drawn from its dependencies, not the DAG node itself
        if (parent && !task.dependencies) {
          graph.setEdge(parent, nodeId);
        }

        // This object contains information about the node that we display to the user when they
        // click on a node in the graph
        const info = new SelectedNodeInfo();
        if (task.when) {
          info.condition = task.when;
        }

        // "Child" here is the template that this task points to. This template should either be a
        // DAG, in which case we recurse, or a container which can be thought of as a leaf node
        const child = templates.get(task.template);
        if (child) {
          if (child.nodeType === 'dag') {
            buildDag(graph, task.template, templates, nodeId);
          } else if (child.nodeType === 'container' ) {
            _populateInfoFromTemplate(info, child.template);
          } else {
            logger.error(`Unknown nodetype: ${child.nodeType} on template: ${child.template}`);
          }
        }

        graph.setNode(nodeId, {
          bgColor: task.when ? 'cornsilk' : undefined,
          height: NODE_HEIGHT,
          info,
          label: task.template,
          width: NODE_WIDTH,
        });
      }

      // DAG tasks can indicate dependencies which are graphically shown as parents with edges
      // pointing to their children (the task(s)).
      // TODO: The addition of the parent prefix to the dependency here is only valid if nodes only
      // ever directly depend on their siblings. This is true now but may change in the future, and
      // this will need to be updated.
      (task.dependencies || []).forEach((dep) => graph.setEdge((parent || '') + '/' + dep, nodeId));
    });
  }
}

export function createGraph(workflow: Workflow): dagre.graphlib.Graph {
  const graph = new dagre.graphlib.Graph();
  graph.setGraph({});
  graph.setDefaultEdgeLabel(() => ({}));

  if (!workflow.spec || !workflow.spec.templates) {
    throw new Error('Could not generate graph. Provided Pipeline had no components.');
  }

  const workflowTemplates = workflow.spec.templates;

  const templates = new Map<string, { nodeType: nodeType, template: Template }>();

  // Iterate through the workflow's templates to construct a map which will be used to traverse and
  // construct the graph
  for (const template of workflowTemplates.filter(t => !!t && !!t.name)) {
    // Argo allows specifying a single global exit handler. We also highlight that node
    if (template.name === workflow.spec.onExit) {
      const info = new SelectedNodeInfo();
      _populateInfoFromTemplate(info, template);
      graph.setNode(template.name, {
        bgColor: '#eee',
        height: NODE_HEIGHT,
        info,
        label: 'onExit - ' + template.name,
        width: NODE_WIDTH,
      });
    }

    if (template.container) {
      templates.set(template.name, { nodeType: 'container', template });
    } else if (template.dag) {
      templates.set(template.name, { nodeType: 'dag', template });
    } else {
      logger.verbose(`Template: ${template.name} was neither a Container nor a DAG`);
    }
  }

  buildDag(graph, workflow.spec.entrypoint, templates);

  // DSL-compiled Pipelines will always include a DAG, so they should never reach this point.
  // It is, however, possible for users to upload manually constructed Pipelines, and extremely
  // simple ones may have no steps or DAGs, just an entry point container.
  if (graph.nodeCount() === 0) {
    const entryPointTemplate = workflowTemplates.find((t) => t.name === workflow.spec.entrypoint);
    if (entryPointTemplate) {
      graph.setNode(entryPointTemplate.name, {
        height: NODE_HEIGHT,
        label: entryPointTemplate.name,
        width: NODE_WIDTH,
      });
    }
  }

  return graph;
}
