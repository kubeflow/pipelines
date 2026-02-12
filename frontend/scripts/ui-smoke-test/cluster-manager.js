#!/usr/bin/env node
/**
 * Kind Cluster Manager for UI Smoke Tests
 *
 * Handles:
 * - Detecting if Kind cluster is running
 * - Starting/stopping the cluster
 * - Building and deploying changed backend components
 * - Port forwarding backend services
 * - Starting/stopping the Node.js frontend server
 * - Health checks
 */

const { execSync, spawn } = require('child_process');
const http = require('http');
const net = require('net');
const path = require('path');

const CLUSTER_NAME = 'dev-pipelines-api';
const NAMESPACE = 'kubeflow';

// Port forwarding config (matches frontend/scripts/start-proxy-and-server.sh)
const PORT_FORWARDS = [
  { service: 'metadata-envoy-service', localPort: 9090, remotePort: 9090 },
  { service: 'ml-pipeline', localPort: 3002, remotePort: 8888 },
  { service: 'minio-service', localPort: 9000, remotePort: 9000 },
];

const FRONTEND_SERVER_PORT = 3001;

// Track spawned processes for cleanup
const processes = [];

function log(msg, type = 'info') {
  const colors = { info: '\x1b[32m', warn: '\x1b[33m', error: '\x1b[31m', debug: '\x1b[36m' };
  const reset = '\x1b[0m';
  console.log(`${colors[type] || ''}[CLUSTER]${reset} ${msg}`);
}

function run(cmd, options = {}) {
  try {
    const result = execSync(cmd, { encoding: 'utf8', stdio: 'pipe', ...options });
    return { success: true, output: result?.trim() || '' };
  } catch (error) {
    return { success: false, error: error.message, output: error.stdout?.trim() || '' };
  }
}

function spawnProcess(cmd, args, options = {}) {
  const proc = spawn(cmd, args, { stdio: 'pipe', ...options });
  processes.push(proc);
  return proc;
}

/**
 * Check if Kind is installed
 */
function isKindInstalled() {
  const result = run('which kind');
  return result.success;
}

/**
 * Check if kubectl is installed
 */
function isKubectlInstalled() {
  const result = run('which kubectl');
  return result.success;
}

/**
 * Check if Docker is running
 */
function isDockerRunning() {
  const result = run('docker info');
  return result.success;
}

/**
 * Check if the Kind cluster exists and is running
 */
function isClusterRunning() {
  const result = run(`kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$" && echo "yes" || echo "no"`);
  return result.success && result.output === 'yes';
}

/**
 * Check if KFP is deployed and healthy
 */
async function isKfpHealthy() {
  try {
    await new Promise((resolve, reject) => {
      const req = http.get(`http://localhost:${FRONTEND_SERVER_PORT}/apis/v2beta1/healthz`, (res) => {
        resolve(res.statusCode === 200);
      });
      req.on('error', reject);
      req.setTimeout(2000, () => { req.destroy(); reject(new Error('timeout')); });
    });
    return true;
  } catch (e) {
    return false;
  }
}

/**
 * Get cluster status
 */
async function getClusterStatus() {
  const status = {
    kindInstalled: isKindInstalled(),
    kubectlInstalled: isKubectlInstalled(),
    dockerRunning: isDockerRunning(),
    clusterRunning: false,
    kfpDeployed: false,
    kfpHealthy: false,
    servicesReady: false,
  };

  if (!status.kindInstalled || !status.kubectlInstalled || !status.dockerRunning) {
    return status;
  }

  status.clusterRunning = isClusterRunning();

  if (status.clusterRunning) {
    run(`kubectl config use-context kind-${CLUSTER_NAME}`);

    const kfpResult = run(`kubectl get deployment ml-pipeline -n ${NAMESPACE} 2>/dev/null`);
    status.kfpDeployed = kfpResult.success;

    if (status.kfpDeployed) {
      const podsResult = run(`kubectl get pods -n ${NAMESPACE} -l app=ml-pipeline -o jsonpath='{.items[*].status.phase}'`);
      status.servicesReady = podsResult.output.includes('Running') || podsResult.output === '';

      status.kfpHealthy = await isKfpHealthy();
    }
  }

  return status;
}

