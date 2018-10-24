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

import { getNodeInputOutputParams, loadNodeOutputPaths, loadAllOutputPaths, parseStoragePath, StorageService, getOutboundNodes } from './WorkflowParser';

describe('WorkflowParser', () => {
  // TODO: createRuntimeGraph tests

  describe('getNodeInputOutputParams', () => {
    it('handles undefined workflow', () => {
      expect(getNodeInputOutputParams(undefined as any, '')).toEqual([[], []]);
    });

    it('handles empty workflow, without status', () => {
      expect(getNodeInputOutputParams({} as any, '')).toEqual([[], []]);
    });

    it('handles workflow without nodes', () => {
      const workflow = { status: {} };
      expect(getNodeInputOutputParams(workflow as any, '')).toEqual([[], []]);
    });

    it('handles node not existing in graph', () => {
      const workflow = { status: { nodes: { node1: {} } } };
      expect(getNodeInputOutputParams(workflow as any, 'node2')).toEqual([[], []]);
    });

    it('handles an empty node', () => {
      const workflow = { status: { nodes: { node1: {} } } };
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([[], []]);
    });

    it('handles a node with inputs but no parameters', () => {
      const workflow = { status: { nodes: { node1: { inputs: {} } } } };
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([[], []]);
    });

    it('handles a node with inputs and empty parameters', () => {
      const workflow = { status: { nodes: { node1: { inputs: { parameters: [] } } } } };
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([[], []]);
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
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
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
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
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
      expect(getNodeInputOutputParams(workflow as any, 'node2')).toEqual([
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
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
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
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
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
      expect(getNodeInputOutputParams(workflow as any, 'node1')).toEqual([
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
      expect(loadNodeOutputPaths(undefined as any)).toEqual([]);
    });

    it('handles an empty node', () => {
      expect(loadNodeOutputPaths({} as any)).toEqual([]);
    });

    it('handles a node with outputs but no artifacts', () => {
      expect(loadNodeOutputPaths({ outputs: {} } as any)).toEqual([]);
    });

    it('handles a node with outputs and empty artifacts', () => {
      expect(loadNodeOutputPaths({ outputs: { artifacts: [] } } as any)).toEqual([]);
    });

    it('handles a node with outputs and no-metadata artifacts', () => {
      expect(loadNodeOutputPaths({
        outputs: {
          artifacts: [{
            name: 'some other artifact',
          }]
        }
      } as any)).toEqual([]);
    });

    it('handles a node a malformed metadata artifact (no s3 field)', () => {
      expect(loadNodeOutputPaths({
        outputs: {
          artifacts: [{
            name: 'mlpipeline-ui-metadata',
          }]
        }
      } as any)).toEqual([]);
    });

    it('returns undefined bucket and key for a metadata artifact with empty s3 field', () => {
      expect(loadNodeOutputPaths({
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
      expect(loadNodeOutputPaths({
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
      expect(loadAllOutputPaths(undefined as any)).toEqual([]);
    });

    it('handle an empty workflow', () => {
      expect(loadAllOutputPaths({} as any)).toEqual([]);
    });

    it('handle an empty workflow status', () => {
      expect(loadAllOutputPaths({ status: {} } as any)).toEqual([]);
    });

    it('handle empty workflow nodes', () => {
      expect(loadAllOutputPaths({ status: { nodes: [] } } as any)).toEqual([]);
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
      expect(loadAllOutputPaths({ status: { nodes: { node1, node2 } } } as any)).toEqual([{
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
      expect(() => parseStoragePath('http://path')).toThrowError(
        'Unsupported storage path: http://path');
    });

    it('handles GCS bucket without key', () => {
      expect(parseStoragePath('gs://testbucket/')).toEqual({
        bucket: 'testbucket',
        key: '',
        source: StorageService.GCS,
      });
    });

    it('handles GCS bucket and key', () => {
      expect(parseStoragePath('gs://testbucket/testkey')).toEqual({
        bucket: 'testbucket',
        key: 'testkey',
        source: StorageService.GCS,
      });
    });

    it('handles GCS bucket and multi-part key', () => {
      expect(parseStoragePath('gs://testbucket/test/key/path')).toEqual({
        bucket: 'testbucket',
        key: 'test/key/path',
        source: StorageService.GCS,
      });
    });
  });

  describe('getOutboundNodes', () => {
    it('handles undefined workflow', () => {
      expect(getOutboundNodes(undefined as any, '')).toEqual([]);
    });

    it('handles an empty workflow', () => {
      expect(getOutboundNodes({} as any, '')).toEqual([]);
    });

    it('handles workflow without nodes', () => {
      expect(getOutboundNodes({ status: {} } as any, '')).toEqual([]);
    });

    it('handles node not in the workflow', () => {
      expect(getOutboundNodes({ status: { nodes: { node1: {} } } } as any, 'node2')).toEqual([]);
    });

    it('handles node with no outbound links', () => {
      expect(getOutboundNodes({
        status: { nodes: { node1: { outboundNodes: [] } } }
      } as any, 'node1')).toEqual([]);
    });

    it('returns the id of a Pod node as its only outbound link', () => {
      expect(getOutboundNodes({
        status:
          { nodes: { node1: { id: 'pod node id', outboundNodes: ['test node'], type: 'Pod' } } }
      } as any, 'node1')).toEqual(['pod node id']);
    });

    it('handles node with an outbound link to a non-existing node', () => {
      expect(getOutboundNodes({
        status:
          { nodes: { node1: { id: 'pod node id', outboundNodes: ['test node'] } } }
      } as any, 'node1')).toEqual([]);
    });

    it('returns the one Pod outbound node', () => {
      expect(getOutboundNodes({
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
      expect(getOutboundNodes({
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
      expect(getOutboundNodes({
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
      expect(getOutboundNodes({
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
});
