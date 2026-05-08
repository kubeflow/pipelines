// Copyright 2018 The Kubeflow Authors
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

import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { URL, URLSearchParams } from 'url';
import { HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS } from './consts.js';

// Cloud metadata endpoints (AWS, GCP, Azure, DigitalOcean, Oracle, Alibaba).
const BLOCKED_HOSTS = new Set([
  '169.254.169.254',
  'metadata.google.internal',
  'metadata.goog',
  '100.100.100.200',
]);

/**
 * Returns true when the IPv4 address (as four numeric octets) falls in a
 * private, loopback, or link-local range.
 */
function _isPrivateIPv4(a: number, b: number): boolean {
  return (
    a === 127 || // 127.0.0.0/8  loopback
    a === 10 || // 10.0.0.0/8   private
    (a === 172 && b >= 16 && b <= 31) || // 172.16.0.0/12 private
    (a === 192 && b === 168) || // 192.168.0.0/16 private
    (a === 169 && b === 254) || // 169.254.0.0/16 link-local
    a === 0 // 0.0.0.0/8
  );
}

/**
 * Returns true when the hostname resolves to a private, loopback, or
 * link-local IP address, or when the hostname matches a known cloud
 * metadata endpoint. These destinations must never be reachable through
 * the proxy.
 *
 * NOTE: This check is hostname-based and does not perform DNS resolution.
 * Wildcard DNS services (e.g., 169.254.169.254.nip.io) or DNS rebinding
 * attacks can bypass it. Defence in depth (network policies, egress
 * firewalls) is recommended in addition to this check.
 */
export function _isBlockedTarget(hostname: string): boolean {
  // Normalise: strip brackets, lowercase, and remove a single trailing dot
  // (FQDN form). Resolvers treat "localhost." the same as "localhost", so
  // without this step an attacker can bypass every string check below.
  let h = hostname.replace(/^\[|\]$/g, '').toLowerCase();
  if (h.endsWith('.')) {
    h = h.slice(0, -1);
  }

  if (BLOCKED_HOSTS.has(h)) {
    return true;
  }

  // Block raw IPv4 loopback and private ranges
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(h)) {
    const parts = h.split('.').map(Number);
    if (_isPrivateIPv4(parts[0], parts[1])) {
      return true;
    }
  }

  // Block IPv6 loopback, link-local, and Unique Local Addresses (ULA)
  if (h === '::1' || h === '' || h === '::') {
    return true;
  }
  if (/^fe80:/i.test(h) || /^fc00:/i.test(h) || /^fd[0-9a-f]{2}:/i.test(h)) {
    return true;
  }

  // IPv4-mapped IPv6 addresses: Node.js normalises ::ffff:127.0.0.1 to the
  // hex form ::ffff:7f00:1, so we must handle both dotted-decimal and hex.
  const mappedDotted = h.match(/^::ffff:(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$/);
  if (mappedDotted) {
    const parts = mappedDotted[1].split('.').map(Number);
    return _isPrivateIPv4(parts[0], parts[1]);
  }
  const mappedHex = h.match(/^::ffff:([0-9a-f]{1,4}):([0-9a-f]{1,4})$/);
  if (mappedHex) {
    const hi = parseInt(mappedHex[1], 16);
    const lo = parseInt(mappedHex[2], 16);
    const a = (hi >> 8) & 0xff;
    const b = hi & 0xff;
    return _isPrivateIPv4(a, b);
  }

  // Block localhost and *.localhost (RFC 6761)
  if (h === 'localhost' || h.endsWith('.localhost')) {
    return true;
  }

  return false;
}

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

  // Validate the proxy target before forwarding. Reject requests aimed at
  // private networks, loopback addresses, or cloud metadata endpoints.
  app.all(proxyPrefix + '*', (req, res, next) => {
    try {
      const target = _routePathWithReferer(proxyPrefix, req.path, req.headers.referer as string);
      const targetUrl = new URL(target);
      if (_isBlockedTarget(targetUrl.hostname)) {
        console.warn(`Blocked proxy request to private/internal target: ${targetUrl.hostname}`);
        res.status(403).send('Proxy target is not allowed.');
        return;
      }
    } catch {
      res.status(400).send('Invalid proxy target URL.');
      return;
    }
    next();
  });

  app.all(
    proxyPrefix + '*',
    createProxyMiddleware({
      changeOrigin: true,
      logLevel: process.env.NODE_ENV === 'test' ? 'warn' : 'debug',
      target: 'http://127.0.0.1',
      router: (req: any) => {
        return _routePathWithReferer(proxyPrefix, req.path, req.headers.referer as string);
      },
      pathRewrite: (_: any, req: any) => {
        return _rewritePath(proxyPrefix, req.path, req.query);
      },
      headers: HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS,
    }),
  );
};
