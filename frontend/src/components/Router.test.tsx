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

import { act, render, screen, waitFor } from '@testing-library/react';
import { Router as ReactRouter } from 'react-router';
import { MemoryRouter } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import { useEffect, useState } from 'react';
import Router, { getSafeReturnPath, RouteConfig, RoutePage, RoutePageFactory } from './Router';
import { Page } from '../pages/Page';
import { ToolbarProps } from './Toolbar';

vi.mock('src/pages/RunDetailsRouter', () => ({
  default: () => <div>Run details</div>,
}));

vi.mock('src/pages/AllRunsAndArchive', () => ({
  default: () => <div>Runs page</div>,
  AllRunsAndArchiveTab: { RUNS: 0, ARCHIVE: 1 },
}));

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

  it('preserves Run Details state when only the query changes', async () => {
    let mountCount = 0;
    const StatefulPage = () => {
      const [selection, setSelection] = useState('initial');
      useEffect(() => {
        mountCount++;
      }, []);
      return <button onClick={() => setSelection('selected')}>{selection}</button>;
    };
    const history = createMemoryHistory({
      initialEntries: ['/runs/details/run-1?task=task-1'],
    });
    const initialLocationKey = history.location.key;
    render(
      <ReactRouter history={history}>
        <Router configs={[{ path: RoutePage.RUN_DETAILS, Component: StatefulPage }]} />
      </ReactRouter>,
    );

    act(() => screen.getByRole('button', { name: 'initial' }).click());
    const mountCountBeforeReplace = mountCount;
    act(() => history.replace('/runs/details/run-1'));

    await waitFor(() => expect(history.location.search).toBe(''));
    expect(history.location.key).not.toBe(initialLocationKey);
    expect(screen.getByRole('button', { name: 'selected' })).toBeVisible();
    expect(mountCount).toBe(mountCountBeforeReplace);
  });

  it('only accepts same-app return paths', () => {
    expect(getSafeReturnPath(RoutePage.RECURRING_RUNS)).toBe(RoutePage.RECURRING_RUNS);
    expect(getSafeReturnPath('https://example.com')).toBeUndefined();
    expect(getSafeReturnPath('//example.com')).toBeUndefined();
    expect(getSafeReturnPath(null)).toBeUndefined();
  });

  it('builds native task links without putting task IDs in the path', () => {
    expect(RoutePageFactory.runDetailsTask('run-1', 'task/iteration 1')).toBe(
      '/runs/details/run-1?task=task%2Fiteration+1',
    );
  });

  it('redirects legacy run execution links to canonical run details', async () => {
    const history = createMemoryHistory({
      initialEntries: ['/runs/details/run-1/execution/123?view=graph#node'],
    });
    render(
      <ReactRouter history={history}>
        <Router />
      </ReactRouter>,
    );

    await waitFor(() => expect(history.location.pathname).toBe('/runs/details/run-1'));
    expect(history.location.search).toBe('?view=graph&executionRedirect=detail');
    expect(history.location.hash).toBe('#node');
    expect(screen.getByText('Run details')).toBeVisible();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'This legacy execution link cannot select the corresponding task automatically.',
    );
    act(() => screen.getByRole('button', { name: 'Close' }).click());
    expect(history.location.search).toBe('?view=graph');
    expect(history.location.hash).toBe('#node');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it.each(['/executions', '/executions/123'])(
    'explains the redirect from %s without guessing a native task ID',
    async (path) => {
      const history = createMemoryHistory({ initialEntries: [path] });
      const view = render(
        <ReactRouter history={history}>
          <Router />
        </ReactRouter>,
      );
      await waitFor(() => expect(history.location.pathname).toBe('/runs'));
      expect(screen.getByText('Runs page')).toBeVisible();
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Execution pages have moved to Runs and task details.',
      );
      expect(new URLSearchParams(history.location.search).has('task')).toBe(false);
      // The URL marker also explains a bookmarked/reloaded redirect destination.
      const destination = `${history.location.pathname}${history.location.search}`;
      view.unmount();
      render(
        <MemoryRouter initialEntries={[destination]}>
          <Router />
        </MemoryRouter>,
      );
      expect(screen.getByRole('alert')).toBeVisible();
      act(() => screen.getByRole('button', { name: 'Close' }).click());
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    },
  );

  it('does not clear page errors or task selection when dismissing navigation guidance', () => {
    class ErrorPage extends Page<{}, {}> {
      public getInitialToolbarState(): ToolbarProps {
        return { pageTitle: 'Run', actions: {}, breadcrumbs: [] };
      }
      public async refresh() {}
      public render() {
        return (
          <button
            onClick={() =>
              this.props.updateBanner({ message: 'Current load error', mode: 'error' })
            }
          >
            Fail loading
          </button>
        );
      }
    }
    const history = createMemoryHistory({
      initialEntries: ['/runs/details/run-1?task=task-1&executionRedirect=detail'],
    });
    render(
      <ReactRouter history={history}>
        <Router configs={[{ path: RoutePage.RUN_DETAILS, Component: ErrorPage }]} />
      </ReactRouter>,
    );
    act(() => screen.getByRole('button', { name: 'Fail loading' }).click());
    act(() => screen.getByRole('button', { name: 'Close' }).click());
    expect(screen.getByText('Current load error')).toBeVisible();
    expect(history.location.search).toBe('?task=task-1');
    expect(screen.queryByText(/Execution pages have moved/)).not.toBeInTheDocument();
  });

  it('does not show redirect guidance on ordinary navigation', () => {
    const history = createMemoryHistory({ initialEntries: ['/runs?executionRedirect=list'] });
    render(
      <ReactRouter history={history}>
        <Router />
      </ReactRouter>,
    );
    expect(screen.getByRole('alert')).toBeVisible();
    act(() => history.push('/runs/details/run-1'));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    act(() => history.push('/runs'));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
