/*
 * Copyright 2018 The Kubeflow Authors
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
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import { Apis } from 'src/lib/Apis';
import TestUtils, { expectErrors } from 'src/TestUtils';
import RecurringRunsManager, { RecurringRunListProps } from './RecurringRunsManager';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

const RECURRINGRUNS: V2beta1RecurringRun[] = [
  {
    created_at: new Date(2018, 10, 9, 8, 7, 6),
    display_name: 'test recurring run name',
    recurring_run_id: 'recurringrun1',
    status: V2beta1RecurringRunStatus.ENABLED,
  },
  {
    created_at: new Date(2018, 10, 9, 8, 7, 6),
    display_name: 'test recurring run name2',
    recurring_run_id: 'recurringrun2',
    status: V2beta1RecurringRunStatus.DISABLED,
  },
  {
    created_at: new Date(2018, 10, 9, 8, 7, 6),
    display_name: 'test recurring run name3',
    recurring_run_id: 'recurringrun3',
    status: V2beta1RecurringRunStatus.STATUSUNSPECIFIED,
  },
];

describe('RecurringRunsManager', () => {
  let updateDialogSpy: ReturnType<typeof vi.fn>;
  let updateSnackbarSpy: ReturnType<typeof vi.fn>;
  let listRecurringRunsSpy: ReturnType<typeof vi.spyOn>;
  let enableRecurringRunSpy: ReturnType<typeof vi.spyOn>;
  let disableRecurringRunSpy: ReturnType<typeof vi.spyOn>;
  let managerRef: React.RefObject<RecurringRunsManager> | null = null;

  function generateProps(): RecurringRunListProps {
    return {
      experimentId: 'test-experiment',
      history: {} as any,
      location: '' as any,
      match: {} as any,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
    };
  }

  function renderManager(): ReturnType<typeof render> {
    managerRef = React.createRef<RecurringRunsManager>();
    return render(
      <MemoryRouter>
        <RecurringRunsManager ref={managerRef} {...generateProps()} />
      </MemoryRouter>,
    );
  }

  function getRowById(id: string): HTMLElement {
    const rows = screen.getAllByTestId('table-row');
    const row = rows.find(element => element.getAttribute('data-row-id') === id);
    if (!row) {
      throw new Error(`Row not found: ${id}`);
    }
    return row;
  }

  beforeEach(() => {
    updateDialogSpy = vi.fn();
    updateSnackbarSpy = vi.fn();
    listRecurringRunsSpy = vi
      .spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns')
      .mockResolvedValue({ recurringRuns: RECURRINGRUNS });
    enableRecurringRunSpy = vi
      .spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun')
      .mockResolvedValue(undefined as any);
    disableRecurringRunSpy = vi
      .spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun')
      .mockResolvedValue(undefined as any);
  });

  afterEach(() => {
    managerRef = null;
    vi.restoreAllMocks();
  });

  it('calls API to load recurring runs', async () => {
    renderManager();
    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalled());

    listRecurringRunsSpy.mockClear();
    await act(async () => {
      await (managerRef!.current as any)._loadRuns({});
    });

    expect(listRecurringRunsSpy).toHaveBeenCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      'test-experiment',
    );
    expect(managerRef!.current!.state.runs).toEqual(RECURRINGRUNS);
    expect(screen.getByText('test recurring run name')).toBeInTheDocument();
  });

  it('shows error dialog if listing fails', async () => {
    const assertErrors = expectErrors();
    renderManager();
    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalled());

    updateDialogSpy.mockClear();
    listRecurringRunsSpy.mockClear();
    TestUtils.makeErrorResponseOnce(listRecurringRunsSpy as any, 'woops!');

    await act(async () => {
      await (managerRef!.current as any)._loadRuns({});
    });

    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'List recurring run configs request failed with:\nwoops!',
        title: 'Error retrieving recurring run configs',
      }),
    );
    expect(managerRef!.current!.state.runs).toEqual([]);
    assertErrors();
  });

  it('calls API to enable run', async () => {
    renderManager();
    await act(async () => {
      await (managerRef!.current as any)._setEnabledState('test-run', true);
    });
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith('test-run');
  });

  it('calls API to disable run', async () => {
    renderManager();
    await act(async () => {
      await (managerRef!.current as any)._setEnabledState('test-run', false);
    });
    expect(disableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(disableRecurringRunSpy).toHaveBeenLastCalledWith('test-run');
  });

  it('shows error if enable API call fails', async () => {
    const assertErrors = expectErrors();
    renderManager();
    TestUtils.makeErrorResponseOnce(enableRecurringRunSpy as any, 'cannot enable');
    await act(async () => {
      await (managerRef!.current as any)._setEnabledState('test-run', true);
    });
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Error changing enabled state of recurring run:\ncannot enable',
        title: 'Error',
      }),
    );
    assertErrors();
  });

  it('shows error if disable API call fails', async () => {
    const assertErrors = expectErrors();
    renderManager();
    TestUtils.makeErrorResponseOnce(disableRecurringRunSpy as any, 'cannot disable');
    await act(async () => {
      await (managerRef!.current as any)._setEnabledState('test-run', false);
    });
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Error changing enabled state of recurring run:\ncannot disable',
        title: 'Error',
      }),
    );
    assertErrors();
  });

  it('renders run name as link to its details page', async () => {
    renderManager();
    const link = await screen.findByRole('link', { name: 'test recurring run name' });
    expect(link).toHaveAttribute('href', '/recurringrun/details/recurringrun1');
  });

  it('renders a disable button if the run is enabled, clicking the button calls disable API', async () => {
    renderManager();
    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalled());

    const row = getRowById('recurringrun1');
    const button = within(row).getByRole('button', { name: 'Enabled' });
    fireEvent.click(button);

    await TestUtils.flushPromises();
    expect(disableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(disableRecurringRunSpy).toHaveBeenLastCalledWith('recurringrun1');
  });

  it('renders an enable button if the run is disabled, clicking the button calls enable API', async () => {
    renderManager();
    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalled());

    const row = getRowById('recurringrun2');
    const button = within(row).getByRole('button', { name: 'Disabled' });
    fireEvent.click(button);

    await TestUtils.flushPromises();
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith('recurringrun2');
  });

  it("renders an enable button if the run's enabled field is undefined, clicking the button calls enable API", async () => {
    renderManager();
    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalled());

    const row = getRowById('recurringrun3');
    const button = within(row).getByRole('button', { name: 'Disabled' });
    fireEvent.click(button);

    await TestUtils.flushPromises();
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith('recurringrun3');
  });

  it('reloads the list of runs after enable/disabling', async () => {
    renderManager();
    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1));

    const row = getRowById('recurringrun1');
    const button = within(row).getByRole('button', { name: 'Enabled' });
    fireEvent.click(button);

    await waitFor(() => expect(listRecurringRunsSpy).toHaveBeenCalledTimes(2));
  });
});
