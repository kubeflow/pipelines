const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');
const { STORAGE_READINESS_SCRIPT } = require('../storage-readiness');

function probe(t, mode) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'smoke-storage-probe-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const log = path.join(root, 'requests');
  fs.writeFileSync(
    path.join(root, 'curl'),
    `#!${process.execPath}
const fs = require('node:fs');
const args = process.argv.slice(2);
const method = args.includes('-X') ? args[args.indexOf('-X') + 1] : 'GET';
fs.appendFileSync(process.env.PROBE_LOG, method + '\\n');
const calls = fs.readFileSync(process.env.PROBE_LOG, 'utf8').trim().split('\\n');
if (process.env.PROBE_MODE === 'unwritable' || (process.env.PROBE_MODE === 'delayed' && calls.length === 1)) process.exit(22);
if (process.env.PROBE_MODE === 'delete-fails' && method === 'DELETE') process.exit(22);
if (process.env.PROBE_MODE === 'filer-unwritable' && args.at(-1).includes(':8888/')) process.exit(22);
if (method === 'PUT') fs.writeFileSync(process.env.PROBE_LOG + '.body', args[args.indexOf('--data') + 1]);
if (method === 'GET') process.stdout.write(process.env.PROBE_MODE === 'corrupt' ? 'wrong' : fs.readFileSync(process.env.PROBE_LOG + '.body'));
`,
    { mode: 0o755 },
  );
  fs.writeFileSync(path.join(root, 'sleep'), '#!/bin/sh\nexit 0\n', { mode: 0o755 });
  const result = spawnSync('sh', ['-c', STORAGE_READINESS_SCRIPT], {
    encoding: 'utf8',
    timeout: 15000,
    env: {
      ...process.env,
      PATH: `${root}:${process.env.PATH}`,
      accesskey: 'test',
      secretkey: 'test',
      PROBE_LOG: log,
      PROBE_MODE: mode,
    },
  });
  return { ...result, calls: fs.readFileSync(log, 'utf8').trim().split('\n') };
}

test('storage readiness requires successful write, exact readback, and deletion', (t) => {
  const result = probe(t, 'healthy');
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(result.calls, ['PUT', 'GET', 'DELETE', 'PUT', 'GET', 'DELETE']);
});
test('storage readiness retries transient startup failures', (t) => {
  const result = probe(t, 'delayed');
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(result.calls, ['PUT', 'PUT', 'GET', 'DELETE', 'PUT', 'GET', 'DELETE']);
});
for (const mode of ['unwritable', 'corrupt', 'delete-fails', 'filer-unwritable']) {
  test(`storage readiness bounds ${mode} failures and cleans up`, (t) => {
    const result = probe(t, mode);
    assert.equal(result.status, 1, result.stderr);
    assert.equal(
      result.calls.filter((method) => method === 'PUT').length,
      mode === 'filer-unwritable' ? 13 : 12,
    );
    assert.equal(result.calls.at(-1), 'DELETE');
    assert.match(result.stderr, /check writable volumes and free disk space/);
  });
}
