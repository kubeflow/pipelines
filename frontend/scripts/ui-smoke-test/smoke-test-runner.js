#!/usr/bin/env node

const crypto = require('crypto');
const fs = require('fs');
const http = require('http');
const https = require('https');
const path = require('path');
const { execFileSync, spawn, spawnSync } = require('child_process');
const { parseArgs } = require('util');

const clusterManager = require('./cluster-manager');
const { COMPONENTS, detectChanges } = require('./detect-changes');
const { seedData } = require('./seed-data');
const { SEMANTIC_ID_NORMALIZATION_MODES } = require('./semantic-id-normalization');
const { combineSemanticManifests } = require('./semantic-manifest');
const {
  CAPTURE_VALIDITY,
  CONTRACT_VERSION: UPGRADE_CONTRACT_VERSION,
  MIGRATION_REQUIREMENT,
  PHASES: UPGRADE_PHASES,
  assessUpgradeCapabilities,
  createResultWriteFailure,
  orchestrateUpgrade,
  validateOperations: validateUpgradeOperations,
  validateRequest: validateUpgradeRequest,
  validateSafeRemovedResources,
  writeUpgradeComparisonArtifacts,
} = require('./upgrade-orchestrator');
const { validateRepository: validateGithubRepository } = require('./upload-to-pr');

const SCRIPT_DIR = __dirname;
const REPO_ROOT = path.resolve(SCRIPT_DIR, '../../..');
const STATE_DIR = path.join(REPO_ROOT, '.ui-smoke-test');
const DEFAULT_REPOSITORY = 'kubeflow/pipelines';
const AUTHORITATIVE_RELEASE_REPOSITORY = `https://github.com/${DEFAULT_REPOSITORY}.git`;
const BASE_PROXY_PORT = 4001;
const HEAD_PROXY_PORT = 4002;
const NODE_VERSION = '24.14.0';
const NPM_VERSION = '11.17.0';
const NODE_IMAGE = `node:${NODE_VERSION}-bookworm`;
const PROCESS_TIMEOUT = 10 * 60 * 1000;
const LOOKS_SAME_TOLERANCE = 2.3;
const LOOKS_SAME_CLUSTER_SIZE = 8;
const EXTERNAL_TOOL_CACHE = '.ui-smoke-tool-cache';
const UPGRADE_CAPABILITY_DESCRIPTOR = '.ui-smoke-upgrade.json';
const STACK_PORTS = Object.freeze({
  base: Object.freeze({ api: 3102, frontendServer: 3101, metadata: 9190, objectStore: 9100 }),
  head: Object.freeze({ api: 3202, frontendServer: 3201, metadata: 9290, objectStore: 9200 }),
});
const FULL_STACK_CAPTURE_VALIDITIES = Object.freeze([
  'valid',
  'ui_rendering_failure',
  'api_incompatibility',
  'seed_failure',
  'missing_fixture',
  'selector_drift',
  'expected_product_removal',
  'infrastructure_failure',
]);
const FULL_STACK_FAILURE_CATEGORIES = Object.freeze(FULL_STACK_CAPTURE_VALIDITIES.slice(1));
const FULL_STACK_DIAGNOSTIC_SCHEMA_VERSION = 'ui-smoke-full-stack-diagnostics/v1';
const CURRENT_ONLY_PAGES = [
  'pipelines',
  'experiments',
  'runs',
  'runs-new',
  'runs-new-pipeline-dialog',
  'runs-new-upload-dialog',
  'recurring-runs',
  'artifacts',
  'artifact-lineage-from-list',
  'executions',
  'pipeline-create',
  'experiment-create',
].join(',');

let verbose = false;
const cleanupActions = [];
const children = new Set();
let cleanupPromise = null;
let signalReceived = null;

function log(message, level = 'info') {
  if (level === 'debug' && !verbose) {
    return;
  }
  const stream = level === 'error' ? process.stderr : process.stdout;
  stream.write(`[${level.toUpperCase()}] ${message}\n`);
}

function helpText() {
  return `UI smoke-test runner

Compare local HEAD with a base ref:
  node smoke-test-runner.js --compare origin/master

Compare a fetched pull request:
  node smoke-test-runner.js --compare origin/master --pr 12345 --trust-pr-code

Capture an already-running UI:
  node smoke-test-runner.js --current-only --use-existing --url http://127.0.0.1:3000

Supported options:
  --compare <ref>                  Base ref for a browser/frontend comparison
  --pr <number>                   Fetch this PR from --repo as the head
  --trust-pr-code                 Confirm that fetched PR code may run in the sandbox
  --repo <owner/name>             Repository used by --pr (default: ${DEFAULT_REPOSITORY})
  --pr-number <number>            Label local HEAD as this PR without fetching it
  --comment                       Create or update the smoke-test PR comment
  --browser-only                  Explicitly ignore server/backend/manifests changes
  --full-stack                    Compare two isolated, revision-matched stacks
  --upgrade                       Upgrade a populated base stack in place before head capture
  --head-checkout <path>          Reviewed local checkout used by --full-stack or --upgrade
  --trust-local-head              Confirm that the selected local runtime may execute
  --viewports <WxH,...>           Capture viewports (default: 1280x800)
  --fail-threshold <percent>      Fail above this visual-difference percentage (default: 0)
  --diff-threshold <percent>      Draw diff markers above this percentage (default: 0)
  --scenario-policy <path>       Optional reviewed per-scenario thresholds and masks
  --current-only                  Capture one existing UI instead of comparing refs
  --use-existing                 Required with --current-only
  --url <http(s)://...>           Full URL used by --current-only
  --teardown                      Delete the managed Kind cluster
  --verbose                       Show command details
  --help                          Show this help

GitHub comments are never posted unless --comment is supplied.`;
}

function parsePercentage(value, optionName) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0 || parsed > 100) {
    throw new Error(`${optionName} must be a number from 0 through 100.`);
  }
  return parsed;
}

function validatePullRequestNumber(value, optionName) {
  if (value === undefined) {
    return null;
  }
  if (!/^[1-9]\d*$/.test(value)) {
    throw new Error(`${optionName} must be a positive integer.`);
  }
  return value;
}

function validateRepository(value) {
  try {
    return validateGithubRepository(value);
  } catch (error) {
    throw new Error(`--repo is invalid: ${error.message}`);
  }
}

function normalizeHttpUrl(value, optionName = '--url') {
  let parsed;
  try {
    parsed = new URL(value);
  } catch (error) {
    throw new Error(`${optionName} must be a valid absolute URL: ${error.message}`);
  }
  if (!['http:', 'https:'].includes(parsed.protocol)) {
    throw new Error(`${optionName} must use http or https.`);
  }
  if (parsed.username || parsed.password) {
    throw new Error(`${optionName} must not contain credentials.`);
  }
  return parsed.toString();
}

function validateViewports(value) {
  const viewports = value.split(',').map((item) => item.trim());
  if (viewports.length === 0 || viewports.some((item) => !/^[1-9]\d*x[1-9]\d*$/.test(item))) {
    throw new Error('--viewports must be a comma-separated WIDTHxHEIGHT list.');
  }
  return viewports.join(',');
}

function parseCli(argv = process.argv.slice(2), env = process.env) {
  let parsed;
  try {
    parsed = parseArgs({
      args: argv,
      allowPositionals: false,
      strict: true,
      options: {
        compare: { type: 'string' },
        pr: { type: 'string' },
        'trust-pr-code': { type: 'boolean', default: false },
        repo: { type: 'string', default: DEFAULT_REPOSITORY },
        'pr-number': { type: 'string' },
        comment: { type: 'boolean', default: false },
        'browser-only': { type: 'boolean', default: false },
        'full-stack': { type: 'boolean', default: false },
        upgrade: { type: 'boolean', default: false },
        'head-checkout': { type: 'string' },
        'trust-local-head': { type: 'boolean', default: false },
        viewports: {
          type: 'string',
          default: env.UI_SMOKE_VIEWPORTS || env.UI_SMOKE_VIEWPORT || '1280x800',
        },
        'fail-threshold': {
          type: 'string',
          default: env.UI_SMOKE_FAIL_THRESHOLD || '0',
        },
        'diff-threshold': {
          type: 'string',
          default: env.UI_SMOKE_DIFF_THRESHOLD || '0',
        },
        'scenario-policy': {
          type: 'string',
          default: env.UI_SMOKE_SCENARIO_POLICY,
        },
        'current-only': { type: 'boolean', default: false },
        'use-existing': { type: 'boolean', default: false },
        url: { type: 'string' },
        teardown: { type: 'boolean', default: false },
        verbose: { type: 'boolean', default: false },
        help: { type: 'boolean', short: 'h', default: false },
      },
    });
  } catch (error) {
    throw new Error(`${error.message}\n\n${helpText()}`);
  }

  const values = parsed.values;
  const options = {
    browserOnly: values['browser-only'],
    comment: values.comment,
    compareRef: values.compare || null,
    currentOnly: values['current-only'],
    diffThreshold: parsePercentage(values['diff-threshold'], '--diff-threshold'),
    failThreshold: parsePercentage(values['fail-threshold'], '--fail-threshold'),
    fullStack: values['full-stack'],
    headCheckout: values['head-checkout'] ? path.resolve(values['head-checkout']) : null,
    help: values.help,
    prNumber: validatePullRequestNumber(values.pr, '--pr'),
    displayPrNumber: validatePullRequestNumber(values['pr-number'], '--pr-number'),
    repository: validateRepository(values.repo),
    scenarioPolicyPath: values['scenario-policy'] ? path.resolve(values['scenario-policy']) : null,
    teardown: values.teardown,
    trustLocalHead: values['trust-local-head'],
    trustPrCode: values['trust-pr-code'],
    upgrade: values.upgrade,
    url: values.url ? normalizeHttpUrl(values.url) : null,
    useExisting: values['use-existing'],
    verbose: values.verbose,
    viewports: validateViewports(values.viewports),
  };

  if (options.help) {
    return options;
  }

  const workflows = [Boolean(options.compareRef), options.currentOnly, options.teardown].filter(
    Boolean,
  );
  if (workflows.length !== 1) {
    throw new Error('Choose exactly one workflow: --compare, --current-only, or --teardown.');
  }
  if (options.currentOnly && (!options.useExisting || !options.url)) {
    throw new Error('--current-only requires both --use-existing and --url.');
  }
  if (!options.currentOnly && (options.useExisting || options.url)) {
    throw new Error('--use-existing and --url are only valid with --current-only.');
  }
  if (options.prNumber && !options.compareRef) {
    throw new Error('--pr is only valid with --compare.');
  }
  if (options.prNumber && !options.trustPrCode) {
    throw new Error(
      '--pr requires --trust-pr-code because the fetched browser build executes PR code.',
    );
  }
  if (options.trustPrCode && !options.prNumber) {
    throw new Error('--trust-pr-code is only valid with --pr.');
  }
  if (options.prNumber && options.displayPrNumber) {
    throw new Error('Use --pr or --pr-number, not both.');
  }
  if (options.comment && !(options.prNumber || options.displayPrNumber)) {
    throw new Error('--comment requires --pr or --pr-number.');
  }
  if (options.comment && !options.compareRef) {
    throw new Error('--comment is only valid with --compare.');
  }
  if (options.browserOnly && !options.compareRef) {
    throw new Error('--browser-only is only valid with --compare.');
  }
  if (options.scenarioPolicyPath && !options.compareRef) {
    throw new Error('--scenario-policy is only valid with --compare.');
  }
  if ((options.fullStack || options.upgrade) && !options.compareRef) {
    throw new Error('--full-stack and --upgrade are only valid with --compare.');
  }
  if ([options.browserOnly, options.fullStack, options.upgrade].filter(Boolean).length > 1) {
    throw new Error('--browser-only, --full-stack, and --upgrade are mutually exclusive.');
  }
  if (options.prNumber && (options.fullStack || options.upgrade || options.headCheckout)) {
    throw new Error(
      'Fetched PR runtime code cannot be executed. Use a reviewed local --head-checkout instead.',
    );
  }
  if ((options.fullStack || options.upgrade) && !options.headCheckout) {
    throw new Error('--full-stack and --upgrade require --head-checkout.');
  }
  if ((options.fullStack || options.upgrade) && !options.trustLocalHead) {
    throw new Error(
      '--full-stack and --upgrade require --trust-local-head for the reviewed local checkout.',
    );
  }
  if (options.headCheckout && !(options.fullStack || options.upgrade)) {
    throw new Error('--head-checkout is only valid with --full-stack or --upgrade.');
  }
  if (options.trustLocalHead && !(options.fullStack || options.upgrade)) {
    throw new Error('--trust-local-head is only valid with --full-stack or --upgrade.');
  }

  return options;
}

