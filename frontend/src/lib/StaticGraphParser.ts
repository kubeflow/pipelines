/*
 * Copyright 2018-2019 Google LLC
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
import { Template, Workflow } from '../../third_party/argo-ui/argo_template';
import { color } from '../Css';
import { Constants } from './Constants';
import { logger } from './Utils';
import { parseTaskDisplayName } from './ParserUtils';

export type nodeType = 'container' | 'resource' | 'dag' | 'unknown';

export interface KeyValue<T> extends Array<any> {
  0?: string;
  1?: T;
}

export class SelectedNodeInfo {
  public args: string[];
  public command: string[];
  public condition: string;
  public image: string;
  public inputs: Array<KeyValue<string>>;
  public nodeType: nodeType;
  public outputs: Array<KeyValue<string>>;
  public volumeMounts: Array<KeyValue<string>>;
  public resource: Array<KeyValue<string>>;

  constructor() {
    this.args = [];
    this.command = [];
    this.condition = '';
    this.image = '';
    this.inputs = [[]];
    this.nodeType = 'unknown';
    this.outputs = [[]];
    this.volumeMounts = [[]];
    this.resource = [[]];
  }
}

export function _populateInfoFromTemplate(
  info: SelectedNodeInfo,
  template?: Template,
): SelectedNodeInfo {
  if (!template || (!template.container && !template.resource)) {
    return info;
  }

  if (template.container) {
    info.nodeType = 'container';
    info.args = template.container.args || [];
    info.command = template.container.command || [];
    info.image = template.container.image || '';
    info.volumeMounts = (template.container.volumeMounts || []).map(v => [v.mountPath, v.name]);
  } else {
    info.nodeType = 'resource';
    if (template.resource && template.resource.action && template.resource.manifest) {
      info.resource = [[template.resource.action, template.resource.manifest]];
    } else {
      info.resource = [[]];
    }
  }

  if (template.inputs) {
    info.inputs = (template.inputs.parameters || []).map(p => [p.name, p.value || '']);
  }
  if (template.outputs) {
    info.outputs = (template.outputs.parameters || []).map(p => {
      let value = '';
      if (p.value) {
        value = p.value;
      } else if (p.valueFrom) {
        value =
          p.valueFrom.jqFilter ||
          p.valueFrom.jsonPath ||
          p.valueFrom.parameter ||
          p.valueFrom.path ||
          '';
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
 * @param parentFullPath The task that was being examined when the function recursed. This string
 * includes all of the nodes above it. For example, with a graph such as:
 *     A
 *    / \
 *   B   C
 *      / \
 *     D   E
 * where A and C are DAGs, the parentFullPath when rootTemplateId is C would be: /A/C
 */
