/*
 * Copyright 2026 The Kubeflow Authors
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
import { createMemoryHistory } from 'history';
import { HashRouter, MemoryRouter } from 'react-router';
import { Redirect, Route, Router, Switch } from './Router';

describe('class-page router boundary', () => {
  it('navigates and replaces query state with the production HashRouter', async () => {
    window.history.replaceState(null, '', '/#/runs/123?old=1');
    const initialLength = window.history.length;
    const view = render(
      <HashRouter>
        <Route
          render={({ history: navigation, location }) => (
            <>
              <span>
                {location.pathname}
                {location.search}
              </span>
              <button onClick={() => navigation.push({ search: '?page=2' })}>push hash</button>
              <button onClick={() => navigation.replace({ search: '?page=3' })}>
                replace hash
              </button>
              <button onClick={navigation.goBack}>back hash</button>
            </>
          )}
        />
      </HashRouter>,
    );
    fireEvent.click(screen.getByText('push hash'));
    expect(window.location.hash).toBe('#/runs/123?page=2');
    expect(window.history.length).toBe(initialLength + 1);
    fireEvent.click(screen.getByText('replace hash'));
    expect(window.location.hash).toBe('#/runs/123?page=3');
    expect(window.history.length).toBe(initialLength + 1);
    fireEvent.click(screen.getByText('back hash'));
    await waitFor(() => expect(screen.getByText('/runs/123?old=1')).toBeInTheDocument());
    view.unmount();
    window.history.replaceState(null, '', '/');
  });

  it('keeps ordered matching and nested artifact routes', () => {
    render(
      <MemoryRouter initialEntries={['/artifacts/123/lineage']}>
        <Switch>
          <Route path='/artifacts' exact>
            list
          </Route>
          <Route
            path='/artifacts/:id'
            render={({ match }) => (
              <Switch>
                <Route path={match.path} exact>
                  overview
                </Route>
                <Route path={`${match.path}/lineage`} exact>
                  {({ match: nested }) => <span>lineage {nested.params.id}</span>}
                </Route>
              </Switch>
            )}
          />
          <Route>missing</Route>
        </Switch>
      </MemoryRouter>,
    );
    expect(screen.getByText('lineage 123')).toBeInTheDocument();
    expect(screen.queryByText('list')).not.toBeInTheDocument();
    expect(screen.queryByText('missing')).not.toBeInTheDocument();
  });

  it('preserves the pathname for query-only navigation and history semantics', () => {
    const history = createMemoryHistory({ initialEntries: ['/runs/123?old=1'] });
    render(
      <Router history={history}>
        <Route
          render={({ history: navigation, location }) => (
            <>
              <span>
                {location.pathname}
                {location.search}
              </span>
              <button
                onClick={() => navigation.push({ search: '?page=2', state: { source: 'query' } })}
              >
                push
              </button>
              <button onClick={() => navigation.replace({ search: '?page=3' })}>replace</button>
            </>
          )}
        />
      </Router>,
    );
    fireEvent.click(screen.getByText('push'));
    expect(history.location.pathname).toBe('/runs/123');
    expect(history.location.state).toEqual({ source: 'query' });
    expect(history.length).toBe(2);
    fireEvent.click(screen.getByText('replace'));
    expect(screen.getByText('/runs/123?page=3')).toBeInTheDocument();
    expect(history.length).toBe(2);
    act(() => history.goBack());
    expect(screen.getByText('/runs/123?old=1')).toBeInTheDocument();
  });

  it('replaces redirects and matches array paths', () => {
    const history = createMemoryHistory({ initialEntries: ['/'] });
    render(
      <Router history={history}>
        <Switch>
          <Route path='/' exact>
            <Redirect to='/runs/123' />
          </Route>
          <Route path={['/runs/:id', '/artifacts/:id']}>
            {({ match }) => <span>resource {match.params.id}</span>}
          </Route>
        </Switch>
      </Router>,
    );
    expect(screen.getByText('resource 123')).toBeInTheDocument();
    expect(history.length).toBe(1);
  });
});
