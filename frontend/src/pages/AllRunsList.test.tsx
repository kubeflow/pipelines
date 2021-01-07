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
import { AllRunsList } from './AllRunsList';
import { PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { RunStorageState } from '../apis/run';
import { shallow, ShallowWrapper } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';
import { TFunction } from 'i18next';

describe('AllRunsList', () => {
  let t: TFunction = (key: string) => key;
  const updateBannerSpy = jest.fn();
  let _toolbarProps: any = {};
  const updateToolbarSpy = jest.fn(toolbarProps => (_toolbarProps = toolbarProps));
  const historyPushSpy = jest.fn();
  const props: PageProps = {
    history: { push: historyPushSpy } as any,
    location: '' as any,
    match: '' as any,
    toolbarProps: _toolbarProps,
    updateBanner: updateBannerSpy,
    updateDialog: jest.fn(),
    updateSnackbar: jest.fn(),
    updateToolbar: updateToolbarSpy,
    t,
  };
  let tree: ShallowWrapper;

  function shallowMountComponent(
    propsPatch: Partial<PageProps & { namespace?: string }> = {},
  ): void {
    tree = shallow(<AllRunsList {...props} {...propsPatch} />);
    // Necessary since the component calls updateToolbar with the toolbar props,
    // then expects to get them back in props
    tree.setProps({ toolbarProps: _toolbarProps });
    updateToolbarSpy.mockClear();
  }

  beforeEach(() => {
    updateBannerSpy.mockClear();
    updateToolbarSpy.mockClear();
    historyPushSpy.mockClear();
  });

  afterEach(() => tree.unmount());

  it('renders all runs', () => {
    shallowMountComponent();
    expect(tree).toMatchSnapshot();
  });

  it('lists all runs in namespace', () => {
    shallowMountComponent({ namespace: 'test-ns' });
    expect(tree.find('RunList').prop('namespaceMask')).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    shallowMountComponent();
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({ t });
  });

  it('only enables clone button when exactly one run is selected', () => {
    shallowMountComponent();
    const cloneBtn = _toolbarProps.actions[ButtonKeys.CLONE_RUN];
    expect(cloneBtn.disabled).toBeTruthy();
    tree.find('RunList').simulate('selectionChange', ['run1']);
    expect(cloneBtn.disabled).toBeFalsy();
    tree.find('RunList').simulate('selectionChange', ['run1', 'run2']);
    expect(cloneBtn.disabled).toBeTruthy();
  });

  it('enables archive button when at least one run is selected', () => {
    shallowMountComponent();
    const archiveBtn = _toolbarProps.actions[ButtonKeys.ARCHIVE];
    expect(archiveBtn.disabled).toBeTruthy();
    tree.find('RunList').simulate('selectionChange', ['run1']);
    expect(archiveBtn.disabled).toBeFalsy();
    tree.find('RunList').simulate('selectionChange', ['run1', 'run2']);
    expect(archiveBtn.disabled).toBeFalsy();
  });

  it('refreshes the run list when refresh button is clicked', () => {
    shallowMountComponent();
    const spy = jest.fn();
    (tree.instance() as any)._runlistRef = { current: { refresh: spy } };
    _toolbarProps.actions[ButtonKeys.REFRESH].action();
    expect(spy).toHaveBeenLastCalledWith();
  });

  it('navigates to new run page when clone is clicked', () => {
    shallowMountComponent();
    tree.find('RunList').simulate('selectionChange', ['run1']);
    _toolbarProps.actions[ButtonKeys.CLONE_RUN].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_RUN + '?cloneFromRun=run1');
  });

  it('navigates to compare page when compare button is clicked', () => {
    shallowMountComponent();
    tree.find('RunList').simulate('selectionChange', ['run1', 'run2', 'run3']);
    _toolbarProps.actions[ButtonKeys.COMPARE].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.COMPARE + '?runlist=run1,run2,run3');
  });

  it('shows thrown error in error banner', () => {
    shallowMountComponent();
    const instance = tree.instance() as AllRunsList;
    const spy = jest.spyOn(instance, 'showPageError');
    instance.forceUpdate();
    const errorMessage = 'test error message';
    const error = new Error('error object message');
    tree.find('RunList').simulate('error', errorMessage, error);
    expect(spy).toHaveBeenLastCalledWith(errorMessage, error);
  });

  it('shows a list of available runs', () => {
    shallowMountComponent();
    expect(tree.find('RunList').prop('storageState')).toBe(RunStorageState.AVAILABLE.toString());
  });
});
