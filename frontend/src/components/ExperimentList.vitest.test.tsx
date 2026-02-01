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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import * as Utils from 'src/lib/Utils';
import { ExperimentList, ExperimentListProps } from './ExperimentList';
import { Apis, ExperimentSortKeys, ListRequest } from 'src/lib/Apis';
import { ExpandState } from './CustomTable';
import { range } from 'lodash';
import { V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import TestUtils from 'src/TestUtils';

type ExperimentListState = ExperimentList['state'];

class ExperimentListTest extends ExperimentList {
  public _loadExperiments(request: ListRequest): Promise<string> {
    return super._loadExperiments(request);
  }
}

class ExperimentListWrapper {
  private _instance: ExperimentListTest;
  private _renderResult: ReturnType<typeof render>;

  public constructor(instance: ExperimentListTest, renderResult: ReturnType<typeof render>) {
    this._instance = instance;
    this._renderResult = renderResult;
  }

  public instance(): ExperimentListTest {
    return this._instance;
  }

  public state<K extends keyof ExperimentListState>(
    key?: K,
  ): ExperimentListState | ExperimentListState[K] {
    const state = this._instance.state;
    return key ? state[key] : state;
  }

  public unmount(): void {
    this._renderResult.unmount();
  }
}

describe('ExperimentList', () => {
  let tree: ExperimentListWrapper | null = null;
  let renderResult: ReturnType<typeof render> | null = null;

  const onErrorSpy = vi.fn();
  const listExperimentsSpy = vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const listRunsSpy = vi.spyOn(Apis.runServiceApiV2, 'listRuns');
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test environments.
  const formatDateStringSpy = vi.spyOn(Utils, 'formatDateString');

  function generateProps(): ExperimentListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
    };
  }

  function mockNExperiments(n: number): void {
    listExperimentsSpy.mockResolvedValue({
      experiments: range(1, n + 1).map(i => ({
        experiment_id: 'testexperiment' + i,
        display_name: 'experiment with id: testexperiment' + i,
      })),
    });
  }

  async function renderExperimentList(props: ExperimentListProps): Promise<ExperimentListWrapper> {
    const experimentListRef = React.createRef<ExperimentListTest>();
    renderResult = render(
      <MemoryRouter>
        <ExperimentListTest ref={experimentListRef} {...props} />
      </MemoryRouter>,
    );
    if (!experimentListRef.current) {
      throw new Error('ExperimentList instance not available');
    }
    tree = new ExperimentListWrapper(experimentListRef.current, renderResult);
    await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalled());
    return tree;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    vi.clearAllMocks();
    listExperimentsSpy.mockResolvedValue({ experiments: [] });
    listRunsSpy.mockResolvedValue({ runs: [] });
  });

  afterEach(() => {
    tree?.unmount();
    tree = null;
    renderResult = null;
  });

  it('renders the empty experience', async () => {
    await renderExperimentList(generateProps());
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders the empty experience in ARCHIVED state', async () => {
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    await renderExperimentList(props);
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('loads experiments whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.AVAILABLE;
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
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
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
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
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({
        filter: encodeURIComponent(
          JSON.stringify({
            predicates: [{ key: 'k', op: 'op', string_value: 'val' }],
          }),
        ),
      });
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
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
    await waitFor(() => expect(wrapper.state('displayExperiments')).toHaveLength(1));
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('reloads the experiment when refresh is called', async () => {
    mockNExperiments(0);
    const props = generateProps();
    const wrapper = await renderExperimentList(props);
    await act(async () => {
      await wrapper.instance().refresh();
    });
    await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalledTimes(2));
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
      '',
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('loads multiple experiments', async () => {
    mockNExperiments(5);
    const props = generateProps();
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
    await waitFor(() => expect(wrapper.state('displayExperiments')).toHaveLength(5));
    expect(props.onError).not.toHaveBeenCalled();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('calls error callback when loading experiment fails', async () => {
    const props = generateProps();
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    TestUtils.makeErrorResponseOnce(listExperimentsSpy as any, 'bad stuff happened');
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to list experiments: ',
      new Error('bad stuff happened'),
    );
  });

  it('loads runs for a given experiment id when it is expanded', async () => {
    mockNExperiments(1);
    const props = generateProps();
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
    await waitFor(() =>
      expect(wrapper.state('displayExperiments')).toEqual([
        {
          expandState: ExpandState.COLLAPSED,
          experiment_id: 'testexperiment1',
          display_name: 'experiment with id: testexperiment1',
        },
      ]),
    );
    fireEvent.click(screen.getAllByLabelText('Expand')[0]);
    await waitFor(() =>
      expect(wrapper.state('displayExperiments')).toEqual([
        {
          expandState: ExpandState.EXPANDED,
          experiment_id: 'testexperiment1',
          display_name: 'experiment with id: testexperiment1',
        },
      ]),
    );
    await waitFor(() => expect(listRunsSpy).toHaveBeenCalledTimes(1));
    expect(listRunsSpy).toHaveBeenLastCalledWith(
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

  it('loads runs for a given experiment id with augmented storage state when it is expanded', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    const wrapper = await renderExperimentList(props);
    listExperimentsSpy.mockClear();
    await act(async () => {
      await wrapper.instance()._loadExperiments({});
    });
    await waitFor(() =>
      expect(wrapper.state('displayExperiments')).toEqual([
        {
          expandState: ExpandState.COLLAPSED,
          experiment_id: 'testexperiment1',
          display_name: 'experiment with id: testexperiment1',
        },
      ]),
    );
    fireEvent.click(screen.getAllByLabelText('Expand')[0]);
    await waitFor(() =>
      expect(wrapper.state('displayExperiments')).toEqual([
        {
          expandState: ExpandState.EXPANDED,
          experiment_id: 'testexperiment1',
          display_name: 'experiment with id: testexperiment1',
        },
      ]),
    );
    await waitFor(() => expect(listRunsSpy).toHaveBeenCalledTimes(1));
    expect(listRunsSpy).toHaveBeenLastCalledWith(
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