function formatCommand(command, args) {
  return [command, ...args]
    .map((value) => (/^[A-Za-z0-9_./:=+@-]+$/.test(value) ? value : JSON.stringify(value)))
    .join(' ');
}

function execute(command, args, options = {}) {
  const commandText = formatCommand(command, args);
  log(`Running: ${commandText}`, 'debug');
  try {
    const encoding = Object.hasOwn(options, 'encoding') ? options.encoding : 'utf8';
    const output = execFileSync(command, args, {
      cwd: options.cwd,
      encoding,
      env: options.env || process.env,
      maxBuffer: 20 * 1024 * 1024,
      stdio: options.stdio || 'pipe',
      timeout: options.timeout || PROCESS_TIMEOUT,
    });
    if (Buffer.isBuffer(output)) return output;
    if (typeof output !== 'string') return '';
    return options.trim === false ? output : output.trim();
  } catch (error) {
    const detail = String(error.stderr || error.stdout || error.message).trim();
    throw new Error(`${commandText} failed${detail ? `: ${detail}` : '.'}`);
  }
}

function commandAvailable(command, args = ['--version']) {
  const result = spawnSync(command, args, { stdio: 'ignore' });
  return result.status === 0;
}

function assertNodeVersion(version = process.versions.node) {
  if (version !== NODE_VERSION) {
    throw new Error(`Node.js ${NODE_VERSION} is required; found ${version}. Use frontend/.nvmrc.`);
  }
}

function assertNpmVersion(version) {
  if (version !== NPM_VERSION) {
    throw new Error(`npm ${NPM_VERSION} is required; found ${version}. Use frontend/package.json.`);
  }
}

function checkPrerequisites({ cluster = false, compare = false, packageManager = true } = {}) {
  assertNodeVersion();
  if (packageManager) assertNpmVersion(execute('npm', ['--version']));
  const missing = [];
  if (compare && !commandAvailable('git')) {
    missing.push('git');
  }
  if (cluster) {
    if (!clusterManager.isKindInstalled()) missing.push('kind');
    if (!clusterManager.isKubectlInstalled()) missing.push('kubectl');
    if (!clusterManager.isDockerRunning()) missing.push('a running Docker daemon');
  }
  if (missing.length > 0) {
    throw new Error(`Missing prerequisite(s): ${missing.join(', ')}.`);
  }
}

function ensureComparisonRuntime() {
  checkPrerequisites({ cluster: true, compare: true });
  ensureToolDependencies();
}

function ensureToolDependencies() {
  log('Restoring the pinned UI smoke-test dependencies from package-lock.json...');
  execute('npm', ['ci'], { cwd: SCRIPT_DIR, stdio: 'inherit' });

  const playwrightPackage = require.resolve('playwright/package.json', { paths: [SCRIPT_DIR] });
  const playwright = require(require.resolve('playwright', { paths: [SCRIPT_DIR] }));
  if (!fs.existsSync(playwright.chromium.executablePath())) {
    log('Installing the pinned Playwright Chromium build...');
    execute(
      process.execPath,
      [path.join(path.dirname(playwrightPackage), 'cli.js'), 'install', 'chromium'],
      { cwd: SCRIPT_DIR, stdio: 'inherit' },
    );
  }
}

function createRunDirectory(
  now = new Date(),
  stateDir = STATE_DIR,
  randomId = crypto.randomBytes(4).toString('hex'),
) {
  const timestamp = now.toISOString().replace(/[:.]/g, '-');
  const runId = `${timestamp}-${process.pid}-${randomId}`;
  const runsDir = path.join(stateDir, 'runs');
  const runDir = path.join(runsDir, runId);
  fs.mkdirSync(stateDir, { recursive: true });
  fs.mkdirSync(runsDir, { recursive: true });
  fs.mkdirSync(runDir, { recursive: false });
  fs.writeFileSync(path.join(stateDir, 'latest-run.txt'), `${runDir}\n`);
  return { runDir, runId };
}

function registerCleanup(label, action) {
  cleanupActions.push({ action, label });
}

async function terminateChild(child, timeout = 3000) {
  return clusterManager.terminateChild(child, timeout);
}

async function executeCleanupActions(actions) {
  const failures = [];
  for (const { action, label } of [...actions].reverse()) {
    try {
      await action();
    } catch (error) {
      const failure = new Error(`Cleanup failed (${label}): ${error.message}`, { cause: error });
      failures.push(failure);
      log(failure.message, 'error');
    }
  }
  if (failures.length > 0) {
    throw new AggregateError(failures, `${failures.length} cleanup action(s) failed.`);
  }
}

async function cleanup() {
  if (cleanupPromise) {
    return cleanupPromise;
  }
  cleanupPromise = (async () => {
    const actions = [...cleanupActions];
    cleanupActions.length = 0;
    await executeCleanupActions(actions);
  })();
  return cleanupPromise;
}

function installSignalHandlers() {
  for (const signal of ['SIGINT', 'SIGTERM']) {
    process.once(signal, () => {
      if (signalReceived) return;
      signalReceived = signal;
      log(`Received ${signal}; cleaning up...`, 'error');
      cleanup()
        .catch((error) => log(`Signal cleanup failed: ${error.message}`, 'error'))
        .finally(() => {
          process.exit(signal === 'SIGINT' ? 130 : 143);
        });
    });
  }
}

function spawnManaged(command, args, options = {}) {
  log(`Starting: ${formatCommand(command, args)}`, 'debug');
  const child = spawn(command, args, {
    cwd: options.cwd,
    env: options.env || process.env,
    stdio: options.stdio || ['ignore', 'ignore', 'pipe'],
  });
  children.add(child);
  child.once('close', () => children.delete(child));
  if (child.stderr) {
    child.stderr.on('data', (chunk) => {
      if (verbose) process.stderr.write(chunk);
    });
  }
  registerCleanup(`stop ${command} (${child.pid || 'not started'})`, () => terminateChild(child));
  return child;
}

function runChild(command, args, options = {}) {
  return new Promise((resolve) => {
    log(`Running: ${formatCommand(command, args)}`, 'debug');
    const spawnFn = options.spawnFn || spawn;
    const timeout = options.timeout ?? PROCESS_TIMEOUT;
    const killTimeout = options.killTimeout ?? 3000;
    if (!Number.isFinite(timeout) || timeout <= 0) {
      throw new Error('Child process timeout must be a positive number.');
    }
    const child = spawnFn(command, args, {
      cwd: options.cwd,
      env: options.env || process.env,
      stdio: 'inherit',
    });
    let settled = false;
    let timedOut = false;
    let timeoutTimer;
    let timeoutError = null;
    const finish = (result, finishOptions = {}) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutTimer);
      if (!finishOptions.retainChild) children.delete(child);
      resolve(result);
    };
    children.add(child);
    registerCleanup(`stop ${command} (${child.pid || 'not started'})`, () => terminateChild(child));
    child.once('error', (error) => {
      finish({ error, success: false, timedOut });
    });
    child.once('close', (code, childSignal) => {
      finish({
        code,
        error: timeoutError,
        signal: childSignal,
        success: code === 0 && !timedOut,
        timedOut,
      });
    });
    timeoutTimer = setTimeout(() => {
      timedOut = true;
      timeoutError = new Error(`${formatCommand(command, args)} timed out after ${timeout}ms`);
      terminateChild(child, killTimeout)
        .then(() => {
          // terminateChild resolves only after the child closes. The close handler normally wins;
          // retain this fallback for test doubles that expose exit state without emitting close.
          finish({ error: timeoutError, success: false, timedOut: true });
        })
        .catch((terminationError) => {
          const error = new AggregateError(
            [timeoutError, terminationError],
            `${formatCommand(command, args)} timed out and could not be terminated.`,
          );
          // Keep a stubborn process registered so global cleanup can make another bounded attempt.
          finish(
            { error, success: false, terminationFailed: true, timedOut: true },
            { retainChild: true },
          );
        });
    }, timeout);
  });
}

function requestSuccessful(urlValue, timeout = 2000, requestFactory = null) {
  return new Promise((resolve) => {
    const parsed = new URL(urlValue);
    const client = parsed.protocol === 'https:' ? https : http;
    const request = (requestFactory || client.get.bind(client))(parsed, (response) => {
      response.resume();
      resolve(response.statusCode >= 200 && response.statusCode < 400);
    });
    request.once('error', () => resolve(false));
    request.setTimeout(timeout, () => {
      request.destroy();
      resolve(false);
    });
  });
}

async function waitForUrl(urlValue, child, timeout = 30000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (child && (child.exitCode !== null || child.signalCode !== null)) {
      throw new Error(`Server for ${urlValue} exited before it became ready.`);
    }
    if (await requestSuccessful(urlValue)) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 300));
  }
  throw new Error(`Timed out waiting for a successful response from ${urlValue}.`);
}

function gitOutput(args, cwd = REPO_ROOT, options = {}) {
  return execute('git', args, { cwd, ...options });
}

function addDetachedWorktree(target, gitRef, repositoryRoot = REPO_ROOT) {
  gitOutput(['worktree', 'add', '--detach', target, gitRef], repositoryRoot);
  registerCleanup(`remove worktree ${target}`, () => {
    execute('git', ['worktree', 'remove', '--force', target], { cwd: repositoryRoot });
  });
}

function fetchPullRequest(repository, prNumber, runId) {
  const temporaryRef = `refs/ui-smoke-test/${runId}/pr-${prNumber}`;
  const repositoryUrl = `https://github.com/${repository}.git`;
  execute(
    'git',
    ['fetch', '--force', '--no-tags', repositoryUrl, `+refs/pull/${prNumber}/head:${temporaryRef}`],
    {
      cwd: REPO_ROOT,
      env: { ...process.env, GIT_TERMINAL_PROMPT: '0' },
      stdio: 'inherit',
    },
  );
  registerCleanup(`delete temporary ref ${temporaryRef}`, () => {
    execute('git', ['update-ref', '-d', temporaryRef], { cwd: REPO_ROOT });
  });
  return temporaryRef;
}

async function runRequiredChild(command, args, options = {}) {
  const result = await runChild(command, args, options);
  if (!result.success) {
    const detail = result.error ? `: ${result.error.message}` : '';
    throw new Error(`${formatCommand(command, args)} failed${detail}.`);
  }
}

async function buildTrustedFrontend(repoRoot) {
  const frontendDir = path.join(repoRoot, 'frontend');
  await runRequiredChild('npm', ['ci'], { cwd: frontendDir });
  await runRequiredChild('npm', ['run', 'build'], { cwd: frontendDir });
}

function externalContainerArguments(repoRoot, command, { network = null } = {}) {
  const args = [
    'run',
    '--rm',
    '--init',
    '--cap-drop',
    'ALL',
    '--security-opt',
    'no-new-privileges',
    '--pids-limit',
    '512',
    '--memory',
    '4g',
    '--cpus',
    '4',
    '--read-only',
    '--tmpfs',
    '/tmp:rw,nosuid,size=1g',
    '--env',
    'CI=true',
    '--env',
    'HOME=/tmp/ui-smoke-home',
    '--env',
    'COREPACK_HOME=/workspace/.ui-smoke-tool-cache/corepack',
    '--env',
    'COREPACK_ENV_FILE=0',
    '--env',
    'COREPACK_ENABLE_PROJECT_SPEC=0',
    '--env',
    'NPM_CONFIG_CACHE=/workspace/.ui-smoke-tool-cache/npm',
    '--env',
    'NPM_CONFIG_AUDIT=false',
    '--env',
    'NPM_CONFIG_FUND=false',
    '--env',
    'NPM_CONFIG_UPDATE_NOTIFIER=false',
    '--volume',
    `${repoRoot}:/workspace`,
    '--workdir',
    '/workspace/frontend',
  ];
  if (network) args.push('--network', network);
  if (typeof process.getuid === 'function' && typeof process.getgid === 'function') {
    args.push('--user', `${process.getuid()}:${process.getgid()}`);
  }
  args.push(NODE_IMAGE, 'bash', '-lc', command);
  return args;
}

function externalInstallArguments(repoRoot) {
  return externalContainerArguments(
    repoRoot,
    `corepack install --global npm@${NPM_VERSION} && test "$(corepack npm --version)" = "${NPM_VERSION}" && corepack npm ci --ignore-scripts && corepack npm --prefix server ci --ignore-scripts && corepack npm --prefix mock-backend ci --ignore-scripts`,
  );
}

