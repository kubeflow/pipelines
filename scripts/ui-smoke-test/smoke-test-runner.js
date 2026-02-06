#!/usr/bin/env node
/**
 * UI Smoke Test Runner
 *
 * Complete workflow for comparing UI screenshots between branches:
 * 1. Detects/starts Kind cluster with KFP backend (or falls back to mock/static mode)
 * 2. Seeds test data for realistic UI states
 * 3. Checks out main and PR branches
 * 4. Builds both
 * 5. Serves them with backend proxying
 * 6. Captures screenshots
 * 7. Generates side-by-side comparisons
 * 8. Uploads results to PR
 *
 * Usage:
 *   node smoke-test-runner.js --pr 12756                    # Full comparison with cluster
 *   node smoke-test-runner.js --pr 12756 --mode static      # Static serve (no backend)
 *   node smoke-test-runner.js --pr 12756 --mode mock        # Use mock backend
 *   node smoke-test-runner.js --current-only                # Just screenshot current build
 *   node smoke-test-runner.js --current-only --seed-data    # Screenshot with seeded data
 *   node smoke-test-runner.js --current-only --use-existing # Screenshot from running server
 *   node smoke-test-runner.js --current-only --use-existing --url http://localhost:3000
 *   node smoke-test-runner.js --proxy --backend http://localhost:3000  # Proxy API to real backend
 */

const { execSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const http = require('http');

// Import helpers
const clusterManager = require('./cluster-manager');
const { seedData, checkHealth } = require('./seed-data');

// Parse arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};
const hasFlag = (name) => args.includes(`--${name}`);

const PR_NUMBER = getArg('pr', '');
const REPO = getArg('repo', 'kubeflow/pipelines');
const BASE_BRANCH = getArg('base', 'master');
const MODE = getArg('mode', 'auto'); // auto, cluster, mock, static
const CURRENT_ONLY = hasFlag('current-only');
const SKIP_BUILD = hasFlag('skip-build');
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

// Directories
const SCRIPT_DIR = __dirname;
const REPO_ROOT = path.resolve(SCRIPT_DIR, '../..');
const WORK_DIR = path.join(REPO_ROOT, '.ui-smoke-test');
const MAIN_DIR = path.join(WORK_DIR, 'main');
const PR_DIR = path.join(WORK_DIR, 'pr');
const SCREENSHOTS_DIR = path.join(WORK_DIR, 'screenshots');

// Ports
const PORT_MAIN = 4001;
const PORT_PR = 4002;
const PROXY_PORT = 3001;

// Track spawned processes for cleanup
const processes = [];

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
    return { success: true, output: result || '' };
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

function spawnProcess(cmd, args, options = {}) {
  log(`Spawning: ${cmd} ${args.join(' ')}`, 'debug');
  const proc = spawn(cmd, args, {
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

process.on('SIGINT', () => {
  cleanup();
  process.exit(1);
});

process.on('exit', () => {
  if (!KEEP_SERVERS) {
    cleanup();
  }
});

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

/**
 * Determine which mode to use based on available infrastructure
 */
async function determineMode() {
  if (MODE !== 'auto') {
    log(`Using specified mode: ${MODE}`);
    return MODE;
  }

  log('Auto-detecting mode...');

  // Check for Kind cluster
  const status = await clusterManager.getClusterStatus();

  if (status.clusterRunning && status.kfpDeployed) {
    log('Kind cluster with KFP detected');
    return 'cluster';
  }

  if (status.kindInstalled && status.dockerRunning && START_CLUSTER) {
    log('Kind available, will start cluster');
    return 'cluster';
  }

  // Check if mock backend exists
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
    const started = await clusterManager.startCluster(REPO_ROOT);
    if (!started) {
      return { success: false, error: 'Failed to start cluster' };
    }
  }

  // Start port forwarding
  clusterManager.startPortForwarding();

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
 * Note: Mock mode falls back to static mode because the production server
 * doesn't have SPA routing for arbitrary paths. Use cluster mode for full testing.
 */
async function setupMockBackend() {
  log('Mock mode: Production server lacks SPA routing, falling back to static mode', 'warn');
  log('For full UI testing with backend, use --mode cluster with Kind', 'warn');
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
    // Use proxy server that forwards API calls to real backend
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
    // Use simple static server with SPA fallback
    // The -s flag enables single-page app mode (serves index.html for unknown routes)
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

  const result = run(
    `node "${path.join(SCRIPT_DIR, 'capture-screenshots.js')}" --port ${port} --output "${outputDir}" --label "${label}"`,
    { cwd: SCRIPT_DIR }
  );

  return result.success;
}

async function generateComparison() {
  log('Generating side-by-side comparisons...');

  const mainScreenshots = path.join(SCREENSHOTS_DIR, 'main');
  const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');
  const comparisonDir = path.join(SCREENSHOTS_DIR, 'comparison');

  const result = run(
    `node "${path.join(SCRIPT_DIR, 'generate-comparison.js')}" --main "${mainScreenshots}" --pr "${prScreenshots}" --output "${comparisonDir}"`,
    { cwd: SCRIPT_DIR }
  );

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

  // If using existing server, skip build and serve steps
  if (USE_EXISTING) {
    log(`Using existing server at ${EXISTING_URL}`);

    // Parse port from URL
    const existingPort = new URL(EXISTING_URL).port || 80;

    // Verify server is reachable
    const reachable = await waitForServer(existingPort, 5000);
    if (!reachable) {
      log(`Server not reachable at ${EXISTING_URL}`, 'error');
      return false;
    }

    // Capture screenshots from existing server
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

  // Check if build exists
  if (!fs.existsSync(buildDir)) {
    log('No build found. Building frontend...', 'warn');
    run('npm ci', { cwd: frontendDir });
    run('npm run build', { cwd: frontendDir });
  }

  // Setup backend if not static mode
  if (mode !== 'static') {
    const backendResult = await setupBackend(mode);
    if (!backendResult.success) {
      log(`Backend setup failed, falling back to static mode`, 'warn');
      mode = 'static';
    }
  }

  // Seed data if requested and backend is available
  if ((SEED_DATA || !SKIP_SEED) && mode === 'cluster') {
    log('Seeding test data...');
    await seedData({ skipIfExists: true });
  }

  // Serve the build
  const serverStarted = await serveBuild(buildDir, PORT_PR, 'current', mode);
  if (!serverStarted) {
    log('Failed to start server', 'error');
    return false;
  }

  // Capture screenshots
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

  // Setup work directory
  fs.mkdirSync(WORK_DIR, { recursive: true });

  // Get PR info
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

  // Setup backend
  const backendResult = await setupBackend(mode);
  if (!backendResult.success && mode !== 'static') {
    log(`Backend setup failed, falling back to static mode`, 'warn');
    mode = 'static';
  }

  // Seed data if backend is available
  if (!SKIP_SEED && mode === 'cluster') {
    log('Seeding test data...');
    await seedData({ skipIfExists: true });
  }

  // Checkout and build main branch
  log('Setting up main branch...');
  let mainBuildDir;
  try {
    mainBuildDir = await checkoutAndBuild(BASE_BRANCH, MAIN_DIR, 'main');
  } catch (e) {
    log(`Failed to setup main branch: ${e.message}`, 'error');
    return false;
  }

  // Use current repo for PR (already built or will build)
  let prBuildDir = path.join(REPO_ROOT, 'frontend', 'build');

  if (!SKIP_BUILD) {
    if (!fs.existsSync(prBuildDir)) {
      log('Building PR branch...');
      run('npm ci', { cwd: path.join(REPO_ROOT, 'frontend') });
      run('npm run build', { cwd: path.join(REPO_ROOT, 'frontend') });
    }
  }

  // Start servers
  // For comparison, both use static mode but point to builds that were tested against backend
  const mainServerStarted = await serveBuild(mainBuildDir, PORT_MAIN, 'main', 'static');
  const prServerStarted = await serveBuild(prBuildDir, PORT_PR, 'PR', 'static');

  if (!mainServerStarted || !prServerStarted) {
    log('Failed to start one or more servers', 'error');
    return false;
  }

  // Capture screenshots
  const mainScreenshots = path.join(SCREENSHOTS_DIR, 'main');
  const prScreenshots = path.join(SCREENSHOTS_DIR, 'pr');

  const mainCaptured = await captureScreenshots(PORT_MAIN, mainScreenshots, 'main');
  const prCaptured = await captureScreenshots(PORT_PR, prScreenshots, 'PR');

  if (!mainCaptured || !prCaptured) {
    log('Failed to capture screenshots', 'error');
    return false;
  }

  // Generate comparisons
  const comparisonGenerated = await generateComparison();
  if (!comparisonGenerated) {
    log('Failed to generate comparisons', 'error');
    return false;
  }

  // Upload to PR
  const uploaded = await uploadToPR();
  if (!uploaded) {
    log('Failed to upload to PR', 'warn');
  }

  log(`\n✓ UI smoke test complete!`);
  log(`Screenshots: ${SCREENSHOTS_DIR}`);

  return true;
}

async function main() {
  console.log(`
╔══════════════════════════════════════════════════════════════╗
║           Kubeflow Pipelines UI Smoke Test                   ║
╚══════════════════════════════════════════════════════════════╝
`);

  log(`PR: ${PR_NUMBER || 'N/A'}`);
  log(`Repository: ${REPO}`);
  log(`Base branch: ${BASE_BRANCH}`);
  log(`Work directory: ${WORK_DIR}`);

  // Ensure dependencies
  const nodeModulesPath = path.join(SCRIPT_DIR, 'node_modules');
  if (!fs.existsSync(nodeModulesPath)) {
    log('Installing smoke test dependencies...');
    run('npm install', { cwd: SCRIPT_DIR });
  }

  // Install Playwright browsers if needed
  run('npx playwright install chromium', { cwd: SCRIPT_DIR });

  // Determine mode
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

main().catch(err => {
  log(`Fatal error: ${err.message}`, 'error');
  cleanup();
  process.exit(1);
});
