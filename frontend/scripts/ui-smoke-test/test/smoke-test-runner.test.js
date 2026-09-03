const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const { EventEmitter } = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const {
  FULL_STACK_CAPTURE_VALIDITIES,
  NODE_IMAGE,
  NODE_VERSION,
  NPM_VERSION,
  assertNodeVersion,
  assertNpmVersion,
  componentsForRevision,
  comparisonServices,
  createRunDirectory,
  executeCleanupActions,
  externalBuildArguments,
  externalInstallArguments,
  fetchedDependencyInputs,
  fullSha,
  fullCaptureEnvironment,
  loadUpgradeAdapter,
  materializeTrustedHeadSnapshot,
  normalizeHttpUrl,
  parseCli,
  persistFullStackFailure,
  readUpgradeCapabilities,
  renderRevisionManifestSources,
  requestSuccessful,
  resolveExactBaseRelease,
  resolvePublishedReleaseCommit,
  revisionUsesMetadataService,
  runChild,
  runComparison,
  seedFailureCategory,
  terminateChild,
  validateExternalBuildArtifact,
  validateFullStackBaseRelease,
} = require('../smoke-test-runner');
const { REQUIRED_OPERATIONS } = require('../upgrade-orchestrator');

function comparisonOptions(overrides = {}) {
  const options = {
    browserOnly: false,
    comment: false,
    compareRef: 'origin/master',
    diffThreshold: 0,
    displayPrNumber: null,
    failThreshold: 0,
    fullStack: false,
    headCheckout: null,
    prNumber: '123',
    repository: 'kubeflow/pipelines',
    trustBaseCode: false,
    trustLocalHead: false,
    trustPrCode: true,
    upgrade: false,
    viewports: '1280x800',
    ...overrides,
  };
  if (
    (options.fullStack || options.upgrade) &&
    !Object.prototype.hasOwnProperty.call(overrides, 'compareRef')
  ) {
    options.compareRef = '2.17.1';
  }
  return options;
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
    renderedManifests: [],
    runtimePrerequisites: [],
    seed: [],
    snapshots: [],
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
    async ensureCluster(repoRoot, options) {
      calls.clusterSources.push({ options, repoRoot });
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
    ensureComparisonRuntime() {
      calls.runtimePrerequisites.push('ensure');
    },
    fullSha(gitRef) {
      return gitRef === changes.baseRef || gitRef.startsWith('refs/tags/')
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
    resolvePublishedReleaseCommit() {
      return 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
    },
    materializeTrustedHeadSnapshot(sourceRoot, targetRoot) {
      calls.snapshots.push({ sourceRoot, targetRoot });
      fs.mkdirSync(targetRoot, { recursive: true });
      return {
        fingerprint: 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
        overlay: { trackedPatchSha256: 'sha256:dddd', untrackedEntries: [] },
        revision: {
          commit: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
          ref: 'HEAD',
          tree: 'eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
        },
        schemaVersion: 'ui-smoke-source/v1',
      };
    },
    registerCleanup(label, action) {
      calls.cleanupRegistrations.push({ action, label });
    },
    renderRevisionManifestSources(target) {
      calls.renderedManifests.push(target);
      return target.includes(`${path.sep}base`)
        ? 'apiVersion: v1\nkind: Service\nmetadata:\n  name: metadata-envoy-service\n'
        : 'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ml-pipeline\n';
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
    validateTrustedHeadCheckout() {
      return repoRoot;
    },
    ...serviceOverrides,
    clusterManager: cluster,
  };

  return { calls, changes, repoRoot, run, services };
}

test('default full-stack services expose revision metadata discovery', () => {
  const services = comparisonServices();

  assert.equal(services.revisionUsesMetadataService, revisionUsesMetadataService);
  assert.equal(services.renderRevisionManifestSources, renderRevisionManifestSources);
});

test('full-stack diagnostics use the public capture-validity vocabulary', () => {
  assert.deepEqual(FULL_STACK_CAPTURE_VALIDITIES, [
    'valid',
    'ui_rendering_failure',
    'api_incompatibility',
    'seed_failure',
    'missing_fixture',
    'selector_drift',
    'expected_product_removal',
    'infrastructure_failure',
  ]);
});

test('an unsuccessful full-stack path never reports an expected removal as complete', async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-failure-diagnostic-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  const headDirectory = path.join(root, 'screenshots', 'head');
  fs.mkdirSync(headDirectory, { recursive: true });
  fs.writeFileSync(
    path.join(headDirectory, 'manifest.json'),
    JSON.stringify({
      complete: false,
      fatalErrors: [],
      results: [
        {
          captureValidity: 'expected_product_removal',
          filename: 'executions-to-runs-1280x800.png',
          page: 'executions-to-runs',
          required: true,
          status: 'failed',
          viewport: { height: 800, width: 1280 },
        },
      ],
      summary: { complete: false },
    }),
  );

  let writtenDiagnostic;
  const state = { diagnosticsPersisted: false, phase: 'capture', stacks: [] };
  await persistFullStackFailure({
    error: new Error('capture process failed'),
    run: { runDir: root, runId: 'test-run' },
    services: {
      async writeFullStackDiagnosticArtifacts({ diagnostic }) {
        writtenDiagnostic = diagnostic;
        return {
          htmlPath: path.join(root, 'full-stack-diagnostics.html'),
          jsonPath: path.join(root, 'full-stack-diagnostics.json'),
        };
      },
    },
    state,
  });

  assert.equal(writtenDiagnostic.category, 'ui_rendering_failure');
  assert.equal(writtenDiagnostic.captureValidity, 'ui_rendering_failure');
  assert.equal(writtenDiagnostic.complete, false);
  assert.equal(writtenDiagnostic.status, 'failed');
  assert.equal(state.diagnosticsPersisted, true);
});

test('successful expected removals do not mask a later infrastructure failure', async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-failure-diagnostic-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  const headDirectory = path.join(root, 'screenshots', 'head');
  fs.mkdirSync(headDirectory, { recursive: true });
  fs.writeFileSync(
    path.join(headDirectory, 'manifest.json'),
    JSON.stringify({
      complete: true,
      fatalErrors: [],
      results: [
        {
          captureValidity: 'expected_product_removal',
          filename: 'executions-to-runs-1280x800.png',
          page: 'executions-to-runs',
          required: true,
          status: 'success',
          viewport: { height: 800, width: 1280 },
        },
      ],
      summary: { complete: true },
    }),
  );

  let writtenDiagnostic;
  await persistFullStackFailure({
    error: new Error('cluster cleanup failed'),
    run: { runDir: root, runId: 'test-run' },
    services: {
      async writeFullStackDiagnosticArtifacts({ diagnostic }) {
        writtenDiagnostic = diagnostic;
        return {
          htmlPath: path.join(root, 'full-stack-diagnostics.html'),
          jsonPath: path.join(root, 'full-stack-diagnostics.json'),
        };
      },
    },
    state: { diagnosticsPersisted: false, phase: 'cleanup', stacks: [] },
  });

  assert.equal(writtenDiagnostic.category, 'infrastructure_failure');
  assert.deepEqual(writtenDiagnostic.captures[1].incomplete, []);
  assert.equal(writtenDiagnostic.captures[1].expectedChanges.length, 1);
  assert.equal(writtenDiagnostic.complete, false);
  assert.equal(writtenDiagnostic.status, 'failed');
});

test('missing capture process artifacts are infrastructure failures', async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-failure-diagnostic-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  let writtenDiagnostic;

  await persistFullStackFailure({
    error: new Error('Capture manifest was not written after the child process exited.'),
    run: { runDir: root, runId: 'test-run' },
    services: {
      async writeFullStackDiagnosticArtifacts({ diagnostic }) {
        writtenDiagnostic = diagnostic;
        return {
          htmlPath: path.join(root, 'full-stack-diagnostics.html'),
          jsonPath: path.join(root, 'full-stack-diagnostics.json'),
        };
      },
    },
    state: { diagnosticsPersisted: false, phase: 'capture', stacks: [] },
  });

  assert.equal(writtenDiagnostic.category, 'infrastructure_failure');
  assert.equal(writtenDiagnostic.captureValidity, 'infrastructure_failure');
  assert.equal(
    writtenDiagnostic.captures.every((capture) => capture.available === false),
    true,
  );
});

test('semantic seed failure codes take precedence over generic message wording', () => {
  assert.equal(
    seedFailureCategory({
      error: 'Semantic binding discovery failed: Native task API request failed',
      failureType: 'API_INCOMPATIBILITY',
      success: false,
    }),
    'api_incompatibility',
  );
  assert.equal(
    seedFailureCategory({
      error: 'Semantic binding discovery failed: required task fixture was absent',
      failureType: 'MISSING_FIXTURE',
      success: false,
    }),
    'missing_fixture',
  );
});

