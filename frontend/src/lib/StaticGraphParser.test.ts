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

import { createGraph, _populateInfoFromTemplate, SelectedNodeInfo } from './StaticGraphParser';

describe('StaticGraphParser', () => {

  function newWorkflow(): any {
    return {
      spec: {
        entrypoint: 'template-1',
        templates: [
          {
            dag: { tasks: [{ name: 'task-1', template: 'container-1' }] },
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

  function newConditionalWorkflow(): any {
    return {
      spec: {
        entrypoint: 'pipeline-flip-coin',
        templates: [
          // Root/entrypoint DAG
          {
            dag: {
              tasks: [
                { name: 'flip-task', template: 'flip' },
                {
                  dependencies: ['flip-task'],
                  name: 'condition-1-task',
                  template: 'condition-1',
                  when: '{{inputs.parameters.flip-output}} == heads',
                },
                {
                  dependencies: ['flip-task'],
                  name: 'condition-2-task',
                  template: 'condition-2',
                  when: '{{inputs.parameters.flip-output}} == tails'
                },
              ]
            },
            name: 'pipeline-flip-coin',
          },
          // DAG representing execution if result is 'heads'
          { dag: { tasks: [{ name: 'heads-task', template: 'heads' }] }, name: 'condition-1' },
          // DAG representing execution if result is 'tails'
          { dag: { tasks: [{ name: 'tails-task', template: 'tails' }] }, name: 'condition-2' },
          // Container that flips the coin and outputs the result
          { container: {}, name: 'flip' },
          // Container that is run if result is 'heads'
          { container: {}, name: 'heads' },
          // Container that is run if result is 'tails'
          { container: {}, name: 'tails' },
        ]
      }
    };
  }

  describe('createGraph', () => {
    it('creates a single node with no edges for a workflow with one step.', () => {
      const workflow = newWorkflow();
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(1);
      expect(g.edgeCount()).toBe(0);
      expect(g.nodes()[0]).toBe('/task-1');
      expect(g.node('/task-1').label).toBe('container-1');
    });

    it('adds an unconnected node for the onExit template, if onExit is specified', () => {
      const workflow = newWorkflow();
      workflow.spec.onExit = 'on-exit';
      workflow.spec.templates.push({ container: {} as any, name: 'on-exit' });
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(2);
      expect(g.edgeCount()).toBe(0);
      // Prefix 'onExit - ' is added to exit-handler nodes
      expect(g.node('on-exit').label).toBe('onExit - ' + 'on-exit');
    });

    it('adds and connects nodes based on listed dependencies', () => {
      const workflow = newWorkflow();
      workflow.spec.templates[0].dag.tasks.push({ name: 'task-2', dependencies: ['task-1'] });
      const g = createGraph(workflow);
      expect(g.nodeCount()).toBe(2);
      expect(g.edgeCount()).toBe(1);
      expect(g.edges()[0].v).toBe('/task-1');
      expect(g.edges()[0].w).toBe('/task-2');
    });

    const nestedWorkflow = {
      spec: {
        entrypoint: 'dag-1',
        templates: [
          {
            dag: { tasks: [{ name: 'task-1-1', template: 'dag-2' }] },
            name: 'dag-1',
          },
          {
            dag: {
              tasks: [
                { name: 'task-2-1', template: 'container-2-1' },
                { name: 'task-2-2', template: 'dag-3' },
              ]
            },
            name: 'dag-2',
          },
          {
            dag: { tasks: [{ name: 'task-3-1', template: 'container-3-1' }] },
            name: 'dag-3',
          },
          { container: {}, name: 'container-2-1' },
          { container: {}, name: 'container-3-1' },
        ],
      },
    } as any;

    it('uses path to task as node ID', () => {
      const g = createGraph(nestedWorkflow);
      expect(g.nodeCount()).toBe(4);
      expect(g.edgeCount()).toBe(3);
      [
        '/task-1-1',
        '/task-1-1/task-2-1',
        '/task-1-1/task-2-2',
        '/task-1-1/task-2-2/task-3-1',
      ].forEach((nodeId) => expect(g.nodes()).toContain(nodeId));
    });

    it('uses task\'s template as node label', () => {
      const g = createGraph(nestedWorkflow);
      expect(g.nodeCount()).toBe(4);
      expect(g.edgeCount()).toBe(3);
      [
        { id: '/task-1-1', label: 'dag-2' },
        { id: '/task-1-1/task-2-1', label: 'container-2-1' },
        { id: '/task-1-1/task-2-2', label: 'dag-3' },
        { id: '/task-1-1/task-2-2/task-3-1', label: 'container-3-1' },
      ].forEach((idAndLabel) => expect(g.node(idAndLabel.id).label).toBe(idAndLabel.label));
    });

    it('connects conditional graph correctly', () => {
      const g = createGraph(newConditionalWorkflow());
      // 'flip' has two possible outcomes: 'condition-1' and 'condition-2'
      expect(g.edges()).toContainEqual({ v: '/flip-task', w: '/condition-1-task' });
      expect(g.edges()).toContainEqual({ v: '/flip-task', w: '/condition-2-task' });
      expect(g.edges()).toContainEqual({ v: '/condition-1-task', w: '/condition-1-task/heads-task' });
      expect(g.edges()).toContainEqual({ v: '/condition-2-task', w: '/condition-2-task/tails-task' });
      expect(g.nodeCount()).toBe(5);
      expect(g.edgeCount()).toBe(4);
    });

    it('finds conditionals and colors them', () => {
      const g = createGraph(newConditionalWorkflow());
      g.nodes().forEach((nodeName) => {
        const node = g.node(nodeName);
        if (nodeName === '/condition-1-task' || nodeName === '/condition-2-task') {
          expect(node.bgColor).toBe('cornsilk');
        } else {
          expect(node.bgColor).toBeUndefined();
        }
      });
    });

    it('includes the conditional itself in the info of conditional nodes', () => {
      const g = createGraph(newConditionalWorkflow());
      g.nodes().forEach((nodeName) => {
        const node = g.node(nodeName);
        if (nodeName.startsWith('condition-1')) {
          expect(node.info.condition).toBe('{{inputs.parameters.flip-output}} == heads');
        } else if (nodeName.startsWith('condition-2')) {
          expect(node.info.condition).toBe('{{inputs.parameters.flip-output}} == tails');
        }
      });
    });

    it('renders extremely simple workflows with no steps or DAGs', () => {
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
    it('does not show any nodes for the entrypoint DAG if it just points to an another DAG', () => {
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

    it('throws an error if the workflow has no spec', () => {
      expect(() => createGraph({} as any)).toThrowError('Could not generate graph.');
    });

    it('throws an error message if the workflow spec has no templates', () => {
      expect(() => createGraph({} as any)).toThrowError('Could not generate graph.');
    });
  });

  describe('populateInfoFromTemplate', () => {
    it('returns default SelectedNodeInfo if template is undefined', () => {
      const defaultSelectedNodeInfo = new SelectedNodeInfo();
      const nodeInfo = _populateInfoFromTemplate(defaultSelectedNodeInfo, undefined);
      expect(nodeInfo).toEqual(defaultSelectedNodeInfo);
    });

    it('returns default SelectedNodeInfo if template has no container', () => {
      const defaultSelectedNodeInfo = new SelectedNodeInfo();
      const nodeInfo = _populateInfoFromTemplate(defaultSelectedNodeInfo, { name: 'template' } as any);
      expect(nodeInfo).toEqual(defaultSelectedNodeInfo);
    });

    it('returns nodeInfo with empty values for args, command, and/or image if container does not have them', () => {
      const template = {
        container: {
          // No args
          // No command
          // No image
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.args).toEqual([]);
      expect(nodeInfo.command).toEqual([]);
      expect(nodeInfo.image).toEqual('');
    });


    it('returns nodeInfo containing container args', () => {
      const template = {
        container: {
          args: ['arg1', 'arg2'],
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.args).toEqual(['arg1', 'arg2']);
    });

    it('returns nodeInfo containing container commands', () => {
      const template = {
        container: {
          command: ['echo', 'some-command'],
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.command).toEqual(['echo', 'some-command']);
    });

    it('returns nodeInfo containing container image', () => {
      const template = {
        container: {
          image: 'some-image'
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.image).toEqual('some-image');
    });

    it('returns nodeInfo with empty values if template does not have inputs and/or outputs', () => {
      const template = {
        container: {},
        dag: [],
        // No inputs
        // No outputs
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.inputs).toEqual([[]]);
      expect(nodeInfo.outputs).toEqual([[]]);
    });

    it('returns nodeInfo containing template inputs params as list of name/value tuples', () => {
      const template = {
        container: {},
        dag: [],
        inputs: {
          parameters: [{ name: 'param1', value: 'val1' }, { name: 'param2', value: 'val2' }]
        },
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.inputs).toEqual([['param1', 'val1'], ['param2', 'val2']]);
    });

    it('returns empty strings for inputs with no specified value', () => {
      const template = {
        container: {},
        dag: [],
        inputs: {
          parameters: [{ name: 'param1' }, { name: 'param2' }]
        },
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.inputs).toEqual([['param1', ''], ['param2', '']]);
    });

    it('returns nodeInfo containing container outputs as list of name/value tuples, pulling from valueFrom if necessary', () => {
      const template = {
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
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.outputs).toEqual([
        ['param1', 'val1'],
        ['param2', 'jsonPath'],
        ['param3', 'path'],
        ['param4', 'parameterReference'],
        ['param5', 'jqFilter'],
      ]);
    });

    it('returns empty strings for outputs with no specified value', () => {
      const template = {
        container: {},
        name: 'template-1',
        outputs: {
          parameters: [
            { name: 'param1' },
            { name: 'param2' },
          ],
        },
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toBe('container');
      expect(nodeInfo.outputs).toEqual([['param1', ''], ['param2', '']]);
    });

  });
});

