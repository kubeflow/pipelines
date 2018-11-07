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

import * as React from 'react';
import RunList, { RunListProps } from './RunList';
import TestUtils from '../TestUtils';
import produce from 'immer';
import { ApiRun, ApiRunDetail, ApiResourceType, ApiRunMetric, RunMetricFormat } from '../apis/run';
import { Apis, RunSortKeys } from '../lib/Apis';
import { MetricMetadata } from 'src/lib/RunUtils';
import { NodePhase } from './Status';
import { range } from 'lodash';
import { shallow } from 'enzyme';

describe('RunList', () => {
  const onErrorSpy = jest.fn();
  const listRunsMock = jest.spyOn(Apis.runServiceApi, 'listRuns');
  const getRunMock = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getPipelineMock = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getExperimentMock = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');

  function generateProps(): RunListProps {
    return {
      history: {} as any,
      location: '' as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function mockNRuns(n: number, runTemplate: Partial<ApiRunDetail>) {
    getRunMock.mockImplementation(id => Promise.resolve(
      produce(runTemplate, draft => {
        draft.pipeline_runtime = draft.pipeline_runtime || { workflow_manifest: '' };
        draft.run = draft.run || {};
        draft.run.id = id;
        draft.run.name = 'run with id: ' + id;
      })
    ));

    listRunsMock.mockImplementation(() => Promise.resolve({
      runs: range(1, n + 1).map(i => ({
        id: 'testrun' + i,
        name: 'run with id: testrun' + i,
      } as ApiRun)),
    }));

    getPipelineMock.mockImplementation(() => ({ name: 'some pipeline' }));
    getExperimentMock.mockImplementation(() => ({ name: 'some experiment' }));
  }

  beforeEach(() => {
    onErrorSpy.mockClear();
    listRunsMock.mockClear();
    getRunMock.mockClear();
    getPipelineMock.mockClear();
    getExperimentMock.mockClear();
  });

  it('renders the empty experience', () => {
    expect(shallow(<RunList {...generateProps()} />)).toMatchSnapshot();
  });

  it('loads one run', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith(undefined, undefined, undefined, undefined, undefined);
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('reloads the run when refresh is called', async () => {
    mockNRuns(0, {});
    const props = generateProps();
    const tree = TestUtils.mountWithRouter(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith('', 10, RunSortKeys.CREATED_AT + ' desc', undefined, undefined);
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads multiple run', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('calls error callback when loading runs fails', async () => {
    TestUtils.makeErrorResponseOnce(jest.spyOn(Apis.runServiceApi, 'listRuns'), 'bad stuff happened');
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).toHaveBeenLastCalledWith('Error: failed to fetch runs.', new Error('bad stuff happened'));
  });

  it('displays error in run row if it failed to parse', async () => {
    mockNRuns(1, { pipeline_runtime: { workflow_manifest: 'bad json' } });
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(tree).toMatchSnapshot();
  });

  it('displays error in run row if pipeline could not be fetched', async () => {
    mockNRuns(1, { run: { pipeline_spec: { pipeline_id: 'test-pipeline-id' } } });
    TestUtils.makeErrorResponseOnce(getPipelineMock, 'bad stuff happened');
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(tree).toMatchSnapshot();
  });

  it('displays error in run row if experiment could not be fetched', async () => {
    mockNRuns(1, {
      run: {
        resource_references: [{
          key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT }
        }]
      }
    });
    TestUtils.makeErrorResponseOnce(getExperimentMock, 'bad stuff happened');
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(tree).toMatchSnapshot();
  });

  it('displays error in run row if it failed to parse (run list mask)', async () => {
    mockNRuns(2, { pipeline_runtime: { workflow_manifest: 'bad json' } });
    const props = generateProps();
    props.runIdListMask = ['testrun1', 'testrun2'];
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(tree).toMatchSnapshot();
  });

  it('shows run time for each run', async () => {
    mockNRuns(1, {
      pipeline_runtime: {
        workflow_manifest: JSON.stringify({
          status: {
            finishedAt: new Date(2018, 10, 10, 11, 11, 11),
            phase: 'Succeeded',
            startedAt: new Date(2018, 10, 10, 10, 10, 10),
          }
        }),
      },
    });
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads runs for a given experiment id', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith(
      undefined, undefined, undefined, ApiResourceType.EXPERIMENT.toString(), 'experiment1');
  });

  it('loads given list of runs only', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['run1', 'run2'];
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApi.listRuns).not.toHaveBeenCalled();
    expect(Apis.runServiceApi.getRun).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApi.getRun).toHaveBeenCalledWith('run1');
    expect(Apis.runServiceApi.getRun).toHaveBeenCalledWith('run2');
  });

  it('adds metrics columns', async () => {
    mockNRuns(2, {
      pipeline_runtime: {
        workflow_manifest: JSON.stringify({
          status: {
            phase: 'Succeeded',
          }
        }),
      },
      run: {
        metrics: [
          { name: 'metric1', number_value: 5 },
          { name: 'metric2', number_value: 10 },
        ],
      }
    });
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('shows pipeline name', async () => {
    mockNRuns(1, { run: { pipeline_spec: { pipeline_id: 'test-pipeline-id' } } });
    getPipelineMock.mockImplementationOnce(() => ({ name: 'test pipeline' }));
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('shows experiment name', async () => {
    mockNRuns(1, {
      run: {
        resource_references: [{
          key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT }
        }]
      }
    });
    getExperimentMock.mockImplementationOnce(() => {
      return ({ name: 'test experiment' });
    });
    const props = generateProps();
    const tree = shallow(<RunList {...props} />);
    await (tree.instance() as any)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('renders run name as link to its details page', () => {
    const tree = TestUtils.mountWithRouter((RunList.prototype as any)
      ._nameCustomRenderer('test run', 'run-id'));
    expect(tree).toMatchSnapshot();
  });

  it('renders pipeline name as link to its details page', () => {
    const tree = TestUtils.mountWithRouter((RunList.prototype as any)
      ._pipelineCustomRenderer({ displayName: 'test pipeline', id: 'pipeline-id' }));
    expect(tree).toMatchSnapshot();
  });

  it('renders no pipeline name', () => {
    const tree = TestUtils.mountWithRouter((RunList.prototype as any)
      ._pipelineCustomRenderer());
    expect(tree).toMatchSnapshot();
  });

  it('renders experiment name as link to its details page', () => {
    const tree = TestUtils.mountWithRouter((RunList.prototype as any)
      ._experimentCustomRenderer({ displayName: 'test experiment', id: 'experiment-id' }));
    expect(tree).toMatchSnapshot();
  });

  it('renders no experiment name', () => {
    const tree = TestUtils.mountWithRouter((RunList.prototype as any)
      ._experimentCustomRenderer());
    expect(tree).toMatchSnapshot();
  });

  it('renders status as icon', () => {
    const tree = shallow((RunList.prototype as any)._statusCustomRenderer(NodePhase.SUCCEEDED));
    expect(tree).toMatchSnapshot();
  });

  it('renders metric buffer', () => {
    const tree = shallow((RunList.prototype as any)._metricBufferCustomRenderer());
    expect(tree).toMatchSnapshot();
  });

  it('renders an empty metric when there is no data', () => {
    const noMetricTree = shallow((RunList.prototype as any)._metricCustomRenderer());
    expect(noMetricTree).toMatchSnapshot();

    const emptyMetricTree = shallow((RunList.prototype as any)._metricCustomRenderer({}));
    expect(emptyMetricTree).toMatchSnapshot();

    const noMetricValueTree = shallow((RunList.prototype as any)._metricCustomRenderer({ metric: {} }));
    expect(noMetricValueTree).toMatchSnapshot();
  });

  it('renders a empty metric container when a metric has value of zero', () => {
    const noMetricTree = shallow((RunList.prototype as any)._metricCustomRenderer(
      { metric: { number_value: 0 } }));
    expect(noMetricTree).toMatchSnapshot();
  });

  it('renders percentage metric', () => {
    const tree = shallow((RunList.prototype as any)._metricCustomRenderer(
      { metric: { number_value: 0.3, format: RunMetricFormat.PERCENTAGE } as ApiRunMetric }));
    expect(tree).toMatchSnapshot();
  });

  it('renders raw metric', () => {
    const tree = shallow((RunList.prototype as any)._metricCustomRenderer(
      {
        metadata: { count: 1, maxValue: 100, minValue: 10 } as MetricMetadata,
        metric: { number_value: 55 } as ApiRunMetric,
      }));
    expect(tree).toMatchSnapshot();
  });

  it('renders raw metric with zero max/min values', () => {
    const tree = shallow((RunList.prototype as any)._metricCustomRenderer(
      {
        metadata: { count: 1, maxValue: 0, minValue: 0 } as MetricMetadata,
        metric: { number_value: 15 } as ApiRunMetric,
      }));
    expect(tree).toMatchSnapshot();
  });

  it('renders raw metric that is less than its min value', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    const tree = shallow((RunList.prototype as any)._metricCustomRenderer(
      {
        metadata: { count: 1, maxValue: 100, minValue: 10 } as MetricMetadata,
        metric: { number_value: 5 } as ApiRunMetric,
      }));
    expect(tree).toMatchSnapshot();
    expect(consoleSpy).toHaveBeenCalled();
  });

  it('renders raw metric that is greater than its max value', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    const tree = shallow((RunList.prototype as any)._metricCustomRenderer(
      {
        metadata: { count: 1, maxValue: 100, minValue: 10 } as MetricMetadata,
        metric: { number_value: 105 } as ApiRunMetric,
      }));
    expect(tree).toMatchSnapshot();
    expect(consoleSpy).toHaveBeenCalled();
  });

});
