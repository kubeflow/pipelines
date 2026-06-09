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

import { createHmac, randomBytes, timingSafeEqual } from 'crypto';
import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { URL, URLSearchParams } from 'url';
import { ViewerTensorboardConfig } from '../configs.js';
import { HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS } from '../consts.js';
import { AuthorizeFn } from '../helpers/auth.js';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';
import { isAllowedResourceName } from '../utils.js';

const DEFAULT_CLUSTER_DOMAIN = '.svc.cluster.local';
const TENSORBOARD_PROXY_PREFIX = '/apps/tensorboard/proxy/';
const TENSORBOARD_PROXY_SECRET = randomBytes(32);

interface TensorboardProxyPayload {
  namespace: string;
  viewerName: string;
}

interface ParsedTensorboardProxyRequest {
  proxyPath: string;
  token: string;
}

function signTensorboardProxyPayload(serializedPayload: string): string {
  return createHmac('sha256', TENSORBOARD_PROXY_SECRET)
    .update(serializedPayload)
    .digest('base64url');
}

function normalizeClusterDomain(clusterDomain: string): string {
  if (!clusterDomain) {
    return DEFAULT_CLUSTER_DOMAIN;
  }
  return clusterDomain.startsWith('.') ? clusterDomain : `.${clusterDomain}`;
}

export function createTensorboardProxyPath(namespace: string, viewerName: string): string {
  const serializedPayload = JSON.stringify({
    namespace,
    viewerName,
  } satisfies TensorboardProxyPayload);
  const encodedPayload = Buffer.from(serializedPayload).toString('base64url');
  const signature = signTensorboardProxyPayload(serializedPayload);
  return `${TENSORBOARD_PROXY_PREFIX.slice(1)}${encodeURIComponent(`${encodedPayload}.${signature}`)}/`;
}

export function buildTensorboardProxyTarget(
  namespace: string,
  viewerName: string,
  clusterDomain: string,
): string {
  return `http://${viewerName}-service.${namespace}${normalizeClusterDomain(clusterDomain)}:80`;
}

export function buildTensorboardProxyUpstreamPath(
  viewerName: string,
  proxyPath: string,
  query: string,
): string {
  const queryString = new URLSearchParams(query).toString();
  const upstreamPath =
    proxyPath === '/' ? `/tensorboard/${viewerName}/` : `/tensorboard/${viewerName}${proxyPath}`;
  return upstreamPath + (queryString ? `?${queryString}` : '');
}

export function getTensorboardProxyBasePath(proxyPrefix: string, referer = ''): string | undefined {
  try {
    const pathname = new URL(referer).pathname;
    const prefixIndex = pathname.indexOf(proxyPrefix);
    if (prefixIndex < 0) {
      return undefined;
    }
    const parsedRequest = parseTensorboardProxyRequest(proxyPrefix, pathname.slice(prefixIndex));
    if (!parsedRequest) {
      return undefined;
    }
    return `${pathname.slice(0, prefixIndex)}${proxyPrefix}${encodeURIComponent(parsedRequest.token)}`;
  } catch {
    return undefined;
  }
}

export function parseTensorboardProxyPayload(token: string): TensorboardProxyPayload | undefined {
  const decodedToken = decodeURIComponent(token);
  const [encodedPayload, signature, extraPart] = decodedToken.split('.');
  if (!encodedPayload || !signature || extraPart) {
    return undefined;
  }

  let serializedPayload: string;
  try {
    serializedPayload = Buffer.from(encodedPayload, 'base64url').toString('utf8');
  } catch {
    return undefined;
  }

  const expectedSignature = signTensorboardProxyPayload(serializedPayload);
  if (signature.length !== expectedSignature.length) {
    return undefined;
  }
  if (!timingSafeEqual(Buffer.from(signature), Buffer.from(expectedSignature))) {
    return undefined;
  }

  let payload: Partial<TensorboardProxyPayload>;
  try {
    payload = JSON.parse(serializedPayload);
  } catch {
    return undefined;
  }

  if (
    typeof payload.namespace !== 'string' ||
    typeof payload.viewerName !== 'string' ||
    !isAllowedResourceName(payload.namespace) ||
    !isAllowedResourceName(payload.viewerName)
  ) {
    return undefined;
  }

  return {
    namespace: payload.namespace,
    viewerName: payload.viewerName,
  };
}

