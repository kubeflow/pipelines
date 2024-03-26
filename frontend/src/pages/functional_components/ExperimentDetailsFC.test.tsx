/*
 * Copyright 2023 The Kubeflow Authors
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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import { ExperimentDetailsFC } from './ExperimentDetailsFC';
import TestUtils from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { Apis } from 'src/lib/Apis';
import { PageProps } from 'src/pages/Page';
import { RouteParams } from 'src/components/Router';
import { range } from 'lodash';

describe('ExperimentDetails', () => {
  const TEST_EXPERIMENT_ID = 'test-experiment-id';
  const TEST_EXPERIMENT_NAME = 'test-experiment-name';
  const updateToolbarSpy = jest.fn();
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApiV2, 'getExperiment');
  const listRecurringRunsSpy = jest.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns');
  const listRunsSpy = jest.spyOn(Apis.runServiceApiV2, 'listRuns');
  const getRunSpy = jest.spyOn(Apis.runServiceApiV2, 'getRun');

  const MOCK_EXPERIMENT = newMockExperiment();

  function newMockExperiment(): V2beta1Experiment {
    return {
      description: 'test-experiment-description',
      display_name: TEST_EXPERIMENT_NAME,
      experiment_id: TEST_EXPERIMENT_ID,
    };
  }

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: {
        params: { [RouteParams.experimentId]: MOCK_EXPERIMENT.experiment_id },
        isExact: true,
        path: '',
        url: '',
      },
      toolbarProps: { actions: {}, breadcrumbs: [], pageTitle: '' },
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  function mockNRecurringRuns(n: number): void {
    listRecurringRunsSpy.mockImplementation(() => ({
      recurringRuns: range(1, n + 1).map(
        i =>
          ({
            display_name: 'test recurring run name' + i,
            recurring_run_id: 'test-recurringrun-id' + i,
            status: V2beta1RecurringRunStatus.ENABLED,
          } as V2beta1RecurringRun),
      ),
    }));
  }

  function mockNRuns(n: number): void {
    listRunsSpy.mockImplementation(() => ({
      runs: range(1, n + 1).map(
        i =>
          ({
            run_id: 'testrun' + i,
            experiment_id: MOCK_EXPERIMENT.experiment_id,
            display_name: 'run with id: testrun' + i,
          } as V2beta1Run),
      ),
    }));
  }

  beforeEach(async () => {
    jest.clearAllMocks();
    getExperimentSpy.mockImplementation(() => newMockExperiment());
  });

  it('renders a page with no runs or recurring runs', async () => {
    listRunsSpy.mockImplementation(() => {}); // listRuns return nothing
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('No available runs found for this experiment.');
  });

  it('uses the experiment ID in props as the page title if the experiment has no name', async () => {
    mockNRecurringRuns(1);
    mockNRuns(1);
    const experiment = newMockExperiment();
    experiment.display_name = '';
    getExperimentSpy.mockImplementationOnce(() => experiment);

    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test-no-name-exp-id' } } as any;
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...props} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRecurringRunsSpy).toHaveBeenCalled();
      expect(listRunsSpy).toHaveBeenCalled();
    });

    expect(updateToolbarSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        pageTitle: 'test-no-name-exp-id',
      }),
    );
  });

  it('uses the experiment name as the page title', async () => {
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
    });

    expect(updateToolbarSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        pageTitle: TEST_EXPERIMENT_NAME,
      }),
    );
  });

  it('removes all description text after second newline and replaces with an ellipsis', async () => {
    const experiment = newMockExperiment();
    experiment.description = 'Line 1\nLine 2\nLine 3\nLine 4';
    getExperimentSpy.mockImplementationOnce(() => experiment);

    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
    });

    screen.getByText('Line 1');
    screen.getByText('Line 2');
    expect(screen.queryByText('Line 3')).toBeNull();
    expect(screen.queryByText('Line 4')).toBeNull();
  });

  it('calls getExperiment with the experiment ID in props', async () => {
    const props = generateProps();
    props.match = { params: { [RouteParams.experimentId]: 'test exp ID' } } as any;
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...props} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalledTimes(1);
      expect(getExperimentSpy).toHaveBeenCalledWith('test exp ID');
    });
  });

  it('shows an error banner if fetching the experiment fails', async () => {
    mockNRecurringRuns(1);
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'There was something wrong!');

    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        additionalInfo: 'There was something wrong!',
        message:
          `Error: failed to retrieve experiment: ${MOCK_EXPERIMENT.experiment_id}. ` +
          `Click Details for more information.`,
      }),
    );
  });

  it('shows a list of available runs', async () => {
    mockNRuns(1);
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('run with id: testrun1');
  });

  it("shows an error banner if fetching the experiment's recurring runs fails", async () => {
    mockNRecurringRuns(1);
    mockNRuns(1);
    TestUtils.makeErrorResponseOnce(listRecurringRunsSpy, 'There was something wrong!');

    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        additionalInfo: 'There was something wrong!',
        message:
          `Error: failed to retrieve recurring runs for experiment: ${MOCK_EXPERIMENT.experiment_id}. ` +
          `Click Details for more information.`,
      }),
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

    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('2 active');
  });

  it("fetches this experiment's recurring runs when recurring run manager modal is opened", async () => {
    mockNRecurringRuns(1);
    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    const recurringRunManageBtn = screen.getByText('Manage');
    expect(recurringRunManageBtn.closest('button')?.disabled).toEqual(false);
    // Open the recurring run manager modal
    fireEvent.click(recurringRunManageBtn);

    // listRecurringRun() will be called again after clicking manage button
    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('test recurring run name1');
  });

  it('refreshes the number of active recurring runs when the recurring run manager is closed', async () => {
    const recurringRuns = [
      {
        recurring_run_id: 'enabled-recurringrun-1-id',
        status: V2beta1RecurringRunStatus.ENABLED,
        display_name: 'enabled-recurringrun-1',
      },
    ];
    listRecurringRunsSpy.mockImplementation(() => ({ recurringRuns }));

    render(
      <CommonTestWrapper>
        <ExperimentDetailsFC {...generateProps()} />
      </CommonTestWrapper>,
    );

    await waitFor(() => {
      expect(getExperimentSpy).toHaveBeenCalled();
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('1 active'); // 1 active recurring run in the beginning

    const recurringRunManageBtn = screen.getByText('Manage');
    expect(recurringRunManageBtn.closest('button')?.disabled).toEqual(false);
    // Open the recurring run manager modal
    fireEvent.click(recurringRunManageBtn);

    // listRecurringRun() will be called again after clicking manage button
    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    const enabledBtnForFirstRRun = screen.getByText('Enabled');
    fireEvent.click(enabledBtnForFirstRRun); // disable the second recurring run

    listRecurringRunsSpy.mockReset();
    const recurringRunsAfterDisabled = [
      {
        recurring_run_id: 'enabled-recurringrun-1-id',
        status: V2beta1RecurringRunStatus.DISABLED,
        display_name: 'enabled-recurringrun-1',
      },
    ];
    listRecurringRunsSpy.mockImplementation(() => ({ recurringRunsAfterDisabled }));

    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    const closeBtn = screen.getByText('Close');
    // Close the recurring run manager modal
    fireEvent.click(closeBtn);

    await waitFor(() => {
      expect(listRecurringRunsSpy).toHaveBeenCalled();
    });

    screen.getByText('0 active'); // 0 active recurring run in thE final
  });

  // TODO(jlyaoyuli): add more unit tests for sub run list.
});
