const assert = require('node:assert/strict');
const { EventEmitter, once } = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { Readable, Writable } = require('node:stream');
const test = require('node:test');

const {
  createRequestHandler,
  isReadOnlyBackendRequest,
  listen,
  parseCliArgs,
  parseRequestTarget,
  validateBackendUrl,
  validatePort,
} = require('../proxy-server');

function createFixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-proxy-hardening-'));
  const buildDir = path.join(root, 'build');
  fs.mkdirSync(buildDir);
  fs.writeFileSync(path.join(buildDir, 'index.html'), '<main>SPA shell</main>');
  fs.writeFileSync(path.join(buildDir, 'asset.js'), 'window.loaded = true;');
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  return { buildDir, root };
}

class MockResponse extends Writable {
  constructor(method) {
    super();
    this.bodyChunks = [];
    this.headers = {};
    this.headersSent = false;
    this.req = { method };
    this.statusCode = null;
  }

  _write(chunk, _encoding, callback) {
    this.bodyChunks.push(Buffer.from(chunk));
    callback();
  }

  writeHead(statusCode, headers = {}) {
    this.statusCode = statusCode;
    this.headers = { ...headers };
    this.headersSent = true;
    return this;
  }

  get body() {
    return Buffer.concat(this.bodyChunks).toString('utf8');
  }
}

function createRequest({ body = '', headers = {}, method = 'GET', url = '/' } = {}) {
  const request = Readable.from(body ? [body] : []);
  request.headers = headers;
  request.method = method;
  request.url = url;
  return request;
}

async function invoke(handler, options = {}) {
  const request = createRequest(options);
  const response = new MockResponse(request.method);
  const finished = Promise.race([once(response, 'finish'), once(response, 'close')]);
  handler(request, response);
  await finished;
  return response;
}

function proxyConfig(buildDir, backendUrl, overrides = {}) {
  return {
    backendUrl: new URL(backendUrl),
    buildDir,
    proxyTimeoutMs: 1000,
    ...overrides,
  };
}

function successfulRequestFactory(observed) {
  return (options, callback) => {
    Object.assign(observed, options);
    const clientRequest = new Writable({
      write(_chunk, _encoding, done) {
        done();
      },
    });
    clientRequest.setTimeout = () => {};
    const backendResponse = Readable.from(['{"ok":true}']);
    backendResponse.headers = { 'content-type': 'application/json' };
    backendResponse.statusCode = 201;
    backendResponse.setTimeout = () => {};
    queueMicrotask(() => callback(backendResponse));
    return clientRequest;
  };
}

test('CLI validation rejects unsafe ports, URLs, and build paths', (t) => {
  const { buildDir, root } = createFixture(t);
  const parsed = parseCliArgs([
    '--build',
    buildDir,
    '--port',
    '4123',
    '--backend',
    'https://backend.example.test/base',
  ]);
  assert.equal(parsed.buildDir, fs.realpathSync(buildDir));
  assert.equal(parsed.port, 4123);
  assert.equal(parsed.backendUrl.href, 'https://backend.example.test/base');

  assert.throws(() => validatePort('4001junk'), /Invalid port/);
  assert.throws(() => validatePort('0'), /between 1 and 65535/);
  assert.throws(() => validatePort('65536'), /between 1 and 65535/);
  assert.throws(() => validateBackendUrl('file:///tmp/backend'), /http: or https:/);
  assert.throws(() => validateBackendUrl('https://user:secret@example.test'), /credentials/);
  assert.throws(() => parseCliArgs(['--build', path.join(root, 'missing')]), /does not exist/);
});

test('static handler supports HEAD and returns 404 for every missing asset path', async (t) => {
  const { buildDir } = createFixture(t);
  const handler = createRequestHandler(proxyConfig(buildDir, 'http://backend.example.test'));

  const asset = await invoke(handler, { url: '/asset.js' });
  assert.equal(asset.statusCode, 200);
  assert.equal(asset.body, 'window.loaded = true;');

  const head = await invoke(handler, { method: 'HEAD', url: '/asset.js' });
  assert.equal(head.statusCode, 200);
  assert.equal(head.body, '');
  assert.equal(Number(head.headers['Content-Length']), Buffer.byteLength('window.loaded = true;'));

  const route = await invoke(handler, { url: '/pipelines/details/123' });
  assert.equal(route.statusCode, 404);
  assert.equal(route.body, 'Not found');

  const missingAsset = await invoke(handler, { url: '/missing.js' });
  assert.equal(missingAsset.statusCode, 404);
  assert.equal(missingAsset.body, 'Not found');

  const extensionlessAsset = await invoke(handler, { url: '/assets/missing' });
  assert.equal(extensionlessAsset.statusCode, 404);
  assert.equal(extensionlessAsset.body, 'Not found');
});

