const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const {
  NODE_IMAGE,
  NODE_VERSION,
  NPM_VERSION,
  assertNodeVersion,
  assertNpmVersion,
  createRunDirectory,
  externalBuildArguments,
  externalInstallArguments,
  fetchedDependencyInputs,
  fullCaptureEnvironment,
  normalizeHttpUrl,
  parseCli,
  requestSuccessful,
  runChild,
  runComparison,
  validateExternalBuildArtifact,
} = require('../smoke-test-runner');

function comparisonOptions(overrides = {}) {
  return {
    browserOnly: false,
    comment: false,
    compareRef: 'origin/master',
    diffThreshold: 0,
    displayPrNumber: null,
    failThreshold: 0,
    prNumber: '123',
    repository: 'kubeflow/pipelines',
    trustPrCode: true,
    viewports: '1280x800',
    ...overrides,
  };
}

function detectedChanges(overrides = {}) {
  return {
    backendChanged: false,
    baseRef: 'resolved-base',
    changedFiles: ['frontend/src/App.tsx'],
    components: [],
    frontendChanged: true,
    manifestsChanged: false,
    serverChanged: false,
    ...overrides,
  };
}

function orchestrationHarness(t, changeOverrides = {}, serviceOverrides = {}) {
  const temporaryDirectory = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-orchestration-'));
  t.after(() => fs.rmSync(temporaryDirectory, { recursive: true, force: true }));

  const calls = {
    buildExternal: [],
    buildTrusted: [],
    capture: [],
    cleanup: 0,
    cleanupRegistrations: [],
    clusterSources: [],
    detect: [],
    deployments: [],
    fetch: [],
    hostServers: [],
    manifests: [],
    portChecks: [],
    proxies: [],
    publish: [],
    provenance: [],
    seed: [],
    waits: [],
    worktrees: [],
  };
  const cluster = {
    FRONTEND_SERVER_PORT: 3000,
    async buildAndDeployComponents(components, repoRoot, options) {
      calls.deployments.push({ components, options, repoRoot });
    },
    async checkPortAvailability(ports) {
      calls.portChecks.push(ports);
      return [];
    },
    async cleanup() {
      calls.cleanup += 1;
    },
    async ensureCluster(repoRoot) {
      calls.clusterSources.push(repoRoot);
    },
    async ensurePortForwarding() {},
    async reapplyManifests(repoRoot) {
      calls.manifests.push(repoRoot);
    },
    async startFrontendServer(repoRoot, options) {
      calls.hostServers.push({ options, repoRoot });
    },
    ...(serviceOverrides.clusterManager || {}),
  };
  const changes = detectedChanges(changeOverrides);
  const repoRoot = path.join(temporaryDirectory, 'repo');
  const run = {
    runDir: path.join(temporaryDirectory, 'run'),
    runId: '2026-01-02T03-04-05-000Z-42-a1b2c3d4',
  };
  const services = {
    addDetachedWorktree(target, gitRef) {
      calls.worktrees.push({ gitRef, target });
    },
    async buildExternalFrontend(target, options) {
      calls.buildExternal.push({ options, target });
    },
    async buildTrustedFrontend(target) {
      calls.buildTrusted.push(target);
    },
    async capturePair(options) {
      calls.capture.push(options);
      return {
        baseCapture: { success: true },
        comparison: { success: true },
        comparisonDir: path.join(options.screenshotsDir, 'comparison'),
        headCapture: { success: true },
      };
    },
    clusterManager: cluster,
    detectChanges(baseRef, headRef, options) {
      calls.detect.push({ baseRef, headRef, options });
      return changes;
    },
    fetchPullRequest(repository, prNumber, runId) {
      calls.fetch.push({ prNumber, repository, runId });
      return 'refs/ui-smoke-test/test-pr';
    },
    fullSha(gitRef) {
      return gitRef === 'resolved-base'
        ? 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
        : 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
    },
    gitOutput() {
      return '';
    },
    async publishReport(options, comparisonDir) {
      calls.publish.push({ comparisonDir, options });
      return true;
    },
    registerCleanup(label, action) {
      calls.cleanupRegistrations.push({ action, label });
    },
    repoRoot,
    scriptDir: '/ui-smoke-script',
    async seedData(options) {
      calls.seed.push(options);
      return { success: true };
    },
    shortSha(gitRef) {
      return `sha-${gitRef}`;
    },
    spawnManaged(command, args, options) {
      const child = { id: calls.proxies.length };
      calls.proxies.push({ args, child, command, options });
      return child;
    },
    async waitForUrl(url, child) {
      calls.waits.push({ child, url });
    },
    async verifyPullRequestHead(repository, prNumber, expectedSha) {
      calls.provenance.push({ expectedSha, prNumber, repository });
    },
    ...serviceOverrides,
    clusterManager: cluster,
  };

  return { calls, changes, repoRoot, run, services };
}

