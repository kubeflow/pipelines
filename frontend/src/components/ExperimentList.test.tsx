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
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import * as Utils from 'src/lib/Utils';
import { ExperimentList, ExperimentListProps } from './ExperimentList';
import { Apis, ExperimentSortKeys } from 'src/lib/Apis';
import { range } from 'lodash';
import { V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { V2beta1RunStorageState } from 'src/apisv2beta1/run';
import { V2beta1Filter, V2beta1PredicateOperation } from 'src/apisv2beta1/filter';
import TestUtils from 'src/TestUtils';

describe('ExperimentList', () => {
  const user = userEvent.setup();
  const onErrorSpy = vi.fn();
  const listExperimentsSpy = vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const listRunsSpy = vi.spyOn(Apis.runServiceApiV2, 'listRuns');
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
      experiments: range(1, n + 1).map((i) => ({
        experiment_id: 'testexperiment' + i,
        display_name: 'experiment with id: testexperiment' + i,
      })),
    });
  }

  function renderExperimentList(props: ExperimentListProps) {
    return render(
      <MemoryRouter>
        <ExperimentList {...props} />
      </MemoryRouter>,
    );
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    vi.clearAllMocks();
    listExperimentsSpy.mockResolvedValue({ experiments: [] });
    listRunsSpy.mockResolvedValue({ runs: [] });
  });

  it('renders the empty experience', async () => {
    renderExperimentList(generateProps());
    await screen.findByText('No archived experiments found.');
  });

  it('renders the empty experience in ARCHIVED state', async () => {
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.ARCHIVED;
    renderExperimentList(props);
    await screen.findByText('No archived experiments found.');
  });

  it('loads experiments whose storage state is not ARCHIVED when storage state equals AVAILABLE', async () => {
    mockNExperiments(1);
    const props = generateProps();
    props.storageState = V2beta1ExperimentStorageState.AVAILABLE;
    renderExperimentList(props);
    await screen.findByText('experiment with id: testexperiment1');
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
      encodeURIComponent(
        JSON.stringify({
          predicates: [
            {
              key: 'storage_state',
              operation: V2beta1PredicateOperation.NOT_EQUALS,
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
    renderExperimentList(props);
    await screen.findByText('experiment with id: testexperiment1');
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
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
    const ref = React.createRef<ExperimentList>();
    render(
      <MemoryRouter>
        <ExperimentList ref={ref} {...props} />
      </MemoryRouter>,
    );
    await screen.findByText('experiment with id: testexperiment1');
    listExperimentsSpy.mockClear();
    await act(async () => {
      await (ref.current as any)._loadExperiments({
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
    renderExperimentList(generateProps());
    await screen.findByText('experiment with id: testexperiment1');
    expect(onErrorSpy).not.toHaveBeenCalled();
  });

  it('reloads the experiment when refresh is called', async () => {
    mockNExperiments(0);
    const ref = React.createRef<ExperimentList>();
    const props = generateProps();
    render(
      <MemoryRouter>
        <ExperimentList ref={ref} {...props} />
      </MemoryRouter>,
    );
    await waitFor(() => expect(listExperimentsSpy).toHaveBeenCalled());
    listExperimentsSpy.mockClear();
    await act(async () => {
      await ref.current!.refresh();
    });
    expect(listExperimentsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      ExperimentSortKeys.CREATED_AT + ' desc',
      '',
      undefined,
    );
    expect(onErrorSpy).not.toHaveBeenCalled();
  });

  it('loads multiple experiments', async () => {
    mockNExperiments(5);
    renderExperimentList(generateProps());
    await screen.findByText('experiment with id: testexperiment5');
    expect(screen.getByText('experiment with id: testexperiment1')).toBeInTheDocument();
    expect(screen.getByText('experiment with id: testexperiment2')).toBeInTheDocument();
    expect(screen.getByText('experiment with id: testexperiment3')).toBeInTheDocument();
    expect(screen.getByText('experiment with id: testexperiment4')).toBeInTheDocument();
    expect(onErrorSpy).not.toHaveBeenCalled();
  });

  it('calls error callback when loading experiment fails', async () => {
    TestUtils.makeErrorResponseOnce(listExperimentsSpy as any, 'bad stuff happened');
    renderExperimentList(generateProps());
    await waitFor(() =>
      expect(onErrorSpy).toHaveBeenLastCalledWith(
        'Error: failed to list experiments: ',
        new Error('bad stuff happened'),
      ),
    );
  });

  it('loads runs for a given experiment id when it is expanded', async () => {
    mockNExperiments(1);
    renderExperimentList(generateProps());
    await screen.findByText('experiment with id: testexperiment1');
    listRunsSpy.mockClear();
    await user.click(screen.getAllByRole('button', { name: /expand/i })[0]);
    await waitFor(() => expect(listRunsSpy).toHaveBeenCalled());
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
              operation: V2beta1PredicateOperation.NOT_EQUALS,
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
    renderExperimentList(props);
    await screen.findByText('experiment with id: testexperiment1');
    listRunsSpy.mockClear();
    await user.click(screen.getAllByRole('button', { name: /expand/i })[0]);
    await waitFor(() => expect(listRunsSpy).toHaveBeenCalled());
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
