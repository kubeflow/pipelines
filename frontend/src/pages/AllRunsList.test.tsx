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
import AllRunsList from './AllRunsList';
import { RoutePage } from '../components/Router';
import { shallow, ShallowWrapper } from 'enzyme';
import { PageProps } from './Page';

describe('AllRunsList', () => {
  const updateBannerSpy = jest.fn();
  let _toolbarProps: any = {};
  const updateToolbarSpy = jest.fn(toolbarProps => _toolbarProps = toolbarProps);
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
  };

  function mountComponent(): ShallowWrapper {
    const tree = shallow(<AllRunsList {...props} />);
    // Necessary since the component calls updateToolbar with the toolbar props,
    // then expects to get them back in props
    tree.setProps({ toolbarProps: _toolbarProps });
    updateToolbarSpy.mockClear();
    return tree;
  }

  beforeEach(() => {
    updateBannerSpy.mockClear();
    updateToolbarSpy.mockClear();
  });

  it('renders all runs', () => {
    expect(mountComponent()).toMatchSnapshot();
  });

  it('removes error banner on unmount', () => {
    const tree = mountComponent();
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  it('enables clone button when one run is selected', () => {
    const tree = mountComponent();

    tree.find('RunList').simulate('selectionChange', ['run1']);
    expect(shallow(<AllRunsList {...props} />)).toMatchSnapshot();
  });

  it('disables clone button and enables compare button when multiple runs are selected', () => {
    const tree = mountComponent();

    tree.find('RunList').simulate('selectionChange', ['run1', 'run2']);
    expect(shallow(<AllRunsList {...props} />)).toMatchSnapshot();
  });

  it('refreshes the run list when refresh button is clicked', () => {
    const tree = mountComponent();
    const spy = jest.fn();
    (tree.instance() as any)._runlistRef = { current: { refresh: spy } };
    _toolbarProps.actions.find((a: any) => a.title === 'Refresh').action();
    expect(spy).toHaveBeenLastCalledWith();
  });

  it('navigates to new run page when clone is clicked', () => {
    const tree = mountComponent();
    tree.find('RunList').simulate('selectionChange', ['run1']);
    _toolbarProps.actions.find((a: any) => a.title === 'Clone run').action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_RUN + '?cloneFromRun=run1');
  });

  it('navigates to compare page when compare button is clicked', () => {
    const tree = mountComponent();
    tree.find('RunList').simulate('selectionChange', ['run1', 'run2', 'run3']);
    _toolbarProps.actions.find((a: any) => a.title === 'Compare runs').action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.COMPARE + '?runlist=run1,run2,run3');
  });

  it('shows thrown error in error banner', () => {
    const tree = mountComponent();
    const instance = tree.instance() as AllRunsList;
    const spy = jest.spyOn(instance, 'showPageError');
    instance.forceUpdate();
    const errorMessage = 'test error message';
    const error = new Error('error object message');
    tree.find('RunList').simulate('error', errorMessage, error);
    expect(spy).toHaveBeenLastCalledWith(errorMessage, error);
  });
});
