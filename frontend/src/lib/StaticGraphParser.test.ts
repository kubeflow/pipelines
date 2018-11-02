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

import { createGraph } from './StaticGraphParser';

describe('StaticGraphParser', () => {

  function newWorkflow(): any {
    return {
      spec: {
        entrypoint: 'template-1',
        templates: [
          {
            dag: {
              tasks: [
                {
                  name: 'task-1',
                  template: 'task-1',
                },
              ],
            },
            name: 'template-1',
          },
        ],
      },
    };
  }

  describe('createGraph', () => {
    it('Creates a single node with no edges for a workflow with one step.', () => {
      const workflow = newWorkflow();
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(1);
      expect(g.edgeCount()).toBe(0);
      expect(g.node(workflow.spec.templates[0].dag.tasks[0].name).label).toBe('task-1');
    });

    it('Adds an unconnected node for the onExit template, if onExit is specified', () => {
      const workflow = newWorkflow();
      workflow.spec.onExit = 'on-exit';
      workflow.spec.templates.push({ container: null, name: 'on-exit' });
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(2);
      expect(g.edgeCount()).toBe(0);
      // Prefix 'onExit - ' is added to exit-handler nodes
      expect(g.node('on-exit').label).toBe('onExit - ' + 'on-exit');
    });

    it('Adds and connects nodes based on listed dependencies', () => {
      const workflow = newWorkflow();
      workflow.spec.templates[0].dag.tasks.push({ name: 'task-2', dependencies: ['task-1'] });
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(2);
      expect(g.edgeCount()).toBe(1);
      expect(g.edges()[0].v).toBe('task-1');
      expect(g.edges()[0].w).toBe('task-2');
    });

    it('Allows there to be and connects nodes based on listed dependencies', () => {
      const workflow = newWorkflow();
      workflow.spec.templates[0].dag.tasks.push({ name: 'task-2', dependencies: ['task-1'] });
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(2);
      expect(g.edgeCount()).toBe(1);
      expect(g.edges()[0].v).toBe('task-1');
      expect(g.edges()[0].w).toBe('task-2');
    });

    // This test covers the way that compiled Pipelines handle DSL-defined exit-handlers.
    // These exit-handlers are different from Argo's 'onExit'.
    // The compiled Pipelines will contain 1 DAG for the entry point, that has a single task which
    // is the exit-handler. The exit-handler will be another DAG that contains all of the Pipeline's
    // components.
    it('Does not show any nodes for the entrypoint DAG if it just points to an another DAG', () => {
      const workflow = {
        spec: {
          entrypoint: 'entrypoint-dag',
          templates: [
            {
              dag: { tasks: [{ name: 'exit-handler-dag', template: 'exit-handler-dag' }] },
              name: 'entrypoint-dag',
            },
            {
              dag: { tasks: [{ name: 'pipeline-component' }] },
              name: 'exit-handler-dag',
            },
          ],
        },
      } as any;
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(1);
      expect(g.edgeCount()).toBe(0);
    });

    it('Throws an error if the workflow has no spec', () => {
      expect(() => createGraph({} as any)).toThrowError('Could not generate graph.');
    });

    it('Throws an error message if the workflow spec has no templates', () => {
      expect(() => createGraph({} as any)).toThrowError('Could not generate graph.');
    });

    it('Throws an error message if the workflow spec has a template with multiple workflow steps', () => {
      const workflow = {
        spec: {
          templates: [
            {
              steps: [
                [{ name: 'workflow-step-1', template: 'workflow-step-1' }],
                [{ name: 'workflow-step-2', template: 'workflow-step-2' }],
              ],
            },
          ],
        },
      } as any;
      expect(() => createGraph(workflow)).toThrowError('Pipeline had template with multiple steps.');
    });

    it('Returns an error message if the workflow spec has a template with multiple steps', () => {
      const workflow = {
        spec: {
          templates: [
            {
              steps: [
                [
                  { name: 'step-1', template: 'step-1' },
                  { name: 'step-2', template: 'step-2' },
                ],
              ],
            },
          ],
        },
      } as any;
      expect(() => createGraph(workflow)).toThrowError('Pipeline had template with multiple steps');
    });
  });
});
