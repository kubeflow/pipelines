/*
 * Copyright 2021 Arrikto Inc.
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
import { act, render } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import produce from 'immer';
import { range } from 'lodash';
import * as Utils from 'src/lib/Utils';
import TestUtils from 'src/TestUtils';
import { Apis, JobSortKeys, ListRequest } from 'src/lib/Apis';
import RecurringRunList, { RecurringRunListProps } from './RecurringRunList';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { color } from 'src/Css';

let lastCustomTableProps: any = null;

vi.mock('src/components/CustomTable', () => {
  return {
    __esModule: true,
    Column: {},
    Row: {},
    CustomRendererProps: {},
    default: React.forwardRef((props: any, ref) => {
      lastCustomTableProps = props;
      React.useImperativeHandle(ref, () => ({
        reload: async () => {
          const sortBy = props.initialSortColumn ? `${props.initialSortColumn} desc` : '';
          return props.reload({
            pageToken: '',
            pageSize: 10,
            sortBy,
            filter: '',
          });
        },
      }));
      return <div data-testid='custom-table' />;
    }),
  };
});

describe('RecurringRunList', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let recurringRunListRef: React.RefObject<RecurringRunList> | null = null;

  const onErrorSpy = vi.fn();
  const listRecurringRunsSpy = vi.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns');
  const getRecurringRunSpy = vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
  const listExperimentsSpy = vi.spyOn(Apis.experimentServiceApiV2, 'listExperiments');
  const formatDateStringSpy = vi.spyOn(Utils, 'formatDateString');

  function generateProps(): RecurringRunListProps {
    return {
      history: {} as any,
      location: { search: '' } as any,
      match: '' as any,
      onError: onErrorSpy,
      refreshCount: 1,
    };
  }

  function renderRecurringRunList(propsPatch: Partial<RecurringRunListProps> = {}): void {
    recurringRunListRef = React.createRef<RecurringRunList>();
    const props = { ...generateProps(), ...propsPatch } as RecurringRunListProps;
    renderResult = render(
      <MemoryRouter>
        <RecurringRunList ref={recurringRunListRef} {...props} />
      </MemoryRouter>,
    );
  }

  async function loadRecurringRuns(request: ListRequest = {}): Promise<void> {
    await act(async () => {
      await (recurringRunListRef!.current as any)._loadRecurringRuns(request);
    });
  }

  function mockNRecurringRuns(n: number, recurringRunTemplate: Partial<V2beta1RecurringRun>): void {
    getRecurringRunSpy.mockImplementation(id =>
      Promise.resolve(
        produce(recurringRunTemplate, draft => {
          draft.recurring_run_id = id;
          draft.display_name = 'recurring run with id: ' + id;
        }),
      ),
    );

    listRecurringRunsSpy.mockImplementation(() =>
      Promise.resolve({
        recurringRuns: range(1, n + 1).map(i =>
          produce(recurringRunTemplate as Partial<V2beta1RecurringRun>, draft => {
            draft.recurring_run_id = 'testrecurringrun' + i;
            draft.display_name = 'recurring run with id: testrecurringrun' + i;
          }),
        ) as V2beta1RecurringRun[],
      }),
    );

    listExperimentsSpy.mockResolvedValue({ experiments: [] });
  }

  function getInstance(): RecurringRunList {
    if (!recurringRunListRef?.current) {
      throw new Error('RecurringRunList instance not available');
    }
    return recurringRunListRef.current;
  }

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
    onErrorSpy.mockClear();
    listRecurringRunsSpy.mockReset();
    getRecurringRunSpy.mockReset();
    listExperimentsSpy.mockReset();
    lastCustomTableProps = null;
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    recurringRunListRef = null;
    vi.resetAllMocks();
  });

  it('renders the empty experience', () => {
    renderRecurringRunList();
    expect(lastCustomTableProps.rows).toEqual([]);
    expect(lastCustomTableProps.emptyMessage).toBe('No available recurring runs found.');
  });

  it('loads one recurring run', async () => {
    mockNRecurringRuns(1, {});
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
    );
    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.rows).toEqual([
      {
        error: undefined,
        id: 'testrecurringrun1',
        otherFields: [
          'recurring run with id: testrecurringrun1',
          undefined,
          undefined,
          undefined,
          '-',
        ],
      },
    ]);
  });

  it('reloads the recurring run when refresh is called', async () => {
    mockNRecurringRuns(0, {});
    renderRecurringRunList();
    await loadRecurringRuns({});
    listRecurringRunsSpy.mockClear();

    await act(async () => {
      await getInstance().refresh();
    });

    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      '',
      10,
      JobSortKeys.CREATED_AT + ' desc',
      undefined,
      '',
      undefined,
    );
    expect(onErrorSpy).not.toHaveBeenCalled();
  });

  it('loads multiple recurring runs', async () => {
    mockNRecurringRuns(5, {});
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.rows).toHaveLength(5);
  });

  it('calls error callback when loading recurring runs fails', async () => {
    TestUtils.makeErrorResponseOnce(listRecurringRunsSpy as any, 'bad stuff happened');
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).toHaveBeenLastCalledWith(
      'Error: failed to fetch recurring runs.',
      new Error('bad stuff happened'),
    );
  });

  it('loads recurring runs for a given experiment id', async () => {
    mockNRecurringRuns(1, {});
    const props = generateProps();
    props.experimentIdMask = 'experiment1';
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      'experiment1',
    );
  });

  it('loads recurring runs for a given namespace', async () => {
    mockNRecurringRuns(1, {});
    const props = generateProps();
    props.namespaceMask = 'namespace1';
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      'namespace1',
      undefined,
      undefined,
    );
  });

  it('loads given list of recurring runs only', async () => {
    mockNRecurringRuns(5, {});
    const props = generateProps();
    props.recurringRunIdListMask = ['recurring run1', 'recurring run2'];
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(listRecurringRunsSpy).not.toHaveBeenCalled();
    expect(getRecurringRunSpy).toHaveBeenCalledTimes(2);
    expect(getRecurringRunSpy).toHaveBeenCalledWith('recurring run1');
    expect(getRecurringRunSpy).toHaveBeenCalledWith('recurring run2');
  });

  it('shows recurring run status', async () => {
    mockNRecurringRuns(1, { status: V2beta1RecurringRunStatus.ENABLED });
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.rows[0].otherFields[1]).toBe(V2beta1RecurringRunStatus.ENABLED);
  });

  it('shows trigger periodic', async () => {
    mockNRecurringRuns(1, {
      trigger: { periodic_schedule: { interval_second: '3600' } },
    });
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.rows[0].otherFields[2]).toEqual({
      periodic_schedule: { interval_second: '3600' },
    });
  });

  it('shows trigger cron', async () => {
    mockNRecurringRuns(1, {
      trigger: { cron_schedule: { cron: '0 * * * * ?' } },
    });
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.rows[0].otherFields[2]).toEqual({
      cron_schedule: { cron: '0 * * * * ?' },
    });
  });

  it('shows experiment name', async () => {
    mockNRecurringRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockResolvedValueOnce({
      experiments: [
        {
          experiment_id: 'test-experiment-id',
          display_name: 'test experiment',
        },
      ],
    });
    const props = generateProps();
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.rows[0].otherFields[3]).toEqual({
      displayName: 'test experiment',
      id: 'test-experiment-id',
    });
  });

  it('hides experiment name if instructed', async () => {
    mockNRecurringRuns(1, {
      experiment_id: 'test-experiment-id',
    });
    listExperimentsSpy.mockResolvedValueOnce({
      experiments: [
        {
          experiment_id: 'test-experiment-id',
          display_name: 'test experiment',
        },
      ],
    });
    const props = generateProps();
    props.hideExperimentColumn = true;
    renderRecurringRunList(props);
    await loadRecurringRuns({});

    expect(props.onError).not.toHaveBeenCalled();
    expect(lastCustomTableProps.columns.map((column: any) => column.label)).not.toContain(
      'Experiment',
    );
    expect(lastCustomTableProps.rows[0].otherFields).toHaveLength(4);
  });

  it('renders recurring run trigger in seconds', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '42' } },
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Every 42 seconds')).toBeInTheDocument();
  });

  it('renders recurring run trigger in minutes', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '120' } },
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Every 2 minutes')).toBeInTheDocument();
  });

  it('renders recurring run trigger in hours', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '7200' } },
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Every 2 hours')).toBeInTheDocument();
  });

  it('renders recurring run trigger in days', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._triggerCustomRenderer({
        value: { periodic_schedule: { interval_second: '86400' } },
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Every 1 days')).toBeInTheDocument();
  });

  it('renders recurring run trigger as cron', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._triggerCustomRenderer({
        value: { cron_schedule: { cron: '0 * * * * ?' } },
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Cron: 0 * * * * ?')).toBeInTheDocument();
  });

  it('renders status enabled', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._statusCustomRenderer({
        value: 'Enabled',
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Enabled')).toHaveStyle({ color: color.errorText });
  });

  it('renders status disabled', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._statusCustomRenderer({
        value: 'Disabled',
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Disabled')).toHaveStyle({ color: color.errorText });
  });

  it('renders status unknown', () => {
    renderRecurringRunList();
    const { getByText } = render(
      getInstance()._statusCustomRenderer({
        value: 'Unknown Status',
        id: 'recurring run-id',
      }),
    );
    expect(getByText('Unknown Status')).toHaveStyle({ color: color.errorText });
  });
});