export function parseTensorboardProxyRequest(
  proxyPrefix: string,
  requestPath: string,
): ParsedTensorboardProxyRequest | undefined {
  if (!requestPath.startsWith(proxyPrefix)) {
    return undefined;
  }

  const remainder = requestPath.slice(proxyPrefix.length);
  if (!remainder) {
    return undefined;
  }

  const firstSlashIndex = remainder.indexOf('/');
  const token = firstSlashIndex < 0 ? remainder : remainder.slice(0, firstSlashIndex);
  if (!token) {
    return undefined;
  }

  return {
    token,
    proxyPath: firstSlashIndex < 0 ? '/' : `/${remainder.slice(firstSlashIndex + 1)}`,
  };
}

export default function registerTensorboardProxy(
  app: express.Application,
  basePath: string,
  tensorboardConfig: ViewerTensorboardConfig,
  authorizeFn: AuthorizeFn,
) {
  app.use((req, _, next) => {
    const proxyBasePath = getTensorboardProxyBasePath(
      TENSORBOARD_PROXY_PREFIX,
      req.headers.referer as string,
    );
    if (proxyBasePath && req.url.indexOf(TENSORBOARD_PROXY_PREFIX) < 0) {
      req.url = `${proxyBasePath}${req.url}`;
    }
    next();
  });

  const proxyRoutes = [`${TENSORBOARD_PROXY_PREFIX}*`, `${basePath}${TENSORBOARD_PROXY_PREFIX}*`];

  app.all(proxyRoutes, async (req, res, next) => {
    try {
      const prefixIndex = req.path.indexOf(TENSORBOARD_PROXY_PREFIX);
      const parsedRequest =
        prefixIndex < 0
          ? undefined
          : parseTensorboardProxyRequest(TENSORBOARD_PROXY_PREFIX, req.path.slice(prefixIndex));
      if (!parsedRequest) {
        res.status(404).send('TensorBoard proxy target not found');
        return;
      }

      const payload = parseTensorboardProxyPayload(parsedRequest.token);
      if (!payload) {
        res.status(403).send('Invalid TensorBoard proxy target');
        return;
      }

      const authError = await authorizeFn(
        {
          verb: AuthorizeRequestVerb.GET,
          resources: AuthorizeRequestResources.VIEWERS,
          namespace: payload.namespace,
        },
        req,
      );
      if (authError) {
        res.status(403).send('Access denied to namespace');
        return;
      }

      (req as any).tensorboardProxy = {
        ...payload,
        proxyPath: parsedRequest.proxyPath,
      };
      next();
    } catch (error) {
      console.error('Failed to authorize TensorBoard proxy request:', error);
      res.status(500).send('Authorization check failed');
    }
  });

  app.all(
    proxyRoutes,
    createProxyMiddleware({
      changeOrigin: true,
      logLevel: process.env.NODE_ENV === 'test' ? 'warn' : 'debug',
      target: 'http://127.0.0.1',
      router: (req: any) => {
        const { namespace, viewerName } = req.tensorboardProxy as TensorboardProxyPayload;
        return buildTensorboardProxyTarget(namespace, viewerName, tensorboardConfig.clusterDomain);
      },
      pathRewrite: (_: any, req: any) => {
        const { proxyPath, viewerName } = req.tensorboardProxy as ParsedTensorboardProxyRequest &
          TensorboardProxyPayload;
        return buildTensorboardProxyUpstreamPath(viewerName, proxyPath, req.query);
      },
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
    }),
  );
}
