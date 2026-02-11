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
import { act, render } from '@testing-library/react';
import { vi } from 'vitest';
import RecurringRunDetails from './RecurringRunDetails';
import TestUtils from 'src/TestUtils';
import { ApiJob, ApiResourceType } from 'src/apis/job';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import { RouteParams, RoutePage, QUERY_PARAMS } from 'src/components/Router';
import { ButtonKeys } from 'src/lib/Buttons';
import * as Utils from 'src/lib/Utils';

describe('RecurringRunDetails', () => {
  let renderResult: ReturnType<typeof render> | null = null;
  let recurringRunDetailsRef: React.RefObject<RecurringRunDetails> | null = null;

  const updateBannerSpy = vi.fn();
  const updateDialogSpy = vi.fn();
  const updateSnackbarSpy = vi.fn();
  const updateToolbarSpy = vi.fn();
  const historyPushSpy = vi.fn();
  const getJobSpy = vi.spyOn(Apis.jobServiceApi, 'getJob');
  const deleteRecurringRunSpy = vi.spyOn(Apis.recurringRunServiceApi, 'deleteRecurringRun');
  const enableRecurringRunSpy = vi.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun');
  const disableRecurringRunSpy = vi.spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun');
  const getExperimentSpy = vi.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const formatDateStringSpy = vi.spyOn(Utils, 'formatDateString');

  let fullTestJob: ApiJob = {};

  function generateProps(): PageProps {
    const match = {
      isExact: true,
      params: { [RouteParams.recurringRunId]: fullTestJob.id },
      path: '',
      url: '',
    };
    return TestUtils.generatePageProps(
      RecurringRunDetails,
      '' as any,
      match as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  async function renderRecurringRunDetails(propsPatch: Partial<PageProps> = {}) {
    recurringRunDetailsRef = React.createRef<RecurringRunDetails>();
    const props = { ...generateProps(), ...propsPatch } as PageProps;
    renderResult = render(<RecurringRunDetails ref={recurringRunDetailsRef} {...props} />);
    await TestUtils.flushPromises();
    return props;
  }

  function getInstance(): RecurringRunDetails {
    if (!recurringRunDetailsRef?.current) {
      throw new Error('RecurringRunDetails instance not available');
    }
    return recurringRunDetailsRef.current;
  }

  beforeEach(() => {
    fullTestJob = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test job description',
      enabled: true,
      id: 'test-job-id',
      max_concurrency: '50',
      no_catchup: true,
      name: 'test job',
      pipeline_spec: {
        parameters: [{ name: 'param1', value: 'value1' }],
        pipeline_id: 'some-pipeline-id',
      },
      trigger: {
        periodic_schedule: {
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          interval_second: '3600',
          start_time: new Date(2018, 9, 8, 7, 6),
        },
      },
    } as ApiJob;

    vi.clearAllMocks();
    getJobSpy.mockResolvedValue(fullTestJob);
    deleteRecurringRunSpy.mockResolvedValue(undefined as any);
    enableRecurringRunSpy.mockResolvedValue(undefined as any);
    disableRecurringRunSpy.mockResolvedValue(undefined as any);
    getExperimentSpy.mockResolvedValue(undefined as any);
    formatDateStringSpy.mockImplementation((date?: Date) => {
      return date ? '1/2/2019, 12:34:56 PM' : '-';
    });
  });

  afterEach(() => {
    renderResult?.unmount();
    renderResult = null;
    recurringRunDetailsRef = null;
  });

  it('renders a recurring run with periodic schedule', async () => {
    await renderRecurringRunDetails();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('renders a recurring run with cron schedule', async () => {
    const cronTestJob = {
      ...fullTestJob,
      no_catchup: undefined,
      trigger: {
        cron_schedule: {
          cron: '* * * 0 0 !',
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          start_time: new Date(2018, 9, 8, 7, 6),
        },
      },
    };
    getJobSpy.mockResolvedValue(cronTestJob);
    await renderRecurringRunDetails();
    expect(renderResult!.asFragment()).toMatchSnapshot();
  });

  it('loads the recurring run given its id in query params', async () => {
    await renderRecurringRunDetails();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).not.toHaveBeenCalled();
  });

  it('shows All runs -> run name when there is no experiment', async () => {
    await renderRecurringRunDetails();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'All runs', href: RoutePage.RUNS }],
        pageTitle: fullTestJob.name,
      }),
    );
  });

  it('loads the recurring run and its experiment if it has one', async () => {
    fullTestJob.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    await renderRecurringRunDetails();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('test-experiment-id');
  });

  it('shows Experiments -> Experiment name -> run name when there is an experiment', async () => {
    fullTestJob.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    getExperimentSpy.mockResolvedValue({ id: 'test-experiment-id', name: 'test experiment name' });
    await renderRecurringRunDetails();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [
          { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
          {
            displayName: 'test experiment name',
            href: RoutePage.EXPERIMENT_DETAILS.replace(
              ':' + RouteParams.experimentId,
              'test-experiment-id',
            ),
          },
        ],
        pageTitle: fullTestJob.name,
      }),
    );
  });

  it('shows error banner if run cannot be fetched', async () => {
    TestUtils.makeErrorResponseOnce(getJobSpy as any, 'woops!');
    await renderRecurringRunDetails();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops!',
        message: `Error: failed to retrieve recurring run: ${fullTestJob.id}. Click Details for more information.`,
        mode: 'error',
      }),
    );
  });

  it('shows warning banner if has experiment but experiment cannot be fetched. still loads run', async () => {
    fullTestJob.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    TestUtils.makeErrorResponseOnce(getExperimentSpy as any, 'woops!');
    await renderRecurringRunDetails();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2);
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops!',
        message: `Error: failed to retrieve this recurring run's experiment. Click Details for more information.`,
        mode: 'warning',
      }),
    );
    expect(getInstance().state.run).toEqual(fullTestJob);
  });

  it('has a Refresh button, clicking it refreshes the run details', async () => {
    await renderRecurringRunDetails();
    const refreshBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.REFRESH];
    expect(refreshBtn).toBeDefined();
    expect(getJobSpy).toHaveBeenCalledTimes(1);
    await act(async () => {
      await refreshBtn!.action();
    });
    expect(getJobSpy).toHaveBeenCalledTimes(2);
  });

  it('has a clone button, clicking it navigates to new run page', async () => {
    await renderRecurringRunDetails();
    const cloneBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.CLONE_RECURRING_RUN];
    expect(cloneBtn).toBeDefined();
    await act(async () => {
      await cloneBtn!.action();
    });
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.cloneFromRecurringRun}=${fullTestJob.id}` +
        `&${QUERY_PARAMS.isRecurring}=1`,
    );
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is enabled', async () => {
    await renderRecurringRunDetails();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const enableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy as any,
      ButtonKeys.ENABLE_RECURRING_RUN,
    );
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(true);
    const disableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy as any,
      ButtonKeys.DISABLE_RECURRING_RUN,
    );
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(false);
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is disabled', async () => {
    fullTestJob.enabled = false;
    await renderRecurringRunDetails();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const enableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy as any,
      ButtonKeys.ENABLE_RECURRING_RUN,
    );
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(false);
    const disableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy as any,
      ButtonKeys.DISABLE_RECURRING_RUN,
    );
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(true);
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is undefined', async () => {
    fullTestJob.enabled = undefined;
    await renderRecurringRunDetails();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const enableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy as any,
      ButtonKeys.ENABLE_RECURRING_RUN,
    );
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(false);
    const disableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy as any,
      ButtonKeys.DISABLE_RECURRING_RUN,
    );
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(true);
  });

  it('calls disable API when disable button is clicked, refreshes the page', async () => {
    await renderRecurringRunDetails();
    const disableBtn = getInstance().getInitialToolbarState().actions[
      ButtonKeys.DISABLE_RECURRING_RUN
    ];
    await act(async () => {
      await disableBtn!.action();
    });
    expect(disableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(disableRecurringRunSpy).toHaveBeenLastCalledWith('test-job-id');
    expect(getJobSpy).toHaveBeenCalledTimes(2);
    expect(getJobSpy).toHaveBeenLastCalledWith('test-job-id');
  });

  it('shows error dialog if disable fails', async () => {
    TestUtils.makeErrorResponseOnce(disableRecurringRunSpy as any, 'could not disable');
    await renderRecurringRunDetails();
    const disableBtn = getInstance().getInitialToolbarState().actions[
      ButtonKeys.DISABLE_RECURRING_RUN
    ];
    await act(async () => {
      await disableBtn!.action();
    });
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'could not disable',
        title: 'Failed to disable recurring run',
      }),
    );
  });

  it('shows error dialog if enable fails', async () => {
    fullTestJob.enabled = false;
    TestUtils.makeErrorResponseOnce(enableRecurringRunSpy as any, 'could not enable');
    await renderRecurringRunDetails();
    const enableBtn = getInstance().getInitialToolbarState().actions[
      ButtonKeys.ENABLE_RECURRING_RUN
    ];
    await act(async () => {
      await enableBtn!.action();
    });
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'could not enable',
        title: 'Failed to enable recurring run',
      }),
    );
  });

  it('calls enable API when enable button is clicked, refreshes the page', async () => {
    fullTestJob.enabled = false;
    await renderRecurringRunDetails();
    const enableBtn = getInstance().getInitialToolbarState().actions[
      ButtonKeys.ENABLE_RECURRING_RUN
    ];
    await act(async () => {
      await enableBtn!.action();
    });
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith('test-job-id');
    expect(getJobSpy).toHaveBeenCalledTimes(2);
    expect(getJobSpy).toHaveBeenLastCalledWith('test-job-id');
  });

  it('shows a delete button', async () => {
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
  });

  it('shows delete dialog when delete button is clicked', async () => {
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await act(async () => {
      await deleteBtn!.action();
    });
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        title: 'Delete this recurring run config?',
      }),
    );
  });

  it('calls delete API when delete confirmation dialog button is clicked', async () => {
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await act(async () => {
      await deleteBtn!.action();
    });
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((button: any) => button.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    expect(deleteRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(deleteRecurringRunSpy).toHaveBeenLastCalledWith('test-job-id');
  });

  it('does not call delete API when delete cancel dialog button is clicked', async () => {
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await act(async () => {
      await deleteBtn!.action();
    });
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((button: any) => button.text === 'Cancel');
    await act(async () => {
      await cancelBtn.onClick();
    });
    expect(deleteRecurringRunSpy).not.toHaveBeenCalled();
    expect(historyPushSpy).not.toHaveBeenCalled();
  });

  it('redirects back to parent experiment after delete', async () => {
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await act(async () => {
      await deleteBtn!.action();
    });
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((button: any) => button.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    expect(deleteRecurringRunSpy).toHaveBeenLastCalledWith('test-job-id');
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.EXPERIMENTS);
  });

  it('shows snackbar after successful deletion', async () => {
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await act(async () => {
      await deleteBtn!.action();
    });
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((button: any) => button.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    expect(updateSnackbarSpy).toHaveBeenCalledTimes(1);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      message: 'Delete succeeded for this recurring run config',
      open: true,
    });
  });

  it('shows error dialog after failing deletion', async () => {
    TestUtils.makeErrorResponseOnce(deleteRecurringRunSpy as any, 'could not delete');
    await renderRecurringRunDetails();
    const deleteBtn = getInstance().getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await act(async () => {
      await deleteBtn!.action();
    });
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((button: any) => button.text === 'Delete');
    await act(async () => {
      await confirmBtn.onClick();
    });
    await TestUtils.flushPromises();
    expect(updateDialogSpy).toHaveBeenCalledTimes(2);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content:
          'Failed to delete recurring run config: test-job-id with error: "could not delete"',
        title: 'Failed to delete recurring run config',
      }),
    );
    expect(historyPushSpy).not.toHaveBeenCalled();
  });
});
