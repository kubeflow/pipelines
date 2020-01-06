// Copyright 2019 Google LLC
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
import * as path from 'path';
import * as express from 'express';
import { Application, static as StaticHandler } from 'express';
import * as proxy from 'http-proxy-middleware';

import { UIConfigs } from './configs';
import { getAddress } from './utils';
import { getBuildMetadata, getHealthzEndpoint, getHealthzHandler } from './handlers/healthz';
import { getArtifactsHandler } from './handlers/artifacts';
import {
  getCreateTensorboardHandler,
  getTensorboardHandler,
  deleteTensorboardHandler,
} from './handlers/tensorboard';
import { getPodLogsHandler } from './handlers/pod-logs';
import { clusterNameHandler, projectIdHandler } from './handlers/gke-metadata';
import { getAllowCustomVisualizationsHandler } from './handlers/vis';
import { getIndexHTMLHandler } from './handlers/index-html';

import proxyMiddleware from './proxy-middleware';
import { Server } from 'http';

function getRegisterHandler(app: Application, basePath: string) {
  return (
    func: (name: string, handler: express.Handler) => express.Application,
    route: string,
    handler: express.Handler,
  ) => {
    func.call(app, route, handler);
    return func.call(app, `${basePath}${route}`, handler);
  };
}

/**
 * UIServer wraps around a express application to:
 * - proxy requests to ml-pipeline api server
 * - retrieve artifacts from the various backend
 * - create and retrieve new viewer instances
 * - serve static front-end resources (i.e. react app)
 */
export class UIServer {
  app: Application;
  httpServer?: Server;

  constructor(public readonly options: UIConfigs) {
    this.app = createUIServer(options);
  }

  /**
   * Starts the http server.
   * @param port optionally overwrite the provided port to listen to.
   */
  start(port?: number | string) {
    if (this.httpServer) {
      throw new Error('UIServer already started.');
    }
    port = port || this.options.server.port;
    this.httpServer = this.app.listen(port, () => {
      console.log('Server listening at http://localhost:' + port);
    });
    return this.httpServer;
  }

  /**
   * Stops the http server.
   */
  close() {
    if (this.httpServer) {
      this.httpServer.close();
    }
    this.httpServer = undefined;
    return this;
  }
}

function createUIServer(options: UIConfigs) {
  const currDir = path.resolve(__dirname);
  const basePath = options.server.basePath;
  const apiVersionPrefix = options.server.apiVersionPrefix;
  const apiServerAddress = getAddress(options.pipeline);
  const envoyServiceAddress = getAddress(options.metadata.envoyService);

  const app: Application = express();
  const registerHandler = getRegisterHandler(app, basePath);

  /** log to stdout */
  app.use((req, _, next) => {
    console.info(req.method + ' ' + req.originalUrl);
    next();
  });

  /** Healthz */
  registerHandler(
    app.get,
    `/${apiVersionPrefix}/healthz`,
    getHealthzHandler({
      healthzEndpoint: getHealthzEndpoint(apiServerAddress, apiVersionPrefix),
      healthzStats: getBuildMetadata(currDir),
    }),
  );

  /** Artifact */
  registerHandler(app.get, '/artifacts/get', getArtifactsHandler(options.artifacts));

  /** Tensorboard viewer */
  registerHandler(app.get, '/apps/tensorboard', getTensorboardHandler);
  registerHandler(app.delete, '/apps/tensorboard', deleteTensorboardHandler);
  registerHandler(
    app.post,
    '/apps/tensorboard',
    getCreateTensorboardHandler(options.viewer.tensorboard.podTemplateSpec),
  );

  /** Pod logs */
  registerHandler(app.get, '/k8s/pod/logs', getPodLogsHandler(options.argo, options.artifacts));

  /** Cluster metadata (GKE only) */
  registerHandler(app.get, '/system/cluster-name', clusterNameHandler);
  registerHandler(app.get, '/system/project-id', projectIdHandler);

  /** Visualization */
  registerHandler(
    app.get,
    '/visualizations/allowed',
    getAllowCustomVisualizationsHandler(options.visualizations.allowCustomVisualizations),
  );

  /** Proxy metadata requests to the Envoy instance which will handle routing to the metadata gRPC server */
  app.all(
    '/ml_metadata.*',
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Metadata proxied request: ', (proxyReq as any).path);
      },
      target: envoyServiceAddress,
    }),
  );

  // Order matters here, since both handlers can match any proxied request with a referer,
  // and we prioritize the basepath-friendly handler
  proxyMiddleware(app, `${basePath}/${apiVersionPrefix}`);
  proxyMiddleware(app, `/${apiVersionPrefix}`);

  /** Proxy to ml-pipeline api server */
  app.all(
    `/${apiVersionPrefix}/*`,
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied request: ', (proxyReq as any).path);
      },
      target: apiServerAddress,
    }),
  );
  app.all(
    `${basePath}/${apiVersionPrefix}/*`,
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied request: ', (proxyReq as any).path);
      },
      pathRewrite: pathStr =>
        pathStr.startsWith(basePath) ? pathStr.substr(basePath.length, pathStr.length) : pathStr,
      target: apiServerAddress,
    }),
  );

  /**
   * Modify index.html.
   * These pathes can be matched by static handler. Putting them before it to
   * override behavior for index html.
   */
  const indexHtmlHandler = getIndexHTMLHandler(options.server);
  registerHandler(app.get, '/', indexHtmlHandler);
  registerHandler(app.get, '/index.html', indexHtmlHandler);

  /** Static resource (i.e. react app) */
  app.use(basePath, StaticHandler(options.server.staticDir));
  app.use(StaticHandler(options.server.staticDir));

  /** Fallback to index.html */
  app.get('*', indexHtmlHandler);

  return app;
}
