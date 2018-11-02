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
import { Workflow, WorkflowStep } from '../../third_party/argo-ui/argo_template';

export interface SelectedNodeInfo {
  nodeType: 'container' | 'steps' | 'unknown';
  containerInfo?: {
    args: string[];
    command: string[];
    image: string;
    inputs: string[][];
    outputs: string[][];
  };
  stepsInfo?: {
    conditional: string;
    parameters: string[][];
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

  const nonEntryPointTemplatesAndDagTasks = new Map<string, string>();
  const templatesAndSteps: Array<[string, string]> = [];

  // Iterate through the workflow's templates to gather the information needed to construct the
  // graph. Some information is stored in the above variables because Containers, Dags, and Steps
  // are all templates, and we do not know in which order we will encounter them while iterating.
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

    // Steps are only used by the DSL as a workaround for handling conditionals because DAGs are not
    // sufficient. (TODO: why?) We keep track of each template and its step(s) so that we can draw
    // edges from the DAG task with that template name to its grandchild. This template/DAG task
    // will always be a conditional, so we can use this list of tuples to do additional formatting
    // of the node as desired.
    if (template.steps) {
      if (template.steps.length > 1 || template.steps[0].length > 1) {
        throw new Error('Pipeline had template with multiple steps. Was this Pipeline compiled?');
      }
      const step = template.steps[0];
      // Though the types system says 'steps' is WorkflowStep[][], sometimes the inner elements
      // are just WorkflowStep, not WorkflowStep[], but we still want to treat them the same.
      (step.length ? step : [step as WorkflowStep]).forEach((s) => {
        templatesAndSteps.push([template.name, s.name!]);
      });
    }

    // DAGs are the main component making up a Pipeline, and each DAG is composed of tasks.
    // TODO: add tests for the assertions below
    // A compiled Pipeline will only have multiple DAGs if:
    //   1) it defines an exit-handler within the DSL, which will result in its own DAG wrapping the
    //      entirety of the Pipeline.
    //   2) it includes conditionals. In this case, there will also be Steps templates corresponding
    //      to those conditions, and each Steps template will have a child DAG representing the rest
    //      of the Pipeline from that conditional on.
    if (template.dag && template.dag.tasks) {
      template.dag.tasks.forEach((task) => {
        g.setNode(task.name, {
          height: NODE_HEIGHT,
          label: task.name,
          width: NODE_WIDTH,
        });

        // DAG tasks can indicate dependencies which are graphically shown as parents with edges
        // pointing to their children (the task(s)).
        (task.dependencies || []).forEach((dep) => g.setEdge(dep, task.name));

        // A Pipeline may contain multiple DAGs, but the Pipeline will only have a single entry
        // point.
        //
        // The first task of the entry point DAG can be thought of as the root of the Pipeline.
        // TODO: add tests for the assertion below
        // Compiled Pipelines should always have exactly 1 root.
        //
        // For DAGs that do not contain the entry point, we keep track of all tasks which don't have
        // dependencies because they are the roots of that DAG (note that a DAG may have multiple
        // roots). These roots are important for stitching together the overall Pipeline graph, but
        // they are not relevant themselves to the user.
        //
        // The non-root tasks within these DAGs, however, are important parts of the Pipeline. We
        // keep a map of DAG templates to their non-root tasks so that we can draw edges between
        // the tasks and their grandparents.
        //
        // We also use the map to remove any of the templates that weren't already skipped for being
        // step templates.
        if (template.name !== workflow.spec.entrypoint
          && (!task.dependencies || !task.dependencies.length)) {
          nonEntryPointTemplatesAndDagTasks.set(template.name, task.name);
        }
      });
    }
  }

  // TODO: do we need to worry about rewiring from parents to children here?
  // Remove all non-entry-point DAG templates. They are artifacts of the compilation system and not
  // useful to users.
  nonEntryPointTemplatesAndDagTasks.forEach((_, k) => g.removeNode(k));

  // Connect conditionals to their grandchildren; the tasks of non-entry-point DAGs.
  // We can also apply any desired formatting to the conditional nodes here.
  templatesAndSteps.forEach((tuple) => {
    // Add styling for conditional nodes
    g.node(tuple[0]).bgColor = 'cornsilk';

    const grandchild = nonEntryPointTemplatesAndDagTasks.get(tuple[1]);
    if (grandchild) {
      g.setEdge(tuple[0], grandchild);
    }
  });

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

export function getNodeInfo(workflow?: Workflow, nodeId?: string): SelectedNodeInfo {
  const info: SelectedNodeInfo = { nodeType: 'unknown' };
  if (!nodeId || !workflow || !workflow.spec || !workflow.spec.templates) {
    return info;
  }

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
    } else if (template.steps && template.steps[0] && template.steps[0].length) {
      // 'template.steps' represent conditionals.
      // There should only ever be 1 WorkflowStep in a steps template, and it should have a 'when'.
      info.nodeType = 'steps';
      info.stepsInfo = { conditional: '', parameters: [[]] };

      info.stepsInfo.conditional = template.steps[0][0].when || '';

      if (template.steps[0][0].arguments) {
        info.stepsInfo.parameters =
          (template.steps[0][0].arguments!.parameters || []).map(p => [p.name, p.value || '']);
      }
    }
  }
  return info;
}
