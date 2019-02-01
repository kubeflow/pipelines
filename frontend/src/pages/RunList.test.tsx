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
import * as Utils from '../lib/Utils';
import RunList, { RunListProps } from './RunList';
import TestUtils from '../TestUtils';
import produce from 'immer';
import { ApiRun, ApiRunDetail, ApiResourceType, ApiRunMetric, RunMetricFormat } from '../apis/run';
import { Apis, RunSortKeys, ListRequest } from '../lib/Apis';
import { MetricMetadata } from '../lib/RunUtils';
import { NodePhase } from './Status';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { range } from 'lodash';

class RunListTest extends RunList {
  public _loadRuns(request: ListRequest): Promise<string> {
    return super._loadRuns(request);
  }
}

describe('RunList', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();
  const listRunsSpy = jest.spyOn(Apis.runServiceApi, 'listRuns');
  const getRunSpy = jest.spyOn(Apis.runServiceApi, 'getRun');
  const getPipelineSpy = jest.spyOn(Apis.pipelineServiceApi, 'getPipeline');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  function generateProps(): RunListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function mockNRuns(n: number, runTemplate: Partial<ApiRunDetail>): void {
    getRunSpy.mockImplementation(id => Promise.resolve(
      produce(runTemplate, draft => {
        draft.pipeline_runtime = draft.pipeline_runtime || { workflow_manifest: '' };
        draft.run = draft.run || {};
        draft.run.id = id;
        draft.run.name = 'run with id: ' + id;
      })
    ));

    listRunsSpy.mockImplementation(() => Promise.resolve({
      runs: range(1, n + 1).map(i => ({
        id: 'testrun' + i,
        name: 'run with id: testrun' + i,
      } as ApiRun)),
    }));

    getPipelineSpy.mockImplementation(() => ({ name: 'some pipeline' }));
    getExperimentSpy.mockImplementation(() => ({ name: 'some experiment' }));
  }

  function getMountedInstance(): RunList {
    tree = TestUtils.mountWithRouter(<RunList {...generateProps()} />);
    return tree.instance() as RunList;
  }

  function getShallowInstance(): RunList {
    tree = shallow(<RunList {...generateProps()} />);
    return tree.instance() as RunList;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listRunsSpy.mockClear();
    getRunSpy.mockClear();
    getPipelineSpy.mockClear();
    getExperimentSpy.mockClear();
  });

  afterEach(async () => {
    // unmount() should be called before resetAllMocks() in case any part of the unmount life cycle
    // depends on mocks/spies
    if (tree) {
      await tree.unmount();
    }
    jest.resetAllMocks();
  });

  it('renders the empty experience', () => {
    expect(shallow(<RunList {...generateProps()} />)).toMatchSnapshot();
  });

  it('loads one run', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith(undefined, undefined, undefined, undefined, undefined, undefined);
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('reloads the run when refresh is called', async () => {
    mockNRuns(0, {});
    const props = generateProps();
    tree = TestUtils.mountWithRouter(<RunList {...props} />);
    await (tree.instance() as RunList).refresh();
    tree.update();
    expect(Apis.runServiceApi.listRuns).toHaveBeenCalledTimes(2);
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith('', 10, RunSortKeys.CREATED_AT + ' desc', undefined, undefined, '');
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads multiple runs', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('calls error callback when loading runs fails', async () => {
    TestUtils.makeErrorResponseOnce(jest.spyOn(Apis.runServiceApi, 'listRuns'), 'bad stuff happened');
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).toHaveBeenLastCalledWith('Error: failed to fetch runs.', new Error('bad stuff happened'));
  });

  it('displays error in run row if it failed to parse', async () => {
    mockNRuns(1, { pipeline_runtime: { workflow_manifest: 'bad json' } });
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(tree).toMatchSnapshot();
  });

  it('displays error in run row if pipeline could not be fetched', async () => {
    mockNRuns(1, { run: { pipeline_spec: { pipeline_id: 'test-pipeline-id' } } });
    TestUtils.makeErrorResponseOnce(getPipelineSpy, 'bad stuff happened');
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
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
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'bad stuff happened');
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(tree).toMatchSnapshot();
  });

  it('displays error in run row if it failed to parse (run list mask)', async () => {
    mockNRuns(2, { pipeline_runtime: { workflow_manifest: 'bad json' } });
    const props = generateProps();
    props.runIdListMask = ['testrun1', 'testrun2'];
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
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
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads runs for a given experiment id', async () => {
    mockNRuns(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith(
      undefined, undefined, undefined, ApiResourceType.EXPERIMENT.toString(), 'experiment1', undefined);
  });

  it('loads given list of runs only', async () => {
    mockNRuns(5, {});
    const props = generateProps();
    props.runIdListMask = ['run1', 'run2'];
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
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
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('shows pipeline name', async () => {
    mockNRuns(1, { run: { pipeline_spec: { pipeline_id: 'test-pipeline-id' } } });
    getPipelineSpy.mockImplementationOnce(() => ({ name: 'test pipeline' }));
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
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
    getExperimentSpy.mockImplementationOnce(() => ({ name: 'test experiment' }));
    const props = generateProps();
    tree = shallow(<RunList {...props} />);
    await (tree.instance() as RunListTest)._loadRuns({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('renders run name as link to its details page', () => {
    expect(getMountedInstance()._nameCustomRenderer({ value: 'test run', id: 'run-id' })).toMatchSnapshot();
  });

  it('renders pipeline name as link to its details page', () => {
    expect(getMountedInstance()._pipelineCustomRenderer({ value: { displayName: 'test pipeline', id: 'pipeline-id', showLink: false }, id: 'run-id' })).toMatchSnapshot();
  });

  it('handles no pipeline id given', () => {
    expect(getMountedInstance()._pipelineCustomRenderer({ value: { displayName: 'test pipeline', showLink: false }, id: 'run-id' })).toMatchSnapshot();
  });

  it('shows "View pipeline" button if pipeline is embedded in run', () => {
    expect(getMountedInstance()._pipelineCustomRenderer({ value: { displayName: 'test pipeline', id: 'pipeline-id', showLink: true }, id: 'run-id' })).toMatchSnapshot();
  });

  it('handles no pipeline name', () => {
    expect(getMountedInstance()._pipelineCustomRenderer({ value: { /* no displayName */ showLink: true }, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders experiment name as link to its details page', () => {
    expect(getMountedInstance()._experimentCustomRenderer({ value: { displayName: 'test experiment', id: 'experiment-id' }, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders no experiment name', () => {
    expect(getMountedInstance()._experimentCustomRenderer({ value: { /* no displayName */ id: 'experiment-id' }, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders status as icon', () => {
    expect(getShallowInstance()._statusCustomRenderer({ value: NodePhase.SUCCEEDED, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders metric buffer', () => {
    expect(getShallowInstance()._metricBufferCustomRenderer({ value: {}, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders an empty metric when there is no metric', () => {
    expect(getShallowInstance()._metricCustomRenderer({ value: undefined, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders an empty metric when metric is empty', () => {
    expect(getShallowInstance()._metricCustomRenderer({ value: {}, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders an empty metric when metric value is empty', () => {
    expect(getShallowInstance()._metricCustomRenderer({ value: { metric: {} }, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders a empty metric container when a metric has value of zero', () => {
    expect(getShallowInstance()._metricCustomRenderer({ value: { metric: { number_value: 0 } }, id: 'run-id' })).toMatchSnapshot();
  });

  it('renders a metric container when a percentage metric has value of zero', () => {
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: { metric: { number_value: 0, format: RunMetricFormat.PERCENTAGE } },
    })).toMatchSnapshot();
  });

  it('renders a metric container when a raw metric has value of zero', () => {
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: {
        metadata: { maxValue: 10, minValue: 0 } as any,
        metric: { number_value: 0, format: RunMetricFormat.RAW },
      },
    })).toMatchSnapshot();
  });

  it('renders percentage metric', () => {
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: { metric: { number_value: 0.3, format: RunMetricFormat.PERCENTAGE } },
    })).toMatchSnapshot();
  });

  it('renders raw metric', () => {
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: {
        metadata: { count: 1, maxValue: 100, minValue: 10 } as MetricMetadata,
        metric: { number_value: 55 } as ApiRunMetric,
      },
    })).toMatchSnapshot();
  });

  it('renders raw metric with zero max/min values', () => {
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: {
        metadata: { count: 1, maxValue: 0, minValue: 0 } as MetricMetadata,
        metric: { number_value: 15 } as ApiRunMetric,
      },
    })).toMatchSnapshot();
  });

  it('renders raw metric that is less than its min value, logs error to console', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: {
        metadata: { count: 1, maxValue: 100, minValue: 10 } as MetricMetadata,
        metric: { number_value: 5 } as ApiRunMetric,
      },
    })).toMatchSnapshot();
    expect(consoleSpy).toHaveBeenCalled();
  });

  it('renders raw metric that is greater than its max value, logs error to console', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    expect(getShallowInstance()._metricCustomRenderer({
      id: 'run-id',
      value: {
        metadata: { count: 1, maxValue: 100, minValue: 10 } as MetricMetadata,
        metric: { number_value: 105 } as ApiRunMetric,
      },
    })).toMatchSnapshot();
    expect(consoleSpy).toHaveBeenCalled();
  });

});
