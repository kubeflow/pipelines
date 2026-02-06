#!/usr/bin/env node
/**
 * Kind Cluster Manager for UI Smoke Tests
 *
 * Handles:
 * - Detecting if Kind cluster is running
 * - Starting/stopping the cluster
 * - Port forwarding backend services
 * - Health checks
 */

const { execSync, spawn } = require('child_process');
const http = require('http');

const CLUSTER_NAME = 'dev-pipelines-api';
const NAMESPACE = 'kubeflow';

// Port mappings from Kind config
const PORTS = {
  mysql: 3306,
  metadata: 8080,
  minio: 9000,
  visualization: 8888,
  apiServer: 8888,
  frontend: 3000,
  proxy: 3001,
};

// Track spawned processes
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
  // Check if ml-pipeline deployment exists
  const deployResult = run(`kubectl get deployment ml-pipeline -n ${NAMESPACE} -o jsonpath='{.status.availableReplicas}' 2>/dev/null`);

  // Also check if API is responding
  try {
    await new Promise((resolve, reject) => {
      const req = http.get('http://localhost:3001/apis/v2beta1/healthz', (res) => {
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
    // Switch context to the cluster
    run(`kubectl config use-context kind-${CLUSTER_NAME}`);

    // Check KFP deployment
    const kfpResult = run(`kubectl get deployment ml-pipeline -n ${NAMESPACE} 2>/dev/null`);
    status.kfpDeployed = kfpResult.success;

    if (status.kfpDeployed) {
      // Check pods are ready
      const podsResult = run(`kubectl get pods -n ${NAMESPACE} -l app=ml-pipeline -o jsonpath='{.items[*].status.phase}'`);
      status.servicesReady = podsResult.output.includes('Running') || podsResult.output === '';

      status.kfpHealthy = await isKfpHealthy();
    }
  }

  return status;
}

/**
 * Start the Kind cluster with KFP
 */
async function startCluster(repoRoot) {
  log('Starting Kind cluster with Kubeflow Pipelines...');
  log('This may take 5-10 minutes on first run.');

  // Use the Makefile target
  const result = run('make kind-cluster-agnostic', {
    cwd: `${repoRoot}/backend`,
    stdio: 'inherit',
    timeout: 600000, // 10 minutes
  });

  if (!result.success) {
    log('Failed to start cluster', 'error');
    return false;
  }

  log('Cluster started successfully');
  return true;
}

/**
 * Start port forwarding for backend services
 */
function startPortForwarding() {
  log('Starting port forwarding for backend services...');

  const forwards = [
    { service: 'metadata-envoy-service', localPort: 9090, remotePort: 9090 },
    { service: 'ml-pipeline', localPort: 3002, remotePort: 8888 },
    { service: 'minio-service', localPort: 9000, remotePort: 9000 },
  ];

  for (const fwd of forwards) {
    const proc = spawnProcess('kubectl', [
      'port-forward',
      '-n', NAMESPACE,
      `svc/${fwd.service}`,
      `${fwd.localPort}:${fwd.remotePort}`,
    ]);

    proc.on('error', (err) => {
      log(`Port forward failed for ${fwd.service}: ${err.message}`, 'warn');
    });

    log(`  ${fwd.service} -> localhost:${fwd.localPort}`);
  }

  return true;
}

/**
 * Start the Node.js proxy server
 */
function startProxyServer(repoRoot) {
  log('Starting Node.js proxy server...');

  const serverDir = `${repoRoot}/frontend/server`;

  // Build server if needed
  const distExists = run(`test -d "${serverDir}/dist" && echo "yes"`);
  if (!distExists.success || distExists.output !== 'yes') {
    log('Building server...');
    run('npm run build', { cwd: serverDir });
  }

  // Start server
  const proc = spawnProcess('node', ['dist/server.js', '../build', '3001'], {
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

  return proc;
}

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

/**
 * Delete the Kind cluster
 */
function deleteCluster() {
  log(`Deleting Kind cluster ${CLUSTER_NAME}...`);
  const result = run(`kind delete cluster --name ${CLUSTER_NAME}`);
  return result.success;
}

module.exports = {
  CLUSTER_NAME,
  NAMESPACE,
  PORTS,
  log,
  run,
  spawnProcess,
  isKindInstalled,
  isKubectlInstalled,
  isDockerRunning,
  isClusterRunning,
  isKfpHealthy,
  getClusterStatus,
  startCluster,
  startPortForwarding,
  startProxyServer,
  waitForService,
  cleanup,
  deleteCluster,
};
