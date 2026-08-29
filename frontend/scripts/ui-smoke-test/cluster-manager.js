#!/usr/bin/env node
/**
 * Kind cluster lifecycle and local proxy support for the UI smoke test.
 */

const { execFileSync, spawn } = require('child_process');
const fs = require('fs');
const http = require('http');
const net = require('net');
const path = require('path');

const CLUSTER_NAME = 'ui-smoke-test';
const KUBE_CONTEXT = `kind-${CLUSTER_NAME}`;
const NAMESPACE = 'kubeflow';
const FRONTEND_SERVER_PORT = 3001;
const PORT_FORWARDS = [
  { service: 'metadata-envoy-service', localPort: 9090, remotePort: 9090 },
  { service: 'ml-pipeline', localPort: 3002, remotePort: 8888 },
  { service: 'seaweedfs', localPort: 9000, remotePort: 9000 },
];
const SEED_RUNTIME_IMAGE =
  'docker.io/library/busybox@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662';

// The platform-agnostic kustomization currently renders these Deployments and no StatefulSets or
// Jobs. Keep this list explicit so a missing workload fails setup instead of being skipped by an
// empty label or namespace query.
const PLATFORM_DEPLOYMENTS = [
  'cache-deployer-deployment',
  'cache-server',
  'metadata-envoy-deployment',
  'metadata-grpc-deployment',
  'metadata-writer',
  'ml-pipeline',
  'ml-pipeline-persistenceagent',
  'ml-pipeline-scheduledworkflow',
  'ml-pipeline-ui',
  'ml-pipeline-viewer-crd',
  'ml-pipeline-visualizationserver',
  'mysql',
  'seaweedfs',
  'workflow-controller',
];

const processes = [];
let frontendServerProcess = null;

function log(message, type = 'info') {
  const colors = {
    info: '\x1b[32m',
    warn: '\x1b[33m',
    error: '\x1b[31m',
    debug: '\x1b[36m',
  };
  console.log(`${colors[type] || ''}[CLUSTER]\x1b[0m ${message}`);
}

/** Execute a program directly so refs and paths are never interpreted by a shell. */
function run(command, args = [], options = {}) {
  try {
    const output = execFileSync(command, args, {
      encoding: 'utf8',
      stdio: 'pipe',
      ...options,
    });
    return { success: true, output: output?.trim() || '' };
  } catch (error) {
    return {
      success: false,
      error: error.message,
      output: typeof error.stdout === 'string' ? error.stdout.trim() : '',
    };
  }
}

function requireSuccess(result, action) {
  if (!result?.success) {
    const detail = result?.error || result?.output || 'unknown error';
    throw new Error(`${action}: ${detail}`);
  }
  return result;
}

function spawnProcess(command, args, options = {}) {
  const child = spawn(command, args, { stdio: 'pipe', ...options });
  child.stdout?.on('data', (data) => {
    if (process.env.VERBOSE) process.stdout.write(data);
  });
  child.stderr?.on('data', (data) => {
    if (process.env.VERBOSE) process.stderr.write(data);
  });
  processes.push(child);
  return child;
}

function kubectlArgs(...args) {
  return ['--context', KUBE_CONTEXT, ...args];
}

function isKindInstalled(runner = run) {
  return runner('kind', ['version']).success;
}

function isKubectlInstalled(runner = run) {
  return runner('kubectl', ['version', '--client']).success;
}

function isDockerRunning(runner = run) {
  return runner('docker', ['info']).success;
}

function isClusterRunning(runner = run) {
  const result = runner('kind', ['get', 'clusters']);
  return result.success && result.output.split('\n').includes(CLUSTER_NAME);
}

function getHttpStatus(url, timeout = 2000, get = http.get) {
  return new Promise((resolve, reject) => {
    const request = get(url, (response) => {
      response.resume?.();
      resolve(response.statusCode || 0);
    });
    request.on('error', reject);
    request.setTimeout(timeout, () => {
      request.destroy();
      reject(new Error(`Timed out requesting ${url}`));
    });
  });
}

