#!/usr/bin/env node
/**
 * Kind cluster lifecycle and local proxy support for the UI smoke test.
 *
 * A stack instance owns its cluster, kubeconfig, ports, image archives, and child processes. This
 * lets base and head run without sharing Kubernetes, database, MLMD, object-store, or host process
 * state.
 */

const { execFileSync, spawn } = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const http = require('http');
const net = require('net');
const os = require('os');
const path = require('path');
const { applyFixtureRuntimeRequirements } = require('./fixture-runtime-requirements');

const CLUSTER_NAME = 'ui-smoke-test';
const KUBE_CONTEXT = `kind-${CLUSTER_NAME}`;
const NAMESPACE = 'kubeflow';
const FRONTEND_SERVER_PORT = 3001;
const LOOPBACK_HOST = '127.0.0.1';
const LOOPBACK_LISTEN_PRELOAD = `'use strict';
const net = require('node:net');
const originalListen = net.Server.prototype.listen;
net.Server.prototype.listen = function loopbackOnlyListen(...args) {
  const endpoint = args[0];
  if (endpoint && typeof endpoint === 'object' && !Array.isArray(endpoint)) {
    if (Object.hasOwn(endpoint, 'port')) args[0] = { ...endpoint, host: '${LOOPBACK_HOST}' };
  } else if (
    typeof endpoint === 'number' ||
    (typeof endpoint === 'string' && /^[0-9]+$/.test(endpoint))
  ) {
    if (args.length === 1) args.push('${LOOPBACK_HOST}');
    else if (typeof args[1] === 'string') args[1] = '${LOOPBACK_HOST}';
    else if (args[1] === undefined || args[1] === null) args[1] = '${LOOPBACK_HOST}';
    else args.splice(1, 0, '${LOOPBACK_HOST}');
  }
  return originalListen.apply(this, args);
};
`;
const DEFAULT_KUBECONFIG = path.join(os.tmpdir(), 'kfp-ui-smoke-test', 'kubeconfig');
const DEFAULT_PORTS = Object.freeze({
  api: 3002,
  frontendServer: FRONTEND_SERVER_PORT,
  metadata: 9090,
  objectStore: 9000,
});
const PORT_FORWARDS = Object.freeze([
  { service: 'metadata-envoy-service', localPort: 9090, remotePort: 9090 },
  { service: 'ml-pipeline', localPort: 3002, remotePort: 8888 },
  { service: 'seaweedfs', localPort: 9000, remotePort: 9000 },
]);
const SEED_RUNTIME_IMAGE =
  'docker.io/library/busybox@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662';
const KUBEFLOW_FIRST_PARTY_IMAGE_PREFIXES = Object.freeze([
  'ghcr.io/kubeflow/',
  'gcr.io/ml-pipeline/',
]);
// The platform-specific child of SEED_RUNTIME_IMAGE. Using the child manifest here is important:
// a multi-platform index would resolve to arm64 inside an arm64 Kind node and would not prove that
// the node can execute an amd64 workload.
const AMD64_EMULATION_CANARY_IMAGE =
  'docker.io/library/busybox@sha256:b7f3d86d6e84fc17718c48bcde1450807faa2d56704205c697b4bd5df7b9e29f';
// Saving an image by digest makes Kind/containerd import it under an anonymous `import-*` name.
// Apply a deterministic local tag before export so imagePullPolicy Never addresses the preloaded
// image instead of producing ErrImageNeverPull without exercising emulation.
const AMD64_EMULATION_CANARY_LOCAL_IMAGE = 'kfp-ui-smoke/amd64-emulation-canary:b7f3d86d6e84';
// These are the two amd64-only workloads in the 2.17.1 platform-agnostic manifests. The metadata
// writer remains amd64-only when it is built from source because ml-metadata does not publish the
// required arm64 wheel. Keep this list exact and fail closed for other image references: executing
// a newly encountered foreign-architecture image is a trust decision, not a registry-error
// fallback.
const MIXED_PLATFORM_WORKLOADS = Object.freeze([
  Object.freeze({
    component: 'metadata-writer',
    container: 'main',
    deployment: 'metadata-writer',
    image: 'ghcr.io/kubeflow/kfp-metadata-writer:2.17.1',
    platform: 'linux/amd64',
  }),
  Object.freeze({
    container: 'container',
    deployment: 'metadata-grpc-deployment',
    image: 'gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0',
    platform: 'linux/amd64',
  }),
]);
const DIAGNOSTIC_LIMITS = Object.freeze({
  maxOutputBytes: 64 * 1024,
  maxPods: 24,
  tailLines: 200,
});
// A cold frontend image build performs three nested npm installs before lint, typecheck, and the
// production bundle. Twenty minutes can expire during the final image export on arm64 hosts, so
// keep a bounded per-component allowance that still terminates a genuinely stuck build.
const COMPONENT_IMAGE_BUILD_TIMEOUT_MS = 30 * 60 * 1000;
const MYSQL_FINAL_SERVER_TIMEOUT_MS = 5 * 60 * 1000;
const MYSQL_FINAL_SERVER_WAIT_SCRIPT = [
  'attempt=0',
  'while [ "$attempt" -lt 300 ]; do',
  '  [ "$(cat /proc/1/comm 2>/dev/null)" = mysqld ] && exit 0',
  '  attempt=$((attempt + 1))',
  '  sleep 1',
  'done',
  'exit 1',
].join('\n');

// Kept for compatibility with callers that display the historical inventory. Readiness is now
// based on the Deployments rendered by the selected revision.
const PLATFORM_DEPLOYMENTS = Object.freeze([
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
]);

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

function boundedDiagnosticText(value, maxBytes) {
  const text = typeof value === 'string' ? value : value ? String(value) : '';
  const bytes = Buffer.byteLength(text);
  if (bytes <= maxBytes) return { bytes, text, truncated: false };
  const suffix = `\n... truncated to ${maxBytes} bytes ...`;
  const suffixBytes = Buffer.byteLength(suffix);
  const body = Buffer.from(text)
    .subarray(0, Math.max(0, maxBytes - suffixBytes))
    .toString('utf8');
  return { bytes, text: `${body}${suffix}`, truncated: true };
}

function redactDiagnosticText(value) {
  return String(value || '')
    .replace(/:\/\/([^\s/:@]+):([^\s/@]+)@/g, '://<redacted>:<redacted>@')
    .replace(/\bBearer\s+[A-Za-z0-9._~+/-]+=*/gi, 'Bearer <redacted>')
    .replace(
      /([?&](?:access_token|api_key|auth|authorization|cookie|credential|password|secret|token)=)[^&\s]*/gi,
      '$1<redacted>',
    )
    .replace(/\b(authorization|cookie|set-cookie|x-api-key)\s*[:=]\s*[^\r\n]*/gi, '$1: <redacted>')
    .replace(
      /\b([A-Z0-9_]*(?:AUTH|COOKIE|CREDENTIAL|PASSWORD|SECRET|TOKEN|API_KEY)[A-Z0-9_]*)=([^\s,;]+)/g,
      '$1=<redacted>',
    );
}

function validateName(value, description) {
  if (typeof value !== 'string' || !/^[a-z0-9](?:[a-z0-9.-]{0,61}[a-z0-9])?$/.test(value)) {
    throw new Error(`${description} must be a lowercase DNS-compatible name.`);
  }
  return value;
}

function validatePort(value, description) {
  const port = Number(value);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`${description} must be an integer from 1 through 65535.`);
  }
  return port;
}

function validatePorts(value = {}) {
  const ports = {
    api: validatePort(value.api ?? DEFAULT_PORTS.api, 'API port'),
    frontendServer: validatePort(
      value.frontendServer ?? DEFAULT_PORTS.frontendServer,
      'frontend-server port',
    ),
    metadata:
      value.metadata === null
        ? null
        : validatePort(value.metadata ?? DEFAULT_PORTS.metadata, 'metadata port'),
    objectStore: validatePort(value.objectStore ?? DEFAULT_PORTS.objectStore, 'object-store port'),
  };
  const configuredPorts = Object.values(ports).filter((port) => port !== null);
  if (new Set(configuredPorts).size !== configuredPorts.length) {
    throw new Error(
      'A stack must use distinct API, frontend-server, metadata, and object-store ports.',
    );
  }
  return Object.freeze(ports);
}

function validateKubeconfigPath(value) {
  if (typeof value !== 'string' || value.length === 0 || !path.isAbsolute(value)) {
    throw new Error('kubeconfigPath must be an absolute path.');
  }
  return path.normalize(value);
}

function sanitizeImageTagPart(value) {
  const safe = String(value)
    .replace(/[^a-zA-Z0-9_.-]/g, '-')
    .replace(/^[.-]+/, '')
    .slice(0, 120);
  return safe || 'local';
}

function localImageTag(component, suffix) {
  return `${component.imageTag}:${sanitizeImageTagPart(suffix)}`;
}