test('parseCli keeps the supported comparison workflow strict', () => {
  const options = parseCli(['--compare', 'origin/master', '--viewports', '1280x800, 390x844']);

  assert.equal(options.compareRef, 'origin/master');
  assert.equal(options.failThreshold, 0);
  assert.equal(options.viewports, '1280x800,390x844');
  assert.throws(() => parseCli(['--mode', 'static']), /Unknown option '--mode'/);
  assert.throws(() => parseCli(['--compare', 'master', '--current-only']), /exactly one workflow/);
});

test('parseCli requires an explicit target before enabling PR comments', () => {
  assert.throws(() => parseCli(['--compare', 'master', '--comment']), /--comment requires/);
  assert.throws(
    () => parseCli(['--compare', 'master', '--pr', '1; echo unsafe']),
    /positive integer/,
  );
  assert.throws(() => parseCli(['--compare', 'master', '--pr', '42']), /requires --trust-pr-code/);
  assert.equal(
    parseCli(['--compare', 'master', '--pr', '42', '--trust-pr-code']).trustPrCode,
    true,
  );
  assert.equal(
    parseCli(['--compare', 'master', '--pr-number', '42', '--comment']).displayPrNumber,
    '42',
  );
});

test('current-only preserves the full configured URL', () => {
  const value = 'https://127.0.0.1:9443/kfp/ui?tenant=one';
  const options = parseCli(['--current-only', '--use-existing', '--url', value]);

  assert.equal(options.url, value);
  assert.equal(normalizeHttpUrl(value), value);
  assert.throws(
    () => parseCli(['--current-only', '--use-existing', '--url', 'file:///tmp/index.html']),
    /must use http or https/,
  );
});

test('fetched PR installs and builds are split across constrained containers', () => {
  const installArgs = externalInstallArguments('/tmp/pr worktree');
  const args = externalBuildArguments('/tmp/pr worktree');

  assert.equal(args[0], 'run');
  assert.ok(args.includes('--read-only'));
  assert.ok(args.includes('no-new-privileges'));
  assert.ok(args.includes('--memory'));
  assert.ok(args.includes('--cpus'));
  assert.ok(args.includes('--network'));
  assert.equal(args[args.indexOf('--network') + 1], 'none');
  assert.ok(args.includes('/tmp/pr worktree:/workspace'));
  assert.equal(args.at(-4), NODE_IMAGE);
  assert.equal(args.at(-3), 'bash');
  assert.equal(args.at(-2), '-lc');
  assert.equal(
    args.at(-1),
    `test "$(corepack npm --version)" = "${NPM_VERSION}" && corepack npm ci --offline && corepack npm run build`,
  );
  assert.ok(installArgs.at(-1).includes(`corepack install --global npm@${NPM_VERSION}`));
  assert.ok(installArgs.at(-1).includes('npm ci --ignore-scripts'));
  assert.ok(installArgs.at(-1).includes('npm --prefix server ci --ignore-scripts'));
  assert.ok(installArgs.at(-1).includes('npm --prefix mock-backend ci --ignore-scripts'));
  assert.ok(!installArgs.includes('none'));
  for (const containerArgs of [installArgs, args]) {
    assert.ok(containerArgs.includes('COREPACK_ENV_FILE=0'));
    assert.ok(containerArgs.includes('COREPACK_ENABLE_PROJECT_SPEC=0'));
  }
  assert.ok(!args.join(' ').includes('GITHUB_TOKEN'));
  assert.ok(!installArgs.join(' ').includes('GITHUB_TOKEN'));
});

