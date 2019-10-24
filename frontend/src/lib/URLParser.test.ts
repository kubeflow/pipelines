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

import { createHashHistory, createLocation } from 'history';
import { URLParser } from './URLParser';

const history = createHashHistory();
const location = createLocation('/');

describe('URLParser', () => {
  const routerProps = { location, history } as any;

  it('gets query string param by name', () => {
    location.pathname = '/test';
    location.search = '?searchkey=searchvalue';

    const params = new URLParser(routerProps).get('searchkey' as any);
    expect(params).toEqual('searchvalue');
  });

  it('returns empty string when getting a nonexistent query string param', () => {
    expect(new URLParser(routerProps).get('nonexistent' as any)).toBeNull();
  });

  it('gets query string param by name where multiple keys defined', () => {
    location.search = '?searchkey=searchvalue&k2=v2';

    const params = new URLParser(routerProps).get('searchkey' as any);
    expect(params).toEqual('searchvalue');
  });

  it('sets query string param by name', () => {
    location.search = '?searchkey=searchvalue';

    new URLParser(routerProps).set('searchkey' as any, 'newvalue');
    expect(history.location.search).toEqual('?searchkey=newvalue');
  });

  it('creates new query string param when using set if not exists', () => {
    location.search = '?searchkey=searchvalue';

    new URLParser(routerProps).set('searchkey2' as any, 'searchvalue2');
    expect(history.location.search).toEqual('?searchkey=searchvalue&searchkey2=searchvalue2');
  });

  it('removes the query string param when setting it to empty value', () => {
    location.search = '?searchkey=searchvalue';

    new URLParser(routerProps).set('searchkey' as any, '');
    expect(history.location.search).toEqual('');
  });

  it('does not create new state when setting query param in-place', () => {
    location.search = '?searchkey=searchvalue';

    const historyLength = history.length;
    new URLParser(routerProps).set('searchkey' as any, 'newvalue');
    expect(history.length).toEqual(historyLength);
  });

  it('does not touch other query string params when setting one', () => {
    location.search = '?searchkey=searchvalue&k2=v2';

    new URLParser(routerProps).set('searchkey' as any, 'newvalue');
    expect(history.location.search).toEqual('?searchkey=newvalue&k2=v2');
  });

  it('does not touch other query string params when clearing one', () => {
    location.search = '?searchkey=searchvalue&k2=v2';

    new URLParser(routerProps).clear('searchkey' as any);
    expect(history.location.search).toEqual('?k2=v2');
  });

  it('sets query string param when using push', () => {
    location.search = '?searchkey=searchvalue&k2=v2';

    new URLParser(routerProps).push('searchkey' as any, 'newvalue');
    expect(history.location.search).toEqual('?searchkey=newvalue&k2=v2');
  });

  it('removes query string param when push an empty value', () => {
    location.search = '?searchkey=searchvalue&k2=v2';

    new URLParser(routerProps).push('searchkey' as any, '');
    expect(history.location.search).toEqual('?k2=v2');
  });

  it('creates a new history entry when using push', () => {
    location.search = '?searchkey=searchvalue&k2=v2';

    const historyLength = history.length;
    new URLParser(routerProps).push('searchkey' as any, 'newvalue');
    expect(history.location.search).toEqual('?searchkey=newvalue&k2=v2');
    expect(history.length).toEqual(historyLength + 1);
  });

  it('can build a search string', () => {
    expect(new URLParser(routerProps).build({ key1: 'value1', key2: 'value2' })).toEqual(
      '?key1=value1&key2=value2',
    );
  });

  it('returns empty query string when given an empty object or no object', () => {
    expect(new URLParser(routerProps).build()).toEqual('?');
    expect(new URLParser(routerProps).build({})).toEqual('?');
  });
});