function componentBuildArguments(component, buildMetadata = {}) {
  const mappings = component.buildArgs || {};
  const args = [];
  for (const [argumentName, metadataName] of Object.entries(mappings)) {
    if (!/^[A-Z][A-Z0-9_]*$/.test(argumentName)) {
      throw new Error(
        `Component ${component.name} declares an invalid Docker build argument: ${argumentName}`,
      );
    }
    const value = buildMetadata[metadataName];
    if (typeof value !== 'string' || value.length === 0 || /[\r\n\0]/.test(value)) {
      throw new Error(
        `Component ${component.name} requires non-empty build metadata ${metadataName}.`,
      );
    }
    args.push('--build-arg', `${argumentName}=${value}`);
  }
  return args;
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
  const failure = new Promise((_resolve, reject) => {
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
  if (!child || child.exitCode !== null || child.signalCode !== null) return true;
  const signalAndWait = (signal, waitTimeout) =>
    new Promise((resolve, reject) => {
      let finished = false;
      let timer;
      const finish = (closed) => {
        if (finished) return;
        finished = true;
        clearTimeout(timer);
        child.off?.('close', onClose);
        resolve(closed || child.exitCode !== null || child.signalCode !== null);
      };
      const onClose = () => finish(true);
      child.once?.('close', onClose);
      timer = setTimeout(() => finish(false), waitTimeout);
      let signaled;
      try {
        signaled = child.kill(signal);
      } catch (error) {
        clearTimeout(timer);
        child.off?.('close', onClose);
        reject(error);
        return;
      }
      if (signaled === false && child.exitCode === null && child.signalCode === null) {
        finish(false);
      }
    });

  if (await signalAndWait('SIGTERM', timeout)) return true;
  if (await signalAndWait('SIGKILL', Math.max(100, Math.min(timeout, 1000)))) return true;
  throw new Error(`Child process ${child.pid || 'unknown'} did not exit after SIGKILL.`);
}

function mergeImageOverrides(...overrides) {
  const merged = { deployments: [], images: {}, runtimeEnvironment: {} };
  for (const value of overrides.filter(Boolean)) {
    Object.assign(merged.images, value.images || {});
    Object.assign(merged.runtimeEnvironment, value.runtimeEnvironment || {});
    merged.deployments.push(...(value.deployments || []));
  }
  return merged;
}

function validatePlatform(value, description) {
  if (typeof value !== 'string' || !/^linux\/[a-z0-9_]+$/.test(value)) {
    throw new Error(`${description} must use linux/<architecture>.`);
  }
  return value;
}

function mixedPlatformWorkloadForImage(image, localWorkloads = new Map()) {
  return (
    localWorkloads.get(image) ||
    MIXED_PLATFORM_WORKLOADS.find((workload) => workload.image === image) ||
    null
  );
}

function imagePlatformForNode(image, nodePlatform, localWorkloads = new Map()) {
  validatePlatform(nodePlatform, 'Kind node platform');
  if (nodePlatform !== 'linux/arm64') return nodePlatform;
  return mixedPlatformWorkloadForImage(image, localWorkloads)?.platform || nodePlatform;
}

function manifestImagePlan(images, nodePlatform, localWorkloads = new Map()) {
  validatePlatform(nodePlatform, 'Kind node platform');
  return images.map((image) =>
    Object.freeze({
      image,
      platform: imagePlatformForNode(image, nodePlatform, localWorkloads),
    }),
  );
}

function isKubeflowFirstPartyImage(image) {
  return KUBEFLOW_FIRST_PARTY_IMAGE_PREFIXES.some((prefix) => image.startsWith(prefix));
}

function createKindStack(config = {}) {
  const clusterName = validateName(config.clusterName || CLUSTER_NAME, 'clusterName');
  const context = validateName(config.context || `kind-${clusterName}`, 'context');
  const namespace = validateName(config.namespace || NAMESPACE, 'namespace');
  const kubeconfigPath = validateKubeconfigPath(config.kubeconfigPath || DEFAULT_KUBECONFIG);
  const ports = validatePorts(config.ports);
  const role = sanitizeImageTagPart(config.role || 'stack');
  const revision = sanitizeImageTagPart(config.revision || 'local');
  const imageScope = sanitizeImageTagPart(config.imageScope || `${role}-${revision}`);
  const isolatedBuildCache = config.isolatedBuildCache === true;
  const buildxBuilderName = validateName(
    `${clusterName.slice(0, 54).replace(/[.-]+$/, '')}-build`,
    'Buildx builder name',
  );
  const archiveDir = path.resolve(
    config.archiveDir || path.join(path.dirname(kubeconfigPath), 'image-archives'),
  );
  const defaultRunner = config.runner || run;
  const defaultSpawn = config.spawn || spawn;
  const processes = [];
  const builtImagePlatforms = new Map();
  const builtMixedPlatformWorkloads = new Map();
  const loadedImages = new Set();
  const ownedPreflightedImages = new Set();
  const preflightedImageIds = new Map();
  const verifiedEmulationPlatforms = new Set();
  let frontendServerProcess = null;
  let seedRuntimeLoaded = false;
  let createdThisRun = false;
  let ownsCluster = false;

  const portForwards = Object.freeze(
    [
      ports.metadata === null
        ? null
        : { service: 'metadata-envoy-service', localPort: ports.metadata, remotePort: 9090 },
      { service: 'ml-pipeline', localPort: ports.api, remotePort: 8888 },
      { service: 'seaweedfs', localPort: ports.objectStore, remotePort: 9000 },
    ].filter(Boolean),
  );
  const deployedUiPortForward = Object.freeze({
    service: 'ml-pipeline-ui',
    localPort: ports.frontendServer,
    remotePort: 80,
  });

  function commandEnvironment(baseEnvironment = process.env) {
    return { ...baseEnvironment, KUBECONFIG: kubeconfigPath };
  }

  function commandOptions(options = {}) {
    return { ...options, env: commandEnvironment(options.env || process.env) };
  }

  function kubectlArgs(...args) {
    return ['--kubeconfig', kubeconfigPath, '--context', context, ...args];
  }

  function stackRunner(options = {}) {
    return options.runner || defaultRunner;
  }

  function spawnProcess(command, args, options = {}) {
    const spawnFn = options.spawnFn || defaultSpawn;
    const spawnOptions = commandOptions({ stdio: 'pipe', ...options });
    delete spawnOptions.spawnFn;
    const child = spawnFn(command, args, spawnOptions);
    child.stdout?.on('data', (data) => {
      if (process.env.VERBOSE) process.stdout.write(data);
    });
    child.stderr?.on('data', (data) => {
      if (process.env.VERBOSE) process.stderr.write(data);
    });
    processes.push(child);
    return child;
  }

  function isKindInstalled(options = {}) {
    return stackRunner(options)('kind', ['version']).success;
  }

  function isKubectlInstalled(options = {}) {
    return stackRunner(options)('kubectl', ['version', '--client']).success;
  }

  function isDockerRunning(options = {}) {
    return stackRunner(options)('docker', ['info']).success;
  }

  function isClusterRunning(options = {}) {
    const result = stackRunner(options)('kind', ['get', 'clusters']);
    return result.success && result.output.split('\n').includes(clusterName);
  }

  async function isKfpHealthy(options = {}) {
    const {
      url = `http://127.0.0.1:${ports.frontendServer}/apis/v2beta1/healthz`,
      get = http.get,
    } = options;
    try {
      return (await getHttpStatus(url, 2000, get)) === 200;
    } catch (error) {
      return false;
    }
  }

  async function getClusterStatus(options = {}) {
    const runner = stackRunner(options);
    const healthCheck = options.healthCheck || isKfpHealthy;
    const status = {
      kindInstalled: isKindInstalled({ runner }),
      kubectlInstalled: isKubectlInstalled({ runner }),
      dockerRunning: isDockerRunning({ runner }),
      clusterRunning: false,
      kfpDeployed: false,
      kfpHealthy: false,
      servicesReady: false,
    };
    if (!status.kindInstalled || !status.kubectlInstalled || !status.dockerRunning) return status;

    status.clusterRunning = isClusterRunning({ runner });
    if (!status.clusterRunning) return status;
    status.kfpDeployed = runner(
      'kubectl',
      kubectlArgs('-n', namespace, 'get', 'deployment', 'ml-pipeline'),
      commandOptions(),
    ).success;
    if (!status.kfpDeployed) return status;
    status.servicesReady = runner(
      'kubectl',
      kubectlArgs(
        '-n',
        namespace,
        'wait',
        '--for=condition=Available',
        '--timeout=1s',
        'deployment/ml-pipeline',
      ),
      commandOptions(),
    ).success;
    status.kfpHealthy = await healthCheck();
    return status;
  }

  function getClusterPlatform(options = {}) {
    const runner = stackRunner(options);
    const result = runner(
      'kubectl',
      kubectlArgs('get', 'nodes', '-o', 'jsonpath={.items[0].status.nodeInfo.architecture}'),
      commandOptions(),
    );
    requireSuccess(result, `Failed to determine the Kind node architecture for ${clusterName}`);
    const architecture = result.output.trim();
    if (!/^[a-z0-9_]+$/.test(architecture)) {
      throw new Error(
        `Kind reported an invalid node architecture: ${JSON.stringify(architecture)}`,
      );
    }
    return `linux/${architecture}`;
  }

  function getDockerPlatform(options = {}) {
    const runner = stackRunner(options);
    const result = runner(
      'docker',
      ['info', '--format', '{{.OSType}}/{{.Architecture}}'],
      commandOptions(),
    );
    requireSuccess(result, 'Failed to determine the Docker server platform');
    const aliases = { aarch64: 'arm64', x86_64: 'amd64' };
    const reported = result.output.trim().toLowerCase();
    const match = reported.match(/^linux\/([a-z0-9_]+)$/);
    if (!match) {
      throw new Error(
        `Docker reported an unsupported server platform: ${JSON.stringify(reported)}`,
      );
    }
    const architecture = aliases[match[1]] || match[1];
    return `linux/${architecture}`;
  }

  function imageArchivePath(stem, platform) {
    return path.join(
      archiveDir,
      `${sanitizeImageTagPart(stem)}-${sanitizeImageTagPart(platform)}.tar`,
    );
  }

  function loadedImageKey(image, platform) {
    return `${validatePlatform(platform, 'Image platform')}\0${image}`;
  }

  function saveAndLoadImage(image, stem, platform, options = {}) {
    const runner = stackRunner(options);
    const imageArchive = imageArchivePath(stem, platform);
    fs.mkdirSync(archiveDir, { recursive: true });
    try {
      exportImageForPlatform(image, imageArchive, platform, {
        nodePlatform: options.nodePlatform || platform,
        runner,
      });
      if (options.removeSourceAfterExport) {
        requireSuccess(
          runner('docker', ['image', 'rm', image], commandOptions()),
          `Failed to release ${image} after exporting it for Kind cluster ${clusterName}`,
        );
      }
      requireSuccess(
        runner(
          'kind',
          ['load', 'image-archive', imageArchive, '--name', clusterName],
          commandOptions({ timeout: 180000 }),
        ),
        `Failed to load ${image} into Kind cluster ${clusterName}`,
      );
      loadedImages.add(loadedImageKey(image, platform));
    } finally {
      fs.rmSync(imageArchive, { force: true });
    }
  }

  function platformImageError(image, platform, action, result, nodePlatform = platform) {
    const detail = result.error || result.output || 'unknown error';
    return new Error(
      `Image ${image} cannot be ${action} for workload platform ${platform} ` +
        `(Kind node platform ${nodePlatform}): ${detail}. Use an image for ${nodePlatform}, or add ` +
        'an exact reviewed workload-platform override and enable container emulation. ' +
        'Unreviewed images never fall back to a different architecture automatically.',
    );
  }

  function pullImageForPlatform(image, platform, options = {}) {
    const runner = stackRunner(options);
    const preflightKey = loadedImageKey(image, platform);
    const expectedImageId = options.reusePreflighted ? preflightedImageIds.get(preflightKey) : null;
    if (expectedImageId) {
      const actualImageId = inspectLocalImageId(image, { runner });
      if (actualImageId !== expectedImageId) {
        throw new Error(
          `Image ${image} changed or disappeared after it was preflighted for ${platform}. ` +
            'Refusing to load an image that was not covered by the successful preflight.',
        );
      }
      return;
    }
    const result = runner(
      'docker',
      ['pull', '--platform', platform, image],
      commandOptions({ timeout: 300000, stdio: 'inherit' }),
    );
    if (!result.success) {
      throw platformImageError(image, platform, 'pulled', result, options.nodePlatform || platform);
    }
  }

  function inspectLocalImageId(image, options = {}) {
    const result = stackRunner(options)(
      'docker',
      ['image', 'inspect', '--format', '{{.Id}}', image],
      commandOptions({ timeout: 30000 }),
    );
    if (!result.success) return null;
    const imageId = result.output.trim();
    return /^sha256:[0-9a-f]{64}$/i.test(imageId) ? imageId.toLowerCase() : null;
  }

  function exportImageForPlatform(image, imageArchive, platform, options = {}) {
    const runner = stackRunner(options);
    const result = runner(
      'docker',
      ['save', '--platform', platform, '--output', imageArchive, image],
      commandOptions({ timeout: 300000, stdio: 'inherit' }),
    );
    if (!result.success) {
      throw platformImageError(
        image,
        platform,
        'exported',
        result,
        options.nodePlatform || platform,
      );
    }
  }

  function verifyImageForPlatform(image, stem, platform, options = {}) {
    const runner = stackRunner(options);
    const imageArchive = imageArchivePath(`preflight-${stem}`, platform);
    const preflightKey = loadedImageKey(image, platform);
    const imageExistedBeforePreflight = inspectLocalImageId(image, { runner }) !== null;
    fs.mkdirSync(archiveDir, { recursive: true });
    try {
      pullImageForPlatform(image, platform, {
        nodePlatform: options.nodePlatform || platform,
        runner,
      });
      exportImageForPlatform(image, imageArchive, platform, {
        nodePlatform: options.nodePlatform || platform,
        runner,
      });
      const imageId = inspectLocalImageId(image, { runner });
      if (imageId) preflightedImageIds.set(preflightKey, imageId);
      else preflightedImageIds.delete(preflightKey);
      if (!imageExistedBeforePreflight && imageId) ownedPreflightedImages.add(preflightKey);
    } finally {
      fs.rmSync(imageArchive, { force: true });
    }
  }

  function preloadSeedRuntimeImage(options = {}) {
    if (seedRuntimeLoaded && options.force !== true) return;
    const runner = stackRunner(options);
    const platform = options.platform || getClusterPlatform({ runner });
    validatePlatform(platform, 'Seed image platform');
    pullImageForPlatform(SEED_RUNTIME_IMAGE, platform, { reusePreflighted: true, runner });
    const preflightKey = loadedImageKey(SEED_RUNTIME_IMAGE, platform);
    const releaseSource =
      options.removeSourceAfterLoad === true && ownedPreflightedImages.has(preflightKey);
    saveAndLoadImage(SEED_RUNTIME_IMAGE, 'seed-runtime', platform, {
      removeSourceAfterExport: releaseSource,
      runner,
    });
    if (releaseSource) {
      ownedPreflightedImages.delete(preflightKey);
      preflightedImageIds.delete(preflightKey);
    }
    seedRuntimeLoaded = true;
  }

  function preflightSeedRuntimeImage(options = {}) {
    const runner = stackRunner(options);
    const platform = options.platform || getDockerPlatform({ runner });
    validatePlatform(platform, 'Seed image platform');
    verifyImageForPlatform(SEED_RUNTIME_IMAGE, 'seed-runtime', platform, { runner });
    return { image: SEED_RUNTIME_IMAGE, platform };
  }

  function scopedImageTag(component, suffix = imageScope) {
    return localImageTag(component, `${clusterName}-${suffix}`);
  }

  async function buildComponentImages(components, repoRoot, options = {}) {
    const runner = stackRunner(options);
    const overrides = { deployments: [], images: {}, runtimeEnvironment: {} };
    if (components.length === 0) return overrides;
    const platform = options.platform || getClusterPlatform({ runner });
    validatePlatform(platform, 'Backend image platform');
    const tagSuffix = sanitizeImageTagPart(options.tagSuffix || imageScope);
    const builtImages = [];
    try {
      for (const component of components) {
        if (!component.dockerfile || !component.imageTag) {
          throw new Error(`Component ${component.name} is missing Docker build metadata.`);
        }
        const image = scopedImageTag(component, tagSuffix);
        const mixedPlatformWorkload = MIXED_PLATFORM_WORKLOADS.find(
          (workload) =>
            workload.component === component.name &&
            workload.deployment === component.deployment &&
            workload.container === component.container,
        );
        const buildPlatform =
          platform === 'linux/arm64' && mixedPlatformWorkload
            ? mixedPlatformWorkload.platform
            : platform;
        validatePlatform(buildPlatform, `Build platform for ${component.name}`);
        const buildArguments = componentBuildArguments(component, options.buildMetadata);
        overrides.images[component.name] = image;
        log(`Building ${component.name} as ${image}...`);
        const buildImage = () =>
          requireSuccess(
            runner(
              'docker',
              isolatedBuildCache
                ? [
                    'buildx',
                    'build',
                    '--builder',
                    buildxBuilderName,
                    '--load',
                    '--platform',
                    buildPlatform,
                    '--tag',
                    image,
                    '--file',
                    component.dockerfile,
                    ...buildArguments,
                    '.',
                  ]
                : [
                    'build',
                    '--platform',
                    buildPlatform,
                    '--tag',
                    image,
                    '--file',
                    component.dockerfile,
                    ...buildArguments,
                    '.',
                  ],
              commandOptions({
                cwd: repoRoot,
                timeout: COMPONENT_IMAGE_BUILD_TIMEOUT_MS,
                stdio: 'inherit',
              }),
            ),
            `Failed to build ${component.name}`,
          );
        if (isolatedBuildCache) {
          requireSuccess(
            runner(
              'docker',
              ['buildx', 'create', '--name', buildxBuilderName, '--driver', 'docker-container'],
              commandOptions(),
            ),
            `Failed to create isolated Buildx builder ${buildxBuilderName}`,
          );
          let buildError;
          try {
            buildImage();
          } catch (error) {
            buildError = error;
          }
          const removal = runner(
            'docker',
            ['buildx', 'rm', '--force', buildxBuilderName],
            commandOptions({ timeout: COMPONENT_IMAGE_BUILD_TIMEOUT_MS, stdio: 'inherit' }),
          );
          const cleanupError = removal.success
            ? undefined
            : new Error(
                `Failed to remove isolated Buildx builder ${buildxBuilderName}: ${
                  removal.error || removal.output || 'unknown error'
                }`,
              );
          if (buildError && cleanupError) {
            throw new AggregateError(
              [buildError, cleanupError],
              `Image build and isolated Buildx cleanup both failed for ${component.name}`,
            );
          }
          if (buildError) throw buildError;
          builtImages.push(image);
          if (cleanupError) throw cleanupError;
        } else {
          buildImage();
          builtImages.push(image);
        }
        builtImagePlatforms.set(image, buildPlatform);
        if (buildPlatform !== platform) {
          builtMixedPlatformWorkloads.set(
            image,
            Object.freeze({ ...mixedPlatformWorkload, image }),
          );
        }
        if (options.load !== false) {
          saveAndLoadImage(image, `component-${component.name}-${tagSuffix}`, buildPlatform, {
            nodePlatform: platform,
            runner,
          });
        }
        if (component.deployment) {
          overrides.deployments.push({
            container: component.container,
            deployment: component.deployment,
            image,
          });
        }
        if (component.runtimeEnv) overrides.runtimeEnvironment[component.runtimeEnv] = image;
      }
    } catch (error) {
      const cleanupErrors = [];
      for (const image of builtImages.reverse()) {
        const removal = runner('docker', ['image', 'rm', image], commandOptions());
        if (!removal.success) {
          cleanupErrors.push(
            new Error(
              `Failed to remove incomplete build image ${image}: ${
                removal.error || removal.output || 'unknown error'
              }`,
            ),
          );
        } else {
          builtImagePlatforms.delete(image);
          builtMixedPlatformWorkloads.delete(image);
        }
      }
      if (cleanupErrors.length === 0) throw error;
      throw new AggregateError(
        [error, ...cleanupErrors],
        `Component image build and partial-image cleanup both failed for ${clusterName}`,
      );
    }
    return overrides;
  }

  function reuseComponentImages(components, sourceOverrides, options = {}) {
    const runner = stackRunner(options);
    const platform = options.platform || getClusterPlatform({ runner });
    validatePlatform(platform, 'Reused image platform');
    const tagSuffix = sanitizeImageTagPart(options.tagSuffix || imageScope);
    const overrides = { deployments: [], images: {}, runtimeEnvironment: {} };
    for (const component of components) {
      const sourceImage = sourceOverrides?.images?.[component.name];
      if (!sourceImage) {
        throw new Error(`Cannot reuse ${component.name}: the base image override is missing.`);
      }
      const image = scopedImageTag(component, tagSuffix);
      requireSuccess(
        runner('docker', ['image', 'tag', sourceImage, image], commandOptions()),
        `Failed to reuse ${component.name}`,
      );
      builtImagePlatforms.set(image, platform);
      overrides.images[component.name] = image;
      if (component.deployment) {
        overrides.deployments.push({
          container: component.container,
          deployment: component.deployment,
          image,
        });
      }
      if (component.runtimeEnv) overrides.runtimeEnvironment[component.runtimeEnv] = image;
    }
    return overrides;
  }

  function loadImageOverrides(imageOverrides, platform, options = {}) {
    const runner = stackRunner(options);
    validatePlatform(platform, 'Local image platform');
    const images = [
      ...new Set([
        ...Object.values(imageOverrides?.images || {}),
        ...Object.values(imageOverrides?.runtimeEnvironment || {}),
      ]),
    ];
    for (const [index, image] of images.entries()) {
      if (typeof image !== 'string' || image.length === 0) {
        throw new Error('Local image overrides must contain non-empty image references.');
      }
      const imagePlatform = builtImagePlatforms.get(image) || platform;
      if (loadedImages.has(loadedImageKey(image, imagePlatform))) continue;
      saveAndLoadImage(image, `local-image-${index}`, imagePlatform, {
        nodePlatform: platform,
        removeSourceAfterExport: options.removeSourceAfterLoad,
        runner,
      });
    }
    return images;
  }

  function deploymentPatches(imageOverrides, pullPolicyOverrides = []) {
    const byDeployment = new Map();
    const containerPatch = (deployment, container) => {
      if (!byDeployment.has(deployment)) byDeployment.set(deployment, new Map());
      const containers = byDeployment.get(deployment);
      if (!containers.has(container)) containers.set(container, { name: container });
      return containers.get(container);
    };
    for (const override of imageOverrides.deployments || []) {
      if (!override.deployment || !override.container || !override.image) {
        throw new Error('Deployment image overrides require deployment, container, and image.');
      }
      Object.assign(containerPatch(override.deployment, override.container), {
        image: override.image,
        imagePullPolicy: 'IfNotPresent',
      });
    }
    for (const override of pullPolicyOverrides) {
      if (
        !override?.deployment ||
        !override.container ||
        override.imagePullPolicy !== 'IfNotPresent'
      ) {
        throw new Error(
          'Deployment pull-policy overrides require deployment, container, and IfNotPresent.',
        );
      }
      containerPatch(override.deployment, override.container).imagePullPolicy = 'IfNotPresent';
    }

    const runtimeEntries = Object.entries(imageOverrides.runtimeEnvironment || {});
    if (runtimeEntries.length > 0) {
      const apiServer = containerPatch('ml-pipeline', 'ml-pipeline-api-server');
      apiServer.env = runtimeEntries.map(([name, value]) => ({ name, value }));
    }

    return [...byDeployment.entries()].map(([deployment, containers]) => ({
      patch: JSON.stringify({
        apiVersion: 'apps/v1',
        kind: 'Deployment',
        metadata: { name: deployment },
        spec: { template: { spec: { containers: [...containers.values()] } } },
      }),
      target: { group: 'apps', kind: 'Deployment', name: deployment, version: 'v1' },
    }));
  }

  function renderRevisionManifests(repoRoot, imageOverrides, options = {}) {
    const runner = stackRunner(options);
    const overlayDir = path.join(archiveDir, 'manifest-overlay');
    const renderedPath = path.join(archiveDir, 'platform-agnostic.yaml');
    const platformAgnostic = path.join(
      repoRoot,
      'manifests',
      'kustomize',
      'env',
      'platform-agnostic',
    );
    const patches = deploymentPatches(imageOverrides, options.pullPolicyOverrides || []);
    const needsOverlay = patches.length > 0;
    fs.mkdirSync(archiveDir, { recursive: true });
    fs.rmSync(renderedPath, { force: true });
    fs.rmSync(overlayDir, { force: true, recursive: true });
    let renderRoot = platformAgnostic;
    const renderArguments = [];
    if (needsOverlay) {
      fs.mkdirSync(overlayDir, { recursive: true });
      const canonicalOverlayDir = fs.realpathSync(overlayDir);
      const canonicalPlatformAgnostic = fs.realpathSync(platformAgnostic);
      const platformAgnosticResource = path.relative(
        canonicalOverlayDir,
        canonicalPlatformAgnostic,
      );
      if (!platformAgnosticResource || path.isAbsolute(platformAgnosticResource)) {
        throw new Error(
          'Could not express the platform-agnostic manifests relative to their overlay.',
        );
      }
      if (
        path.resolve(canonicalOverlayDir, platformAgnosticResource) !== canonicalPlatformAgnostic
      ) {
        throw new Error('The platform-agnostic overlay resource resolved to an unexpected path.');
      }
      fs.writeFileSync(
        path.join(overlayDir, 'kustomization.yaml'),
        JSON.stringify(
          {
            apiVersion: 'kustomize.config.k8s.io/v1beta1',
            kind: 'Kustomization',
            patches,
            resources: [platformAgnosticResource],
          },
          null,
          2,
        ),
      );
      renderRoot = overlayDir;
      renderArguments.push('--load-restrictor=LoadRestrictionsNone');
    }
    const renderResult = runner(
      'kubectl',
      ['kustomize', renderRoot, ...renderArguments, '--output', renderedPath],
      commandOptions({ timeout: 180000 }),
    );
    requireSuccess(
      renderResult,
      `Failed to render revision manifests with exact image overrides for ${clusterName}`,
    );
    // Test runners commonly return rendered text instead of implementing kubectl's --output flag.
    if (!fs.existsSync(renderedPath) && renderResult.output) {
      fs.writeFileSync(renderedPath, renderResult.output);
    }
    if (!fs.existsSync(renderedPath)) {
      throw new Error(`kubectl kustomize did not create ${renderedPath}.`);
    }
    return { overlayDir, renderedPath };
  }

  function extractManifestImages(manifestContents) {
    const images = [];
    let runtimeImageName = null;
    for (const line of manifestContents.split(/\r?\n/)) {
      const embeddedKubeflowImages = line.match(
        /ghcr\.io\/kubeflow\/[a-z0-9._/-]+(?::[a-z0-9._-]+|@sha256:[a-f0-9]{64})/gi,
      );
      if (embeddedKubeflowImages) images.push(...embeddedKubeflowImages);
      const imageMatch = line.match(/^\s*(?:-\s*)?image:\s*["']?([^\s"'#]+)["']?\s*(?:#.*)?$/);
      if (imageMatch) images.push(imageMatch[1]);

      const nameMatch = line.match(/^\s*-?\s*name:\s*(V2_(?:DRIVER|LAUNCHER)_IMAGE)\s*$/);
      if (nameMatch) {
        runtimeImageName = nameMatch[1];
        continue;
      }
      if (runtimeImageName) {
        const valueMatch = line.match(/^\s*value:\s*["']?([^\s"'#]+)["']?\s*(?:#.*)?$/);
        if (valueMatch) images.push(valueMatch[1]);
        if (line.trim() && !/^\s*(?:value:|valueFrom:)/.test(line)) runtimeImageName = null;
      }
    }
    return [...new Set(images)];
  }

  function mixedPlatformPullPolicyOverrides(images, nodePlatform) {
    if (nodePlatform !== 'linux/arm64') return [];
    return images
      .map((image) => mixedPlatformWorkloadForImage(image, builtMixedPlatformWorkloads))
      .filter((workload) => workload && workload.platform !== nodePlatform)
      .map((workload) => ({
        container: workload.container,
        deployment: workload.deployment,
        imagePullPolicy: 'IfNotPresent',
      }));
  }

  function manifestContainerImages(manifestContents) {
    // Keep YAML out of the module's startup path so help and teardown remain usable before the
    // smoke-test dependencies are restored.
    const { parseAllDocuments } = require('yaml');
    const documents = parseAllDocuments(manifestContents, { maxAliasCount: 100 });
    const errors = documents.flatMap((document) => document.errors || []);
    if (errors.length > 0) {
      throw new Error(`Rendered revision manifests are invalid YAML: ${errors[0].message}`);
    }
    const containers = [];
    for (const document of documents) {
      const resource = document.toJS({ maxAliasCount: 100 });
      if (!resource || typeof resource !== 'object') continue;
      const seen = new WeakSet();
      const visit = (value) => {
        if (!value || typeof value !== 'object' || seen.has(value)) return;
        seen.add(value);
        for (const containerType of ['containers', 'initContainers']) {
          if (!Array.isArray(value[containerType])) continue;
          for (const container of value[containerType]) {
            if (!container || typeof container.image !== 'string') continue;
            containers.push({
              container: typeof container.name === 'string' ? container.name : null,
              containerType,
              deployment: resource.kind === 'Deployment' ? resource.metadata?.name || null : null,
              environment: Array.isArray(container.env)
                ? container.env.map((entry) => ({
                    name: typeof entry?.name === 'string' ? entry.name : null,
                    value: typeof entry?.value === 'string' ? entry.value : null,
                  }))
                : [],
              image: container.image,
              imagePullPolicy:
                typeof container.imagePullPolicy === 'string' ? container.imagePullPolicy : null,
              kind: typeof resource.kind === 'string' ? resource.kind : null,
              resource: resource.metadata?.name || null,
            });
          }
        }
        for (const nested of Array.isArray(value) ? value : Object.values(value)) visit(nested);
      };
      visit(resource);
    }
    return containers;
  }

  function validateMixedPlatformPullPolicies(manifestContents, imagePlan, nodePlatform) {
    const mixedPlatformImages = imagePlan.filter(({ platform }) => platform !== nodePlatform);
    if (mixedPlatformImages.length === 0) return;
    const containerImages = manifestContainerImages(manifestContents);
    for (const { image } of mixedPlatformImages) {
      const workload = mixedPlatformWorkloadForImage(image, builtMixedPlatformWorkloads);
      if (!workload) {
        throw new Error(`Mixed-platform image ${image} does not have an exact reviewed workload.`);
      }
      const occurrences = containerImages.filter((entry) => entry.image === image);
      if (occurrences.length === 0) {
        throw new Error(
          `Mixed-platform image ${image} was discovered outside a supported workload container.`,
        );
      }
      const unexpectedPlacement = occurrences.find(
        (entry) =>
          entry.kind !== 'Deployment' ||
          entry.deployment !== workload.deployment ||
          entry.containerType !== 'containers' ||
          entry.container !== workload.container,
      );
      if (unexpectedPlacement) {
        throw new Error(
          `Mixed-platform image ${image} is only approved for Deployment ` +
            `${workload.deployment}, container ${workload.container}; rendered placement was ` +
            `${unexpectedPlacement.kind || 'unknown kind'} ` +
            `${unexpectedPlacement.resource || 'unknown resource'}, ` +
            `${unexpectedPlacement.containerType}/${unexpectedPlacement.container || 'unnamed'}.`,
        );
      }
      if (occurrences.some(({ imagePullPolicy }) => imagePullPolicy !== 'IfNotPresent')) {
        throw new Error(
          `Mixed-platform image ${image} must use imagePullPolicy IfNotPresent after it is preloaded.`,
        );
      }
    }
  }

  function renderRevisionManifestsForPlatform(
    repoRoot,
    imageOverrides,
    nodePlatform,
    options = {},
  ) {
    const runner = stackRunner(options);
    validatePlatform(nodePlatform, 'Kind node platform');
    let rendered = renderRevisionManifests(repoRoot, imageOverrides, { runner });
    try {
      let manifestContents = fs.readFileSync(rendered.renderedPath, 'utf8');
      let images = extractManifestImages(manifestContents);
      const pullPolicyOverrides = mixedPlatformPullPolicyOverrides(images, nodePlatform);
      if (pullPolicyOverrides.length > 0) {
        cleanRenderedManifests(rendered);
        rendered = renderRevisionManifests(repoRoot, imageOverrides, {
          pullPolicyOverrides,
          runner,
        });
        manifestContents = fs.readFileSync(rendered.renderedPath, 'utf8');
        const patchedImages = extractManifestImages(manifestContents);
        if (JSON.stringify(patchedImages) !== JSON.stringify(images)) {
          throw new Error('Pull-policy patches unexpectedly changed the rendered image inventory.');
        }
        images = patchedImages;
      }
      const configuredManifest = applyFixtureRuntimeRequirements(
        manifestContents,
        options.fixtureRequirements,
      );
      if (configuredManifest !== manifestContents) {
        fs.writeFileSync(rendered.renderedPath, configuredManifest);
        manifestContents = configuredManifest;
      }
      const imagePlan = manifestImagePlan(images, nodePlatform, builtMixedPlatformWorkloads);
      validateMixedPlatformPullPolicies(manifestContents, imagePlan, nodePlatform);
      return { ...rendered, imagePlan, images, manifestContents };
    } catch (error) {
      cleanRenderedManifests(rendered);
      throw error;
    }
  }

  function verifyAmd64WorkloadEmulation(imagePlan, nodePlatform, options = {}) {
    const runner = stackRunner(options);
    if (
      nodePlatform !== 'linux/arm64' ||
      !imagePlan.some(({ platform }) => platform === 'linux/amd64') ||
      verifiedEmulationPlatforms.has('linux/amd64')
    ) {
      return { required: false, verified: true };
    }

    const image = AMD64_EMULATION_CANARY_LOCAL_IMAGE;
    const canaryImageKey = loadedImageKey(image, 'linux/amd64');
    if (!loadedImages.has(canaryImageKey)) {
      pullImageForPlatform(AMD64_EMULATION_CANARY_IMAGE, 'linux/amd64', {
        nodePlatform,
        runner,
      });
      requireSuccess(
        runner(
          'docker',
          ['tag', AMD64_EMULATION_CANARY_IMAGE, image],
          commandOptions({ timeout: 30000 }),
        ),
        'Failed to tag the pinned amd64 emulation canary image',
      );
      saveAndLoadImage(image, 'amd64-emulation-canary', 'linux/amd64', {
        nodePlatform,
        runner,
      });
    }

    const jobName = 'ui-smoke-amd64-canary';
    const canaryPath = path.join(archiveDir, `${jobName}.json`);
    fs.mkdirSync(archiveDir, { recursive: true });
    fs.writeFileSync(
      canaryPath,
      JSON.stringify({
        apiVersion: 'batch/v1',
        kind: 'Job',
        metadata: { name: jobName, namespace: 'default' },
        spec: {
          backoffLimit: 0,
          template: {
            metadata: { labels: { 'ui-smoke-test': 'amd64-emulation-canary' } },
            spec: {
              containers: [
                {
                  command: ['/bin/sh', '-c', 'exit 0'],
                  image,
                  imagePullPolicy: 'Never',
                  name: 'canary',
                  securityContext: {
                    allowPrivilegeEscalation: false,
                    capabilities: { drop: ['ALL'] },
                    runAsNonRoot: true,
                    runAsUser: 65534,
                  },
                },
              ],
              restartPolicy: 'Never',
              securityContext: { seccompProfile: { type: 'RuntimeDefault' } },
            },
          },
        },
      }),
      { mode: 0o600 },
    );

    let canaryError = null;
    try {
      const applyResult = runner(
        'kubectl',
        kubectlArgs('apply', '-f', canaryPath),
        commandOptions({ timeout: 30000 }),
      );
      if (!applyResult.success) {
        throw new Error(applyResult.error || applyResult.output || 'could not create canary Job');
      }
      const waitResult = runner(
        'kubectl',
        kubectlArgs(
          '-n',
          'default',
          'wait',
          '--for=condition=complete',
          '--timeout=60s',
          `job/${jobName}`,
        ),
        commandOptions({ timeout: 70000 }),
      );
      if (!waitResult.success) {
        throw new Error(waitResult.error || waitResult.output || 'canary Job did not complete');
      }
    } catch (error) {
      canaryError = new Error(
        `The arm64 Kind cluster ${clusterName} cannot execute a preloaded amd64 workload: ` +
          `${error.message}. Enable Docker workload emulation (QEMU or Rosetta) and ` +
          'keep the Kind node on arm64.',
        { cause: error },
      );
    }

    const deleteResult = runner(
      'kubectl',
      kubectlArgs(
        '-n',
        'default',
        'delete',
        `job/${jobName}`,
        '--ignore-not-found=true',
        '--wait=true',
      ),
      commandOptions({ timeout: 30000 }),
    );
    fs.rmSync(canaryPath, { force: true });
    const cleanupError = deleteResult.success
      ? null
      : new Error(
          `Failed to remove amd64 emulation canary from ${clusterName}: ${deleteResult.error || deleteResult.output || 'unknown error'}`,
        );
    if (canaryError && cleanupError) {
      throw new AggregateError(
        [canaryError, cleanupError],
        `amd64 workload emulation failed and its canary could not be removed from ${clusterName}.`,
      );
    }
    if (canaryError) throw canaryError;
    if (cleanupError) throw cleanupError;

    verifiedEmulationPlatforms.add('linux/amd64');
    return { image, required: true, verified: true };
  }

  function validateReleaseManifestImages(images, release) {
    if (!/^\d+\.\d+\.\d+$/.test(release)) {
      throw new Error(`Expected release must be an exact semantic version, received ${release}.`);
    }
    const firstPartyImages = images.filter(isKubeflowFirstPartyImage);
    if (firstPartyImages.length === 0) {
      throw new Error(
        `Rendered manifests did not contain any Kubeflow release images for ${release}.`,
      );
    }
    const mismatched = firstPartyImages.filter((image) => !image.endsWith(`:${release}`));
    if (mismatched.length > 0) {
      throw new Error(
        `Rendered Kubeflow images do not match release ${release}: ${mismatched.join(', ')}.`,
      );
    }
    return firstPartyImages;
  }

  function validateLocalManifestImages(images, imageOverrides, manifestContents) {
    const firstPartyImages = images.filter(isKubeflowFirstPartyImage);
    if (firstPartyImages.length > 0) {
      throw new Error(
        `Rendered revision manifests retain non-local Kubeflow images: ${firstPartyImages.join(', ')}. ` +
          'Add an exact component build and manifest override before applying the local stack.',
      );
    }
    const expectedImages = new Set([
      ...Object.values(imageOverrides.images || {}),
      ...Object.values(imageOverrides.runtimeEnvironment || {}),
    ]);
    const missing = [...expectedImages].filter((image) => !images.includes(image));
    if (missing.length > 0) {
      throw new Error(
        `Rendered revision manifests omitted locally built images: ${missing.join(', ')}.`,
      );
    }

    const deploymentOverrides = imageOverrides.deployments || [];
    const runtimeEnvironment = Object.entries(imageOverrides.runtimeEnvironment || {});
    if (deploymentOverrides.length > 0 || runtimeEnvironment.length > 0) {
      const containers = manifestContainerImages(manifestContents);
      for (const override of deploymentOverrides) {
        const matches = containers.filter(
          (entry) =>
            entry.kind === 'Deployment' &&
            entry.deployment === override.deployment &&
            entry.containerType === 'containers' &&
            entry.container === override.container,
        );
        if (matches.length !== 1 || matches[0].image !== override.image) {
          throw new Error(
            `Rendered revision manifests did not set Deployment ${override.deployment}, ` +
              `container ${override.container} to locally built image ${override.image}.`,
          );
        }
      }

      const apiServerContainers = containers.filter(
        (entry) =>
          entry.kind === 'Deployment' &&
          entry.deployment === 'ml-pipeline' &&
          entry.containerType === 'containers' &&
          entry.container === 'ml-pipeline-api-server',
      );
      for (const [name, value] of runtimeEnvironment) {
        const matchingEnvironment = apiServerContainers.flatMap((container) =>
          container.environment.filter((entry) => entry.name === name),
        );
        if (matchingEnvironment.length !== 1 || matchingEnvironment[0].value !== value) {
          throw new Error(
            `Rendered revision manifests did not set ml-pipeline/ml-pipeline-api-server ` +
              `${name} to locally built image ${value}.`,
          );
        }
      }
    }
    return [...expectedImages];
  }

  function preloadManifestImages(manifestPath, platform, options = {}) {
    const runner = stackRunner(options);
    validatePlatform(platform, 'Manifest node platform');
    const images = extractManifestImages(fs.readFileSync(manifestPath, 'utf8'));
    const imagePlan = manifestImagePlan(images, platform, builtMixedPlatformWorkloads);
    validateMixedPlatformPullPolicies(fs.readFileSync(manifestPath, 'utf8'), imagePlan, platform);
    for (const [index, { image, platform: imagePlatform }] of imagePlan.entries()) {
      if (loadedImages.has(loadedImageKey(image, imagePlatform))) continue;
      pullImageForPlatform(image, imagePlatform, {
        nodePlatform: platform,
        reusePreflighted: true,
        runner,
      });
      const preflightKey = loadedImageKey(image, imagePlatform);
      const releaseSource =
        options.removeSourceAfterLoad === true && ownedPreflightedImages.has(preflightKey);
      saveAndLoadImage(image, `manifest-image-${index}`, imagePlatform, {
        nodePlatform: platform,
        removeSourceAfterExport: releaseSource,
        runner,
      });
      if (releaseSource) {
        ownedPreflightedImages.delete(preflightKey);
        preflightedImageIds.delete(preflightKey);
      }
    }
    verifyAmd64WorkloadEmulation(imagePlan, platform, { runner });
    return images;
  }

  function cleanRenderedManifests(rendered) {
    fs.rmSync(rendered.overlayDir, { force: true, recursive: true });
    fs.rmSync(rendered.renderedPath, { force: true });
  }

  function preflightReleaseImages(repoRoot, options = {}) {
    const runner = stackRunner(options);
    const platform = options.platform || getDockerPlatform({ runner });
    const rendered = renderRevisionManifestsForPlatform(
      repoRoot,
      mergeImageOverrides(options.imageOverrides),
      platform,
      { runner },
    );
    try {
      const { imagePlan, images } = rendered;
      if (options.expectedRelease) validateReleaseManifestImages(images, options.expectedRelease);
      for (const [index, { image, platform: imagePlatform }] of imagePlan.entries()) {
        verifyImageForPlatform(image, `release-image-${index}`, imagePlatform, {
          nodePlatform: platform,
          runner,
        });
      }
      return {
        imagePlatforms: Object.fromEntries(
          imagePlan.map(({ image, platform }) => [image, platform]),
        ),
        images,
        platform,
      };
    } finally {
      cleanRenderedManifests(rendered);
    }
  }

  function preflightThirdPartyImages(repoRoot, options = {}) {
    const runner = stackRunner(options);
    const platform = options.platform || getDockerPlatform({ runner });
    const rendered = renderRevisionManifestsForPlatform(
      repoRoot,
      mergeImageOverrides(options.imageOverrides),
      platform,
      { runner },
    );
    try {
      const imagePlan = rendered.imagePlan.filter(({ image }) => !isKubeflowFirstPartyImage(image));
      for (const [index, { image, platform: imagePlatform }] of imagePlan.entries()) {
        verifyImageForPlatform(image, `third-party-image-${index}`, imagePlatform, {
          nodePlatform: platform,
          runner,
        });
      }
      return {
        imagePlatforms: Object.fromEntries(
          imagePlan.map(({ image, platform }) => [image, platform]),
        ),
        images: imagePlan.map(({ image }) => image),
        platform,
      };
    } finally {
      cleanRenderedManifests(rendered);
    }
  }

  function collectDiagnostics(options = {}) {
    const runner = stackRunner(options);
    const maxOutputBytes = Number(options.maxOutputBytes ?? DIAGNOSTIC_LIMITS.maxOutputBytes);
    const maxPods = Number(options.maxPods ?? DIAGNOSTIC_LIMITS.maxPods);
    const tailLines = Number(options.tailLines ?? DIAGNOSTIC_LIMITS.tailLines);
    if (
      !Number.isInteger(maxOutputBytes) ||
      maxOutputBytes < 1024 ||
      maxOutputBytes > 1024 * 1024
    ) {
      throw new Error('Diagnostic maxOutputBytes must be an integer from 1024 through 1048576.');
    }
    if (!Number.isInteger(maxPods) || maxPods < 1 || maxPods > 100) {
      throw new Error('Diagnostic maxPods must be an integer from 1 through 100.');
    }
    if (!Number.isInteger(tailLines) || tailLines < 1 || tailLines > 1000) {
      throw new Error('Diagnostic tailLines must be an integer from 1 through 1000.');
    }

    const artifactRoot = path.resolve(options.artifactRoot || archiveDir);
    const outputDir = path.resolve(options.outputDir || path.join(artifactRoot, 'diagnostics'));
    const relativeOutputDir = path.relative(artifactRoot, outputDir);
    if (
      relativeOutputDir === '..' ||
      relativeOutputDir.startsWith(`..${path.sep}`) ||
      path.isAbsolute(relativeOutputDir)
    ) {
      throw new Error('Diagnostic outputDir must stay inside artifactRoot.');
    }
    fs.mkdirSync(outputDir, { recursive: true });

    const diagnosticCommand = (name, args) => {
      const scopedArgs = kubectlArgs(...args);
      const result = runner(
        'kubectl',
        scopedArgs,
        commandOptions({ maxBuffer: maxOutputBytes * 2, timeout: 30000 }),
      );
      const output = boundedDiagnosticText(redactDiagnosticText(result.output), maxOutputBytes);
      const error = boundedDiagnosticText(
        redactDiagnosticText(result.error),
        Math.min(maxOutputBytes, 16 * 1024),
      );
      const filename = `${sanitizeImageTagPart(name)}.log`;
      const artifactPath = path.join(outputDir, filename);
      const artifactContents = [
        `command: kubectl ${scopedArgs
          .map((argument) => (argument === kubeconfigPath ? '<run-scoped-kubeconfig>' : argument))
          .join(' ')}`,
        `success: ${result.success === true}`,
        '',
        output.text,
        error.text ? `\nstderr/error:\n${error.text}` : '',
      ].join('\n');
      const preview = boundedDiagnosticText(artifactContents, Math.min(maxOutputBytes, 32 * 1024));
      fs.writeFileSync(artifactPath, artifactContents, { mode: 0o600 });
      return {
        artifactPath: path.relative(artifactRoot, artifactPath).split(path.sep).join('/'),
        bytes: Buffer.byteLength(artifactContents),
        command: [
          'kubectl',
          ...scopedArgs.map((argument) =>
            argument === kubeconfigPath ? '<run-scoped-kubeconfig>' : argument,
          ),
        ],
        diagnosticOutput: output.text,
        name,
        outputBytes: output.bytes,
        preview: preview.text,
        sha256: crypto.createHash('sha256').update(artifactContents).digest('hex'),
        success: result.success === true,
        truncated: output.truncated || error.truncated || preview.truncated,
      };
    };

    const diagnostic = {
      clusterName,
      collected: ownsCluster,
      context,
      limits: { maxOutputBytes, maxPods, tailLines },
      logs: [],
      namespace,
      owned: ownsCluster,
      role,
      status: [],
    };
    if (!ownsCluster) {
      diagnostic.reason = 'cluster_not_owned';
      return diagnostic;
    }

    diagnostic.status.push(
      diagnosticCommand('deployments', ['-n', namespace, 'get', 'deployments', '-o', 'wide']),
      diagnosticCommand('pods', ['-n', namespace, 'get', 'pods', '-o', 'wide']),
      diagnosticCommand('events', ['-n', namespace, 'get', 'events', '--sort-by=.lastTimestamp']),
    );
    const podInventory = diagnosticCommand('pod-names', [
      '-n',
      namespace,
      'get',
      'pods',
      '-o',
      'jsonpath={range .items[*]}{.metadata.name}{"\\n"}{end}',
    ]);
    if (!podInventory.success) {
      delete podInventory.diagnosticOutput;
      diagnostic.status.push(podInventory);
      for (const status of diagnostic.status) delete status.diagnosticOutput;
      return diagnostic;
    }
    const allPodNames = [
      ...new Set(podInventory.diagnosticOutput.split('\n').map((name) => name.trim())),
    ]
      .filter((name) => /^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$/.test(name))
      .sort();
    const podNames = allPodNames
      .filter((name) =>
        PLATFORM_DEPLOYMENTS.some(
          (deployment) => name === deployment || name.startsWith(`${deployment}-`),
        ),
      )
      .slice(0, maxPods);
    delete podInventory.diagnosticOutput;
    diagnostic.status.push(podInventory);
    diagnostic.ignoredPodCount = allPodNames.length - podNames.length;
    diagnostic.podCount = podNames.length;
    for (const podName of podNames) {
      const podLog = diagnosticCommand(`pod-${podName}`, [
        '-n',
        namespace,
        'logs',
        `pod/${podName}`,
        '--all-containers=true',
        `--tail=${tailLines}`,
        '--timestamps=true',
        '--prefix=true',
      ]);
      delete podLog.diagnosticOutput;
      diagnostic.logs.push(podLog);
    }
    for (const status of diagnostic.status) delete status.diagnosticOutput;
    return diagnostic;
  }

  function listDeployments(options = {}) {
    const runner = stackRunner(options);
    const result = runner(
      'kubectl',
      kubectlArgs(
        '-n',
        namespace,
        'get',
        'deployments',
        '-o',
        'jsonpath={range .items[*]}{.metadata.name}{"\\n"}{end}',
      ),
      commandOptions(),
    );
    requireSuccess(result, `Failed to list Deployments in ${namespace} for ${clusterName}`);
    const deployments = result.output
      .split('\n')
      .map((name) => name.trim())
      .filter(Boolean);
    if (deployments.length === 0) {
      throw new Error(`No Deployments were rendered for ${clusterName} in namespace ${namespace}.`);
    }
    const invalid = deployments.find((name) => !/^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$/.test(name));
    if (invalid) {
      throw new Error(`kubectl returned an invalid Deployment name: ${JSON.stringify(invalid)}`);
    }
    return [...new Set(deployments)].sort();
  }

  function applyKfpManifests(repoRoot, options = {}) {
    const runner = stackRunner(options);
    const clusterScoped = path.join(repoRoot, 'manifests', 'kustomize', 'cluster-scoped-resources');
    const imageOverrides = mergeImageOverrides(options.imageOverrides);
    const platform = options.platform || getClusterPlatform({ runner });
    const rendered = renderRevisionManifestsForPlatform(repoRoot, imageOverrides, platform, {
      fixtureRequirements: options.fixtureRequirements,
      runner,
    });

    try {
      const renderedImages = rendered.images;
      if (options.expectedRelease) {
        validateReleaseManifestImages(renderedImages, options.expectedRelease);
      }
      if (options.requireLocalFirstParty) {
        validateLocalManifestImages(renderedImages, imageOverrides, rendered.manifestContents);
      }
      // Pull and archive every rendered runtime image before any workload is created. A missing
      // architecture therefore fails explicitly instead of surfacing as an opaque ImagePullBackOff.
      preloadManifestImages(rendered.renderedPath, platform, {
        removeSourceAfterLoad: options.removePreflightedSourcesAfterLoad,
        runner,
      });
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs('apply', '-k', clusterScoped),
          commandOptions({ timeout: 120000, stdio: 'inherit' }),
        ),
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
          commandOptions({ timeout: 70000 }),
        ),
        'KFP application CRD was not established',
      );
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs('apply', '-f', rendered.renderedPath),
          commandOptions({ timeout: 180000, stdio: 'inherit' }),
        ),
        'Failed to apply platform-agnostic KFP manifests',
      );

      const deployments = listDeployments({ runner });
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            namespace,
            'wait',
            '--for=condition=Available',
            '--timeout=10m',
            ...deployments.map((deployment) => `deployment/${deployment}`),
          ),
          commandOptions({ timeout: 610000 }),
        ),
        'Platform-agnostic KFP deployments did not all become available',
      );
      if (deployments.includes('mysql')) {
        // The official image exposes a temporary initialization server before its entrypoint execs
        // the final mysqld. Deployment availability and the API health endpoint can both pass in
        // that window, after which fixture writes fail while the temporary server shuts down.
        requireSuccess(
          runner(
            'kubectl',
            kubectlArgs(
              '-n',
              namespace,
              'exec',
              'deployment/mysql',
              '-c',
              'mysql',
              '--',
              'sh',
              '-c',
              MYSQL_FINAL_SERVER_WAIT_SCRIPT,
            ),
            commandOptions({ timeout: MYSQL_FINAL_SERVER_TIMEOUT_MS + 10000 }),
          ),
          'MySQL did not finish first-run initialization',
        );
      }
      return { deployments, renderedImages };
    } finally {
      cleanRenderedManifests(rendered);
    }
  }

  function deleteCluster(options = {}) {
    const runner = stackRunner(options);
    log(`Deleting Kind cluster ${clusterName}...`);
    const result = runner(
      'kind',
      ['delete', 'cluster', '--name', clusterName, '--kubeconfig', kubeconfigPath],
      commandOptions(),
    );
    if (result.success) {
      createdThisRun = false;
      ownsCluster = false;
      seedRuntimeLoaded = false;
      loadedImages.clear();
      verifiedEmulationPlatforms.clear();
    }
    return result;
  }

  function destroyOwnedCluster(options = {}) {
    if (!ownsCluster) {
      log(`Skipping deletion of unowned Kind cluster ${clusterName}.`, 'debug');
      return { skipped: true, success: true };
    }
    return deleteCluster(options);
  }

  function throwWithRollback(setupErrors, options = {}) {
    const runner = stackRunner(options);
    const clusterMayExist = ownsCluster && (createdThisRun || isClusterRunning({ runner }));
    let rollbackError = null;
    if (clusterMayExist) {
      log(`Rolling back failed Kind cluster setup for ${clusterName}...`, 'warn');
      try {
        requireSuccess(
          deleteCluster({ runner }),
          `Failed to roll back managed Kind cluster ${clusterName}`,
        );
      } catch (error) {
        rollbackError = error;
      }
    } else {
      ownsCluster = false;
    }
    if (rollbackError) {
      throw new AggregateError(
        [...setupErrors, rollbackError],
        `Kind cluster ${clusterName} setup and rollback both failed`,
      );
    }
    if (setupErrors.length > 1) {
      throw new AggregateError(setupErrors, `Kind cluster ${clusterName} setup failed`);
    }
    throw setupErrors[0];
  }

  async function createCluster(options = {}) {
    const runner = stackRunner(options);
    if (!isKindInstalled({ runner })) throw new Error('kind is not installed');
    if (!isKubectlInstalled({ runner })) throw new Error('kubectl is not installed');
    if (!isDockerRunning({ runner })) throw new Error('Docker is not running');
    if (isClusterRunning({ runner })) {
      throw new Error(
        `Managed Kind cluster ${clusterName} already exists. Refusing to reuse potentially stale ` +
          'backend images or data; destroy that exact stack before comparing again.',
      );
    }

    fs.mkdirSync(path.dirname(kubeconfigPath), { recursive: true });
    log(`Creating Kind cluster ${clusterName}...`);
    ownsCluster = true;
    const result = runner(
      'kind',
      ['create', 'cluster', '--name', clusterName, '--kubeconfig', kubeconfigPath],
      commandOptions({ timeout: 600000 }),
    );
    if (!result.success) {
      throwWithRollback(
        [
          new Error(
            `Failed to create Kind cluster ${clusterName}: ${result.error || result.output}`,
          ),
        ],
        { runner },
      );
    }
    createdThisRun = true;
    return { clusterName, context, created: true, kubeconfigPath };
  }

  async function deployRevision(repoRoot, options = {}) {
    const runner = stackRunner(options);
    const platform = options.platform || getClusterPlatform({ runner });
    preloadSeedRuntimeImage({
      force: options.forceSeedRuntime,
      platform,
      removeSourceAfterLoad: options.removePreflightedSourcesAfterLoad,
      runner,
    });
    const builtOverrides = options.components
      ? await buildComponentImages(options.components, repoRoot, {
          buildMetadata: options.buildMetadata,
          platform,
          runner,
          tagSuffix: options.tagSuffix,
        })
      : null;
    const imageOverrides = mergeImageOverrides(options.imageOverrides, builtOverrides);
    const result = applyKfpManifests(repoRoot, {
      expectedRelease: options.expectedRelease,
      fixtureRequirements: options.fixtureRequirements,
      imageOverrides,
      platform,
      requireLocalFirstParty: options.requireLocalFirstParty,
      removePreflightedSourcesAfterLoad: options.removePreflightedSourcesAfterLoad,
      runner,
    });
    return { ...result, images: imageOverrides.images };
  }

  async function ensureCluster(repoRoot, options = {}) {
    const runner = stackRunner(options);
    try {
      const creation = await createCluster({ runner });
      await deployRevision(repoRoot, { ...options, runner });
      return creation;
    } catch (error) {
      if (!createdThisRun) throw error;
      throwWithRollback([error], { runner });
    }
  }

  function teardownCluster(options = {}) {
    return deleteCluster(options).success;
  }

  function applyLiveImageOverrides(imageOverrides, options = {}) {
    const runner = stackRunner(options);
    const deployments = new Set();
    for (const override of imageOverrides.deployments || []) {
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            namespace,
            'set',
            'image',
            `deployment/${override.deployment}`,
            `${override.container}=${override.image}`,
          ),
          commandOptions(),
        ),
        `Failed to set image on deployment/${override.deployment}`,
      );
      const pullPolicyPatch = JSON.stringify({
        spec: {
          template: {
            spec: {
              containers: [{ name: override.container, imagePullPolicy: 'IfNotPresent' }],
            },
          },
        },
      });
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            namespace,
            'patch',
            `deployment/${override.deployment}`,
            '--type=strategic',
            '-p',
            pullPolicyPatch,
          ),
          commandOptions(),
        ),
        `Failed to set IfNotPresent on deployment/${override.deployment}`,
      );
      deployments.add(override.deployment);
    }

    const runtimeEntries = Object.entries(imageOverrides.runtimeEnvironment || {});
    if (runtimeEntries.length > 0) {
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            namespace,
            'set',
            'env',
            'deployment/ml-pipeline',
            ...runtimeEntries.map(([name, image]) => `${name}=${image}`),
          ),
          commandOptions(),
        ),
        'Failed to configure local KFP runtime images',
      );
      deployments.add('ml-pipeline');
    }
    for (const deployment of deployments) {
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs('-n', namespace, 'rollout', 'restart', `deployment/${deployment}`),
          commandOptions(),
        ),
        `Failed to restart deployment/${deployment}`,
      );
      requireSuccess(
        runner(
          'kubectl',
          kubectlArgs(
            '-n',
            namespace,
            'rollout',
            'status',
            `deployment/${deployment}`,
            '--timeout=180s',
          ),
          commandOptions({ timeout: 190000 }),
        ),
        `Deployment ${deployment} did not become ready`,
      );
    }
  }

  async function buildAndDeployComponents(components, repoRoot, options = {}) {
    if (components.length === 0) {
      log('No backend components to rebuild');
      return { images: {} };
    }
    const imageOverrides = await buildComponentImages(components, repoRoot, options);
    applyLiveImageOverrides(imageOverrides, options);
    return { images: imageOverrides.images };
  }

  function reapplyManifests(repoRoot, options = {}) {
    log(`Re-applying KFP manifests to ${clusterName}...`);
    return applyKfpManifests(repoRoot, options);
  }

  async function ensurePortForwards(forwards, options = {}) {
    const spawnFn =
      options.spawnFn ||
      ((command, args, spawnOptions) => spawnProcess(command, args, spawnOptions));
    const portInUse = options.portInUse || isPortInUse;
    const waitForTcpFn = options.waitForTcpFn || waitForTcp;
    const timeout = options.timeout || 15000;
    const started = [];
    try {
      for (const forward of forwards) {
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
            namespace,
            `svc/${forward.service}`,
            `${forward.localPort}:${forward.remotePort}`,
          ),
          commandOptions(),
        );
        if (!processes.includes(child)) processes.push(child);
        started.push(child);
        const ready = await waitForChildReadiness(child, () =>
          waitForTcpFn(forward.localPort, timeout, { child }),
        );
        if (!ready || child.exitCode !== null || child.signalCode !== null) {
          throw new Error(`Port forward for ${forward.service} did not become ready.`);
        }
        log(`${clusterName}: ${forward.service} -> localhost:${forward.localPort}`);
      }
      return started;
    } catch (error) {
      for (const child of started) child.kill('SIGTERM');
      throw error;
    }
  }

  async function ensureDeployedUiPortForwarding(options = {}) {
    return ensurePortForwards([deployedUiPortForward], options);
  }

  async function ensurePortForwarding(options = {}) {
    return ensurePortForwards(portForwards, options);
  }

  function frontendServerEnvironment(baseEnvironment = process.env) {
    const endpointRewrite = [
      `seaweedfs.${namespace}:9000=localhost:${ports.objectStore}`,
      `seaweedfs.${namespace}:80=localhost:${ports.objectStore}`,
      `seaweedfs.${namespace}.svc:9000=localhost:${ports.objectStore}`,
      `seaweedfs.${namespace}.svc:80=localhost:${ports.objectStore}`,
      `seaweedfs.${namespace}.svc.cluster.local:9000=localhost:${ports.objectStore}`,
      `seaweedfs.${namespace}.svc.cluster.local:80=localhost:${ports.objectStore}`,
    ].join(',');
    const environment = {
      ...commandEnvironment(baseEnvironment),
      FRONTEND_SERVER_NAMESPACE: namespace,
      MINIO_ENDPOINT_REWRITE: endpointRewrite,
      MINIO_HOST: 'localhost',
      MINIO_NAMESPACE: '',
      MINIO_PORT: String(ports.objectStore),
      ML_PIPELINE_SERVICE_HOST: LOOPBACK_HOST,
      ML_PIPELINE_SERVICE_PORT: String(ports.api),
      ML_PIPELINE_SERVICE_SCHEME: 'http',
    };
    if (ports.metadata === null) {
      delete environment.METADATA_ENVOY_SERVICE_SERVICE_HOST;
      delete environment.METADATA_ENVOY_SERVICE_SERVICE_PORT;
      delete environment.METADATA_ENVOY_SERVICE_SERVICE_SCHEME;
    } else {
      environment.METADATA_ENVOY_SERVICE_SERVICE_HOST = LOOPBACK_HOST;
      environment.METADATA_ENVOY_SERVICE_SERVICE_PORT = String(ports.metadata);
      environment.METADATA_ENVOY_SERVICE_SERVICE_SCHEME = 'http';
    }
    return environment;
  }

  function writeLoopbackListenPreload() {
    fs.mkdirSync(archiveDir, { recursive: true });
    const preloadPath = path.join(archiveDir, 'loopback-listen.cjs');
    fs.writeFileSync(preloadPath, LOOPBACK_LISTEN_PRELOAD, { mode: 0o600 });
    return preloadPath;
  }

  async function startFrontendServer(repoRoot, options = {}) {
    const runner = stackRunner(options);
    const spawnFn =
      options.spawnFn ||
      ((command, args, spawnOptions) => spawnProcess(command, args, spawnOptions));
    const waitForServiceFn = options.waitForServiceFn || waitForService;
    const serverDir = path.join(repoRoot, 'frontend', 'server');
    const serverEntry = path.join(serverDir, 'dist', 'server.js');
    if (options.skipBuild) {
      if (!fs.existsSync(serverEntry)) {
        throw new Error(`Cannot skip server build: ${serverEntry} does not exist.`);
      }
    } else {
      requireSuccess(
        runner(
          'npm',
          ['ci'],
          commandOptions({ cwd: serverDir, timeout: 120000, stdio: 'inherit' }),
        ),
        'Failed to install frontend server dependencies',
      );
      requireSuccess(
        runner('npm', ['run', 'build'], commandOptions({ cwd: serverDir, timeout: 120000 })),
        'Failed to build frontend server',
      );
    }

    const buildDir = path.join(repoRoot, 'frontend', 'build');
    const loopbackPreload = writeLoopbackListenPreload();
    const child = spawnFn(
      'node',
      ['--require', loopbackPreload, 'dist/server.js', buildDir, String(ports.frontendServer)],
      {
        cwd: serverDir,
        env: frontendServerEnvironment(options.env || process.env),
      },
    );
    if (!processes.includes(child)) processes.push(child);
    frontendServerProcess = child;
    const healthUrl = `http://127.0.0.1:${ports.frontendServer}/apis/v2beta1/healthz`;
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

  async function cleanup(options = {}) {
    const terminate = options.terminate || terminateChild;
    log(`Cleaning up processes for ${clusterName}...`);
    const tracked = [...processes];
    const results = await Promise.allSettled(tracked.map((child) => terminate(child)));
    const failures = results
      .map((result, index) => ({ child: tracked[index], result }))
      .filter(({ result }) => result.status === 'rejected');
    processes.splice(
      0,
      processes.length,
      ...failures
        .map(({ child }) => child)
        .filter((child) => child.exitCode === null && child.signalCode === null),
    );
    if (!processes.includes(frontendServerProcess)) frontendServerProcess = null;
    if (failures.length > 0) {
      throw new AggregateError(
        failures.map(({ result }) => result.reason),
        `Failed to stop ${failures.length} process(es) for ${clusterName}.`,
      );
    }
  }

  return Object.freeze({
    CLUSTER_NAME: clusterName,
    FRONTEND_SERVER_PORT: ports.frontendServer,
    KUBE_CONTEXT: context,
    NAMESPACE: namespace,
    PORT_FORWARDS: portForwards,
    applyKfpManifests,
    archiveDir,
    buildAndDeployComponents,
    buildComponentImages,
    cleanup,
    collectDiagnostics,
    clusterName,
    commandEnvironment,
    context,
    createCluster,
    deployRevision,
    destroyCluster: destroyOwnedCluster,
    destroyOwnedCluster,
    deployedUiPortForward,
    deployedUiUrl: `http://127.0.0.1:${ports.frontendServer}`,
    ensureDeployedUiPortForwarding,
    ensureCluster,
    ensurePortForwarding,
    extractManifestImages,
    frontendServerEnvironment,
    frontendServerUrl: `http://127.0.0.1:${ports.frontendServer}`,
    getClusterPlatform,
    getDockerPlatform,
    getClusterStatus,
    imageScope,
    isClusterRunning,
    isDockerRunning,
    isKfpHealthy,
    isKindInstalled,
    isKubectlInstalled,
    kubeconfigPath,
    loadImageOverrides,
    kubectlArgs,
    namespace,
    portForwards,
    ports,
    preflightReleaseImages,
    preflightSeedRuntimeImage,
    preflightThirdPartyImages,
    preloadManifestImages,
    preloadSeedRuntimeImage,
    reapplyManifests,
    reuseComponentImages,
    revision,
    role,
    saveAndLoadImage,
    scopedImageTag,
    spawnProcess,
    startFrontendServer,
    stopFrontendServer,
    teardownCluster,
    validateReleaseManifestImages,
    validateLocalManifestImages,
  });
}

