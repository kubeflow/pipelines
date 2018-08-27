// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { assert } from 'chai';
import 'mocha';
import {
  _extractUrlFromReferer,
  _rewritePath,
  _routePathWithReferer,
  _trimProxyPrefix,
} from '../../server/proxy-middleware';

const proxyPrefix = 'apis/vtest/_proxy/';

describe('proxy middleware', () => {

  it('extracts nothing when there is no proxied url in referer', () => {
    const referer = 'http://path/with/no/proxy';
    assert.equal(_extractUrlFromReferer(proxyPrefix, referer), '');
  });

  it('extracts nothing when there is no referer header', () => {
    assert.equal(_extractUrlFromReferer(proxyPrefix, ''), '');
  });

  it('extracts simple referer urls', () => {
    const referer = 'http://path/with/proxy/apis/vtest/_proxy/someurl';
    assert.equal(_extractUrlFromReferer(proxyPrefix, referer), 'someurl');
  });

  it('extracts complex referer urls', () => {
    const referer = 'http://path/with/proxy/apis/vtest/_proxy/http://someurl.com';
    assert.equal(_extractUrlFromReferer(proxyPrefix, referer), 'http://someurl.com');
  });

  it('extracts encoded referer urls', () => {
    const encodedUrl = encodeURIComponent('http://someurl.com');
    const referer = 'http://path/with/proxy/apis/vtest/_proxy/' + encodedUrl;
    assert.equal(_extractUrlFromReferer(proxyPrefix, referer), encodedUrl);
  });

  it('extracts referer urls with querystring', () => {
    const url = 'http://someurl.com?q1=one&q2=two';
    assert.equal(_extractUrlFromReferer(proxyPrefix, url), '');
  });

  it('extracts referer urls with querystring and hash', () => {
    const url = 'http://someurl.com?q1=one&q2=two#hash';
    assert.equal(_extractUrlFromReferer(proxyPrefix, url), '');
  });

  it('trims proxy prefix if exists', () => {
    const url = 'http://someurl.com';
    assert.equal(_trimProxyPrefix(proxyPrefix, 'apis/vtest/_proxy/' + url), url);
  });

  it('does not trim proxy prefix if not exists', () => {
    const url = 'http://someurl.com';
    assert.equal(_trimProxyPrefix(proxyPrefix, url), url);
  });

  it('routes to decoded referer if included in request', () => {
    const path = 'path1/path2';
    const proxiedUrl = 'http://proxiedurl.com';
    const referer = 'http://someurl.com/' + proxyPrefix + encodeURIComponent(proxiedUrl);
    assert.equal(_routePathWithReferer(proxyPrefix, path, referer), proxiedUrl);
  });

  it('auto-prepends http:// if proxied URL does not have protocol', () => {
    const path = 'path1/path2';
    const proxiedUrl = 'proxiedurl.com';
    const referer = 'http://someurl.com/' + proxyPrefix + encodeURIComponent(proxiedUrl);
    assert.equal(_routePathWithReferer(proxyPrefix, path, referer), 'http://' + proxiedUrl);
  });

  it('routes to origin of url if no referer included', () => {
    const path = proxyPrefix + 'http://someurl.com/path1/path2';
    assert.equal(_routePathWithReferer(proxyPrefix, path), 'http://someurl.com');
  });

  it('routes to origin of absolute url if no referer included', () => {
    const path = 'http://someurl.com/path1/path2';
    assert.equal(_routePathWithReferer(proxyPrefix, path), 'http://someurl.com');
  });

  it('rewrites path while keeping querystring', () => {
    const path = '/path1/path2';
    const query = 'q1=one&q2=two';
    assert.equal(_rewritePath(proxyPrefix, path, query), path + '?' + query);
  });

  it('rewrites path with no querystring', () => {
    const path = '/path1/path2';
    const query = '';
    assert.equal(_rewritePath(proxyPrefix, path, query), path);
  });
});
