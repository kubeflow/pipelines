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

import * as UrlParser from './UrlParser';

(global as any).window = {
  location: {
    hash: '#myhash',
    search: '?mysearch',
  },
};

describe('UrlParser', () => {
  describe('Search', () => {
    it('gets query string param by name', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue');

      const params = UrlParser.from('search').get('searchkey' as any);
      expect(params).toEqual('searchvalue');
    });

    it('returns empty string when getting a nonexistent query string param', () => {
      expect(UrlParser.from('search').get('nonexistent' as any)).toEqual('');
    });

    it('gets query string param by name where multiple keys defined', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue&k2=v2');

      const params = UrlParser.from('search').get('searchkey' as any);
      expect(params).toEqual('searchvalue');
    });

    it('gets query string param by name where hash key defined with the same name', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue#searchkey=othervalue');

      const params = UrlParser.from('search').get('searchkey' as any);
      expect(params).toEqual('searchvalue');
    });

    it('sets query string param by name', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue');

      UrlParser.from('search').set('searchkey' as any, 'newvalue');
      expect(location.search).toEqual('?searchkey=newvalue');
    });

    it('creates new query string param when using set if not exists', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue');

      UrlParser.from('search').set('searchkey2' as any, 'searchvalue2');
      expect(location.search).toEqual('?searchkey=searchvalue&searchkey2=searchvalue2');
    });

    it('removes the query string param when setting it to empty value', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue');

      UrlParser.from('search').set('searchkey' as any, '');
      expect(location.search).toEqual('');
    });

    it('does not create new state when setting query param in-place', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue');

      const historyLength = window.history.length;
      UrlParser.from('search').set('searchkey' as any, 'newvalue');
      expect(history.length).toEqual(historyLength);
    });

    it('does not touch other query string params when setting one', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue&k2=v2');

      UrlParser.from('search').set('searchkey' as any, 'newvalue');
      expect(location.search).toEqual('?searchkey=newvalue&k2=v2');
    });

    it('does not touch other query string params when clearing one', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue&k2=v2');

      UrlParser.from('search').clear('searchkey' as any);
      expect(location.search).toEqual('?k2=v2');
    });

    it('sets query string param when using push', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue&k2=v2');

      UrlParser.from('search').push('searchkey' as any, 'newvalue');
      expect(location.search).toEqual('?searchkey=newvalue&k2=v2');
    });

    it('removes query string param when push an empty value', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue&k2=v2');

      UrlParser.from('search').push('searchkey' as any, '');
      expect(location.search).toEqual('?k2=v2');
    });

    it('creates a new history entry when using push', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue&k2=v2');

      const historyLength = history.length;
      UrlParser.from('search').push('searchkey' as any, 'newvalue');
      expect(location.search).toEqual('?searchkey=newvalue&k2=v2');
      expect(history.length).toEqual(historyLength + 1);
    });

    it('does not modify hash when setting query string params', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue#hashkey1=hashvalue1');

      UrlParser.from('search').set('searchkey' as any, 'newvalue');
      expect(location.hash).toEqual('#hashkey1=hashvalue1');
    });
  });

  describe('Hash', () => {
    it('gets hash param by name', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue');

      const params = UrlParser.from('hash').get('searchkey' as any);
      expect(params).toEqual('searchvalue');
    });

    it('returns empty string when getting a nonexistent hash param', () => {
      expect(UrlParser.from('hash').get('nonexistent' as any)).toEqual('');
    });

    it('gets hash param by name where multiple keys defined', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue&k2=v2');

      const params = UrlParser.from('hash').get('searchkey' as any);
      expect(params).toEqual('searchvalue');
    });

    it('gets hash param by name where a search key defined with the same name', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue#searchkey=othervalue');

      const params = UrlParser.from('hash').get('searchkey' as any);
      expect(params).toEqual('othervalue');
    });

    it('sets hash param by name', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue');

      UrlParser.from('hash').set('searchkey' as any, 'newvalue');
      expect(location.hash).toEqual('#searchkey=newvalue');
    });

    it('creates new hash param when using set if not exists', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue');

      UrlParser.from('hash').set('searchkey2' as any, 'searchvalue2');
      expect(location.hash).toEqual('#searchkey=searchvalue&searchkey2=searchvalue2');
    });

    it('removes the hash param when setting it to empty value', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue');

      UrlParser.from('hash').set('searchkey' as any, '');
      expect(location.search).toEqual('');
    });

    it('does not create new state when setting hash param in-place', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue');

      const historyLength = window.history.length;
      UrlParser.from('hash').set('searchkey' as any, 'newvalue');
      expect(history.length).toEqual(historyLength);
    });

    it('does not touch other hash string params when setting one', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue&k2=v2');

      UrlParser.from('hash').set('searchkey' as any, 'newvalue');
      expect(location.hash).toEqual('#searchkey=newvalue&k2=v2');
    });

    it('does not touch other hash string params when clearing one', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue&k2=v2');

      UrlParser.from('hash').clear('searchkey' as any);
      expect(location.hash).toEqual('#k2=v2');
    });

    it('sets hash string param when using push', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue&k2=v2');

      UrlParser.from('hash').push('searchkey' as any, 'newvalue');
      expect(location.hash).toEqual('#searchkey=newvalue&k2=v2');
    });

    it('removes query string param when push an empty value', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue&k2=v2');

      UrlParser.from('hash').push('searchkey' as any, '');
      expect(location.hash).toEqual('#k2=v2');
    });

    it('creates a new history entry when using push', () => {
      window.history.replaceState({}, '', '/test.html#searchkey=searchvalue&k2=v2');

      const historyLength = history.length;
      UrlParser.from('hash').push('searchkey' as any, 'newvalue');
      expect(location.hash).toEqual('#searchkey=newvalue&k2=v2');
      expect(history.length).toEqual(historyLength + 1);
    });

    it('does not modify search when setting hash params', () => {
      window.history.replaceState({}, '', '/test.html?searchkey=searchvalue#hashkey1=hashvalue1');

      UrlParser.from('hash').set('hashkey1' as any, 'newvalue');
      expect(location.search).toEqual('?searchkey=searchvalue');
    });
  });

  describe('Build', () => {
    it('can build a search string', () => {
      expect(UrlParser.from('search').build({key1: 'value1', key2: 'value2'}))
        .toEqual('?key1=value1&key2=value2');
    });

    it('can build a hash string', () => {
      expect(UrlParser.from('hash').build({key1: 'value1', key2: 'value2'}))
        .toEqual('#key1=value1&key2=value2');
    });

    it('returns empty query string when given an empty object or no object', () => {
      expect(UrlParser.from('search').build()).toEqual('?');
      expect(UrlParser.from('search').build({})).toEqual('?');
    });

    it('returns empty hash string when given an empty object or no object', () => {
      expect(UrlParser.from('hash').build()).toEqual('#');
      expect(UrlParser.from('hash').build({})).toEqual('#');
    });
  });
});