const defaultStack = createKindStack({
  clusterName: CLUSTER_NAME,
  context: KUBE_CONTEXT,
  kubeconfigPath: DEFAULT_KUBECONFIG,
  namespace: NAMESPACE,
  ports: DEFAULT_PORTS,
  role: 'compatibility',
});

function isKindInstalled(runner = run) {
  return defaultStack.isKindInstalled({ runner });
}

function isKubectlInstalled(runner = run) {
  return defaultStack.isKubectlInstalled({ runner });
}

function isDockerRunning(runner = run) {
  return defaultStack.isDockerRunning({ runner });
}

function isClusterRunning(runner = run) {
  return defaultStack.isClusterRunning({ runner });
}

function isKfpHealthy(options = {}) {
  return defaultStack.isKfpHealthy(options);
}

function getClusterStatus(options = {}) {
  return defaultStack.getClusterStatus(options);
}

function preloadSeedRuntimeImage(runner = run, options = {}) {
  return defaultStack.preloadSeedRuntimeImage({ ...options, runner });
}

function applyKfpManifests(repoRoot, runner = run) {
  return defaultStack.applyKfpManifests(repoRoot, { runner });
}

async function ensureCluster(repoRoot, options = {}) {
  const result = await defaultStack.ensureCluster(repoRoot, options);
  return { created: result.created, context: result.context };
}

