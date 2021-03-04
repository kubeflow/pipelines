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
import { AllRecurringRunsList } from './AllRecurringRunsList';
import { PageProps } from './Page';
import { RoutePage } from '../components/Router';
import { shallow, ShallowWrapper } from 'enzyme';
import { ButtonKeys } from '../lib/Buttons';
import TestUtils from '../TestUtils';
import { fireEvent, render } from '@testing-library/react';

describe('AllRecurringRunsList', () => {
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
      toolbarProps: new AllRecurringRunsList(props).getInitialToolbarState(),
    });
  }

  function shallowMountComponent(
    propsPatch: Partial<PageProps & { namespace?: string }> = {},
  ): void {
    tree = shallow(<AllRecurringRunsList {...generateProps()} {...propsPatch} />);
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

  it('renders all recurring runs', () => {
    shallowMountComponent();
    expect(tree).toMatchInlineSnapshot(`
      <div
        className="page"
      >
        <RecurringRunList
          history={
            Object {
              "push": [MockFunction],
            }
          }
          location=""
          match=""
          onError={[Function]}
          onSelectionChange={[Function]}
          refreshCount={0}
          selectedIds={Array []}
          toolbarProps={
            Object {
              "actions": Object {
                "newRun": Object {
                  "action": [Function],
                  "icon": [Function],
                  "id": "createNewRunBtn",
                  "outlined": true,
                  "primary": true,
                  "style": Object {
                    "minWidth": 130,
                  },
                  "title": "Create run",
                  "tooltip": "Create a new run",
                },
                "refresh": Object {
                  "action": [Function],
                  "id": "refreshBtn",
                  "title": "Refresh",
                  "tooltip": "Refresh the list",
                },
              },
              "breadcrumbs": Array [],
              "pageTitle": "Recurring Runs",
            }
          }
          updateBanner={[MockFunction]}
          updateDialog={[MockFunction]}
          updateSnackbar={[MockFunction]}
          updateToolbar={[MockFunction]}
        />
      </div>
    `);
  });

  it('lists all recurring runs in namespace', () => {
    shallowMountComponent({ namespace: 'test-ns' });
    expect(tree.find('RecurringRunList').prop('namespaceMask')).toEqual('test-ns');
  });

  it('removes error banner on unmount', () => {
    shallowMountComponent();
    tree.unmount();
    expect(updateBannerSpy).toHaveBeenCalledWith({});
  });

  // TODO: We want to test that clicking the refresh button in AllRecurringRunsList calls the
  //  RecurringRunList.refresh method. This is not straightforward because `render` does not
  //  render the toolbar in this case. RoutedPage is where the page level common elements are
  //  rendered in KFP UI. However, in tests, we built a util that generates similar page callbacks
  //  and passes them to the tested component without actually rendering the page common elements.
  // it('refreshes the recurring run list when refresh button is clicked', async () => {
  //   const tree = render(<AllRecurringRunsList {...generateProps()} />);
  //   await TestUtils.flushPromises()
  //   fireEvent.click(tree.getByText('Refresh'));
  // });

  it('navigates to new run page when new run is clicked', () => {
    shallowMountComponent();

    _toolbarProps.actions[ButtonKeys.NEW_RUN].action();
    expect(historyPushSpy).toHaveBeenLastCalledWith(RoutePage.NEW_RUN + '?experimentId=');
  });
});
