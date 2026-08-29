const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const {
  COMPONENTS,
  detectChanges,
  getChangedFiles,
  isBackendBuildInput,
  validateGitRef,
} = require('../detect-changes');

function git(cwd, ...args) {
  return execFileSync('git', args, { cwd, encoding: 'utf8' }).trim();
}

function write(repo, relativePath, contents) {
  const filePath = path.join(repo, relativePath);
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, contents);
}

function commit(repo, message) {
  git(repo, 'add', '.');
  git(repo, 'commit', '-m', message);
}

function makeRepo(t) {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'detect-changes-test-'));
  t.after(() => fs.rmSync(repo, { recursive: true, force: true }));
  git(repo, 'init', '-b', 'main');
  git(repo, 'config', 'user.name', 'UI Smoke Test');
  git(repo, 'config', 'user.email', 'ui-smoke@example.invalid');
  write(repo, 'README.md', 'initial\n');
  commit(repo, 'initial');
  return repo;
}

test('uses merge-base semantics and excludes base-only commits', (t) => {
  const repo = makeRepo(t);
  git(repo, 'checkout', '-b', 'feature');
  write(repo, 'frontend/src/feature.ts', 'feature\n');
  commit(repo, 'feature');

  git(repo, 'checkout', 'main');
  write(repo, 'backend/src/base-only.go', 'package base\n');
  commit(repo, 'base advancement');

  assert.deepEqual(getChangedFiles('main', 'feature', { cwd: repo }), ['frontend/src/feature.ts']);
});

test('optionally includes staged, unstaged, and untracked working-tree files', (t) => {
  const repo = makeRepo(t);
  const trackedPath = ' tracked\nfile.txt';
  const stagedPath = 'staged\n file.txt';
  const untrackedPath = 'untracked\nfile.txt';
  write(repo, trackedPath, 'one\n');
  write(repo, stagedPath, 'one\n');
  commit(repo, 'tracked files');
  const base = git(repo, 'rev-parse', 'HEAD');

  write(repo, trackedPath, 'two\n');
  write(repo, stagedPath, 'two\n');
  git(repo, 'add', stagedPath);
  write(repo, untrackedPath, 'new\n');

  assert.deepEqual(getChangedFiles(base, 'HEAD', { cwd: repo }), []);
  assert.deepEqual(
    getChangedFiles(base, 'HEAD', { cwd: repo, includeWorkingTree: true }),
    [stagedPath, trackedPath, untrackedPath].sort(),
  );
  assert.throws(
    () => getChangedFiles(base, 'main', { cwd: repo, includeWorkingTree: true }),
    /only be included when headRef is HEAD/,
  );
});

test('preserves unusual filenames and detects committed and dirty type changes', (t) => {
  const repo = makeRepo(t);
  const typeChangePath = 'frontend/src/type change.ts';
  const committedUnusualPath = 'frontend/src/line\nbreak .ts';
  const dirtyUnusualPath = 'frontend/src/untracked\n file.ts';
  write(repo, typeChangePath, 'export const value = 1;\n');
  commit(repo, 'add type-change fixture');
  const base = git(repo, 'rev-parse', 'HEAD');

  write(repo, committedUnusualPath, 'export const unusual = true;\n');
  fs.unlinkSync(path.join(repo, typeChangePath));
  fs.symlinkSync('../../README.md', path.join(repo, typeChangePath));
  commit(repo, 'commit unusual path and type change');
  write(repo, dirtyUnusualPath, 'export const dirty = true;\n');

  assert.deepEqual(
    getChangedFiles(base, 'HEAD', { cwd: repo }),
    [committedUnusualPath, typeChangePath].sort(),
  );
  assert.deepEqual(
    getChangedFiles(base, 'HEAD', { cwd: repo, includeWorkingTree: true }),
    [committedUnusualPath, dirtyUnusualPath, typeChangePath].sort(),
  );
});