test('static handler rejects sibling-prefix and symlink traversal', async (t) => {
  const { buildDir, root } = createFixture(t);
  const siblingDir = path.join(root, 'build-secret');
  fs.mkdirSync(siblingDir);
  fs.writeFileSync(path.join(siblingDir, 'secret.txt'), 'sibling secret');
  const outsideFile = path.join(root, 'outside.txt');
  fs.writeFileSync(outsideFile, 'symlink secret');
  fs.symlinkSync(outsideFile, path.join(buildDir, 'leak.txt'));
  const handler = createRequestHandler(proxyConfig(buildDir, 'http://backend.example.test'));

  const sibling = await invoke(handler, { url: '/../build-secret/secret.txt' });
  assert.equal(sibling.statusCode, 403);
  assert.equal(sibling.body, 'Forbidden');

  const symlink = await invoke(handler, { url: '/leak.txt' });
  assert.equal(symlink.statusCode, 403);
  assert.equal(symlink.body, 'Forbidden');
});

test('absolute-form request targets are rejected before proxying', async (t) => {
  const { buildDir } = createFixture(t);
  let backendRequests = 0;
  const handler = createRequestHandler(
    proxyConfig(buildDir, 'http://backend.example.test', {
      requestFactory() {
        backendRequests++;
        throw new Error('must not proxy');
      },
    }),
  );

  assert.throws(
    () => parseRequestTarget('http://attacker.invalid/apis/v2beta1/runs'),
    /origin-form/,
  );
  const response = await invoke(handler, {
    url: 'http://attacker.invalid/apis/v2beta1/runs',
  });
  assert.equal(response.statusCode, 400);
  assert.equal(backendRequests, 0);
});

test('proxy fixes the backend host and preserves its path prefix and query', async (t) => {
  const { buildDir } = createFixture(t);
  const observed = {};
  const handler = createRequestHandler(
    proxyConfig(buildDir, 'http://backend.example.test:8123/prefix/', {
      requestFactory: successfulRequestFactory(observed),
    }),
  );

  const response = await invoke(handler, { url: '/apis/v2beta1/runs?page_size=10' });
  assert.equal(response.statusCode, 201);
  assert.equal(response.body, '{"ok":true}');
  assert.equal(observed.hostname, 'backend.example.test');
  assert.equal(observed.path, '/prefix/apis/v2beta1/runs?page_size=10');
  assert.equal(observed.headers.host, 'backend.example.test:8123');
});

test('capture proxy blocks backend mutations while allowing MLMD read RPCs', async (t) => {
  const { buildDir } = createFixture(t);
  let backendRequests = 0;
  const handler = createRequestHandler(
    proxyConfig(buildDir, 'http://backend.example.test', {
      requestFactory: (options, callback) => {
        backendRequests++;
        return successfulRequestFactory({})(options, callback);
      },
    }),
  );

  for (const request of [
    { method: 'POST', url: '/apis/v2beta1/runs' },
    { method: 'DELETE', url: '/apis/v2beta1/pipelines/pipeline-1' },
    { method: 'POST', url: '/ml_metadata.MetadataStoreService/PutArtifacts' },
  ]) {
    const response = await invoke(handler, request);
    assert.equal(response.statusCode, 405);
    assert.match(response.body, /mutation is disabled/);
  }
  assert.equal(backendRequests, 0);

  const readResponse = await invoke(handler, {
    body: 'grpc-web request',
    method: 'POST',
    url: '/ml_metadata.MetadataStoreService/GetArtifacts',
  });
  assert.equal(readResponse.statusCode, 201);
  assert.equal(backendRequests, 1);
  assert.equal(isReadOnlyBackendRequest('GET', '/apis/v2beta1/runs'), true);
  assert.equal(isReadOnlyBackendRequest('POST', '/apis/v2beta1/runs'), false);
});

test('proxy timeout is generic and listen binds to loopback', async (t) => {
  const { buildDir } = createFixture(t);
  const requestFactory = (_options, _callback) => {
    const request = new EventEmitter();
    request.write = () => true;
    request.end = () => {};
    request.setTimeout = (_timeout, callback) => queueMicrotask(callback);
    request.destroy = (error) => queueMicrotask(() => request.emit('error', error));
    return request;
  };
  const handler = createRequestHandler(
    proxyConfig(buildDir, 'http://backend.example.test', {
      proxyTimeoutMs: 50,
      requestFactory,
    }),
  );
  const originalError = console.error;
  console.error = () => {};
  t.after(() => {
    console.error = originalError;
  });

  const response = await invoke(handler, { url: '/apis/v2beta1/healthz' });
  assert.equal(response.statusCode, 504);
  assert.equal(response.body, 'Gateway timeout');
  assert.doesNotMatch(response.body, /timed out/i);

  let listenArguments;
  const fakeServer = {
    listen(...args) {
      listenArguments = args;
      return this;
    },
  };
  assert.equal(
    listen(fakeServer, 4123, () => {}),
    fakeServer,
  );
  assert.equal(listenArguments[0], 4123);
  assert.equal(listenArguments[1], '127.0.0.1');
});