test('external build artifact validation rejects symlink escapes', (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-artifact-'));
  const outside = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-artifact-outside-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  t.after(() => fs.rmSync(outside, { force: true, recursive: true }));
  fs.mkdirSync(path.join(root, 'frontend'), { recursive: true });
  fs.writeFileSync(path.join(outside, 'index.html'), 'outside');
  fs.symlinkSync(outside, path.join(root, 'frontend', 'build'));

  assert.throws(() => validateExternalBuildArtifact(root), /non-symlink directory/);
});

test('full comparisons discard ambient page subsets', () => {
  const environment = fullCaptureEnvironment(
    { viewports: '1280x800' },
    { KEEP_ME: 'yes', UI_SMOKE_PAGES: 'pipelines' },
  );
  assert.equal(environment.KEEP_ME, 'yes');
  assert.equal(environment.UI_SMOKE_VIEWPORTS, '1280x800');
  assert.equal(environment.UI_SMOKE_PAGES, undefined);
});

test('runtime checks enforce the documented pinned Node and npm versions', () => {
  assert.doesNotThrow(() => assertNodeVersion(NODE_VERSION));
  assert.doesNotThrow(() => assertNpmVersion(NPM_VERSION));
  assert.throws(() => assertNodeVersion('26.0.0'), /Node\.js 24\.14\.0 is required/);
  assert.throws(() => assertNpmVersion('11.19.0'), /npm 11\.17\.0 is required/);
});

test('run directories are unique and never reuse another run output', (t) => {
  const stateDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-runner-'));
  t.after(() => fs.rmSync(stateDir, { recursive: true, force: true }));
  const now = new Date('2026-01-02T03:04:05.000Z');

  const first = createRunDirectory(now, stateDir, 'aaaaaaaa');
  const second = createRunDirectory(now, stateDir, 'bbbbbbbb');

  assert.notEqual(first.runDir, second.runDir);
  assert.ok(fs.statSync(first.runDir).isDirectory());
  assert.ok(fs.statSync(second.runDir).isDirectory());
  assert.equal(
    fs.readFileSync(path.join(stateDir, 'latest-run.txt'), 'utf8'),
    `${second.runDir}\n`,
  );
});

test('readiness accepts only successful HTTP status codes', async () => {
  const requestWithStatus = (statusCode) => (_url, callback) => {
    const request = new EventEmitter();
    request.setTimeout = () => {};
    request.destroy = () => {};
    queueMicrotask(() => callback({ resume() {}, statusCode }));
    return request;
  };

  assert.equal(await requestSuccessful('http://example.test', 2000, requestWithStatus(204)), true);
  assert.equal(await requestSuccessful('http://example.test', 2000, requestWithStatus(500)), false);
});

test('managed child commands time out and escalate termination', async () => {
  const child = new EventEmitter();
  child.exitCode = null;
  child.pid = 123;
  child.signalCode = null;
  const signals = [];
  child.kill = (signal) => {
    signals.push(signal);
    if (signal === 'SIGKILL') {
      child.signalCode = signal;
      queueMicrotask(() => child.emit('close', null, signal));
    }
    return true;
  };

  const result = await runChild('never-finishes', [], {
    killTimeout: 1,
    spawnFn: () => child,
    timeout: 1,
  });

  assert.equal(result.success, false);
  assert.equal(result.timedOut, true);
  assert.deepEqual(signals, ['SIGTERM', 'SIGKILL']);
});

