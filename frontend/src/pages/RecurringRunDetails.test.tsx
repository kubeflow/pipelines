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
import produce from 'immer';
import { ApiJob, ApiResourceType } from '../apis/job';
import { Apis } from '../lib/Apis';
import { PageProps } from './Page';
import { RouteParams, RoutePage } from '../components/Router';
import { ToolbarActionConfig } from '../components/Toolbar';
import { shallow } from 'enzyme';

class TestRecurringRunDetails extends RecurringRunDetails {
  public async _deleteDialogClosed(confirmed: boolean): Promise<void> {
    super._deleteDialogClosed(confirmed);
  }
}

describe('RecurringRunDetails', () => {
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

  const fullTestJob = {
    created_at: new Date(2018, 8, 5, 4, 3, 2),
    description: 'test job description',
    enabled: true,
    id: 'test-job-id',
    max_concurrency: '50',
    name: 'test job',
    pipeline_spec: { pipeline_id: 'some-pipeline-id' },
    trigger: {
      periodic_schedule: {
        end_time: new Date(2018, 10, 9, 8, 7, 6),
        interval_second: '3600',
        start_time: new Date(2018, 9, 8, 7, 6),
      }
    },
  } as ApiJob;

  function generateProps(): PageProps {
    return {
      history: { push: historyPushSpy } as any,
      location: '' as any,
      match: { params: { [RouteParams.runId]: fullTestJob.id }, isExact: true, path: '', url: '' },
      toolbarProps: RecurringRunDetails.prototype.getInitialToolbarState(),
      updateBanner: updateBannerSpy,
      updateDialog: updateDialogSpy,
      updateSnackbar: updateSnackbarSpy,
      updateToolbar: updateToolbarSpy,
    };
  }

  beforeEach(() => {
    historyPushSpy.mockClear();
    updateBannerSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
    updateToolbarSpy.mockClear();
    getJobSpy.mockImplementation(() => fullTestJob);
    getJobSpy.mockClear();
    deleteJobSpy.mockClear();
    deleteJobSpy.mockImplementation();
    enableJobSpy.mockClear();
    disableJobSpy.mockClear();
    getExperimentSpy.mockImplementation();
    getExperimentSpy.mockClear();
  });

  it('renders a recurring run with periodic schedule', () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    tree.setState({
      run: fullTestJob,
    });
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('renders a recurring run with cron schedule', () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    tree.setState({
      run: produce(fullTestJob, draft => {
        draft.trigger = {
          cron_schedule: {
            cron: '* * * 0 0 !',
            end_time: new Date(2018, 10, 9, 8, 7, 6),
            start_time: new Date(2018, 9, 8, 7, 6),
          }
        };
      }),
    });
    expect(tree).toMatchSnapshot();
    tree.unmount();
  });

  it('loads the recurring run given its id in query params', async () => {
    fullTestJob.resource_references = [];
    shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).not.toHaveBeenCalled();
  });

  it('shows All runs -> run name when there is no experiment', async () => {
    fullTestJob.resource_references = [];
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      breadcrumbs: [
        { displayName: 'All runs', href: RoutePage.RUNS },
        { displayName: fullTestJob.name, href: '' },
      ],
    }));
    tree.unmount();
  });

  it('loads the recurring run and its experiment if it has one', async () => {
    fullTestJob.resource_references = [{ key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } }];
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(getJobSpy).toHaveBeenLastCalledWith(fullTestJob.id);
    expect(getExperimentSpy).toHaveBeenLastCalledWith('test-experiment-id');
    tree.unmount();
  });

  it('shows Experiments -> Experiment name -> run name when there is no experiment', async () => {
    fullTestJob.resource_references = [{ key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } }];
    getExperimentSpy.mockImplementation(id => ({ id, name: 'test experiment name' }));
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      breadcrumbs: [
        { displayName: 'Experiments', href: RoutePage.EXPERIMENTS },
        {
          displayName: 'test experiment name',
          href: RoutePage.EXPERIMENT_DETAILS.replace(
            ':' + RouteParams.experimentId, 'test-experiment-id'),
        },
        { displayName: fullTestJob.name, href: '' },
      ],
    }));
    tree.unmount();
  });

  it('shows error banner if run cannot be fetched', async () => {
    fullTestJob.resource_references = [];
    TestUtils.makeErrorResponseOnce(getJobSpy, 'woops!');
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops!',
      message: `Error: failed to retrieve recurring run: ${fullTestJob.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  it('shows error banner if has experiment but experiment cannot be fetched', async () => {
    fullTestJob.resource_references = [{ key: { id: 'test-experiment-id', type: ApiResourceType.EXPERIMENT } }];
    TestUtils.makeErrorResponseOnce(getExperimentSpy, 'woops!');
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateBannerSpy).toHaveBeenCalledTimes(2); // Once to clear, once to show error
    expect(updateBannerSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      additionalInfo: 'woops!',
      message: `Error: failed to retrieve recurring run: ${fullTestJob.id}. Click Details for more information.`,
      mode: 'error',
    }));
    tree.unmount();
  });

  it('has a Refresh button, clicking it refreshes the run details', async () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    const instance = tree.instance() as RecurringRunDetails;
    const refreshBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(refreshBtn).toBeDefined();
    expect(getJobSpy).toHaveBeenCalledTimes(1);
    await refreshBtn!.action();
    expect(getJobSpy).toHaveBeenCalledTimes(2);
    tree.unmount();
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is enabled', async () => {
    fullTestJob.enabled = true;
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const lastToolbarButtons = updateToolbarSpy.mock.calls[1][0].actions as ToolbarActionConfig[];
    const enableBtn = lastToolbarButtons.find(b => b.title === 'Enable');
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(true);
    const disableBtn = lastToolbarButtons.find(b => b.title === 'Disable');
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(false);
    tree.unmount();
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is disabled', async () => {
    fullTestJob.enabled = false;
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const lastToolbarButtons = updateToolbarSpy.mock.calls[1][0].actions as ToolbarActionConfig[];
    const enableBtn = lastToolbarButtons.find(b => b.title === 'Enable');
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(false);
    const disableBtn = lastToolbarButtons.find(b => b.title === 'Disable');
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(true);
    tree.unmount();
  });

  it('shows enabled Disable, and disabled Enable buttons if the run is undefined', async () => {
    fullTestJob.enabled = undefined;
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    expect(updateToolbarSpy).toHaveBeenCalledTimes(2);
    const lastToolbarButtons = updateToolbarSpy.mock.calls[1][0].actions as ToolbarActionConfig[];
    const enableBtn = lastToolbarButtons.find(b => b.title === 'Enable');
    expect(enableBtn).toBeDefined();
    expect(enableBtn!.disabled).toBe(false);
    const disableBtn = lastToolbarButtons.find(b => b.title === 'Disable');
    expect(disableBtn).toBeDefined();
    expect(disableBtn!.disabled).toBe(true);
    tree.unmount();
  });

  it('shows a delete button', async () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Refresh');
    expect(deleteBtn).toBeDefined();
    tree.unmount();
  });

  it('shows delete dialog when delete button is clicked', async () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      title: 'Delete this recurring run?',
    }));
    tree.unmount();
  });

  it('calls delete API when delete confirmation dialog button is clicked', async () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Delete');
    await confirmBtn.onClick();
    expect(deleteJobSpy).toHaveBeenCalledTimes(1);
    expect(deleteJobSpy).toHaveBeenLastCalledWith('test-job-id');
    tree.unmount();
  });

  it('does not call delete API when delete cancel dialog button is clicked', async () => {
    const tree = shallow(<RecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as RecurringRunDetails;
    const deleteBtn = instance.getInitialToolbarState().actions.find(b => b.title === 'Delete');
    await deleteBtn!.action();
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'Cancel');
    await confirmBtn.onClick();
    expect(deleteJobSpy).not.toHaveBeenCalled();
    // Should not reroute
    expect(historyPushSpy).not.toHaveBeenCalled();
    tree.unmount();
  });

  it('redirects back to parent experiment after delete', async () => {
    const tree = shallow(<TestRecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as TestRecurringRunDetails;
    await instance._deleteDialogClosed(true);
    expect(historyPushSpy).toHaveBeenCalledTimes(1);
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.EXPERIMENTS);
    tree.unmount();
  });

  it('shows snackbar after successful deletion', async () => {
    const tree = shallow(<TestRecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as TestRecurringRunDetails;
    await instance._deleteDialogClosed(true);
    expect(updateSnackbarSpy).toHaveBeenCalledTimes(1);
    expect(updateSnackbarSpy).toHaveBeenLastCalledWith({
      message: 'Successfully deleted recurring run: ' + fullTestJob.name,
      open: true,
    });
    tree.unmount();
  });

  it('shows error dialog after failing deletion', async () => {
    TestUtils.makeErrorResponseOnce(deleteJobSpy, 'could not delete');
    const tree = shallow(<TestRecurringRunDetails {...generateProps()} />);
    await TestUtils.flushPromises();
    const instance = tree.instance() as TestRecurringRunDetails;
    await instance._deleteDialogClosed(true);
    await TestUtils.flushPromises();
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(expect.objectContaining({
      content: 'could not delete',
      title: 'Failed to delete recurring run',
    }));
    // Should not reroute
    expect(historyPushSpy).not.toHaveBeenCalled();
    tree.unmount();
  });

});