function buildDag(
  graph: dagre.graphlib.Graph,
  rootTemplateId: string,
  templates: Map<string, { nodeType: nodeType; template: Template }>,
  alreadyVisited: Map<string, string>,
  parentFullPath: string,
): void {
  const root = templates.get(rootTemplateId);

  if (root && root.nodeType === 'dag') {
    // Mark that we have visited this DAG, and save the original qualified path to it.
    alreadyVisited.set(rootTemplateId, parentFullPath || '/' + rootTemplateId);
    const template = root.template;

    (template.dag.tasks || []).forEach(task => {
      const nodeId = parentFullPath + '/' + task.name;

      // If the user specifies an exit handler, then the compiler will wrap the entire Pipeline
      // within an additional DAG template prefixed with "exit-handler".
      // If this is the case, we simply treat it as the root of the graph and work from there
      if (task.name.startsWith('exit-handler')) {
        buildDag(graph, task.template, templates, alreadyVisited, '');
        return;
      }

      // If this task has already been visited, retrieve the qualified path name that was assigned
      // to it, add an edge, and move on to the next task
      if (alreadyVisited.has(task.name)) {
        graph.setEdge(parentFullPath, alreadyVisited.get(task.name)!);
        return;
      }

      // Parent here will be the task that pointed to this DAG template.
      // Within a DAG template, tasks can have dependencies on one another, and long chains of
      // dependencies can be present within a single DAG. In these cases, we choose not to draw an
      // edge from the DAG node itself to these tasks with dependencies because such an edge would
      // not be meaningful to the user. For example, consider a DAG A with two tasks, B and C, where
      // task C depends on the output of task B. C is a task of A, but it's much more semantically
      // important that C depends on B, so to avoid cluttering the graph, we simply omit the edge
      // between A and C:
      //      A                  A
      //    /   \    becomes    /
      //   B <-- C             B
      //                      /
      //                     C
      if (parentFullPath && !task.dependencies) {
        graph.setEdge(parentFullPath, nodeId);
      }

      // This object contains information about the node that we display to the user when they
      // click on a node in the graph
      const info = new SelectedNodeInfo();
      if (task.when) {
        info.condition = task.when;
      }

      // "Child" here is the template that this task points to. This template should either be a
      // DAG, in which case we recurse, or a container/resource which can be thought of as a
      // leaf node
      const child = templates.get(task.template);
      let nodeLabel = task.template;
      if (child) {
        if (child.nodeType === 'dag') {
          buildDag(graph, task.template, templates, alreadyVisited, nodeId);
        } else if (child.nodeType === 'container' || child.nodeType === 'resource') {
          nodeLabel = parseTaskDisplayName(child.template.metadata) || nodeLabel;
          _populateInfoFromTemplate(info, child.template);
        } else {
          throw new Error(
            `Unknown nodetype: ${child.nodeType} on workflow template: ${child.template}`,
          );
        }
      }

      graph.setNode(nodeId, {
        bgColor: task.when ? 'cornsilk' : undefined,
        height: Constants.NODE_HEIGHT,
        info,
        label: nodeLabel,
        width: Constants.NODE_WIDTH,
      });

      // DAG tasks can indicate dependencies which are graphically shown as parents with edges
      // pointing to their children (the task(s)).
      // TODO: The addition of the parent prefix to the dependency here is only valid if nodes only
      // ever directly depend on their siblings. This is true now but may change in the future, and
      // this will need to be updated.
      (task.dependencies || []).forEach(dep => graph.setEdge(parentFullPath + '/' + dep, nodeId));
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

  const templates = new Map<string, { nodeType: nodeType; template: Template }>();

  // Iterate through the workflow's templates to construct a map which will be used to traverse and
  // construct the graph
  for (const template of workflowTemplates.filter(t => !!t && !!t.name)) {
    // Argo allows specifying a single global exit handler. We also highlight that node
    if (template.name === workflow.spec.onExit) {
      const info = new SelectedNodeInfo();
      _populateInfoFromTemplate(info, template);
      graph.setNode(template.name, {
        bgColor: color.lightGrey,
        height: Constants.NODE_HEIGHT,
        info,
        label: 'onExit - ' + template.name,
        width: Constants.NODE_WIDTH,
      });
    }

    if (template.container) {
      templates.set(template.name, { nodeType: 'container', template });
    } else if (template.resource) {
      templates.set(template.name, { nodeType: 'resource', template });
    } else if (template.dag) {
      templates.set(template.name, { nodeType: 'dag', template });
    } else {
      logger.verbose(`Template: ${template.name} was neither a Container/Resource nor a DAG`);
    }
  }

  buildDag(graph, workflow.spec.entrypoint, templates, new Map(), '');

  // DSL-compiled Pipelines will always include a DAG, so they should never reach this point.
  // It is, however, possible for users to upload manually constructed Pipelines, and extremely
  // simple ones may have no steps or DAGs, just an entry point container.
  if (graph.nodeCount() === 0) {
    const entryPointTemplate = workflowTemplates.find(t => t.name === workflow.spec.entrypoint);
    if (entryPointTemplate) {
      graph.setNode(entryPointTemplate.name, {
        height: Constants.NODE_HEIGHT,
        label: entryPointTemplate.name,
        width: Constants.NODE_WIDTH,
      });
    }
  }

  return graph;
}
