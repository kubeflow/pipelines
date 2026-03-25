#!/usr/bin/env node
/**
 * UI Smoke Test Runner
 *
 * Complete workflow for comparing UI screenshots between branches.
 *
 * Primary workflow — compare a PR against a base ref with live backend:
 *   node smoke-test-runner.js --compare master
 *   node smoke-test-runner.js --compare release
 *   node smoke-test-runner.js --compare 2.15.0
 *
 * This detects changes, ensures a Kind cluster, builds/deploys only
 * the backend components that changed, checks out the base ref via
 * git worktree, builds both frontends, captures screenshots, and
 * generates side-by-side comparisons.
 *
 * Test someone else's PR against a base ref:
 *   node smoke-test-runner.js --compare master --pr 12756
 *
 * Label local HEAD screenshots with a PR number:
 *   node smoke-test-runner.js --compare master --pr-number 12793
 *
 * Skip backend rebuild for frontend-only PRs:
 *   node smoke-test-runner.js --compare master --skip-backend
 *
 * Other workflows:
 *   node smoke-test-runner.js --current-only --use-existing --url http://localhost:3000
 *   node smoke-test-runner.js --current-only
 *   node smoke-test-runner.js --current-only --proxy --backend http://localhost:3000
 *   node smoke-test-runner.js --pr 12756
 *   node smoke-test-runner.js --teardown
 */

const { execSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const http = require('http');

// Import helpers
const clusterManager = require('./cluster-manager');
const { detectChanges } = require('./detect-changes');
const { seedData, checkHealth } = require('./seed-data');

// Parse arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};
const hasFlag = (name) => args.includes(`--${name}`);

// --compare <ref> is the primary workflow
const COMPARE_REF = getArg('compare', '');
const TEARDOWN = hasFlag('teardown');

// Legacy flags (still supported)
const PR_NUMBER = getArg('pr', '');
const PR_LABEL_NUMBER = getArg('pr-number', process.env.UI_SMOKE_PR_NUMBER || '');
const REPO = getArg('repo', 'kubeflow/pipelines');
const BASE_BRANCH = getArg('base', 'master');
const MODE = getArg('mode', 'auto'); // auto, cluster, mock, static
const CURRENT_ONLY = hasFlag('current-only');
const SKIP_BUILD = hasFlag('skip-build');
const SKIP_BACKEND = hasFlag('skip-backend');
const SKIP_UPLOAD = hasFlag('skip-upload');
const SKIP_SEED = hasFlag('skip-seed');
const SEED_DATA = hasFlag('seed-data');
const KEEP_SERVERS = hasFlag('keep-servers');
const VERBOSE = hasFlag('verbose');
const START_CLUSTER = hasFlag('start-cluster');
const USE_EXISTING = hasFlag('use-existing');
const EXISTING_URL = getArg('url', 'http://localhost:3000');
const USE_PROXY = hasFlag('proxy');
const PROXY_BACKEND = getArg('backend', 'http://localhost:3000');
const VIEWPORTS = getArg('viewports', process.env.UI_SMOKE_VIEWPORTS || '');
const FAIL_THRESHOLD = getArg('fail-threshold', process.env.UI_SMOKE_FAIL_THRESHOLD || '0');
const DIFF_THRESHOLD = getArg('diff-threshold', process.env.UI_SMOKE_DIFF_THRESHOLD || '0');

// Directories
const SCRIPT_DIR = __dirname;
const REPO_ROOT = path.resolve(SCRIPT_DIR, '../../..');
const WORK_DIR = path.join(REPO_ROOT, '.ui-smoke-test');
const MAIN_DIR = path.join(WORK_DIR, 'main');
const SCREENSHOTS_DIR = path.join(WORK_DIR, 'screenshots');
const WORKTREE_DIR = path.join(WORK_DIR, 'base');
const PR_WORKTREE_DIR = path.join(WORK_DIR, 'pr-branch');
const SEED_MANIFEST_PATH = path.join(WORK_DIR, 'seed-manifest.json');

// Ports
const PORT_MAIN = 4001;
const PORT_PR = 4002;

// Track spawned processes for cleanup
const processes = [];

// ────────────────────────────────────────────────────────────────
//  Resource tracker — guaranteed cleanup in LIFO order (Issue 5)
// ────────────────────────────────────────────────────────────────

const cleanupActions = [];
let cleanupRan = false;

/**
 * Register a cleanup action. Runs in LIFO order during cleanup.
 * Each action is { label, fn } where fn is a sync or async function.
 */
function registerCleanup(label, fn) {
  cleanupActions.push({ label, fn });
}

/**
 * Run all registered cleanup actions in reverse order.
 * Error-tolerant: logs failures but continues cleanup.
 */
async function runCleanupActions() {
  if (cleanupRan) return;
  cleanupRan = true;

  const reversed = [...cleanupActions].reverse();
  for (const { label, fn } of reversed) {
    try {
      log(`Cleanup: ${label}`, 'debug');
      await fn();
    } catch (e) {
      log(`Cleanup failed (${label}): ${e.message}`, 'warn');
    }
  }
  cleanupActions.length = 0;
}

// ────────────────────────────────────────────────────────────────
//  Step progress tracker (Issue 7)
// ────────────────────────────────────────────────────────────────

/**
 * Create a step tracker for numbered progress output.
 *
 * @param {number} totalSteps - Total number of steps in the workflow
 * @returns {{ step(msg: string): void, skip(msg: string): void }}
 */
function createStepTracker(totalSteps) {
  let current = 0;
  const bold = '\x1b[1m';
  const dim = '\x1b[2m';
  const green = '\x1b[32m';
  const yellow = '\x1b[33m';
  const reset = '\x1b[0m';

  return {
    step(msg) {
      current++;
      console.log(`${bold}${green}[${current}/${totalSteps}]${reset} ${msg}`);
    },
    skip(msg) {
      current++;
      console.log(`${bold}${yellow}[${current}/${totalSteps}]${reset} ${dim}${msg} (skipped)${reset}`);
    },
  };
}

// ────────────────────────────────────────────────────────────────
//  Logging and command execution
// ────────────────────────────────────────────────────────────────

function log(msg, type = 'info') {
  const colors = {
    info: '\x1b[32m',
    warn: '\x1b[33m',
    error: '\x1b[31m',
    debug: '\x1b[36m',
  };
  const reset = '\x1b[0m';
  const prefix = {
    info: '[INFO]',
    warn: '[WARN]',
    error: '[ERROR]',
    debug: '[DEBUG]',
  };

  if (type === 'debug' && !VERBOSE) return;

  console.log(`${colors[type]}${prefix[type]}${reset} ${msg}`);
}

function run(cmd, options = {}) {
  log(`Running: ${cmd}`, 'debug');
  try {
    const result = execSync(cmd, {
      encoding: 'utf8',
      stdio: 'pipe',
      ...options,
    });
    if (VERBOSE && result) {
      console.log(result);
    }
    return { success: true, output: result?.trim() || '' };
  } catch (error) {
    if (VERBOSE && error.stdout) {
      console.log(error.stdout);
    }
    if (VERBOSE && error.stderr) {
      console.error(error.stderr);
    }
    return { success: false, error: error.message, output: '' };
  }
}

function shellEscape(value) {
  return "'" + String(value).replace(/'/g, "'\\''") + "'";
}

function ensurePlaywright() {
  const result = run('npx playwright install --dry-run chromium 2>&1', { cwd: SCRIPT_DIR });
  if (result.success && /already installed/i.test(result.output)) {
    log('Playwright Chromium already installed', 'debug');
    return;
  }
  log('Installing Playwright Chromium...');
  run('npx playwright install chromium', { cwd: SCRIPT_DIR });
}

function resolveDisplayPrNumber() {
  if (PR_NUMBER) {
    return PR_NUMBER;
  }
  if (PR_LABEL_NUMBER) {
    return PR_LABEL_NUMBER;
  }

  // Best effort auto-detection for local HEAD comparisons.
  // Match current HEAD SHA to open PR head SHAs in the target repository.
  const headSha = run('git rev-parse HEAD', { cwd: REPO_ROOT });
  const sha = (headSha.output || '').trim();
  if (headSha.success && /^[0-9a-f]{7,40}$/i.test(sha)) {
    const autoPr = run(
      `gh pr list --repo ${REPO} --state open --limit 200 --json number,headRefOid --jq ".[] | select(.headRefOid==\\\"${sha}\\\") | .number"`,
      { cwd: REPO_ROOT },
    );
    if (autoPr.success) {
      const value = (autoPr.output || '')
        .split(/\r?\n/)
        .map(line => line.trim())
        .find(line => /^[0-9]+$/.test(line));
      if (value) {
        return value;
      }
    }
  }

  return '';
}

function spawnProcess(cmd, spawnArgs, options = {}) {
  log(`Spawning: ${cmd} ${spawnArgs.join(' ')}`, 'debug');
  const proc = spawn(cmd, spawnArgs, {
    stdio: VERBOSE ? 'inherit' : 'ignore',
    detached: false,
    ...options,
  });
  processes.push(proc);
  return proc;
}

function cleanup() {
  log('Cleaning up processes...');
  for (const proc of processes) {
    try {
      proc.kill('SIGTERM');
    } catch (e) {
      // Ignore
    }
  }
  clusterManager.cleanup();
}

process.on('SIGINT', async () => {
  await runCleanupActions();
  cleanup();
  process.exit(1);
});

process.on('exit', () => {
  if (!KEEP_SERVERS) {
    cleanup();
  }
});

// ────────────────────────────────────────────────────────────────
//  Prerequisite checking (Issue 4)
// ────────────────────────────────────────────────────────────────

/**
 * Check prerequisites before starting a workflow.
 * Fails fast with actionable install commands.
 *
 * @param {{ compare: boolean, pr: boolean }} options
 */
function checkPrerequisites({ compare = false, pr = false } = {}) {
  const errors = [];

  // Node version >= 18
  const nodeVersion = parseInt(process.versions.node.split('.')[0], 10);
  if (nodeVersion < 18) {
    errors.push(`Node.js >= 18 required (found ${process.versions.node}). Install: https://nodejs.org/`);
  }

  // git
  const gitResult = run('git --version');
  if (!gitResult.success) {
    errors.push('git is not installed. Install: https://git-scm.com/');
  }

  // git worktree support (needed for --compare)
  if (compare) {
    const worktreeResult = run('git worktree list');
    if (!worktreeResult.success) {
      errors.push('git worktree not supported. Update git to >= 2.5.');
    }
  }

  // --compare needs: kind, kubectl, docker
  if (compare) {
    if (!clusterManager.isKindInstalled()) {
      errors.push('kind is not installed. Install: https://kind.sigs.k8s.io/docs/user/quick-start/#installation');
    }
    if (!clusterManager.isKubectlInstalled()) {
      errors.push('kubectl is not installed. Install: https://kubernetes.io/docs/tasks/tools/');
    }
    if (!clusterManager.isDockerRunning()) {
      errors.push('Docker is not running. Start Docker Desktop or run: sudo systemctl start docker');
    }
  }

  // --pr needs: gh CLI
  if (pr) {
    const ghResult = run('gh --version');
    if (!ghResult.success) {
      errors.push('gh CLI is not installed (required for --pr). Install: https://cli.github.com/');
    }
  }

  // Auto-clean stale worktrees from previous failed runs
  if (compare) {
    for (const dir of [WORKTREE_DIR, PR_WORKTREE_DIR]) {
      if (fs.existsSync(dir)) {
        log(`Cleaning stale worktree: ${dir}`);
        run(`git worktree remove "${dir}" --force`, { cwd: REPO_ROOT });
        // If git worktree remove failed, try manual cleanup
        if (fs.existsSync(dir)) {
          try {
            fs.rmSync(dir, { recursive: true, force: true });
          } catch (e) {
            // Ignore
          }
        }
        run('git worktree prune', { cwd: REPO_ROOT });
      }
    }
  }

  if (errors.length > 0) {
    log('Prerequisite check failed:', 'error');
    for (const err of errors) {
      log(`  - ${err}`, 'error');
    }
    return false;
  }

  log('All prerequisites satisfied', 'debug');
  return true;
}

async function waitForServer(port, timeout = 30000) {
  const start = Date.now();

  while (Date.now() - start < timeout) {
    try {
      await new Promise((resolve, reject) => {
        const req = http.get(`http://localhost:${port}`, (res) => {
          resolve(true);
        });
        req.on('error', reject);
        req.setTimeout(1000, () => {
          req.destroy();
          reject(new Error('timeout'));
        });
      });
      return true;
    } catch (e) {
      await new Promise(r => setTimeout(r, 500));
    }
  }
  return false;
}

// ────────────────────────────────────────────────────────────────
//  --compare workflow (primary)
// ────────────────────────────────────────────────────────────────

/**
 * Full comparison workflow with step progress, --skip-backend,
 * --pr support, and guaranteed cleanup.
 *
 * Steps:
 *  1. Check port availability
 *  2. Detect changes
 *  3. Ensure Kind cluster
 *  4. Build/deploy changed backend components (or skip)
 *  5. Re-apply manifests (or skip)
 *  6. Set up port forwarding + frontend server
 *  7. Seed test data
 *  8. Fetch PR ref (if --pr)
 *  9. Check out base ref via git worktree, build its frontend
 * 10. Build PR frontend
 * 11. Serve both via proxy-server.js
 * 12. Capture screenshots + generate comparison
 */
async function runComparison(baseRef, displayPrNumber = '') {
  const totalSteps = 12;
  const tracker = createStepTracker(totalSteps);

  // Determine head ref: if --pr is given, we'll fetch and use it
  const usingPR = !!(COMPARE_REF && PR_NUMBER);
  const headLabel = displayPrNumber ? `PR #${displayPrNumber}` : 'HEAD';

  log(`Comparing ${headLabel} against ${baseRef}...`);

  // Step 1: Check port availability (Issue 6)
  tracker.step('Checking port availability...');
  const portsToCheck = [PORT_MAIN, PORT_PR, clusterManager.FRONTEND_SERVER_PORT, 3002, 9000, 9090];
  const conflicts = clusterManager.checkPortAvailability(portsToCheck);
  if (conflicts.length > 0) {
    log('Port conflicts detected:', 'error');
    for (const c of conflicts) {
      log(`  Port ${c.port} is in use by ${c.process} (PID ${c.pid})`, 'error');
      log(`  Fix: kill ${c.pid}  # or: lsof -i :${c.port}`, 'error');
    }
    return false;
  }

  // Step 2: Detect changes
  let prRef = 'HEAD';
  if (usingPR) {
    // Fetch the PR ref first so we can use it for change detection
    tracker.step(`Fetching PR #${PR_NUMBER} and detecting changes...`);

    const prNumberStr = String(PR_NUMBER);
    if (!/^\d+$/.test(prNumberStr)) {
      log(`Invalid PR number '${PR_NUMBER}'. Expected a numeric value.`, 'error');
      return false;
    }

    // Prefer "upstream" remote so contributors whose "origin" points to
    // their fork can still fetch PRs from the main repo.
    let prRemote = 'origin';
    const upstreamCheck = run('git remote get-url upstream', { cwd: REPO_ROOT });
    if (upstreamCheck.success) {
      prRemote = 'upstream';
    }

    const fetchResult = run(`git fetch ${prRemote} pull/${prNumberStr}/head:pr-${prNumberStr}`, { cwd: REPO_ROOT });
    if (!fetchResult.success) {
      log(`Failed to fetch PR #${PR_NUMBER}: ${fetchResult.error}`, 'error');
      return false;
    }
    prRef = `pr-${prNumberStr}`;
  } else {
    tracker.step('Detecting changes...');
  }

  let changes;
  try {
    changes = detectChanges(baseRef, prRef);
  } catch (e) {
    log(`Change detection failed: ${e.message}`, 'error');
    return false;
  }

  log(`Resolved base ref: ${changes.baseRef}`);
  log(`Head ref: ${changes.headRef}`);
  log(`Changed files: ${changes.changedFiles.length}`);
  log(`Frontend changed: ${changes.frontendChanged}`);
  log(`Server changed: ${changes.serverChanged}`);
  log(`Backend changed: ${changes.backendChanged} ${SKIP_BACKEND ? '(skip requested)' : ''}`);
  log(`Manifests changed: ${changes.manifestsChanged} ${SKIP_BACKEND ? '(skip requested)' : ''}`);

  if (changes.components.length > 0 && !SKIP_BACKEND) {
    log(`Backend components to rebuild: ${changes.components.map(c => c.name).join(', ')}`);
  }

  // Step 3: Ensure Kind cluster (always needed — frontend server proxies through it)
  tracker.step('Ensuring Kind cluster...');
  try {
    await clusterManager.ensureCluster(REPO_ROOT);
  } catch (e) {
    log(`Cluster setup failed: ${e.message}`, 'error');
    return false;
  }

  // Step 4: Build/deploy changed backend components
  // Auto-skips when change detection finds no backend changes.
  // --skip-backend forces skip even if detection says otherwise.
  if (!SKIP_BACKEND && changes.backendChanged) {
    tracker.step(`Building ${changes.components.length} backend component(s)...`);
    try {
      await clusterManager.buildAndDeployComponents(changes.components, REPO_ROOT);
    } catch (e) {
      log(`Component build/deploy failed: ${e.message}`, 'error');
      return false;
    }
  } else if (SKIP_BACKEND && changes.backendChanged) {
    tracker.skip(`Backend rebuild (skipped via --skip-backend, ${changes.components.length} component(s) detected)`);
  } else {
    tracker.skip('Backend rebuild (no changes detected)');
  }

  // Step 5: Re-apply manifests
  if (!SKIP_BACKEND && changes.manifestsChanged) {
    tracker.step('Re-applying kustomize manifests...');
    clusterManager.reapplyManifests(REPO_ROOT);
  } else if (SKIP_BACKEND && changes.manifestsChanged) {
    tracker.skip('Re-apply manifests (skipped via --skip-backend)');
  } else {
    tracker.skip('Re-apply manifests (no changes detected)');
  }

  // Step 6: Set up port forwarding + frontend server
  tracker.step('Setting up port forwarding and frontend server...');
  await clusterManager.ensurePortForwarding();
  try {
    await clusterManager.startFrontendServer(REPO_ROOT);
    // Register cleanup for ml-pipeline-ui restore (Issue 5)
    registerCleanup('Restore ml-pipeline-ui', () => {
      clusterManager.stopFrontendServer();
    });
  } catch (e) {
    log(`Frontend server failed: ${e.message}`, 'error');
    return false;
  }

  // Step 7: Seed data
  if (!SKIP_SEED) {
    tracker.step('Seeding test data...');
    try {
      await seedData({ skipIfExists: true });
    } catch (e) {
      log(`Data seeding failed: ${e.message}`, 'warn');
    }
  } else {
    tracker.skip('Seed test data');
  }

  // Step 8: Fetch PR code (if --pr in compare mode)
  fs.mkdirSync(WORK_DIR, { recursive: true });

  let prFrontendDir;
  let prBuildDir;

  if (usingPR) {
    tracker.step(`Setting up PR #${PR_NUMBER} via git worktree...`);

    const prWorktreeResult = run(`git worktree add "${PR_WORKTREE_DIR}" pr-${PR_NUMBER}`, { cwd: REPO_ROOT });
    if (!prWorktreeResult.success) {
      log(`Failed to create PR worktree: ${prWorktreeResult.error}`, 'error');
      return false;
    }
    // Register cleanup for PR worktree (Issue 5)
    registerCleanup('Remove PR worktree', () => {
      run(`git worktree remove "${PR_WORKTREE_DIR}" --force`, { cwd: REPO_ROOT });
    });

    prFrontendDir = path.join(PR_WORKTREE_DIR, 'frontend');
    prBuildDir = path.join(prFrontendDir, 'build');
  } else {
    tracker.skip(`Fetch PR code (using local HEAD)`);
    prFrontendDir = path.join(REPO_ROOT, 'frontend');
    prBuildDir = path.join(prFrontendDir, 'build');
  }

  // Step 9: Check out base ref via git worktree and build its frontend
  tracker.step(`Building base frontend (${changes.baseRef})...`);

  const worktreeResult = run(`git worktree add "${WORKTREE_DIR}" ${changes.baseRef}`, { cwd: REPO_ROOT });
  if (!worktreeResult.success) {
    log(`Failed to create worktree: ${worktreeResult.error}`, 'error');
    return false;
  }
  // Register cleanup for base worktree (Issue 5)
  registerCleanup('Remove base worktree', () => {
    run(`git worktree remove "${WORKTREE_DIR}" --force`, { cwd: REPO_ROOT });
  });

  const baseFrontendDir = path.join(WORKTREE_DIR, 'frontend');
  if (!SKIP_BUILD) {
    const installResult = run('npm ci', { cwd: baseFrontendDir, timeout: 120000 });
    if (!installResult.success) {
      log(`Base npm ci failed: ${installResult.error}`, 'error');
      return false;
    }
    const buildResult = run('npm run build', { cwd: baseFrontendDir, timeout: 120000 });
    if (!buildResult.success) {
      log(`Base build failed: ${buildResult.error}`, 'error');
      return false;
    }
  }
  const baseBuildDir = path.join(baseFrontendDir, 'build');

  // Step 10: Build PR frontend
  tracker.step(`Building ${headLabel} frontend...`);
  if (!SKIP_BUILD) {
    const installResult = run('npm ci', { cwd: prFrontendDir, timeout: 120000 });
    if (!installResult.success) {
      log(`${headLabel} npm ci failed: ${installResult.error}`, 'error');
      return false;
    }

    const buildResult = run('npm run build', { cwd: prFrontendDir, timeout: 120000 });
    if (!buildResult.success) {
      log(`${headLabel} build failed: ${buildResult.error}`, 'error');
      return false;
    }
  }

  // Step 11: Serve both via proxy-server.js → API proxied to localhost:3001
  tracker.step('Starting proxy servers...');
  const backendUrl = `http://localhost:${clusterManager.FRONTEND_SERVER_PORT}`;

  spawnProcess('node', [
    path.join(SCRIPT_DIR, 'proxy-server.js'),
    '--build', baseBuildDir,
    '--port', String(PORT_MAIN),
    '--backend', backendUrl,
  ], { cwd: SCRIPT_DIR });

  spawnProcess('node', [
    path.join(SCRIPT_DIR, 'proxy-server.js'),
    '--build', prBuildDir,
    '--port', String(PORT_PR),
    '--backend', backendUrl,
  ], { cwd: SCRIPT_DIR });

  const baseReady = await waitForServer(PORT_MAIN);
  const prReady = await waitForServer(PORT_PR);

  if (!baseReady || !prReady) {
    log('Failed to start one or more proxy servers', 'error');
    return false;
  }

  log(`Base server ready at http://localhost:${PORT_MAIN}`);
  log(`${headLabel} server ready at http://localhost:${PORT_PR}`);

  // Step 12: Capture screenshots + generate comparison
  tracker.step('Capturing screenshots and generating comparison...');
  const mainScreenshots = path.join(SCREENSHOTS_DIR, 'main');
  const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');

  // Resolve short SHAs for reproducibility labels
  const baseSha = run(`git rev-parse --short ${changes.baseRef}`, { cwd: REPO_ROOT });
  const headSha = run(`git rev-parse --short ${prRef}`, { cwd: REPO_ROOT });
  const baseShaStr = baseSha.success ? baseSha.output : '?';
  const headShaStr = headSha.success ? headSha.output : '?';

  const baseLabel = `base: ${changes.baseRef} (${baseShaStr})`;
  const prLabel = displayPrNumber
    ? `PR #${displayPrNumber} (${headShaStr})`
    : `HEAD (${headShaStr})`;

  const mainCaptured = await captureScreenshots(PORT_MAIN, mainScreenshots, baseLabel);
  const prCaptured = await captureScreenshots(PORT_PR, prScreenshots, prLabel);

  if (!mainCaptured || !prCaptured) {
    log('Failed to capture screenshots', 'error');
    // Continue to cleanup via registered actions
  }

  let comparisonGenerated = false;
  if (mainCaptured && prCaptured) {
    comparisonGenerated = await generateComparison();
  }

  if (comparisonGenerated && PR_NUMBER) {
    await uploadToPR();
  }

  // Cleanup runs via registered actions (worktrees, ml-pipeline-ui)
  await runCleanupActions();

  if (comparisonGenerated) {
    log(`\nComparison complete!`);
    log(`Screenshots: ${SCREENSHOTS_DIR}`);
    log(`Comparison: ${path.join(SCREENSHOTS_DIR, 'comparison')}`);
  }

  return mainCaptured && prCaptured && comparisonGenerated;
}

// ────────────────────────────────────────────────────────────────
//  Legacy workflows (still supported)
// ────────────────────────────────────────────────────────────────

/**
 * Determine which mode to use based on available infrastructure
 */
async function determineMode() {
  if (MODE !== 'auto') {
    log(`Using specified mode: ${MODE}`);
    return MODE;
  }

  log('Auto-detecting mode...');

  const status = await clusterManager.getClusterStatus();

  if (status.clusterRunning && status.kfpDeployed) {
    log('Kind cluster with KFP detected');
    return 'cluster';
  }

  if (status.kindInstalled && status.dockerRunning && START_CLUSTER) {
    log('Kind available, will start cluster');
    return 'cluster';
  }

  const mockBackendExists = fs.existsSync(path.join(REPO_ROOT, 'frontend/mock-backend'));
  if (mockBackendExists) {
    log('Mock backend available, using mock mode');
    return 'mock';
  }

  log('Falling back to static mode (no backend)');
  return 'static';
}

/**
 * Setup backend based on mode
 */
async function setupBackend(mode) {
  switch (mode) {
    case 'cluster':
      return await setupClusterBackend();
    case 'mock':
      return await setupMockBackend();
    case 'static':
    default:
      log('Static mode: no backend setup needed');
      return { success: true, mode: 'static' };
  }
}

/**
 * Setup Kind cluster backend
 */
async function setupClusterBackend() {
  log('Setting up Kind cluster backend...');

  const status = await clusterManager.getClusterStatus();

  if (!status.clusterRunning) {
    if (!START_CLUSTER) {
      log('Cluster not running. Use --start-cluster to create one.', 'warn');
      return { success: false, error: 'Cluster not running' };
    }

    log('Starting Kind cluster...');
    try {
      await clusterManager.ensureCluster(REPO_ROOT);
    } catch (e) {
      return { success: false, error: e.message };
    }
  }

  // Start port forwarding
  await clusterManager.ensurePortForwarding();

  // Wait for services
  await new Promise(r => setTimeout(r, 3000));

  // Verify API health
  const healthy = await checkHealth();
  if (!healthy) {
    log('API health check failed', 'warn');
  }

  return { success: true, mode: 'cluster', healthy };
}

/**
 * Setup mock backend
 */
async function setupMockBackend() {
  log('Mock mode: Production server lacks SPA routing, falling back to static mode', 'warn');
  log('For full UI testing with backend, use --compare or --mode cluster with Kind', 'warn');
  return { success: true, mode: 'static' };
}

async function checkoutAndBuild(branchRef, targetDir, label) {
  log(`Setting up ${label} at ${targetDir}...`);

  if (!fs.existsSync(targetDir)) {
    fs.mkdirSync(targetDir, { recursive: true });

    log(`Cloning ${branchRef} to ${targetDir}...`);
    run(`git clone --depth 1 --branch ${branchRef} --single-branch https://github.com/${REPO}.git "${targetDir}"`, {
      cwd: WORK_DIR,
    });
  }

  if (!SKIP_BUILD) {
    const frontendDir = path.join(targetDir, 'frontend');
    log(`Installing dependencies for ${label}...`);
    run('npm ci', { cwd: frontendDir });

    log(`Building ${label}...`);
    run('npm run build', { cwd: frontendDir });
  }

  return path.join(targetDir, 'frontend', 'build');
}

async function serveBuild(buildDir, port, label, mode = 'static') {
  log(`Starting server for ${label} on port ${port}...`);

  let proc;
  if (USE_PROXY) {
    log(`Using proxy server with backend: ${PROXY_BACKEND}`);
    proc = spawnProcess('node', [
      path.join(SCRIPT_DIR, 'proxy-server.js'),
      '--build', buildDir,
      '--port', String(port),
      '--backend', PROXY_BACKEND,
    ], {
      cwd: SCRIPT_DIR,
    });
  } else {
    proc = spawnProcess('npx', ['serve', '-s', buildDir, '-l', String(port)], {
      cwd: SCRIPT_DIR,
    });
  }

  const ready = await waitForServer(port);
  if (!ready) {
    log(`Server for ${label} failed to start`, 'error');
    return false;
  }

  log(`${label} server ready at http://localhost:${port}`);
  return true;
}

async function captureScreenshots(port, outputDir, label) {
  log(`Capturing screenshots for ${label}...`);
  fs.mkdirSync(outputDir, { recursive: true });

  const env = {
    ...process.env,
    UI_SMOKE_SEED_MANIFEST: SEED_MANIFEST_PATH,
  };
  if (VIEWPORTS) {
    env.UI_SMOKE_VIEWPORTS = VIEWPORTS;
  }

  const result = run(
    `node "${path.join(SCRIPT_DIR, 'capture-screenshots.js')}" --port ${port} --output "${outputDir}" --label ${shellEscape(label)} --seed-manifest ${shellEscape(SEED_MANIFEST_PATH)}`,
    { cwd: SCRIPT_DIR, env }
  );

  return result.success;
}

async function generateComparison() {
  log('Generating side-by-side comparisons...');

  const mainScreenshots = path.join(SCREENSHOTS_DIR, 'main');
  const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');
  const comparisonDir = path.join(SCREENSHOTS_DIR, 'comparison');

  let command =
    `node "${path.join(SCRIPT_DIR, 'generate-comparison.js')}" --main "${mainScreenshots}" --pr "${prScreenshots}" --output "${comparisonDir}"`;
  if (FAIL_THRESHOLD !== '') {
    command += ` --fail-threshold ${shellEscape(FAIL_THRESHOLD)}`;
  }
  if (DIFF_THRESHOLD !== '') {
    command += ` --diff-threshold ${shellEscape(DIFF_THRESHOLD)}`;
  }
  const result = run(command, { cwd: SCRIPT_DIR });

  return result.success;
}

async function uploadToPR() {
  if (SKIP_UPLOAD || !PR_NUMBER) {
    log('Skipping upload to PR');
    return true;
  }

  log(`Uploading results to PR #${PR_NUMBER}...`);

  const comparisonDir = path.join(SCREENSHOTS_DIR, 'comparison');

  const result = run(
    `node "${path.join(SCRIPT_DIR, 'upload-to-pr.js')}" --pr ${PR_NUMBER} --repo ${REPO} --screenshots "${comparisonDir}"`,
    { cwd: SCRIPT_DIR }
  );

  return result.success;
}

async function runCurrentOnly(mode) {
  log('Running in current-only mode');

  if (USE_EXISTING) {
    log(`Using existing server at ${EXISTING_URL}`);

    const existingPort = new URL(EXISTING_URL).port || 80;

    const reachable = await waitForServer(existingPort, 5000);
    if (!reachable) {
      log(`Server not reachable at ${EXISTING_URL}`, 'error');
      return false;
    }

    const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');
    const captured = await captureScreenshots(existingPort, prScreenshots, 'current');

    if (!captured) {
      log('Failed to capture screenshots', 'error');
      return false;
    }

    log(`Screenshots saved to: ${prScreenshots}`);
    return true;
  }

  const frontendDir = path.join(REPO_ROOT, 'frontend');
  const buildDir = path.join(frontendDir, 'build');

  if (!fs.existsSync(buildDir)) {
    log('No build found. Building frontend...', 'warn');
    run('npm ci', { cwd: frontendDir });
    run('npm run build', { cwd: frontendDir });
  }

  if (mode !== 'static') {
    const backendResult = await setupBackend(mode);
    if (!backendResult.success) {
      log(`Backend setup failed, falling back to static mode`, 'warn');
      mode = 'static';
    }
  }

  if ((SEED_DATA || !SKIP_SEED) && mode === 'cluster') {
    log('Seeding test data...');
    await seedData({ skipIfExists: true });
  }

  const serverStarted = await serveBuild(buildDir, PORT_PR, 'current', mode);
  if (!serverStarted) {
    log('Failed to start server', 'error');
    return false;
  }

  const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');
  const captured = await captureScreenshots(PORT_PR, prScreenshots, 'current');

  if (!captured) {
    log('Failed to capture screenshots', 'error');
    return false;
  }

  log(`Screenshots saved to: ${prScreenshots}`);
  return true;
}

async function runFullComparison(mode) {
  log(`Running full comparison: ${BASE_BRANCH} vs PR #${PR_NUMBER || 'current'}`);

  fs.mkdirSync(WORK_DIR, { recursive: true });

  let prBranch = '';
  if (PR_NUMBER) {
    log(`Fetching PR #${PR_NUMBER} info...`);
    const prInfo = run(`gh pr view ${PR_NUMBER} --repo ${REPO} --json headRefName --jq '.headRefName'`);
    if (prInfo.success && prInfo.output) {
      prBranch = prInfo.output.trim();
      log(`PR branch: ${prBranch}`);
    } else {
      log('Could not fetch PR info. Using current directory for PR build.', 'warn');
    }
  }

  const backendResult = await setupBackend(mode);
  if (!backendResult.success && mode !== 'static') {
    log(`Backend setup failed, falling back to static mode`, 'warn');
    mode = 'static';
  }

  if (!SKIP_SEED && mode === 'cluster') {
    log('Seeding test data...');
    await seedData({ skipIfExists: true });
  }

  log('Setting up main branch...');
  let mainBuildDir;
  try {
    mainBuildDir = await checkoutAndBuild(BASE_BRANCH, MAIN_DIR, 'main');
  } catch (e) {
    log(`Failed to setup main branch: ${e.message}`, 'error');
    return false;
  }

  let prBuildDir = path.join(REPO_ROOT, 'frontend', 'build');

  if (!SKIP_BUILD) {
    if (!fs.existsSync(prBuildDir)) {
      log('Building PR branch...');
      run('npm ci', { cwd: path.join(REPO_ROOT, 'frontend') });
      run('npm run build', { cwd: path.join(REPO_ROOT, 'frontend') });
    }
  }

  const mainServerStarted = await serveBuild(mainBuildDir, PORT_MAIN, 'main', 'static');
  const prServerStarted = await serveBuild(prBuildDir, PORT_PR, 'PR', 'static');

  if (!mainServerStarted || !prServerStarted) {
    log('Failed to start one or more servers', 'error');
    return false;
  }

  const mainScreenshots = path.join(SCREENSHOTS_DIR, 'main');
  const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');

  const mainCaptured = await captureScreenshots(PORT_MAIN, mainScreenshots, 'main');
  const prCaptured = await captureScreenshots(PORT_PR, prScreenshots, 'PR');

  if (!mainCaptured || !prCaptured) {
    log('Failed to capture screenshots', 'error');
    return false;
  }

  const comparisonGenerated = await generateComparison();
  if (!comparisonGenerated) {
    log('Failed to generate comparisons', 'error');
    return false;
  }

  const uploaded = await uploadToPR();
  if (!uploaded) {
    log('Failed to upload to PR', 'warn');
  }

  log(`\nUI smoke test complete!`);
  log(`Screenshots: ${SCREENSHOTS_DIR}`);

  return true;
}

// ────────────────────────────────────────────────────────────────
//  Main
// ────────────────────────────────────────────────────────────────

async function main() {
  console.log(`
╔══════════════════════════════════════════════════════════════╗
║           Kubeflow Pipelines UI Smoke Test                  ║
╚══════════════════════════════════════════════════════════════╝
`);

  // Handle --teardown
  if (TEARDOWN) {
    log('Tearing down Kind cluster...');
    clusterManager.teardownCluster();
    process.exit(0);
  }

  // Handle --compare <ref> (primary workflow)
  if (COMPARE_REF) {
    const usingPR = !!PR_NUMBER;
    const displayPrNumber = resolveDisplayPrNumber();
    const compareHeadLabel = displayPrNumber ? `PR #${displayPrNumber}` : 'HEAD';
    log(`Compare mode: ${compareHeadLabel} vs ${COMPARE_REF}`);
    if (SKIP_BACKEND) log('Backend rebuild: skipped (--skip-backend)');
    if (VIEWPORTS) log(`Viewports: ${VIEWPORTS}`);
    if (FAIL_THRESHOLD !== '') log(`Fail threshold: ${FAIL_THRESHOLD}%`);
    if (DIFF_THRESHOLD !== '') log(`Diff marker threshold: ${DIFF_THRESHOLD}%`);
    log(`Work directory: ${WORK_DIR}`);

    // Prerequisite check (Issue 4)
    if (!checkPrerequisites({ compare: true, pr: usingPR })) {
      process.exit(1);
    }

    // Ensure dependencies
    const nodeModulesPath = path.join(SCRIPT_DIR, 'node_modules');
    if (!fs.existsSync(nodeModulesPath)) {
      log('Installing smoke test dependencies...');
      run('npm ci', { cwd: SCRIPT_DIR });
    }

    ensurePlaywright();

    const success = await runComparison(COMPARE_REF, displayPrNumber);
    await runCleanupActions();
    cleanup();
    process.exit(success ? 0 : 1);
  }

  // Legacy mode
  const displayPrNumber = resolveDisplayPrNumber();
  log(`PR: ${displayPrNumber || 'N/A'}`);
  log(`Repository: ${REPO}`);
  log(`Base branch: ${BASE_BRANCH}`);
  if (VIEWPORTS) log(`Viewports: ${VIEWPORTS}`);
  if (FAIL_THRESHOLD !== '') log(`Fail threshold: ${FAIL_THRESHOLD}%`);
  if (DIFF_THRESHOLD !== '') log(`Diff marker threshold: ${DIFF_THRESHOLD}%`);
  log(`Work directory: ${WORK_DIR}`);

  // Ensure dependencies
  const nodeModulesPath = path.join(SCRIPT_DIR, 'node_modules');
  if (!fs.existsSync(nodeModulesPath)) {
    log('Installing smoke test dependencies...');
    run('npm install', { cwd: SCRIPT_DIR });
  }

  ensurePlaywright();

  const mode = await determineMode();
  log(`Mode: ${mode}`);

  let success;
  if (CURRENT_ONLY) {
    success = await runCurrentOnly(mode);
  } else {
    success = await runFullComparison(mode);
  }

  cleanup();

  process.exit(success ? 0 : 1);
}

main().catch(async (err) => {
  log(`Fatal error: ${err.message}`, 'error');
  await runCleanupActions();
  cleanup();
  process.exit(1);
});