/**
 * Synchronously check if a TCP port is in use on localhost.
 * Spawns a short Node subprocess to attempt binding.
 *
 * @param {number} port
 * @returns {boolean}
 */
function isPortInUseSync(port) {
  const script = `const s=require("net").createServer();s.once("error",()=>process.exit(1));s.once("listening",()=>{s.close(()=>process.exit(0))});s.listen(${port},"127.0.0.1")`;
  try {
    execSync(`node -e '${script}'`, { stdio: 'ignore' });
    return false;
  } catch (e) {
    return true;
  }
}

/**
 * Check if the given ports are available. Returns an array of conflicts.
 *
 * Port availability is determined via a net-based check; lsof is used
 * only on a best-effort basis to obtain PID and process details.
 *
 * @param {number[]} ports - Ports to check
 * @returns {{ port: number, pid: string, process: string }[]}
 */
function checkPortAvailability(ports) {
  const conflicts = [];
  for (const port of ports) {
    if (!isPortInUseSync(port)) {
      continue;
    }

    // Port is in use; try to enrich with PID/process info via lsof
    let pid = 'unknown';
    let processName = 'unknown';

    const pidResult = run(`lsof -i :${port} -t 2>/dev/null`);
    if (pidResult.success && pidResult.output) {
      pid = pidResult.output.split('\n')[0].trim() || 'unknown';
      if (pid !== 'unknown') {
        const psResult = run(`ps -p ${pid} -o comm= 2>/dev/null`);
        if (psResult.success && psResult.output) {
          processName = psResult.output.trim();
        }
      }
    }

    conflicts.push({ port, pid, process: processName });
  }
  return conflicts;
}

// ────────────────────────────────────────────────────────────────
//  Cluster lifecycle
// ────────────────────────────────────────────────────────────────

/**
 * Ensure the Kind cluster is running and KFP is deployed.
 * Starts the cluster if it's not already running.
 */
async function ensureCluster(repoRoot) {
  if (!isKindInstalled()) throw new Error('kind is not installed');
  if (!isKubectlInstalled()) throw new Error('kubectl is not installed');
  if (!isDockerRunning()) throw new Error('Docker is not running');

  if (isClusterRunning()) {
    log('Kind cluster already running');
    run(`kubectl config use-context kind-${CLUSTER_NAME}`);

    // Verify KFP is deployed
    const kfpResult = run(`kubectl get deployment ml-pipeline -n ${NAMESPACE} 2>/dev/null`);
    if (!kfpResult.success) {
      throw new Error('Cluster is running but KFP is not deployed. Run: make -C backend kind-cluster-agnostic');
    }
    return;
  }

  log('Starting Kind cluster with Kubeflow Pipelines...');
  log('This may take 5-10 minutes on first run.');

  const result = run('make kind-cluster-agnostic', {
    cwd: path.join(repoRoot, 'backend'),
    stdio: 'inherit',
    timeout: 600000, // 10 minutes
  });

  if (!result.success) {
    throw new Error(`Failed to start cluster: ${result.error}`);
  }

  log('Cluster started successfully');
}

/**
 * Delete the Kind cluster entirely.
 */
function teardownCluster() {
  log(`Deleting Kind cluster ${CLUSTER_NAME}...`);
  const result = run(`kind delete cluster --name ${CLUSTER_NAME}`);
  if (!result.success) {
    log('Failed to delete cluster', 'warn');
  }
  return result.success;
}

// ────────────────────────────────────────────────────────────────
//  Component build & deploy
// ────────────────────────────────────────────────────────────────

/**
 * Build Docker images for changed components, load them into Kind,
 * and restart the relevant K8s deployments.
 *
 * @param {object[]} components - Array from detect-changes.js COMPONENTS
 * @param {string} repoRoot - Repo root path
 */
