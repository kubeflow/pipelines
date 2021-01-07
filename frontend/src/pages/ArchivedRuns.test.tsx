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
import { ArchivedRuns } from './ArchivedRuns';
import TestUtils from '../TestUtils';
import { PageProps } from './Page';
import { RunStorageState } from '../apis/run';
import { ShallowWrapper, shallow } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';
import { Apis } from '../lib/Apis';
import { TFunction } from 'i18next';

describe('ArchivedRuns', () => {
  const updateBannerSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  const deleteRunSpy = jest.spyOn(Apis.runServiceApi, 'deleteRun');
  const updateDialogSpy = jest.fn();
  const updateSnackbarSpy = jest.fn();
  let tree: ShallowWrapper;
  let t: TFunction = (key: string) => key;

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      ArchivedRuns,
      {} as any,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      updateDialogSpy,
      updateToolbarSpy,
      updateSnackbarSpy,
      { t },
    );
  }

  beforeEach(() => {
    updateBannerSpy.mockClear();
    updateToolbarSpy.mockClear();
    historyPushSpy.mockClear();
    deleteRunSpy.mockClear();
    updateDialogSpy.mockClear();
    updateSnackbarSpy.mockClear();
  });

  afterEach(() => tree.unmount());

  it('renders archived runs', () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);
    expect(tree).toMatchSnapshot();
  });

  it('lists archived runs in namespace', () => {
    tree = shallow(<ArchivedRuns {...generateProps()} namespace='test-ns' />);
    expect(tree.find('RunList').prop('namespaceMask')).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({ t });
  });

  it('enables restore and delete button when at least one run is selected', () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);
    TestUtils.flushPromises();
    tree.update();
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeTruthy();
    expect(
      TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.DELETE_RUN).disabled,
    ).toBeTruthy();
    tree.find('RunList').simulate('selectionChange', ['run1']);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeFalsy();
    expect(
      TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.DELETE_RUN).disabled,
    ).toBeFalsy();
    tree.find('RunList').simulate('selectionChange', ['run1', 'run2']);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeFalsy();
    expect(
      TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.DELETE_RUN).disabled,
    ).toBeFalsy();
    tree.find('RunList').simulate('selectionChange', []);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeTruthy();
    expect(
      TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.DELETE_RUN).disabled,
    ).toBeTruthy();
  });

  it('refreshes the run list when refresh button is clicked', async () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);
    const spy = jest.fn();
    (tree.instance() as any)._runlistRef = { current: { refresh: spy } };
    await TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.REFRESH).action();
    expect(spy).toHaveBeenLastCalledWith();
  });

  it('shows a list of available runs', () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);
    expect(tree.find('RunList').prop('storageState')).toBe(RunStorageState.ARCHIVED.toString());
  });

  it('cancells deletion when Cancel is clicked', async () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);

    // Click delete button to delete selected ids.
    const deleteBtn = (tree.instance() as ArchivedRuns).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();

    // Dialog pops up to confirm the deletion.
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'common:deleteSelectedRuns',
      }),
    );

    // Cancel deletion.
    const call = updateDialogSpy.mock.calls[0][0];
    const cancelBtn = call.buttons.find((b: any) => b.text === 'common:cancel');
    await cancelBtn.onClick();
    expect(deleteRunSpy).not.toHaveBeenCalled();
  });

  it('deletes selected ids when Confirm is clicked', async () => {
    tree = shallow(<ArchivedRuns {...generateProps()} />);
    tree.setState({ selectedIds: ['id1', 'id2', 'id3'] });

    // Mock the behavior where the deletion of id1 fails, the deletion of id2 and id3 succeed.
    TestUtils.makeErrorResponseOnce(deleteRunSpy, 'woops');
    deleteRunSpy.mockImplementation(() => Promise.resolve({}));

    // Click delete button to delete selected ids.
    const deleteBtn = (tree.instance() as ArchivedRuns).getInitialToolbarState().actions[
      ButtonKeys.DELETE_RUN
    ];
    await deleteBtn!.action();

    // Dialog pops up to confirm the deletion.
    expect(updateDialogSpy).toHaveBeenCalledTimes(1);
    expect(updateDialogSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        content: 'common:deleteSelectedRuns',
      }),
    );

    // Confirm.
    const call = updateDialogSpy.mock.calls[0][0];
    const confirmBtn = call.buttons.find((b: any) => b.text === 'common:delete');
    await confirmBtn.onClick();
    await deleteRunSpy;
    await TestUtils.flushPromises();
    tree.update();
    expect(deleteRunSpy).toHaveBeenCalledTimes(3);
    expect(deleteRunSpy).toHaveBeenCalledWith('id1');
    expect(deleteRunSpy).toHaveBeenCalledWith('id2');
    expect(deleteRunSpy).toHaveBeenCalledWith('id3');
    expect(tree.state('selectedIds')).toEqual(['id1']); // id1 is left over since its deletion failed.
  });
});