function externalBuildArguments(repoRoot) {
  return externalContainerArguments(
    repoRoot,
    `test "$(corepack npm --version)" = "${NPM_VERSION}" && corepack npm ci --offline && corepack npm run build`,
    { network: 'none' },
  );
}

function isPathInside(rootPath, candidatePath) {
  const relative = path.relative(rootPath, candidatePath);
  return (
    relative === '' ||
    (!relative.startsWith(`..${path.sep}`) && relative !== '..' && !path.isAbsolute(relative))
  );
}

function validateExternalBuildArtifact(repoRoot) {
  const realRepoRoot = fs.realpathSync(repoRoot);
  const buildDir = path.join(repoRoot, 'frontend', 'build');
  const buildStat = fs.lstatSync(buildDir);
  if (!buildStat.isDirectory() || buildStat.isSymbolicLink()) {
    throw new Error(`External frontend build must be a non-symlink directory: ${buildDir}`);
  }
  const realBuildDir = fs.realpathSync(buildDir);
  if (!isPathInside(realRepoRoot, realBuildDir)) {
    throw new Error(`External frontend build escaped its worktree: ${realBuildDir}`);
  }
  const indexPath = path.join(buildDir, 'index.html');
  const indexStat = fs.lstatSync(indexPath);
  const realIndexPath = fs.realpathSync(indexPath);
  if (
    !indexStat.isFile() ||
    indexStat.isSymbolicLink() ||
    !isPathInside(realBuildDir, realIndexPath)
  ) {
    throw new Error(
      `External frontend index must be a regular file inside the build: ${indexPath}`,
    );
  }
  return realBuildDir;
}

async function buildExternalFrontend(repoRoot) {
  const cachePath = path.join(repoRoot, EXTERNAL_TOOL_CACHE);
  if (fs.existsSync(cachePath)) {
    throw new Error(`Refusing to reuse an external build cache path: ${cachePath}`);
  }
  fs.mkdirSync(cachePath, { recursive: false });
  try {
    await runRequiredChild('docker', externalInstallArguments(repoRoot), { cwd: REPO_ROOT });
    await runRequiredChild('docker', externalBuildArguments(repoRoot), { cwd: REPO_ROOT });
    validateExternalBuildArtifact(repoRoot);
  } finally {
    fs.rmSync(cachePath, { force: true, recursive: true });
  }
}

function shortSha(gitRef, cwd = REPO_ROOT) {
  return gitOutput(['rev-parse', '--short=12', `${gitRef}^{commit}`], cwd);
}

function fullSha(gitRef, cwd = REPO_ROOT) {
  const sha = gitOutput(['rev-parse', `${gitRef}^{commit}`], cwd);
  if (!/^[0-9a-f]{40,64}$/i.test(sha)) {
    throw new Error(`Could not resolve a full commit SHA for ${gitRef}.`);
  }
  return sha.toLowerCase();
}

function revisionBuildMetadata(repoRoot, commitSha, git = gitOutput) {
  if (!/^[0-9a-f]{40,64}$/i.test(commitSha)) {
    throw new Error(`Cannot build revision images with invalid commit SHA ${commitSha}.`);
  }
  const nodeVersionPath = path.join(repoRoot, 'frontend', '.nvmrc');
  const nodeVersion = fs.readFileSync(nodeVersionPath, 'utf8').trim().replace(/^v/, '');
  if (!/^\d+\.\d+\.\d+$/.test(nodeVersion)) {
    throw new Error(
      `Invalid frontend Node version in ${nodeVersionPath}: ${nodeVersion || 'empty'}.`,
    );
  }
  const buildDate = git(['show', '-s', '--format=%cI', commitSha], repoRoot);
  if (!buildDate || /[\r\n\0]/.test(buildDate)) {
    throw new Error(`Could not resolve deterministic build metadata for ${commitSha}.`);
  }
  return Object.freeze({
    buildDate,
    commitSha: commitSha.toLowerCase(),
    nodeVersion,
    tagName: commitSha.toLowerCase(),
  });
}

function validateTrustedHeadCheckout(candidatePath, options = {}) {
  const { git = gitOutput, repositoryRoot = REPO_ROOT } = options;
  if (!candidatePath) throw new Error('A reviewed local head checkout is required.');

  let checkoutRoot;
  let expectedRoot;
  try {
    checkoutRoot = fs.realpathSync(candidatePath);
    expectedRoot = fs.realpathSync(repositoryRoot);
  } catch (error) {
    throw new Error(`Cannot resolve the selected local checkout: ${error.message}`);
  }
  if (!fs.statSync(checkoutRoot).isDirectory()) {
    throw new Error(`Selected local checkout is not a directory: ${checkoutRoot}`);
  }

  const selectedTopLevel = fs.realpathSync(git(['rev-parse', '--show-toplevel'], checkoutRoot));
  if (selectedTopLevel !== checkoutRoot) {
    throw new Error(`--head-checkout must name the Git worktree root: ${selectedTopLevel}`);
  }
  const commonDirectory = (root) => {
    const value = git(['rev-parse', '--git-common-dir'], root);
    return fs.realpathSync(path.resolve(root, value));
  };
  if (commonDirectory(checkoutRoot) !== commonDirectory(expectedRoot)) {
    throw new Error('--head-checkout must be a worktree of the same repository.');
  }
  return checkoutRoot;
}

function parseCommitSha(value, description) {
  const sha = String(value || '')
    .trim()
    .toLowerCase();
  if (!/^[0-9a-f]{40,64}$/.test(sha)) {
    throw new Error(`Could not resolve a full commit SHA for ${description}.`);
  }
  return sha;
}

function hashParts(parts) {
  const hash = crypto.createHash('sha256');
  for (const part of parts) {
    const value = Buffer.isBuffer(part) ? part : Buffer.from(String(part));
    hash.update(`${value.length}:`);
    hash.update(value);
  }
  return hash.digest('hex');
}

function validateSnapshotRelativePath(relativePath) {
  if (
    !relativePath ||
    path.isAbsolute(relativePath) ||
    relativePath === '..' ||
    relativePath.startsWith(`..${path.sep}`)
  ) {
    throw new Error(`Git returned an unsafe untracked path: ${JSON.stringify(relativePath)}.`);
  }
  return relativePath;
}

function decodeGitPath(pathBytes) {
  const relativePath = pathBytes.toString('utf8');
  if (!Buffer.from(relativePath, 'utf8').equals(pathBytes)) {
    throw new Error(
      'Trusted source snapshot contains a non-UTF-8 Git path, which cannot be represented safely.',
    );
  }
  return relativePath;
}

function parseNullDelimitedGitPaths(output) {
  const bytes = Buffer.isBuffer(output) ? output : Buffer.from(output, 'utf8');
  if (bytes.length === 0) return [];
  if (bytes.at(-1) !== 0) {
    throw new Error('Git returned a truncated NUL-delimited path list.');
  }
  const paths = [];
  let start = 0;
  for (let index = 0; index < bytes.length; index++) {
    if (bytes[index] !== 0) continue;
    if (index > start) paths.push(decodeGitPath(bytes.subarray(start, index)));
    start = index + 1;
  }
  return paths;
}

function captureWorkingTreeOverlay(sourceRoot, git = gitOutput) {
  const headCommit = parseCommitSha(
    git(['rev-parse', 'HEAD^{commit}'], sourceRoot),
    'the reviewed checkout HEAD',
  );
  const trackedPatch = git(
    ['diff', '--binary', '--full-index', '--no-ext-diff', '--no-textconv', 'HEAD', '--'],
    sourceRoot,
    { encoding: 'buffer', trim: false },
  );
  const untrackedPaths = parseNullDelimitedGitPaths(
    git(['ls-files', '--others', '--exclude-standard', '-z'], sourceRoot, {
      encoding: 'buffer',
      trim: false,
    }),
  ).sort();
  const untrackedEntries = untrackedPaths.map((relativePath) => {
    validateSnapshotRelativePath(relativePath);
    const sourcePath = path.join(sourceRoot, relativePath);
    const stat = fs.lstatSync(sourcePath);
    if (!stat.isFile() && !stat.isSymbolicLink()) {
      throw new Error(
        `Untracked snapshot input must be a regular file or symlink: ${relativePath}.`,
      );
    }
    const content = stat.isSymbolicLink()
      ? Buffer.from(fs.readlinkSync(sourcePath))
      : fs.readFileSync(sourcePath);
    return {
      content,
      executable: stat.isFile() && (stat.mode & 0o111) !== 0,
      path: relativePath,
      type: stat.isSymbolicLink() ? 'symlink' : 'file',
    };
  });
  const endingHeadCommit = parseCommitSha(
    git(['rev-parse', 'HEAD^{commit}'], sourceRoot),
    'the reviewed checkout HEAD',
  );
  const fingerprint = hashParts([
    'ui-smoke-working-tree/v1',
    headCommit,
    endingHeadCommit,
    trackedPatch,
    ...untrackedEntries.flatMap((entry) => [
      entry.path,
      entry.type,
      entry.executable ? 'executable' : 'non-executable',
      entry.content,
    ]),
  ]);
  return { endingHeadCommit, fingerprint, headCommit, trackedPatch, untrackedEntries };
}

function copyOverlayEntry(targetRoot, entry) {
  const destination = path.join(targetRoot, entry.path);
  if (!isPathInside(targetRoot, destination)) {
    throw new Error(`Snapshot overlay escaped its worktree: ${entry.path}.`);
  }
  if (fs.existsSync(destination)) {
    throw new Error(`Untracked snapshot path unexpectedly exists in HEAD: ${entry.path}.`);
  }
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  if (entry.type === 'symlink') {
    fs.symlinkSync(entry.content.toString(), destination);
    return;
  }
  fs.writeFileSync(destination, entry.content, { mode: entry.executable ? 0o755 : 0o644 });
}

function validateSnapshotSymlinks(snapshotRoot) {
  const pending = [snapshotRoot];
  while (pending.length > 0) {
    const directory = pending.pop();
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      if (directory === snapshotRoot && entry.name === '.git') continue;
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) {
        pending.push(entryPath);
        continue;
      }
      if (!entry.isSymbolicLink()) continue;
      const linkedPath = path.resolve(path.dirname(entryPath), fs.readlinkSync(entryPath));
      let realLinkedPath;
      try {
        realLinkedPath = fs.realpathSync(linkedPath);
      } catch (error) {
        throw new Error(
          `Trusted source snapshot contains a dangling symlink: ${path.relative(snapshotRoot, entryPath)}.`,
        );
      }
      if (
        !isPathInside(snapshotRoot, linkedPath) ||
        !isPathInside(fs.realpathSync(snapshotRoot), realLinkedPath)
      ) {
        throw new Error(
          `Trusted source snapshot symlink escapes its worktree: ${path.relative(snapshotRoot, entryPath)}.`,
        );
      }
    }
  }
}

function sourceFingerprintLabel(provenance) {
  if (!provenance?.fingerprint?.startsWith('sha256:')) return null;
  return `source ${provenance.fingerprint.slice('sha256:'.length, 'sha256:'.length + 12)}`;
}

function formatSourceRevision(provenance, fallbackCommit) {
  const label = sourceFingerprintLabel(provenance);
  return label ? `HEAD@${fallbackCommit} (${label})` : `HEAD@${fallbackCommit}`;
}