async function buildAndDeployComponents(components, repoRoot) {
  if (components.length === 0) {
    log('No backend components to rebuild');
    return;
  }

  const backendDir = path.join(repoRoot, 'backend');

  for (const component of components) {
    log(`Building ${component.name} (make ${component.makeTarget})...`);
    const buildResult = run(`make ${component.makeTarget}`, {
      cwd: backendDir,
      stdio: 'inherit',
      timeout: 300000, // 5 minutes per component
    });

    if (!buildResult.success) {
      throw new Error(`Failed to build ${component.name}: ${buildResult.error}`);
    }

    // Load the image into Kind
    // Image tags follow the pattern from the Makefile: IMG_TAG_<name>
    log(`Loading ${component.imageTag} image into Kind...`);
    const loadResult = run(`kind load docker-image ${component.imageTag} --name ${CLUSTER_NAME}`, {
      timeout: 120000,
    });

    if (!loadResult.success) {
      log(`Failed to load ${component.imageTag} into Kind: ${loadResult.error}`, 'warn');
    }

    // Restart the deployment if there is one
    if (component.deployment) {
      log(`Restarting deployment/${component.deployment}...`);
      run(`kubectl -n ${NAMESPACE} rollout restart deployment/${component.deployment}`);
    }
  }

  // Wait for deployments to be ready
  const deploymentsToWait = components
    .filter(c => c.deployment)
    .map(c => c.deployment);

  if (deploymentsToWait.length > 0) {
    log('Waiting for deployments to be ready...');
    for (const deployment of deploymentsToWait) {
      const waitResult = run(
        `kubectl -n ${NAMESPACE} rollout status deployment/${deployment} --timeout=120s`,
        { timeout: 130000 },
      );
      if (!waitResult.success) {
        log(`Deployment ${deployment} did not become ready`, 'warn');
      }
    }
  }

  log(`${components.length} component(s) rebuilt and deployed`);
}

/**
 * Re-apply kustomize manifests (when manifests/ files changed).
 */
function reapplyManifests(repoRoot) {
  log('Re-applying kustomize manifests...');
  const result = run(
    `kubectl apply -k manifests/kustomize/env/platform-agnostic`,
    { cwd: repoRoot, stdio: 'inherit', timeout: 60000 },
  );

  if (!result.success) {
    log('Failed to apply manifests', 'warn');
  }
  return result.success;
}

// ────────────────────────────────────────────────────────────────
//  Port forwarding
// ────────────────────────────────────────────────────────────────

/**
 * Check if a local port is already in use.
 */
function isPortInUse(port) {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once('error', () => resolve(true));
    server.once('listening', () => { server.close(); resolve(false); });
    server.listen(port, '127.0.0.1');
  });
}

/**
 * Start port forwarding for backend services, skipping ports already in use.
 */
async function ensurePortForwarding() {
  log('Ensuring port forwards are active...');

  for (const fwd of PORT_FORWARDS) {
    const inUse = await isPortInUse(fwd.localPort);
    if (inUse) {
      log(`  Port ${fwd.localPort} already in use (${fwd.service}) — skipping`);
      continue;
    }

    const proc = spawnProcess('kubectl', [
      'port-forward',
      '-n', NAMESPACE,
      `svc/${fwd.service}`,
      `${fwd.localPort}:${fwd.remotePort}`,
    ]);

    proc.on('error', (err) => {
      log(`Port forward failed for ${fwd.service}: ${err.message}`, 'warn');
    });

    log(`  ${fwd.service} → localhost:${fwd.localPort}`);
  }

  // Give port forwards a moment to establish
  await new Promise(r => setTimeout(r, 2000));
}

// ────────────────────────────────────────────────────────────────
//  Frontend server (the real proxy to K8s services)
// ────────────────────────────────────────────────────────────────

/**
 * Start the Node.js frontend server on port 3001.
 *
 * This is the real proxy layer that forwards API calls to the
 * port-forwarded K8s services. proxy-server.js instances on
 * ports 4001/4002 forward their API calls here.
 *
 * Follows the pattern from frontend/scripts/start-proxy-and-server.sh:
 *   - Scale down ml-pipeline-ui so the local server owns the proxy layer
 *   - Build the frontend server
 *   - Start with ML_PIPELINE_SERVICE_PORT=3002
 */
// Prior replica count for ml-pipeline-ui, captured before scaling down.
let priorUiReplicas = 1;