test('revision manifest discovery renders only the selected platform-agnostic overlay', (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-rendered-manifests-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  const overlay = path.join(root, 'manifests', 'kustomize', 'env', 'platform-agnostic');
  fs.mkdirSync(overlay, { recursive: true });
  const calls = [];

  assert.equal(
    renderRevisionManifestSources(root, {
      execFileSync(command, args, options) {
        calls.push({ args, command, options });
        return 'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ml-pipeline\n';
      },
    }),
    'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ml-pipeline\n',
  );
  assert.deepEqual(calls, [
    {
      args: ['kustomize', overlay],
      command: 'kubectl',
      options: { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, timeout: 180000 },
    },
  ]);
});

test('revision manifest discovery rejects missing, failed, and empty renders', (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-rendered-manifests-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));

  assert.throws(
    () => renderRevisionManifestSources(root),
    /missing the platform-agnostic manifest/,
  );

  const overlay = path.join(root, 'manifests', 'kustomize', 'env', 'platform-agnostic');
  fs.mkdirSync(overlay, { recursive: true });
  assert.throws(
    () =>
      renderRevisionManifestSources(root, {
        execFileSync() {
          throw new Error('render failed');
        },
      }),
    /Failed to render revision manifests/,
  );
  assert.throws(
    () => renderRevisionManifestSources(root, { execFileSync: () => '  \n' }),
    /were empty/,
  );
});

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

test('full-stack and upgrade modes require an explicitly trusted local checkout', () => {
  assert.throws(() => parseCli(['--compare', '2.17.1', '--full-stack']), /require --head-checkout/);
  assert.throws(
    () =>
      parseCli(['--compare', '2.17.1', '--full-stack', '--head-checkout', '/tmp/reviewed-head']),
    /require --trust-local-head/,
  );
  const fullStack = parseCli([
    '--compare',
    '2.17.1',
    '--full-stack',
    '--head-checkout',
    '/tmp/reviewed-head',
    '--trust-local-head',
  ]);
  assert.equal(fullStack.fullStack, true);
  assert.equal(fullStack.headCheckout, path.resolve('/tmp/reviewed-head'));
  assert.equal(fullStack.trustLocalHead, true);
  assert.equal(fullStack.trustBaseCode, false);

  const upgrade = parseCli([
    '--compare',
    '2.17.1',
    '--upgrade',
    '--head-checkout',
    '/tmp/reviewed-head',
    '--trust-local-head',
  ]);
  assert.equal(upgrade.upgrade, true);
  assert.throws(
    () =>
      parseCli([
        '--compare',
        '2.17.1',
        '--upgrade',
        '--browser-only',
        '--head-checkout',
        '/tmp/reviewed-head',
        '--trust-local-head',
      ]),
    /mutually exclusive/,
  );

  assert.throws(
    () =>
      parseCli([
        '--compare',
        'origin/master',
        '--full-stack',
        '--head-checkout',
        '/tmp/reviewed-head',
        '--trust-local-head',
      ]),
    /requires --trust-base-code/,
  );
  const trustedBase = parseCli([
    '--compare',
    'origin/master',
    '--full-stack',
    '--head-checkout',
    '/tmp/reviewed-head',
    '--trust-local-head',
    '--trust-base-code',
  ]);
  assert.equal(trustedBase.trustBaseCode, true);
  assert.throws(
    () =>
      parseCli([
        '--compare',
        '2.17.1',
        '--upgrade',
        '--head-checkout',
        '/tmp/reviewed-head',
        '--trust-local-head',
        '--trust-base-code',
      ]),
    /--trust-base-code is only valid with --full-stack/,
  );
});