test('non-browser changes require explicit browser-only scope before side effects', async (t) => {
  const { calls, run, services } = orchestrationHarness(t, {
    backendChanged: true,
    manifestsChanged: true,
    serverChanged: true,
  });

  await assert.rejects(
    runComparison(comparisonOptions(), run, services),
    /can attribute only browser\/frontend regressions.*frontend\/server, backend, manifests/,
  );

  assert.equal(calls.fetch.length, 1);
  assert.equal(calls.detect.length, 1);
  assert.deepEqual(calls.worktrees, []);
  assert.deepEqual(calls.buildTrusted, []);
});

test('fetched browser-only comparison uses the trusted base runtime for both bundles', async (t) => {
  const { calls, run, services } = orchestrationHarness(t, {
    backendChanged: true,
    components: [{ name: 'api-server' }],
    manifestsChanged: true,
    serverChanged: true,
  });
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');
  const headWorktree = path.join(run.runDir, 'worktrees', 'head');

  assert.equal(await runComparison(comparisonOptions({ browserOnly: true }), run, services), true);

  assert.deepEqual(calls.worktrees, [
    { gitRef: 'refs/ui-smoke-test/test-pr', target: headWorktree },
    { gitRef: 'resolved-base', target: baseWorktree },
  ]);
  assert.deepEqual(calls.buildTrusted, []);
  assert.deepEqual(calls.buildExternal, [
    { options: undefined, target: baseWorktree },
    { options: undefined, target: headWorktree },
  ]);
  assert.deepEqual(calls.clusterSources, [baseWorktree]);
  assert.deepEqual(calls.hostServers, [{ options: { skipBuild: false }, repoRoot: baseWorktree }]);
  assert.deepEqual(calls.manifests, []);
  assert.deepEqual(calls.deployments, []);
  assert.deepEqual(calls.seed, [
    {
      apiBase: 'http://127.0.0.1:3000',
      manifestPath: path.join(run.runDir, 'seed-manifest.json'),
    },
  ]);
  assert.equal(calls.cleanupRegistrations.length, 1);
  assert.equal(calls.cleanupRegistrations[0].label, 'stop cluster-managed local processes');
  await calls.cleanupRegistrations[0].action();
  assert.equal(calls.cleanup, 1);

  const proxyBuildRoots = calls.proxies.map(({ args }) => args[args.indexOf('--build') + 1]);
  assert.deepEqual(proxyBuildRoots, [
    path.join(baseWorktree, 'frontend', 'build'),
    path.join(headWorktree, 'frontend', 'build'),
  ]);
  assert.match(
    calls.capture[0].labels.head,
    /browser-only; ignored frontend\/server, backend, manifests/,
  );
});

test('fetched dependency-source changes are rejected before installing packages', async (t) => {
  const { calls, run, services } = orchestrationHarness(t, {
    changedFiles: ['frontend/package-lock.json'],
  });

  await assert.rejects(
    runComparison(comparisonOptions(), run, services),
    /dependency sources cannot be installed safely.*frontend\/package-lock\.json/,
  );
  assert.deepEqual(calls.worktrees, []);
  assert.deepEqual(calls.buildExternal, []);
});

test('fetched npm shrinkwrap files cannot override reviewed lockfile sources', async (t) => {
  const { calls, run, services } = orchestrationHarness(t, {
    changedFiles: ['frontend/server/npm-shrinkwrap.json'],
  });

  await assert.rejects(
    runComparison(comparisonOptions(), run, services),
    /dependency sources cannot be installed safely.*npm-shrinkwrap\.json/,
  );
  assert.deepEqual(calls.buildExternal, []);
});

