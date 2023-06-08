/*
 * Copyright 2020 The Kubeflow Authors
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
import * as Utils from 'src/lib/Utils';
import { ExperimentList, ExperimentListProps } from './ExperimentList';
import TestUtils from 'src/TestUtils';
import { V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { ExpandState } from './CustomTable';

import { Apis, ExperimentSortKeys, ListRequest } from 'src/lib/Apis';
import { ReactWrapper, ShallowWrapper, shallow } from 'enzyme';
import { range } from 'lodash';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';

class ExperimentListTest extends ExperimentList {
  public _loadExperiments(request: ListRequest): Promise<string> {
    return super._loadExperiments(request);
  }
}

describe('ExperimentList', () => {
  let tree: ShallowWrapper | ReactWrapper;

  const onErrorSpy = jest.fn();
  const listExperimentsSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');
  const listRunsSpy = jest.spyOn(Apis.runServiceApiV2, 'listRuns');

  function generateProps(): ExperimentListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function mockNExperiments(n: number): void {
    getExperimentSpy.mockImplementation(id =>
      Promise.resolve({
        experiment_id: 'testexperiment' + id,
        display_name: 'experiment with id: testexperiment' + id,
      }),
    );
    listExperimentsSpy.mockImplementation(() =>
      Promise.resolve({
        experiments: range(1, n + 1).map(i => {
          return {
            experiment_id: 'testexperiment' + i,
            display_name: 'experiment with id: testexperiment' + i,
          };
        }),
      }),
    );
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
    expect(shallow(<ExperimentList {...generateProps()} />)).toMatchSnapshot();
  });

  it('renders the empty experience in ARCHIVED state', () => {
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    expect(shallow(<ExperimentList {...props} />)).toMatchSnapshot();
  });

  it('loads experiments whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.AVAILABLE;
    tree = shallow(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2beta1PredicateOperation.NOTEQUALS,
              string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as V2beta1Filter),
      ),
      undefined,
    );
  });

  it('loads experiments whose storage state is ARCHIVED when storage state equals ARCHIVED', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    tree = shallow(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2beta1PredicateOperation.EQUALS,
              string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as V2beta1Filter),
      ),
      undefined,
    );
  });

  it('augments request filter with storage state predicates', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    tree = shallow(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({
      filter: encodeURIComponent(
        JSON.stringify({
          predicates: [{ key: 'k', op: 'op', string_value: 'val' }],
        }),
      ),
    });
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
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
              operation: V2beta1PredicateOperation.EQUALS,
              string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as V2beta1Filter),
      ),
      undefined,
    );
  });

  it('loads one experiment', async () => {
    mockNExperiments(1);
    const props = generateProps();
    tree = shallow(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
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
    tree = TestUtils.mountWithRouter(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentList).refresh();
    tree.update();
    expect(listExperimentsSpy).toHaveBeenCalledTimes(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
      '',
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('loads multiple experiments', async () => {
    mockNExperiments(5);
    const props = generateProps();
    tree = shallow(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree).toMatchSnapshot();
  });

  it('calls error callback when loading experiment fails', async () => {
    TestUtils.makeErrorResponseOnce(
      jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments'),
      'bad stuff happened',
    );
    const props = generateProps();
    tree = shallow(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to list experiments: ',
      new Error('bad stuff happened'),
    );
  });

  it('loads runs for a given experiment id when it is expanded', async () => {
    listRunsSpy.mockImplementation(() => {});
    mockNExperiments(1);
    const props = generateProps();
    tree = TestUtils.mountWithRouter(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    tree.update();
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.COLLAPSED,
        experiment_id: 'testexperiment1',
        display_name: 'experiment with id: testexperiment1',
      },
    ]);
    // Expand the first experiment
    tree
      .find('button[aria-label="Expand"]')
      .at(0)
      .simulate('click');
    await listRunsSpy;
    tree.update();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.EXPANDED,
        experiment_id: 'testexperiment1',
        display_name: 'experiment with id: testexperiment1',
      },
    ]);
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1);
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      undefined,
      'testexperiment1',
      '',
      10,
      'created_at desc',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2beta1PredicateOperation.NOTEQUALS,
              string_value: V2beta1RunStorageState.ARCHIVED.toString(),
            },
          ],
        } as V2beta1Filter),
      ),
    );
  });

  it('loads runs for a given experiment id with augumented storage state when it is expanded', async () => {
    listRunsSpy.mockImplementation(() => {});
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    tree = TestUtils.mountWithRouter(<ExperimentList {...props} />);
    await (tree.instance() as ExperimentListTest)._loadExperiments({});
    tree.update();
    expect(props.onError).not.toHaveBeenCalled();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.COLLAPSED,
        experiment_id: 'testexperiment1',
        display_name: 'experiment with id: testexperiment1',
      },
    ]);
    // Expand the first experiment
    tree
      .find('button[aria-label="Expand"]')
      .at(0)
      .simulate('click');
    await listRunsSpy;
    tree.update();
    expect(tree.state()).toHaveProperty('displayExperiments', [
      {
        expandState: ExpandState.EXPANDED,
        experiment_id: 'testexperiment1',
        display_name: 'experiment with id: testexperiment1',
      },
    ]);
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1);
    expect(Apis.runServiceApiV2.listRuns).toHaveBeenLastCalledWith(
      undefined,
      'testexperiment1',
      '',
      10,
      'created_at desc',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2beta1PredicateOperation.EQUALS,
              string_value: V2beta1ExperimentStorageState.ARCHIVED.toString(),
            },
          ],
        } as V2beta1Filter),
      ),
    );
  });
});