test('fetched PRs cannot opt into full runtime execution', () => {
  assert.throws(
    () =>
      parseCli([
        '--compare',
        '2.17.1',
        '--pr',
        '13986',
        '--trust-pr-code',
        '--full-stack',
        '--head-checkout',
        '/tmp/reviewed-head',
        '--trust-local-head',
      ]),
    /Fetched PR runtime code cannot be executed/,
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

test('revision component discovery excludes workloads absent from selected manifests', (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-components-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  fs.mkdirSync(path.join(root, 'manifests'), { recursive: true });
  fs.mkdirSync(path.join(root, 'backend'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'manifests', 'stack.yaml'),
    'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: removed-deployment\n',
  );
  fs.writeFileSync(path.join(root, 'backend', 'present.Dockerfile'), 'FROM scratch\n');
  fs.writeFileSync(path.join(root, 'backend', 'removed.Dockerfile'), 'FROM scratch\n');

  assert.deepEqual(
    componentsForRevision(
      root,
      [
        {
          deployment: 'present-deployment',
          dockerfile: 'backend/present.Dockerfile',
          name: 'present',
        },
        {
          deployment: 'removed-deployment',
          dockerfile: 'backend/removed.Dockerfile',
          name: 'removed',
        },
        { deployment: null, dockerfile: 'backend/present.Dockerfile', name: 'runtime' },
      ],
      'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: present-deployment\n',
    ).map(({ name }) => name),
    ['present', 'runtime'],
  );
});

test('upgrade capability descriptors and adapters stay inside the reviewed checkout', (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-upgrade-adapter-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  const adapterPath = path.join(root, 'adapter.js');
  fs.writeFileSync(adapterPath, 'module.exports = { createOperations() { return {}; } };\n');
  fs.writeFileSync(
    path.join(root, '.ui-smoke-upgrade.json'),
    JSON.stringify({
      adapter: 'adapter.js',
      capabilities: {
        migration: { available: true, version: 'v1' },
        startupGate: { available: true, migrationVersion: 'v1' },
      },
      schemaVersion: 'ui-smoke-upgrade/v1',
    }),
  );

  const descriptor = readUpgradeCapabilities(root);
  assert.equal(descriptor.adapter, 'adapter.js');
  assert.equal(typeof loadUpgradeAdapter(root, descriptor).createOperations, 'function');
  assert.throws(
    () => loadUpgradeAdapter(root, { adapter: '../outside.js' }),
    /escapes the reviewed checkout/,
  );

  const outsideDescriptor = path.join(path.dirname(root), `${path.basename(root)}-outside.json`);
  fs.writeFileSync(outsideDescriptor, JSON.stringify({ schemaVersion: 'ui-smoke-upgrade/v1' }));
  t.after(() => fs.rmSync(outsideDescriptor, { force: true }));
  fs.rmSync(path.join(root, '.ui-smoke-upgrade.json'));
  fs.symlinkSync(outsideDescriptor, path.join(root, '.ui-smoke-upgrade.json'));
  assert.throws(
    () => readUpgradeCapabilities(root),
    /must be a regular file inside the reviewed checkout/,
  );
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

test('release base validation accepts only exact published-release version syntax', () => {
  assert.equal(validateFullStackBaseRelease('2.17.1'), '2.17.1');
  assert.throws(
    () => validateFullStackBaseRelease('origin/master'),
    /require an exact version.*origin\/master/,
  );
});

test('full commit resolution uses an option-safe verified Git ref', () => {
  const calls = [];
  const commit = 'a'.repeat(40);

  assert.equal(
    fullSha('origin/master', '/repo', {
      git(args, cwd) {
        calls.push({ args, cwd });
        return commit;
      },
    }),
    commit,
  );
  assert.deepEqual(calls, [
    {
      args: ['rev-parse', '--verify', '--end-of-options', 'origin/master^{commit}'],
      cwd: '/repo',
    },
  ]);
  assert.throws(
    () =>
      fullSha('--help', '/repo', {
        git() {
          throw new Error('must not execute');
        },
      }),
    /invalid Git ref/,
  );
});

function runGit(args, cwd, options = {}) {
  const encoding = options.encoding === 'buffer' ? 'buffer' : 'utf8';
  const output = execFileSync('git', args, { cwd, encoding });
  if (Buffer.isBuffer(output)) return output;
  return options.trim === false ? output : output.trim();
}

function initializeSnapshotRepository(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-source-snapshot-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  const sourceRoot = path.join(root, 'source');
  fs.mkdirSync(sourceRoot);
  runGit(['init'], sourceRoot);
  runGit(['config', 'user.name', 'UI Smoke Test'], sourceRoot);
  runGit(['config', 'user.email', 'ui-smoke@example.test'], sourceRoot);
  fs.writeFileSync(path.join(sourceRoot, '.gitignore'), 'ignored.txt\n');
  for (const name of ['both.txt', 'staged.txt', 'unstaged.txt']) {
    fs.writeFileSync(path.join(sourceRoot, name), `base ${name}\n`);
  }
  runGit(['add', '--all'], sourceRoot);
  runGit(['commit', '-m', 'base'], sourceRoot);
  return { root, sourceRoot };
}

test('trusted local head is materialized as an immutable Git snapshot with every non-ignored input', (t) => {
  const { root, sourceRoot } = initializeSnapshotRepository(t);
  fs.writeFileSync(path.join(sourceRoot, 'staged.txt'), 'staged value\n');
  runGit(['add', 'staged.txt'], sourceRoot);
  fs.writeFileSync(path.join(sourceRoot, 'unstaged.txt'), 'unstaged value\n');
  fs.writeFileSync(path.join(sourceRoot, 'both.txt'), 'staged intermediate\n');
  runGit(['add', 'both.txt'], sourceRoot);
  fs.writeFileSync(path.join(sourceRoot, 'both.txt'), 'final working value\n');
  fs.writeFileSync(path.join(sourceRoot, 'untracked.txt'), 'untracked value\n');
  fs.symlinkSync('untracked.txt', path.join(sourceRoot, 'untracked-link'));
  fs.writeFileSync(path.join(sourceRoot, 'ignored.txt'), 'must not execute\n');

  const addWorktree = (target, gitRef, repositoryRoot) => {
    runGit(['worktree', 'add', '--detach', target, gitRef], repositoryRoot);
  };
  const firstTarget = path.join(root, 'snapshot-one');
  const secondTarget = path.join(root, 'snapshot-two');
  const first = materializeTrustedHeadSnapshot(sourceRoot, firstTarget, {
    addWorktree,
    git: runGit,
  });
  const second = materializeTrustedHeadSnapshot(sourceRoot, secondTarget, {
    addWorktree,
    git: runGit,
  });

  assert.equal(first.fingerprint, second.fingerprint);
  assert.equal(first.revision.tree, second.revision.tree);
  assert.equal(first.schemaVersion, 'ui-smoke-source/v1');
  assert.deepEqual(
    first.overlay.untrackedEntries.map(({ path: relativePath, type }) => ({
      path: relativePath,
      type,
    })),
    [
      { path: 'untracked-link', type: 'symlink' },
      { path: 'untracked.txt', type: 'file' },
    ],
  );
  assert.equal(runGit(['rev-parse', '--is-inside-work-tree'], firstTarget), 'true');
  assert.equal(fs.readFileSync(path.join(firstTarget, 'staged.txt'), 'utf8'), 'staged value\n');
  assert.equal(fs.readFileSync(path.join(firstTarget, 'unstaged.txt'), 'utf8'), 'unstaged value\n');
  assert.equal(
    fs.readFileSync(path.join(firstTarget, 'both.txt'), 'utf8'),
    'final working value\n',
  );
  assert.equal(
    fs.readFileSync(path.join(firstTarget, 'untracked.txt'), 'utf8'),
    'untracked value\n',
  );
  assert.equal(fs.readlinkSync(path.join(firstTarget, 'untracked-link')), 'untracked.txt');
  assert.equal(fs.existsSync(path.join(firstTarget, 'ignored.txt')), false);

  fs.writeFileSync(path.join(sourceRoot, 'staged.txt'), 'mutated after snapshot\n');
  fs.writeFileSync(path.join(sourceRoot, 'untracked.txt'), 'mutated after snapshot\n');
  fs.writeFileSync(path.join(sourceRoot, 'late-file.txt'), 'late mutation\n');
  assert.equal(fs.readFileSync(path.join(firstTarget, 'staged.txt'), 'utf8'), 'staged value\n');
  assert.equal(
    fs.readFileSync(path.join(firstTarget, 'untracked.txt'), 'utf8'),
    'untracked value\n',
  );
  assert.equal(fs.existsSync(path.join(firstTarget, 'late-file.txt')), false);
});

test('trusted source snapshot preserves non-UTF-8 bytes in tracked patches', (t) => {
  const { root, sourceRoot } = initializeSnapshotRepository(t);
  const sourcePath = path.join(sourceRoot, 'raw-bytes.bin');
  fs.writeFileSync(sourcePath, Buffer.alloc(0));
  runGit(['add', 'raw-bytes.bin'], sourceRoot);
  runGit(['commit', '-m', 'add empty byte fixture'], sourceRoot);
  const expected = Buffer.from([0x61, 0xff, 0x62, 0x0a]);
  fs.writeFileSync(sourcePath, expected);

  const target = path.join(root, 'snapshot-raw-bytes');
  materializeTrustedHeadSnapshot(sourceRoot, target, {
    addWorktree(snapshotTarget, gitRef, repositoryRoot) {
      runGit(['worktree', 'add', '--detach', snapshotTarget, gitRef], repositoryRoot);
    },
  });

  assert.deepEqual(fs.readFileSync(path.join(target, 'raw-bytes.bin')), expected);
});

test('trusted source snapshot rejects non-UTF-8 untracked paths before creating a worktree', (t) => {
  const { root, sourceRoot } = initializeSnapshotRepository(t);
  let worktreeCreated = false;
  const invalidPathGit = (args, cwd, options) => {
    if (args[0] === 'ls-files') return Buffer.from([0xff, 0x00]);
    return runGit(args, cwd, options);
  };

  assert.throws(
    () =>
      materializeTrustedHeadSnapshot(sourceRoot, path.join(root, 'snapshot-invalid-path'), {
        addWorktree() {
          worktreeCreated = true;
        },
        git: invalidPathGit,
      }),
    /non-UTF-8 Git path/,
  );
  assert.equal(worktreeCreated, false);
});

test('trusted source snapshot fails closed when the selected checkout mutates during capture', (t) => {
  const { root, sourceRoot } = initializeSnapshotRepository(t);
  fs.writeFileSync(path.join(sourceRoot, 'unstaged.txt'), 'first value\n');
  let diffCalls = 0;
  let worktreeCreated = false;
  const mutatingGit = (args, cwd, options) => {
    const output = runGit(args, cwd, options);
    if (args[0] === 'diff' && ++diffCalls === 1) {
      fs.writeFileSync(path.join(sourceRoot, 'unstaged.txt'), 'second value\n');
    }
    return output;
  };

  assert.throws(
    () =>
      materializeTrustedHeadSnapshot(sourceRoot, path.join(root, 'snapshot'), {
        addWorktree() {
          worktreeCreated = true;
        },
        git: mutatingGit,
      }),
    /changed while its source snapshot was being captured/,
  );
  assert.equal(worktreeCreated, false);
});

test('trusted source snapshot rejects symlinks that escape the reviewed tree', (t) => {
  const { root, sourceRoot } = initializeSnapshotRepository(t);
  fs.writeFileSync(path.join(root, 'outside.txt'), 'outside source\n');
  fs.symlinkSync('../outside.txt', path.join(sourceRoot, 'escape'));

  assert.throws(
    () =>
      materializeTrustedHeadSnapshot(sourceRoot, path.join(root, 'snapshot'), {
        addWorktree(target, gitRef, repositoryRoot) {
          runGit(['worktree', 'add', '--detach', target, gitRef], repositoryRoot);
        },
        git: runGit,
      }),
    /snapshot symlink escapes its worktree: escape/,
  );
});

test('exact release resolution requires an unambiguous fully qualified tag', (t) => {
  const { sourceRoot } = initializeSnapshotRepository(t);
  runGit(['tag', '2.17.1'], sourceRoot);
  const expectedCommit = runGit(['rev-parse', 'HEAD'], sourceRoot);

  assert.deepEqual(
    resolveExactBaseRelease('2.17.1', {
      cwd: sourceRoot,
      git: runGit,
      resolvePublishedCommit: () => expectedCommit,
    }),
    {
      commit: expectedCommit,
      tagRef: 'refs/tags/2.17.1',
      version: '2.17.1',
    },
  );
  assert.throws(
    () => resolveExactBaseRelease('2.17.2', { cwd: sourceRoot, git: runGit }),
    /Exact release tag refs\/tags\/2\.17\.2 is missing/,
  );
  runGit(['branch', '2.17.1'], sourceRoot);
  assert.throws(
    () => resolveExactBaseRelease('2.17.1', { cwd: sourceRoot, git: runGit }),
    /ambiguous release base.*refs\/heads\/2\.17\.1/,
  );
});

test('published release resolution peels annotated tags and accepts lightweight tags', () => {
  const commit = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
  const tagObject = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
  const tagRef = 'refs/tags/2.17.1';
  for (const [name, advertisement] of [
    ['lightweight', `${commit}\t${tagRef}\n`],
    ['annotated', `${tagObject}\t${tagRef}\n${commit}\t${tagRef}^{}\n`],
  ]) {
    assert.equal(
      resolvePublishedReleaseCommit('2.17.1', {
        git(args, _cwd, options) {
          assert.equal(args[0], 'ls-remote', name);
          assert.equal(args.includes(tagRef), true, name);
          assert.equal(args.includes(`${tagRef}^{}`), true, name);
          assert.equal(options.env.GIT_TERMINAL_PROMPT, '0', name);
          return advertisement;
        },
      }),
      commit,
      name,
    );
  }
});

test('exact release resolution rejects a counterfeit local release tag', (t) => {
  const { sourceRoot } = initializeSnapshotRepository(t);
  runGit(['tag', '2.17.1'], sourceRoot);
  const localCommit = runGit(['rev-parse', 'HEAD'], sourceRoot);
  const publishedCommit = 'ffffffffffffffffffffffffffffffffffffffff';

  assert.throws(
    () =>
      resolveExactBaseRelease('2.17.1', {
        cwd: sourceRoot,
        git: runGit,
        resolvePublishedCommit: () => publishedCommit,
      }),
    new RegExp(
      `Local release tag refs/tags/2\\.17\\.1 resolves to ${localCommit}, but the published kubeflow/pipelines tag resolves to ${publishedCommit}`,
    ),
  );
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

test('runner child termination waits for close after escalation', async () => {
  const child = new EventEmitter();
  child.exitCode = null;
  child.pid = 124;
  child.signalCode = null;
  let closed = false;
  child.kill = (signal) => {
    if (signal === 'SIGKILL') {
      setTimeout(() => {
        closed = true;
        child.signalCode = signal;
        child.emit('close', null, signal);
      }, 10);
    }
    return true;
  };

  await terminateChild(child, 1);
  assert.equal(closed, true);
});

test('timed-out runner child reports a bounded termination failure when signals are ignored', async () => {
  const child = new EventEmitter();
  child.exitCode = null;
  child.pid = 125;
  child.signalCode = null;
  const signals = [];
  child.kill = (signal) => {
    signals.push(signal);
    return true;
  };

  const result = await runChild('ignores-signals', [], {
    killTimeout: 1,
    spawnFn: () => child,
    timeout: 1,
  });
  assert.equal(result.success, false);
  assert.equal(result.timedOut, true);
  assert.equal(result.terminationFailed, true);
  assert.match(result.error.message, /timed out and could not be terminated/);
  assert.deepEqual(signals, ['SIGTERM', 'SIGKILL']);

  child.signalCode = 'SIGKILL';
  child.emit('close', null, 'SIGKILL');
});

test('cleanup actions run in reverse order and propagate every failure', async () => {
  const calls = [];
  await assert.rejects(
    executeCleanupActions([
      {
        label: 'first',
        async action() {
          calls.push('first');
          throw new Error('first failed');
        },
      },
      {
        label: 'second',
        async action() {
          calls.push('second');
          throw new Error('second failed');
        },
      },
    ]),
    (error) => {
      assert.ok(error instanceof AggregateError);
      assert.equal(error.errors.length, 2);
      assert.match(error.errors[0].message, /second.*second failed/);
      assert.match(error.errors[1].message, /first.*first failed/);
      return true;
    },
  );
  assert.deepEqual(calls, ['second', 'first']);
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

test('matched-stack modes reject a missing exact release tag before snapshot or runtime setup', async (t) => {
  for (const mode of ['fullStack', 'upgrade']) {
    await t.test(mode, async (t) => {
      const { calls, run, services } = orchestrationHarness(
        t,
        {},
        {
          resolveExactBaseRelease() {
            throw new Error('Exact release tag refs/tags/2.17.1 is missing.');
          },
        },
      );
      await assert.rejects(
        runComparison(
          comparisonOptions({
            [mode]: true,
            headCheckout: '/reviewed/head',
            prNumber: null,
            trustLocalHead: true,
            trustPrCode: false,
          }),
          run,
          services,
        ),
        /Exact release tag refs\/tags\/2\.17\.1 is missing/,
      );
      assert.deepEqual(calls.snapshots, []);
      assert.deepEqual(calls.runtimePrerequisites, []);
      assert.deepEqual(calls.buildTrusted, []);
    });
  }
});

test('matched-stack change detection stays pinned when the verified release tag moves', async (t) => {
  const verifiedCommit = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
  const movedCommit = 'dddddddddddddddddddddddddddddddddddddddd';
  let currentTagCommit = verifiedCommit;
  const { calls, run, services } = orchestrationHarness(
    t,
    {},
    {
      resolveExactBaseRelease() {
        const release = {
          commit: currentTagCommit,
          tagRef: 'refs/tags/2.17.1',
          version: '2.17.1',
        };
        currentTagCommit = movedCommit;
        return release;
      },
    },
  );

  assert.equal(
    await runComparison(
      comparisonOptions({
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
        upgrade: true,
      }),
      run,
      services,
    ),
    false,
  );
  assert.equal(currentTagCommit, movedCommit);
  assert.equal(calls.detect[0].baseRef, verifiedCommit);
  assert.equal(calls.snapshots.length, 1);
  assert.deepEqual(calls.runtimePrerequisites, []);
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
  assert.deepEqual(calls.clusterSources, [
    {
      options: { fixtureRequirements: { argoRetryPolicy: 'OnFailure' } },
      repoRoot: baseWorktree,
    },
  ]);
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
  assert.deepEqual(calls.clusterSources, [
    {
      options: { fixtureRequirements: { argoRetryPolicy: 'OnFailure' } },
      repoRoot: baseWorktree,
    },
  ]);
  assert.deepEqual(calls.manifests, []);
  assert.deepEqual(calls.deployments, []);
  assert.deepEqual(calls.buildTrusted, [baseWorktree, repoRoot]);
  assert.deepEqual(calls.hostServers, [{ options: { skipBuild: false }, repoRoot: baseWorktree }]);
});

test('trusted full-stack comparison isolates runtimes, state, and seed manifests', async (t) => {
  const stackOperations = [];
  let baseDeploymentComplete = false;
  const { calls, repoRoot, run, services } = orchestrationHarness(
    t,
    {
      baseRef: '2.17.1',
      backendChanged: true,
      manifestsChanged: true,
      serverChanged: true,
    },
    {
      clusterManager: {
        createKindStack(configuration) {
          const role = configuration.role;
          const deployedUiUrl = role === 'base' ? 'http://127.0.0.1:3101' : 'http://127.0.0.1:3201';
          const record = (operation, detail = {}) => {
            stackOperations.push({ operation, role, ...detail });
          };
          return {
            clusterName: configuration.clusterName,
            deployedUiUrl,
            role,
            async cleanup() {
              record('cleanup');
            },
            async buildComponentImages(components, target, options) {
              record('buildComponentImages', { components, options, target });
              return {
                deployments: [
                  {
                    container: 'ml-pipeline-api-server',
                    deployment: 'ml-pipeline',
                    image: 'kfp-ui-smoke/apiserver:test',
                  },
                ],
                images: { apiserver: 'kfp-ui-smoke/apiserver:test' },
                runtimeEnvironment: {},
              };
            },
            async createCluster() {
              record('createCluster');
            },
            async deployRevision(target, options) {
              record('deployRevision:start', { options, target });
              if (role === 'base') {
                await Promise.resolve();
                baseDeploymentComplete = true;
              } else {
                assert.equal(
                  baseDeploymentComplete,
                  true,
                  'the release stack must deploy before the reviewed head build begins',
                );
              }
              record('deployRevision:complete', { options, target });
            },
            async destroyCluster() {
              record('destroyCluster');
              return true;
            },
            async ensureDeployedUiPortForwarding() {
              const child = { role };
              record('ensureDeployedUiPortForwarding', { child });
              return [child];
            },
            getClusterPlatform() {
              record('getClusterPlatform');
              return 'linux/amd64';
            },
            getDockerPlatform() {
              record('getDockerPlatform');
              return 'linux/amd64';
            },
            loadImageOverrides(imageOverrides, platform, options) {
              record('loadImageOverrides', { imageOverrides, options, platform });
            },
            preflightReleaseImages(target, options) {
              record('preflightReleaseImages', { options, target });
              return { images: ['release-image'], platform: options.platform };
            },
            preflightSeedRuntimeImage(options) {
              record('preflightSeedRuntimeImage', { options });
              return { image: 'seed-image', platform: options.platform };
            },
            preflightThirdPartyImages(target, options) {
              record('preflightThirdPartyImages', { options, target });
              return { images: ['third-party-image'], platform: options.platform };
            },
          };
        },
      },
      combineSemanticManifests(manifests, options) {
        return { manifests, options, schemaVersion: 'ui-smoke-semantic/v3' };
      },
      componentsForRevision() {
        return [{ name: 'apiserver' }];
      },
      revisionUsesMetadataService(target) {
        return target.includes('base');
      },
      async seedData(options) {
        calls.seed.push(options);
        const revisionFlavor = options.apiBase.endsWith(':3101')
          ? 'legacy-mlmd'
          : 'native-task-artifact';
        fs.mkdirSync(path.dirname(options.manifestPath), { recursive: true });
        fs.writeFileSync(
          options.manifestPath,
          JSON.stringify({
            apiBase: options.apiBase,
            defaults: {},
            resources: {},
            semantic: {
              logical: { runs: {} },
              revisionFlavor,
              validation: { valid: true },
            },
          }),
        );
        return { success: true };
      },
    },
  );
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');
  const headWorktree = path.join(run.runDir, 'worktrees', 'head');

  assert.equal(
    await runComparison(
      comparisonOptions({
        displayPrNumber: '13986',
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    true,
  );

  assert.deepEqual(calls.worktrees, [
    { gitRef: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', target: baseWorktree },
  ]);
  assert.deepEqual(calls.snapshots, [{ sourceRoot: repoRoot, targetRoot: headWorktree }]);
  assert.deepEqual(calls.renderedManifests, [baseWorktree, headWorktree]);
  assert.deepEqual(calls.buildTrusted, []);
  assert.equal(stackOperations.filter(({ operation }) => operation === 'createCluster').length, 2);
  const seedPreflightIndex = stackOperations.findIndex(
    ({ operation, role }) => operation === 'preflightSeedRuntimeImage' && role === 'base',
  );
  const firstClusterIndex = stackOperations.findIndex(
    ({ operation }) => operation === 'createCluster',
  );
  assert.ok(seedPreflightIndex >= 0 && seedPreflightIndex < firstClusterIndex);
  const releasePreflight = stackOperations.find(
    ({ operation, role }) => operation === 'preflightReleaseImages' && role === 'base',
  );
  assert.deepEqual(releasePreflight, {
    operation: 'preflightReleaseImages',
    options: { expectedRelease: '2.17.1', platform: 'linux/amd64' },
    role: 'base',
    target: baseWorktree,
  });
  assert.ok(
    stackOperations.indexOf(releasePreflight) <
      stackOperations.findIndex(({ operation }) => operation === 'createCluster'),
  );
  assert.deepEqual(
    stackOperations
      .filter(({ operation }) => operation === 'deployRevision:start')
      .map(({ role, target }) => ({ role, target })),
    [
      { role: 'base', target: baseWorktree },
      { role: 'head', target: headWorktree },
    ],
  );
  assert.equal(
    stackOperations.find(
      ({ operation, role }) => operation === 'deployRevision:start' && role === 'base',
    ).options.expectedRelease,
    '2.17.1',
  );
  assert.equal(
    stackOperations.find(
      ({ operation, role }) => operation === 'deployRevision:start' && role === 'base',
    ).options.platform,
    'linux/amd64',
  );
  assert.equal(
    stackOperations.find(
      ({ operation, role }) => operation === 'deployRevision:start' && role === 'head',
    ).options.requireLocalFirstParty,
    true,
  );
  for (const deployment of stackOperations.filter(
    ({ operation }) => operation === 'deployRevision:start',
  )) {
    assert.deepEqual(deployment.options.fixtureRequirements, {
      argoRetryPolicy: 'OnFailure',
    });
  }
  assert.equal(calls.seed.length, 2);
  assert.notEqual(calls.seed[0].apiBase, calls.seed[1].apiBase);
  assert.notEqual(calls.seed[0].manifestPath, calls.seed[1].manifestPath);
  assert.deepEqual(calls.proxies, []);
  assert.deepEqual(
    calls.waits.map(({ child, url }) => ({ child, url })),
    [
      { child: { role: 'base' }, url: 'http://127.0.0.1:3101' },
      { child: { role: 'head' }, url: 'http://127.0.0.1:3201' },
    ],
  );
  assert.equal(calls.capture[0].baseUrl, 'http://127.0.0.1:3101');
  assert.equal(calls.capture[0].headUrl, 'http://127.0.0.1:3201');
  assert.equal(calls.capture[0].baseSeedManifestPath, calls.seed[0].manifestPath);
  assert.equal(calls.capture[0].headSeedManifestPath, calls.seed[1].manifestPath);
  assert.equal(
    calls.capture[0].semanticManifestPath,
    path.join(run.runDir, 'semantic-fixtures.json'),
  );
  assert.equal(
    calls.capture[0].sourceProvenancePath,
    path.join(run.runDir, 'source-provenance.json'),
  );
  assert.match(calls.capture[0].labels.head, /source cccccccccccc/);
  assert.ok(fs.existsSync(path.join(run.runDir, 'source-provenance.json')));
  assert.ok(fs.existsSync(path.join(run.runDir, 'semantic-fixtures.json')));
  const semanticManifest = JSON.parse(
    fs.readFileSync(path.join(run.runDir, 'semantic-fixtures.json'), 'utf8'),
  );
  assert.deepEqual(semanticManifest.options.revisions.base, {
    commit: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    ref: 'refs/tags/2.17.1',
  });
  assert.deepEqual(semanticManifest.options.revisions.head, {
    commit: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    ref: 'HEAD',
    sourceFingerprint: 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
    tree: 'eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
  });
  assert.equal(calls.cleanupRegistrations.length, 4);
  assert.equal(stackOperations.filter(({ operation }) => operation === 'cleanup').length, 2);
  assert.equal(stackOperations.filter(({ operation }) => operation === 'destroyCluster').length, 2);
});

test('untrusted arbitrary full-stack bases fail before source or runtime side effects', async (t) => {
  const { calls, run, services } = orchestrationHarness(t);
  services.fullSha = () => {
    throw new Error('Git resolution must not run before the base trust gate.');
  };

  await assert.rejects(
    runComparison(
      comparisonOptions({
        compareRef: 'origin/master',
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustBaseCode: false,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    /requires --trust-base-code/,
  );

  assert.deepEqual(calls.detect, []);
  assert.deepEqual(calls.runtimePrerequisites, []);
  assert.deepEqual(calls.snapshots, []);
  assert.deepEqual(calls.worktrees, []);
  assert.deepEqual(calls.portChecks, []);
});

test('trusted arbitrary full-stack bases are SHA-pinned and built as isolated local stacks', async (t) => {
  const baseSha = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
  const headSha = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
  const stackOperations = [];
  const { calls, repoRoot, run, services } = orchestrationHarness(
    t,
    {
      backendChanged: true,
      baseRef: baseSha,
      manifestsChanged: true,
      serverChanged: true,
    },
    {
      clusterManager: {
        createKindStack(configuration) {
          const { role } = configuration;
          const record = (operation, detail = {}) =>
            stackOperations.push({ operation, role, ...detail });
          return {
            clusterName: configuration.clusterName,
            deployedUiUrl: `http://127.0.0.1:${configuration.ports.frontendServer}`,
            role,
            async buildComponentImages(components, target, options) {
              record('buildComponentImages', { components, options, target });
              return {
                deployments: [
                  {
                    container: 'ml-pipeline-api-server',
                    deployment: 'ml-pipeline',
                    image: `kfp-ui-smoke/${role}:test`,
                  },
                ],
                images: { apiserver: `kfp-ui-smoke/${role}:test` },
                runtimeEnvironment: {},
              };
            },
            async cleanup() {
              record('cleanup');
            },
            async createCluster() {
              record('createCluster');
            },
            async deployRevision(target, options) {
              record('deployRevision', { options, target });
            },
            async destroyCluster() {
              record('destroyCluster');
              return true;
            },
            async ensureDeployedUiPortForwarding() {
              return [{ role }];
            },
            getClusterPlatform() {
              return 'linux/amd64';
            },
            getDockerPlatform() {
              return 'linux/amd64';
            },
            loadImageOverrides(imageOverrides, platform, options) {
              record('loadImageOverrides', { imageOverrides, options, platform });
            },
            preflightReleaseImages() {
              throw new Error('arbitrary bases must not use published release images');
            },
            preflightSeedRuntimeImage(options) {
              return { image: 'seed-image', platform: options.platform };
            },
            preflightThirdPartyImages(target, options) {
              record('preflightThirdPartyImages', { options, target });
              return { images: ['dependency'], platform: options.platform };
            },
          };
        },
      },
      combineSemanticManifests(manifests, options) {
        return { manifests, options, schemaVersion: 'ui-smoke-semantic/v3' };
      },
      componentsForRevision(target) {
        return [{ name: target.includes('base') ? 'base-apiserver' : 'head-apiserver' }];
      },
      fullSha(gitRef) {
        if (gitRef === 'origin/master' || gitRef === baseSha) return baseSha;
        return headSha;
      },
      resolveExactBaseRelease() {
        throw new Error('non-release bases must not use release resolution');
      },
      revisionUsesMetadataService(target) {
        return target.includes('base');
      },
      async seedData(options) {
        calls.seed.push(options);
        const revisionFlavor = options.apiBase.endsWith(':3101')
          ? 'legacy-mlmd'
          : 'native-task-artifact';
        fs.mkdirSync(path.dirname(options.manifestPath), { recursive: true });
        fs.writeFileSync(
          options.manifestPath,
          JSON.stringify({
            defaults: {},
            resources: {},
            semantic: {
              logical: { runs: {} },
              revisionFlavor,
              validation: { valid: true },
            },
          }),
        );
        return { success: true };
      },
    },
  );
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');
  const headWorktree = path.join(run.runDir, 'worktrees', 'head');

  assert.equal(
    await runComparison(
      comparisonOptions({
        compareRef: 'origin/master',
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustBaseCode: true,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    true,
  );

  assert.equal(calls.detect[0].baseRef, baseSha);
  assert.deepEqual(calls.worktrees, [{ gitRef: baseSha, target: baseWorktree }]);
  assert.deepEqual(calls.snapshots, [{ sourceRoot: repoRoot, targetRoot: headWorktree }]);
  assert.deepEqual(
    stackOperations
      .filter(({ operation }) => operation === 'preflightThirdPartyImages')
      .map(({ role, target }) => ({ role, target })),
    [
      { role: 'base', target: baseWorktree },
      { role: 'head', target: headWorktree },
    ],
  );
  assert.deepEqual(
    stackOperations
      .filter(({ operation }) => operation === 'buildComponentImages')
      .map(({ role, target }) => ({ role, target })),
    [
      { role: 'base', target: baseWorktree },
      { role: 'head', target: headWorktree },
    ],
  );
  const firstCluster = stackOperations.findIndex(({ operation }) => operation === 'createCluster');
  const lastBuild = stackOperations.reduce(
    (index, entry, current) => (entry.operation === 'buildComponentImages' ? current : index),
    -1,
  );
  assert.ok(lastBuild >= 0 && lastBuild < firstCluster);
  assert.deepEqual(
    stackOperations
      .filter(({ operation }) => operation === 'deployRevision')
      .map(({ options, role, target }) => ({
        expectedRelease: options.expectedRelease,
        fixtureRequirements: options.fixtureRequirements,
        requireLocalFirstParty: options.requireLocalFirstParty,
        role,
        target,
      })),
    [
      {
        expectedRelease: undefined,
        fixtureRequirements: { argoRetryPolicy: 'OnFailure' },
        requireLocalFirstParty: true,
        role: 'base',
        target: baseWorktree,
      },
      {
        expectedRelease: undefined,
        fixtureRequirements: { argoRetryPolicy: 'OnFailure' },
        requireLocalFirstParty: true,
        role: 'head',
        target: headWorktree,
      },
    ],
  );
  assert.deepEqual(
    stackOperations
      .filter(({ operation }) => operation === 'loadImageOverrides')
      .map(({ options, role }) => ({ options, role })),
    [
      { options: { removeSourceAfterLoad: true }, role: 'base' },
      { options: { removeSourceAfterLoad: true }, role: 'head' },
    ],
  );
  const semanticManifest = JSON.parse(
    fs.readFileSync(path.join(run.runDir, 'semantic-fixtures.json'), 'utf8'),
  );
  assert.deepEqual(semanticManifest.options.revisions.base, {
    commit: baseSha,
    ref: 'origin/master',
  });
});

test('full-stack seed failures persist categorized JSON, HTML, and stack diagnostics', async (t) => {
  const collected = [];
  const { run, services } = orchestrationHarness(
    t,
    { backendChanged: true, baseRef: '2.17.1' },
    {
      clusterManager: {
        createKindStack(configuration) {
          return {
            clusterName: configuration.clusterName,
            deployedUiUrl: `http://127.0.0.1:${configuration.ports.frontendServer}`,
            role: configuration.role,
            async cleanup() {},
            async collectDiagnostics(options) {
              collected.push({ options, role: configuration.role });
              return {
                clusterName: configuration.clusterName,
                collected: true,
                logs: [],
                role: configuration.role,
                status: [],
              };
            },
            async createCluster() {},
            async deployRevision() {},
            async destroyCluster() {
              return true;
            },
            async ensureDeployedUiPortForwarding() {
              return [{}];
            },
            getClusterPlatform() {
              return 'linux/amd64';
            },
            getDockerPlatform() {
              return 'linux/amd64';
            },
            preflightReleaseImages(_target, options) {
              return { images: [], platform: options.platform };
            },
            preflightSeedRuntimeImage(options) {
              return { image: 'seed-image', platform: options.platform };
            },
            preflightThirdPartyImages(_target, options) {
              return { images: [], platform: options.platform };
            },
          };
        },
      },
      componentsForRevision() {
        return [];
      },
      revisionUsesMetadataService(target) {
        return target.includes('base');
      },
      async seedData(options) {
        return options.apiBase.endsWith(':3101')
          ? { success: false, error: 'executor submission rejected' }
          : { success: true };
      },
    },
  );

  await assert.rejects(
    runComparison(
      comparisonOptions({
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    /Revision-aware fixture seeding failed/,
  );

  const jsonPath = path.join(run.runDir, 'full-stack-diagnostics.json');
  const htmlPath = path.join(run.runDir, 'full-stack-diagnostics.html');
  assert.equal(fs.existsSync(jsonPath), true);
  assert.equal(fs.existsSync(htmlPath), true);
  const diagnostic = JSON.parse(fs.readFileSync(jsonPath, 'utf8'));
  assert.equal(diagnostic.schemaVersion, 'ui-smoke-full-stack-diagnostics/v1');
  assert.equal(diagnostic.category, 'seed_failure');
  assert.equal(diagnostic.captureValidity, 'seed_failure');
  assert.equal(diagnostic.phase, 'fixture_seeding');
  assert.equal(diagnostic.stacks.length, 2);
  assert.deepEqual(
    collected.map(({ options, role }) => ({
      artifactRoot: options.artifactRoot,
      outputDir: options.outputDir,
      role,
    })),
    [
      {
        artifactRoot: run.runDir,
        outputDir: path.join(run.runDir, 'diagnostics', 'base'),
        role: 'base',
      },
      {
        artifactRoot: run.runDir,
        outputDir: path.join(run.runDir, 'diagnostics', 'head'),
        role: 'head',
      },
    ],
  );
  assert.match(fs.readFileSync(htmlPath, 'utf8'), /seed_failure/);
});

test('incomplete full-stack captures retain browser diagnostics and attribute selector drift', async (t) => {
  const { run, services } = orchestrationHarness(
    t,
    { backendChanged: true, baseRef: '2.17.1' },
    {
      async capturePair(options) {
        for (const role of ['base', 'head']) {
          const directory = path.join(options.screenshotsDir, role);
          fs.mkdirSync(directory, { recursive: true });
          const degraded = role === 'head';
          fs.writeFileSync(
            path.join(directory, 'manifest.json'),
            JSON.stringify({
              browserDiagnostics: degraded
                ? {
                    consoleErrors: ['render failed'],
                    failedRequests: [{ method: 'GET', status: 500, url: '/apis/v2beta1/runs' }],
                  }
                : { consoleErrors: [], failedRequests: [] },
              complete: !degraded,
              fatalErrors: [],
              results: [
                {
                  captureValidity: degraded ? 'selector_drift' : 'valid',
                  diagnostics: degraded
                    ? {
                        consoleErrors: ['route component failed'],
                        droppedConsoleErrors: 3,
                        droppedFailedRequests: 2,
                        failedRequests: [{ method: 'GET', status: 500, url: '/apis/v2beta1/runs' }],
                      }
                    : { consoleErrors: [], failedRequests: [] },
                  filename: 'runs-1280x800.png',
                  page: 'runs',
                  required: true,
                  status: degraded ? 'degraded' : 'success',
                  viewport: { height: 800, width: 1280 },
                },
              ],
              summary: { complete: !degraded },
            }),
          );
        }
        return {
          baseCapture: { success: true },
          comparison: { success: false },
          comparisonDir: path.join(options.screenshotsDir, 'comparison'),
          headCapture: { success: false },
        };
      },
      clusterManager: {
        createKindStack(configuration) {
          return {
            clusterName: configuration.clusterName,
            deployedUiUrl: `http://127.0.0.1:${configuration.ports.frontendServer}`,
            role: configuration.role,
            async cleanup() {},
            async collectDiagnostics() {
              return {
                clusterName: configuration.clusterName,
                collected: true,
                logs: [],
                role: configuration.role,
                status: [],
              };
            },
            async createCluster() {},
            async deployRevision() {},
            async destroyCluster() {
              return true;
            },
            async ensureDeployedUiPortForwarding() {
              return [{}];
            },
            getClusterPlatform() {
              return 'linux/amd64';
            },
            getDockerPlatform() {
              return 'linux/amd64';
            },
            preflightReleaseImages(_target, options) {
              return { images: [], platform: options.platform };
            },
            preflightSeedRuntimeImage(options) {
              return { image: 'seed-image', platform: options.platform };
            },
            preflightThirdPartyImages(_target, options) {
              return { images: [], platform: options.platform };
            },
          };
        },
      },
      combineSemanticManifests(manifests) {
        return { manifests, schemaVersion: 'ui-smoke-semantic/v3' };
      },
      componentsForRevision() {
        return [];
      },
      revisionUsesMetadataService(target) {
        return target.includes('base');
      },
      async seedData(options) {
        fs.mkdirSync(path.dirname(options.manifestPath), { recursive: true });
        fs.writeFileSync(
          options.manifestPath,
          JSON.stringify({
            semantic: {
              logical: {},
              revisionFlavor: options.apiBase.endsWith(':3101')
                ? 'legacy-mlmd'
                : 'native-task-artifact',
              validation: { valid: true },
            },
          }),
        );
        return { success: true };
      },
    },
  );

  assert.equal(
    await runComparison(
      comparisonOptions({
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    false,
  );

  const diagnostic = JSON.parse(
    fs.readFileSync(path.join(run.runDir, 'full-stack-diagnostics.json'), 'utf8'),
  );
  assert.equal(diagnostic.category, 'selector_drift');
  assert.equal(diagnostic.captureValidity, 'selector_drift');
  const headCapture = diagnostic.captures.find(({ role }) => role === 'head');
  assert.deepEqual(headCapture.browserDiagnostics, {
    consoleErrors: ['render failed'],
    failedRequests: [{ method: 'GET', status: 500, url: '/apis/v2beta1/runs' }],
  });
  assert.equal(headCapture.incomplete[0].category, 'selector_drift');
  assert.deepEqual(headCapture.incomplete[0].diagnostics, {
    consoleErrors: ['route component failed'],
    droppedConsoleErrors: 3,
    droppedFailedRequests: 2,
    failedRequests: [{ method: 'GET', status: 500, url: '/apis/v2beta1/runs' }],
  });
});

test('full-stack comparison fails when an isolated cluster cannot be deleted', async (t) => {
  const { run, services } = orchestrationHarness(
    t,
    { backendChanged: true, baseRef: '2.17.1' },
    {
      clusterManager: {
        createKindStack(configuration) {
          return {
            clusterName: configuration.clusterName,
            deployedUiUrl: `http://127.0.0.1:${configuration.ports.frontendServer}`,
            role: configuration.role,
            async cleanup() {},
            async createCluster() {},
            async deployRevision() {},
            async destroyCluster() {
              return configuration.role === 'base';
            },
            async ensureDeployedUiPortForwarding() {
              return [{}];
            },
            getClusterPlatform() {
              return 'linux/amd64';
            },
            getDockerPlatform() {
              return 'linux/amd64';
            },
            preflightReleaseImages(_target, options) {
              return { images: [], platform: options.platform };
            },
            preflightSeedRuntimeImage(options) {
              return { image: 'seed-image', platform: options.platform };
            },
            preflightThirdPartyImages(_target, options) {
              return { images: [], platform: options.platform };
            },
          };
        },
      },
      combineSemanticManifests() {
        return { schemaVersion: 'ui-smoke-semantic/v1' };
      },
      componentsForRevision() {
        return [];
      },
      revisionUsesMetadataService(target) {
        return target.includes('base');
      },
      async seedData(options) {
        fs.mkdirSync(path.dirname(options.manifestPath), { recursive: true });
        fs.writeFileSync(
          options.manifestPath,
          JSON.stringify({
            semantic: {
              logical: {},
              revisionFlavor: options.apiBase.endsWith(':3101')
                ? 'legacy-mlmd'
                : 'native-task-artifact',
              validation: { valid: true },
            },
          }),
        );
        return { success: true };
      },
    },
  );

  await assert.rejects(
    runComparison(
      comparisonOptions({
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    /Failed to delete isolated head cluster/,
  );
});

test('full-stack comparison rejects an unexpected revision data model', async (t) => {
  const { run, services } = orchestrationHarness(
    t,
    { backendChanged: true, baseRef: '2.17.1' },
    {
      clusterManager: {
        createKindStack(configuration) {
          return {
            clusterName: configuration.clusterName,
            deployedUiUrl: `http://127.0.0.1:${configuration.ports.frontendServer}`,
            role: configuration.role,
            async cleanup() {},
            async createCluster() {},
            async deployRevision() {},
            async destroyCluster() {
              return true;
            },
            async ensureDeployedUiPortForwarding() {
              return [{}];
            },
            getClusterPlatform() {
              return 'linux/amd64';
            },
            getDockerPlatform() {
              return 'linux/amd64';
            },
            preflightReleaseImages(_target, options) {
              return { images: [], platform: options.platform };
            },
            preflightSeedRuntimeImage(options) {
              return { image: 'seed-image', platform: options.platform };
            },
            preflightThirdPartyImages(_target, options) {
              return { images: [], platform: options.platform };
            },
          };
        },
      },
      componentsForRevision() {
        return [];
      },
      revisionUsesMetadataService() {
        return true;
      },
      async seedData(options) {
        fs.mkdirSync(path.dirname(options.manifestPath), { recursive: true });
        fs.writeFileSync(
          options.manifestPath,
          JSON.stringify({
            semantic: {
              logical: {},
              revisionFlavor: 'native-task-artifact',
              validation: { valid: true },
            },
          }),
        );
        return { success: true };
      },
    },
  );

  await assert.rejects(
    runComparison(
      comparisonOptions({
        fullStack: true,
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
      }),
      run,
      services,
    ),
    /base revision data model mismatch: expected legacy-mlmd, received native-task-artifact/,
  );
});

test('upgrade mode records migration_unavailable before any cluster mutation', async (t) => {
  const { calls, run, services } = orchestrationHarness(t, {
    backendChanged: true,
    manifestsChanged: true,
    serverChanged: true,
  });

  const success = await runComparison(
    comparisonOptions({
      displayPrNumber: '13986',
      headCheckout: '/reviewed/head',
      prNumber: null,
      trustLocalHead: true,
      trustPrCode: false,
      upgrade: true,
    }),
    run,
    services,
  );

  assert.equal(success, false);
  assert.deepEqual(calls.worktrees, []);
  assert.deepEqual(calls.buildTrusted, []);
  assert.deepEqual(calls.clusterSources, []);
  const result = JSON.parse(fs.readFileSync(path.join(run.runDir, 'upgrade-result.json'), 'utf8'));
  assert.equal(result.captureValidity, 'migration_unavailable');
  assert.equal(result.baseCaptured, false);
  assert.equal(result.headCaptured, false);
  assert.equal(result.migration.requirement.issueNumber, 14029);
  assert.equal(result.request.baseRevision.startsWith('2.17.1@'), true);
  assert.match(result.request.headRevision, /source cccccccccccc/);
  assert.equal(result.request.sourceProvenance.schemaVersion, 'ui-smoke-source/v1');
  assert.deepEqual(calls.runtimePrerequisites, []);
});

test('available upgrade validates a side-effect-free adapter before runtime setup and registers cleanup', async (t) => {
  const order = [];
  let cleanupCalls = 0;
  const operations = Object.fromEntries(
    REQUIRED_OPERATIONS.map((name) => [name, async () => ({ success: true })]),
  );
  operations.cleanupEnvironment = async () => {
    cleanupCalls++;
    return { success: true };
  };
  const { calls, run, services } = orchestrationHarness(
    t,
    { backendChanged: true },
    {
      ensureComparisonRuntime() {
        order.push('runtime');
        calls.runtimePrerequisites.push('ensure');
      },
      loadUpgradeAdapter() {
        order.push('load-adapter');
        return {
          async createOperations(context) {
            order.push('create-operations');
            assert.equal(Object.hasOwn(context, 'services'), false);
            assert.equal(Object.hasOwn(context, 'clusterManager'), false);
            assert.equal(context.baseRef, 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa');
            assert.equal(context.baseTagRef, 'refs/tags/2.17.1');
            assert.equal(context.baseVersion, '2.17.1');
            assert.equal(context.headRoot, path.join(run.runDir, 'worktrees', 'head'));
            assert.equal(context.sourceProvenance.schemaVersion, 'ui-smoke-source/v1');
            assert.equal(typeof context.writeComparisonArtifacts, 'function');
            return operations;
          },
        };
      },
      async orchestrateUpgrade({ operations: received }) {
        order.push('orchestrate');
        assert.equal(typeof received.cleanupEnvironment, 'function');
        return {
          captureValidity: 'infrastructure_failure',
          complete: false,
          phase: 'deploy_base',
        };
      },
      readUpgradeCapabilities() {
        order.push('read-capabilities');
        return {
          adapter: 'upgrade-adapter.js',
          migration: { available: true, version: 'v1' },
          removedResources: [],
          startupGate: { available: true, migrationVersion: 'v1' },
        };
      },
    },
  );

  assert.equal(
    await runComparison(
      comparisonOptions({
        headCheckout: '/reviewed/head',
        prNumber: null,
        trustLocalHead: true,
        trustPrCode: false,
        upgrade: true,
      }),
      run,
      services,
    ),
    false,
  );
  assert.deepEqual(order, [
    'read-capabilities',
    'load-adapter',
    'create-operations',
    'runtime',
    'orchestrate',
  ]);
  const cleanup = calls.cleanupRegistrations.find(({ label }) =>
    label.startsWith('clean upgrade environment'),
  );
  assert.ok(cleanup);
  await cleanup.action();
  await cleanup.action();
  assert.equal(cleanupCalls, 1);
  assert.equal(fs.existsSync(path.join(run.runDir, 'upgrade-result.json')), false);
});

test('upgrade adapter setup failures persist before lifecycle mutation', async (t) => {
  const availableCapabilities = {
    adapter: 'upgrade-adapter.js',
    migration: { available: true, version: 'v1' },
    removedResources: [],
    startupGate: { available: true, migrationVersion: 'v1' },
  };
  const validOperations = Object.fromEntries(
    REQUIRED_OPERATIONS.map((name) => [name, async () => ({ success: true })]),
  );
  validOperations.cleanupEnvironment = async () => ({ success: true });

  for (const scenario of [
    {
      name: 'adapter load',
      phase: 'load_adapter',
      overrides: {
        loadUpgradeAdapter() {
          throw new Error('adapter load failed');
        },
      },
    },
    {
      name: 'operation factory',
      phase: 'create_operations',
      overrides: {
        loadUpgradeAdapter() {
          return {
            async createOperations() {
              throw new Error('operation factory failed');
            },
          };
        },
      },
    },
    {
      category: 'infrastructure_failure',
      name: 'runtime preflight',
      phase: 'runtime_preflight',
      overrides: {
        async ensureComparisonRuntime() {
          throw new Error('Docker is unavailable');
        },
        loadUpgradeAdapter() {
          return {
            async createOperations() {
              return validOperations;
            },
          };
        },
      },
    },
  ]) {
    await t.test(scenario.name, async (t) => {
      let lifecycleCalls = 0;
      const { calls, run, services } = orchestrationHarness(
        t,
        { backendChanged: true },
        {
          async orchestrateUpgrade() {
            lifecycleCalls++;
            throw new Error('lifecycle must not run');
          },
          readUpgradeCapabilities() {
            return availableCapabilities;
          },
          ...scenario.overrides,
        },
      );

      const success = await runComparison(
        comparisonOptions({
          headCheckout: '/reviewed/head',
          prNumber: null,
          trustLocalHead: true,
          trustPrCode: false,
          upgrade: true,
        }),
        run,
        services,
      );

      assert.equal(success, false);
      assert.equal(lifecycleCalls, 0);
      assert.equal(
        calls.cleanupRegistrations.some(({ label }) =>
          label.startsWith('clean upgrade environment'),
        ),
        false,
      );
      const result = JSON.parse(
        fs.readFileSync(path.join(run.runDir, 'upgrade-result.json'), 'utf8'),
      );
      assert.equal(result.complete, false);
      assert.equal(result.captureValidity, 'infrastructure_failure');
      assert.equal(result.error.category, scenario.category || 'configuration_failure');
      assert.equal(result.phase, scenario.phase);
      assert.match(result.request.headRevision, /source cccccccccccc/);
      assert.equal(result.request.sourceProvenance.schemaVersion, 'ui-smoke-source/v1');
    });
  }
});

test('cleanup failure rewrites an optimistic upgrade result as failed', async (t) => {
  let cleanupCalls = 0;
  const operations = Object.fromEntries(
    REQUIRED_OPERATIONS.map((name) => [name, async () => ({ success: true })]),
  );
  operations.cleanupEnvironment = async () => {
    cleanupCalls++;
    throw new Error('cluster cleanup failed');
  };
  const { run, services } = orchestrationHarness(
    t,
    { backendChanged: true },
    {
      loadUpgradeAdapter() {
        return {
          async createOperations() {
            return operations;
          },
        };
      },
      async orchestrateUpgrade({ operations: received, request }) {
        const result = {
          baseCaptured: true,
          captureValidity: 'valid',
          comparisonPassed: true,
          complete: true,
          headCaptured: true,
          phase: 'complete',
          phaseHistory: [{ phase: 'complete', status: 'completed' }],
          request,
        };
        await received.writeResult(result);
        return result;
      },
      readUpgradeCapabilities() {
        return {
          adapter: 'upgrade-adapter.js',
          migration: { available: true, version: 'v1' },
          removedResources: [],
          startupGate: { available: true, migrationVersion: 'v1' },
        };
      },
    },
  );

  const success = await runComparison(
    comparisonOptions({
      headCheckout: '/reviewed/head',
      prNumber: null,
      trustLocalHead: true,
      trustPrCode: false,
      upgrade: true,
    }),
    run,
    services,
  );

  assert.equal(success, false);
  assert.equal(cleanupCalls, 1);
  const result = JSON.parse(fs.readFileSync(path.join(run.runDir, 'upgrade-result.json'), 'utf8'));
  assert.equal(result.complete, false);
  assert.equal(result.captureValidity, 'infrastructure_failure');
  assert.equal(result.error.category, 'cleanup_failure');
  assert.equal(result.phase, 'cleanup_environment');
  assert.match(result.cleanupError.message, /cluster cleanup failed/);
});

test('upgrade result persistence failures are recorded locally and force a failed run', async (t) => {
  const operations = Object.fromEntries(
    REQUIRED_OPERATIONS.map((name) => [name, async () => ({ success: true })]),
  );
  operations.cleanupEnvironment = async () => ({ success: true });
  operations.writeResult = async () => {
    throw new Error('adapter storage unavailable');
  };
  const { run, services } = orchestrationHarness(
    t,
    { backendChanged: true },
    {
      loadUpgradeAdapter() {
        return {
          async createOperations() {
            return operations;
          },
        };
      },
      async orchestrateUpgrade({ operations: received }) {
        const optimisticResult = {
          captureValidity: 'valid',
          comparisonPassed: true,
          complete: true,
          phase: 'complete',
          phaseHistory: [{ phase: 'complete', status: 'completed' }],
        };
        try {
          await received.writeResult(optimisticResult);
          return optimisticResult;
        } catch (error) {
          return error.persistedResult;
        }
      },
      readUpgradeCapabilities() {
        return {
          adapter: 'upgrade-adapter.js',
          migration: { available: true, version: 'v1' },
          removedResources: [],
          startupGate: { available: true, migrationVersion: 'v1' },
        };
      },
    },
  );

  const success = await runComparison(
    comparisonOptions({
      headCheckout: '/reviewed/head',
      prNumber: null,
      trustLocalHead: true,
      trustPrCode: false,
      upgrade: true,
    }),
    run,
    services,
  );

  assert.equal(success, false);
  const persisted = JSON.parse(
    fs.readFileSync(path.join(run.runDir, 'upgrade-result.json'), 'utf8'),
  );
  assert.equal(persisted.complete, false);
  assert.equal(persisted.captureValidity, 'infrastructure_failure');
  assert.equal(persisted.phase, 'persist_result');
  assert.match(persisted.resultWriteError.message, /adapter storage unavailable/);
});

test('upgrade success requires a persisted, valid comparison that passes its threshold', async (t) => {
  for (const [comparisonPassed, expected] of [
    [true, true],
    [false, false],
  ]) {
    await t.test(`comparisonPassed=${comparisonPassed}`, async (t) => {
      const operations = Object.fromEntries(
        REQUIRED_OPERATIONS.map((name) => [name, async () => ({ success: true })]),
      );
      operations.cleanupEnvironment = async () => ({ success: true });
      const { run, services } = orchestrationHarness(
        t,
        { backendChanged: true },
        {
          loadUpgradeAdapter() {
            return {
              async createOperations() {
                return operations;
              },
            };
          },
          async orchestrateUpgrade({ operations: received }) {
            const result = {
              captureValidity: 'valid',
              comparisonPassed,
              complete: true,
              phase: 'complete',
            };
            await received.writeResult(result);
            return result;
          },
          readUpgradeCapabilities() {
            return {
              adapter: 'upgrade-adapter.js',
              migration: { available: true, version: 'v1' },
              removedResources: [],
              startupGate: { available: true, migrationVersion: 'v1' },
            };
          },
        },
      );

      const success = await runComparison(
        comparisonOptions({
          headCheckout: '/reviewed/head',
          prNumber: null,
          trustLocalHead: true,
          trustPrCode: false,
          upgrade: true,
        }),
        run,
        services,
      );

      assert.equal(success, expected);
      assert.equal(fs.existsSync(path.join(run.runDir, 'upgrade-result.json')), true);
    });
  }
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