test('validates refs without evaluating shell syntax', (t) => {
  const repo = makeRepo(t);
  assert.throws(() => validateGitRef('HEAD; echo unsafe', repo), /does not resolve/);
  assert.throws(() => validateGitRef('--help', repo), /Invalid git ref/);
});

test('any backend build input rebuilds every image and manifests are detected', (t) => {
  const repo = makeRepo(t);
  const base = git(repo, 'rev-parse', 'HEAD');
  write(repo, 'backend/src/v2/driver/driver.go', 'package driver\n');
  write(repo, 'manifests/kustomize/example.yaml', 'kind: ConfigMap\n');
  commit(repo, 'backend and manifest');

  const result = detectChanges(base, 'HEAD', { cwd: repo });
  assert.equal(result.backendChanged, true);
  assert.equal(result.manifestsChanged, true);
  assert.deepEqual(
    result.components.map((component) => component.name),
    COMPONENTS.map((c) => c.name),
  );
});

test('rename-outs retain deleted backend and frontend-server source paths', (t) => {
  const repo = makeRepo(t);
  write(repo, 'backend/src/renamed.go', 'package backend\n');
  write(repo, 'frontend/server/src/renamed.ts', 'export const serverValue = true;\n');
  commit(repo, 'add rename fixtures');
  const base = git(repo, 'rev-parse', 'HEAD');

  fs.mkdirSync(path.join(repo, 'docs'), { recursive: true });
  git(repo, 'mv', 'backend/src/renamed.go', 'docs/backend-renamed.go');
  git(repo, 'mv', 'frontend/server/src/renamed.ts', 'docs/server-renamed.ts');
  commit(repo, 'move backend and server files out of classified trees');

  const result = detectChanges(base, 'HEAD', { cwd: repo });
  assert.deepEqual(result.changedFiles, [
    'backend/src/renamed.go',
    'docs/backend-renamed.go',
    'docs/server-renamed.ts',
    'frontend/server/src/renamed.ts',
  ]);
  assert.equal(result.backendChanged, true);
  assert.equal(result.frontendChanged, true);
  assert.equal(result.serverChanged, true);
});

test('staged rename-outs retain deleted backend and frontend-server source paths', (t) => {
  const repo = makeRepo(t);
  write(repo, 'backend/src/staged.go', 'package backend\n');
  write(repo, 'frontend/server/src/staged.ts', 'export const serverValue = true;\n');
  commit(repo, 'add staged rename fixtures');
  const base = git(repo, 'rev-parse', 'HEAD');

  fs.mkdirSync(path.join(repo, 'docs'), { recursive: true });
  git(repo, 'mv', 'backend/src/staged.go', 'docs/backend-staged.go');
  git(repo, 'mv', 'frontend/server/src/staged.ts', 'docs/server-staged.ts');

  const result = detectChanges(base, 'HEAD', { cwd: repo, includeWorkingTree: true });
  assert.ok(result.changedFiles.includes('backend/src/staged.go'));
  assert.ok(result.changedFiles.includes('frontend/server/src/staged.ts'));
  assert.equal(result.backendChanged, true);
  assert.equal(result.frontendChanged, true);
  assert.equal(result.serverChanged, true);
});

test('covers external Docker inputs and all deployed local images', () => {
  for (const file of [
    'kubernetes_platform/go/kubernetes_executor.go',
    'samples/core/hello_world/component.py',
    'third_party/metadata_envoy/envoy.yaml',
    'third_party/license.txt',
  ]) {
    assert.equal(isBackendBuildInput(file), true, file);
  }
  for (const name of ['metadata-writer', 'cache-deployer', 'metadata-envoy']) {
    const component = COMPONENTS.find((candidate) => candidate.name === name);
    assert.ok(component, `${name} component is present`);
    assert.ok(component.dockerfile);
    assert.ok(component.deployment);
    assert.ok(component.container);
  }
});
