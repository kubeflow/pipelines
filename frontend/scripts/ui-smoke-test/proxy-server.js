#!/usr/bin/env node
/**
 * Proxy Server for UI Smoke Tests
 *
 * Serves static files from a build directory while proxying API calls
 * to a real backend server.
 *
 * Usage: node proxy-server.js --build ./build --port 4001 --backend http://localhost:3000
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const url = require('url');

// Parse arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};

const BUILD_DIR = getArg('build', './build');
const PORT = parseInt(getArg('port', '4001'), 10);
const BACKEND_URL = getArg('backend', 'http://localhost:3000');

// MIME types
const MIME_TYPES = {
  '.html': 'text/html',
  '.js': 'application/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.gif': 'image/gif',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf',
  '.map': 'application/json',
};

// Paths that should be proxied to backend
const PROXY_PATHS = [
  '/apis/',
  '/system/',
  '/artifacts/',
  '/visualizations/',
  '/k8s/',
  '/apps/',
  '/ml_metadata.',
];

function shouldProxy(pathname) {
  return PROXY_PATHS.some(prefix => pathname.startsWith(prefix));
}

function proxyRequest(req, res) {
  const targetUrl = new URL(req.url, BACKEND_URL);
  const protocol = targetUrl.protocol === 'https:' ? https : http;

  const options = {
    hostname: targetUrl.hostname,
    port: targetUrl.port,
    path: targetUrl.pathname + targetUrl.search,
    method: req.method,
    headers: {
      ...req.headers,
      host: targetUrl.host,
    },
  };

  const proxyReq = protocol.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (err) => {
    console.error(`Proxy error: ${err.message}`);
    res.writeHead(502);
    res.end(`Proxy error: ${err.message}`);
  });

  req.pipe(proxyReq);
}

function serveStatic(req, res) {
  const parsedUrl = url.parse(req.url);
  let pathname = parsedUrl.pathname;

  // Default to index.html for root or SPA routes
  if (pathname === '/' || !path.extname(pathname)) {
    pathname = '/index.html';
  }

  const filePath = path.join(BUILD_DIR, pathname);

  // Prevent path traversal attacks
  const resolved = path.resolve(filePath);
  if (!resolved.startsWith(path.resolve(BUILD_DIR))) {
    res.writeHead(403);
    res.end('Forbidden');
    return;
  }

  const ext = path.extname(filePath);
  const contentType = MIME_TYPES[ext] || 'application/octet-stream';

  fs.readFile(filePath, (err, data) => {
    if (err) {
      // SPA fallback: serve index.html for missing files
      if (err.code === 'ENOENT') {
        fs.readFile(path.join(BUILD_DIR, 'index.html'), (err2, indexData) => {
          if (err2) {
            res.writeHead(404);
            res.end('Not found');
          } else {
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end(indexData);
          }
        });
      } else {
        res.writeHead(500);
        res.end(`Server error: ${err.code}`);
      }
    } else {
      res.writeHead(200, { 'Content-Type': contentType });
      res.end(data);
    }
  });
}

const server = http.createServer((req, res) => {
  const parsedUrl = url.parse(req.url);

  if (shouldProxy(parsedUrl.pathname)) {
    proxyRequest(req, res);
  } else {
    serveStatic(req, res);
  }
});

server.listen(PORT, () => {
  console.log(`Proxy server running on http://localhost:${PORT}`);
  console.log(`Serving static files from: ${BUILD_DIR}`);
  console.log(`Proxying API calls to: ${BACKEND_URL}`);
});
