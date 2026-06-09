// Copyright 2026 The Kubeflow Authors
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
import { URL } from 'url';

const TENSORBOARD_PROXY_PREFIX = '/apps/tensorboard/proxy/';

/**
 * Extracts the scoped TensorBoard proxy prefix from a referer so mock absolute
 * subrequests follow the same path rewriting behavior as the real server.
 */
function getTensorboardProxyBasePath(referer = ''): string | undefined {
  try {
    const pathname = new URL(referer).pathname;
    const prefixIndex = pathname.indexOf(TENSORBOARD_PROXY_PREFIX);
    if (prefixIndex < 0) {
      return undefined;
    }
    const remainder = pathname.slice(prefixIndex + TENSORBOARD_PROXY_PREFIX.length);
    const token = remainder.split('/').filter(Boolean)[0];
    if (!token) {
      return undefined;
    }
    return `${pathname.slice(0, prefixIndex)}${TENSORBOARD_PROXY_PREFIX}${token}`;
  } catch {
    return undefined;
  }
}

/**
 * Registers the mock TensorBoard proxy routes used by the frontend mock backend.
 */
// tslint:disable-next-line:no-default-export
export default (app: express.Application) => {
  app.use((req, _, next) => {
    const proxyBasePath = getTensorboardProxyBasePath(req.headers.referer as string);
    if (proxyBasePath && req.url.indexOf(TENSORBOARD_PROXY_PREFIX) < 0) {
      req.url = `${proxyBasePath}${req.url}`;
    }
    next();
  });

  app.all(`${TENSORBOARD_PROXY_PREFIX}*`, (req, res) => {
    if (req.method === 'HEAD') {
      res.status(200).end();
      return;
    }
    res.status(200).send('Mock TensorBoard proxy');
  });
};
