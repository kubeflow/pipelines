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
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { range } from 'lodash';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import '@testing-library/jest-dom';

class ExperimentListTest extends ExperimentList {
  public _loadExperiments(request: ListRequest): Promise<string> {
    return super._loadExperiments(request);
  }
}

describe('ExperimentList', () => {
  let onErrorSpy: jest.SpyInstance;
  let listExperimentsSpy: jest.SpyInstance;
  let getExperimentSpy: jest.SpyInstance;
  let formatDateStringSpy: jest.SpyInstance;
  let listRunsSpy: jest.SpyInstance;

  function generateProps(): ExperimentListProps {
    return {
      onError: onErrorSpy as any,
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
    onErrorSpy = jest.fn();
    listExperimentsSpy = jest.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
    getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
    formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');
    listRunsSpy = jest.spyOn(Apis.runServiceApiV2, 'listRuns');

    formatDateStringSpy.mockImplementation((date?: Date | string) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listExperimentsSpy.mockClear();
    getExperimentSpy.mockClear();
  });

  afterEach(async () => {
    jest.resetAllMocks();
  });

  it('renders the empty experience', () => {
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...generateProps()} />);
    expect(container).toMatchSnapshot();
  });

  it('renders the empty experience in ARCHIVED state', () => {
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);
    expect(container).toMatchSnapshot();
  });

  it('loads experiments whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.AVAILABLE;
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    // Create a test instance to access the protected method
    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({});

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
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({});

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
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({
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
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({});

    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(container).toMatchSnapshot();
  });

  it('reloads the experiment when refresh is called', async () => {
    mockNExperiments(0);
    const props = generateProps();

    // Create a ref to access the component instance
    const ref = React.createRef<ExperimentList>();
    const { container } = TestUtils.renderWithRouter(<ExperimentList ref={ref} {...props} />);

    // Wait for initial load
    await TestUtils.flushPromises();

    // Call refresh method
    if (ref.current) {
      await ref.current.refresh();
    }

    expect(listExperimentsSpy).toHaveBeenCalledTimes(2);
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
      '',
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(container).toMatchSnapshot();
  });

  it('loads multiple experiments', async () => {
    mockNExperiments(5);
    const props = generateProps();
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(container).toMatchSnapshot();
  });

  it('calls error callback when loading experiment fails', async () => {
    listExperimentsSpy.mockRejectedValueOnce(new Error('bad stuff happened'));

    const props = generateProps();
    const testInstance = new ExperimentListTest(props);

    await testInstance._loadExperiments({});

    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to list experiments: ',
      new Error('bad stuff happened'),
    );
  });

  it('loads runs for a given experiment id when it is expanded', async () => {
    listRunsSpy.mockImplementation(() =>
      Promise.resolve({
        runs: [],
        total_size: 0,
        next_page_token: '',
      }),
    );
    mockNExperiments(1);
    const props = generateProps();
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({});
    await TestUtils.flushPromises();

    expect(props.onError).not.toHaveBeenCalled();

    // Look for expand button
    const expandButton = container.querySelector('button[aria-label="Expand"]');
    if (expandButton) {
      fireEvent.click(expandButton);
      await waitFor(() => {
        expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1);
      });

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
    }
  });

  it('loads runs for a given experiment id with augumented storage state when it is expanded', async () => {
    listRunsSpy.mockImplementation(() =>
      Promise.resolve({
        runs: [],
        total_size: 0,
        next_page_token: '',
      }),
    );
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    const { container } = TestUtils.renderWithRouter(<ExperimentList {...props} />);

    const testInstance = new ExperimentListTest(props);
    await testInstance._loadExperiments({});
    await TestUtils.flushPromises();

    expect(props.onError).not.toHaveBeenCalled();

    // Look for expand button
    const expandButton = container.querySelector('button[aria-label="Expand"]');
    if (expandButton) {
      fireEvent.click(expandButton);
      await waitFor(() => {
        expect(Apis.runServiceApiV2.listRuns).toHaveBeenCalledTimes(1);
      });

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
    }
  });
});
