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

import { createGraph, SelectedNodeInfo, _populateInfoFromTemplate } from './StaticGraphParser';

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
          {
            // Root/entrypoint DAG
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
                  when: '{{inputs.parameters.flip-output}} == tails',
                },
              ],
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
        ],
      },
    };
  }

  function newResourceCreatingWorkflow(): any {
    return {
      spec: {
        entrypoint: 'template-1',
        templates: [
          {
            dag: {
              tasks: [
                { name: 'create-pvc-task', template: 'create-pvc' },
                {
                  dependencies: ['create-pvc-task'],
                  name: 'container-1',
                  template: 'container-1',
                },
                {
                  dependencies: ['container-1'],
                  name: 'create-snapshot-task',
                  template: 'create-snapshot',
                },
              ],
            },
            name: 'template-1',
          },
          {
            name: 'create-pvc',
            resource: {
              action: 'create',
              manifest: 'apiVersion: v1\nkind: PersistentVolumeClaim',
            },
          },
          {
            container: {},
            name: 'container-1',
          },
          {
            name: 'create-snapshot',
            resource: {
              action: 'create',
              manifest: 'apiVersion: snapshot.storage.k8s.io/v1alpha1\nkind: VolumeSnapshot',
            },
          },
        ],
      },
    };
  }

  describe('createGraph', () => {
    it('creates a single node with no edges for a workflow with one step.', () => {
      const workflow = newWorkflow();
      const g = createGraph(workflow);
      expect(g.nodeCount()).toEqual(1);
      expect(g.edgeCount()).toEqual(0);
      expect(g.nodes()).toContain('/task-1');
      expect(g.node('/task-1').label).toEqual('container-1');
    });

    it("uses task's annotation task_display_name as node label when present", () => {
      const workflow = newWorkflow();
      workflow.spec.templates[1].metadata = {
        annotations: {
          'pipelines.kubeflow.org/task_display_name': 'TaskDisplayName',
        },
      };
      const g = createGraph(workflow);
      expect(g.node('/task-1').label).toEqual('TaskDisplayName');
    });

    it("uses task's annotation component_name as node label when present", () => {
      const workflow = newWorkflow();
      workflow.spec.templates[1].metadata = {
        annotations: {
          'pipelines.kubeflow.org/component_spec': '{"name":"Component Display Name"}',
        },
      };
      const g = createGraph(workflow);
      expect(g.node('/task-1').label).toEqual('Component Display Name');
    });

    it("uses task's default node label when component_name is malformed", () => {
      const workflow = newWorkflow();
      workflow.spec.templates[1].metadata = {
        annotations: {
          'pipelines.kubeflow.org/component_spec': '"name":"Component Display Name"}',
        },
      };
      const g = createGraph(workflow);
      expect(g.node('/task-1').label).not.toEqual('Component Display Name');
    });

    it('adds an unconnected node for the onExit template, if onExit is specified', () => {
      const workflow = newWorkflow();
      workflow.spec.onExit = 'on-exit';
      workflow.spec.templates.push({ container: {} as any, name: 'on-exit' });
      const g = createGraph(workflow);
      expect(g.edgeCount()).toEqual(0);
      const expectedNodes = ['/task-1', 'on-exit'];
      expectedNodes.forEach(nodeId => expect(g.nodes()).toContain(nodeId));
      expect(g.nodeCount()).toEqual(expectedNodes.length);
      // Prefix 'onExit - ' is added to exit-handler nodes
      expect(g.node('on-exit').label).toEqual('onExit - ' + 'on-exit');
    });

    it('adds and connects nodes based on listed dependencies', () => {
      const workflow = newWorkflow();
      workflow.spec.templates[0].dag.tasks.push({ name: 'task-2', dependencies: ['task-1'] });
      const g = createGraph(workflow);
      expect(g.nodeCount()).toEqual(2);
      expect(g.edges()).toEqual([{ v: '/task-1', w: '/task-2' }]);
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
              ],
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
      expect(g.edgeCount()).toEqual(3);
      const expectedNodes = [
        '/task-1-1',
        '/task-1-1/task-2-1',
        '/task-1-1/task-2-2',
        '/task-1-1/task-2-2/task-3-1',
      ];
      expectedNodes.forEach(nodeId => expect(g.nodes()).toContain(nodeId));
      expect(g.nodeCount()).toEqual(expectedNodes.length);
    });

    it("uses task's template as node label", () => {
      const g = createGraph(nestedWorkflow);
      expect(g.edgeCount()).toEqual(3);
      const expectedNodes = [
        { id: '/task-1-1', label: 'dag-2' },
        { id: '/task-1-1/task-2-1', label: 'container-2-1' },
        { id: '/task-1-1/task-2-2', label: 'dag-3' },
        { id: '/task-1-1/task-2-2/task-3-1', label: 'container-3-1' },
      ];
      expectedNodes.forEach(idAndLabel =>
        expect(g.node(idAndLabel.id).label).toEqual(idAndLabel.label),
      );
      expect(g.nodeCount()).toEqual(expectedNodes.length);
    });

    it('connects conditional graph correctly', () => {
      const g = createGraph(newConditionalWorkflow());
      // 'flip' has two possible outcomes: 'condition-1' and 'condition-2'
      const expectedEdges = [
        { v: '/flip-task', w: '/condition-1-task' },
        { v: '/flip-task', w: '/condition-2-task' },
        { v: '/condition-1-task', w: '/condition-1-task/heads-task' },
        { v: '/condition-2-task', w: '/condition-2-task/tails-task' },
      ];
      expectedEdges.forEach(edge => expect(g.edges()).toContainEqual(edge));
      expect(g.edgeCount()).toEqual(expectedEdges.length);
      expect(g.nodeCount()).toEqual(5);
    });

    it('finds conditionals and colors them', () => {
      const g = createGraph(newConditionalWorkflow());
      g.nodes().forEach(nodeName => {
        const node = g.node(nodeName);
        if (nodeName === '/condition-1-task' || nodeName === '/condition-2-task') {
          expect(node.bgColor).toEqual('cornsilk');
        } else {
          expect(node.bgColor).toBeUndefined();
        }
      });
    });

    it('includes the conditional itself in the info of conditional nodes', () => {
      const g = createGraph(newConditionalWorkflow());
      g.nodes().forEach(nodeName => {
        const node = g.node(nodeName);
        if (nodeName.startsWith('condition-1')) {
          expect(node.info.condition).toEqual('{{inputs.parameters.flip-output}} == heads');
        } else if (nodeName.startsWith('condition-2')) {
          expect(node.info.condition).toEqual('{{inputs.parameters.flip-output}} == tails');
        }
      });
    });

    it("includes the resource's action and manifest itself in the info of resource nodes", () => {
      const g = createGraph(newResourceCreatingWorkflow());
      g.nodes().forEach(nodeName => {
        const node = g.node(nodeName);
        if (nodeName.startsWith('create-pvc')) {
          expect(node.info.resource).toEqual([
            ['create', 'apiVersion: v1\nkind: PersistentVolumeClaim'],
          ]);
        } else if (nodeName.startsWith('create-snapshot')) {
          expect(node.info.resource).toEqual([
            ['create', 'apiVersion: snapshot.storage.k8s.io\nkind: VolumeSnapshot'],
          ]);
        }
      });
    });

    it('renders extremely simple workflows with no steps or DAGs', () => {
      const simpleWorkflow = {
        spec: {
          entrypoint: 'template-1',
          templates: [{ container: {}, name: 'template-1' }],
        },
      } as any;
      const g = createGraph(simpleWorkflow);
      expect(g.nodes()).toEqual(['template-1']);
      expect(g.edgeCount()).toEqual(0);
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
      expect(g.nodes()).toEqual(['/pipeline-component']);
      expect(g.edgeCount()).toEqual(0);
    });

    it('throws an error if the workflow has no spec', () => {
      expect(() => createGraph({} as any)).toThrowError('Could not generate graph.');
    });

    it('throws an error message if the workflow spec has no templates', () => {
      expect(() => createGraph({} as any)).toThrowError('Could not generate graph.');
    });

    it('supports simple recursive workflows', () => {
      const workflow = {
        spec: {
          entrypoint: 'entrypoint-dag',
          templates: [
            {
              dag: { tasks: [{ name: 'start', template: 'start' }] },
              name: 'entrypoint-dag',
            },
            {
              dag: { tasks: [{ name: 'recurse', template: 'recurse' }] },
              name: 'start',
            },
            {
              dag: { tasks: [{ name: 'start', template: 'start' }] },
              name: 'recurse',
            },
          ],
        },
      } as any;
      const g = createGraph(workflow);
      const expectedNodes = ['/start', '/start/recurse'];
      expectedNodes.forEach(node => expect(g.nodes()).toContain(node));
      expect(g.nodeCount()).toEqual(expectedNodes.length);
      const expectedEdges = [
        { v: '/start', w: '/start/recurse' },
        { v: '/start/recurse', w: '/start' },
      ];
      expectedEdges.forEach(edge => expect(g.edges()).toContainEqual(edge));
      expect(g.edgeCount()).toEqual(expectedEdges.length);
    });

    it('supports recursive workflows with onExit defined', () => {
      const workflow = {
        spec: {
          entrypoint: 'entrypoint-dag',
          onExit: 'exit',
          templates: [
            {
              dag: { tasks: [{ name: 'start', template: 'start' }] },
              name: 'entrypoint-dag',
            },
            {
              dag: { tasks: [{ name: 'recurse', template: 'recurse' }] },
              name: 'start',
            },
            {
              dag: { tasks: [{ name: 'start', template: 'start' }] },
              name: 'recurse',
            },
            {
              container: {},
              name: 'exit',
            },
          ],
        },
      } as any;
      const g = createGraph(workflow);
      const expectedNodes = ['exit', '/start', '/start/recurse'];
      expectedNodes.forEach(node => expect(g.nodes()).toContain(node));
      expect(g.nodeCount()).toEqual(expectedNodes.length);
      const expectedEdges = [
        { v: '/start', w: '/start/recurse' },
        { v: '/start/recurse', w: '/start' },
      ];
      expectedEdges.forEach(edge => expect(g.edges()).toContainEqual(edge));
      expect(g.edgeCount()).toEqual(expectedEdges.length);
    });

    it('supports complex recursive workflows', () => {
      const workflow = {
        spec: {
          entrypoint: 'entrypoint-dag',
          templates: [
            {
              dag: { tasks: [{ name: 'start', template: 'start' }] },
              name: 'entrypoint-dag',
            },
            {
              dag: {
                tasks: [
                  { name: 'recurse-1', template: 'recurse-1' },
                  { name: 'leaf-1', template: 'leaf-1' },
                ],
              },
              name: 'start',
            },
            {
              dag: {
                tasks: [
                  { name: 'start', template: 'start' },
                  { name: 'recurse-2', template: 'recurse-2' },
                  { name: 'leaf-2', template: 'leaf-2' },
                ],
              },
              name: 'recurse-1',
            },
            {
              dag: {
                tasks: [
                  { name: 'start', template: 'start' },
                  { name: 'leaf-3', template: 'leaf-3' },
                  { name: 'leaf-4', template: 'leaf-4' },
                ],
              },
              name: 'recurse-2',
            },
            { container: {}, name: 'leaf-1' },
            { container: {}, name: 'leaf-2' },
            { container: {}, name: 'leaf-3' },
            { container: {}, name: 'leaf-4' },
          ],
        },
      } as any;
      const g = createGraph(workflow);
      const expectedNodes = [
        '/start',
        '/start/recurse-1',
        '/start/leaf-1',
        '/start/recurse-1/recurse-2',
        '/start/recurse-1/leaf-2',
        '/start/recurse-1/recurse-2/leaf-3',
        '/start/recurse-1/recurse-2/leaf-4',
      ];
      expectedNodes.forEach(node => expect(g.nodes()).toContain(node));
      expect(g.nodeCount()).toEqual(expectedNodes.length);
      const expectedEdges = [
        { v: '/start', w: '/start/recurse-1' },
        { v: '/start', w: '/start/leaf-1' },
        { v: '/start/recurse-1', w: '/start' },
        { v: '/start/recurse-1', w: '/start/recurse-1/recurse-2' },
        { v: '/start/recurse-1', w: '/start/recurse-1/leaf-2' },
        { v: '/start/recurse-1/recurse-2', w: '/start' },
        { v: '/start/recurse-1/recurse-2', w: '/start/recurse-1/recurse-2/leaf-3' },
        { v: '/start/recurse-1/recurse-2', w: '/start/recurse-1/recurse-2/leaf-4' },
      ];
      expectedEdges.forEach(edge => expect(g.edges()).toContainEqual(edge));
      expect(g.edgeCount()).toEqual(expectedEdges.length);
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
      const nodeInfo = _populateInfoFromTemplate(defaultSelectedNodeInfo, {
        name: 'template',
      } as any);
      expect(nodeInfo).toEqual(defaultSelectedNodeInfo);
    });

    it('returns nodeInfo of a container with empty values for args, command, image and/or volumeMounts if container does not have them', () => {
      const template = {
        container: {
          // No args
          // No command
          // No image
          // No volumeMounts
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.args).toEqual([]);
      expect(nodeInfo.command).toEqual([]);
      expect(nodeInfo.image).toEqual('');
      expect(nodeInfo.volumeMounts).toEqual([]);
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
      expect(nodeInfo.nodeType).toEqual('container');
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
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.command).toEqual(['echo', 'some-command']);
    });

    it('returns nodeInfo containing container image', () => {
      const template = {
        container: {
          image: 'some-image',
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.image).toEqual('some-image');
    });

    it('returns nodeInfo containing container volumeMounts', () => {
      const template = {
        container: {
          volumeMounts: [{ mountPath: '/some/path', name: 'some-vol' }],
        },
        dag: [],
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.volumeMounts).toEqual([['/some/path', 'some-vol']]);
    });

    it('returns nodeInfo of a resource with empty values for action and manifest', () => {
      const template = {
        dag: [],
        name: 'template-1',
        resource: {
          // No action
          // No manifest
        },
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.resource).toEqual([[]]);
    });

    it('returns nodeInfo containing resource action and manifest', () => {
      const template = {
        dag: [],
        name: 'template-1',
        resource: {
          action: 'create',
          manifest: 'manifest',
        },
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.resource).toEqual([['create', 'manifest']]);
    });

    it('returns nodeInfo of a container with empty values if template does not have inputs and/or outputs', () => {
      const template = {
        container: {},
        dag: [],
        // No inputs
        // No outputs
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.inputs).toEqual([[]]);
      expect(nodeInfo.outputs).toEqual([[]]);
    });

    it('returns nodeInfo of a container containing template inputs params as list of name/value tuples', () => {
      const template = {
        container: {},
        dag: [],
        inputs: {
          parameters: [
            { name: 'param1', value: 'val1' },
            { name: 'param2', value: 'val2' },
          ],
        },
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.inputs).toEqual([
        ['param1', 'val1'],
        ['param2', 'val2'],
      ]);
    });

    it('returns nodeInfo of a container with empty strings for inputs with no specified value', () => {
      const template = {
        container: {},
        dag: [],
        inputs: {
          parameters: [{ name: 'param1' }, { name: 'param2' }],
        },
        name: 'template-1',
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.inputs).toEqual([
        ['param1', ''],
        ['param2', ''],
      ]);
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
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.outputs).toEqual([
        ['param1', 'val1'],
        ['param2', 'jsonPath'],
        ['param3', 'path'],
        ['param4', 'parameterReference'],
        ['param5', 'jqFilter'],
      ]);
    });

    it('returns nodeInfo of a container with empty strings for outputs with no specified value', () => {
      const template = {
        container: {},
        name: 'template-1',
        outputs: {
          parameters: [{ name: 'param1' }, { name: 'param2' }],
        },
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('container');
      expect(nodeInfo.outputs).toEqual([
        ['param1', ''],
        ['param2', ''],
      ]);
    });

    it('returns nodeInfo of a resource with empty values if template does not have inputs and/or outputs', () => {
      const template = {
        dag: [],
        // No inputs
        // No outputs
        name: 'template-1',
        resource: {},
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.inputs).toEqual([[]]);
      expect(nodeInfo.outputs).toEqual([[]]);
    });

    it('returns nodeInfo of a resource containing template inputs params as list of name/value tuples', () => {
      const template = {
        dag: [],
        inputs: {
          parameters: [
            { name: 'param1', value: 'val1' },
            { name: 'param2', value: 'val2' },
          ],
        },
        name: 'template-1',
        resource: {},
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.inputs).toEqual([
        ['param1', 'val1'],
        ['param2', 'val2'],
      ]);
    });

    it('returns nodeInfo of a resource with empty strings for inputs with no specified value', () => {
      const template = {
        dag: [],
        inputs: {
          parameters: [{ name: 'param1' }, { name: 'param2' }],
        },
        name: 'template-1',
        resource: {},
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.inputs).toEqual([
        ['param1', ''],
        ['param2', ''],
      ]);
    });

    it('returns nodeInfo containing resource outputs as list of name/value tuples, pulling from valueFrom if necessary', () => {
      const template = {
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
        resource: {},
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.outputs).toEqual([
        ['param1', 'val1'],
        ['param2', 'jsonPath'],
        ['param3', 'path'],
        ['param4', 'parameterReference'],
        ['param5', 'jqFilter'],
      ]);
    });

    it('returns nodeInfo of a resource with empty strings for outputs with no specified value', () => {
      const template = {
        name: 'template-1',
        outputs: {
          parameters: [{ name: 'param1' }, { name: 'param2' }],
        },
        resource: {},
      } as any;
      const nodeInfo = _populateInfoFromTemplate(new SelectedNodeInfo(), template);
      expect(nodeInfo.nodeType).toEqual('resource');
      expect(nodeInfo.outputs).toEqual([
        ['param1', ''],
        ['param2', ''],
      ]);
    });
  });
});