async function isKfpHealthy(options = {}) {
  const { url = `http://localhost:${FRONTEND_SERVER_PORT}/apis/v2beta1/healthz`, get = http.get } =
    options;
  try {
    return (await getHttpStatus(url, 2000, get)) === 200;
  } catch (error) {
    return false;
  }
}

async function getClusterStatus(options = {}) {
  const { runner = run, healthCheck = isKfpHealthy } = options;
  const status = {
    kindInstalled: isKindInstalled(runner),
    kubectlInstalled: isKubectlInstalled(runner),
    dockerRunning: isDockerRunning(runner),
    clusterRunning: false,
    kfpDeployed: false,
    kfpHealthy: false,
    servicesReady: false,
  };
  if (!status.kindInstalled || !status.kubectlInstalled || !status.dockerRunning) return status;

  status.clusterRunning = isClusterRunning(runner);
  if (!status.clusterRunning) return status;

  const deployment = runner(
    'kubectl',
    kubectlArgs('-n', NAMESPACE, 'get', 'deployment', 'ml-pipeline'),
  );
  status.kfpDeployed = deployment.success;
  if (!status.kfpDeployed) return status;

  status.servicesReady = runner(
    'kubectl',
    kubectlArgs(
      '-n',
      NAMESPACE,
      'wait',
      '--for=condition=Available',
      '--timeout=1s',
      'deployment/ml-pipeline',
    ),
  ).success;
  status.kfpHealthy = await healthCheck();
  return status;
}

function isPortInUseSync(port) {
  const script = [
    'const net=require("net");',
    'const server=net.createServer();',
    'server.once("error",()=>process.exit(1));',
    'server.once("listening",()=>server.close(()=>process.exit(0)));',
    `server.listen(${Number(port)},"127.0.0.1");`,
  ].join('');
  try {
    execFileSync(process.execPath, ['-e', script], { stdio: 'ignore' });
    return false;
  } catch (error) {
    return true;
  }
}

function checkPortAvailability(ports, options = {}) {
  const { runner = run, portInUse = isPortInUseSync } = options;
  const conflicts = [];
  for (const port of ports) {
    if (!portInUse(port)) continue;

    let pid = 'unknown';
    let processName = 'unknown';
    const pidResult = runner('lsof', ['-i', `:${port}`, '-t']);
    if (pidResult.success && pidResult.output) {
      pid = pidResult.output.split('\n')[0].trim() || 'unknown';
      if (pid !== 'unknown') {
        const processResult = runner('ps', ['-p', pid, '-o', 'comm=']);
        if (processResult.success && processResult.output) {
          processName = processResult.output.trim();
        }
      }
    }
    conflicts.push({ port, pid, process: processName });
  }
  return conflicts;
}

function restoreKubectlContext(previousContext, runner = run) {
  const result = previousContext
    ? runner('kubectl', ['config', 'use-context', previousContext])
    : runner('kubectl', ['config', 'unset', 'current-context']);
  requireSuccess(result, 'Failed to restore the previous kubectl context');
}

function preloadSeedRuntimeImage(runner = run) {
  requireSuccess(
    runner('docker', ['pull', SEED_RUNTIME_IMAGE], {
      timeout: 300000,
      stdio: 'inherit',
    }),
    `Failed to pull deterministic seed image ${SEED_RUNTIME_IMAGE}`,
  );
  requireSuccess(
    runner('kind', ['load', 'docker-image', SEED_RUNTIME_IMAGE, '--name', CLUSTER_NAME], {
      timeout: 180000,
    }),
    `Failed to load deterministic seed image ${SEED_RUNTIME_IMAGE} into Kind`,
  );
}