function materializeTrustedHeadSnapshot(sourceRoot, targetRoot, options = {}) {
  const { addWorktree = addDetachedWorktree, git = gitOutput } = options;
  const firstCapture = captureWorkingTreeOverlay(sourceRoot, git);
  const secondCapture = captureWorkingTreeOverlay(sourceRoot, git);
  if (
    firstCapture.headCommit !== firstCapture.endingHeadCommit ||
    secondCapture.headCommit !== secondCapture.endingHeadCommit ||
    firstCapture.fingerprint !== secondCapture.fingerprint
  ) {
    throw new Error(
      'The reviewed local checkout changed while its source snapshot was being captured. Retry after edits stop.',
    );
  }

  addWorktree(targetRoot, firstCapture.headCommit, sourceRoot);
  const patchPath = path.join(
    path.dirname(targetRoot),
    `.${path.basename(targetRoot)}-${firstCapture.fingerprint.slice(0, 12)}.patch`,
  );
  try {
    if (firstCapture.trackedPatch.length > 0) {
      fs.writeFileSync(patchPath, firstCapture.trackedPatch);
      git(['apply', '--binary', '--whitespace=nowarn', patchPath], targetRoot);
    }
    for (const entry of firstCapture.untrackedEntries) copyOverlayEntry(targetRoot, entry);
    validateSnapshotSymlinks(targetRoot);
  } finally {
    fs.rmSync(patchPath, { force: true });
  }

  git(['add', '--all', '--', '.'], targetRoot);
  const tree = git(['write-tree'], targetRoot).trim().toLowerCase();
  if (!/^[0-9a-f]{40,64}$/.test(tree)) {
    throw new Error('Could not compute the trusted source snapshot tree.');
  }
  const snapshotCommit = parseCommitSha(
    git(['rev-parse', 'HEAD^{commit}'], targetRoot),
    'the trusted source snapshot HEAD',
  );
  if (snapshotCommit !== firstCapture.headCommit) {
    throw new Error('The trusted source snapshot detached at an unexpected commit.');
  }

  const fingerprint = `sha256:${hashParts(['ui-smoke-source/v1', snapshotCommit, tree])}`;
  return Object.freeze({
    fingerprint,
    overlay: Object.freeze({
      trackedPatchSha256: `sha256:${crypto
        .createHash('sha256')
        .update(firstCapture.trackedPatch)
        .digest('hex')}`,
      untrackedEntries: Object.freeze(
        firstCapture.untrackedEntries.map((entry) =>
          Object.freeze({
            executable: entry.executable,
            path: entry.path,
            sha256: `sha256:${crypto.createHash('sha256').update(entry.content).digest('hex')}`,
            type: entry.type,
          }),
        ),
      ),
    }),
    revision: Object.freeze({ commit: snapshotCommit, ref: 'HEAD', tree }),
    schemaVersion: 'ui-smoke-source/v1',
  });
}

function readManifestSources(repoRoot) {
  const manifestRoot = path.join(repoRoot, 'manifests');
  const sources = [];
  const pending = [manifestRoot];
  while (pending.length > 0) {
    const directory = pending.pop();
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) pending.push(entryPath);
      if (entry.isFile() && /\.ya?ml$/i.test(entry.name)) {
        sources.push(fs.readFileSync(entryPath, 'utf8'));
      }
    }
  }
  return sources.join('\n');
}

function renderRevisionManifestSources(repoRoot, options = {}) {
  const execute = options.execFileSync || execFileSync;
  const platformAgnostic = path.join(
    repoRoot,
    'manifests',
    'kustomize',
    'env',
    'platform-agnostic',
  );
  if (!fs.statSync(platformAgnostic, { throwIfNoEntry: false })?.isDirectory()) {
    throw new Error(`Revision is missing the platform-agnostic manifest at ${platformAgnostic}.`);
  }

  let rendered;
  try {
    rendered = execute('kubectl', ['kustomize', platformAgnostic], {
      encoding: 'utf8',
      maxBuffer: 64 * 1024 * 1024,
      timeout: 180000,
    });
  } catch (error) {
    throw new Error(`Failed to render revision manifests from ${platformAgnostic}.`, {
      cause: error,
    });
  }
  const contents = Buffer.isBuffer(rendered) ? rendered.toString('utf8') : rendered;
  if (typeof contents !== 'string' || contents.trim() === '') {
    throw new Error(`Rendered revision manifests from ${platformAgnostic} were empty.`);
  }
  return contents;
}

function manifestDefinesName(manifestSources, name) {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`^\\s*name:\\s*["']?${escapedName}["']?\\s*$`, 'm').test(manifestSources);
}

function componentsForRevision(repoRoot, components = COMPONENTS, manifestSources = null) {
  const sources = manifestSources === null ? readManifestSources(repoRoot) : manifestSources;
  return components.filter(
    (component) =>
      fs.existsSync(path.join(repoRoot, component.dockerfile)) &&
      (!component.deployment || manifestDefinesName(sources, component.deployment)),
  );
}

function revisionUsesMetadataService(repoRoot, manifestSources = null) {
  const sources = manifestSources === null ? readManifestSources(repoRoot) : manifestSources;
  return manifestDefinesName(sources, 'metadata-envoy-service');
}

function stackConfiguration(run, role, revision, options = {}) {
  if (!Object.hasOwn(STACK_PORTS, role)) throw new Error(`Unknown stack role: ${role}`);
  const digest = crypto
    .createHash('sha256')
    .update(`${run.runId}:${role}`)
    .digest('hex')
    .slice(0, 12);
  const ports = { ...STACK_PORTS[role] };
  if (options.metadata === false) ports.metadata = null;
  return {
    archiveDir: path.join(run.runDir, 'images', role),
    clusterName: `ui-smoke-${role}-${digest}`,
    kubeconfigPath: path.join(run.runDir, 'kubeconfigs', `${role}.yaml`),
    ports,
    revision,
    role,
  };
}

function validateFullStackBaseRelease(baseRef) {
  if (!/^\d+\.\d+\.\d+$/.test(baseRef)) {
    throw new Error(
      `--full-stack and --upgrade require an exact release base such as 2.17.1; received ${baseRef}.`,
    );
  }
  return baseRef;
}

function resolvePublishedReleaseCommit(version, options = {}) {
  const { cwd = REPO_ROOT, git = gitOutput } = options;
  const tagRef = `refs/tags/${validateFullStackBaseRelease(version)}`;
  const peeledRef = `${tagRef}^{}`;
  let output;
  try {
    output = git(
      ['ls-remote', '--exit-code', AUTHORITATIVE_RELEASE_REPOSITORY, tagRef, peeledRef],
      cwd,
      {
        env: { ...process.env, GIT_TERMINAL_PROMPT: '0' },
        trim: false,
      },
    );
  } catch (error) {
    throw new Error(
      `Could not resolve published release tag ${tagRef} from ${AUTHORITATIVE_RELEASE_REPOSITORY}: ${error.message}`,
    );
  }

  const advertised = new Map();
  for (const line of output.split(/\r?\n/).filter(Boolean)) {
    const match = line.match(/^([0-9a-f]{40,64})\t(.+)$/i);
    if (!match || (match[2] !== tagRef && match[2] !== peeledRef) || advertised.has(match[2])) {
      throw new Error(`Published release tag ${tagRef} returned an invalid Git advertisement.`);
    }
    advertised.set(match[2], match[1].toLowerCase());
  }
  const directCommit = advertised.get(tagRef);
  if (!directCommit) {
    throw new Error(`Published release tag ${tagRef} was not advertised by ${DEFAULT_REPOSITORY}.`);
  }
  return advertised.get(peeledRef) || directCommit;
}

function resolveExactBaseRelease(baseRef, options = {}) {
  const {
    cwd = REPO_ROOT,
    git = gitOutput,
    resolveCommit = fullSha,
    resolvePublishedCommit = resolvePublishedReleaseCommit,
  } = options;
  const version = validateFullStackBaseRelease(baseRef);
  const branchRef = `refs/heads/${version}`;
  const matchingBranch = git(['for-each-ref', '--format=%(refname)', branchRef], cwd).trim();
  if (matchingBranch) {
    throw new Error(
      `Refusing ambiguous release base ${version}: local branch ${branchRef} exists. Use a checkout without a semver-named branch.`,
    );
  }

  const tagRef = `refs/tags/${version}`;
  let commit;
  try {
    commit = resolveCommit(tagRef, cwd);
  } catch (error) {
    throw new Error(`Exact release tag ${tagRef} is missing or does not resolve to a commit.`);
  }
  const publishedCommit = parseCommitSha(
    resolvePublishedCommit(version, { tagRef }),
    `published release tag ${tagRef}`,
  );
  if (publishedCommit !== commit) {
    throw new Error(
      `Local release tag ${tagRef} resolves to ${commit}, but the published ${DEFAULT_REPOSITORY} tag resolves to ${publishedCommit}.`,
    );
  }
  return Object.freeze({ commit, tagRef, version });
}

function readUpgradeCapabilities(headRoot) {
  const descriptorPath = path.join(headRoot, UPGRADE_CAPABILITY_DESCRIPTOR);
  let descriptorStat;
  try {
    descriptorStat = fs.lstatSync(descriptorPath);
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
    return {
      descriptorPath,
      migration: {
        available: false,
        reason: `Missing reviewed capability descriptor ${UPGRADE_CAPABILITY_DESCRIPTOR}`,
      },
      startupGate: {
        available: false,
        reason: `Missing reviewed capability descriptor ${UPGRADE_CAPABILITY_DESCRIPTOR}`,
      },
    };
  }

  const realHeadRoot = fs.realpathSync(headRoot);
  const realDescriptorPath = fs.realpathSync(descriptorPath);
  if (
    !descriptorStat.isFile() ||
    descriptorStat.isSymbolicLink() ||
    !isPathInside(realHeadRoot, realDescriptorPath)
  ) {
    throw new Error(
      `${UPGRADE_CAPABILITY_DESCRIPTOR} must be a regular file inside the reviewed checkout.`,
    );
  }

  let descriptor;
  try {
    descriptor = JSON.parse(fs.readFileSync(realDescriptorPath, 'utf8'));
  } catch (error) {
    throw new Error(`Invalid ${UPGRADE_CAPABILITY_DESCRIPTOR}: ${error.message}`);
  }
  if (descriptor.schemaVersion !== 'ui-smoke-upgrade/v1') {
    throw new Error(`${UPGRADE_CAPABILITY_DESCRIPTOR} must use schemaVersion ui-smoke-upgrade/v1.`);
  }
  return {
    ...descriptor.capabilities,
    adapter: descriptor.adapter || null,
    descriptorPath,
    removedResources: descriptor.removedResources || [],
  };
}

function loadUpgradeAdapter(headRoot, descriptor) {
  if (!descriptor.adapter) return null;
  const realHeadRoot = fs.realpathSync(headRoot);
  const candidate = path.resolve(realHeadRoot, descriptor.adapter);
  if (!isPathInside(realHeadRoot, candidate)) {
    throw new Error(`${UPGRADE_CAPABILITY_DESCRIPTOR} adapter escapes the reviewed checkout.`);
  }
  const stat = fs.lstatSync(candidate);
  const realCandidate = fs.realpathSync(candidate);
  if (!stat.isFile() || stat.isSymbolicLink() || !isPathInside(realHeadRoot, realCandidate)) {
    throw new Error('The reviewed upgrade adapter must be a regular file inside the checkout.');
  }
  const adapter = require(realCandidate);
  if (!adapter || typeof adapter.createOperations !== 'function') {
    throw new Error('The reviewed upgrade adapter must export createOperations(context).');
  }
  return adapter;
}

