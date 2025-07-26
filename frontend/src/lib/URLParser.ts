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

import { QUERY_PARAMS } from '../components/Router';
import { useLocation, useNavigate } from 'react-router-dom';
import { useMemo } from 'react';

// Hook-based URLParser for React Router v6
export function useURLParser() {
  const location = useLocation();
  const navigate = useNavigate();

  const paramMap = useMemo(() => {
    return new URLSearchParams(location.search);
  }, [location.search]);

  const get = (key: QUERY_PARAMS): string | null => {
    return paramMap.get(key);
  };

  const set = (key: QUERY_PARAMS, value: string): void => {
    const newParams = new URLSearchParams(location.search);
    if (value) {
      newParams.set(key, value);
    } else {
      newParams.delete(key);
    }
    navigate({ search: newParams.toString() }, { replace: true });
  };

  const push = (key: QUERY_PARAMS, value: string): void => {
    const newParams = new URLSearchParams(location.search);
    if (value) {
      newParams.set(key, value);
    } else {
      newParams.delete(key);
    }
    navigate({ search: newParams.toString() }, { replace: false });
  };

  const clear = (key: QUERY_PARAMS): void => {
    const newParams = new URLSearchParams(location.search);
    newParams.delete(key);
    navigate({ search: newParams.toString() }, { replace: true });
  };

  const build = (searchTerms?: { [param: string]: string }): string => {
    const obj = searchTerms || {};
    return (
      '?' +
      Object.keys(obj)
        .map(k => k + '=' + obj[k])
        .join('&')
    );
  };

  return { get, set, push, clear, build };
}

// Legacy class-based URLParser for backwards compatibility
// TODO: Remove this once all components are migrated to hooks
export class URLParser {
  private _paramMap: URLSearchParams;
  private _location: any;
  private _navigate: any;

  constructor(routeProps: any) {
    // For backwards compatibility, accept either RouteComponentProps or location/navigate
    if (routeProps.location && routeProps.history) {
      // Legacy RouteComponentProps
      this._location = routeProps.location;
      this._navigate = routeProps.history;
      this._paramMap = new URLSearchParams(routeProps.location.search);
    } else {
      // New pattern with location and navigate passed separately
      this._location = routeProps.location;
      this._navigate = routeProps.navigate;
      this._paramMap = new URLSearchParams(routeProps.location.search);
    }
  }

  public get(key: QUERY_PARAMS): string | null {
    return this._paramMap.get(key);
  }

  public set(key: QUERY_PARAMS, value: string): void {
    if (value) {
      this._paramMap.set(key, value);
    } else {
      this._paramMap.delete(key);
    }
    this._update();
  }

  public push(key: QUERY_PARAMS, value: string): void {
    if (value) {
      this._paramMap.set(key, value);
    } else {
      this._paramMap.delete(key);
    }
    this._update(false);
  }

  public clear(key: QUERY_PARAMS): void {
    this._paramMap.delete(key);
    this._update();
  }

  // TODO: create interface for this param.
  public build(searchTerms?: { [param: string]: string }): string {
    const obj = searchTerms || {};
    return (
      '?' +
      Object.keys(obj)
        .map(k => k + '=' + obj[k])
        .join('&')
    );
  }

  private _update(replace = true): void {
    if (this._navigate.replace && this._navigate.push) {
      // Legacy history object
      if (replace) {
        this._navigate.replace({ search: this._paramMap.toString() });
      } else {
        this._navigate.push({ search: this._paramMap.toString() });
      }
    } else {
      // New navigate function
      this._navigate({ search: this._paramMap.toString() }, { replace });
    }
  }
}
