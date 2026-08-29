#!/usr/bin/env node

const crypto = require('crypto');
const fs = require('fs');
const http = require('http');
const https = require('https');
const path = require('path');
const { execFileSync, spawn, spawnSync } = require('child_process');
const { parseArgs } = require('util');

const clusterManager = require('./cluster-manager');
const { detectChanges } = require('./detect-changes');
const { seedData } = require('./seed-data');
const { validateRepository: validateGithubRepository } = require('./upload-to-pr');

const SCRIPT_DIR = __dirname;
const REPO_ROOT = path.resolve(SCRIPT_DIR, '../../..');
const STATE_DIR = path.join(REPO_ROOT, '.ui-smoke-test');
const DEFAULT_REPOSITORY = 'kubeflow/pipelines';
const BASE_PROXY_PORT = 4001;
const HEAD_PROXY_PORT = 4002;
const NODE_VERSION = '24.14.0';
const NPM_VERSION = '11.17.0';
const NODE_IMAGE = `node:${NODE_VERSION}-bookworm`;
const PROCESS_TIMEOUT = 10 * 60 * 1000;
const LOOKS_SAME_TOLERANCE = 2.3;
const LOOKS_SAME_CLUSTER_SIZE = 8;
const EXTERNAL_TOOL_CACHE = '.ui-smoke-tool-cache';
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
  --viewports <WxH,...>           Capture viewports (default: 1280x800)
  --fail-threshold <percent>      Fail above this visual-difference percentage (default: 0)
  --diff-threshold <percent>      Draw diff markers above this percentage (default: 0)
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
    help: values.help,
    prNumber: validatePullRequestNumber(values.pr, '--pr'),
    displayPrNumber: validatePullRequestNumber(values['pr-number'], '--pr-number'),
    repository: validateRepository(values.repo),
    teardown: values.teardown,
    trustPrCode: values['trust-pr-code'],
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
    const output = execFileSync(command, args, {
      cwd: options.cwd,
      encoding: 'utf8',
      env: options.env || process.env,
      maxBuffer: 20 * 1024 * 1024,
      stdio: options.stdio || 'pipe',
      timeout: options.timeout || PROCESS_TIMEOUT,
    });
    return typeof output === 'string' ? output.trim() : '';
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

