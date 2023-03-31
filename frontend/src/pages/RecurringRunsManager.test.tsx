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
import TestUtils from 'src/TestUtils';
import { ListRequest, Apis } from 'src/lib/Apis';
import { shallow, ReactWrapper, ShallowWrapper } from 'enzyme';
import RecurringRunsManager, { RecurringRunListProps } from './RecurringRunsManager';
import { V2beta1RecurringRun, V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';

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
  const listRecurringRunsSpy = jest.spyOn(Apis.recurringRunServiceApi, 'listRecurringRuns');
  const enableRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'enableRecurringRun');
  const disableRecurringRunSpy = jest.spyOn(Apis.recurringRunServiceApi, 'disableRecurringRun');
  jest.spyOn(console, 'error').mockImplementation();

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
    listRecurringRunsSpy.mockReset();
    listRecurringRunsSpy.mockImplementation(() => ({ recurringRuns: RECURRINGRUNS }));
    enableRecurringRunSpy.mockReset();
    disableRecurringRunSpy.mockReset();
    updateDialogSpy.mockReset();
    updateSnackbarSpy.mockReset();
  });

  afterEach(() => tree.unmount());

  it('calls API to load recurring runs', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._loadRuns({});
    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    expect(listRecurringRunsSpy).toHaveBeenLastCalledWith(
      undefined,
      undefined,
      undefined,
      undefined,
      undefined,
      'test-experiment',
    );
    expect(tree.state('runs')).toEqual(RECURRINGRUNS);
    expect(tree).toMatchSnapshot();
  });

  it('shows error dialog if listing fails', async () => {
    TestUtils.makeErrorResponseOnce(listRecurringRunsSpy, 'woops!');
    jest.spyOn(console, 'error').mockImplementation();
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._loadRuns({});
    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'List recurring run configs request failed with:\nwoops!',
        title: 'Error retrieving recurring run configs',
      }),
    );
    expect(tree.state('runs')).toEqual([]);
  });

  it('calls API to enable run', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', true);
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith('test-run');
  });

  it('calls API to disable run', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', false);
    expect(disableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(disableRecurringRunSpy).toHaveBeenLastCalledWith('test-run');
  });

  it('shows error if enable API call fails', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(enableRecurringRunSpy, 'cannot enable');
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', true);
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Error changing enabled state of recurring run:\ncannot enable',
        title: 'Error',
      }),
    );
  });

  it('shows error if disable API call fails', async () => {
    tree = shallow(<TestRecurringRunsManager {...generateProps()} />);
    TestUtils.makeErrorResponseOnce(disableRecurringRunSpy, 'cannot disable');
    await (tree.instance() as TestRecurringRunsManager)._setEnabledState('test-run', false);
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'Error changing enabled state of recurring run:\ncannot disable',
        title: 'Error',
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
    expect(disableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(disableRecurringRunSpy).toHaveBeenLastCalledWith(RECURRINGRUNS[0].recurring_run_id);
  });

  it('renders an enable button if the run is disabled, clicking the button calls enable API', async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(1);
    expect(enableBtn).toMatchSnapshot();

    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith(RECURRINGRUNS[1].recurring_run_id);
  });

  it("renders an enable button if the run's enabled field is undefined, clicking the button calls enable API", async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(2);
    expect(enableBtn).toMatchSnapshot();

    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(enableRecurringRunSpy).toHaveBeenCalledTimes(1);
    expect(enableRecurringRunSpy).toHaveBeenLastCalledWith(RECURRINGRUNS[2].recurring_run_id);
  });

  it('reloads the list of runs after enable/disabling', async () => {
    tree = TestUtils.mountWithRouter(<RecurringRunsManager {...generateProps()} />);
    await TestUtils.flushPromises();
    tree.update();

    const enableBtn = tree.find('.tableRow Button').at(0);
    expect(enableBtn).toMatchSnapshot();

    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(1);
    enableBtn.simulate('click');
    await TestUtils.flushPromises();
    expect(listRecurringRunsSpy).toHaveBeenCalledTimes(2);
  });
});
