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

import WorkflowParser, { StorageService } from './WorkflowParser';
import { NodePhase } from '../pages/Status';
import { color } from '../Css';
import { Constants } from './Constants';

describe('WorkflowParser', () => {
  describe('createRuntimeGraph', () => {
    it('handles an undefined workflow', () => {
      const g = WorkflowParser.createRuntimeGraph(undefined as any);
      expect(g.nodes()).toEqual([]);
      expect(g.edges()).toEqual([]);
    });

    it('handles an empty workflow', () => {
      const g = WorkflowParser.createRuntimeGraph({} as any);
      expect(g.nodes()).toEqual([]);
      expect(g.edges()).toEqual([]);
    });

    it('handles a workflow without nodes', () => {
      const g = WorkflowParser.createRuntimeGraph({ status: {} } as any);
      expect(g.nodes()).toEqual([]);
      expect(g.edges()).toEqual([]);
    });

    it('handles a workflow without a metadata', () => {
      const g = WorkflowParser.createRuntimeGraph({ status: { nodes: [{ key: 'value' }] } } as any);
      expect(g.nodes()).toEqual([]);
      expect(g.edges()).toEqual([]);
    });

    it('handles a workflow without a name', () => {
      const g = WorkflowParser.createRuntimeGraph({ status: { nodes: [{ key: 'value' }] }, metadata: {} } as any);
      expect(g.nodes()).toEqual([]);
      expect(g.edges()).toEqual([]);
    });

    it('creates a two-node graph', () => {
      const workflow = {
        metadata: { name: 'testWorkflow' },
        status: {
          nodes: {
            node1: {
              displayName: 'node1',
              id: 'node1',
              name: 'node1',
              outboundNodes: ['node2'],
              phase: 'Succeeded',
              type: 'Steps',
            },
            node2: {
              displayName: 'node2',
              id: 'node2',
              name: 'node2',
              phase: 'Succeeded',
              type: 'Pod',
            }
          },
        }
      };
      const g = WorkflowParser.createRuntimeGraph(workflow as any);
      expect(g.nodes()).toEqual(['node1', 'node2']);
      expect(g.edges()).toEqual([]);
    });

    it('creates graph with exit handler attached', () => {
      const workflow = {
        metadata: { name: 'virtualRoot' },
        status: {
          nodes: {
            node1: {
              displayName: 'node1',
              id: 'node1',
              name: 'node1',
              phase: 'Succeeded',
              type: 'Pod',
            },
            node2: {
              displayName: 'node2',
              id: 'node2',
              name: 'virtualRoot.onExit',
              phase: 'Succeeded',
              type: 'Pod',
            },
            virtualRoot: {
              displayName: 'virtualRoot',
              id: 'virtualRoot',
              name: 'virtualRoot',
              outboundNodes: ['node1'],
              phase: 'Succeeded',
              type: 'Steps',
            },
          },
        }
      };
      const g = WorkflowParser.createRuntimeGraph(workflow as any);
      expect(g.nodes()).toEqual(['node1', 'node2']);
      expect(g.edges()).toEqual([{ v: 'node1', w: 'node2' }]);
    });

    it('creates a graph with placeholder nodes for steps that are not finished', () => {
      const workflow = {
        metadata: { name: 'testWorkflow' },
        status: {
          nodes: {
            finishedNode: {
              displayName: 'finishedNode',
              id: 'finishedNode',
              name: 'finishedNode',
              phase: 'Succeeded',
              type: 'Pod',
            },
            pendingNode: {
              displayName: 'pendingNode',
              id: 'pendingNode',
              name: 'pendingNode',
              phase: 'Pending',
              type: 'Pod',
            },
            root: {
              children: ['pendingNode', 'runningNode', 'finishedNode'],
              displayName: 'root',
              id: 'root',
              name: 'root',
              phase: 'Succeeded',
              type: 'Pod',
            },
            runningNode: {
              displayName: 'runningNode',
              id: 'runningNode',
              name: 'runningNode',
              phase: 'Running',
              type: 'Pod',
            },
          },
        }
      };
      const g = WorkflowParser.createRuntimeGraph(workflow as any);
      expect(g.nodes()).toEqual([
        'finishedNode',
        'pendingNode',
        'pendingNode-running-placeholder',
        'root',
        'runningNode',
        'runningNode-running-placeholder'
      ]);
      expect(g.edges()).toEqual(expect.arrayContaining([
        { v: 'root', w: 'pendingNode' },
        { v: 'root', w: 'runningNode' },
        { v: 'root', w: 'finishedNode' },
        { v: 'pendingNode', w: 'pendingNode-running-placeholder' },
        { v: 'runningNode', w: 'runningNode-running-placeholder' },
      ]));
    });

    it('sets specific properties for placeholder nodes', () => {
      const workflow = {
        metadata: { name: 'testWorkflow' },
        status: {
          nodes: {
            root: {
              children: ['runningNode'],
              displayName: 'root',
              id: 'root',
              name: 'root',
              phase: 'Succeeded',
              type: 'Pod',
            },
            runningNode: {
              displayName: 'runningNode',
              id: 'runningNode',
              name: 'runningNode',
              phase: 'Running',
              type: 'Pod',
            },
          },
        }
      };
      const g = WorkflowParser.createRuntimeGraph(workflow as any);

      const runningNode = g.node('runningNode');
      expect(runningNode.height).toEqual(Constants.NODE_HEIGHT);
      expect(runningNode.width).toEqual(Constants.NODE_WIDTH);
      expect(runningNode.label).toEqual('runningNode');
      expect(runningNode.isPlaceholder).toBeUndefined();

      const placeholderNode = g.node('runningNode-running-placeholder');
      expect(placeholderNode.height).toEqual(28);
      expect(placeholderNode.width).toEqual(28);
      expect(placeholderNode.label).toBeUndefined();
      expect(placeholderNode.isPlaceholder).toBe(true);
    });

    it('sets extra properties for placeholder node edges', () => {
      const workflow = {
        metadata: { name: 'testWorkflow' },
        status: {
          nodes: {
            root: {
              children: ['runningNode'],
              displayName: 'root',
              id: 'root',
              name: 'root',
              phase: 'Succeeded',
              type: 'Pod',
            },
            runningNode: {
              displayName: 'runningNode',
              id: 'runningNode',
              name: 'runningNode',
              phase: 'Running',
              type: 'Pod',
            },
          },
        }
      };
      const g = WorkflowParser.createRuntimeGraph(workflow as any);

      g.edges().map(edgeInfo => g.edge(edgeInfo)).forEach(edge => {
        if (edge.isPlaceholder) {
          expect(edge.color).toEqual(color.weak);
        } else {
          expect(edge.color).toBeUndefined();
        }
      });

    });

    it('deletes virtual nodes', () => {
      const workflow = {
        metadata: { name: 'testWorkflow' },
        status: {
          nodes: {
            node1: {
              children: ['node2'],
              id: 'node1',
              name: 'node1',
              phase: 'Succeeded',
              type: 'Steps',
            },
            node2: {
              boundaryID: 'node2',
              children: ['node3'],
              id: 'node2',
              name: 'node2',
              phase: 'Succeeded',
              type: 'StepGroup',
            },
            node3: {
              id: 'node3',
              name: 'node3',
              phase: 'Succeeded',
              type: 'Pod',
            }
          },
        }
      };
      const g = WorkflowParser.createRuntimeGraph(workflow as any);
      expect(g.nodes()).toEqual(['node1', 'node3']);
      expect(g.edges()).toEqual([{ v: 'node1', w: 'node3' }]);
    });
  });

  describe('getNodeInputOutputParams', () => {
    it('handles undefined workflow', () => {
      expect(WorkflowParser.getNodeInputOutputParams(undefined as any, '')).toEqual([[], []]);
    });

    it('handles empty workflow, without status', () => {
      expect(WorkflowParser.getNodeInputOutputParams({} as any, '')).toEqual([[], []]);
    });

    it('handles workflow without nodes', () => {
      const workflow = { status: {} };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, '')).toEqual([[], []]);
    });

    it('handles node not existing in graph', () => {
      const workflow = { status: { nodes: { node1: {} } } };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node2')).toEqual([[], []]);
    });

    it('handles an empty node', () => {
      const workflow = { status: { nodes: { node1: {} } } };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([[], []]);
    });

    it('handles a node with inputs but no parameters', () => {
      const workflow = { status: { nodes: { node1: { inputs: {} } } } };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([[], []]);
    });

    it('handles a node with inputs and empty parameters', () => {
      const workflow = { status: { nodes: { node1: { inputs: { parameters: [] } } } } };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([[], []]);
    });

    it('handles a node with one input parameter', () => {
      const workflow = {
        status: {
          nodes: {
            node1: {
              inputs: {
                parameters: [{
                  name: 'input param1',
                  value: 'input param1 value'
                }]
              }
            }
          }
        }
      };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
        [
          ['input param1', 'input param1 value']
        ], []
      ]);
    });

    it('handles a node with one input parameter that has no value', () => {
      const workflow = {
        status: {
          nodes: {
            node1: {
              inputs: {
                parameters: [{
                  name: 'input param1',
                }]
              }
            }
          }
        }
      };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
        [
          ['input param1', '']
        ], []
      ]);
    });

    it('handles a node with one input parameter that is not the first node', () => {
      const workflow = {
        status: {
          nodes: {
            node1: {
              inputs: {
                parameters: [{
                  name: 'input param1',
                  value: 'input param1 value'
                }]
              }
            },
            node2: {
              inputs: {
                parameters: [{
                  name: 'node2 input param1',
                  value: 'node2 input param1 value'
                }]
              }
            }
          }
        }
      };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node2')).toEqual([
        [
          ['node2 input param1', 'node2 input param1 value']
        ], []
      ]);
    });

    it('handles a node with one output parameter', () => {
      const workflow = {
        status: {
          nodes: {
            node1: {
              outputs: {
                parameters: [{
                  name: 'output param1',
                  value: 'output param1 value'
                }]
              }
            }
          }
        }
      };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
        [],
        [
          ['output param1', 'output param1 value']
        ],
      ]);
    });

    it('handles a node with one input and one output parameter', () => {
      const workflow = {
        status: {
          nodes: {
            node1: {
              inputs: {
                parameters: [{
                  name: 'input param1',
                  value: 'input param1 value'
                }]
              },
              outputs: {
                parameters: [{
                  name: 'output param1',
                  value: 'output param1 value'
                }]
              },
            }
          }
        }
      };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
        [
          ['input param1', 'input param1 value']
        ],
        [
          ['output param1', 'output param1 value']
        ],
      ]);
    });

    it('handles a node with multiple input and output parameter', () => {
      const workflow = {
        status: {
          nodes: {
            node1: {
              inputs: {
                parameters: [{
                  name: 'input param1',
                  value: 'input param1 value'
                }, {
                  name: 'input param2',
                  value: 'input param2 value'
                }, {
                  name: 'input param3',
                  value: 'input param3 value'
                }],
              },
              outputs: {
                parameters: [{
                  name: 'output param1',
                  value: 'output param1 value'
                }, {
                  name: 'output param2',
                  value: 'output param2 value'
                }],
              },
            }
          }
        }
      };
      expect(WorkflowParser.getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
        [
          ['input param1', 'input param1 value'],
          ['input param2', 'input param2 value'],
          ['input param3', 'input param3 value'],
        ],
        [
          ['output param1', 'output param1 value'],
          ['output param2', 'output param2 value'],
        ],
      ]);
    });
  });

  describe('loadNodeOutputPaths', () => {
    it('handles an undefined node', () => {
      expect(WorkflowParser.loadNodeOutputPaths(undefined as any)).toEqual([]);
    });

    it('handles an empty node', () => {
      expect(WorkflowParser.loadNodeOutputPaths({} as any)).toEqual([]);
    });

    it('handles a node with outputs but no artifacts', () => {
      expect(WorkflowParser.loadNodeOutputPaths({ outputs: {} } as any)).toEqual([]);
    });

    it('handles a node with outputs and empty artifacts', () => {
      expect(WorkflowParser.loadNodeOutputPaths({ outputs: { artifacts: [] } } as any)).toEqual([]);
    });

    it('handles a node with outputs and no-metadata artifacts', () => {
      expect(WorkflowParser.loadNodeOutputPaths({
        outputs: {
          artifacts: [{
            name: 'some other artifact',
          }]
        }
      } as any)).toEqual([]);
    });

    it('handles a node a malformed metadata artifact (no s3 field)', () => {
      expect(WorkflowParser.loadNodeOutputPaths({
        outputs: {
          artifacts: [{
            name: 'mlpipeline-ui-metadata',
          }]
        }
      } as any)).toEqual([]);
    });

    it('returns undefined bucket and key for a metadata artifact with empty s3 field', () => {
      expect(WorkflowParser.loadNodeOutputPaths({
        outputs: {
          artifacts: [{
            name: 'mlpipeline-ui-metadata',
            s3: {},
          }]
        }
      } as any)).toEqual([{
        bucket: undefined,
        key: undefined,
        source: 'minio',
      }]);
    });

    it('returns the right bucket and key for a correct metadata artifact', () => {
      expect(WorkflowParser.loadNodeOutputPaths({
        outputs: {
          artifacts: [{
            name: 'mlpipeline-ui-metadata',
            s3: {
              bucket: 'test bucket',
              key: 'test key',
            },
          }]
        }
      } as any)).toEqual([{
        bucket: 'test bucket',
        key: 'test key',
        source: 'minio',
      }]);
    });
  });

  describe('loadAllOutputPaths', () => {
    it('handle an undefined workflow', () => {
      expect(WorkflowParser.loadAllOutputPaths(undefined as any)).toEqual([]);
    });

    it('handle an empty workflow', () => {
      expect(WorkflowParser.loadAllOutputPaths({} as any)).toEqual([]);
    });

    it('handle an empty workflow status', () => {
      expect(WorkflowParser.loadAllOutputPaths({ status: {} } as any)).toEqual([]);
    });

    it('handle empty workflow nodes', () => {
      expect(WorkflowParser.loadAllOutputPaths({ status: { nodes: [] } } as any)).toEqual([]);
    });

    it('loads output paths from all workflow nodes', () => {
      const node1 = {
        outputs: {
          artifacts: [{
            name: 'mlpipeline-ui-metadata',
            s3: {
              bucket: 'test bucket',
              key: 'test key',
            },
          }],
        },
      };
      const node2 = {
        outputs: {
          artifacts: [{
            name: 'mlpipeline-ui-metadata',
            s3: {
              bucket: 'test bucket2',
              key: 'test key2',
            },
          }],
        },
      };
      expect(WorkflowParser.loadAllOutputPaths({ status: { nodes: { node1, node2 } } } as any)).toEqual([{
        bucket: 'test bucket',
        key: 'test key',
        source: 'minio',
      }, {
        bucket: 'test bucket2',
        key: 'test key2',
        source: 'minio',
      }]);
    });
  });

  describe('parseStoragePath', () => {
    it('throws for unsupported protocol', () => {
      expect(() => WorkflowParser.parseStoragePath('http://path')).toThrowError(
        'Unsupported storage path: http://path');
    });

    it('handles GCS bucket without key', () => {
      expect(WorkflowParser.parseStoragePath('gs://testbucket/')).toEqual({
        bucket: 'testbucket',
        key: '',
        source: StorageService.GCS,
      });
    });

    it('handles GCS bucket and key', () => {
      expect(WorkflowParser.parseStoragePath('gs://testbucket/testkey')).toEqual({
        bucket: 'testbucket',
        key: 'testkey',
        source: StorageService.GCS,
      });
    });

    it('handles GCS bucket and multi-part key', () => {
      expect(WorkflowParser.parseStoragePath('gs://testbucket/test/key/path')).toEqual({
        bucket: 'testbucket',
        key: 'test/key/path',
        source: StorageService.GCS,
      });
    });

    it('handles Minio bucket without key', () => {
      expect(WorkflowParser.parseStoragePath('minio://testbucket/')).toEqual({
        bucket: 'testbucket',
        key: '',
        source: StorageService.MINIO,
      });
    });

    it('handles Minio bucket and key', () => {
      expect(WorkflowParser.parseStoragePath('minio://testbucket/testkey')).toEqual({
        bucket: 'testbucket',
        key: 'testkey',
        source: StorageService.MINIO,
      });
    });

    it('handles Minio bucket and multi-part key', () => {
      expect(WorkflowParser.parseStoragePath('minio://testbucket/test/key/path')).toEqual({
        bucket: 'testbucket',
        key: 'test/key/path',
        source: StorageService.MINIO,
      });
    });
  });

  describe('getOutboundNodes', () => {
    it('handles undefined workflow', () => {
      expect(WorkflowParser.getOutboundNodes(undefined as any, '')).toEqual([]);
    });

    it('handles an empty workflow', () => {
      expect(WorkflowParser.getOutboundNodes({} as any, '')).toEqual([]);
    });

    it('handles workflow without nodes', () => {
      expect(WorkflowParser.getOutboundNodes({ status: {} } as any, '')).toEqual([]);
    });

    it('handles node not in the workflow', () => {
      expect(WorkflowParser.getOutboundNodes({ status: { nodes: { node1: {} } } } as any, 'node2')).toEqual([]);
    });

    it('handles node with no outbound links', () => {
      expect(WorkflowParser.getOutboundNodes({
        status: { nodes: { node1: { outboundNodes: [] } } }
      } as any, 'node1')).toEqual([]);
    });

    it('returns the id of a Pod node as its only outbound link', () => {
      expect(WorkflowParser.getOutboundNodes({
        status:
          { nodes: { node1: { id: 'pod node id', outboundNodes: ['test node'], type: 'Pod' } } }
      } as any, 'node1')).toEqual(['pod node id']);
    });

    it('handles node with an outbound link to a non-existing node', () => {
      expect(WorkflowParser.getOutboundNodes({
        status:
          { nodes: { node1: { id: 'pod node id', outboundNodes: ['test node'] } } }
      } as any, 'node1')).toEqual([]);
    });

    it('returns the one Pod outbound node', () => {
      expect(WorkflowParser.getOutboundNodes({
        status:
        {
          nodes: {
            node1: { id: 'pod node id', outboundNodes: ['node2', 'node3'] },
            node2: { id: 'node2 id', type: 'Pod' }
          }
        }
      } as any, 'node1')).toEqual(['node2']);
    });

    it('returns all Pod outbound nodes', () => {
      expect(WorkflowParser.getOutboundNodes({
        status:
        {
          nodes: {
            node1: { id: 'pod node id', outboundNodes: ['node2', 'node3'] },
            node2: { id: 'node2 id', type: 'Pod' },
            node3: { id: 'node3 id', type: 'Pod' },
          }
        }
      } as any, 'node1')).toEqual(['node2', 'node3']);
    });

    it('returns all Pod outbound nodes', () => {
      expect(WorkflowParser.getOutboundNodes({
        status:
        {
          nodes: {
            node1: { id: 'pod node id', outboundNodes: ['node2', 'node3'] },
            node2: { id: 'node2 id' },
            node3: { id: 'node3 id', type: 'Pod' },
          }
        }
      } as any, 'node1')).toEqual(['node3']);
    });

    it('recursively returns Pod outbound nodes', () => {
      expect(WorkflowParser.getOutboundNodes({
        status:
        {
          nodes: {
            node1: { id: 'pod node id', outboundNodes: ['node2', 'node3'] },
            node2: { id: 'node2 id', outboundNodes: ['node4'] },
            node3: { id: 'node3 id', type: 'Pod' },
            node4: { id: 'node4 id', type: 'Pod' },
          }
        }
      } as any, 'node1')).toEqual(['node4', 'node3']);
    });
  });

  describe('getWorkflowError', () => {
    it('handles undefined workflow', () => {
      expect(WorkflowParser.getWorkflowError(undefined as any)).toEqual('');
    });

    it('handles empty workflow', () => {
      expect(WorkflowParser.getWorkflowError({} as any)).toEqual('');
    });

    it('handles empty status workflow', () => {
      expect(WorkflowParser.getWorkflowError({ status: {} } as any)).toEqual('');
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.SKIPPED, NodePhase.SUCCEEDED].map(phase => {
      it('returns empty string for workflow with no message and phase: ' + phase, () => {
        expect(WorkflowParser.getWorkflowError({ status: { phase } } as any)).toEqual('');
      });
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.SKIPPED, NodePhase.SUCCEEDED].map(phase => {
      it('returns empty string for workflow with a message and phase: ' + phase, () => {
        expect(WorkflowParser.getWorkflowError({ status: { message: 'woops!', phase } } as any)).toEqual('');
      });
    });

    [NodePhase.ERROR, NodePhase.FAILED].map(phase => {
      it('returns no error for workflow with no message and phase: ' + phase, () => {
        expect(WorkflowParser.getWorkflowError({
          status: {
            phase,
          },
        } as any)).toEqual('');
      });
    });

    [NodePhase.ERROR, NodePhase.FAILED].map(phase => {
      it('returns message string for workflow with a message and phase: ' + phase, () => {
        expect(WorkflowParser.getWorkflowError({
          status: {
            message: 'woops!',
            phase,
          },
        } as any)).toEqual('woops!');
      });
    });

  });
});
