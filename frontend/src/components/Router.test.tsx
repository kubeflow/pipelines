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

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Router as ReactRouter } from '../lib/Router';
import { HashRouter, MemoryRouter } from 'react-router';
import { createMemoryHistory } from 'history';
import Router, { getSafeReturnPath, RouteConfig, RoutePage } from './Router';
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
    act(() => {
      history.push('/pear');
    });
    await waitFor(() => expect(screen.getByTestId('page-title')).toHaveTextContent(''));
  });

  it('preserves form and toolbar state across HashRouter query changes', async () => {
    class FormPage extends Page<{}, { value: string }> {
      public state = { value: '' };
      public getInitialToolbarState(): ToolbarProps {
        return { pageTitle: 'Initial form', actions: {}, breadcrumbs: [] };
      }
      public async refresh() {}
      public render() {
        return (
          <>
            <input
              aria-label='draft name'
              value={this.state.value}
              onChange={(event) => this.setState({ value: event.target.value })}
            />
            <button onClick={() => this.props.updateToolbar({ pageTitle: 'Edited form' })}>
              edit toolbar
            </button>
            <button onClick={() => this.props.history.push({ search: '?experiment=1' })}>
              select experiment
            </button>
            <button onClick={() => this.props.history.replace({ search: '?experiment=2' })}>
              replace experiment
            </button>
            <button onClick={() => this.props.history.push('/form/second')}>next form</button>
          </>
        );
      }
    }
    window.history.replaceState(null, '', '/#/form/first');
    const view = render(
      <HashRouter>
        <Router configs={[{ path: '/form/:id', Component: FormPage }]} />
      </HashRouter>,
    );
    fireEvent.change(screen.getByLabelText('draft name'), { target: { value: 'Unsaved run' } });
    fireEvent.click(screen.getByText('edit toolbar'));
    for (const button of ['select experiment', 'replace experiment']) {
      fireEvent.click(screen.getByText(button));
      expect(screen.getByLabelText('draft name')).toHaveValue('Unsaved run');
      expect(screen.getByTestId('page-title')).toHaveTextContent('Edited form');
    }
    fireEvent.click(screen.getByText('next form'));
    await waitFor(() => expect(screen.getByLabelText('draft name')).toHaveValue(''));
    expect(screen.getByTestId('page-title')).toHaveTextContent('Initial form');
    view.unmount();
    window.history.replaceState(null, '', '/');
  });

  it('only accepts same-app return paths', () => {
    expect(getSafeReturnPath(RoutePage.RECURRING_RUNS)).toBe(RoutePage.RECURRING_RUNS);
    expect(getSafeReturnPath('https://example.com')).toBeUndefined();
    expect(getSafeReturnPath('//example.com')).toBeUndefined();
    expect(getSafeReturnPath(null)).toBeUndefined();
  });
});
