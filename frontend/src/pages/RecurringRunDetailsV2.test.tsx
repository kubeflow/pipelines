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

import { render, screen, waitFor } from '@testing-library/react';
import * as React from 'react';
import fs from 'fs';
import { CommonTestWrapper } from 'src/TestWrapper';
import RecurringRunDetailsRouter from './RecurringRunDetailsRouter';
import RecurringRunDetailsV2 from './RecurringRunDetailsV2';
import TestUtils from 'src/TestUtils';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { Apis } from 'src/lib/Apis';
import { PageProps } from './Page';
import { RouteParams, RoutePage } from 'src/components/Router';
import * as features from 'src/features';

const V2_PIPELINESPEC_PATH = 'src/data/test/lightweight_python_functions_v2_pipeline_rev.yaml';
const v2YamlTemplateString = fs.readFileSync(V2_PIPELINESPEC_PATH, 'utf8');

describe('RecurringRunDetailsV2', () => {
  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun');
  const deleteRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'deleteRecurringRun');
  const enableRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun');
  const disableRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  const getPipelineVersionTemplateSpy = jest.spyOn(
    Apis.pipelineServiceApi,
    'getPipelineVersionTemplate',
  );

  let fullTestV2RecurringRun: V2beta1RecurringRun = {};

  function generateProps(): PageProps {
    const match = {
      isExact: true,
      params: { [RouteParams.recurringRunId]: fullTestV2RecurringRun.recurring_run_id },
      path: '',
      url: '',
    };
    return TestUtils.generatePageProps(
      RecurringRunDetailsV2,
      '' as any,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
    );
  }

  beforeEach(() => {
    fullTestV2RecurringRun = {
      created_at: new Date(2018, 8, 5, 4, 3, 2),
      description: 'test recurring run description',
      display_name: 'test recurring run',
      max_concurrency: '50',
      no_catchup: true,
      pipeline_version_reference: {
        pipeline_id: 'test-pipeline-id',
        pipeline_version_id: 'test-pipeline-version-id',
      },
      recurring_run_id: 'test-recurring-run-id',
      runtime_config: { parameters: { param1: 'value1' } },
      status: V2beta1RecurringRunStatus.ENABLED,
      trigger: {
        periodic_schedule: {
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          interval_second: '3600',
          start_time: new Date(2018, 9, 8, 7, 6),
        },
      },
    } as V2beta1RecurringRun;

    jest.clearAllMocks();
    jest.spyOn(features, 'isFeatureEnabled').mockReturnValue(true);

    getRecurringRunSpy.mockImplementation(() => fullTestV2RecurringRun);
    getPipelineVersionTemplateSpy.mockImplementation(() =>
      Promise.resolve({ template: v2YamlTemplateString }),
    );
    deleteRecurringRunSpy.mockImplementation();
    enableRecurringRunSpy.mockImplementation();
    disableRecurringRunSpy.mockImplementation();
    getExperimentSpy.mockImplementation();
  });

  it('renders a recurring run with periodic schedule', async () => {
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalledTimes(2);
      expect(getPipelineVersionTemplateSpy).toHaveBeenCalled();
    });

    screen.getByText('Enabled');
    screen.getByText('Yes');
    screen.getByText('Trigger');
    screen.getByText('Every 1 hours');
    screen.getByText('Max. concurrent runs');
    screen.getByText('50');
    screen.getByText('Catchup');
    screen.getByText('false');
    screen.getByText('param1');
    screen.getByText('value1');
  });

  it('renders a recurring run with cron schedule', async () => {
    const cronTestRecurringRun = {
      ...fullTestV2RecurringRun,
      no_catchup: undefined, // in api requests, it's undefined when false
      trigger: {
        cron_schedule: {
          cron: '* * * 0 0 !',
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          start_time: new Date(2018, 9, 8, 7, 6),
        },
      },
    };
    getRecurringRunSpy.mockImplementation(() => cronTestRecurringRun);

    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
    });

    screen.getByText('Enabled');
    screen.getByText('Yes');
    screen.getByText('Trigger');
    screen.getByText('* * * 0 0 !');
    screen.getByText('Max. concurrent runs');
    screen.getByText('50');
    screen.getByText('Catchup');
    screen.getByText('true');
  });

  it('loads the recurring run given its id in query params', async () => {
    // The run id is in the router match object, defined inside generateProps
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
    });

    expect(getRecurringRunSpy).toHaveBeenLastCalledWith(fullTestV2RecurringRun.recurring_run_id);
    expect(getExperimentSpy).not.toHaveBeenCalled();
  });

  it('shows All runs -> run name when there is no experiment', async () => {
    // The run id is in the router match object, defined inside generateProps
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
    });

    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'All runs', href: RoutePage.RUNS }],
        pageTitle: fullTestV2RecurringRun.display_name,
      }),
    );
  });

  it('loads the recurring run and its experiment if it has one', async () => {
    fullTestV2RecurringRun.experiment_id = 'test-experiment-id';
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
    });

    expect(getRecurringRunSpy).toHaveBeenLastCalledWith(fullTestV2RecurringRun.recurring_run_id);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('test-experiment-id');
  });

  it('shows Experiments -> Experiment name -> run name when there is an experiment', async () => {
    fullTestV2RecurringRun.experiment_id = 'test-experiment-id';
    getExperimentSpy.mockImplementation(id => ({ id, name: 'test experiment name' }));
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
      expect(getExperimentSpy).toHaveBeenCalled();
    });

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
        pageTitle: fullTestV2RecurringRun.display_name,
      }),
    );
  });

  it('shows error banner if run cannot be fetched', async () => {
    TestUtils.makeErrorResponseOnce(getRecurringRunSpy, 'woops!');
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops!',
        message: `Error: failed to retrieve recurring run: ${fullTestV2RecurringRun.recurring_run_id}. Click Details for more information.`,
        mode: 'error',
      }),
    );
  });

  it('shows warning banner if has experiment but experiment cannot be fetched. still loads run', async () => {
    fullTestV2RecurringRun.experiment_id = 'test-experiment-id';
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'woops!');
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
    });

    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops!',
        message: `Error: failed to retrieve this recurring run's experiment. Click Details for more information.`,
        mode: 'warning',
      }),
    );

    // "Still loads run" means that the details are still rendered successfully.
    screen.getByText('Enabled');
    screen.getByText('Yes');
    screen.getByText('Trigger');
    screen.getByText('Every 1 hours');
    screen.getByText('Max. concurrent runs');
    screen.getByText('50');
    screen.getByText('Catchup');
    screen.getByText('false');
    screen.getByText('param1');
    screen.getByText('value1');
  });

  it('shows top bar buttons', async () => {
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          actions: expect.objectContaining({
            cloneRecurringRun: expect.objectContaining({ title: 'Clone recurring run' }),
            refresh: expect.objectContaining({ title: 'Refresh' }),
            enableRecurringRun: expect.objectContaining({ title: 'Enable', disabled: true }),
            disableRecurringRun: expect.objectContaining({ title: 'Disable', disabled: false }),
            deleteRun: expect.objectContaining({ title: 'Delete' }),
          }),
        }),
      );
    });
  });

  it('enables Enable buttons if the run is disabled', async () => {
    fullTestV2RecurringRun.status = V2beta1RecurringRunStatus.DISABLED;
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          actions: expect.objectContaining({
            cloneRecurringRun: expect.objectContaining({ title: 'Clone recurring run' }),
            refresh: expect.objectContaining({ title: 'Refresh' }),
            enableRecurringRun: expect.objectContaining({ title: 'Enable', disabled: false }),
            disableRecurringRun: expect.objectContaining({ title: 'Disable', disabled: true }),
            deleteRun: expect.objectContaining({ title: 'Delete' }),
          }),
        }),
      );
    });
  });

  it('shows enables Enable buttons if the run is undefined', async () => {
    fullTestV2RecurringRun.status = undefined;
    render(
      <CommonTestWrapper>
        <RecurringRunDetailsRouter {...generateProps()} />
      </CommonTestWrapper>,
    );
    await waitFor(() => {
      expect(getRecurringRunSpy).toHaveBeenCalled();
      expect(updateToolbarSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          actions: expect.objectContaining({
            cloneRecurringRun: expect.objectContaining({ title: 'Clone recurring run' }),
            refresh: expect.objectContaining({ title: 'Refresh' }),
            enableRecurringRun: expect.objectContaining({ title: 'Enable', disabled: false }),
            disableRecurringRun: expect.objectContaining({ title: 'Disable', disabled: true }),
            deleteRun: expect.objectContaining({ title: 'Delete' }),
          }),
        }),
      );
    });
  });
});