test('fetched dependency and package-manager configuration uses a strict filename matrix', () => {
  assert.deepEqual(
    fetchedDependencyInputs([
      'frontend/package-lock.json',
      'frontend/npm-shrinkwrap.json',
      'frontend/server/package-lock.json',
      'frontend/server/npm-shrinkwrap.json',
      'frontend/mock-backend/package-lock.json',
      'frontend/mock-backend/npm-shrinkwrap.json',
      'frontend/server/NPM-SHRINKWRAP.JSON',
      'frontend/mock-backend/Package-Lock.json',
      '.npmrc',
      'frontend/.npmrc',
      'frontend/server/.npmrc',
      'frontend/server/.NPMRC',
      'frontend/src/.npmrc',
      'docs/.npmrc',
      '.corepack.env',
      'frontend/.corepack.env',
      'frontend/scripts/ui-smoke-test/package-lock.json',
      'test/frontend-integration-test/package-lock.json',
      'frontend/package-lock.json.bak',
      'frontend/server/package.json',
      'frontend/src/App.tsx',
    ]),
    [
      'frontend/package-lock.json',
      'frontend/npm-shrinkwrap.json',
      'frontend/server/package-lock.json',
      'frontend/server/npm-shrinkwrap.json',
      'frontend/mock-backend/package-lock.json',
      'frontend/mock-backend/npm-shrinkwrap.json',
      'frontend/server/NPM-SHRINKWRAP.JSON',
      'frontend/mock-backend/Package-Lock.json',
      '.npmrc',
      'frontend/.npmrc',
      'frontend/server/.npmrc',
      'frontend/server/.NPMRC',
      'frontend/src/.npmrc',
      'docs/.npmrc',
      '.corepack.env',
      'frontend/.corepack.env',
    ],
  );
});

test('fetched Corepack configuration is rejected before creating worktrees', async (t) => {
  const { calls, run, services } = orchestrationHarness(t, {
    changedFiles: ['frontend/.corepack.env'],
  });

  await assert.rejects(
    runComparison(comparisonOptions(), run, services),
    /dependency sources cannot be installed safely.*frontend\/\.corepack\.env/,
  );
  assert.deepEqual(calls.worktrees, []);
  assert.deepEqual(calls.buildExternal, []);
});

test('local browser-only comparison also uses the base runtime', async (t) => {
  const { calls, repoRoot, run, services } = orchestrationHarness(t, {
    backendChanged: true,
    components: [{ name: 'api-server' }],
    manifestsChanged: true,
  });
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');

  assert.equal(
    await runComparison(
      comparisonOptions({ browserOnly: true, prNumber: null, trustPrCode: false }),
      run,
      services,
    ),
    true,
  );
  assert.deepEqual(calls.clusterSources, [baseWorktree]);
  assert.deepEqual(calls.manifests, []);
  assert.deepEqual(calls.deployments, []);
  assert.deepEqual(calls.buildTrusted, [baseWorktree, repoRoot]);
  assert.deepEqual(calls.hostServers, [{ options: { skipBuild: false }, repoRoot: baseWorktree }]);
});

test('unchanged fetched server uses trusted base and still publishes a failed comparison', async (t) => {
  const { calls, run, services } = orchestrationHarness(
    t,
    {},
    {
      async capturePair(options) {
        calls.capture.push(options);
        return {
          baseCapture: { success: false },
          comparison: { success: false },
          comparisonDir: path.join(options.screenshotsDir, 'comparison'),
          headCapture: { success: true },
        };
      },
    },
  );
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');

  const success = await runComparison(comparisonOptions({ comment: true }), run, services);

  assert.equal(success, false);
  assert.deepEqual(calls.hostServers, [{ options: { skipBuild: false }, repoRoot: baseWorktree }]);
  assert.deepEqual(
    calls.buildExternal.map((call) => call.options),
    [undefined, undefined],
  );
  assert.equal(calls.capture.length, 1);
  assert.equal(calls.publish.length, 1);
  assert.equal(calls.publish[0].comparisonDir, path.join(run.runDir, 'screenshots', 'comparison'));
  assert.deepEqual(calls.provenance, [
    {
      expectedSha: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
      prNumber: '123',
      repository: 'kubeflow/pipelines',
    },
  ]);
});