function applyKfpManifests(repoRoot, runner = run) {
  const clusterScoped = path.join(repoRoot, 'manifests', 'kustomize', 'cluster-scoped-resources');
  const platformAgnostic = path.join(
    repoRoot,
    'manifests',
    'kustomize',
    'env',
    'platform-agnostic',
  );

  requireSuccess(
    runner('kubectl', kubectlArgs('apply', '-k', clusterScoped), {
      timeout: 120000,
      stdio: 'inherit',
    }),
    'Failed to apply cluster-scoped KFP manifests',
  );
  requireSuccess(
    runner(
      'kubectl',
      kubectlArgs(
        'wait',
        '--for=condition=established',
        '--timeout=1m',
        'crd/applications.app.k8s.io',
      ),
      { timeout: 70000 },
    ),
    'KFP application CRD was not established',
  );
  requireSuccess(
    runner('kubectl', kubectlArgs('apply', '-k', platformAgnostic), {
      timeout: 180000,
      stdio: 'inherit',
    }),
    'Failed to apply platform-agnostic KFP manifests',
  );

  requireSuccess(
    runner(
      'kubectl',
      kubectlArgs(
        '-n',
        NAMESPACE,
        'wait',
        '--for=condition=Available',
        '--timeout=10m',
        ...PLATFORM_DEPLOYMENTS.map((deployment) => `deployment/${deployment}`),
      ),
      { timeout: 610000 },
    ),
    'Platform-agnostic KFP deployments did not all become available',
  );
}

async function ensureCluster(repoRoot, options = {}) {
  const { runner = run } = options;
  if (!isKindInstalled(runner)) throw new Error('kind is not installed');
  if (!isKubectlInstalled(runner)) throw new Error('kubectl is not installed');
  if (!isDockerRunning(runner)) throw new Error('Docker is not running');

  if (isClusterRunning(runner)) {
    throw new Error(
      `Managed Kind cluster ${CLUSTER_NAME} already exists. Refusing to reuse potentially stale ` +
        'backend images or data; run smoke-test-runner.js --teardown before comparing again.',
    );
  }

  const setupErrors = [];
  let created = false;
  log(`Creating Kind cluster ${CLUSTER_NAME}...`);
  const currentContextResult = runner('kubectl', ['config', 'current-context']);
  const previousContext = currentContextResult.success ? currentContextResult.output : '';
  try {
    requireSuccess(
      runner('kind', ['create', 'cluster', '--name', CLUSTER_NAME], { timeout: 600000 }),
      `Failed to create Kind cluster ${CLUSTER_NAME}`,
    );
    created = true;
  } catch (error) {
    setupErrors.push(error);
  }

  try {
    restoreKubectlContext(previousContext, runner);
  } catch (error) {
    setupErrors.push(error);
  }

  if (setupErrors.length === 0) {
    try {
      preloadSeedRuntimeImage(runner);
      applyKfpManifests(repoRoot, runner);
    } catch (error) {
      setupErrors.push(error);
    }
  }

  if (setupErrors.length > 0) {
    const clusterMayExist = created || isClusterRunning(runner);
    let rollbackError = null;
    if (clusterMayExist) {
      log(`Rolling back failed Kind cluster setup for ${CLUSTER_NAME}...`, 'warn');
      try {
        requireSuccess(
          runner('kind', ['delete', 'cluster', '--name', CLUSTER_NAME]),
          `Failed to roll back managed Kind cluster ${CLUSTER_NAME}`,
        );
      } catch (error) {
        rollbackError = error;
      }
    }

    if (rollbackError) {
      throw new AggregateError(
        [...setupErrors, rollbackError],
        `Kind cluster ${CLUSTER_NAME} setup and rollback both failed`,
      );
    }
    if (setupErrors.length > 1) {
      throw new AggregateError(setupErrors, `Kind cluster ${CLUSTER_NAME} setup failed`);
    }
    throw setupErrors[0];
  }

  return { created, context: KUBE_CONTEXT };
}

function teardownCluster(options = {}) {
  const { runner = run } = options;
  log(`Deleting Kind cluster ${CLUSTER_NAME}...`);
  return runner('kind', ['delete', 'cluster', '--name', CLUSTER_NAME]).success;
}

function localImageTag(component, suffix) {
  const safeSuffix = String(suffix).replace(/[^a-zA-Z0-9_.-]/g, '-');
  return `${component.imageTag}:${safeSuffix}`;
}