function checkPrerequisites({ cluster = false, compare = false } = {}) {
  assertNodeVersion();
  assertNpmVersion(execute('npm', ['--version']));
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

async function terminateChild(child) {
  if (!child || child.exitCode !== null || child.signalCode !== null) {
    return;
  }
  await new Promise((resolve) => {
    let settled = false;
    const finish = () => {
      if (!settled) {
        settled = true;
        resolve();
      }
    };
    child.once('close', finish);
    child.kill('SIGTERM');
    const timer = setTimeout(() => {
      if (child.exitCode === null && child.signalCode === null) {
        child.kill('SIGKILL');
      }
      finish();
    }, 3000);
  });
}

async function cleanup() {
  if (cleanupPromise) {
    return cleanupPromise;
  }
  cleanupPromise = (async () => {
    for (const { action, label } of [...cleanupActions].reverse()) {
      try {
        await action();
      } catch (error) {
        log(`Cleanup failed (${label}): ${error.message}`, 'error');
      }
    }
    cleanupActions.length = 0;
  })();
  return cleanupPromise;
}

function installSignalHandlers() {
  for (const signal of ['SIGINT', 'SIGTERM']) {
    process.once(signal, () => {
      if (signalReceived) return;
      signalReceived = signal;
      log(`Received ${signal}; cleaning up...`, 'error');
      cleanup().finally(() => {
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
    let killTimer;
    const finish = (result) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutTimer);
      clearTimeout(killTimer);
      children.delete(child);
      resolve(result);
    };
    children.add(child);
    registerCleanup(`stop ${command} (${child.pid || 'not started'})`, () => terminateChild(child));
    child.once('error', (error) => {
      finish({ error, success: false, timedOut });
    });
    child.once('close', (code, childSignal) => {
      finish({ code, signal: childSignal, success: code === 0 && !timedOut, timedOut });
    });
    timeoutTimer = setTimeout(() => {
      timedOut = true;
      child.kill('SIGTERM');
      killTimer = setTimeout(() => {
        if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
        finish({
          error: new Error(`${formatCommand(command, args)} timed out after ${timeout}ms`),
          success: false,
          timedOut: true,
        });
      }, killTimeout);
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

function gitOutput(args, cwd = REPO_ROOT) {
  return execute('git', args, { cwd });
}

function addDetachedWorktree(target, gitRef) {
  gitOutput(['worktree', 'add', '--detach', target, gitRef]);
  registerCleanup(`remove worktree ${target}`, () => {
    execute('git', ['worktree', 'remove', '--force', target], { cwd: REPO_ROOT });
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

function shortSha(gitRef) {
  return gitOutput(['rev-parse', '--short=12', `${gitRef}^{commit}`]);
}

function fullSha(gitRef) {
  const sha = gitOutput(['rev-parse', `${gitRef}^{commit}`]);
  if (!/^[0-9a-f]{40,64}$/i.test(sha)) {
    throw new Error(`Could not resolve a full commit SHA for ${gitRef}.`);
  }
  return sha.toLowerCase();
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

function captureArguments(baseUrl, outputDir, label, seedManifestPath) {
  return [
    path.join(SCRIPT_DIR, 'capture-screenshots.js'),
    '--base-url',
    baseUrl,
    '--output',
    outputDir,
    '--label',
    label,
    '--seed-manifest',
    seedManifestPath,
  ];
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
  seedManifestPath,
}) {
  const captureEnvironment = fullCaptureEnvironment(options);
  const baseDir = path.join(screenshotsDir, 'base');
  const headDir = path.join(screenshotsDir, 'head');
  const [baseCapture, headCapture] = await Promise.all([
    runChild(process.execPath, captureArguments(baseUrl, baseDir, labels.base, seedManifestPath), {
      cwd: SCRIPT_DIR,
      env: captureEnvironment,
    }),
    runChild(process.execPath, captureArguments(headUrl, headDir, labels.head, seedManifestPath), {
      cwd: SCRIPT_DIR,
      env: captureEnvironment,
    }),
  ]);

  const comparisonDir = path.join(screenshotsDir, 'comparison');
  const comparison = await runChild(
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
    ],
    { cwd: SCRIPT_DIR },
  );

  return { baseCapture, comparison, comparisonDir, headCapture };
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
    buildExternalFrontend,
    buildTrustedFrontend,
    capturePair,
    clusterManager,
    detectChanges,
    fetchPullRequest,
    fullSha,
    gitOutput,
    publishReport,
    registerCleanup,
    repoRoot: REPO_ROOT,
    scriptDir: SCRIPT_DIR,
    seedData,
    shortSha,
    spawnManaged,
    waitForUrl,
    verifyPullRequestHead,
    ...overrides,
  };
}

async function runComparison(options, run, overrides = {}) {
  const services = comparisonServices(overrides);
  const managedCluster = services.clusterManager;
  const screenshotsDir = path.join(run.runDir, 'screenshots');
  const baseWorktree = path.join(run.runDir, 'worktrees', 'base');
  const headWorktree = path.join(run.runDir, 'worktrees', 'head');
  const seedManifestPath = path.join(run.runDir, 'seed-manifest.json');
  fs.mkdirSync(path.dirname(baseWorktree), { recursive: true });

  const conflicts = await managedCluster.checkPortAvailability([
    BASE_PROXY_PORT,
    HEAD_PROXY_PORT,
    managedCluster.FRONTEND_SERVER_PORT,
    3002,
    9000,
    9090,
  ]);
  if (conflicts.length > 0) {
    const details = conflicts
      .map(
        (conflict) =>
          `${conflict.port} (${conflict.process || 'unknown'}, PID ${conflict.pid || 'unknown'})`,
      )
      .join(', ');
    throw new Error(
      `Required local ports are already in use: ${details}. Stop those processes and retry.`,
    );
  }

  let headRef = 'HEAD';
  let headRoot = services.repoRoot;
  if (options.prNumber) {
    headRef = services.fetchPullRequest(options.repository, options.prNumber, run.runId);
  }

  const changes = services.detectChanges(options.compareRef, headRef, {
    cwd: services.repoRoot,
    includeWorkingTree: !options.prNumber,
  });
  logChanges(changes);
  const baseCommitSha = services.fullSha(changes.baseRef);
  const expectedHeadSha = services.fullSha(headRef);
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
  if (ignoredSurfaces.length > 0 && !options.browserOnly) {
    throw new Error(
      `This utility can attribute only browser/frontend regressions; the comparison changes ${ignoredSurfaces.join(
        ', ',
      )}. Use --browser-only to explicitly ignore those changes and compare only browser bundles.`,
    );
  }
  if (ignoredSurfaces.length > 0) {
    log(
      `Browser-only comparison: ignoring changed ${ignoredSurfaces.join(', ')} and using the trusted base runtime for both captures.`,
      'error',
    );
  }

  if (options.prNumber) {
    services.addDetachedWorktree(headWorktree, headRef);
    headRoot = headWorktree;
  }
  services.addDetachedWorktree(baseWorktree, changes.baseRef);

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
  const baseProxy = services.spawnManaged(
    process.execPath,
    [
      path.join(services.scriptDir, 'proxy-server.js'),
      '--build',
      path.join(baseWorktree, 'frontend', 'build'),
      '--port',
      String(BASE_PROXY_PORT),
      '--backend',
      backendUrl,
    ],
    { cwd: services.scriptDir },
  );
  const headProxy = services.spawnManaged(
    process.execPath,
    [
      path.join(services.scriptDir, 'proxy-server.js'),
      '--build',
      path.join(headRoot, 'frontend', 'build'),
      '--port',
      String(HEAD_PROXY_PORT),
      '--backend',
      backendUrl,
    ],
    { cwd: services.scriptDir },
  );
  await Promise.all([
    services.waitForUrl(baseUrl, baseProxy),
    services.waitForUrl(headUrl, headProxy),
  ]);

  const baseLabel = `base: ${changes.baseRef} (${baseCommitSha})`;
  const dirty = !options.prNumber && services.gitOutput(['status', '--porcelain']) ? '+dirty' : '';
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
  });
  if (options.comment) {
    if (!options.prNumber) {
      if (services.gitOutput(['status', '--porcelain'])) {
        throw new Error('Refusing to publish a local comparison from a dirty working tree.');
      }
      const currentHeadSha = services.fullSha('HEAD');
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
  const publicationSucceeded = await services.publishReport(options, results.comparisonDir);
  const success =
    results.baseCapture.success &&
    results.headCapture.success &&
    results.comparison.success &&
    publicationSucceeded;

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

  checkPrerequisites({
    cluster: Boolean(options.compareRef),
    compare: Boolean(options.compareRef),
  });
  ensureToolDependencies();
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
      await cleanup();
      process.exitCode = 1;
    });
}

module.exports = {
  BASE_PROXY_PORT,
  HEAD_PROXY_PORT,
  LOOKS_SAME_CLUSTER_SIZE,
  LOOKS_SAME_TOLERANCE,
  NODE_IMAGE,
  NODE_VERSION,
  NPM_VERSION,
  PROCESS_TIMEOUT,
  assertNodeVersion,
  assertNpmVersion,
  createRunDirectory,
  externalBuildArguments,
  externalInstallArguments,
  fetchedDependencyInputs,
  fullCaptureEnvironment,
  helpText,
  normalizeHttpUrl,
  parseCli,
  parsePercentage,
  requestSuccessful,
  runChild,
  runComparison,
  validateExternalBuildArtifact,
  validateRepository,
  validateViewports,
  verifyPullRequestHead,
};
