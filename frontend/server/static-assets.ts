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

import { stat } from 'node:fs/promises';
import path from 'node:path';
import express from 'express';

async function isFile(filename: string): Promise<boolean> {
  try {
    return (await stat(filename)).isFile();
  } catch (error) {
    // Match send's lookup fallthrough; permission and I/O failures remain errors.
    if (
      ['ENOENT', 'ENOTDIR', 'ENAMETOOLONG'].includes((error as NodeJS.ErrnoException).code || '')
    ) {
      return false;
    }
    throw error;
  }
}

/** Serve build-time gzip representations without changing the public asset URLs. */
export function createPrecompressedStaticApp(
  staticDir: string,
  options: { cacheControl?: string } = {},
): express.Application {
  const app = express();
  app.disable('x-powered-by');
  app.use(async (req, res, next) => {
    // Vite emits flat public assets. Never handle HTML, API/proxy responses,
    // user artifacts, encoded path separators, or nested/traversal paths.
    if (
      !['GET', 'HEAD'].includes(req.method) ||
      !/^\/static\/[A-Za-z0-9_.-]+\.(js|css)$/.test(req.path)
    ) {
      next();
      return;
    }
    try {
      const original = path.resolve(staticDir, `.${req.path}`);
      if (!(await isFile(original))) {
        next();
        return;
      }
      const gzip = `${original}.gz`;
      const available = (await isFile(gzip)) ? ['gzip', 'identity'] : ['identity'];
      const encoding = req.acceptsEncodings(...available);
      res.vary('Accept-Encoding');
      if (!encoding) {
        res.sendStatus(406);
        return;
      }
      const compressed = encoding === 'gzip';
      res.type(path.extname(original));
      if (compressed) res.setHeader('Content-Encoding', 'gzip');
      res.sendFile(
        compressed ? gzip : original,
        {
          // Ignore Range for compressed representations; identity retains the
          // existing static handler's range behavior. sendFile owns validators.
          acceptRanges: !compressed,
          ...(options.cacheControl ? { headers: { 'Cache-Control': options.cacheControl } } : {}),
        },
        (error) => {
          if (error) {
            if (!res.headersSent) res.removeHeader('Content-Encoding');
            next(error);
          }
        },
      );
    } catch (error) {
      next(error);
    }
  });
  return app;
}
