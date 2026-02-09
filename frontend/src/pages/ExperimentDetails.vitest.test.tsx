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
import EnhancedExperimentDetails, { ExperimentDetails } from './ExperimentDetails';
import TestUtils from 'src/TestUtils';
import { V2beta1Experiment, V2beta1ExperimentStorageState } from 'src/apisv2beta1/experiment';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import { RoutePage, RouteParams, QUERY_PARAMS } from 'src/components/Router';
import { range } from 'lodash';
import { ButtonKeys } from 'src/lib/Buttons';
import { CommonTestWrapper } from 'src/TestWrapper';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NamespaceContext } from 'src/lib/KubeflowClient';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { V2beta1PredicateOperation } from 'src/apisv2beta1/filter/api';
import { vi } from 'vitest';

describe('ExperimentDetails', () => {
  const consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => null);
  const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => null);

  const updateToolbarSpy = vi.fn();
  const updateBannerSpy = vi.fn();
  const updateDialogSpy = vi.fn();
  const updateSnackbarSpy = vi.fn();
  const historyPushSpy = vi.fn();
  const getExperimentSpy = vi.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  const listRecurringRunsSpy = vi.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns');
  const listRunsSpy = vi.spyOn(Apis.runServiceApiV2, 'listRuns');

  const MOCK_EXPERIMENT = newMockExperiment();

  function newMockExperiment(): V2beta1Experiment {
    return {
      description: 'mock experiment description',
      experiment_id: 'some-mock-experiment-id',
      display_name: 'some mock experiment name',
    };
  }

  function generateProps(): PageProps {
    const match = { params: { [RouteParams.experimentId]: MOCK_EXPERIMENT.experiment_id } } as any;
    return TestUtils.generatePageProps(
      ExperimentDetails,
      {} as any,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  async function mockNRecurringRuns(n: number): Promise<void> {
    listRecurringRunsSpy.mockImplementation(() => ({
      recurringRuns: range(n).map(i => ({
        display_name: 'test job name' + i,
        recurring_run_id: 'test-recurringrun-id' + i,
        status: V2beta1RecurringRunStatus.ENABLED,
      })),
    }));
  }

  async function mockNRuns(n: number): Promise<void> {
    listRunsSpy.mockImplementation(() => ({
      runs: range(n).map(i => ({ run_id: 'test-run-id' + i, display_name: 'test run name' + i })),
    }));
  }

  async function renderExperimentDetails(props?: PageProps) {
    const utils = render(
      <CommonTestWrapper>
        <ExperimentDetails {...(props || generateProps())} />
      </CommonTestWrapper>,
    );
    await TestUtils.flushPromises();
    return utils;
  }

  async function waitForExperimentLoad(): Promise<void> {
    await screen.findByText('Recurring run configs');
  }

  async function waitForTableRows(expectedCount?: number): Promise<HTMLElement[]> {
    await waitFor(() => {
      const rows = screen.queryAllByTestId('table-row');
      if (expectedCount !== undefined) {
        expect(rows).toHaveLength(expectedCount);
      } else {
        expect(rows.length).toBeGreaterThan(0);
      }
    });
    return screen.queryAllByTestId('table-row');
  }

  function getRunListFilter() {
    const calls = listRunsSpy.mock.calls;
    if (!calls.length) {
      throw new Error('listRuns was not called');
    }
    const filterArg = calls[calls.length - 1][5];
    const filterString = decodeURIComponent(filterArg || '{"predicates": []}');
    return JSON.parse(filterString) as { predicates?: Array<{ key: string; operation: string }> };
  }

  function expectRunStorageFilter(operation: V2beta1PredicateOperation) {
    const filter = getRunListFilter();
    const predicate = (filter.predicates || []).find(p => p.key === 'storage_state');
    expect(predicate).toBeDefined();
    expect(predicate?.operation).toBe(operation);
  }

  beforeEach(async () => {
    consoleLogSpy.mockReset();
    consoleErrorSpy.mockReset();
    updateBannerSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
    updateToolbarSpy.mockReset();
    getExperimentSpy.mockReset();
    historyPushSpy.mockReset();
    listRecurringRunsSpy.mockReset();
    listRunsSpy.mockReset();

    getExperimentSpy.mockImplementation(() => newMockExperiment());

    await mockNRecurringRuns(0);
    await mockNRuns(0);
  });

  it('renders a page with no runs or recurring runs', async () => {
    const { asFragment } = await renderExperimentDetails();
    await waitForExperimentLoad();
    await screen.findByText('No available runs found for this experiment.');
    expect(updateBannerSpy).toHaveBeenCalledTimes(1);
    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
    expect(asFragment()).toMatchSnapshot();
  });

  it('uses the experiment ID in props as the page title if the experiment has no name', async () => {
    const experiment = newMockExperiment();
    experiment.display_name = '';

    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test exp ID' } } as any;

    getExperimentSpy.mockImplementationOnce(() => experiment);

    await renderExperimentDetails(props);
    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          pageTitle: 'test exp ID',
          pageTitleTooltip: 'test exp ID',
        }),
      );
    });
  });

  it('uses the experiment name as the page title', async () => {
    const experiment = newMockExperiment();
    experiment.display_name = 'A Test Experiment';

    getExperimentSpy.mockImplementationOnce(() => experiment);

    await renderExperimentDetails();
    await waitFor(() => {
      expect(updateToolbarSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          pageTitle: 'A Test Experiment',
          pageTitleTooltip: 'A Test Experiment',
        }),
      );
    });
  });

  it('uses an empty string if the experiment has no description', async () => {
    const experiment = newMockExperiment();
    delete experiment.description;

    getExperimentSpy.mockImplementationOnce(() => experiment);

    const { asFragment } = await renderExperimentDetails();
    await waitForExperimentLoad();
    await screen.findByText('No available runs found for this experiment.');
    expect(asFragment()).toMatchSnapshot();
  });

  it('removes all description text after second newline and replaces with an ellipsis', async () => {
    const experiment = newMockExperiment();
    experiment.description = 'Line 1\nLine 2\nLine 3\nLine 4';

    getExperimentSpy.mockImplementationOnce(() => experiment);

    await renderExperimentDetails();
    await waitForExperimentLoad();
    screen.getByText('Line 1');
    screen.getByText('Line 2');
    screen.getByText('...');
  });

  it('opens the expanded description modal when the expand button is clicked', async () => {
    const { container } = await renderExperimentDetails();
    await waitForExperimentLoad();

    const expandButton = container.querySelector('#expandExperimentDescriptionBtn');
    expect(expandButton).not.toBeNull();
    fireEvent.click(expandButton as HTMLElement);
    await TestUtils.flushPromises();
    expect(updateDialogSpy).toHaveBeenCalledWith({
      content: MOCK_EXPERIMENT.description,
      title: 'Experiment description',
    });
  }, 20000);

  it('calls getExperiment with the experiment ID in props', async () => {
    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test exp ID' } } as any;
    await renderExperimentDetails(props);
    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenCalledWith('test exp ID');
    });
  });

  it('shows an error banner if fetching the experiment fails', async () => {
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error');

    await renderExperimentDetails();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test error',
          message:
            'Error: failed to retrieve experiment: ' +
            MOCK_EXPERIMENT.experiment_id +
            '. Click Details for more information.',
          mode: 'error',
        }),
      );
    });
    expect(
      consoleErrorSpy.mock.calls.some(
        ([message]) => message === 'Error loading experiment: ' + MOCK_EXPERIMENT.experiment_id,
      ),
    ).toBe(true);
  });

  it('shows a list of available runs', async () => {
    await mockNRecurringRuns(1);
    await renderExperimentDetails();
    await waitFor(() => expect(listRunsSpy).toHaveBeenCalled());
    expectRunStorageFilter(V2beta1PredicateOperation.NOTEQUALS);
  });

  it('shows a list of archived runs', async () => {
    await mockNRecurringRuns(1);

    getExperimentSpy.mockImplementation(() => {
      const apiExperiment = newMockExperiment();
      apiExperiment['storage_state'] = V2beta1ExperimentStorageState.ARCHIVED;
      return apiExperiment;
    });

    await renderExperimentDetails();
    await waitFor(() => expect(listRunsSpy).toHaveBeenCalled());
    expectRunStorageFilter(V2beta1PredicateOperation.EQUALS);
  });

  it("fetches this experiment's recurring runs", async () => {
    await mockNRecurringRuns(1);

    const { asFragment } = await renderExperimentDetails();
    await waitForExperimentLoad();

    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      100,
      '',
      undefined,
      undefined,
      MOCK_EXPERIMENT.experiment_id,
    );
    screen.getByText('1 active');
    await screen.findByText('No available runs found for this experiment.');
    expect(asFragment()).toMatchSnapshot();
  }, 20000);

  it("shows an error banner if fetching the experiment's recurring runs fails", async () => {
    TestUtils.makeErrorResponseOnce(listRecurringRunsSpy, 'test error');

    await renderExperimentDetails();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith(
        expect.objectContaining({
          additionalInfo: 'test error',
          message:
            'Error: failed to retrieve recurring runs for experiment: ' +
            MOCK_EXPERIMENT.experiment_id +
            '. Click Details for more information.',
          mode: 'error',
        }),
      );
    });
    expect(consoleErrorSpy.mock.calls[0][0]).toBe(
      'Error fetching recurring runs for experiment: ' + MOCK_EXPERIMENT.experiment_id,
    );
  });

  it('only counts enabled recurring runs as active', async () => {
    const recurringRuns = [
      {
        recurring_run_id: 'enabled-recurringrun-1-id',
        status: V2beta1RecurringRunStatus.ENABLED,
        display_name: 'enabled-recurringrun-1',
      },
      {
        recurring_run_id: 'enabled-recurringrun-2-id',
        status: V2beta1RecurringRunStatus.ENABLED,
        display_name: 'enabled-recurringrun-2',
      },
      {
        recurring_run_id: 'disabled-recurringrun-1-id',
        status: V2beta1RecurringRunStatus.DISABLED,
        display_name: 'disabled-recurringrun-1',
      },
    ];
    listRecurringRunsSpy.mockImplementationOnce(() => ({ recurringRuns }));

    await renderExperimentDetails();
    await waitForExperimentLoad();

    screen.getByText('2 active');
  });

  it("opens the recurring run manager modal when 'manage' is clicked", async () => {
    await mockNRecurringRuns(1);
    await renderExperimentDetails();
    await waitForExperimentLoad();

    fireEvent.click(screen.getByRole('button', { name: 'Manage' }));
    await screen.findByRole('button', { name: 'Close' });
  });

  it('closes the recurring run manager modal', async () => {
    await mockNRecurringRuns(1);
    await renderExperimentDetails();
    await waitForExperimentLoad();

    fireEvent.click(screen.getByRole('button', { name: 'Manage' }));
    const closeButton = await screen.findByRole('button', { name: 'Close' });
    fireEvent.click(closeButton);

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Close' })).toBeNull();
    });
  });

  it('refreshes the number of active recurring runs when the recurring run manager is closed', async () => {
    await mockNRecurringRuns(1);
    await renderExperimentDetails();
    await waitForExperimentLoad();

    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    });

    fireEvent.click(screen.getByRole('button', { name: 'Manage' }));
    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalledTimes(2);
    });

    fireEvent.click(await screen.findByRole('button', { name: 'Close' }));
    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalledTimes(3);
    });
  });

  it('clears the error banner on refresh', async () => {
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'test error');

    await renderExperimentDetails();

    await waitFor(() => {
      expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({ mode: 'error' }));
    });

    const lastToolbarCall = updateToolbarSpy.mock.calls[updateToolbarSpy.mock.calls.length - 1];
    const refreshAction = lastToolbarCall?.[0]?.actions?.[ButtonKeys.REFRESH];
    expect(refreshAction).toBeDefined();

    await refreshAction!.action();
    await TestUtils.flushPromises();

    expect(updateBannerSpy).toHaveBeenLastCalledWith({});
  });

  it('navigates to the compare runs page', async () => {
    const runs = [
      { run_id: 'run-1-id', display_name: 'run-1' },
      { run_id: 'run-2-id', display_name: 'run-2' },
    ];
    listRunsSpy.mockImplementation(() => ({ runs }));

    await renderExperimentDetails();
    const rows = await waitForTableRows(2);

    fireEvent.click(rows[0]);
    fireEvent.click(rows[1]);

    const compareButton = await screen.findByRole('button', { name: 'Compare runs' });
    await waitFor(() => expect(compareButton).toBeEnabled());
    fireEvent.click(compareButton);

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.COMPARE + `?${QUERY_PARAMS.runlist}=run-1-id,run-2-id`,
    );
  });

  it('navigates to the new run page and passes this experiments ID as a query param', async () => {
    await renderExperimentDetails();
    await waitForExperimentLoad();

    const newRunButton = screen.getByRole('button', { name: 'Create run' });
    fireEvent.click(newRunButton);

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.experiment_id}`,
    );
  });

  it('navigates to the new run page with query param indicating it will be a recurring run', async () => {
    await renderExperimentDetails();
    await waitForExperimentLoad();

    const newRecurringButton = screen.getByRole('button', { name: 'Create recurring run' });
    fireEvent.click(newRecurringButton);

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.experimentId}=${MOCK_EXPERIMENT.experiment_id}` +
        `&${QUERY_PARAMS.isRecurring}=1`,
    );
  });

  it('supports cloning a selected run', async () => {
    const runs = [{ run_id: 'run-1-id', display_name: 'run-1' }];
    listRunsSpy.mockImplementation(() => ({ runs }));

    await renderExperimentDetails();
    const rows = await waitForTableRows(1);

    fireEvent.click(rows[0]);

    const cloneButton = await screen.findByRole('button', { name: 'Clone run' });
    await waitFor(() => expect(cloneButton).toBeEnabled());
    fireEvent.click(cloneButton);

    expect(historyPushSpy).toHaveBeenCalledWith(
      RoutePage.NEW_RUN + `?${QUERY_PARAMS.cloneFromRun}=run-1-id`,
    );
  });

  it('enables the compare runs button only when between 2 and 10 runs are selected', async () => {
    await mockNRuns(12);

    await renderExperimentDetails();
    const rows = await waitForTableRows(12);

    for (let i = 0; i < 12; i++) {
      fireEvent.click(rows[i]);
      const selectedCount = i + 1;
      await waitFor(() => {
        const compareButton = screen.getByRole('button', { name: 'Compare runs' });
        if (selectedCount < 2 || selectedCount > 10) {
          expect(compareButton).toBeDisabled();
        } else {
          expect(compareButton).toBeEnabled();
        }
      });
    }
  }, 20000);

  it('enables the clone run button only when 1 run is selected', async () => {
    await mockNRuns(4);

    await renderExperimentDetails();
    const rows = await waitForTableRows(4);

    for (let i = 0; i < 4; i++) {
      fireEvent.click(rows[i]);
      const selectedCount = i + 1;
      await waitFor(() => {
        const cloneButton = screen.getByRole('button', { name: 'Clone run' });
        if (selectedCount === 1) {
          expect(cloneButton).toBeEnabled();
        } else {
          expect(cloneButton).toBeDisabled();
        }
      });
    }
  });

  it('enables Archive button when at least one run is selected', async () => {
    await mockNRuns(4);

    await renderExperimentDetails();
    const rows = await waitForTableRows(4);

    for (let i = 0; i < 4; i++) {
      fireEvent.click(rows[i]);
      const selectedCount = i + 1;
      await waitFor(() => {
        const archiveButton = screen.getByRole('button', { name: 'Archive' });
        if (selectedCount >= 1) {
          expect(archiveButton).toBeEnabled();
        } else {
          expect(archiveButton).toBeDisabled();
        }
      });
    }
  }, 20000);

  it('enables Restore button when at least one run is selected', async () => {
    await mockNRuns(4);

    await renderExperimentDetails();
    await waitForTableRows(4);

    fireEvent.click(screen.getByRole('button', { name: 'Archived' }));
    await waitForTableRows(4);

    const rows = screen.queryAllByTestId('table-row');
    for (let i = 0; i < 4; i++) {
      fireEvent.click(rows[i]);
      const selectedCount = i + 1;
      await waitFor(() => {
        const restoreButton = screen.getByRole('button', { name: 'Restore' });
        if (selectedCount >= 1) {
          expect(restoreButton).toBeEnabled();
        } else {
          expect(restoreButton).toBeDisabled();
        }
      });
    }
  });

  it('switches to another tab will change Archive/Restore button', async () => {
    await mockNRuns(4);

    await renderExperimentDetails();
    await waitForTableRows(4);

    fireEvent.click(screen.getByRole('button', { name: 'Archived' }));
    await waitForTableRows(4);
    expect(screen.queryByRole('button', { name: 'Archive' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Restore' })).toBeDefined();

    fireEvent.click(screen.getByRole('button', { name: 'Active' }));
    await waitForTableRows(4);
    expect(screen.getByRole('button', { name: 'Archive' })).toBeDefined();
    expect(screen.queryByRole('button', { name: 'Restore' })).toBeNull();
  });

  it('switches to active/archive tab will show active/archive runs', async () => {
    await mockNRuns(4);
    await renderExperimentDetails();
    await waitForTableRows(4);

    await mockNRuns(2);
    fireEvent.click(screen.getByRole('button', { name: 'Archived' }));
    await waitForTableRows(2);
  });

  it('switches to another tab will change Archive/Restore button', async () => {
    await mockNRuns(4);

    await renderExperimentDetails();
    await waitForTableRows(4);

    fireEvent.click(screen.getByRole('button', { name: 'Archived' }));
    await waitForTableRows(4);
    expect(screen.queryByRole('button', { name: 'Archive' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Restore' })).toBeDefined();

    fireEvent.click(screen.getByRole('button', { name: 'Active' }));
    await waitForTableRows(4);
    expect(screen.getByRole('button', { name: 'Archive' })).toBeDefined();
    expect(screen.queryByRole('button', { name: 'Restore' })).toBeNull();
  });

  describe('EnhancedExperimentDetails', () => {
    it('renders ExperimentDetails initially', () => {
      render(<EnhancedExperimentDetails {...generateProps()}></EnhancedExperimentDetails>);
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
    });

    it('redirects to ExperimentList page if namespace changes', () => {
      const history = createMemoryHistory();
      const { rerender } = render(
        <Router history={history}>
          <NamespaceContext.Provider value='test-ns-1'>
            <EnhancedExperimentDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      rerender(
        <Router history={history}>
          <NamespaceContext.Provider value='test-ns-2'>
            <EnhancedExperimentDetails {...generateProps()} />
          </NamespaceContext.Provider>
        </Router>,
      );
      expect(history.location.pathname).toEqual(RoutePage.EXPERIMENTS);
    });
  });
});
