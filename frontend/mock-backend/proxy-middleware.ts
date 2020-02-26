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

import * as express from 'express';
import proxy from 'http-proxy-middleware';
import { URL, URLSearchParams } from 'url';

export function _extractUrlFromReferer(proxyPrefix: string, referer = ''): string {
  const index = referer.indexOf(proxyPrefix);
  return index > -1 ? referer.substr(index + proxyPrefix.length) : '';
}

export function _trimProxyPrefix(proxyPrefix: string, path: string): string {
  return path.indexOf(proxyPrefix) === 0 ? (path = path.substr(proxyPrefix.length)) : path;
}

export function _routePathWithReferer(proxyPrefix: string, path: string, referer = ''): string {
  // If a referer header is included, extract the referer URL, otherwise
  // just trim out the /_proxy/ prefix. Use the origin of the resulting URL.
  const proxiedUrlInReferer = _extractUrlFromReferer(proxyPrefix, referer);
  let decodedPath = decodeURIComponent(proxiedUrlInReferer || _trimProxyPrefix(proxyPrefix, path));
  if (!decodedPath.startsWith('http://') && !decodedPath.startsWith('https://')) {
    decodedPath = 'http://' + decodedPath;
  }
  return new URL(decodedPath).origin;
}

export function _rewritePath(proxyPrefix: string, path: string, query: string): string {
  // Trim the proxy prefix if exists. It won't exist for any requests made
  // to absolute paths by the proxied resource.
  const querystring = new URLSearchParams(query).toString();
  const decodedPath = decodeURIComponent(path);
  return _trimProxyPrefix(proxyPrefix, decodedPath) + (querystring && '?' + querystring);
}

export default (app: express.Application, apisPrefix: string) => {
  const proxyPrefix = apisPrefix + '/_proxy/';

  app.use((req, _, next) => {
    // For any request that has a proxy referer header but no proxy prefix,
    // prepend the proxy prefix to it and redirect.
    if (req.headers.referer) {
      const refererUrl = _extractUrlFromReferer(proxyPrefix, req.headers.referer as string);
      if (refererUrl && req.url.indexOf(proxyPrefix) !== 0) {
        let proxiedUrl = decodeURIComponent(
          _extractUrlFromReferer(proxyPrefix, req.headers.referer as string),
        );
        if (!proxiedUrl.startsWith('http://') && !proxiedUrl.startsWith('https://')) {
          proxiedUrl = 'http://' + proxiedUrl;
        }
        const proxiedOrigin = new URL(proxiedUrl).origin;
        req.url = proxyPrefix + encodeURIComponent(proxiedOrigin + req.url);
      }
    }
    next();
  });

  app.all(
    proxyPrefix + '*',
    proxy({
      changeOrigin: true,
      logLevel: 'debug',
      target: 'http://127.0.0.1',

      router: (req: any) => {
        return _routePathWithReferer(proxyPrefix, req.path, req.headers.referer as string);
      },

      pathRewrite: (_: any, req: any) => {
        return _rewritePath(proxyPrefix, req.path, req.query);
      },
    }),
  );
};
