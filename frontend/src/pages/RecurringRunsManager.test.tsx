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
import TestUtils from '../TestUtils';
import { ListRequest, Apis } from '../lib/Apis';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import RecurringRunsManager, { RecurringRunListProps } from './RecurringRunsManager';
import { ApiJob, ApiResourceType } from '../apis/job';

let mockedValue = '';
jest.mock('react-i18next', () => ({
  // this mock makes sure any components using the translate HoC receive the t function as a prop
  withTranslation: () => (Component: { defaultProps: any }) => {
    Component.defaultProps = { ...Component.defaultProps, t: () => mockedValue };
    return Component;
  },
}));

describe('RecurringRunsManager', () => {
  class TestRecurringRunsManager extends RecurringRunsManager {
    public async _loadRuns(request: ListRequest): Promise<string> {
      return super._loadRuns(request);
    }
    public _setEnabledState(id: string, enabled: boolean): Promise<void> {
      return super._setEnabledState(id, enabled);
    }
  }

  let tree: ReactWrapper | ShallowWrapper;

  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  const listJobsSpy = jest.spyOn(Apis.jobServiceApi, 'listJobs');
  const enableJobSpy = jest.spyOn(Apis.jobServiceApi, 'enableJob');
  const disableJobSpy = jest.spyOn(Apis.jobServiceApi, 'disableJob');
  jest.spyOn(console, 'error').mockImplementation();

  const JOBS: ApiJob[] = [
    {
      created_at: new Date(2018, 10, 9, 8, 7, 6),
      enabled: true,
      id: 'job1',
      name: 'test recurring run name',
    },
    {
      created_at: new Date(2018, 10, 9, 8, 7, 6),
      enabled: false,
      id: 'job2',
      name: 'test recurring run name2',
    },
    {
      created_at: new Date(2018, 10, 9, 8, 7, 6),
      id: 'job3',
      name: 'test recurring run name3',
    },
  ];

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

  beforeEach(() => {
    listJobsSpy.mockReset();
    listJobsSpy.mockImplementation(() => ({ jobs: JOBS }));
    enableJobSpy.mockReset();
    disableJobSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
  });

  afterEach(() => tree.unmount());

  it('calls API to load recurring runs', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._loadRuns({});
    expect(listJobsSpy).toHaveBeenCalledTimes(1);
    expect(listJobsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      ApiResourceType.EXPERIMENT,
      'test-experiment',
      undefined,
    );
    expect(tree.state('runs')).toEqual(JOBS);
    expect(tree).toMatchSnapshot();
  });

  it('shows error dialog if listing fails', async () => {
    mockedValue = 'example1';
    TestUtils.makeErrorResponseOnce(listJobsSpy, 'woops!');
    jest.spyOn(console, 'error').mockImplementation();
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._loadRuns({});
    expect(listJobsSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        //content: 'List recurring run configs request failed with:\nwoops!',
        //title: 'Error retrieving recurring run configs',
        content: 'example1:\nwoops!',
        title: 'example1',
      }),
    );
    expect(tree.state('runs')).toEqual([]);
  });

  it('calls API to enable run', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', true);
    expect(enableJobSpy).toHaveBeenCalledTimes(1);
    expect(enableJobSpy).toHaveBeenLastCalledWith('test-run');
  });

  it('calls API to disable run', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', false);
    expect(disableJobSpy).toHaveBeenCalledTimes(1);
    expect(disableJobSpy).toHaveBeenLastCalledWith('test-run');
  });

  it('shows error if enable API call fails', async () => {
    mockedValue = 'example1';
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(enableJobSpy, 'cannot enable');
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', true);
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        //content: 'Error changing enabled state of recurring run:\ncannot enable',
        //title: 'Error',
        content: 'example1:\ncannot enable',
        title: 'example1',
      }),
    );
  });

  it('shows error if disable API call fails', async () => {
    mockedValue = 'example1';
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(disableJobSpy, 'cannot disable');
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', false);
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        //content: 'Error changing enabled state of recurring run:\ncannot disable',
        //title: 'Error',
        content: 'example1:\ncannot disable',
        title: 'example1',
      }),
    );
  });

  it('renders run name as link to its details page', () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    expect(
      (tree.instance() as RecurringRunsManager)._nameCustomRenderer({
        id: 'run-id',
        value: 'test-run',
      }),
    ).toMatchSnapshot();
  });

  it('renders a disable button if the run is enabled, clicking the button calls disable API', async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(0);
    expect(enableBtn).toMatchSnapshot();

    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(disableJobSpy).toHaveBeenCalledTimes(1);
    expect(disableJobSpy).toHaveBeenLastCalledWith(JOBS[0].id);
  });

  it('renders an enable button if the run is disabled, clicking the button calls enable API', async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(1);
    expect(enableBtn).toMatchSnapshot();

    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(enableJobSpy).toHaveBeenCalledTimes(1);
    expect(enableJobSpy).toHaveBeenLastCalledWith(JOBS[1].id);
  });

  it("renders an enable button if the run's enabled field is undefined, clicking the button calls enable API", async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(2);
    expect(enableBtn).toMatchSnapshot();

    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(enableJobSpy).toHaveBeenCalledTimes(1);
    expect(enableJobSpy).toHaveBeenLastCalledWith(JOBS[2].id);
  });

  it('reloads the list of runs after enable/disabling', async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(0);
    expect(enableBtn).toMatchSnapshot();

    expect(listJobsSpy).toHaveBeenCalledTimes(1);
    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(listJobsSpy).toHaveBeenCalledTimes(2);
  });
});
