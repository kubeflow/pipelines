// Copyright 2019 The Kubeflow Authors
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
import path from 'path';
import express from 'express';
import { Application, static as StaticHandler } from 'express';
import proxy from 'http-proxy-middleware';

import { UIConfigs } from './configs';
import { getAddress } from './utils';
import { getBuildMetadata, getHealthzEndpoint, getHealthzHandler } from './handlers/healthz';
import {
  getArtifactsHandler,
  getArtifactsProxyHandler,
  getArtifactServiceGetter,
} from './handlers/artifacts';
import { getTensorboardHandlers } from './handlers/tensorboard';
import { getAuthorizeFn } from './helpers/auth';
import { getPodLogsHandler } from './handlers/pod-logs';
import { podInfoHandler, podEventsHandler } from './handlers/pod-info';
import { getClusterNameHandler, getProjectIdHandler } from './handlers/gke-metadata';
import { getAllowCustomVisualizationsHandler } from './handlers/vis';
import { getIndexHTMLHandler } from './handlers/index-html';

import proxyMiddleware from './proxy-middleware';
import { Server } from 'http';
import { HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS } from './consts';

function getRegisterHandler(app: Application, basePath: string) {
  return (
    func: (name: string | string[], handler: express.Handler) => express.Application,
    route: string | string[],
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
  const apiVersion1Prefix = options.server.apiVersion1Prefix;
  const apiVersion2Prefix = options.server.apiVersion2Prefix;
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
    `/${apiVersion1Prefix}/healthz`,
    getHealthzHandler({
      healthzEndpoint: getHealthzEndpoint(apiServerAddress, apiVersion1Prefix),
      healthzStats: getBuildMetadata(currDir),
    }),
  );

  registerHandler(
    app.get,
    `/${apiVersion2Prefix}/healthz`,
    getHealthzHandler({
      healthzEndpoint: getHealthzEndpoint(apiServerAddress, apiVersion2Prefix),
      healthzStats: getBuildMetadata(currDir),
    }),
  );

  /** Artifact */
  registerHandler(
    app.get,
    '/artifacts/*',
    getArtifactsProxyHandler({
      enabled: options.artifacts.proxy.enabled,
      allowedDomain: options.artifacts.allowedDomain,
      namespacedServiceGetter: getArtifactServiceGetter(options.artifacts.proxy),
    }),
  );
  // /artifacts/get endpoint tries to extract the artifact to return pure text content
  registerHandler(
    app.get,
    '/artifacts/get',
    getArtifactsHandler({
      artifactsConfigs: options.artifacts,
      useParameter: false,
      tryExtract: true,
      options: options,
    }),
  );
  // /artifacts/ endpoint downloads the artifact as is, it does not try to unzip or untar.
  registerHandler(
    app.get,
    // The last * represents object key. Key could contain special characters like '/',
    // so we cannot use `:key` as the placeholder.
    // It is important to include the original object's key at the end of the url, because
    // browser automatically determines file extension by the url. A wrong extension may affect
    // whether the file can be opened by the correct application by default.
    '/artifacts/:source/:bucket/*',
    getArtifactsHandler({
      artifactsConfigs: options.artifacts,
      useParameter: true,
      tryExtract: false,
      options: options,
    }),
  );

  /** Authorize function */
  const authorizeFn = getAuthorizeFn(options.auth, { apiServerAddress });

  /** Tensorboard viewer */
  const {
    get: tensorboardGetHandler,
    create: tensorboardCreateHandler,
    delete: tensorboardDeleteHandler,
  } = getTensorboardHandlers(options.viewer.tensorboard, authorizeFn);
  registerHandler(app.get, '/apps/tensorboard', tensorboardGetHandler);
  registerHandler(app.delete, '/apps/tensorboard', tensorboardDeleteHandler);
  registerHandler(app.post, '/apps/tensorboard', tensorboardCreateHandler);

  /** Pod logs - conditionally stream through API server, otherwise directly from k8s and archive */
  if (options.artifacts.streamLogsFromServerApi) {
    app.all(
      '/k8s/pod/logs',
      proxy({
        changeOrigin: true,
        onProxyReq: proxyReq => {
          console.log('Proxied log request: ', proxyReq.path);
        },
        headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
        pathRewrite: (pathStr: string, req: any) => {
          /** Argo nodeId is just POD name */
          const nodeId = req.query.podname;
          const runId = req.query.runid;
          return `/${apiVersion1Prefix}/runs/${runId}/nodes/${nodeId}/log`;
        },
        target: apiServerAddress,
      }),
    );
  } else {
    registerHandler(
      app.get,
      '/k8s/pod/logs',
      getPodLogsHandler(options.argo, options.artifacts, options.pod.logContainerName),
    );
  }

  if (options.artifacts.streamLogsFromServerApi) {
    app.all(
      '/k8s/pod/logs',
      proxy({
        changeOrigin: true,
        onProxyReq: proxyReq => {
          console.log('Proxied log request: ', proxyReq.path);
        },
        headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
        pathRewrite: (pathStr: string, req: any) => {
          /** Argo nodeId is just POD name */
          const nodeId = req.query.podname;
          const runId = req.query.runid;
          return `/${apiVersion2Prefix}/runs/${runId}/nodes/${nodeId}/log`;
        },
        target: apiServerAddress,
      }),
    );
  } else {
    registerHandler(
      app.get,
      '/k8s/pod/logs',
      getPodLogsHandler(options.argo, options.artifacts, options.pod.logContainerName),
    );
  }

  /** Pod info */
  registerHandler(app.get, '/k8s/pod', podInfoHandler);
  registerHandler(app.get, '/k8s/pod/events', podEventsHandler);

  /** Cluster metadata (GKE only) */
  registerHandler(app.get, '/system/cluster-name', getClusterNameHandler(options.gkeMetadata));
  registerHandler(app.get, '/system/project-id', getProjectIdHandler(options.gkeMetadata));

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
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
      target: envoyServiceAddress,
    }),
  );

  registerHandler(
    app.use,
    [
      // Original API endpoint is /runs/{run_id}:reportMetrics, but ':reportMetrics' means a url parameter, so we don't use : here.
      `/${apiVersion1Prefix}/runs/*reportMetrics`,
      `/${apiVersion1Prefix}/workflows`,
      `/${apiVersion1Prefix}/scheduledworkflows`,
      `/${apiVersion2Prefix}/workflows`,
      `/${apiVersion2Prefix}/scheduledworkflows`,
    ],
    (req, res) => {
      res.status(403).send(`${req.originalUrl} endpoint is not meant for external usage.`);
    },
  );

  // Order matters here, since both handlers can match any proxied request with a referer,
  // and we prioritize the basepath-friendly handler
  proxyMiddleware(app, `${basePath}/${apiVersion1Prefix}`);
  proxyMiddleware(app, `${basePath}/${apiVersion2Prefix}`);
  proxyMiddleware(app, `/${apiVersion1Prefix}`);
  proxyMiddleware(app, `/${apiVersion2Prefix}`);

  /** Proxy to ml-pipeline api server */
  app.all(
    `/${apiVersion1Prefix}/*`,
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied request: ', proxyReq.path);
      },
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
      target: apiServerAddress,
    }),
  );
  app.all(
    `/${apiVersion2Prefix}/*`,
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied request: ', proxyReq.path);
      },
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
      target: apiServerAddress,
    }),
  );

  app.all(
    `${basePath}/${apiVersion1Prefix}/*`,
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied request: ', proxyReq.path);
      },
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
      pathRewrite: pathStr =>
        pathStr.startsWith(basePath) ? pathStr.substr(basePath.length, pathStr.length) : pathStr,
      target: apiServerAddress,
    }),
  );
  app.all(
    `${basePath}/${apiVersion2Prefix}/*`,
    proxy({
      changeOrigin: true,
      onProxyReq: proxyReq => {
        console.log('Proxied request: ', proxyReq.path);
      },
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
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

  return app;
}
