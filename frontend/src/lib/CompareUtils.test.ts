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

import CompareUtils from './CompareUtils';
import { ApiRun } from '../apis/run';
import { Workflow } from 'third_party/argo-ui/argo_template';

describe('CompareUtils', () => {
  describe('getParamsCompareProps', () => {
    it('works with no runs', () => {
      expect(CompareUtils.getParamsCompareProps([], [])).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('works with one run', () => {
      const runs = [{ run: { name: 'run1' } }];
      const workflowObjects = [
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param1',
                  value: 'value1',
                },
              ],
            },
          },
        } as Workflow,
      ];
      expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual({
        rows: [['value1']],
        xLabels: ['run1'],
        yLabels: ['param1'],
      });
    });

    it('works with one run and several parameters', () => {
      const runs = [{ run: { name: 'run1' } }];
      const workflowObjects = [
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param1',
                  value: 'value1',
                },
                {
                  name: 'param2',
                  value: 'value2',
                },
                {
                  name: 'param3',
                  value: 'value3',
                },
              ],
            },
          },
        } as Workflow,
      ];
      expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual({
        rows: [['value1'], ['value2'], ['value3']],
        xLabels: ['run1'],
        yLabels: ['param1', 'param2', 'param3'],
      });
    });

    it('works with two runs and one parameter', () => {
      const runs = [{ run: { name: 'run1' } }, { run: { name: 'run2' } }];
      const workflowObjects = [
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param1',
                  value: 'value1',
                },
              ],
            },
          },
        } as Workflow,
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param1',
                  value: 'value2',
                },
              ],
            },
          },
        } as Workflow,
      ];
      expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual({
        rows: [['value1', 'value2']],
        xLabels: ['run1', 'run2'],
        yLabels: ['param1'],
      });
    });

    it('sorts on parameter occurrence frequency', () => {
      const runs = [
        { run: { name: 'run1' } },
        { run: { name: 'run2' } },
        { run: { name: 'run3' } },
      ];
      const workflowObjects = [
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param1',
                  value: 'value1',
                },
                {
                  name: 'param2',
                  value: 'value2',
                },
                {
                  name: 'param3',
                  value: 'value3',
                },
              ],
            },
          },
        } as Workflow,
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param2',
                  value: 'value4',
                },
                {
                  name: 'param3',
                  value: 'value5',
                },
              ],
            },
          },
        } as Workflow,
        {
          spec: {
            arguments: {
              parameters: [
                {
                  name: 'param3',
                  value: 'value6',
                },
                {
                  name: 'param4',
                  value: 'value7',
                },
              ],
            },
          },
        } as Workflow,
      ];
      expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual({
        rows: [
          ['value3', 'value5', 'value6'],
          ['value2', 'value4', ''],
          ['value1', '', ''],
          ['', '', 'value7'],
        ],
        xLabels: ['run1', 'run2', 'run3'],
        yLabels: ['param3', 'param2', 'param1', 'param4'],
      });
    });
  });

  describe('multiRunMetricsCompareProps', () => {
    it('returns empty props when passed no runs', () => {
      expect(CompareUtils.multiRunMetricsCompareProps([])).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('returns only x labels when passed a run with no metrics', () => {
      const runs = [{ name: 'run1' } as ApiRun];
      expect(CompareUtils.multiRunMetricsCompareProps(runs)).toEqual({
        rows: [],
        xLabels: ['run1'],
        yLabels: [],
      });
    });

    it('returns a row when passed a run with a metric', () => {
      const runs = [
        { name: 'run1', metrics: [{ name: 'some-metric', number_value: 0.33 }] } as ApiRun,
      ];
      expect(CompareUtils.multiRunMetricsCompareProps(runs)).toEqual({
        rows: [['0.330']],
        xLabels: ['run1'],
        yLabels: ['some-metric'],
      });
    });

    it('returns multiple rows when passed a run with multiple metrics', () => {
      const runs = [
        {
          metrics: [
            { name: 'some-metric', number_value: 0.33 },
            { name: 'another-metric', number_value: 0.554 },
          ],
          name: 'run1',
        } as ApiRun,
      ];
      expect(CompareUtils.multiRunMetricsCompareProps(runs)).toEqual({
        rows: [['0.330'], ['0.554']],
        xLabels: ['run1'],
        yLabels: ['some-metric', 'another-metric'],
      });
    });

    it('returns a row when passed multiple runs with the same metric', () => {
      const runs = [
        {
          metrics: [{ name: 'some-metric', number_value: 0.33 }],
          name: 'run1',
        } as ApiRun,
        {
          metrics: [{ name: 'some-metric', number_value: 0.66 }],
          name: 'run2',
        } as ApiRun,
      ];
      expect(CompareUtils.multiRunMetricsCompareProps(runs)).toEqual({
        rows: [['0.330', '0.660']],
        xLabels: ['run1', 'run2'],
        yLabels: ['some-metric'],
      });
    });

    it('returns multiple rows when passed multiple runs with the different metrics', () => {
      const runs = [
        {
          metrics: [{ name: 'some-metric', number_value: 0.33 }],
          name: 'run1',
        } as ApiRun,
        {
          metrics: [{ name: 'another-metric', number_value: 0.66 }],
          name: 'run2',
        } as ApiRun,
      ];
      expect(CompareUtils.multiRunMetricsCompareProps(runs)).toEqual({
        rows: [
          ['0.330', ''],
          ['', '0.660'],
        ],
        xLabels: ['run1', 'run2'],
        yLabels: ['some-metric', 'another-metric'],
      });
    });
  });

  describe('singleRunToMetricsCompareProps', () => {
    it('returns empty props when passed no run', () => {
      expect(CompareUtils.singleRunToMetricsCompareProps()).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('returns empty props when passed a run with no metrics', () => {
      const run = { name: 'run1' } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('ignores metrics with no name', () => {
      const run = {
        metrics: [{ node_id: 'some-node-id', number_value: 0.33 }],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('ignores metrics with no node ID', () => {
      const run = {
        metrics: [{ name: 'some-metric-name', number_value: 0.33 }],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('ignores metrics with no number value', () => {
      const run = {
        metrics: [{ name: 'some-metric-name', node_id: 'someNodeId' }],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('ignores metrics with NaN number values', () => {
      const run = {
        metrics: [{ name: 'some-metric-name', node_id: 'someNodeId', number_value: NaN }],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [],
        xLabels: [],
        yLabels: [],
      });
    });

    it('returns a row when run has a metric', () => {
      const run = {
        metrics: [{ name: 'some-metric-name', node_id: 'someNodeId', number_value: 0.23 }],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [['0.230']],
        xLabels: ['some-metric-name'],
        yLabels: ['someNodeId'],
      });
    });

    it("uses a node's display name if present in the workflow", () => {
      const run = {
        metrics: [{ name: 'some-metric-name', node_id: 'someNodeId', number_value: 0.23 }],
        name: 'run1',
      } as ApiRun;
      const workflow = {
        status: {
          nodes: {
            someNodeId: {
              displayName: 'some-display-name',
            },
          },
        },
      } as any;
      expect(CompareUtils.singleRunToMetricsCompareProps(run, workflow)).toEqual({
        rows: [['0.230']],
        xLabels: ['some-metric-name'],
        yLabels: ['some-display-name'],
      });
    });

    it('returns a single row when a run has a single step that produces multiple metrics', () => {
      const run = {
        metrics: [
          { name: 'some-metric-name', node_id: 'someNodeId', number_value: 0.23 },
          { name: 'another-metric-name', node_id: 'someNodeId', number_value: 0.54 },
        ],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [['0.230', '0.540']],
        xLabels: ['some-metric-name', 'another-metric-name'],
        yLabels: ['someNodeId'],
      });
    });

    it('returns multiple rows when a run has a multiple steps that produce one metric', () => {
      const run = {
        metrics: [
          { name: 'some-metric-name', node_id: 'someNodeId', number_value: 0.23 },
          { name: 'another-metric-name', node_id: 'anotherNodeId', number_value: 0.54 },
        ],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [
          ['0.230', ''],
          ['', '0.540'],
        ],
        xLabels: ['some-metric-name', 'another-metric-name'],
        yLabels: ['someNodeId', 'anotherNodeId'],
      });
    });

    it('returns a single column when a run has a multiple steps that produces the same metric', () => {
      const run = {
        metrics: [
          { name: 'some-metric-name', node_id: 'someNodeId', number_value: 0.23 },
          { name: 'some-metric-name', node_id: 'anotherNodeId', number_value: 0.54 },
        ],
        name: 'run1',
      } as ApiRun;
      expect(CompareUtils.singleRunToMetricsCompareProps(run)).toEqual({
        rows: [['0.230'], ['0.540']],
        xLabels: ['some-metric-name'],
        yLabels: ['someNodeId', 'anotherNodeId'],
      });
    });

    it('displays the correct row name when task_display_name annotation is present on the template', () => {
      const run = {
        metrics: [
          { name: 'some-metric-name', node_id: 'node-1', number_value: 0.12 },
          { name: 'some-metric-name', node_id: 'node-2', number_value: 0.34 },
        ],
      } as ApiRun;
      const workflow = {
        spec: {
          templates: [
            {
              name: 'template-1',
              metadata: {
                annotations: {
                  'pipelines.kubeflow.org/task_display_name': 'Some display annotation',
                },
              },
            },
            {
              name: 'template-2',
              metadata: {
                annotations: {},
              },
            },
          ],
        },
        status: {
          nodes: {
            'node-1': {
              id: 'node-1',
              displayName: 'node-1-display',
              templateName: 'template-1',
            },
            'node-2': {
              id: 'node-2',
              displayName: 'node-2-display',
              templateName: 'template-2',
            },
          },
        },
      } as any;
      expect(CompareUtils.singleRunToMetricsCompareProps(run, workflow)).toEqual({
        rows: [['0.120'], ['0.340']],
        xLabels: ['some-metric-name'],
        yLabels: ['Some display annotation', 'node-2-display'],
      });
    });
  });
});
