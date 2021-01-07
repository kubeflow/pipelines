/*
 * Copyright 2018 Google LLC
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
import RecurringRunDetails from './RecurringRunDetails';
import TestUtils from '../TestUtils';
import { ApiJob, ApiResourceType } from '../apis/job';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RouteParams, RoutePage, QUERY_PARAMS } from '../components/Router';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';
import { TFunction } from 'i18next';

jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate hook can use it without a warning being shown
  withTranslation: () => (component: React.ComponentClass) => {
    component.defaultProps = { ...component.defaultProps, t: (key: string) => key };
    return component;
  },
}));

describe('RecurringRunDetails', () => {
  let tree: ReactWrapper<any> | ShallowWrapper<any>;

  const updateBannerSpy = jest.fn();
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const getJobSpy = jest.spyOn(Apis.jobServiceApi, 'getJob');
  const deleteJobSpy = jest.spyOn(Apis.jobServiceApi, 'deleteJob');
  const enableJobSpy = jest.spyOn(Apis.jobServiceApi, 'enableJob');
  const disableJobSpy = jest.spyOn(Apis.jobServiceApi, 'disableJob');
  const getExperimentSpy = jest.spyOn(Apis.experimentServiceApi, 'getExperiment');
  let t: TFunction = (key: string) => key;
  let fullTestJob: ApiJob = {};

  function generateProps(): PageProps {
    const match = {
      isExact: true,
      params: { [RouteParams.runId]: fullTestJob.id },
      path: '',
      url: '',
    };
    return TestUtils.generatePageProps(
      RecurringRunDetails,
      '' as any,
      match,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
      { t },
    );
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

    jest.clearAllMocks();
    getJobSpy.mockImplementation(() => fullTestJob);
    deleteJobSpy.mockImplementation();
    enableJobSpy.mockImplementation();
    disableJobSpy.mockImplementation();
    getExperimentSpy.mockImplementation();
  });

  afterEach(() => tree.unmount());

  it('renders a recurring run with periodic schedule', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('renders a recurring run with cron schedule', async () => {
    const cronTestJob = {
      ...fullTestJob,
      no_catchup: undefined, // in api requests, it's undefined when false
      trigger: {
        cron_schedule: {
          cron: '* * * 0 0 !',
          end_time: new Date(2018, 10, 9, 8, 7, 6),
          start_time: new Date(2018, 9, 8, 7, 6),
        },
      },
    };
    getJobSpy.mockImplementation(() => cronTestJob);
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(tree).toMatchSnapshot();
  });

  it('loads the recurring run given its id in query params', async () => {
    // The run id is in the router match object, defined inside generateProps
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).not.toHaveBeenCalled();
  });

  it('shows All runs -> run name when there is no experiment', async () => {
    // The run id is in the router match object, defined inside generateProps
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [{ displayName: 'allRuns', href: RoutePage.RUNS }],
        pageTitle: fullTestJob.name,
      }),
    );
  });

  it('loads the recurring run and its experiment if it has one', async () => {
    fullTestJob.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('test-experiment-id');
  });

  it('shows Experiments -> Experiment name -> run name when there is an experiment', async () => {
    fullTestJob.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    getExperimentSpy.mockImplementation(id => ({ id, name: 'test experiment name' }));
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        breadcrumbs: [
          { displayName: 'common:experiments', href: RoutePage.EXPERIMENTS },
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
    TestUtils.makeErrorResponseOnce(getJobSpy, 'woops!');
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops!',
        message: `errorRetrieveRecurrRun: test-job-id. common:clickDetails`,
        mode: 'error',
      }),
    );
  });

  it('shows warning banner if has experiment but experiment cannot be fetched. still loads run', async () => {
    fullTestJob.resource_references = [
      { key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } },
    ];
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'woops!');
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        additionalInfo: 'woops!',
        message: `errorRetrieveExperimentRecurrRun' common:clickDetails`,
        mode: 'common:warning',
      }),
    );
    expect(tree.state('run')).toEqual(fullTestJob);
  });

  it('has a Refresh button, clicking it refreshes the run details', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    const instance = tree.instance() as RecurringRunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions[ButtonKeys.REFRESH];
    expect(refreshBtn).toBeDefined();
    expect(getJobSpy).toHaveBeenCalledTimes(1);
    await refreshBtn!.action();
    expect(getJobSpy).toHaveBeenCalledTimes(2);
  });

  it('has a clone button, clicking it navigates to new run page', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const cloneBtn = instance.getInitialToolbarState().actions[ButtonKeys.CLONE_RECURRING_RUN];
    expect(cloneBtn).toBeDefined();
    await cloneBtn!.action();
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(
      RoutePage.NEW_RUN +
        `?${QUERY_PARAMS.cloneFromRecurringRun}=${fullTestJob!.id}` +
        `&${QUERY_PARAMS.isRecurring}=1`,
    );
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is enabled', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const enableBtn = TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ENABLE_RECURRING_RUN);
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(true);
    const disableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy,
      ButtonKeys.DISABLE_RECURRING_RUN,
    );
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(false);
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is disabled', async () => {
    fullTestJob.enabled = false;
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const enableBtn = TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ENABLE_RECURRING_RUN);
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(false);
    const disableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy,
      ButtonKeys.DISABLE_RECURRING_RUN,
    );
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(true);
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is undefined', async () => {
    fullTestJob.enabled = undefined;
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const enableBtn = TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.ENABLE_RECURRING_RUN);
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(false);
    const disableBtn = TestUtils.getToolbarButton(
      updateToolbarSpy,
      ButtonKeys.DISABLE_RECURRING_RUN,
    );
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(true);
  });

  it('calls disable API when disable button is clicked, refreshes the page', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const disableBtn = instance.getInitialToolbarState().actions[ButtonKeys.DISABLE_RECURRING_RUN];
    await disableBtn!.action();
    expect(disableJobSpy).toHaveBeenCalledTimes(1);
    expect(disableJobSpy).toHaveBeenLastCalledWith('test-job-id');
    expect(getJobSpy).toHaveBeenCalledTimes(2);
    expect(getJobSpy).toHaveBeenLastCalledWith('test-job-id');
  });

  it('shows error dialog if disable fails', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(disableJobSpy, 'could not disable');
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const disableBtn = instance.getInitialToolbarState().actions[ButtonKeys.DISABLE_RECURRING_RUN];
    await disableBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'could not disable',
        title: 'common:failedTo common:disable common:recurringRun',
      }),
    );
  });

  it('shows error dialog if enable fails', async () => {
    fullTestJob.enabled = false;
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(enableJobSpy, 'could not enable');
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const enableBtn = instance.getInitialToolbarState().actions[ButtonKeys.ENABLE_RECURRING_RUN];
    await enableBtn!.action();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'could not enable',
        title: 'common:failedTo common:enable common:recurringRun',
      }),
    );
  });

  it('calls enable API when enable button is clicked, refreshes the page', async () => {
    fullTestJob.enabled = false;
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const enableBtn = instance.getInitialToolbarState().actions[ButtonKeys.ENABLE_RECURRING_RUN];
    await enableBtn!.action();
    expect(enableJobSpy).toHaveBeenCalledTimes(1);
    expect(enableJobSpy).toHaveBeenLastCalledWith('test-job-id');
    expect(getJobSpy).toHaveBeenCalledTimes(2);
    expect(getJobSpy).toHaveBeenLastCalledWith('test-job-id');
  });

  it('shows a delete button', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    expect(deleteBtn).toBeDefined();
  });

  it('shows delete dialog when delete button is clicked', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await deleteBtn!.action();
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        title: 'common:delete common:this recurring run config?',
      }),
    );
  });

  it('calls delete API when delete confirmation dialog button is clicked', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(deleteJobSpy).toHaveBeenCalledTimes(1);
    expect(deleteJobSpy).toHaveBeenLastCalledWith('test-job-id');
  });

  it('does not call delete API when delete cancel dialog button is clicked', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions[ButtonKeys.DELETE_RUN];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:cancel');
    await confirmBtn.onClick();
    expect(deleteJobSpy).not.toHaveBeenCalled();
    // Should not reroute
    expect(historyPushSpy).not.toHaveBeenCalled();
  });

  // TODO: need to test the dismiss path too--when the dialog is dismissed using ESC
  // or clicking outside it, it should be treated the same way as clicking Cancel.

  it('redirects back to parent experiment after delete', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as RecurringRunDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(deleteJobSpy).toHaveBeenLastCalledWith('test-job-id');
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.EXPERIMENTS);
  });

  it('shows snackbar after successful deletion', async () => {
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as RecurringRunDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    expect(updateSnackbarSpy).toHaveBeenCalledTimes(1);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      message: 'common:delete common:succeededFor common:this recurring run config',
      open: true,
    });
  });

  it('shows error dialog after failing deletion', async () => {
    TestUtils.makeErrorResponseOnce(deleteJobSpy, 'could not delete');
    tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const deleteBtn = (tree.instance() as RecurringRunDetails).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    await TestUtils.flushPromises();
    expect(updateDialogSpy).toHaveBeenCalledTimes(2);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content:
          'common:failedTo common:delete recurring run config: test-job-id common:withError: "could not delete"',
        title: 'common:failedTo common:delete recurring run config',
      }),
    );
    // Should not reroute
    expect(historyPushSpy).not.toHaveBeenCalled();
  });
});
