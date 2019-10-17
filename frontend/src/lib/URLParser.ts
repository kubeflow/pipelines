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

import { QUERY_PARAMS } from '../components/Router';
import { RouteComponentProps } from 'react-router';

export class URLParser {
  private _paramMap: URLSearchParams;
  private _routeProps: RouteComponentProps;

  constructor(routeProps: RouteComponentProps) {
    this._routeProps = routeProps;
    this._paramMap = new URLSearchParams(routeProps.location.search);
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
    if (replace) {
      this._routeProps.history.replace({ search: this._paramMap.toString() });
    } else {
      this._routeProps.history.push({ search: this._paramMap.toString() });
    }
  }
}
