/*
 * Copyright 2021 Arrikto Inc.
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
import { AllJobsList } from './AllJobsList';
import { PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { shallow, ShallowWrapper } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';
import TestUtils from '../TestUtils';
import { fireEvent, render } from '@testing-library/react';

describe('AllJobsList', () => {
  const updateBannerSpy = jest.fn();
  let _toolbarProps: any = { actions: {}, breadcrumbs: [], pageTitle: '' };
  const updateToolbarSpy = jest.fn(toolbarProps => (_toolbarProps = toolbarProps));
  const historyPushSpy = jest.fn();

  let tree: ShallowWrapper;

  function generateProps(): PageProps {
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
    return Object.assign(props, {
      toolbarProps: new AllJobsList(props).getInitialToolbarState(),
    });
  }

  function shallowMountComponent(
    propsPatch: Partial<PageProps & { namespace?: string }> = {},
  ): void {
    tree = shallow(<AllJobsList {...generateProps()} {...propsPatch} />);
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

  it('renders all jobs', () => {
    shallowMountComponent();
    expect(tree).toMatchSnapshot();
  });

  it('lists all jobs in namespace', () => {
    shallowMountComponent({ namespace: 'test-ns' });
    expect(tree.find('JobList').prop('namespaceMask')).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    shallowMountComponent();
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  // it('refreshes the job list when refresh button is clicked', async () => {
  //   const tree = render(<AllJobsList {...generateProps()} />);
  //   await TestUtils.flushPromises()
  //   fireEvent.click(tree.getByText('Refresh'));
  // });

  it('navigates to new run page when new run is clicked', () => {
    shallowMountComponent();

    _toolbarProps.actions[ButtonKeys.NEW_RUN].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_RUN + '?experimentId=');
  });
});
