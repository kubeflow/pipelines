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

import { shallow, mount } from 'enzyme';
import Router, { RouteConfig } from './Router';
import { Router as ReactRouter } from 'react-router';
import { Page } from '../pages/Page';
import { ToolbarProps } from './Toolbar';
import { createMemoryHistory } from 'history';
import { TFunction } from 'i18next';

describe('Router', () => {
  it('initial render', () => {
    const tree = shallow(<Router />);
    expect(tree).toMatchSnapshot();
  });

  it('does not share state between pages', () => {
    class ApplePage extends Page<{}, {}> {
      getInitialToolbarState(): ToolbarProps {
        return {
          pageTitle: 'Apple',
          actions: {},
          breadcrumbs: [],
        };
      }
      async refresh() {}
      render() {
        return <div>apple</div>;
      }
    }
    const configs: RouteConfig[] = [
      {
        path: '/apple',
        Component: ApplePage,
      },
      {
        path: '/pear',
        Component: () => {
          return <div>pear</div>;
        },
      },
    ];
    const history = createMemoryHistory({
      initialEntries: ['/apple'],
    });
    const tree = mount(
      <ReactRouter history={history}>
        <Router configs={configs} />
      </ReactRouter>,
    );
    expect(tree.getDOMNode().querySelector('[data-testid=page-title]')!.textContent).toEqual(
      'Apple',
    );
    // When visiting the second page, page title should be reset automatically.
    history.push('/pear');
    expect(tree.getDOMNode().querySelector('[data-testid=page-title]')!.textContent).toEqual('');
  });
});
