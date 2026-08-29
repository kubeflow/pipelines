#!/usr/bin/env node
/**
 * Serve a built KFP frontend while proxying API requests to a local backend.
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');

const LISTEN_HOST = '127.0.0.1';
const DEFAULT_PROXY_TIMEOUT_MS = 15000;

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.webp': 'image/webp',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf',
  '.map': 'application/json; charset=utf-8',
};

const PROXY_PATHS = [
  '/apis/',
  '/system/',
  '/artifacts/',
  '/visualizations/',
  '/k8s/',
  '/apps/',
  '/ml_metadata.',
];

const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'proxy-connection',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
]);
const READ_ONLY_HTTP_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);
const READ_ONLY_MLMD_RPC = /^\/ml_metadata\.MetadataStoreService\/Get[A-Za-z0-9_]*$/;

function parseArgumentValues(argv, defaults = {}) {
  const values = { ...defaults };
  const seen = new Set();
  const knownArguments = new Set(['build', 'port', 'backend']);

  for (let index = 0; index < argv.length; index++) {
    const argument = argv[index];
    if (!argument.startsWith('--')) {
      throw new Error(`Unexpected argument: ${argument}`);
    }

    const name = argument.slice(2);
    if (!knownArguments.has(name)) {
      throw new Error(`Unknown argument: --${name}`);
    }
    if (seen.has(name)) {
      throw new Error(`Argument --${name} may only be provided once`);
    }

    const value = argv[index + 1];
    if (value === undefined || value.startsWith('--')) {
      throw new Error(`Argument --${name} requires a value`);
    }
    values[name] = value;
    seen.add(name);
    index++;
  }

  return values;
}

function validatePort(value) {
  const text = String(value);
  if (!/^\d+$/.test(text)) {
    throw new Error(`Invalid port: ${text}`);
  }
  const port = Number(text);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`Port must be between 1 and 65535: ${text}`);
  }
  return port;
}

function validateBackendUrl(value) {
  let backendUrl;
  try {
    backendUrl = new URL(value);
  } catch (_error) {
    throw new Error(`Invalid backend URL: ${value}`);
  }

  if (backendUrl.protocol !== 'http:' && backendUrl.protocol !== 'https:') {
    throw new Error('Backend URL must use http: or https:');
  }
  if (backendUrl.username || backendUrl.password) {
    throw new Error('Backend URL must not include credentials');
  }
  if (backendUrl.search || backendUrl.hash) {
    throw new Error('Backend URL must not include a query string or fragment');
  }
  return backendUrl;
}

function validateBuildDirectory(value, cwd = process.cwd()) {
  const resolved = path.resolve(cwd, value);
  let stat;
  try {
    stat = fs.statSync(resolved);
  } catch (_error) {
    throw new Error(`Build directory does not exist: ${resolved}`);
  }
  if (!stat.isDirectory()) {
    throw new Error(`Build path is not a directory: ${resolved}`);
  }
  return fs.realpathSync(resolved);
}

function parseCliArgs(argv, options = {}) {
  const values = parseArgumentValues(argv, {
    build: './build',
    port: '4001',
    backend: 'http://localhost:3000',
  });
  return {
    backendUrl: validateBackendUrl(values.backend),
    buildDir: validateBuildDirectory(values.build, options.cwd),
    port: validatePort(values.port),
    proxyTimeoutMs: DEFAULT_PROXY_TIMEOUT_MS,
  };
}

function validateServerConfig(config) {
  if (!config || typeof config !== 'object') {
    throw new Error('Proxy server configuration is required');
  }
  const proxyTimeoutMs = config.proxyTimeoutMs ?? DEFAULT_PROXY_TIMEOUT_MS;
  if (!Number.isFinite(proxyTimeoutMs) || proxyTimeoutMs <= 0) {
    throw new Error('Proxy timeout must be a positive number');
  }
  if (config.requestFactory !== undefined && typeof config.requestFactory !== 'function') {
    throw new Error('Proxy request factory must be a function when provided');
  }
  return {
    backendUrl: validateBackendUrl(config.backendUrl),
    buildDir: validateBuildDirectory(config.buildDir),
    proxyTimeoutMs,
    requestFactory: config.requestFactory || null,
  };
}

function parseRequestTarget(requestTarget) {
  if (
    typeof requestTarget !== 'string' ||
    requestTarget.length === 0 ||
    !requestTarget.startsWith('/') ||
    requestTarget.startsWith('//') ||
    requestTarget.includes('\\') ||
    /[\u0000-\u001f\u007f]/.test(requestTarget)
  ) {
    throw new Error('Request target must use origin-form');
  }

  if (requestTarget.includes('#')) {
    throw new Error('Request target must not include a fragment');
  }
  const queryIndex = requestTarget.indexOf('?');
  const rawPathname = queryIndex === -1 ? requestTarget : requestTarget.slice(0, queryIndex);
  const search = queryIndex === -1 ? '' : requestTarget.slice(queryIndex);

  let pathname;
  try {
    pathname = decodeURIComponent(rawPathname);
  } catch (_error) {
    throw new Error('Request path contains invalid percent encoding');
  }
  if (pathname.includes('\0') || pathname.includes('\\')) {
    throw new Error('Request path contains invalid characters');
  }
  return { pathname, rawPathname, search };
}

function shouldProxy(pathname) {
  return PROXY_PATHS.some((prefix) => pathname.startsWith(prefix));
}

function isReadOnlyBackendRequest(method, pathname) {
  const normalizedMethod = String(method || '').toUpperCase();
  return (
    READ_ONLY_HTTP_METHODS.has(normalizedMethod) ||
    (normalizedMethod === 'POST' && READ_ONLY_MLMD_RPC.test(pathname))
  );
}

function isPathInside(rootPath, candidatePath) {
  const relative = path.relative(rootPath, candidatePath);
  return (
    relative === '' ||
    (!relative.startsWith(`..${path.sep}`) && relative !== '..' && !path.isAbsolute(relative))
  );
}

async function resolveExistingFile(buildDir, pathname) {
  const candidate = path.resolve(buildDir, `.${pathname}`);
  if (!isPathInside(buildDir, candidate)) {
    return { status: 'forbidden' };
  }

  let realPath;
  try {
    realPath = await fs.promises.realpath(candidate);
  } catch (error) {
    if (error.code === 'ENOENT' || error.code === 'ENOTDIR') {
      return { status: 'missing' };
    }
    throw error;
  }

  if (!isPathInside(buildDir, realPath)) {
    return { status: 'forbidden' };
  }

  const stat = await fs.promises.stat(realPath);
  if (!stat.isFile()) {
    return { status: 'missing' };
  }
  return { path: realPath, stat, status: 'file' };
}

function sendText(res, statusCode, text, headers = {}) {
  if (res.headersSent) {
    res.destroy();
    return;
  }
  const body = Buffer.from(text);
  res.writeHead(statusCode, {
    'Content-Length': body.length,
    'Content-Type': 'text/plain; charset=utf-8',
    ...headers,
  });
  res.end(res.req?.method === 'HEAD' ? undefined : body);
}

async function serveStatic(req, res, buildDir, parsedTarget) {
  if (req.method !== 'GET' && req.method !== 'HEAD') {
    sendText(res, 405, 'Method not allowed', { Allow: 'GET, HEAD' });
    return;
  }

  let requestedPath = parsedTarget.pathname === '/' ? '/index.html' : parsedTarget.pathname;
  let resolved = await resolveExistingFile(buildDir, requestedPath);

  if (resolved.status === 'forbidden') {
    sendText(res, 403, 'Forbidden');
    return;
  }

  if (resolved.status === 'missing') {
    sendText(res, 404, 'Not found');
    return;
  }

  const data = await fs.promises.readFile(resolved.path);
  const contentType =
    MIME_TYPES[path.extname(resolved.path).toLowerCase()] || 'application/octet-stream';
  res.writeHead(200, {
    'Content-Length': data.length,
    'Content-Type': contentType,
  });
  if (req.method === 'HEAD') {
    res.end();
  } else {
    res.end(data);
  }
}

function filterHeaders(headers) {
  return Object.fromEntries(
    Object.entries(headers).filter(
      ([name, value]) => value !== undefined && !HOP_BY_HOP_HEADERS.has(name.toLowerCase()),
    ),
  );
}

function buildBackendPath(backendUrl, parsedTarget) {
  const basePath = backendUrl.pathname === '/' ? '' : backendUrl.pathname.replace(/\/$/, '');
  return `${basePath}${parsedTarget.rawPathname}${parsedTarget.search}`;
}

function proxyRequest(
  req,
  res,
  backendUrl,
  parsedTarget,
  timeoutMs = DEFAULT_PROXY_TIMEOUT_MS,
  requestFactory = null,
) {
  const protocol = backendUrl.protocol === 'https:' ? https : http;
  let timedOut = false;
  const requestHeaders = filterHeaders(req.headers);
  requestHeaders.host = backendUrl.host;

  const proxyReq = (requestFactory || protocol.request.bind(protocol))(
    {
      headers: requestHeaders,
      hostname: backendUrl.hostname,
      method: req.method,
      path: buildBackendPath(backendUrl, parsedTarget),
      port: backendUrl.port || undefined,
      protocol: backendUrl.protocol,
    },
    (proxyRes) => {
      const responseHeaders = filterHeaders(proxyRes.headers);
      res.writeHead(proxyRes.statusCode || 502, responseHeaders);

      proxyRes.setTimeout(timeoutMs, () => {
        timedOut = true;
        proxyRes.destroy(new Error('Backend response timed out'));
      });
      proxyRes.on('error', () => {
        if (!res.destroyed) res.destroy();
      });

      if (req.method === 'HEAD') {
        proxyRes.resume();
        res.end();
      } else {
        proxyRes.pipe(res);
      }
    },
  );

  proxyReq.setTimeout(timeoutMs, () => {
    timedOut = true;
    proxyReq.destroy(new Error('Backend request timed out'));
  });
  proxyReq.on('error', (error) => {
    console.error(`Proxy request failed: ${error.message}`);
    if (!res.headersSent) {
      sendText(res, timedOut ? 504 : 502, timedOut ? 'Gateway timeout' : 'Bad gateway');
    } else if (!res.destroyed) {
      res.destroy();
    }
  });
  req.on('aborted', () => proxyReq.destroy());
  req.pipe(proxyReq);
}

function createRequestHandler(config) {
  const validatedConfig = validateServerConfig(config);
  return (req, res) => {
    let parsedTarget;
    try {
      parsedTarget = parseRequestTarget(req.url);
    } catch (error) {
      sendText(res, 400, error.message);
      return;
    }

    if (shouldProxy(parsedTarget.pathname)) {
      if (!isReadOnlyBackendRequest(req.method, parsedTarget.pathname)) {
        sendText(res, 405, 'Backend mutation is disabled during screenshot capture', {
          Allow: 'GET, HEAD, OPTIONS',
        });
        return;
      }
      proxyRequest(
        req,
        res,
        validatedConfig.backendUrl,
        parsedTarget,
        validatedConfig.proxyTimeoutMs,
        validatedConfig.requestFactory,
      );
      return;
    }

    serveStatic(req, res, validatedConfig.buildDir, parsedTarget).catch((error) => {
      console.error(`Static file error: ${error.message}`);
      sendText(res, 500, 'Internal server error');
    });
  };
}

function createProxyServer(config) {
  return http.createServer(createRequestHandler(config));
}

function listen(server, port, callback) {
  return server.listen(port, LISTEN_HOST, callback);
}

function main(argv = process.argv.slice(2)) {
  let config;
  try {
    config = parseCliArgs(argv);
  } catch (error) {
    console.error(`Error: ${error.message}`);
    process.exitCode = 1;
    return;
  }

  const server = createProxyServer(config);
  server.on('error', (error) => {
    console.error(`Proxy server failed: ${error.message}`);
    process.exitCode = 1;
  });
  listen(server, config.port, () => {
    console.log(`Proxy server running on http://${LISTEN_HOST}:${config.port}`);
    console.log(`Serving static files from: ${config.buildDir}`);
    console.log(`Proxying API calls to: ${config.backendUrl.origin}`);
  });
}

if (require.main === module) {
  main();
}

module.exports = {
  DEFAULT_PROXY_TIMEOUT_MS,
  LISTEN_HOST,
  buildBackendPath,
  createProxyServer,
  createRequestHandler,
  filterHeaders,
  isPathInside,
  isReadOnlyBackendRequest,
  listen,
  main,
  parseCliArgs,
  parseRequestTarget,
  proxyRequest,
  resolveExistingFile,
  serveStatic,
  shouldProxy,
  validateBackendUrl,
  validateBuildDirectory,
  validatePort,
  validateServerConfig,
};
