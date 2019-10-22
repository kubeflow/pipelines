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
import Archive from './Archive';
import TestUtils from '../TestUtils';
import { PageProps } from './Page';
import { RunStorageState } from '../apis/run';
import { ShallowWrapper, shallow } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';

describe('Archive', () => {
  const updateBannerSpy = jest.fn();
  const updateToolbarSpy = jest.fn();
  const historyPushSpy = jest.fn();
  let tree: ShallowWrapper;

  function generateProps(): PageProps {
    return TestUtils.generatePageProps(
      Archive,
      {} as any,
      {} as any,
      historyPushSpy,
      updateBannerSpy,
      null,
      updateToolbarSpy,
      null,
    );
  }

  beforeEach(() => {
    updateBannerSpy.mockClear();
    updateToolbarSpy.mockClear();
    historyPushSpy.mockClear();
  });

  afterEach(() => tree.unmount());

  it('renders archived runs', () => {
    tree = shallow(<Archive {...generateProps()} />);
    expect(tree).toMatchSnapshot();
  });

  it('removes error banner on unmount', () => {
    tree = shallow(<Archive {...generateProps()} />);
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('only enables restore button when at least one run is selected', () => {
    tree = shallow(<Archive {...generateProps()} />);
    TestUtils.flushPromises();
    tree.update();
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeTruthy();
    tree.find('RunList').simulate('selectionChange', ['run1']);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeFalsy();
    tree.find('RunList').simulate('selectionChange', ['run1', 'run2']);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeFalsy();
    tree.find('RunList').simulate('selectionChange', []);
    expect(TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.RESTORE).disabled).toBeTruthy();
  });

  it('refreshes the run list when refresh button is clicked', async () => {
    tree = shallow(<Archive {...generateProps()} />);
    const spy = jest.fn();
    (tree.instance() as any)._runlistRef = { current: { refresh: spy } };
    await TestUtils.getToolbarButton(updateToolbarSpy, ButtonKeys.REFRESH).action();
    expect(spy).toHaveBeenLastCalledWith();
  });

  it('shows a list of available runs', () => {
    tree = shallow(<Archive {...generateProps()} />);
    expect(tree.find('RunList').prop('storageState')).toBe(RunStorageState.ARCHIVED.toString());
  });
});
