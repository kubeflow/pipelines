// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

// Local sign-in fixture for clusters deployed without the distribution's OAuth gateway.
// Only this loopback proxy supplies identity. The real frontend and API enforce RBAC.
const http = require('node:http');
const { randomBytes } = require('node:crypto');

const loginPath = '/__profile_fixture/login';
const sessionPath = '/__profile_fixture/session';
const loginForm = `<!doctype html><title>Profile fixture sign in</title><h1>Sign in</h1>
<form method="post"><label>Username <input type="text" name="username" required></label>
<label>Password <input type="password" name="password" required></label>
<button type="submit">Sign in</button></form>`;

async function startFixtureGateway(upstreamBaseURL, identities) {
  const upstream = new URL(upstreamBaseURL);
  if (upstream.protocol !== 'http:' || upstream.hostname !== '127.0.0.1' ||
      upstream.username || upstream.password || upstream.pathname !== '/' || upstream.search || upstream.hash) {
    throw new Error('Expected the HTTP frontend port-forward on 127.0.0.1');
  }
  if (identities.length !== 2 || new Set(identities).size !== 2 ||
      identities.some(id => typeof id !== 'string' || !id)) {
    throw new Error('Expected two distinct fixture identities');
  }
  const passwords = new Map();
  for (const id of identities) {
    http.validateHeaderValue('kubeflow-userid', id);
    passwords.set(id, randomBytes(32).toString('hex'));
  }
  const sessions = new Map();
  const cookie = 'profile_session_' + randomBytes(8).toString('hex');
  let url;
  const server = http.createServer(async (req, res) => {
    const reply = (status, body = '', headers = {}) => {
      res.writeHead(status, { 'Cache-Control': 'no-store', ...headers });
      res.end(body);
    };
    if ('http://' + req.headers.host !== url || !req.url.startsWith('/') || req.url.startsWith('//')) {
      return reply(400);
    }
    if (req.headers.origin && req.headers.origin !== url) return reply(403);
    if (req.url === loginPath) {
      if (req.method === 'GET') return reply(200, loginForm, { 'Content-Type': 'text/html' });
      if (req.method !== 'POST' || req.headers.origin !== url) return reply(403);
      let body = '';
      try {
        for await (const chunk of req) {
          body += chunk;
          if (body.length > 1024) return reply(413);
        }
      } catch {
        return reply(400);
      }
      const fields = new URLSearchParams(body);
      const identity = fields.get('username');
      if (!passwords.has(identity) || passwords.get(identity) !== fields.get('password')) return reply(401);
      const token = randomBytes(32).toString('hex');
      sessions.set(token, identity);
      return reply(303, '', {
        Location: sessionPath,
        'Set-Cookie': cookie + '=' + token + '; HttpOnly; SameSite=Strict; Path=/',
      });
    }
    const session = (req.headers.cookie || '').split(';').map(value => value.trim())
      .find(value => value.startsWith(cookie + '='));
    const identity = session && sessions.get(session.slice(cookie.length + 1));
    if (!identity) {
      if (req.method === 'GET' && req.headers['sec-fetch-mode'] === 'navigate') {
        return reply(303, '', { Location: loginPath });
      }
      return reply(401);
    }
    if (req.url === sessionPath) return reply(200, 'Signed in', { 'Content-Type': 'text/plain' });

    // Allowlist ordinary request headers. Caller-supplied identities, credentials,
    // session cookies, and proxy headers never reach the port-forward.
    const headers = { 'kubeflow-userid': identity };
    for (const key of ['accept', 'accept-encoding', 'accept-language', 'content-length', 'content-type',
      'grpc-timeout', 'if-modified-since', 'if-none-match', 'range', 'user-agent', 'x-grpc-web', 'x-user-agent']) {
      if (req.headers[key] !== undefined) headers[key] = req.headers[key];
    }
    const proxy = http.request(upstream, { path: req.url, method: req.method, headers }, response => {
      res.writeHead(response.statusCode, response.headers);
      response.on('error', () => res.destroy());
      response.pipe(res);
    });
    proxy.on('error', () => res.headersSent ? res.destroy() : reply(502, 'Frontend unavailable'));
    res.on('close', () => proxy.destroy());
    req.pipe(proxy);
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  url = 'http://127.0.0.1:' + server.address().port;
  return {
    url,
    async signIn(page, identity) {
      if (!passwords.has(identity)) throw new Error('Unknown fixture identity');
      await page.goto(url + loginPath);
      await page.getByLabel('Username').fill(identity);
      await page.getByLabel('Password').fill(passwords.get(identity));
      await page.getByRole('button', { name: 'Sign in' }).click();
      await page.waitForURL(url + sessionPath);
    },
    close() {
      const closed = new Promise((resolve, reject) => server.close(err => err ? reject(err) : resolve()));
      server.closeAllConnections();
      return closed;
    },
  };
}

async function openFixturePage(browser, gateway, t, { identity, namespace }) {
  const context = await browser.newContext({ baseURL: gateway.url });
  t.after(() => context.close());
  // The disposable deployment has no Central Dashboard. Supply its namespace-selection event.
  await context.addInitScript(selected => {
    window.centraldashboard = {
      CentralDashboardEventHandler: {
        init(register) {
          const handler = {};
          register(handler);
          handler.onNamespaceSelected(selected);
        },
      },
    };
  }, namespace);
  await context.route('**/dashboard_lib.bundle.js', route =>
    route.fulfill({ status: 200, contentType: 'application/javascript', body: '' }),
  );
  const page = await context.newPage();
  page.setDefaultTimeout(30000);
  await gateway.signIn(page, identity);
  return page;
}

module.exports = { openFixturePage, startFixtureGateway };
