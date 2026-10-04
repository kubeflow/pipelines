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
import {
  createMemoryRouter,
  HashRouter,
  MemoryRouter,
  RouterProvider,
  useLocation,
} from 'react-router';
import { NavigationProps } from 'src/lib/Navigation';
import { NavigationErrorBoundary } from 'src/atoms/NavigationErrorBoundary';
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
    expect(screen.getByText('404')).toBeVisible();
    expect(screen.getByText('Page Not Found: /does-not-exist')).toBeVisible();
    expect(renderResult.asFragment()).toMatchSnapshot();
  });

  it('updates the not-found pathname and recovers when navigating to a known route', async () => {
    const router = createMemoryRouter([{ path: '*', element: <Router /> }], {
      initialEntries: ['/does-not-exist?view=graph'],
    });
    render(<RouterProvider router={router} />);
    expect(screen.getByText('Page Not Found: /does-not-exist')).toBeVisible();

    await act(() => router.navigate('/another-missing-page'));
    expect(screen.getByText('Page Not Found: /another-missing-page')).toBeVisible();
    expect(screen.queryByText('Page Not Found: /does-not-exist')).not.toBeInTheDocument();

    await act(() => router.navigate('/runs'));
    expect(screen.getByText('Runs page')).toBeVisible();
    expect(screen.queryByText('404')).not.toBeInTheDocument();
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
    const router = createMemoryRouter([{ path: '*', element: <Router configs={configs} /> }], {
      initialEntries: ['/apple'],
    });
    render(<RouterProvider router={router} />);
    expect(screen.getByTestId('page-title')).toHaveTextContent('Apple');
    act(() => {
      router.navigate('/pear');
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
    const router = createMemoryRouter(
      [
        {
          path: '*',
          element: (
            <NavigationErrorBoundary>
              <Router configs={[{ path: RoutePage.RUN_DETAILS, Component: StatefulPage }]} />
            </NavigationErrorBoundary>
          ),
        },
      ],
      {
        initialEntries: ['/runs/details/run-1?task=task-1'],
      },
    );
    const initialLocationKey = router.state.location.key;
    render(<RouterProvider router={router} />);

    act(() => screen.getByRole('button', { name: 'initial' }).click());
    const mountCountBeforeReplace = mountCount;
    await act(() => router.navigate('/runs/details/run-1', { replace: true }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
    expect(router.state.location.key).not.toBe(initialLocationKey);
    expect(screen.getByRole('button', { name: 'selected' })).toBeVisible();
    expect(mountCount).toBe(mountCountBeforeReplace);
    await act(() => router.navigate('/runs/details/run-1?task=task-2'));
    expect(screen.getByRole('button', { name: 'selected' })).toBeVisible();
    await act(() => router.navigate(-1));
    expect(screen.getByRole('button', { name: 'selected' })).toBeVisible();
    expect(mountCount).toBe(mountCountBeforeReplace);
    await act(() => router.navigate('/runs/details/run-2'));
    expect(screen.getByRole('button', { name: 'initial' })).toBeVisible();
    expect(mountCount).toBeGreaterThan(mountCountBeforeReplace);
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
    const router = createMemoryRouter([{ path: '*', element: <Router /> }], {
      initialEntries: ['/runs/details/run-1/execution/123?view=graph#node'],
    });
    render(<RouterProvider router={router} />);

    await waitFor(() => expect(router.state.location.pathname).toBe('/runs/details/run-1'));
    expect(router.state.location.search).toBe('?view=graph&executionRedirect=detail');
    expect(router.state.location.hash).toBe('#node');
    expect(screen.getByText('Run details')).toBeVisible();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'This legacy execution link cannot select the corresponding task automatically.',
    );
    act(() => screen.getByRole('button', { name: 'Close' }).click());
    expect(router.state.location.search).toBe('?view=graph');
    expect(router.state.location.hash).toBe('#node');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it.each(['/executions', '/executions/123'])(
    'explains the redirect from %s without guessing a native task ID',
    async (path) => {
      const router = createMemoryRouter([{ path: '*', element: <Router /> }], {
        initialEntries: [path],
      });
      const view = render(<RouterProvider router={router} />);
      await waitFor(() => expect(router.state.location.pathname).toBe('/runs'));
      expect(screen.getByText('Runs page')).toBeVisible();
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Execution pages have moved to Runs and task details.',
      );
      expect(new URLSearchParams(router.state.location.search).has('task')).toBe(false);
      // The URL marker also explains a bookmarked/reloaded redirect destination.
      const destination = `${router.state.location.pathname}${router.state.location.search}`;
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
    const router = createMemoryRouter(
      [
        {
          path: '*',
          element: <Router configs={[{ path: RoutePage.RUN_DETAILS, Component: ErrorPage }]} />,
        },
      ],
      {
        initialEntries: ['/runs/details/run-1?task=task-1&executionRedirect=detail'],
      },
    );
    render(<RouterProvider router={router} />);
    act(() => screen.getByRole('button', { name: 'Fail loading' }).click());
    act(() => screen.getByRole('button', { name: 'Close' }).click());
    expect(screen.getByText('Current load error')).toBeVisible();
    expect(router.state.location.search).toBe('?task=task-1');
    expect(screen.queryByText(/Execution pages have moved/)).not.toBeInTheDocument();
  });

  it('does not show redirect guidance on ordinary navigation', async () => {
    const router = createMemoryRouter([{ path: '*', element: <Router /> }], {
      initialEntries: ['/runs?executionRedirect=list'],
    });
    render(<RouterProvider router={router} />);
    expect(screen.getByRole('alert')).toBeVisible();
    await act(() => router.navigate('/runs/details/run-1'));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    await act(() => router.navigate('/runs'));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
  it.each([
    ['/pipelines/details', {}],
    ['/pipelines/details/pipeline%20one', { pid: 'pipeline one' }],
    ['/pipelines/details/pipeline%2520one', { pid: 'pipeline%20one' }],
    ['/pipelines/details/pipeline%2Fone/version', { pid: 'pipeline/one' }],
    [
      '/pipelines/details/pipeline-1/version/version%20one',
      { pid: 'pipeline-1', vid: 'version one' },
    ],
  ])('matches optional and encoded pipeline parameters in %s', (path, params) => {
    const ParamsPage = ({ params }: NavigationProps) => <output>{JSON.stringify(params)}</output>;
    render(
      <MemoryRouter initialEntries={[path as string]}>
        <Router
          configs={[
            { path: RoutePage.PIPELINE_DETAILS, Component: ParamsPage },
            { path: RoutePage.PIPELINE_DETAILS_NO_VERSION, Component: ParamsPage },
          ]}
        />
      </MemoryRouter>,
    );
    expect(screen.getByRole('status')).toHaveTextContent(JSON.stringify(params));
  });

  it('preserves encoded hash URLs and task query links', () => {
    const originalUrl = window.location.href;
    window.history.replaceState(
      null,
      '',
      '/#/runs/details/run%252Fone?task=task%2Fiteration+1#node',
    );
    const ParamsPage = ({ params, location }: NavigationProps) => (
      <output>{JSON.stringify({ params, search: location.search, hash: location.hash })}</output>
    );
    try {
      render(
        <HashRouter>
          <NavigationErrorBoundary>
            <Router configs={[{ path: RoutePage.RUN_DETAILS, Component: ParamsPage }]} />
          </NavigationErrorBoundary>
        </HashRouter>,
      );
      expect(screen.getByRole('status')).toHaveTextContent(
        JSON.stringify({
          params: { rid: 'run%2Fone' },
          search: '?task=task%2Fiteration+1',
          hash: '#node',
        }),
      );
    } finally {
      window.history.replaceState(null, '', originalUrl);
    }
  });

  it('reinitializes query-derived forms on navigation', async () => {
    const FormPage = ({ location }: NavigationProps) => {
      const [name, setName] = useState(new URLSearchParams(location.search).get('name') || '');
      return (
        <input aria-label='Name' value={name} onChange={(event) => setName(event.target.value)} />
      );
    };
    const router = createMemoryRouter(
      [
        {
          path: '*',
          element: (
            <NavigationErrorBoundary>
              <Router configs={[{ path: RoutePage.NEW_RUN, Component: FormPage }]} />
            </NavigationErrorBoundary>
          ),
        },
      ],
      { initialEntries: ['/runs/new?name=first'] },
    );
    render(<RouterProvider router={router} />);
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveValue('first');
    await act(() => router.navigate('/runs/new?name=second'));
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveValue('second');
  });

  it('recovers a captured render error after navigation', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const ErrorPage = () => {
      if (!useLocation().search) throw new Error('route failed');
      return <div>Recovered page</div>;
    };
    const router = createMemoryRouter(
      [
        {
          path: '*',
          element: (
            <NavigationErrorBoundary>
              <Router configs={[{ path: RoutePage.RUN_DETAILS, Component: ErrorPage }]} />
            </NavigationErrorBoundary>
          ),
        },
      ],
      { initialEntries: ['/runs/details/run-1'] },
    );
    try {
      render(<RouterProvider router={router} />);
      expect(screen.getByText('Something went wrong.')).toBeVisible();
      await act(() => router.navigate('/runs/details/run-1?view=graph'));
      expect(screen.getByText('Recovered page')).toBeVisible();
      expect(screen.queryByText('Something went wrong.')).not.toBeInTheDocument();
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it('replaces encoded legacy execution links and preserves state when dismissing guidance', async () => {
    const router = createMemoryRouter([{ path: '*', element: <Router /> }], {
      initialEntries: ['/runs', '/runs/details/run%252Fone/execution/123?task=task%2Fone#node'],
    });
    render(<RouterProvider router={router} />);
    await waitFor(() => expect(router.state.location.pathname).toBe('/runs/details/run%252Fone'));
    expect(router.state.historyAction).toBe('REPLACE');
    expect(router.state.location.search).toBe('?task=task%2Fone&executionRedirect=detail');
    expect(router.state.location.hash).toBe('#node');
    await act(() => router.navigate(-1));
    expect(router.state.location.pathname).toBe('/runs');
    await act(() =>
      router.navigate('/runs/details/run-1?task=task-1&executionRedirect=detail#node', {
        state: { from: 'runs' },
      }),
    );
    act(() => screen.getByRole('button', { name: 'Close' }).click());
    expect(router.state.location.search).toBe('?task=task-1');
    expect(router.state.location.hash).toBe('#node');
    expect(router.state.location.state).toEqual({ from: 'runs' });
  });
});
