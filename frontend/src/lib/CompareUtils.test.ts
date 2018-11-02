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
import { Workflow } from 'third_party/argo-ui/argo_template';

describe('CompareUtils', () => {
  it('works with no runs', () => {
    expect(CompareUtils.getParamsCompareProps([], [])).toEqual(
      { rows: [], xLabels: [], yLabels: [] });
  });

  it('works with one run', () => {
    const runs = [{ run: { name: 'run1' } }];
    const workflowObjects = [{
      spec: {
        arguments: {
          parameters: [{
            name: 'param1',
            value: 'value1',
          }],
        },
      },
    } as Workflow];
    expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual(
      { rows: [['value1']], xLabels: ['run1'], yLabels: ['param1'] });
  });

  it('works with one run and several parameters', () => {
    const runs = [{ run: { name: 'run1' } }];
    const workflowObjects = [{
      spec: {
        arguments: {
          parameters: [{
            name: 'param1',
            value: 'value1',
          }, {
            name: 'param2',
            value: 'value2',
          }, {
            name: 'param3',
            value: 'value3',
          }],
        },
      },
    } as Workflow];
    expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual(
      {
        rows: [['value1'], ['value2'], ['value3']],
        xLabels: ['run1'],
        yLabels: ['param1', 'param2', 'param3'],
      });
  });

  it('works with two runs and one parameter', () => {
    const runs = [{ run: { name: 'run1' } }, { run: { name: 'run2' } }];
    const workflowObjects = [{
      spec: {
        arguments: {
          parameters: [{
            name: 'param1',
            value: 'value1',
          }],
        },
      },
    } as Workflow, {
      spec: {
        arguments: {
          parameters: [{
            name: 'param1',
            value: 'value2',
          }],
        },
      },
    } as Workflow];
    expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual(
      {
        rows: [['value1', 'value2']],
        xLabels: ['run1', 'run2'],
        yLabels: ['param1'],
      });
  });

  it('sorts on parameter occurrence frequency', () => {
    const runs = [{ run: { name: 'run1' } }, { run: { name: 'run2' } }, { run: { name: 'run3' } }];
    const workflowObjects = [{
      spec: {
        arguments: {
          parameters: [{
            name: 'param1',
            value: 'value1',
          }, {
            name: 'param2',
            value: 'value2',
          }, {
            name: 'param3',
            value: 'value3',
          }],
        },
      },
    } as Workflow, {
      spec: {
        arguments: {
          parameters: [{
            name: 'param2',
            value: 'value4',
          }, {
            name: 'param3',
            value: 'value5',
          }],
        },
      },
    } as Workflow, {
      spec: {
        arguments: {
          parameters: [{
            name: 'param3',
            value: 'value6',
          }, {
            name: 'param4',
            value: 'value7',
          }],
        },
      },
    } as Workflow];
    expect(CompareUtils.getParamsCompareProps(runs, workflowObjects)).toEqual(
      {
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