function teardownCluster(options = {}) {
  return defaultStack.teardownCluster(options);
}

function getClusterPlatform(runner = run) {
  return defaultStack.getClusterPlatform({ runner });
}

function buildAndDeployComponents(components, repoRoot, options = {}) {
  return defaultStack.buildAndDeployComponents(components, repoRoot, options);
}

function reapplyManifests(repoRoot, options = {}) {
  return defaultStack.reapplyManifests(repoRoot, options);
}

function ensurePortForwarding(options = {}) {
  return defaultStack.ensurePortForwarding(options);
}

function frontendServerEnvironment(baseEnvironment = process.env) {
  return defaultStack.frontendServerEnvironment(baseEnvironment);
}

function startFrontendServer(repoRoot, options = {}) {
  return defaultStack.startFrontendServer(repoRoot, options);
}

function stopFrontendServer() {
  return defaultStack.stopFrontendServer();
}

function cleanup(options = {}) {
  return defaultStack.cleanup(options);
}

function spawnProcess(command, args, options = {}) {
  return defaultStack.spawnProcess(command, args, options);
}

module.exports = {
  AMD64_EMULATION_CANARY_IMAGE,
  AMD64_EMULATION_CANARY_LOCAL_IMAGE,
  CLUSTER_NAME,
  DEFAULT_KUBECONFIG,
  DEFAULT_PORTS,
  DIAGNOSTIC_LIMITS,
  FRONTEND_SERVER_PORT,
  KUBE_CONTEXT,
  MIXED_PLATFORM_WORKLOADS,
  NAMESPACE,
  PLATFORM_DEPLOYMENTS,
  PORT_FORWARDS,
  SEED_RUNTIME_IMAGE,
  applyKfpManifests,
  buildAndDeployComponents,
  checkPortAvailability,
  cleanup,
  createKindStack,
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