function getClusterPlatform(runner = run) {
  const result = runner(
    'kubectl',
    kubectlArgs('get', 'nodes', '-o', 'jsonpath={.items[0].status.nodeInfo.architecture}'),
  );
  requireSuccess(result, 'Failed to determine the Kind node architecture');
  const architecture = result.output.trim();
  if (!/^[a-z0-9_]+$/.test(architecture)) {
    throw new Error(`Kind reported an invalid node architecture: ${JSON.stringify(architecture)}`);
  }
  return `linux/${architecture}`;
}

async function buildAndDeployComponents(components, repoRoot, options = {}) {
  const { runner = run, tagSuffix = `${process.pid}-${Date.now()}` } = options;
  if (components.length === 0) {
    log('No backend components to rebuild');
    return { images: {} };
  }

  const images = {};
  const deployments = new Set();
  const runtimeEnvironment = {};
  const platform = options.platform || getClusterPlatform(runner);
  if (!/^linux\/[a-z0-9_]+$/.test(platform)) {
    throw new Error(`Invalid backend image platform: ${JSON.stringify(platform)}`);
  }
  for (const component of components) {
    if (!component.dockerfile || !component.imageTag) {
      throw new Error(`Component ${component.name} is missing Docker build metadata.`);
    }
    const image = localImageTag(component, tagSuffix);
    images[component.name] = image;
    log(`Building ${component.name} as ${image}...`);
    requireSuccess(
      runner(
        'docker',
        ['build', '--platform', platform, '--tag', image, '--file', component.dockerfile, '.'],
        { cwd: repoRoot, timeout: 600000, stdio: 'inherit' },
      ),
      `Failed to build ${component.name}`,
    );
    requireSuccess(
      runner('kind', ['load', 'docker-image', image, '--name', CLUSTER_NAME], {
        timeout: 180000,
      }),
      `Failed to load ${image} into Kind`,
    );

    if (component.deployment) {
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            NAMESPACE,
            'set',
            'image',
            `deployment/${component.deployment}`,
            `${component.container}=${image}`,
          ),
        ),
        `Failed to set image on deployment/${component.deployment}`,
      );
      const pullPolicyPatch = JSON.stringify({
        spec: {
          template: {
            spec: {
              containers: [{ name: component.container, imagePullPolicy: 'IfNotPresent' }],
            },
          },
        },
      });
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            NAMESPACE,
            'patch',
            `deployment/${component.deployment}`,
            '--type=strategic',
            '-p',
            pullPolicyPatch,
          ),
        ),
        `Failed to set IfNotPresent on deployment/${component.deployment}`,
      );
      deployments.add(component.deployment);
    }
    if (component.runtimeEnv) runtimeEnvironment[component.runtimeEnv] = image;
  }

  const runtimeEntries = Object.entries(runtimeEnvironment);
  if (runtimeEntries.length > 0) {
    requireSuccess(
      runner(
        'kubectl',
        kubectlArgs(
          '-n',
          NAMESPACE,
          'set',
          'env',
          'deployment/ml-pipeline',
          ...runtimeEntries.map(([name, image]) => `${name}=${image}`),
        ),
      ),
      'Failed to configure local KFP runtime images',
    );
    deployments.add('ml-pipeline');
  }

  for (const deployment of deployments) {
    requireSuccess(
      runner(
        'kubectl',
        kubectlArgs('-n', NAMESPACE, 'rollout', 'restart', `deployment/${deployment}`),
      ),
      `Failed to restart deployment/${deployment}`,
    );
    requireSuccess(
      runner(
        'kubectl',
        kubectlArgs(
          '-n',
          NAMESPACE,
          'rollout',
          'status',
          `deployment/${deployment}`,
          '--timeout=180s',
        ),
        { timeout: 190000 },
      ),
      `Deployment ${deployment} did not become ready`,
    );
  }
  return { images };
}

function reapplyManifests(repoRoot, options = {}) {
  const { runner = run } = options;
  log('Re-applying KFP manifests...');
  applyKfpManifests(repoRoot, runner);
  return true;
}

function isPortInUse(port) {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once('error', () => resolve(true));
    server.once('listening', () => server.close(() => resolve(false)));
    server.listen(port, '127.0.0.1');
  });
}

