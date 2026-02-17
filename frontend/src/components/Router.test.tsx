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
import { render, screen, waitFor } from '@testing-library/react';
import { Router as ReactRouter } from 'react-router';
import { MemoryRouter } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import Router, { RouteConfig } from './Router';
import { Page } from '../pages/Page';
import { ToolbarProps } from './Toolbar';

describe('Router', () => {
  it('initial render', () => {
    const renderResult = render(
      <MemoryRouter initialEntries={['/does-not-exist']}>
        <Router />
      </MemoryRouter>,
    );
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('does not share state between pages', async () => {
    class ApplePage extends Page<{}, {}> {
      public getInitialToolbarState(): ToolbarProps {
        return {
          pageTitle: 'Apple',
          actions: {},
          breadcrumbs: [],
        };
      }
      public async refresh() {}
      public render() {
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
    render(
      <ReactRouter history={history}>
        <Router configs={configs} />
      </ReactRouter>,
    );
    expect(screen.getByTestId('page-title')).toHaveTextContent('Apple');
    history.push('/pear');
    await waitFor(() => expect(screen.getByTestId('page-title')).toHaveTextContent(''));
  });
});
