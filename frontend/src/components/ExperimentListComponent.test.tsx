/*
 * Copyright 2020 Google LLC
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
import {ExperimentListComponent, ExperimentListComponentProps } from './ExperimentListComponent';
import TestUtils from '../TestUtils';
import produce from 'immer';
import { ApiFilter, PredicateOp } from '../apis/filter';
import {
  ApiRun,
//   ApiRunDetail,
//   ApiResourceType,
//   ApiRunMetric,
//   RunMetricFormat,
  RunStorageState,
} from '../apis/run';
import {
    ApiExperiment,
    // ApiResourceType,
    ExperimentStorageState,
} from '../apis/experiment';
import { ExpandState } from '../components/CustomTable';

import { Apis, ExperimentSortKeys, ListRequest } from '../lib/Apis';
import { MetricMetadata } from '../lib/RunUtils';
import { NodePhase } from '../lib/StatusUtils';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { range } from 'lodash';

class ExperimentListComponentTest extends ExperimentListComponent {
  public _loadExperiments(request: ListRequest): Promise<string> {
    return super._loadExperiments(request);
  }
}

describe('ExperimentListComponent', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();
  const listExperimentsSpy = jest.spyOn(Apis.experimentServiceApi, 'listExperiment');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');
  const listRunsSpy = jest.spyOn(Apis.runServiceApi, 'listRuns');

  function generateProps(): ExperimentListComponentProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function mockNExperiments(n: number): void {
    getExperimentSpy.mockImplementation(id =>
      Promise.resolve(
        {
          id: 'testexperiment' + id,
          name: 'experiment with id: testexperiment' + id,
        },
      ),
    );
    listExperimentsSpy.mockImplementation(() =>
      Promise.resolve({
        experiments: range(1, n + 1).map(i => {
            return {
              id: 'testexperiment' + i,
              name: 'experiment with id: testexperiment' + i,
            };
        }),
      }),
    );
  }

  function getMountedInstance(): ExperimentListComponent {
    tree = TestUtils.mountWithRouter(<ExperimentListComponent {...generateProps()} />);
    return tree.instance() as ExperimentListComponent;
  }

  function getShallowInstance(): ExperimentListComponent {
    tree = shallow(<ExperimentListComponent {...generateProps()} />);
    return tree.instance() as ExperimentListComponent;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listExperimentsSpy.mockClear();
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
    expect(shallow(<ExperimentListComponent {...generateProps()} />)).toMatchSnapshot();
  });

  it('renders the empty experience in ARCHIVED state', () => {
    const props = generateProps();
    props.storageState = ExperimentStorageState.ARCHIVED;
    expect(shallow(<ExperimentListComponent {...props} />)).toMatchSnapshot();
  });

  it('loads experiments whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = ExperimentStorageState.AVAILABLE;
    tree = shallow(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    expect(Apis.experimentServiceApi.listExperiment).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              op: PredicateOp.NOTEQUALS,
              string_value: ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as ApiFilter),
      ),
      undefined,
      undefined,
    );
  });

  it('loads experiments whose storage state is ARCHIVED when storage state equals ARCHIVED', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = ExperimentStorageState.ARCHIVED;
    tree = shallow(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    expect(Apis.experimentServiceApi.listExperiment).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              op: PredicateOp.EQUALS,
              string_value: ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as ApiFilter),
      ),
      undefined,
      undefined,
    );
  });

  it('augments request filter with storage state predicates', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = ExperimentStorageState.ARCHIVED;
    tree = shallow(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({
      filter: encodeURIComponent(
        JSON.stringify({
          predicates: [{ key: 'k', op: 'op', string_value: 'val' }],
        }),
      ),
    });
    expect(Apis.experimentServiceApi.listExperiment).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'k',
              op: 'op',
              string_value: 'val',
            },
            {
              key: 'storage_state',
              op: PredicateOp.EQUALS,
              string_value: ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as ApiFilter),
      ),
      undefined,
      undefined,
    );
  });

  it('loads one experiment', async () => {
    mockNExperiments(1);
    const props = generateProps();
    tree = shallow(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    expect(Apis.experimentServiceApi.listExperiment).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('reloads the experiment when refresh is called', async () => {
    mockNExperiments(0);
    const props = generateProps();
    tree = TestUtils.mountWithRouter(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponent).refresh();
    tree.update();
    expect(Apis.experimentServiceApi.listExperiment).toHaveBeenCalledTimes(2);
    expect(Apis.experimentServiceApi.listExperiment).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
      '',
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads multiple experiments', async () => {
    mockNExperiments(5);
    const props = generateProps();
    tree = shallow(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('calls error callback when loading experiment fails', async () => {
    TestUtils.makeErrorResponseOnce(
      jest.spyOn(Apis.experimentServiceApi, 'listExperiment'),
      'bad stuff happened',
    );
    const props = generateProps();
    tree = shallow(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to list experiments: ',
      new Error('bad stuff happened'),
    );
  });

  it('loads runs for a given experiment id when it is expanded', async () => {
    listRunsSpy.mockImplementation(() => {})
    mockNExperiments(1);
    const props = generateProps();
    // tree = shallow(<ExperimentListComponent {...props} />);
    tree = TestUtils.mountWithRouter(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    tree.update();
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.COLLAPSED,
        id: 'testexperiment1',
        name: 'experiment with id: testexperiment1',
      },
    ]);
    // Expand the first experiment
    tree.find('button[aria-label="Expand"]').at(0).simulate('click');
    await listRunsSpy;
    tree.update();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.EXPANDED,
        id: 'testexperiment1',
        name: 'experiment with id: testexperiment1',
      },
    ]);
    expect(Apis.runServiceApi.listRuns).toHaveBeenCalledTimes(1);
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith(
      "",
      10,
      "created_at desc",
      'EXPERIMENT',
      'testexperiment1',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              op: PredicateOp.NOTEQUALS,
              string_value: RunStorageState.ARCHIVED.toString(),
            },
          ],
        } as ApiFilter),
      ),
    );
  });

  it('loads runs for a given experiment id with augumented storage state when it is expanded', async () => {
    listRunsSpy.mockImplementation(() => {})
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = ExperimentStorageState.ARCHIVED;
    tree = TestUtils.mountWithRouter(<ExperimentListComponent {...props} />);
    await (tree.instance() as ExperimentListComponentTest)._loadExperiments({});
    tree.update();
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.COLLAPSED,
        id: 'testexperiment1',
        name: 'experiment with id: testexperiment1',
      },
    ]);
    // Expand the first experiment
    tree.find('button[aria-label="Expand"]').at(0).simulate('click');
    await listRunsSpy;
    tree.update();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.EXPANDED,
        id: 'testexperiment1',
        name: 'experiment with id: testexperiment1',
      },
    ]);
    expect(Apis.runServiceApi.listRuns).toHaveBeenCalledTimes(1);
    expect(Apis.runServiceApi.listRuns).toHaveBeenLastCalledWith(
      "",
      10,
      "created_at desc",
      'EXPERIMENT',
      'testexperiment1',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              op: PredicateOp.EQUALS,
              string_value: RunStorageState.ARCHIVED.toString(),
            },
          ],
        } as ApiFilter),
      ),
    );
  });
})