function canConnect(port, host = '127.0.0.1') {
  return new Promise((resolve) => {
    const socket = net.createConnection({ host, port });
    const finish = (ready) => {
      socket.removeAllListeners();
      socket.destroy();
      resolve(ready);
    };
    socket.setTimeout(500);
    socket.once('connect', () => finish(true));
    socket.once('timeout', () => finish(false));
    socket.once('error', () => finish(false));
  });
}

async function waitForTcp(port, timeout = 15000, options = {}) {
  const { child = null, connect = canConnect, interval = 100 } = options;
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeout) {
    if (child && (child.exitCode !== null || child.signalCode !== null)) {
      throw new Error(
        `Port-forward process exited (code=${child.exitCode}, signal=${child.signalCode})`,
      );
    }
    if (await connect(port)) return true;
    await new Promise((resolve) => setTimeout(resolve, interval));
  }
  return false;
}

async function waitForChildReadiness(child, readiness) {
  if (child.exitCode !== null || child.signalCode !== null) {
    throw new Error(
      `Child process already exited (code=${child.exitCode}, signal=${child.signalCode})`,
    );
  }
  let onError;
  let onExit;
  const failure = new Promise((resolve, reject) => {
    onError = (error) => reject(error);
    onExit = (code, signal) => {
      reject(new Error(`Child process exited before readiness (code=${code}, signal=${signal})`));
    };
    child.once('error', onError);
    child.once('exit', onExit);
  });
  try {
    return await Promise.race([Promise.resolve().then(readiness), failure]);
  } finally {
    child.off('error', onError);
    child.off('exit', onExit);
  }
}

async function ensurePortForwarding(options = {}) {
  const {
    spawnFn = spawnProcess,
    portInUse = isPortInUse,
    waitForTcpFn = waitForTcp,
    timeout = 15000,
  } = options;
  const started = [];
  try {
    for (const forward of PORT_FORWARDS) {
      if (await portInUse(forward.localPort)) {
        throw new Error(
          `Port ${forward.localPort} became occupied before ${forward.service} forwarding started.`,
        );
      }
      const child = spawnFn(
        'kubectl',
        kubectlArgs(
          'port-forward',
          '-n',
          NAMESPACE,
          `svc/${forward.service}`,
          `${forward.localPort}:${forward.remotePort}`,
        ),
      );
      started.push(child);
      const ready = await waitForChildReadiness(child, () =>
        waitForTcpFn(forward.localPort, timeout, { child }),
      );
      if (!ready || child.exitCode !== null || child.signalCode !== null) {
        throw new Error(`Port forward for ${forward.service} did not become ready.`);
      }
      log(`${forward.service} -> localhost:${forward.localPort}`);
    }
    return started;
  } catch (error) {
    for (const child of started) child.kill('SIGTERM');
    throw error;
  }
}

function frontendServerEnvironment(baseEnvironment = process.env) {
  const endpointRewrite = [
    `seaweedfs.${NAMESPACE}:9000=localhost:9000`,
    `seaweedfs.${NAMESPACE}:80=localhost:9000`,
    `seaweedfs.${NAMESPACE}.svc:9000=localhost:9000`,
    `seaweedfs.${NAMESPACE}.svc:80=localhost:9000`,
    `seaweedfs.${NAMESPACE}.svc.cluster.local:9000=localhost:9000`,
    `seaweedfs.${NAMESPACE}.svc.cluster.local:80=localhost:9000`,
  ].join(',');
  return {
    ...baseEnvironment,
    FRONTEND_SERVER_NAMESPACE: NAMESPACE,
    MINIO_ENDPOINT_REWRITE: endpointRewrite,
    MINIO_HOST: 'localhost',
    MINIO_NAMESPACE: '',
    ML_PIPELINE_SERVICE_PORT: '3002',
  };
}

