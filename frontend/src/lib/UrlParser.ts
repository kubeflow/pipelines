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

export enum QUERY_PARAMS {
  cloneFromJob = 'cloneFromJob',
  cloneFromRun = 'cloneFromRun',
  pipelineId = 'pipelineId',
  runlist = 'runlist',
  view = 'view',
}

export function from(part: 'search' | 'hash') {
  return new Parser(part);
}

class Parser {

  private _type: 'search' | 'hash';
  private _paramMap: URLSearchParams | Map<string, string>;

  constructor(type: 'search' | 'hash') {
    this._type = type;
    this._paramMap = type === 'search' ?
      new URLSearchParams(location.search) :
      this._parseHash();
  }

  public get(key: QUERY_PARAMS): string {
    return this._paramMap.get(key) || '';
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
    return (this._type === 'search' ? '?' : '#') +
      Object.keys(obj).map(k => k + '=' + obj[k]).join('&');
  }

  private _parseHash() {
    const str = location.hash.substr(1); // Remove the initial '#'
    const result = new Map<string, string>();
    str.split('&').filter(p => !!p).forEach(param => {
      const pieces = param.split('=');
      result.set(pieces[0], pieces[1]);
    });
    return result;
  }

  private _update(replace = true): void {
    const newSearch = this._type === 'search' ?
      '?' + this._paramMap.toString() : location.search;
    const newHash = this._type === 'hash' ?
      '#' + Array.from((this._paramMap as Map<string, string>)
        .entries())
        .map(e => e[0] + '=' + e[1])
        .join('&') :
      location.hash;
    const newUrl =
      `${location.protocol}//${location.host}${location.pathname}${newSearch}${newHash}`;
    if (replace) {
      history.replaceState({ path: newUrl }, '', newUrl);
    } else {
      history.pushState({ path: newUrl }, '', newUrl);
    }
  }
}