function writeJson(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function categorizedFullStackError(category, message, details = null) {
  if (!FULL_STACK_FAILURE_CATEGORIES.includes(category)) {
    throw new Error(`Invalid full-stack failure category: ${category}`);
  }
  const error = new Error(message);
  error.failureCategory = category;
  if (details) error.failureDetails = details;
  return error;
}

function seedFailureCategory(seedResult) {
  switch (String(seedResult?.failureType || '').toUpperCase()) {
    case 'API_INCOMPATIBILITY':
      return 'api_incompatibility';
    case 'MISSING_FIXTURE':
      return 'missing_fixture';
    default:
      break;
  }
  const detail = String(seedResult?.error || '').toLowerCase();
  if (/api not healthy|inventory api|detail route|http|status|endpoint/.test(detail)) {
    return 'api_incompatibility';
  }
  if (/missing required|semantic binding|missing fixture/.test(detail)) return 'missing_fixture';
  return 'seed_failure';
}

function classifyCaptureResult(result) {
  if (FULL_STACK_CAPTURE_VALIDITIES.includes(result?.captureValidity)) {
    if (result.captureValidity === 'valid') return null;
    if (result.captureValidity !== 'expected_product_removal') {
      return result.captureValidity;
    }
    if (result.status === 'success') return 'expected_product_removal';
  }
  if (FULL_STACK_FAILURE_CATEGORIES.includes(result?.failureCategory)) {
    return result.failureCategory;
  }
  if (
    result?.status === 'success' &&
    (result?.expectedProductRemoval === true || result?.expectation === 'expected_product_removal')
  ) {
    return 'expected_product_removal';
  }
  const detail = `${result?.error || ''} ${result?.reason || ''}`.toLowerCase();
  if (/missing seed|missing fixture|semantic fixture/.test(detail)) return 'missing_fixture';
  if (result?.status === 'degraded') return 'selector_drift';
  if (
    result?.status === 'failed' &&
    /\bhttp\b|status(?: code)? [45]\d\d|api|network|request|response|fetch|err_connection/.test(
      detail,
    )
  ) {
    return 'api_incompatibility';
  }
  if (result?.status === 'failed') return 'ui_rendering_failure';
  return null;
}

function redactFullStackDiagnosticText(value) {
  return String(value || '')
    .replace(/:\/\/([^\s/:@]+):([^\s/@]+)@/g, '://<redacted>:<redacted>@')
    .replace(/\bBearer\s+[A-Za-z0-9._~+/-]+=*/gi, 'Bearer <redacted>')
    .replace(
      /([?&](?:access_token|api_key|auth|authorization|cookie|credential|password|secret|token)=)[^&\s]*/gi,
      '$1<redacted>',
    );
}

function boundedCaptureDiagnostic(value, maxBytes = 256 * 1024) {
  if (value === null || value === undefined) return null;
  let serialized;
  try {
    serialized = JSON.stringify(value, (key, nestedValue) => {
      if (/auth|cookie|credential|password|secret|token|api.?key/i.test(key)) {
        return '<redacted>';
      }
      return typeof nestedValue === 'string'
        ? redactFullStackDiagnosticText(nestedValue)
        : nestedValue;
    });
  } catch (error) {
    return { error: `Diagnostic value could not be serialized: ${error.message}` };
  }
  const bytes = Buffer.byteLength(serialized);
  if (bytes <= maxBytes) return JSON.parse(serialized);
  return {
    bytes,
    preview: Buffer.from(serialized).subarray(0, maxBytes).toString('utf8'),
    truncated: true,
  };
}

function captureDiagnostic(runDir, role) {
  const manifestPath = path.join(runDir, 'screenshots', role, 'manifest.json');
  const relativeManifestPath = path.relative(runDir, manifestPath).split(path.sep).join('/');
  if (!fs.existsSync(manifestPath)) return { available: false, role };
  try {
    const stat = fs.lstatSync(manifestPath);
    if (!stat.isFile() || stat.isSymbolicLink()) {
      throw new Error('capture manifest is not a non-symlink regular file');
    }
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    const captureResults = Array.isArray(manifest.results) ? manifest.results : [];
    const summarizeResult = (result) => ({
      category: classifyCaptureResult(result),
      diagnostics: boundedCaptureDiagnostic(result.diagnostics || null),
      error: result.error ? redactFullStackDiagnosticText(result.error).slice(0, 4096) : null,
      filename: result.filename || null,
      page: result.page || null,
      reason: result.reason ? redactFullStackDiagnosticText(result.reason).slice(0, 4096) : null,
      required: result.required === true,
      expectedProductRemoval: classifyCaptureResult(result) === 'expected_product_removal',
      status: result.status || 'unknown',
      viewport: result.viewport || null,
    });
    const incomplete = captureResults
      .filter(
        (result) =>
          result.status !== 'success' ||
          !['valid', 'expected_product_removal'].includes(result.captureValidity),
      )
      .map(summarizeResult);
    const expectedChanges = captureResults
      .filter(
        (result) =>
          result.status === 'success' && result.captureValidity === 'expected_product_removal',
      )
      .map(summarizeResult);
    const browserDiagnostics = boundedCaptureDiagnostic(
      manifest.browserDiagnostics || manifest.diagnostics?.browser || manifest.diagnostics || null,
    );
    return {
      available: true,
      browserDiagnostics,
      complete: manifest.complete === true,
      expectedChanges,
      fatalErrors: Array.isArray(manifest.fatalErrors)
        ? manifest.fatalErrors
            .slice(0, 100)
            .map((error) => redactFullStackDiagnosticText(error).slice(0, 4096))
        : [],
      incomplete,
      manifestPath: relativeManifestPath,
      role,
      summary: boundedCaptureDiagnostic(manifest.summary || null),
    };
  } catch (error) {
    return { available: false, error: error.message, manifestPath: relativeManifestPath, role };
  }
}

function classifyFullStackFailure(error, state, captures) {
  if (FULL_STACK_FAILURE_CATEGORIES.includes(error?.failureCategory)) {
    return error.failureCategory;
  }
  const captureCategories = captures.flatMap((capture) =>
    (capture.incomplete || []).map((result) => result.category).filter(Boolean),
  );
  for (const category of [
    'missing_fixture',
    'selector_drift',
    'api_incompatibility',
    'ui_rendering_failure',
    'expected_product_removal',
  ]) {
    if (captureCategories.includes(category)) return category;
  }
  return 'infrastructure_failure';
}

function writeFullStackDiagnosticArtifacts({ diagnostic, runDir }) {
  const jsonPath = path.join(runDir, 'full-stack-diagnostics.json');
  const htmlPath = path.join(runDir, 'full-stack-diagnostics.html');
  writeJson(jsonPath, diagnostic);
  const title = `Full-stack diagnostics: ${diagnostic.category}`;
  const json = escapeHtml(JSON.stringify(diagnostic, null, 2));
  fs.writeFileSync(
    htmlPath,
    `<!doctype html>\n<html lang="en"><head><meta charset="utf-8">` +
      `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'">` +
      `<meta name="viewport" content="width=device-width,initial-scale=1">` +
      `<title>${escapeHtml(title)}</title><style>` +
      'body{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;max-width:1200px;margin:2rem auto;padding:0 1rem;color:#17202a}' +
      'h1{font:600 1.4rem system-ui,sans-serif}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f4f6f7;border:1px solid #d5d8dc;padding:1rem}' +
      '</style></head><body>' +
      `<h1>${escapeHtml(title)}</h1><p>Phase: ${escapeHtml(diagnostic.phase)}</p>` +
      `<p>Machine-readable artifact: <code>${escapeHtml(path.basename(jsonPath))}</code></p>` +
      `<pre>${json}</pre></body></html>\n`,
  );
  return { htmlPath, jsonPath };
}

async function persistFullStackFailure({ error, run, services, state }) {
  const captures = ['base', 'head'].map((role) => captureDiagnostic(run.runDir, role));
  const stackDiagnostics = [];
  for (const stack of state.stacks) {
    if (typeof stack.collectDiagnostics !== 'function') {
      stackDiagnostics.push({
        clusterName: stack.clusterName,
        collected: false,
        reason: 'diagnostic_collection_unsupported',
        role: stack.role,
      });
      continue;
    }
    try {
      stackDiagnostics.push(
        await stack.collectDiagnostics({
          artifactRoot: run.runDir,
          outputDir: path.join(run.runDir, 'diagnostics', stack.role),
        }),
      );
    } catch (collectionError) {
      stackDiagnostics.push({
        clusterName: stack.clusterName,
        collected: false,
        error: collectionError.message,
        reason: 'diagnostic_collection_failed',
        role: stack.role,
      });
    }
  }
  const category = classifyFullStackFailure(error, state, captures);
  const diagnostic = {
    allowedCaptureValidities: FULL_STACK_CAPTURE_VALIDITIES,
    captures,
    category,
    captureValidity: category,
    complete: false,
    error: {
      details: boundedCaptureDiagnostic(error?.failureDetails || null),
      message: redactFullStackDiagnosticText(
        error instanceof Error ? error.message : String(error),
      ).slice(0, 16 * 1024),
      name: error instanceof Error ? error.name : 'Error',
    },
    mode: 'isolated-full-stack',
    phase: state.phase,
    runId: run.runId,
    schemaVersion: FULL_STACK_DIAGNOSTIC_SCHEMA_VERSION,
    stacks: stackDiagnostics,
    status: 'failed',
    timestamp: new Date().toISOString(),
  };
  const paths = await services.writeFullStackDiagnosticArtifacts({
    diagnostic,
    runDir: run.runDir,
  });
  state.diagnosticsPersisted = true;
  log(`Full-stack diagnostics: ${paths.htmlPath}`);
  return paths;
}

function pullRequestHeadSha(repository, prNumber) {
  const repositoryUrl = `https://github.com/${repository}.git`;
  const ref = `refs/pull/${prNumber}/head`;
  const output = gitOutput(['ls-remote', '--exit-code', repositoryUrl, ref]);
  const [sha, returnedRef, ...extra] = output.split(/\s+/);
  if (extra.length > 0 || returnedRef !== ref || !/^[0-9a-f]{40,64}$/i.test(sha || '')) {
    throw new Error(`Could not verify ${repository}#${prNumber} head revision.`);
  }
  return sha.toLowerCase();
}

function verifyPullRequestHead(repository, prNumber, expectedSha) {
  const currentSha = pullRequestHeadSha(repository, prNumber);
  if (currentSha !== expectedSha.toLowerCase()) {
    throw new Error(
      `Refusing to publish a stale report: ${repository}#${prNumber} is now ${currentSha}, expected ${expectedSha}.`,
    );
  }
  return currentSha;
}

function captureArguments(baseUrl, outputDir, label, seedManifestPath, provenance = {}) {
  if (!Object.values(SEMANTIC_ID_NORMALIZATION_MODES).includes(provenance.normalizationMode)) {
    throw new Error('Capture arguments require an explicit semantic ID normalization mode.');
  }
  const args = [
    path.join(SCRIPT_DIR, 'capture-screenshots.js'),
    '--base-url',
    baseUrl,
    '--output',
    outputDir,
    '--label',
    label,
    '--normalization-mode',
    provenance.normalizationMode,
    '--seed-manifest',
    seedManifestPath,
  ];
  if (provenance.revisionRole) {
    args.push('--revision-role', provenance.revisionRole);
  }
  if (provenance.semanticManifestPath) {
    args.push('--semantic-manifest', provenance.semanticManifestPath);
  }
  if (provenance.sourceProvenancePath) {
    args.push('--source-provenance', provenance.sourceProvenancePath);
  }
  return args;
}

function fullCaptureEnvironment(options, env = process.env) {
  const captureEnvironment = {
    ...env,
    UI_SMOKE_VIEWPORTS: options.viewports,
  };
  delete captureEnvironment.UI_SMOKE_PAGES;
  return captureEnvironment;
}

async function capturePair({
  baseUrl,
  headUrl,
  screenshotsDir,
  labels,
  options,
  baseSeedManifestPath,
  headSeedManifestPath,
  semanticManifestPath,
  normalizationMode,
  seedManifestPath,
  sourceProvenancePath,
  runChildImpl = runChild,
  scenarioCatalog = null,
  writeScenarioConfig = null,
}) {
  const resolvedBaseSeedManifest = baseSeedManifestPath || seedManifestPath;
  const resolvedHeadSeedManifest = headSeedManifestPath || seedManifestPath;
  if (!resolvedBaseSeedManifest || !resolvedHeadSeedManifest) {
    throw new Error('Both base and head seed manifests are required for paired capture.');
  }
  if (!Object.values(SEMANTIC_ID_NORMALIZATION_MODES).includes(normalizationMode)) {
    throw new Error('Paired capture requires an explicit semantic ID normalization mode.');
  }
  if (
    normalizationMode === SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK &&
    (!semanticManifestPath || !sourceProvenancePath)
  ) {
    throw new Error('Semantic full-stack capture requires semantic and source provenance.');
  }
  if (
    normalizationMode === SEMANTIC_ID_NORMALIZATION_MODES.BROWSER_COMPATIBILITY &&
    (semanticManifestPath || sourceProvenancePath)
  ) {
    throw new Error('Browser-compatibility capture cannot accept semantic or source provenance.');
  }
  const captureEnvironment = fullCaptureEnvironment(options);
  const baseDir = path.join(screenshotsDir, 'base');
  const headDir = path.join(screenshotsDir, 'head');
  const [baseCapture, headCapture] = await Promise.all([
    runChildImpl(
      process.execPath,
      captureArguments(baseUrl, baseDir, labels.base, resolvedBaseSeedManifest, {
        normalizationMode,
        revisionRole: 'base',
        semanticManifestPath,
        sourceProvenancePath,
      }),
      {
        cwd: SCRIPT_DIR,
        env: captureEnvironment,
      },
    ),
    runChildImpl(
      process.execPath,
      captureArguments(headUrl, headDir, labels.head, resolvedHeadSeedManifest, {
        normalizationMode,
        revisionRole: 'head',
        semanticManifestPath,
        sourceProvenancePath,
      }),
      {
        cwd: SCRIPT_DIR,
        env: captureEnvironment,
      },
    ),
  ]);

  const scenarioConfigPath = path.join(screenshotsDir, 'scenario-config.json');
  const writeConfig =
    writeScenarioConfig || require('./generate-comparison').writeBoundScenarioConfig;
  const resolvedScenarioCatalog =
    scenarioCatalog || require('./semantic-capture-scenarios').getSemanticScenarioCatalog();
  writeConfig({
    baseDir,
    defaults: {
      diffThreshold: options.diffThreshold,
      failThreshold: options.failThreshold,
      looksSameTolerance: LOOKS_SAME_TOLERANCE,
    },
    headDir,
    expectedViewports: options.viewports,
    outputPath: scenarioConfigPath,
    policyPath: options.scenarioPolicyPath || null,
    scenarioCatalog: resolvedScenarioCatalog,
  });
  const comparisonDir = path.join(screenshotsDir, 'comparison');
  const comparison = await runChildImpl(
    process.execPath,
    [
      path.join(SCRIPT_DIR, 'generate-comparison.js'),
      '--main',
      baseDir,
      '--pr',
      headDir,
      '--output',
      comparisonDir,
      '--fail-threshold',
      String(options.failThreshold),
      '--diff-threshold',
      String(options.diffThreshold),
      '--looksame-tolerance',
      String(LOOKS_SAME_TOLERANCE),
      '--looksame-cluster-size',
      String(LOOKS_SAME_CLUSTER_SIZE),
      '--scenario-config',
      scenarioConfigPath,
    ],
    { cwd: SCRIPT_DIR },
  );

  return { baseCapture, comparison, comparisonDir, headCapture, scenarioConfigPath };
}

async function publishReport(options, comparisonDir) {
  if (!options.comment) {
    return true;
  }
  const targetPr = options.prNumber || options.displayPrNumber;
  const summaryPath = path.join(comparisonDir, 'summary.json');
  if (!fs.existsSync(summaryPath)) {
    log('Cannot publish the PR report because comparison/summary.json was not created.', 'error');
    return false;
  }
  const result = await runChild(
    process.execPath,
    [
      path.join(SCRIPT_DIR, 'upload-to-pr.js'),
      '--pr',
      targetPr,
      '--repo',
      options.repository,
      '--screenshots',
      comparisonDir,
    ],
    { cwd: SCRIPT_DIR },
  );
  return result.success;
}

function logChanges(changes) {
  log(`Resolved base: ${changes.baseRef}`);
  log(`Changed files: ${changes.changedFiles.length}`);
  log(`Frontend changed: ${changes.frontendChanged}`);
  log(`Frontend server changed: ${changes.serverChanged}`);
  log(`Backend changed: ${changes.backendChanged}`);
  log(`Manifests changed: ${changes.manifestsChanged}`);
  if (changes.components.length > 0) {
    log(`Backend images: ${changes.components.map((component) => component.name).join(', ')}`);
  }
}

function fetchedDependencyInputs(changedFiles) {
  return changedFiles.filter(
    (filename) =>
      /^frontend\/(?:(?:server|mock-backend)\/)?(?:package-lock|npm-shrinkwrap)\.json$/i.test(
        filename,
      ) || /(?:^|\/)\.(?:npmrc|corepack\.env)$/i.test(filename),
  );
}

function comparisonServices(overrides = {}) {
  return {
    addDetachedWorktree,
    assessUpgradeCapabilities,
    buildExternalFrontend,
    buildTrustedFrontend,
    capturePair,
    clusterManager,
    combineSemanticManifests,
    componentsForRevision,
    components: COMPONENTS,
    detectChanges,
    ensureComparisonRuntime,
    fetchPullRequest,
    fullSha,
    gitOutput,
    loadUpgradeAdapter,
    materializeTrustedHeadSnapshot,
    orchestrateUpgrade,
    publishReport,
    readUpgradeCapabilities,
    registerCleanup,
    renderRevisionManifestSources,
    repoRoot: REPO_ROOT,
    revisionBuildMetadata,
    revisionUsesMetadataService,
    resolveExactBaseRelease,
    resolvePublishedReleaseCommit,
    scriptDir: SCRIPT_DIR,
    seedData,
    shortSha,
    spawnManaged,
    stackConfiguration,
    waitForUrl,
    verifyPullRequestHead,
    validateTrustedHeadCheckout,
    validateSafeRemovedResources,
    validateUpgradeOperations,
    validateUpgradeRequest,
    writeUpgradeComparisonArtifacts,
    writeFullStackDiagnosticArtifacts,
    writeJson,
    ...overrides,
  };
}

function loadJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function portConflictError(conflicts) {
  const details = conflicts
    .map(
      (conflict) =>
        `${conflict.port} (${conflict.process || 'unknown'}, PID ${conflict.pid || 'unknown'})`,
    )
    .join(', ');
  return new Error(
    `Required local ports are already in use: ${details}. Stop those processes and retry.`,
  );
}

function startStaticProxy(services, { backendUrl, buildRoot, port }) {
  return services.spawnManaged(
    process.execPath,
    [
      path.join(services.scriptDir, 'proxy-server.js'),
      '--build',
      path.join(buildRoot, 'frontend', 'build'),
      '--port',
      String(port),
      '--backend',
      backendUrl,
    ],
    { cwd: services.scriptDir },
  );
}

async function validatePublicationHead(options, services, headRoot, expectedHeadSha) {
  if (!options.comment) return;
  if (!options.prNumber) {
    if (services.gitOutput(['status', '--porcelain'], headRoot)) {
      throw new Error('Refusing to publish a local comparison from a dirty working tree.');
    }
    const currentHeadSha = services.fullSha('HEAD', headRoot);
    if (currentHeadSha !== expectedHeadSha) {
      throw new Error(
        `Refusing to publish a stale local comparison: HEAD moved from ${expectedHeadSha} to ${currentHeadSha}.`,
      );
    }
  }
  await services.verifyPullRequestHead(
    options.repository,
    options.prNumber || options.displayPrNumber,
    expectedHeadSha,
  );
}

async function finishComparison(options, services, results, headRoot, expectedHeadSha) {
  await validatePublicationHead(options, services, headRoot, expectedHeadSha);
  const publicationSucceeded = await services.publishReport(options, results.comparisonDir);
  return (
    results.baseCapture.success &&
    results.headCapture.success &&
    results.comparison.success &&
    publicationSucceeded
  );
}

async function requireStackDestroyed(stack) {
  const destroyed = await stack.destroyCluster();
  if (!(destroyed === true || destroyed?.success === true)) {
    throw new Error(`Failed to delete isolated ${stack.role} cluster ${stack.clusterName}.`);
  }
}

function onceAsync(operation) {
  let pending = null;
  return (...args) => {
    if (!pending) pending = Promise.resolve().then(() => operation(...args));
    return pending;
  };
}

async function requireUpgradeEnvironmentCleaned(cleanupEnvironment, request) {
  const result = await cleanupEnvironment({ reason: 'runner-cleanup', request });
  if (!(result === true || result?.success === true)) {
    throw new Error('Upgrade adapter failed to clean its owned environment.');
  }
}

async function runUpgradeComparison({
  baseCommitSha,
  changes,
  expectedHeadSha,
  headRoot,
  options,
  run,
  services,
  sourceProvenance,
}) {
  const resultPath = path.join(run.runDir, 'upgrade-result.json');
  const request = {
    artifactRoot: run.runDir,
    baseRevision: `${changes.baseRef}@${baseCommitSha}`,
    headRevision: formatSourceRevision(sourceProvenance, expectedHeadSha),
    runId: run.runId,
    sourceProvenance,
  };
  let capabilities = { removedResources: [] };
  let capabilityAssessment = null;
  const persistSetupFailure = (error, phase, category) => {
    const serializedError = {
      category,
      message: error instanceof Error ? error.message : String(error),
      name: error instanceof Error ? error.name : 'Error',
    };
    const failure = {
      baseCaptured: false,
      captureValidity: CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE,
      complete: false,
      contractVersion: UPGRADE_CONTRACT_VERSION,
      error: serializedError,
      headCaptured: false,
      migration: capabilityAssessment || { requirement: MIGRATION_REQUIREMENT },
      mode: 'upgrade-in-place',
      phase,
      phaseHistory: [{ phase, status: 'failed' }],
      request,
    };
    try {
      services.writeJson(resultPath, failure);
    } catch (writeError) {
      log(
        `Upgrade setup failed and its result could not be persisted: ${serializedError.message}; ${writeError.message}`,
        'error',
      );
      return false;
    }
    log(`Upgrade comparison stopped at ${phase}: ${category}. Details: ${resultPath}`, 'error');
    return false;
  };
  try {
    capabilities = services.readUpgradeCapabilities(headRoot);
    capabilityAssessment = services.assessUpgradeCapabilities(capabilities);
  } catch (error) {
    return persistSetupFailure(error, UPGRADE_PHASES.CONFIGURATION_CHECK, 'configuration_failure');
  }
  let adapterOperations = {};
  let cleanupUpgradeEnvironment = null;
  let configurationValid = false;
  if (capabilityAssessment.available) {
    try {
      services.validateUpgradeRequest(request);
      services.validateSafeRemovedResources(capabilities.removedResources);
      configurationValid = true;
    } catch (error) {
      // The orchestrator records the configuration failure in upgrade-result.json below.
    }
  }

  if (configurationValid && capabilities.adapter) {
    let adapter;
    try {
      adapter = services.loadUpgradeAdapter(headRoot, capabilities);
    } catch (error) {
      return persistSetupFailure(error, UPGRADE_PHASES.LOAD_ADAPTER, 'configuration_failure');
    }
    try {
      adapterOperations = await adapter.createOperations(
        Object.freeze({
          baseCommitSha,
          baseRef: baseCommitSha,
          baseTagRef: changes.baseDisplayRef,
          baseVersion: changes.baseRef,
          expectedHeadSha,
          headRoot,
          options: Object.freeze({
            diffThreshold: options.diffThreshold,
            failThreshold: options.failThreshold,
            viewports: options.viewports,
          }),
          runDir: run.runDir,
          runId: run.runId,
          sourceProvenance,
          writeComparisonArtifacts: (artifactOptions) =>
            services.writeUpgradeComparisonArtifacts({
              ...artifactOptions,
              artifactRoot: run.runDir,
            }),
        }),
      );
    } catch (error) {
      return persistSetupFailure(error, UPGRADE_PHASES.CREATE_OPERATIONS, 'configuration_failure');
    }
    let operationsValid = false;
    try {
      services.validateUpgradeOperations(adapterOperations);
      operationsValid = true;
    } catch (error) {
      // The orchestrator records the operation contract failure below.
    }
    if (operationsValid) {
      try {
        await services.ensureComparisonRuntime();
      } catch (error) {
        return persistSetupFailure(
          error,
          UPGRADE_PHASES.RUNTIME_PREFLIGHT,
          CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE,
        );
      }
      const cleanupEnvironment = onceAsync(adapterOperations.cleanupEnvironment);
      adapterOperations = { ...adapterOperations, cleanupEnvironment };
      cleanupUpgradeEnvironment = () =>
        requireUpgradeEnvironmentCleaned(cleanupEnvironment, request);
      services.registerCleanup(`clean upgrade environment ${run.runId}`, cleanupUpgradeEnvironment);
    }
  }
  const adapterWriteResult = adapterOperations.writeResult;
  let result;
  let orchestrationError = null;
  let cleanupError = null;
  try {
    result = await services.orchestrateUpgrade({
      capabilities,
      operations: {
        ...adapterOperations,
        async writeResult(value) {
          let adapterError = null;
          if (typeof adapterWriteResult === 'function') {
            try {
              await adapterWriteResult(value);
            } catch (error) {
              adapterError = error;
            }
          }
          const persistedValue = adapterError
            ? createResultWriteFailure(value, adapterError)
            : value;
          try {
            services.writeJson(resultPath, persistedValue);
          } catch (localError) {
            const message = adapterError
              ? `Adapter and local result persistence failed: ${adapterError.message}; ${localError.message}`
              : `Local result persistence failed: ${localError.message}`;
            throw new Error(message);
          }
          if (adapterError) {
            const error = new Error(
              `Upgrade adapter result persistence failed: ${adapterError.message}`,
            );
            error.persistedResult = persistedValue;
            throw error;
          }
        },
      },
      removedResources: capabilities.removedResources,
      request,
    });
  } catch (error) {
    orchestrationError = error;
  } finally {
    if (cleanupUpgradeEnvironment) {
      try {
        await cleanupUpgradeEnvironment();
      } catch (error) {
        cleanupError = error;
      }
    }
  }

  if (cleanupError) {
    const serializedCleanupError = {
      message: cleanupError instanceof Error ? cleanupError.message : String(cleanupError),
      name: cleanupError instanceof Error ? cleanupError.name : 'Error',
    };
    result = {
      ...(result || {}),
      baseCaptured: result?.baseCaptured === true,
      captureValidity: CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE,
      cleanupError: serializedCleanupError,
      complete: false,
      contractVersion: result?.contractVersion || UPGRADE_CONTRACT_VERSION,
      error: {
        category: 'cleanup_failure',
        message: `Upgrade environment cleanup failed: ${serializedCleanupError.message}`,
        name: serializedCleanupError.name,
      },
      headCaptured: result?.headCaptured === true,
      migration: result?.migration || capabilityAssessment,
      mode: 'upgrade-in-place',
      phase: UPGRADE_PHASES.CLEANUP_ENVIRONMENT,
      phaseHistory: [
        ...(Array.isArray(result?.phaseHistory) ? result.phaseHistory : []),
        { phase: UPGRADE_PHASES.CLEANUP_ENVIRONMENT, status: 'failed' },
      ],
      request: result?.request || request,
    };
    try {
      services.writeJson(resultPath, result);
    } catch (writeError) {
      log(
        `Upgrade cleanup failed and the result could not be rewritten: ${serializedCleanupError.message}; ${writeError.message}`,
        'error',
      );
      return false;
    }
  } else if (orchestrationError) {
    return persistSetupFailure(
      orchestrationError,
      UPGRADE_PHASES.RUNTIME_PREFLIGHT,
      CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE,
    );
  }

  let persistedResult;
  try {
    const resultStat = fs.lstatSync(resultPath);
    if (!resultStat.isFile() || resultStat.isSymbolicLink()) {
      throw new Error('the result is not a non-symlink regular file');
    }
    persistedResult = JSON.parse(fs.readFileSync(resultPath, 'utf8'));
    if (!persistedResult || typeof persistedResult !== 'object' || Array.isArray(persistedResult)) {
      throw new Error('the result JSON is not an object');
    }
  } catch (error) {
    log(`Upgrade comparison result was not persisted safely: ${error.message}`, 'error');
    return false;
  }
  if (
    persistedResult.complete !== result.complete ||
    persistedResult.captureValidity !== result.captureValidity ||
    persistedResult.comparisonPassed !== result.comparisonPassed ||
    Boolean(persistedResult.resultWriteError) !== Boolean(result.resultWriteError) ||
    persistedResult.phase !== result.phase
  ) {
    log('Persisted upgrade result does not match the orchestrator result.', 'error');
    return false;
  }

  const succeeded =
    result.complete === true &&
    result.captureValidity === CAPTURE_VALIDITY.VALID &&
    result.comparisonPassed === true &&
    persistedResult.comparisonPassed === true &&
    !result.resultWriteError;
  if (succeeded) {
    log(`Upgrade comparison completed. Result: ${resultPath}`);
    return true;
  }
  log(
    `Upgrade comparison stopped at ${result.phase}: ${result.captureValidity}. Details: ${resultPath}`,
    'error',
  );
  return false;
}

async function runFullStackComparisonOrchestration({
  baseCommitSha,
  baseWorktree,
  changes,
  expectedHeadSha,
  headRoot,
  options,
  run,
  services,
  sourceProvenance,
  sourceProvenancePath,
  state,
}) {
  const manager = services.clusterManager;
  state.phase = 'configuration';
  if (typeof manager.createKindStack !== 'function') {
    throw new Error('The cluster manager does not support isolated revision stacks.');
  }

  // Use the exact overlay that will be deployed. Scanning every YAML file in the repository can
  // falsely discover workloads that are not part of this revision's platform-agnostic stack.
  const baseManifestSources = services.renderRevisionManifestSources(baseWorktree);
  const headManifestSources = services.renderRevisionManifestSources(headRoot);
  const baseConfiguration = services.stackConfiguration(run, 'base', baseCommitSha, {
    metadata: services.revisionUsesMetadataService(baseWorktree, baseManifestSources),
  });
  const headConfiguration = services.stackConfiguration(run, 'head', expectedHeadSha, {
    metadata: services.revisionUsesMetadataService(headRoot, headManifestSources),
  });
  const conflicts = await manager.checkPortAvailability([
    baseConfiguration.ports.frontendServer,
    headConfiguration.ports.frontendServer,
  ]);
  if (conflicts.length > 0) throw portConflictError(conflicts);

  const baseStack = manager.createKindStack(baseConfiguration);
  const headStack = manager.createKindStack(headConfiguration);
  state.stacks.push(baseStack, headStack);
  for (const stack of [baseStack, headStack]) {
    services.registerCleanup(`destroy isolated ${stack.role} cluster ${stack.clusterName}`, () =>
      requireStackDestroyed(stack),
    );
    services.registerCleanup(`stop isolated ${stack.role} stack processes`, () => stack.cleanup());
  }

  state.phase = 'image_preflight';
  const targetPlatform = baseStack.getDockerPlatform();
  const seedRuntimePreflight = baseStack.preflightSeedRuntimeImage({
    platform: targetPlatform,
  });
  if (seedRuntimePreflight.platform !== targetPlatform) {
    throw new Error('Seed runtime preflight did not preserve the selected Kind target platform.');
  }
  const releasePreflight = baseStack.preflightReleaseImages(baseWorktree, {
    expectedRelease: changes.baseRef,
    platform: targetPlatform,
  });
  if (releasePreflight.platform !== targetPlatform) {
    throw new Error('Release image preflight did not preserve the selected Kind target platform.');
  }

  const headDependencyPreflight = headStack.preflightThirdPartyImages(headRoot, {
    platform: targetPlatform,
  });
  if (headDependencyPreflight.platform !== targetPlatform) {
    throw new Error('Head image preflight did not preserve the selected Kind target platform.');
  }

  const headComponents = services.componentsForRevision(
    headRoot,
    services.components,
    headManifestSources,
  );
  const headBuildMetadata = headComponents.some(
    (component) => Object.keys(component.buildArgs || {}).length > 0,
  )
    ? services.revisionBuildMetadata(headRoot, expectedHeadSha)
    : undefined;
  state.phase = 'head_image_build';
  const headImageOverrides =
    headComponents.length > 0
      ? await headStack.buildComponentImages(headComponents, headRoot, {
          buildMetadata: headBuildMetadata,
          load: false,
          platform: targetPlatform,
          tagSuffix: `${run.runId}-head`,
        })
      : { deployments: [], images: {}, runtimeEnvironment: {} };

  state.phase = 'cluster_creation';
  await Promise.all([baseStack.createCluster(), headStack.createCluster()]);
  for (const stack of [baseStack, headStack]) {
    const actualPlatform = stack.getClusterPlatform();
    if (actualPlatform !== targetPlatform) {
      throw new Error(
        `Kind cluster ${stack.clusterName} uses ${actualPlatform}, but images were preflighted for ${targetPlatform}.`,
      );
    }
  }
  // Establish the exact release stack before starting any head workload. All reviewed head images
  // were already built for the validated target platform while no cluster existed.
  state.phase = 'base_deployment';
  await baseStack.deployRevision(baseWorktree, {
    expectedRelease: changes.baseRef,
    platform: targetPlatform,
  });
  if (Object.keys(headImageOverrides.images).length > 0) {
    headStack.loadImageOverrides(headImageOverrides, targetPlatform);
  }
  state.phase = 'head_deployment';
  await headStack.deployRevision(headRoot, {
    imageOverrides: headImageOverrides,
    platform: targetPlatform,
    requireLocalFirstParty: true,
    tagSuffix: `${run.runId}-head`,
  });
  state.phase = 'ui_readiness';
  const [[baseUiForward], [headUiForward]] = await Promise.all([
    baseStack.ensureDeployedUiPortForwarding(),
    headStack.ensureDeployedUiPortForwarding(),
  ]);
  const baseUrl = baseStack.deployedUiUrl;
  const headUrl = headStack.deployedUiUrl;
  await Promise.all([
    services.waitForUrl(baseUrl, baseUiForward),
    services.waitForUrl(headUrl, headUiForward),
  ]);

  state.phase = 'fixture_seeding';
  const baseSeedManifestPath = path.join(run.runDir, 'seed', 'base.json');
  const headSeedManifestPath = path.join(run.runDir, 'seed', 'head.json');
  const [baseSeed, headSeed] = await Promise.all([
    services.seedData({
      apiBase: baseUrl,
      manifestPath: baseSeedManifestPath,
    }),
    services.seedData({
      apiBase: headUrl,
      manifestPath: headSeedManifestPath,
    }),
  ]);
  if (!baseSeed.success || !headSeed.success) {
    const failures = [
      !baseSeed.success ? `base: ${baseSeed.error || 'unknown error'}` : null,
      !headSeed.success ? `head: ${headSeed.error || 'unknown error'}` : null,
    ].filter(Boolean);
    const seedCategories = [baseSeed, headSeed]
      .filter((seed) => !seed.success)
      .map(seedFailureCategory);
    const category = seedCategories.includes('api_incompatibility')
      ? 'api_incompatibility'
      : seedCategories.includes('missing_fixture')
        ? 'missing_fixture'
        : 'seed_failure';
    throw categorizedFullStackError(
      category,
      `Revision-aware fixture seeding failed (${failures.join('; ')}).`,
      { base: baseSeed, head: headSeed },
    );
  }

  state.phase = 'fixture_validation';
  const baseSeedManifest = loadJson(baseSeedManifestPath);
  const headSeedManifest = loadJson(headSeedManifestPath);
  for (const [role, manifest, configuration] of [
    ['base', baseSeedManifest, baseConfiguration],
    ['head', headSeedManifest, headConfiguration],
  ]) {
    if (manifest.semantic?.validation?.valid !== true) {
      const errors = manifest.semantic?.validation?.errors || ['semantic validation was absent'];
      throw categorizedFullStackError(
        'missing_fixture',
        `${role} semantic fixture validation failed: ${errors.join('; ')}`,
      );
    }
    const expectedFlavor =
      configuration.ports.metadata === null ? 'native-task-artifact' : 'legacy-mlmd';
    if (manifest.semantic.revisionFlavor !== expectedFlavor) {
      throw categorizedFullStackError(
        'missing_fixture',
        `${role} revision data model mismatch: expected ${expectedFlavor}, received ${manifest.semantic.revisionFlavor || 'unknown'}.`,
      );
    }
  }
  const semanticManifest = services.combineSemanticManifests(
    {
      base: baseSeedManifest,
      head: headSeedManifest,
    },
    {
      revisions: {
        base: { commit: baseCommitSha, ref: changes.baseDisplayRef || changes.baseRef },
        head: {
          commit: expectedHeadSha,
          ref: 'HEAD',
          sourceFingerprint: sourceProvenance?.fingerprint || null,
          tree: sourceProvenance?.revision?.tree || null,
        },
      },
    },
  );
  const semanticManifestPath = path.join(run.runDir, 'semantic-fixtures.json');
  services.writeJson(semanticManifestPath, semanticManifest);

  const dirty = services.gitOutput(['status', '--porcelain'], headRoot) ? '+dirty' : '';
  const displayNumber = options.displayPrNumber;
  const snapshotLabel = sourceFingerprintLabel(sourceProvenance);
  const headIdentity = `${expectedHeadSha}${dirty}${snapshotLabel ? `; ${snapshotLabel}` : ''}`;
  const headLabel = displayNumber
    ? `PR #${displayNumber} (${headIdentity}) [isolated full stack]`
    : `HEAD (${headIdentity}) [isolated full stack]`;
  state.phase = 'capture';
  const results = await services.capturePair({
    baseSeedManifestPath,
    baseUrl,
    headSeedManifestPath,
    headUrl,
    labels: {
      base: `base: ${changes.baseRef} (${baseCommitSha}) [isolated full stack]`,
      head: headLabel,
    },
    options,
    screenshotsDir: path.join(run.runDir, 'screenshots'),
    semanticManifestPath,
    normalizationMode: SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK,
    sourceProvenancePath,
  });
  state.captureResults = results;
  state.phase = 'comparison';
  const success = await finishComparison(options, services, results, headRoot, expectedHeadSha);

  if (!results.baseCapture.success || !results.headCapture.success) {
    state.phase = 'capture';
    await persistFullStackFailure({
      error: new Error('One or more full-stack captures were incomplete.'),
      run,
      services,
      state,
    });
  }

  state.phase = 'cleanup';
  await Promise.all([baseStack.cleanup(), headStack.cleanup()]);
  await Promise.all([requireStackDestroyed(baseStack), requireStackDestroyed(headStack)]);

  log(`Semantic fixture map: ${semanticManifestPath}`);
  log(`Run artifacts: ${run.runDir}`);
  log(`Comparison report: ${results.comparisonDir}`);
  state.phase = 'complete';
  return success;
}

async function runFullStackComparison(parameters) {
  const state = {
    captureResults: null,
    diagnosticsPersisted: false,
    phase: 'configuration',
    stacks: [],
  };
  try {
    return await runFullStackComparisonOrchestration({ ...parameters, state });
  } catch (error) {
    if (!state.diagnosticsPersisted) {
      try {
        await persistFullStackFailure({
          error,
          run: parameters.run,
          services: parameters.services,
          state,
        });
      } catch (persistenceError) {
        throw new AggregateError(
          [error, persistenceError],
          `Full-stack comparison failed and diagnostics could not be persisted: ${error.message}`,
        );
      }
    }
    throw error;
  }
}

async function runComparison(options, run, overrides = {}) {
  const services = comparisonServices(overrides);
  const managedCluster = services.clusterManager;
  const screenshotsDir = path.join(run.runDir, 'screenshots');
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');
  const headWorktree = path.join(run.runDir, 'worktrees', 'head');
  const seedManifestPath = path.join(run.runDir, 'seed-manifest.json');
  fs.mkdirSync(path.dirname(baseWorktree), { recursive: true });

  let headRef = 'HEAD';
  let headRoot = services.repoRoot;
  let comparisonRoot = services.repoRoot;
  let sourceProvenance = null;
  let sourceProvenancePath = null;
  if (options.prNumber) {
    headRef = services.fetchPullRequest(options.repository, options.prNumber, run.runId);
  } else if (options.headCheckout) {
    headRoot = services.validateTrustedHeadCheckout(options.headCheckout, {
      git: services.gitOutput,
      repositoryRoot: services.repoRoot,
    });
    comparisonRoot = headRoot;
  }

  const exactBaseRelease =
    options.fullStack || options.upgrade
      ? services.resolveExactBaseRelease(options.compareRef, {
          cwd: comparisonRoot,
          git: services.gitOutput,
          resolveCommit: services.fullSha,
          resolvePublishedCommit: services.resolvePublishedReleaseCommit,
        })
      : null;

  if (options.headCheckout) {
    const selectedHeadRoot = headRoot;
    sourceProvenance = services.materializeTrustedHeadSnapshot(selectedHeadRoot, headWorktree, {
      addWorktree: services.addDetachedWorktree,
      git: services.gitOutput,
    });
    sourceProvenancePath = path.join(run.runDir, 'source-provenance.json');
    services.writeJson(sourceProvenancePath, sourceProvenance);
    headRoot = headWorktree;
    comparisonRoot = headWorktree;
    log(`Trusted head snapshot: ${sourceFingerprintLabel(sourceProvenance)}`);
  }

  const detectedChanges = services.detectChanges(
    exactBaseRelease ? exactBaseRelease.commit : options.compareRef,
    headRef,
    {
      cwd: comparisonRoot,
      includeWorkingTree: !options.prNumber,
    },
  );
  const baseCommitSha = exactBaseRelease
    ? exactBaseRelease.commit
    : services.fullSha(detectedChanges.baseRef, comparisonRoot);
  const changes = exactBaseRelease
    ? {
        ...detectedChanges,
        baseDisplayRef: exactBaseRelease.tagRef,
        baseRef: exactBaseRelease.version,
      }
    : detectedChanges;
  logChanges(changes);
  const expectedHeadSha = services.fullSha(headRef, comparisonRoot);
  if (sourceProvenance && sourceProvenance.revision.commit !== expectedHeadSha) {
    throw new Error('Trusted source provenance does not match the materialized snapshot HEAD.');
  }
  const dependencyInputs = options.prNumber ? fetchedDependencyInputs(changes.changedFiles) : [];
  if (dependencyInputs.length > 0) {
    throw new Error(
      `Fetched PR dependency sources cannot be installed safely (${dependencyInputs.join(
        ', ',
      )}). Review and check out the PR locally before comparing it.`,
    );
  }

  const ignoredSurfaces = [
    changes.serverChanged ? 'frontend/server' : null,
    changes.backendChanged ? 'backend' : null,
    changes.manifestsChanged ? 'manifests' : null,
  ].filter(Boolean);
  if (
    ignoredSurfaces.length > 0 &&
    !options.browserOnly &&
    !options.fullStack &&
    !options.upgrade
  ) {
    throw new Error(
      `This utility can attribute only browser/frontend regressions; the comparison changes ${ignoredSurfaces.join(
        ', ',
      )}. Use --full-stack with an explicitly trusted local checkout, or --browser-only to limit the result to browser compatibility.`,
    );
  }
  if (ignoredSurfaces.length > 0 && options.browserOnly) {
    log(
      `Browser-only comparison: ignoring changed ${ignoredSurfaces.join(', ')} and using the trusted base runtime for both captures.`,
      'error',
    );
  }

  if (options.upgrade) {
    return runUpgradeComparison({
      baseCommitSha,
      changes,
      expectedHeadSha,
      headRoot,
      options,
      run,
      services,
      sourceProvenance,
    });
  }

  services.ensureComparisonRuntime();

  if (options.prNumber) {
    services.addDetachedWorktree(headWorktree, headRef);
    headRoot = headWorktree;
  }
  services.addDetachedWorktree(
    baseWorktree,
    exactBaseRelease ? baseCommitSha : changes.baseRef,
    comparisonRoot,
  );

  if (options.fullStack) {
    return runFullStackComparison({
      baseCommitSha,
      baseWorktree,
      changes,
      expectedHeadSha,
      headRoot,
      options,
      run,
      services,
      sourceProvenance,
      sourceProvenancePath,
    });
  }

  if (options.prNumber) {
    log(`Building base frontend from ${changes.baseRef} in the pinned container...`);
    await services.buildExternalFrontend(baseWorktree);
    log(`Building PR #${options.prNumber} frontend in the pinned container...`);
    await services.buildExternalFrontend(headRoot);
  } else {
    log(`Building base frontend from ${changes.baseRef}...`);
    await services.buildTrustedFrontend(baseWorktree);
    log('Building local HEAD frontend...');
    await services.buildTrustedFrontend(headRoot);
  }

  const conflicts = await managedCluster.checkPortAvailability([
    BASE_PROXY_PORT,
    HEAD_PROXY_PORT,
    managedCluster.FRONTEND_SERVER_PORT,
    3002,
    9000,
    9090,
  ]);
  if (conflicts.length > 0) throw portConflictError(conflicts);

  await managedCluster.ensureCluster(baseWorktree);
  services.registerCleanup('stop cluster-managed local processes', () => managedCluster.cleanup());

  await managedCluster.ensurePortForwarding();
  await managedCluster.startFrontendServer(baseWorktree, { skipBuild: false });

  const seedResult = await services.seedData({
    apiBase: `http://127.0.0.1:${managedCluster.FRONTEND_SERVER_PORT}`,
    manifestPath: seedManifestPath,
  });
  if (!seedResult.success) {
    throw new Error(`Deterministic data seeding failed: ${seedResult.error || 'unknown error'}`);
  }

  const backendUrl = `http://127.0.0.1:${managedCluster.FRONTEND_SERVER_PORT}`;
  const baseUrl = `http://127.0.0.1:${BASE_PROXY_PORT}`;
  const headUrl = `http://127.0.0.1:${HEAD_PROXY_PORT}`;
  const baseProxy = startStaticProxy(services, {
    backendUrl,
    buildRoot: baseWorktree,
    port: BASE_PROXY_PORT,
  });
  const headProxy = startStaticProxy(services, {
    backendUrl,
    buildRoot: headRoot,
    port: HEAD_PROXY_PORT,
  });
  await Promise.all([
    services.waitForUrl(baseUrl, baseProxy),
    services.waitForUrl(headUrl, headProxy),
  ]);

  const baseLabel = `base: ${changes.baseRef} (${baseCommitSha})`;
  const dirty =
    !options.prNumber && services.gitOutput(['status', '--porcelain'], headRoot) ? '+dirty' : '';
  const displayNumber = options.prNumber || options.displayPrNumber;
  let headLabel = displayNumber
    ? `PR #${displayNumber} (${expectedHeadSha}${dirty})`
    : `HEAD (${expectedHeadSha}${dirty})`;
  if (ignoredSurfaces.length > 0) {
    headLabel += ` [browser-only; ignored ${ignoredSurfaces.join(', ')}]`;
  }
  const results = await services.capturePair({
    baseUrl,
    headUrl,
    labels: { base: baseLabel, head: headLabel },
    options,
    screenshotsDir,
    seedManifestPath,
    normalizationMode: SEMANTIC_ID_NORMALIZATION_MODES.BROWSER_COMPATIBILITY,
  });
  const success = await finishComparison(options, services, results, headRoot, expectedHeadSha);

  log(`Run artifacts: ${run.runDir}`);
  log(`Comparison report: ${results.comparisonDir}`);
  return success;
}

async function runCurrentOnly(options, run) {
  const ready = await requestSuccessful(options.url);
  if (!ready) {
    throw new Error(`The existing UI did not return a successful response: ${options.url}`);
  }
  const outputDir = path.join(run.runDir, 'screenshots', 'current');
  const result = await runChild(
    process.execPath,
    captureArguments(
      options.url,
      outputDir,
      options.displayPrNumber ? `PR #${options.displayPrNumber}` : 'current',
      path.join(run.runDir, 'seed-manifest.json'),
      { normalizationMode: SEMANTIC_ID_NORMALIZATION_MODES.BROWSER_COMPATIBILITY },
    ),
    {
      cwd: SCRIPT_DIR,
      env: {
        ...process.env,
        UI_SMOKE_PAGES: process.env.UI_SMOKE_PAGES || CURRENT_ONLY_PAGES,
        UI_SMOKE_VIEWPORTS: options.viewports,
      },
    },
  );
  log(`Run artifacts: ${run.runDir}`);
  return result.success;
}

async function main(argv = process.argv.slice(2)) {
  const options = parseCli(argv);
  verbose = options.verbose;
  if (options.help) {
    console.log(helpText());
    return true;
  }
  if (options.teardown) {
    checkPrerequisites({ cluster: true });
    if (!(await clusterManager.teardownCluster())) {
      throw new Error(`Failed to delete Kind cluster ${clusterManager.CLUSTER_NAME}.`);
    }
    return true;
  }

  if (options.compareRef) {
    checkPrerequisites({
      compare: true,
      packageManager: !(options.fullStack || options.upgrade),
    });
  } else {
    checkPrerequisites();
    ensureToolDependencies();
  }
  const run = createRunDirectory();
  registerCleanup('stop remaining child processes', async () => {
    await Promise.all([...children].map(terminateChild));
  });
  installSignalHandlers();

  try {
    return options.compareRef
      ? await runComparison(options, run)
      : await runCurrentOnly(options, run);
  } finally {
    await cleanup();
  }
}

if (require.main === module) {
  main()
    .then((success) => {
      process.exitCode = success ? 0 : 1;
    })
    .catch(async (error) => {
      log(error.message, 'error');
      try {
        await cleanup();
      } catch (cleanupError) {
        if (cleanupError !== error) log(cleanupError.message, 'error');
      }
      process.exitCode = 1;
    });
}

module.exports = {
  BASE_PROXY_PORT,
  FULL_STACK_CAPTURE_VALIDITIES,
  FULL_STACK_DIAGNOSTIC_SCHEMA_VERSION,
  FULL_STACK_FAILURE_CATEGORIES,
  HEAD_PROXY_PORT,
  LOOKS_SAME_CLUSTER_SIZE,
  LOOKS_SAME_TOLERANCE,
  NODE_IMAGE,
  NODE_VERSION,
  NPM_VERSION,
  PROCESS_TIMEOUT,
  assertNodeVersion,
  assertNpmVersion,
  capturePair,
  componentsForRevision,
  comparisonServices,
  createRunDirectory,
  executeCleanupActions,
  externalBuildArguments,
  externalInstallArguments,
  fetchedDependencyInputs,
  fullCaptureEnvironment,
  helpText,
  loadUpgradeAdapter,
  materializeTrustedHeadSnapshot,
  normalizeHttpUrl,
  parseCli,
  parsePercentage,
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
  stackConfiguration,
  terminateChild,
  validateExternalBuildArtifact,
  validateFullStackBaseRelease,
  validateRepository,
  validateTrustedHeadCheckout,
  validateViewports,
  verifyPullRequestHead,
  writeFullStackDiagnosticArtifacts,
};