async function startFrontendServer(repoRoot) {
  const serverDir = path.join(repoRoot, 'frontend', 'server');

  // Capture prior replica count so we can restore it on cleanup
  const replicaResult = run(
    `kubectl -n ${NAMESPACE} get deployment ml-pipeline-ui -o jsonpath='{.spec.replicas}' 2>/dev/null`,
  );
  if (replicaResult.success && /^\d+$/.test(replicaResult.output)) {
    priorUiReplicas = parseInt(replicaResult.output, 10);
  }

  // Scale down the in-cluster UI pod so it doesn't compete
  log('Scaling down ml-pipeline-ui in cluster...');
  run(`kubectl -n ${NAMESPACE} scale deployment/ml-pipeline-ui --replicas=0`);

  try {
    // Build server if dist/ is missing or stale
    log('Building frontend server...');
    const buildResult = run('npm run build', { cwd: serverDir, timeout: 60000 });
    if (!buildResult.success) {
      throw new Error(`Failed to build frontend server: ${buildResult.error}`);
    }

    // Start the server
    log(`Starting frontend server on port ${FRONTEND_SERVER_PORT}...`);
    const buildDir = path.join(repoRoot, 'frontend', 'build');
    const proc = spawnProcess('node', ['dist/server.js', buildDir, String(FRONTEND_SERVER_PORT)], {
      cwd: serverDir,
      env: {
        ...process.env,
        ML_PIPELINE_SERVICE_PORT: '3002',
      },
    });

    proc.stdout?.on('data', (data) => {
      if (process.env.VERBOSE) console.log(data.toString());
    });
    proc.stderr?.on('data', (data) => {
      if (process.env.VERBOSE) console.error(data.toString());
    });

    // Wait for it to be ready
    const ready = await waitForService(`http://localhost:${FRONTEND_SERVER_PORT}`, 15000);
    if (!ready) {
      throw new Error('Frontend server failed to start on port ' + FRONTEND_SERVER_PORT);
    }

    log(`Frontend server ready at http://localhost:${FRONTEND_SERVER_PORT}`);
    return proc;
  } catch (e) {
    // Restore ml-pipeline-ui on failure so the cluster isn't left broken
    stopFrontendServer();
    throw e;
  }
}

/**
 * Stop the frontend server and restore the in-cluster UI pod
 * to its prior replica count.
 */
function stopFrontendServer() {
  log(`Scaling ml-pipeline-ui back to ${priorUiReplicas} replica(s)...`);
  run(`kubectl -n ${NAMESPACE} scale deployment/ml-pipeline-ui --replicas=${priorUiReplicas}`);
}

// ────────────────────────────────────────────────────────────────
//  Utilities
// ────────────────────────────────────────────────────────────────

/**
 * Wait for a service to be ready
 */
async function waitForService(url, timeout = 30000) {
  const start = Date.now();

  while (Date.now() - start < timeout) {
    try {
      await new Promise((resolve, reject) => {
        const req = http.get(url, (res) => {
          resolve(res.statusCode < 500);
        });
        req.on('error', reject);
        req.setTimeout(2000, () => { req.destroy(); reject(new Error('timeout')); });
      });
      return true;
    } catch (e) {
      await new Promise(r => setTimeout(r, 1000));
    }
  }
  return false;
}

/**
 * Cleanup spawned processes
 */
function cleanup() {
  log('Cleaning up processes...');
  for (const proc of processes) {
    try {
      proc.kill('SIGTERM');
    } catch (e) {
      // Ignore
    }
  }
  processes.length = 0;
}

module.exports = {
  CLUSTER_NAME,
  NAMESPACE,
  FRONTEND_SERVER_PORT,
  PORT_FORWARDS,
  log,
  run,
  spawnProcess,
  isKindInstalled,
  isKubectlInstalled,
  isDockerRunning,
  isClusterRunning,
  isKfpHealthy,
  getClusterStatus,
  checkPortAvailability,
  ensureCluster,
  teardownCluster,
  buildAndDeployComponents,
  reapplyManifests,
  ensurePortForwarding,
  startFrontendServer,
  stopFrontendServer,
  waitForService,
  cleanup,
};
