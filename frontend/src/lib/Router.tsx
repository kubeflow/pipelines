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

import * as React from 'react';
import type { History, LocationDescriptor } from 'history';
import {
  Location,
  matchPath,
  Navigate,
  Router as NativeRouter,
  useLocation,
  useNavigate,
} from 'react-router';

// Class pages still receive route props; matching and navigation use the current
// router. Keeping this boundary avoids coupling page/data migrations to routing.
export interface match<Params = Record<string, string | undefined>> {
  params: Params;
  path: string;
  url: string;
  isExact: boolean;
}

export interface RouteComponentProps<Params = Record<string, string | undefined>> {
  history: Pick<History, 'push' | 'replace' | 'length' | 'goBack'>;
  location: Location;
  match: match<Params>;
}

interface RouteProps {
  path?: string | string[];
  exact?: boolean;
  render?: (props: RouteComponentProps) => React.ReactNode;
  children?: React.ReactNode | ((props: RouteComponentProps) => React.ReactNode);
}

const HistoryContext = React.createContext<History | null>(null);

const MatchContext = React.createContext<match | null>(null);

function findMatch(path: RouteProps['path'], exact: boolean, pathname: string): match | null {
  for (const pattern of Array.isArray(path) ? path : [path || '/']) {
    const found = matchPath({ path: pattern, end: exact }, pathname);
    if (found) {
      return {
        params: found.params,
        path: pattern,
        url: found.pathname,
        isExact: found.pathname.replace(/\/$/, '') === pathname.replace(/\/$/, ''),
      };
    }
  }
  return null;
}

export function Route({ path, exact = false, render, children }: RouteProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const inherited = React.useContext(MatchContext);
  const externalHistory = React.useContext(HistoryContext);
  const matched = path
    ? findMatch(path, exact, location.pathname)
    : inherited || findMatch('/', false, location.pathname);
  const history = React.useMemo(() => {
    const move = (to: LocationDescriptor, state: unknown, replace: boolean) => {
      const target =
        typeof to === 'string'
          ? to
          : {
              pathname: to.pathname || location.pathname,
              search: to.search,
              hash: to.hash,
            };
      void navigate(target, {
        replace,
        state: typeof to === 'string' ? state : (to.state ?? state),
      });
    };
    return {
      get length() {
        return externalHistory?.length ?? window.history.length;
      },
      goBack: () => {
        void navigate(-1);
      },
      push: (to: LocationDescriptor, state?: unknown) => move(to, state, false),
      replace: (to: LocationDescriptor, state?: unknown) => move(to, state, true),
    };
  }, [navigate, location.pathname, externalHistory]);
  if (!matched) return null;
  const props = { history, location, match: matched };
  return (
    <MatchContext.Provider value={matched}>
      {render ? render(props) : typeof children === 'function' ? children(props) : children}
    </MatchContext.Provider>
  );
}

// Preserve ordered matching for the application's overlapping detail/list URLs.
export function Switch({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const route = React.Children.toArray(children).find((child) => {
    if (!React.isValidElement<RouteProps>(child)) return false;
    return !child.props.path || findMatch(child.props.path, !!child.props.exact, location.pathname);
  });
  return <>{route}</>;
}

export function Redirect({ to, push = false }: { to: LocationDescriptor; push?: boolean }) {
  const { state, ...target } = typeof to === 'string' ? { pathname: to, state: undefined } : to;
  return <Navigate to={typeof to === 'string' ? to : target} state={state} replace={!push} />;
}

// Test utilities supply history v4 objects. Subscribe to that external history
// while rendering the same router/context used by production HashRouter.
export function Router({ history, children }: { history: History; children: React.ReactNode }) {
  const subscribe = React.useCallback(
    (callback: () => void) => history.listen(callback),
    [history],
  );
  const getSnapshot = React.useCallback(() => history.location, [history]);
  const location = React.useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
  return (
    <HistoryContext.Provider value={history}>
      <NativeRouter location={location} navigator={history}>
        {children}
      </NativeRouter>
    </HistoryContext.Provider>
  );
}
