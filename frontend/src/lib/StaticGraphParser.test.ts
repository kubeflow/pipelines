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

import { createGraph, getNodeInfo } from './StaticGraphParser';

describe('StaticGraphParser', () => {

  function newWorkflow(): any {
    return {
      spec: {
        entrypoint: 'template-1',
        templates: [
          {
            dag: { tasks: [{ name: 'task-1', template: 'task-1', },], },
            name: 'template-1',
          },
          {
            container: {},
            name: 'container-1',
          },
        ],
      },
    };
  }

  // In this pipeline, the conditionals are the templates: 'condition-1' and 'condition-2'
  // 'condition-1-child' and 'condition-2-child' are not displayed in the static graph, but they
  // are used by the parser to properly connect the nodes.
  function newConditionalWorkflow(): any {
    return {
      spec: {
        entrypoint: 'pipeline-flip-coin',
        templates: [
          {
            name: 'condition-1',
            steps: [[{
              name: 'condition-1-child',
              template: 'condition-1-child',
              when: '{{inputs.parameters.flip-output}} == heads'
            }]]
          },
          { dag: { tasks: [{ name: 'heads', template: 'heads' }] }, name: 'condition-1-child' },
          {
            name: 'condition-2',
            steps: [[{
              name: 'condition-2-child',
              template: 'condition-2-child',
              when: '{{inputs.parameters.flip-output}} == tails'
            }]]
          },
          { dag: { tasks: [{ name: 'tails', template: 'tails' }] }, name: 'condition-2-child' },
          {
            dag: {
              tasks: [
                { dependencies: ['flip'], name: 'condition-1', template: 'condition-1' },
                { dependencies: ['flip'], name: 'condition-2', template: 'condition-2' },
                { name: 'flip', template: 'flip' }
              ]
            },
            name: 'pipeline-flip-coin'
          },
          {
            container: { args: [ /* omitted */], command: ['sh', '-c'], },
            name: 'flip',
            outputs: { parameters: [{ name: 'flip-output', valueFrom: { path: '/tmp/output' } }] }
          },
          { container: { command: ['echo', '\'heads\''], }, name: 'heads', },
          { container: { command: ['echo', '\'tails\''], }, name: 'tails', },
        ]
      }
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

    it('Shows conditional nodes without adding conditional children as nodes', () => {
      const g = createGraph(newConditionalWorkflow());
      expect(g.nodeCount()).toBe(5);
      ['flip', 'condition-1', 'condition-2', 'heads', 'tails'].forEach((nodeName) => {
        expect(g.nodes()).toContain(nodeName);
      });
    });

    it('Connects conditional graph correctly', () => {
      const g = createGraph(newConditionalWorkflow());
      // 'flip' has two possible outcomes: 'condition-1' and 'condition-2'
      expect(g.edges()[0].v).toBe('flip');
      expect(g.edges()[0].w).toBe('condition-1');
      expect(g.edges()[1].v).toBe('flip');
      expect(g.edges()[1].w).toBe('condition-2');
      // condition-1 means the 'heads' node will run
      expect(g.edges()[2].v).toBe('condition-1');
      expect(g.edges()[2].w).toBe('heads');
      // condition-2 means the 'tails' node will run
      expect(g.edges()[3].v).toBe('condition-2');
      expect(g.edges()[3].w).toBe('tails');
      // Confirm there are no other nodes or edges
      expect(g.nodeCount()).toBe(5);
      expect(g.edgeCount()).toBe(4);
    });

    it('Finds conditionals and colors them', () => {
      const g = createGraph(newConditionalWorkflow());
      g.nodes().forEach((nodeName) => {
        const node = g.node(nodeName);
        if (nodeName.startsWith('condition')) {
          expect(node.bgColor).toBe('cornsilk');
        } else {
          expect(node.bgColor).toBeUndefined();
        }
      });
    });

    it('Renders extremely simple workflows with no steps or DAGs', () => {
      const simpleWorkflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [ { container: {}, name: 'template-1', }, ],
        },
      } as any;
      const g = createGraph(simpleWorkflow);
      expect(g.nodeCount()).toBe(1);
      expect(g.edgeCount()).toBe(0);
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

    it('Throws an error message if the workflow spec has a template with multiple steps', () => {
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

  describe('getNodeInfo', () => {
    it('Returns nodeInfo containing only \'unknown\' nodeType if nodeId is undefined', () => {
      const nodeInfo = getNodeInfo(newWorkflow(), undefined);
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if nodeId is empty', () => {
      const nodeInfo = getNodeInfo(newWorkflow(), '');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if workflow is undefined', () => {
      const nodeInfo = getNodeInfo(undefined, 'someId');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if workflow.spec is undefined', () => {
      const workflow = newWorkflow();
      workflow.spec = undefined;
      const nodeInfo = getNodeInfo(workflow, 'someId');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if workflow.spec.templates is undefined', () => {
      const workflow = newWorkflow();
      workflow.spec.templates = undefined;
      const nodeInfo = getNodeInfo(workflow, 'someId');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if no template matches provided ID', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {},
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'id-not-in-spec');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if template matching provided ID has no container or steps', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              // No container or steps here
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if template matching provided ID has no container and empty steps', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              // No container here
              name: 'template-1',
              steps: [],
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing only \'unknown\' nodeType if template matching provided ID has no container and array of empty steps', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              // No container here
              name: 'template-1',
              steps: [[]],
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo).toEqual({ nodeType: 'unknown' });
    });

    it('Returns nodeInfo containing container args', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {
                args: ['arg1', 'arg2'],
              },
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.args).toEqual(['arg1', 'arg2']);
    });

    it('Returns nodeInfo containing container commands', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {
                command: ['command1', 'command2']
              },
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.command).toEqual(['command1', 'command2']);
    });

    it('Returns nodeInfo containing container image name', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {
                image: 'someImageID'
              },
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.image).toBe('someImageID');
    });

    it('Returns nodeInfo containing container inputs as list of name/value duples', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {},
              inputs: {
                parameters: [
                  { name: 'param1', value: 'val1' },
                  { name: 'param2', value: 'val2' },
                ],
              },
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.inputs).toEqual([['param1', 'val1'], ['param2', 'val2']]);
    });

    it('Returns empty strings for inputs with no specified value', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {},
              inputs: {
                parameters: [
                  { name: 'param1' },
                  { name: 'param2' },
                ],
              },
              name: 'template-1',
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.inputs).toEqual([['param1', ''], ['param2', '']]);
    });

    it('Returns nodeInfo containing container outputs as list of name/value duples, pulling from valueFrom if necessary', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {},
              name: 'template-1',
              outputs: {
                parameters: [
                  { name: 'param1', value: 'val1' },
                  { name: 'param2', valueFrom: { jsonPath: 'jsonPath' } },
                  { name: 'param3', valueFrom: { path: 'path' } },
                  { name: 'param4', valueFrom: { parameter: 'parameterReference' } },
                  { name: 'param5', valueFrom: { jqFilter: 'jqFilter' } },
                ],
              },
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.outputs).toEqual([
        ['param1', 'val1'],
        ['param2', 'jsonPath'],
        ['param3', 'path'],
        ['param4', 'parameterReference'],
        ['param5', 'jqFilter'],
      ]);
    });

    it('Returns empty strings for outputs with no specified value', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              container: {},
              name: 'template-1',
              outputs: {
                parameters: [
                  { name: 'param1' },
                  { name: 'param2' },
                ],
              },
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.containerInfo!.outputs).toEqual([['param1', ''], ['param2', '']]);
    });

    it('Returns nodeType \'steps\' when node template has steps', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              name: 'template-1',
              steps: [[
                'something',
              ]]
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('steps');
      expect(nodeInfo.stepsInfo).toEqual({ conditional: '', parameters: [[]]);
    });

    it('Returns nodeInfo with step template\'s conditional when node template has \'when\' property', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              name: 'template-1',
              steps: [[ { when: '{{someVar}} == something' } ]]
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('steps');
      expect(nodeInfo.stepsInfo).toEqual({ conditional: '{{someVar}} == something', parameters: [[]]);
    });

    it('Returns nodeInfo with step template\'s arguments when node template has \'when\' property', () => {
      const workflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [
            {
              name: 'template-1',
              steps: [[{
                arguments: {
                  parameters: [
                    { name: 'param1', value: 'val1' },
                    { name: 'param2', value: 'val2' },
                  ],
                },
              }]]
            },
          ],
        },
      } as any;
      const nodeInfo = getNodeInfo(workflow, 'template-1');
      expect(nodeInfo.nodeType).toBe('steps');
      expect(nodeInfo.stepsInfo)
        .toEqual({ conditional: '', parameters: [['param1', 'val1'], ['param2', 'val2']]);
    });
  });
});