async function startFrontendServer(repoRoot, options = {}) {
  const {
    skipBuild = false,
    runner = run,
    spawnFn = spawnProcess,
    waitForServiceFn = waitForService,
  } = options;
  const serverDir = path.join(repoRoot, 'frontend', 'server');
  const serverEntry = path.join(serverDir, 'dist', 'server.js');
  if (skipBuild) {
    if (!fs.existsSync(serverEntry)) {
      throw new Error(`Cannot skip server build: ${serverEntry} does not exist.`);
    }
  } else {
    requireSuccess(
      runner('npm', ['ci'], { cwd: serverDir, timeout: 120000, stdio: 'inherit' }),
      'Failed to install frontend server dependencies',
    );
    requireSuccess(
      runner('npm', ['run', 'build'], { cwd: serverDir, timeout: 120000 }),
      'Failed to build frontend server',
    );
  }

  const buildDir = path.join(repoRoot, 'frontend', 'build');
  const child = spawnFn('node', ['dist/server.js', buildDir, String(FRONTEND_SERVER_PORT)], {
    cwd: serverDir,
    env: frontendServerEnvironment(),
  });
  frontendServerProcess = child;
  const healthUrl = `http://localhost:${FRONTEND_SERVER_PORT}/apis/v2beta1/healthz`;
  try {
    const ready = await waitForChildReadiness(child, () =>
      waitForServiceFn(healthUrl, 15000, { child }),
    );
    if (!ready) throw new Error(`Frontend server did not become healthy at ${healthUrl}.`);
    return child;
  } catch (error) {
    child.kill('SIGTERM');
    frontendServerProcess = null;
    throw error;
  }
}

function stopFrontendServer() {
  if (!frontendServerProcess) return;
  frontendServerProcess.kill('SIGTERM');
  frontendServerProcess = null;
}

async function waitForService(url, timeout = 30000, options = {}) {
  const { child = null, get = http.get, interval = 250 } = options;
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeout) {
    if (child && (child.exitCode !== null || child.signalCode !== null)) {
      throw new Error(
        `Service process exited (code=${child.exitCode}, signal=${child.signalCode})`,
      );
    }
    try {
      const status = await getHttpStatus(url, Math.min(2000, timeout), get);
      if (status >= 200 && status < 300) return true;
    } catch (error) {
      // Retry until timeout.
    }
    await new Promise((resolve) => setTimeout(resolve, interval));
  }
  return false;
}

async function terminateChild(child, timeout = 3000) {
  if (!child || child.exitCode !== null || child.signalCode !== null) return;
  await new Promise((resolve) => {
    let finished = false;
    let timer;
    const finish = () => {
      if (finished) return;
      finished = true;
      clearTimeout(timer);
      child.off?.('close', finish);
      resolve();
    };
    child.once?.('close', finish);
    child.kill('SIGTERM');
    timer = setTimeout(() => {
      if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
      finish();
    }, timeout);
  });
}

async function cleanup(options = {}) {
  const { runner = run, terminate = terminateChild } = options;
  log('Cleaning up processes...');
  await Promise.all(
    processes.map(async (child) => {
      try {
        await terminate(child);
      } catch (error) {
        // The process may already have exited.
      }
    }),
  );
  processes.length = 0;
  frontendServerProcess = null;
}

module.exports = {
  CLUSTER_NAME,
  KUBE_CONTEXT,
  NAMESPACE,
  FRONTEND_SERVER_PORT,
  PLATFORM_DEPLOYMENTS,
  PORT_FORWARDS,
  SEED_RUNTIME_IMAGE,
  applyKfpManifests,
  buildAndDeployComponents,
  checkPortAvailability,
  cleanup,
  ensureCluster,
  ensurePortForwarding,
  frontendServerEnvironment,
  getClusterPlatform,
  getClusterStatus,
  getHttpStatus,
  isClusterRunning,
  isDockerRunning,
  isKfpHealthy,
  isKindInstalled,
  isKubectlInstalled,
  isPortInUse,
  localImageTag,
  log,
  preloadSeedRuntimeImage,
  reapplyManifests,
  restoreKubectlContext,
  run,
  spawnProcess,
  startFrontendServer,
  stopFrontendServer,
  teardownCluster,
  terminateChild,
  waitForChildReadiness,
  waitForService,
  waitForTcp,
};
