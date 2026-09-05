const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');
const { parseAllDocuments, parseDocument } = require('yaml');

const cluster = require('../cluster-manager');
const { COMPONENTS } = require('../detect-changes');

const BUILD_METADATA = Object.freeze({
  buildDate: '2026-09-01T12:00:00Z',
  commitSha: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
  nodeVersion: '24.14.0',
  tagName: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
});
const TEST_IMAGE_ID = `sha256:${'a'.repeat(64)}`;

class FakeChild extends EventEmitter {
  constructor(id = '') {
    super();
    this.exitCode = null;
    this.id = id;
    this.signalCode = null;
    this.killedWith = null;
  }

  kill(signal) {
    this.killedWith = signal;
    return true;
  }
}

function success(output = '') {
  return { success: true, output };
}

function createTestStack(t, overrides = {}) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-kind-stack-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  return cluster.createKindStack({
    archiveDir: path.join(root, 'archives'),
    clusterName: 'ui-smoke-base-test',
    context: 'kind-ui-smoke-base-test',
    kubeconfigPath: path.join(root, 'kubeconfig'),
    ports: {
      api: 3102,
      frontendServer: 3101,
      metadata: 9190,
      objectStore: 9100,
    },
    revision: 'aaaaaaaaaaaa',
    role: 'base',
    ...overrides,
  });
}

function createRevisionRoot(stack, name = 'revision') {
  const repoRoot = path.join(path.dirname(stack.archiveDir), name);
  fs.mkdirSync(path.join(repoRoot, 'manifests', 'kustomize', 'env', 'platform-agnostic'), {
    recursive: true,
  });
  return repoRoot;
}

function renderedManifest(images = ['docker.io/library/alpine:3.23']) {
  return [
    'apiVersion: apps/v1',
    'kind: Deployment',
    'metadata:',
    '  name: ml-pipeline',
    'spec:',
    '  template:',
    '    spec:',
    '      containers:',
    '        - name: ml-pipeline-api-server',
    `          image: ${images[0]}`,
    ...(images[1] ? ['        - name: helper', `          image: ${images[1]}`] : []),
    '---',
    'apiVersion: v1',
    'kind: ConfigMap',
    'metadata:',
    '  name: runtime-images',
    'data:',
    '  env: |',
    '    - name: V2_DRIVER_IMAGE',
    '      value: ghcr.io/kubeflow/kfp-driver:2.17.1',
  ].join('\n');
}

function renderedManifestWithWorkflowDefaults() {
  return [
    renderedManifest(),
    '---',
    'apiVersion: v1',
    'kind: ConfigMap',
    'metadata:',
    '  name: workflow-controller-configmap',
    'data:',
    '  workflowDefaults: |',
    '    spec:',
    '      ttlStrategy:',
    '        secondsAfterCompletion: 3600',
    '      templateDefaults:',
    '        retryStrategy:',
    "          limit: '2'",
    '          retryPolicy: OnError',
  ].join('\n');
}

function mixedPlatformManifest(options = {}) {
  const pullPolicy = options.pullPolicy ? `\n          imagePullPolicy: ${options.pullPolicy}` : '';
  const metadataWriterDeployment = options.metadataWriterDeployment || 'metadata-writer';
  const metadataWriterImage =
    options.metadataWriterImage || 'ghcr.io/kubeflow/kfp-metadata-writer:2.17.1';
  return [
    'apiVersion: apps/v1',
    'kind: Deployment',
    'metadata:',
    `  name: ${metadataWriterDeployment}`,
    'spec:',
    '  template:',
    '    spec:',
    '      containers:',
    '        - name: main',
    `          image: ${metadataWriterImage}${pullPolicy}`,
    '---',
    'apiVersion: apps/v1',
    'kind: Deployment',
    'metadata:',
    '  name: metadata-grpc-deployment',
    'spec:',
    '  template:',
    '    spec:',
    '      containers:',
    '        - name: container',
    `          image: gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0${pullPolicy}`,
    '---',
    'apiVersion: apps/v1',
    'kind: Deployment',
    'metadata:',
    '  name: mysql',
    'spec:',
    '  template:',
    '    spec:',
    '      containers:',
    '        - name: mysql',
    '          image: mysql:8.4',
  ].join('\n');
}

function mixedPlatformManifestWithWorkflowDefaults(options = {}) {
  return [
    mixedPlatformManifest(options),
    '---',
    'apiVersion: v1',
    'kind: ConfigMap',
    'metadata:',
    '  name: workflow-controller-configmap',
    'data:',
    '  workflowDefaults: |',
    '    spec:',
    '      ttlStrategy:',
    '        secondsAfterCompletion: 3600',
    '      templateDefaults:',
    '        retryStrategy:',
    "          limit: '2'",
    '          retryPolicy: OnError',
  ].join('\n');
}

function deploymentRunner(calls, options = {}) {
  const {
    architecture = 'amd64',
    deployments = 'ml-pipeline\nmysql',
    manifest = renderedManifest(),
  } = options;
  return (command, args, commandOptions) => {
    calls.push({ args, command, options: commandOptions });
    if (command === 'kind' && args[0] === 'get') return success('');
    if (command === 'kubectl' && args.includes('nodes')) return success(architecture);
    if (command === 'docker' && args[0] === 'info' && args.includes('--format')) {
      return success(`linux/${architecture}`);
    }
    if (command === 'docker' && args[0] === 'image' && args[1] === 'inspect') {
      return success(TEST_IMAGE_ID);
    }
    if (command === 'kubectl' && args[0] === 'kustomize') {
      const outputPath = args[args.indexOf('--output') + 1];
      fs.mkdirSync(path.dirname(outputPath), { recursive: true });
      fs.writeFileSync(outputPath, manifest);
    }
    if (
      command === 'kubectl' &&
      args.includes('get') &&
      args.includes('deployments') &&
      args.some((argument) => argument.startsWith('jsonpath='))
    ) {
      return success(deployments);
    }
    return success();
  };
}

test('default wrappers retain the SeaweedFS development environment', () => {
  assert.equal(cluster.PORT_FORWARDS[2].service, 'seaweedfs');
  const environment = cluster.frontendServerEnvironment({ KEEP: 'yes' });
  assert.equal(environment.KEEP, 'yes');
  assert.equal(environment.KUBECONFIG, cluster.DEFAULT_KUBECONFIG);
  assert.equal(environment.MINIO_HOST, 'localhost');
  assert.equal(environment.MINIO_NAMESPACE, '');
  assert.equal(environment.MINIO_PORT, '9000');
  assert.equal(environment.ML_PIPELINE_SERVICE_HOST, '127.0.0.1');
  assert.equal(environment.ML_PIPELINE_SERVICE_SCHEME, 'http');
  assert.equal(environment.METADATA_ENVOY_SERVICE_SERVICE_HOST, '127.0.0.1');
  assert.equal(environment.METADATA_ENVOY_SERVICE_SERVICE_SCHEME, 'http');
  assert.equal(environment.FRONTEND_SERVER_NAMESPACE, 'kubeflow');
  assert.match(environment.MINIO_ENDPOINT_REWRITE, /seaweedfs\.kubeflow\.svc\.cluster\.local:80/);
});

test('stack ports configure every server dependency and permit a revision without MLMD', (t) => {
  const stack = createTestStack(t, {
    clusterName: 'ui-smoke-head-test',
    context: 'kind-ui-smoke-head-test',
    ports: {
      api: 3202,
      frontendServer: 3201,
      metadata: null,
      objectStore: 9200,
    },
    role: 'head',
  });
  const environment = stack.frontendServerEnvironment({
    METADATA_ENVOY_SERVICE_SERVICE_HOST: 'ambient-host',
    METADATA_ENVOY_SERVICE_SERVICE_PORT: '1234',
    METADATA_ENVOY_SERVICE_SERVICE_SCHEME: 'https',
    ML_PIPELINE_SERVICE_HOST: 'ambient-api',
    ML_PIPELINE_SERVICE_SCHEME: 'https',
  });

  assert.deepEqual(
    stack.portForwards.map(({ service, localPort }) => [service, localPort]),
    [
      ['ml-pipeline', 3202],
      ['seaweedfs', 9200],
    ],
  );
  assert.equal(environment.ML_PIPELINE_SERVICE_PORT, '3202');
  assert.equal(environment.ML_PIPELINE_SERVICE_HOST, '127.0.0.1');
  assert.equal(environment.ML_PIPELINE_SERVICE_SCHEME, 'http');
  assert.equal(environment.MINIO_PORT, '9200');
  assert.equal(environment.METADATA_ENVOY_SERVICE_SERVICE_HOST, undefined);
  assert.equal(environment.METADATA_ENVOY_SERVICE_SERVICE_PORT, undefined);
  assert.equal(environment.METADATA_ENVOY_SERVICE_SERVICE_SCHEME, undefined);
  assert.match(environment.MINIO_ENDPOINT_REWRITE, /=localhost:9200/);
  assert.equal(stack.frontendServerUrl, 'http://127.0.0.1:3201');
  assert.equal(stack.role, 'head');
  assert.equal(stack.revision, 'aaaaaaaaaaaa');
});

test('Docker architecture discovery normalizes the Kind target platform', (t) => {
  const arm = createTestStack(t, {
    runner(command, args) {
      if (command === 'docker' && args[0] === 'info') return success('linux/aarch64');
      return success();
    },
  });
  const amd = createTestStack(t, {
    clusterName: 'ui-smoke-amd-platform',
    runner(command, args) {
      if (command === 'docker' && args[0] === 'info') return success('linux/x86_64');
      return success();
    },
  });

  assert.equal(arm.getDockerPlatform(), 'linux/arm64');
  assert.equal(amd.getDockerPlatform(), 'linux/amd64');
});

test('rejects ambiguous stack identities and overlapping ports', () => {
  assert.throws(() => cluster.createKindStack({ clusterName: '../unsafe' }), /DNS-compatible/);
  assert.throws(
    () =>
      cluster.createKindStack({
        ports: { api: 3001, frontendServer: 3001, metadata: 9090, objectStore: 9000 },
      }),
    /distinct/,
  );
  assert.throws(
    () => cluster.createKindStack({ kubeconfigPath: 'relative/kubeconfig' }),
    /absolute path/,
  );
});

test('HTTP readiness accepts only successful responses', async () => {
  let status = 500;
  const get = (_url, callback) => {
    const request = new EventEmitter();
    request.setTimeout = () => {};
    request.destroy = () => {};
    queueMicrotask(() => callback({ statusCode: status, resume() {} }));
    return request;
  };
  const url = 'http://frontend.test/healthz';

  assert.equal(await cluster.isKfpHealthy({ url, get }), false);
  assert.equal(await cluster.waitForService(url, 20, { interval: 1, get }), false);
  status = 404;
  assert.equal(await cluster.waitForService(url, 20, { interval: 1, get }), false);
  status = 200;
  assert.equal(await cluster.isKfpHealthy({ url, get }), true);
  assert.equal(await cluster.waitForService(url, 100, { interval: 1, get }), true);
});

test('two stack instances route creation and kubectl through independent kubeconfigs', async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-two-stacks-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  const calls = [];
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (command === 'kind' && args[0] === 'get') return success('');
    return success();
  };
  const base = cluster.createKindStack({
    clusterName: 'ui-smoke-run-base',
    kubeconfigPath: path.join(root, 'base', 'kubeconfig'),
    ports: { api: 3302, frontendServer: 3301, metadata: 9390, objectStore: 9300 },
    role: 'base',
    runner,
  });
  const head = cluster.createKindStack({
    clusterName: 'ui-smoke-run-head',
    kubeconfigPath: path.join(root, 'head', 'kubeconfig'),
    ports: { api: 3402, frontendServer: 3401, metadata: null, objectStore: 9400 },
    role: 'head',
    runner,
  });

  const baseResult = await base.createCluster();
  const headResult = await head.createCluster();

  assert.equal(baseResult.context, 'kind-ui-smoke-run-base');
  assert.equal(headResult.context, 'kind-ui-smoke-run-head');
  const creates = calls.filter((call) => call.command === 'kind' && call.args[0] === 'create');
  assert.deepEqual(
    creates.map((call) => call.args),
    [
      [
        'create',
        'cluster',
        '--name',
        'ui-smoke-run-base',
        '--kubeconfig',
        path.join(root, 'base', 'kubeconfig'),
      ],
      [
        'create',
        'cluster',
        '--name',
        'ui-smoke-run-head',
        '--kubeconfig',
        path.join(root, 'head', 'kubeconfig'),
      ],
    ],
  );
  assert.equal(
    calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args[0] === 'config' &&
        ['current-context', 'use-context', 'unset'].some((value) => call.args.includes(value)),
    ),
    false,
  );
  assert.equal(creates[0].options.env.KUBECONFIG, path.join(root, 'base', 'kubeconfig'));
  assert.equal(creates[1].options.env.KUBECONFIG, path.join(root, 'head', 'kubeconfig'));
});

test('deployRevision loads single-platform archives before applying revision manifests', async (t) => {
  const calls = [];
  const stack = createTestStack(t, { runner: deploymentRunner(calls) });

  const result = await stack.ensureCluster('/revision');

  assert.equal(result.clusterName, stack.clusterName);
  const seedPull = calls.find(
    (call) => call.command === 'docker' && call.args.includes(cluster.SEED_RUNTIME_IMAGE),
  );
  assert.deepEqual(seedPull.args.slice(0, 3), ['pull', '--platform', 'linux/amd64']);
  const saves = calls.filter((call) => call.command === 'docker' && call.args[0] === 'save');
  assert.ok(saves.length >= 3);
  assert.ok(saves.every((call) => call.args[1] === '--platform' && call.args[2] === 'linux/amd64'));
  const loads = calls.filter((call) => call.command === 'kind' && call.args[0] === 'load');
  assert.ok(loads.every((call) => call.args[1] === 'image-archive'));
  assert.ok(loads.every((call) => call.args.includes(stack.clusterName)));
  assert.equal(
    loads.some((call) => call.args.includes('docker-image')),
    false,
  );

  const firstApplyIndex = calls.findIndex(
    (call) => call.command === 'kubectl' && call.args.includes('apply'),
  );
  assert.ok(firstApplyIndex > calls.map((call) => call.command).lastIndexOf('kind'));
  const applyCalls = calls.filter(
    (call) => call.command === 'kubectl' && call.args.includes('apply'),
  );
  assert.equal(applyCalls.length, 2);
  assert.ok(applyCalls[0].args.at(-1).endsWith('manifests/kustomize/cluster-scoped-resources'));
  assert.ok(applyCalls[1].args.includes('-f'));

  const kubectlClusterCalls = calls.filter(
    (call) =>
      call.command === 'kubectl' && call.args[0] !== 'kustomize' && call.args[0] !== 'version',
  );
  assert.ok(
    kubectlClusterCalls.every(
      (call) =>
        call.args.includes('--kubeconfig') &&
        call.args.includes(stack.kubeconfigPath) &&
        call.args.includes('--context') &&
        call.args.includes(stack.context) &&
        call.options.env.KUBECONFIG === stack.kubeconfigPath,
    ),
  );
  const workloadWait = calls.find(
    (call) => call.command === 'kubectl' && call.args.includes('--timeout=10m'),
  );
  assert.deepEqual(
    workloadWait.args.filter((argument) => argument.startsWith('deployment/')),
    ['deployment/ml-pipeline', 'deployment/mysql'],
  );
  const mysqlFinalServerWait = calls.find(
    (call) =>
      call.command === 'kubectl' &&
      call.args.includes('exec') &&
      call.args.includes('deployment/mysql'),
  );
  assert.ok(mysqlFinalServerWait);
  assert.ok(mysqlFinalServerWait.args.includes('mysql'));
  assert.match(mysqlFinalServerWait.args.at(-1), /\/proc\/1\/comm/);
  assert.match(mysqlFinalServerWait.args.at(-1), /= mysqld/);
  assert.ok(calls.indexOf(mysqlFinalServerWait) > calls.indexOf(workloadWait));
});

for (const writable of [true, false]) {
  test(`deployment requires writable SeaweedFS before seeding: ${writable}`, async (t) => {
    const calls = [];
    const delegate = deploymentRunner(calls, { deployments: 'ml-pipeline\nseaweedfs' });
    const runner = (command, args, options) => {
      const result = delegate(command, args, options);
      if (args.includes('exec') && args.includes('deployment/seaweedfs')) {
        return writable ? result : { success: false, error: 'No writable volumes', output: '' };
      }
      return result;
    };
    const stack = createTestStack(t, { runner });
    if (writable) await stack.ensureCluster('/revision');
    else await assert.rejects(stack.ensureCluster('/revision'), /Artifact storage is not writable/);
    const waitIndex = calls.findIndex((call) => call.args.includes('--timeout=10m'));
    const probeIndex = calls.findIndex(
      (call) => call.args.includes('exec') && call.args.includes('deployment/seaweedfs'),
    );
    assert.ok(probeIndex > waitIndex);
    assert.match(calls[probeIndex].args.at(-1), /--aws-sigv4/);
    assert.match(calls[probeIndex].args.at(-1), /-X PUT/);
    assert.match(calls[probeIndex].args.at(-1), /-X DELETE/);
  });
}

test('full-stack deployment releases only preflight images pulled by the stack', async (t) => {
  const calls = [];
  const initiallyPresent = new Set(['mysql:8.4']);
  const present = new Set(initiallyPresent);
  const delegate = deploymentRunner(calls, {
    manifest: renderedManifest(['docker.io/library/alpine:3.23', 'mysql:8.4']),
  });
  const runner = (command, args, options) => {
    if (command === 'docker' && args[0] === 'image' && args[1] === 'inspect') {
      calls.push({ args, command, options });
      return present.has(args.at(-1))
        ? success(TEST_IMAGE_ID)
        : { error: 'No such image', output: '', success: false };
    }
    if (command === 'docker' && args[0] === 'pull') {
      present.add(args.at(-1));
    }
    if (command === 'docker' && args[0] === 'image' && args[1] === 'rm') {
      present.delete(args.at(-1));
    }
    return delegate(command, args, options);
  };
  const stack = createTestStack(t, { runner });
  const revisionRoot = createRevisionRoot(stack);

  stack.preflightSeedRuntimeImage({ platform: 'linux/amd64' });
  stack.preflightThirdPartyImages(revisionRoot, { platform: 'linux/amd64' });
  await stack.deployRevision(revisionRoot, {
    platform: 'linux/amd64',
    removePreflightedSourcesAfterLoad: true,
  });

  const removals = calls
    .filter(({ args, command }) => command === 'docker' && args[0] === 'image' && args[1] === 'rm')
    .map(({ args }) => args.at(-1));
  assert.deepEqual(
    removals.sort(),
    [cluster.SEED_RUNTIME_IMAGE, 'docker.io/library/alpine:3.23'].sort(),
  );
  assert.equal(removals.includes('mysql:8.4'), false);
  assert.equal(present.has('mysql:8.4'), true);

  for (const image of removals) {
    const removalIndex = calls.findIndex(
      ({ args, command }) =>
        command === 'docker' && args[0] === 'image' && args[1] === 'rm' && args.at(-1) === image,
    );
    const saveIndex = calls.findLastIndex(
      ({ args, command }, index) =>
        index < removalIndex && command === 'docker' && args[0] === 'save' && args.at(-1) === image,
    );
    const archive = calls[saveIndex].args[calls[saveIndex].args.indexOf('--output') + 1];
    const loadIndex = calls.findIndex(
      ({ args, command }) => command === 'kind' && args[0] === 'load' && args.includes(archive),
    );
    assert.ok(saveIndex >= 0 && saveIndex < removalIndex && removalIndex < loadIndex);
  }
});

test('deployRevision applies fixture runtime requirements only to its rendered manifest', async (t) => {
  const calls = [];
  let appliedManifest = null;
  const delegate = deploymentRunner(calls, { manifest: renderedManifestWithWorkflowDefaults() });
  const runner = (command, args, options) => {
    if (command === 'kubectl' && args.includes('apply') && args.includes('-f')) {
      appliedManifest = fs.readFileSync(args.at(-1), 'utf8');
    }
    return delegate(command, args, options);
  };
  const stack = createTestStack(t, { runner });

  await stack.deployRevision('/revision', {
    fixtureRequirements: { argoRetryPolicy: 'OnFailure' },
    platform: 'linux/amd64',
  });

  assert.match(appliedManifest, /retryPolicy: OnFailure/);
  assert.match(appliedManifest, /secondsAfterCompletion: 3600/);
  assert.match(appliedManifest, /limit: '2'/);
  assert.doesNotMatch(renderedManifestWithWorkflowDefaults(), /retryPolicy: OnFailure/);
});

test('fixture runtime requirements fail before any Kubernetes resource is applied', async (t) => {
  const calls = [];
  const stack = createTestStack(t, { runner: deploymentRunner(calls) });

  await assert.rejects(
    stack.deployRevision('/revision', {
      fixtureRequirements: { argoRetryPolicy: 'OnFailure' },
      platform: 'linux/amd64',
    }),
    /exactly one workflow-controller-configmap; found 0/,
  );
  assert.equal(
    calls.some(({ args, command }) => command === 'kubectl' && args.includes('apply')),
    false,
  );
});

test('seed runtime architecture is preflighted without creating or loading a cluster', (t) => {
  const calls = [];
  const stack = createTestStack(t, { runner: deploymentRunner(calls) });

  assert.deepEqual(stack.preflightSeedRuntimeImage({ platform: 'linux/amd64' }), {
    image: cluster.SEED_RUNTIME_IMAGE,
    platform: 'linux/amd64',
  });
  const dockerCalls = calls.filter(({ command }) => command === 'docker');
  assert.deepEqual(
    dockerCalls.map(({ args }) => args.slice(0, 3)),
    [
      ['image', 'inspect', '--format'],
      ['pull', '--platform', 'linux/amd64'],
      ['save', '--platform', 'linux/amd64'],
      ['image', 'inspect', '--format'],
    ],
  );
  assert.equal(
    calls.some(({ command }) => command === 'kind'),
    false,
  );
});

test('deployment reuses only the unchanged image proven by preflight', (t) => {
  const calls = [];
  const image = 'docker.io/library/alpine:3.23';
  const stack = createTestStack(t, {
    runner: deploymentRunner(calls, { manifest: renderedManifest([image]) }),
  });
  const manifestPath = path.join(stack.archiveDir, 'preflighted.yaml');
  fs.mkdirSync(stack.archiveDir, { recursive: true });
  fs.writeFileSync(manifestPath, renderedManifest([image]));

  stack.preflightReleaseImages('/revision', { platform: 'linux/amd64' });
  const pullsAfterPreflight = calls.filter(
    ({ args, command }) => command === 'docker' && args[0] === 'pull',
  ).length;
  stack.preloadManifestImages(manifestPath, 'linux/amd64');

  assert.equal(
    calls.filter(({ args, command }) => command === 'docker' && args[0] === 'pull').length,
    pullsAfterPreflight,
  );
  assert.ok(
    calls.some(
      ({ args, command }) => command === 'docker' && args[0] === 'save' && args.at(-1) === image,
    ),
  );
  assert.ok(calls.some(({ command }) => command === 'kind'));
});

test('deployment rejects a mutable image tag changed after preflight', (t) => {
  const calls = [];
  const image = 'docker.io/library/alpine:3.23';
  let imageId = TEST_IMAGE_ID;
  const delegate = deploymentRunner(calls, { manifest: renderedManifest([image]) });
  const runner = (command, args, options) => {
    if (command === 'docker' && args[0] === 'image' && args[1] === 'inspect') {
      calls.push({ args, command, options });
      return success(imageId);
    }
    return delegate(command, args, options);
  };
  const stack = createTestStack(t, { runner });
  const manifestPath = path.join(stack.archiveDir, 'changed-after-preflight.yaml');
  fs.mkdirSync(stack.archiveDir, { recursive: true });
  fs.writeFileSync(manifestPath, renderedManifest([image]));

  stack.preflightReleaseImages('/revision', { platform: 'linux/amd64' });
  imageId = `sha256:${'b'.repeat(64)}`;

  assert.throws(
    () => stack.preloadManifestImages(manifestPath, 'linux/amd64'),
    /changed or disappeared after it was preflighted.*not covered by the successful preflight/,
  );
});

test('preflight fails clearly when a rendered release image lacks the node architecture', (t) => {
  const calls = [];
  const runner = deploymentRunner(calls, {
    architecture: 'arm64',
    manifest: renderedManifest(['ghcr.io/example/amd64-only:1']),
  });
  const stack = createTestStack(t, { runner });
  const originalRunner = stack.preloadManifestImages;
  void originalRunner;
  const failingRunner = (command, args, options) => {
    const result = runner(command, args, options);
    if (
      command === 'docker' &&
      args[0] === 'pull' &&
      args.includes('ghcr.io/example/amd64-only:1')
    ) {
      return { success: false, error: 'no matching manifest for linux/arm64', output: '' };
    }
    return result;
  };
  const checked = createTestStack(t, {
    archiveDir: stack.archiveDir,
    clusterName: 'ui-smoke-arm-check',
    kubeconfigPath: path.join(path.dirname(stack.kubeconfigPath), 'arm-kubeconfig'),
    runner: failingRunner,
  });

  assert.throws(
    () => checked.preflightReleaseImages('/revision', { platform: 'linux/arm64' }),
    /cannot be pulled.*linux\/arm64.*never fall back to a different architecture/,
  );
});

test('preflight also attributes a platform mismatch discovered while exporting', (t) => {
  const calls = [];
  const image = 'gcr.io/example/amd64-only:1';
  const runner = deploymentRunner(calls, {
    architecture: 'arm64',
    manifest: renderedManifest([image]),
  });
  const failingRunner = (command, args, options) => {
    const result = runner(command, args, options);
    if (command === 'docker' && args[0] === 'save' && args.at(-1) === image) {
      return {
        success: false,
        error: 'image was found but does not provide the specified platform (linux/arm64)',
        output: '',
      };
    }
    return result;
  };
  const stack = createTestStack(t, {
    clusterName: 'ui-smoke-arm-export-check',
    runner: failingRunner,
  });

  assert.throws(
    () => stack.preflightReleaseImages('/revision', { platform: 'linux/arm64' }),
    /cannot be exported.*linux\/arm64.*never fall back to a different architecture/,
  );
});

test('mixed-platform deploy export failures retain the native Kind node platform', (t) => {
  const calls = [];
  const image = 'ghcr.io/kubeflow/kfp-metadata-writer:2.17.1';
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (command === 'docker' && args[0] === 'save' && args.at(-1) === image) {
      return { success: false, error: 'export failed', output: '' };
    }
    return success();
  };
  const stack = createTestStack(t, { runner });
  const manifestPath = path.join(stack.archiveDir, 'mixed-platform.yaml');
  fs.mkdirSync(stack.archiveDir, { recursive: true });
  fs.writeFileSync(manifestPath, mixedPlatformManifest({ pullPolicy: 'IfNotPresent' }));

  assert.throws(
    () => stack.preloadManifestImages(manifestPath, 'linux/arm64'),
    /cannot be exported for workload platform linux\/amd64 \(Kind node platform linux\/arm64\)/,
  );
});

test('arm64 preflight uses exact amd64 workload exceptions and leaves other images native', (t) => {
  const calls = [];
  const overlays = [];
  let overlayResourceTarget = null;
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (command === 'kubectl' && args[0] === 'kustomize') {
      const outputPath = args[args.indexOf('--output') + 1];
      let manifest = mixedPlatformManifest();
      const kustomizationPath = path.join(args[1], 'kustomization.yaml');
      if (fs.existsSync(kustomizationPath)) {
        assert.equal(fs.existsSync(path.join(args[1], 'kustomization.json')), false);
        const overlay = JSON.parse(fs.readFileSync(kustomizationPath, 'utf8'));
        overlays.push(overlay);
        overlayResourceTarget = path.resolve(fs.realpathSync(args[1]), overlay.resources[0]);
        manifest = mixedPlatformManifest({ pullPolicy: 'IfNotPresent' });
      }
      fs.mkdirSync(path.dirname(outputPath), { recursive: true });
      fs.writeFileSync(outputPath, manifest);
    }
    return success();
  };
  const stack = createTestStack(t, { runner });
  const revisionRoot = createRevisionRoot(stack);

  const result = stack.preflightReleaseImages(revisionRoot, {
    expectedRelease: '2.17.1',
    platform: 'linux/arm64',
  });

  assert.deepEqual(result.imagePlatforms, {
    'gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0': 'linux/amd64',
    'ghcr.io/kubeflow/kfp-metadata-writer:2.17.1': 'linux/amd64',
    'mysql:8.4': 'linux/arm64',
  });
  const pullPlatform = (image) =>
    calls.find(
      ({ args, command }) => command === 'docker' && args[0] === 'pull' && args.at(-1) === image,
    )?.args[2];
  assert.equal(pullPlatform('ghcr.io/kubeflow/kfp-metadata-writer:2.17.1'), 'linux/amd64');
  assert.equal(
    pullPlatform('gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0'),
    'linux/amd64',
  );
  assert.equal(pullPlatform('mysql:8.4'), 'linux/arm64');
  assert.equal(
    calls.some(({ args }) => args.includes('mysql:8.0.3')),
    false,
  );

  assert.equal(overlays.length, 1);
  const [platformAgnosticResource] = overlays[0].resources;
  assert.equal(path.isAbsolute(platformAgnosticResource), false);
  assert.equal(
    overlayResourceTarget,
    fs.realpathSync(path.join(revisionRoot, 'manifests', 'kustomize', 'env', 'platform-agnostic')),
  );
  const pullPolicyPatches = overlays[0].patches.map(({ patch, target }) => ({
    container: JSON.parse(patch).spec.template.spec.containers[0],
    target,
  }));
  assert.deepEqual(
    pullPolicyPatches.map(({ container, target }) => ({
      container: container.name,
      deployment: target.name,
      imagePullPolicy: container.imagePullPolicy,
    })),
    [
      {
        container: 'main',
        deployment: 'metadata-writer',
        imagePullPolicy: 'IfNotPresent',
      },
      {
        container: 'container',
        deployment: 'metadata-grpc-deployment',
        imagePullPolicy: 'IfNotPresent',
      },
    ],
  );
});

test('arm64 source builds keep the metadata writer on its reviewed amd64 workload', async (t) => {
  const calls = [];
  let localMetadataWriterImage;
  let renderedWithPullPolicy;
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (command === 'kubectl' && args[0] === 'kustomize') {
      const outputPath = args[args.indexOf('--output') + 1];
      const kustomizationPath = path.join(args[1], 'kustomization.yaml');
      const pullPolicy = fs.existsSync(kustomizationPath) ? 'IfNotPresent' : undefined;
      fs.mkdirSync(path.dirname(outputPath), { recursive: true });
      const manifest = mixedPlatformManifest({
        metadataWriterImage: localMetadataWriterImage,
        pullPolicy,
      });
      if (pullPolicy) renderedWithPullPolicy = manifest;
      fs.writeFileSync(outputPath, manifest);
    }
    if (
      command === 'kubectl' &&
      args.includes('get') &&
      args.includes('deployments') &&
      args.some((argument) => argument.startsWith('jsonpath='))
    ) {
      return success('metadata-grpc-deployment\nmetadata-writer\nmysql');
    }
    return success();
  };
  const stack = createTestStack(t, { runner });
  const revisionRoot = createRevisionRoot(stack);
  const metadataWriter = COMPONENTS.find((component) => component.name === 'metadata-writer');
  const driver = COMPONENTS.find((component) => component.name === 'driver');

  const overrides = await stack.buildComponentImages([metadataWriter, driver], revisionRoot, {
    load: false,
    platform: 'linux/arm64',
    tagSuffix: 'head-sha',
  });
  localMetadataWriterImage = overrides.images['metadata-writer'];

  const buildPlatform = (component) =>
    calls.find(
      ({ args, command }) =>
        command === 'docker' && args[0] === 'build' && args.includes(component.dockerfile),
    )?.args[2];
  assert.equal(buildPlatform(metadataWriter), 'linux/amd64');
  assert.equal(buildPlatform(driver), 'linux/arm64');

  stack.loadImageOverrides(overrides, 'linux/arm64', { removeSourceAfterLoad: true });
  const metadataWriterSave = calls.find(
    ({ args, command }) =>
      command === 'docker' && args[0] === 'save' && args.at(-1) === localMetadataWriterImage,
  );
  assert.deepEqual(metadataWriterSave.args.slice(0, 3), ['save', '--platform', 'linux/amd64']);
  const metadataWriterLoadIndex = calls.findIndex(
    ({ args, command }) =>
      command === 'kind' && args[0] === 'load' && args.includes(metadataWriterSave.args[4]),
  );
  const metadataWriterRemovalIndex = calls.findIndex(
    ({ args, command }) =>
      command === 'docker' &&
      args[0] === 'image' &&
      args[1] === 'rm' &&
      args[2] === localMetadataWriterImage,
  );
  assert.ok(metadataWriterLoadIndex >= 0);
  assert.ok(metadataWriterRemovalIndex < metadataWriterLoadIndex);

  stack.applyKfpManifests(revisionRoot, {
    imageOverrides: overrides,
    platform: 'linux/arm64',
  });
  assert.match(renderedWithPullPolicy, /imagePullPolicy: IfNotPresent/);
});

test('isolated component builds release their private Buildx cache before cluster creation', async (t) => {
  const calls = [];
  const stack = createTestStack(t, {
    isolatedBuildCache: true,
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      return success();
    },
  });
  const selected = [
    COMPONENTS.find((component) => component.name === 'driver'),
    COMPONENTS.find((component) => component.name === 'launcher'),
  ];

  await stack.buildComponentImages(selected, '/repo', {
    load: false,
    platform: 'linux/arm64',
    tagSuffix: 'head-sha',
  });

  const createCalls = calls.filter(
    ({ args, command }) => command === 'docker' && args[0] === 'buildx' && args[1] === 'create',
  );
  const buildCalls = calls.filter(
    ({ args, command }) => command === 'docker' && args[0] === 'buildx' && args[1] === 'build',
  );
  const removalCalls = calls.filter(
    ({ args, command }) => command === 'docker' && args[0] === 'buildx' && args[1] === 'rm',
  );
  assert.deepEqual(
    calls.map(({ args }) => args.slice(0, 2).join(' ')),
    ['buildx create', 'buildx build', 'buildx rm', 'buildx create', 'buildx build', 'buildx rm'],
  );
  assert.equal(createCalls.length, 2);
  assert.equal(buildCalls.length, 2);
  assert.equal(removalCalls.length, 2);
  assert.ok(buildCalls.every(({ args }) => args.includes('--load')));
  const builderName = createCalls[0].args[createCalls[0].args.indexOf('--name') + 1];
  assert.ok(createCalls.every(({ args }) => args[args.indexOf('--name') + 1] === builderName));
  assert.ok(buildCalls.every(({ args }) => args[args.indexOf('--builder') + 1] === builderName));
  assert.ok(removalCalls.every(({ args }) => args.at(-1) === builderName));
});

test('isolated component builds release private caches and completed images after a build failure', async (t) => {
  const calls = [];
  let buildCount = 0;
  const stack = createTestStack(t, {
    isolatedBuildCache: true,
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      if (command === 'docker' && args[0] === 'buildx' && args[1] === 'build') {
        buildCount += 1;
        if (buildCount === 2) {
          return { success: false, error: 'injected build failure', output: '' };
        }
      }
      return success();
    },
  });
  const selected = [
    COMPONENTS.find((component) => component.name === 'driver'),
    COMPONENTS.find((component) => component.name === 'launcher'),
  ];

  await assert.rejects(
    stack.buildComponentImages(selected, '/repo', {
      load: false,
      platform: 'linux/arm64',
    }),
    /Failed to build launcher/,
  );
  assert.ok(
    calls.some(
      ({ args, command }) => command === 'docker' && args[0] === 'buildx' && args[1] === 'rm',
    ),
  );
  assert.ok(
    calls.some(
      ({ args, command }) =>
        command === 'docker' &&
        args[0] === 'image' &&
        args[1] === 'rm' &&
        args[2].includes('/driver:'),
    ),
  );
});

test('byte-identical component images are retagged per stack before normal image loading', (t) => {
  const calls = [];
  const stack = createTestStack(t, { runner: deploymentRunner(calls) });
  const visualization = COMPONENTS.find((component) => component.name === 'visualization');
  const sourceImage = 'kfp-ui-smoke/visualization:base-source';

  const overrides = stack.reuseComponentImages(
    [visualization],
    { deployments: [], images: { visualization: sourceImage }, runtimeEnvironment: {} },
    { platform: 'linux/arm64', tagSuffix: 'head-sha' },
  );

  assert.equal(
    overrides.images.visualization,
    'kfp-ui-smoke/visualization:ui-smoke-base-test-head-sha',
  );
  assert.deepEqual(overrides.deployments, [
    {
      container: 'ml-pipeline-visualizationserver',
      deployment: 'ml-pipeline-visualizationserver',
      image: overrides.images.visualization,
    },
  ]);
  const tagCall = calls.find(({ command, args }) => command === 'docker' && args[0] === 'image');
  assert.deepEqual(tagCall.args, ['image', 'tag', sourceImage, overrides.images.visualization]);

  stack.loadImageOverrides(overrides, 'linux/arm64');
  assert.ok(
    calls.some(
      ({ command, args }) =>
        command === 'docker' && args[0] === 'save' && args.includes(overrides.images.visualization),
    ),
  );
});

test('manifest overlays resolve resources from canonical paths across directory aliases', (t) => {
  const fixtureRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-overlay-alias-'));
  t.after(() => fs.rmSync(fixtureRoot, { force: true, recursive: true }));
  const canonicalWorkspace = path.join(fixtureRoot, 'real', 'workspace');
  const workspaceAlias = path.join(fixtureRoot, 'workspace-alias');
  fs.mkdirSync(canonicalWorkspace, { recursive: true });
  fs.symlinkSync(
    canonicalWorkspace,
    workspaceAlias,
    process.platform === 'win32' ? 'junction' : 'dir',
  );
  const revisionRoot = path.join(fixtureRoot, 'revision');
  const platformAgnostic = path.join(
    revisionRoot,
    'manifests',
    'kustomize',
    'env',
    'platform-agnostic',
  );
  fs.mkdirSync(platformAgnostic, { recursive: true });

  let overlayResource = null;
  let resolvedOverlayResource = null;
  const runner = (command, args) => {
    if (command === 'kubectl' && args[0] === 'kustomize') {
      const outputPath = args[args.indexOf('--output') + 1];
      const kustomizationPath = path.join(args[1], 'kustomization.yaml');
      let manifest = mixedPlatformManifest();
      if (fs.existsSync(kustomizationPath)) {
        const overlay = JSON.parse(fs.readFileSync(kustomizationPath, 'utf8'));
        [overlayResource] = overlay.resources;
        resolvedOverlayResource = path.resolve(fs.realpathSync(args[1]), overlayResource);
        manifest = mixedPlatformManifest({ pullPolicy: 'IfNotPresent' });
      }
      fs.mkdirSync(path.dirname(outputPath), { recursive: true });
      fs.writeFileSync(outputPath, manifest);
    }
    return success();
  };
  const stack = createTestStack(t, {
    archiveDir: path.join(workspaceAlias, 'archives'),
    runner,
  });

  stack.preflightReleaseImages(revisionRoot, {
    expectedRelease: '2.17.1',
    platform: 'linux/arm64',
  });

  assert.equal(path.isAbsolute(overlayResource), false);
  assert.equal(resolvedOverlayResource, fs.realpathSync(platformAgnostic));
});

test('mixed-platform manifest validation rejects stale workload placement and pull policy', (t) => {
  const calls = [];
  const stack = createTestStack(t, { runner: deploymentRunner(calls) });
  const missingPolicyPath = path.join(stack.archiveDir, 'missing-policy.yaml');
  const stalePlacementPath = path.join(stack.archiveDir, 'stale-placement.yaml');
  fs.mkdirSync(stack.archiveDir, { recursive: true });
  fs.writeFileSync(missingPolicyPath, mixedPlatformManifest());
  fs.writeFileSync(
    stalePlacementPath,
    mixedPlatformManifest({
      metadataWriterDeployment: 'renamed-metadata-writer',
      pullPolicy: 'IfNotPresent',
    }),
  );

  assert.throws(
    () => stack.preloadManifestImages(missingPolicyPath, 'linux/arm64'),
    /must use imagePullPolicy IfNotPresent/,
  );
  assert.throws(
    () => stack.preloadManifestImages(stalePlacementPath, 'linux/arm64'),
    /only approved for Deployment metadata-writer, container main.*renamed-metadata-writer/,
  );
  assert.equal(
    calls.some(({ command }) => command === 'docker'),
    false,
  );
});

test('arm64 preload runs one Kubernetes amd64 emulation canary from a pinned image', (t) => {
  const calls = [];
  let canary;
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (
      command === 'kubectl' &&
      args.includes('apply') &&
      args.at(-1).endsWith('ui-smoke-amd64-canary.json')
    ) {
      canary = JSON.parse(fs.readFileSync(args.at(-1), 'utf8'));
    }
    return success();
  };
  const stack = createTestStack(t, { runner });
  const manifestPath = path.join(stack.archiveDir, 'mixed-platform.yaml');
  fs.mkdirSync(stack.archiveDir, { recursive: true });
  fs.writeFileSync(manifestPath, mixedPlatformManifest({ pullPolicy: 'IfNotPresent' }));

  stack.preloadManifestImages(manifestPath, 'linux/arm64');

  assert.ok(canary);
  const container = canary.spec.template.spec.containers[0];
  assert.equal(container.image, cluster.AMD64_EMULATION_CANARY_LOCAL_IMAGE);
  assert.equal(container.imagePullPolicy, 'Never');
  assert.deepEqual(container.command, ['/bin/sh', '-c', 'exit 0']);
  assert.equal(container.securityContext.runAsNonRoot, true);
  const canaryPull = calls.find(
    ({ args, command }) =>
      command === 'docker' &&
      args[0] === 'pull' &&
      args.at(-1) === cluster.AMD64_EMULATION_CANARY_IMAGE,
  );
  assert.deepEqual(canaryPull.args.slice(0, 3), ['pull', '--platform', 'linux/amd64']);
  assert.ok(
    calls.some(
      ({ args, command }) =>
        command === 'docker' &&
        args[0] === 'tag' &&
        args[1] === cluster.AMD64_EMULATION_CANARY_IMAGE &&
        args[2] === cluster.AMD64_EMULATION_CANARY_LOCAL_IMAGE,
    ),
  );
  const canarySave = calls.find(
    ({ args, command }) =>
      command === 'docker' &&
      args[0] === 'save' &&
      args.at(-1) === cluster.AMD64_EMULATION_CANARY_LOCAL_IMAGE,
  );
  assert.deepEqual(canarySave.args.slice(0, 3), ['save', '--platform', 'linux/amd64']);
  assert.ok(
    calls.some(
      ({ args, command }) =>
        command === 'kubectl' &&
        args.includes('delete') &&
        args.includes('job/ui-smoke-amd64-canary'),
    ),
  );
  assert.equal(fs.existsSync(path.join(stack.archiveDir, 'ui-smoke-amd64-canary.json')), false);

  const callCount = calls.length;
  stack.preloadManifestImages(manifestPath, 'linux/arm64');
  assert.equal(calls.length, callCount);
});

test('arm64 deployment composes mixed-platform patches with fixture retry requirements', async (t) => {
  const calls = [];
  let appliedManifest = null;
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (command === 'kubectl' && args[0] === 'kustomize') {
      const outputPath = args[args.indexOf('--output') + 1];
      const kustomizationPath = path.join(args[1], 'kustomization.yaml');
      const manifest = fs.existsSync(kustomizationPath)
        ? mixedPlatformManifestWithWorkflowDefaults({ pullPolicy: 'IfNotPresent' })
        : mixedPlatformManifestWithWorkflowDefaults();
      fs.mkdirSync(path.dirname(outputPath), { recursive: true });
      fs.writeFileSync(outputPath, manifest);
    }
    if (
      command === 'kubectl' &&
      args.includes('apply') &&
      args.includes('-f') &&
      args.at(-1).endsWith('platform-agnostic.yaml')
    ) {
      appliedManifest = fs.readFileSync(args.at(-1), 'utf8');
    }
    if (
      command === 'kubectl' &&
      args.includes('get') &&
      args.includes('deployments') &&
      args.some((argument) => argument.startsWith('jsonpath='))
    ) {
      return success('metadata-grpc-deployment\nmetadata-writer\nmysql');
    }
    return success();
  };
  const stack = createTestStack(t, { runner });
  const revisionRoot = createRevisionRoot(stack);

  await stack.deployRevision(revisionRoot, {
    fixtureRequirements: { argoRetryPolicy: 'OnFailure' },
    platform: 'linux/arm64',
  });

  assert.ok(appliedManifest);
  const resources = parseAllDocuments(appliedManifest).map((document) => document.toJS());
  const container = (deployment, name) =>
    resources
      .find((resource) => resource.kind === 'Deployment' && resource.metadata.name === deployment)
      .spec.template.spec.containers.find((candidate) => candidate.name === name);
  assert.equal(container('metadata-writer', 'main').imagePullPolicy, 'IfNotPresent');
  assert.equal(container('metadata-grpc-deployment', 'container').imagePullPolicy, 'IfNotPresent');
  const workflowConfig = resources.find(
    (resource) =>
      resource.kind === 'ConfigMap' && resource.metadata.name === 'workflow-controller-configmap',
  );
  const workflowDefaults = parseDocument(workflowConfig.data.workflowDefaults);
  assert.equal(
    workflowDefaults.getIn(['spec', 'templateDefaults', 'retryStrategy', 'retryPolicy']),
    'OnFailure',
  );
  assert.equal(workflowDefaults.getIn(['spec', 'templateDefaults', 'retryStrategy', 'limit']), '2');

  const canaryCompletion = calls.findIndex(
    ({ args, command }) =>
      command === 'kubectl' && args.includes('wait') && args.includes('job/ui-smoke-amd64-canary'),
  );
  const manifestApply = calls.findIndex(
    ({ args, command }) =>
      command === 'kubectl' &&
      args.includes('apply') &&
      args.at(-1).endsWith('platform-agnostic.yaml'),
  );
  assert.ok(canaryCompletion >= 0 && canaryCompletion < manifestApply);
});

test('amd64 emulation failure is actionable and always removes the Kubernetes canary', (t) => {
  const calls = [];
  const runner = (command, args, options) => {
    calls.push({ args, command, options });
    if (
      command === 'kubectl' &&
      args.includes('wait') &&
      args.includes('job/ui-smoke-amd64-canary')
    ) {
      return { success: false, error: 'exec format error', output: '' };
    }
    return success();
  };
  const stack = createTestStack(t, { runner });
  const manifestPath = path.join(stack.archiveDir, 'mixed-platform.yaml');
  fs.mkdirSync(stack.archiveDir, { recursive: true });
  fs.writeFileSync(manifestPath, mixedPlatformManifest({ pullPolicy: 'IfNotPresent' }));

  assert.throws(
    () => stack.preloadManifestImages(manifestPath, 'linux/arm64'),
    /cannot execute a preloaded amd64 workload.*exec format error.*QEMU or Rosetta/,
  );
  assert.ok(
    calls.some(
      ({ args, command }) =>
        command === 'kubectl' &&
        args.includes('delete') &&
        args.includes('job/ui-smoke-amd64-canary'),
    ),
  );
  assert.equal(fs.existsSync(path.join(stack.archiveDir, 'ui-smoke-amd64-canary.json')), false);
});

test('release deployments reject rendered Kubeflow images from another revision', (t) => {
  const stack = createTestStack(t);
  assert.deepEqual(
    stack.validateReleaseManifestImages(
      [
        'ghcr.io/kubeflow/kfp-api-server:2.17.1',
        'ghcr.io/kubeflow/kfp-frontend:2.17.1',
        'docker.io/library/mysql:8.0',
      ],
      '2.17.1',
    ),
    ['ghcr.io/kubeflow/kfp-api-server:2.17.1', 'ghcr.io/kubeflow/kfp-frontend:2.17.1'],
  );
  assert.throws(
    () =>
      stack.validateReleaseManifestImages(
        ['ghcr.io/kubeflow/kfp-api-server:2.17.1', 'ghcr.io/kubeflow/kfp-frontend:master'],
        '2.17.1',
      ),
    /do not match release 2\.17\.1.*kfp-frontend:master/,
  );
});

test('head deployment rejects an unbuilt first-party image before workload apply', (t) => {
  const calls = [];
  const localApi = 'kfp-ui-smoke/apiserver:head-sha';
  const runner = deploymentRunner(calls, {
    manifest: renderedManifest([localApi, 'ghcr.io/kubeflow/kfp-new-component:master']),
  });
  const stack = createTestStack(t, { runner });
  const headRoot = createRevisionRoot(stack, 'head');

  assert.throws(
    () =>
      stack.applyKfpManifests(headRoot, {
        imageOverrides: {
          deployments: [
            {
              container: 'ml-pipeline-api-server',
              deployment: 'ml-pipeline',
              image: localApi,
            },
          ],
          images: { apiserver: localApi },
          runtimeEnvironment: {},
        },
        platform: 'linux/amd64',
        requireLocalFirstParty: true,
      }),
    /retain non-local Kubeflow images.*kfp-new-component:master/,
  );
  assert.equal(
    calls.some((call) => call.command === 'kubectl' && call.args.includes('apply')),
    false,
  );
});

test('local deployment rejects retained legacy first-party images before workload apply', (t) => {
  const calls = [];
  const localApi = 'kfp-ui-smoke/apiserver:head-sha';
  const legacyFrontend = 'gcr.io/ml-pipeline/frontend:master';
  const runner = deploymentRunner(calls, {
    manifest: renderedManifest([localApi, legacyFrontend]),
  });
  const stack = createTestStack(t, { runner });
  const headRoot = createRevisionRoot(stack, 'head');

  assert.throws(
    () =>
      stack.applyKfpManifests(headRoot, {
        imageOverrides: {
          deployments: [
            {
              container: 'ml-pipeline-api-server',
              deployment: 'ml-pipeline',
              image: localApi,
            },
          ],
          images: { apiserver: localApi },
          runtimeEnvironment: {},
        },
        platform: 'linux/amd64',
        requireLocalFirstParty: true,
      }),
    /retain non-local Kubeflow images.*gcr\.io\/ml-pipeline\/frontend:master/,
  );
  assert.equal(
    calls.some((call) => call.command === 'kubectl' && call.args.includes('apply')),
    false,
  );
});

test('local deployment validates the exact workload container for every image override', (t) => {
  const calls = [];
  const localApi = 'kfp-ui-smoke/apiserver:head-sha';
  const manifest = renderedManifest(['docker.io/library/alpine:3.23', localApi]).replace(
    'ghcr.io/kubeflow/kfp-driver:2.17.1',
    'docker.io/library/busybox:1.37',
  );
  const stack = createTestStack(t, {
    runner: deploymentRunner(calls, { manifest }),
  });
  const headRoot = createRevisionRoot(stack, 'head');

  assert.throws(
    () =>
      stack.applyKfpManifests(headRoot, {
        imageOverrides: {
          deployments: [
            {
              container: 'ml-pipeline-api-server',
              deployment: 'ml-pipeline',
              image: localApi,
            },
          ],
          images: { apiserver: localApi },
          runtimeEnvironment: {},
        },
        platform: 'linux/amd64',
        requireLocalFirstParty: true,
      }),
    /did not set Deployment ml-pipeline, container ml-pipeline-api-server.*head-sha/,
  );
  assert.equal(
    calls.some((call) => call.command === 'kubectl' && call.args.includes('apply')),
    false,
  );
});

test('local deployment validates runtime images on the exact API server environment', (t) => {
  const calls = [];
  const localApi = 'kfp-ui-smoke/apiserver:head-sha';
  const localDriver = 'kfp-ui-smoke/driver:head-sha';
  const manifest = [
    'apiVersion: apps/v1',
    'kind: Deployment',
    'metadata:',
    '  name: ml-pipeline',
    'spec:',
    '  template:',
    '    spec:',
    '      containers:',
    '        - name: ml-pipeline-api-server',
    `          image: ${localApi}`,
    '          env:',
    '            - name: V2_DRIVER_IMAGE',
    '              value: docker.io/library/alpine:3.23',
    '---',
    'apiVersion: v1',
    'kind: ConfigMap',
    'metadata:',
    '  name: unrelated-runtime-images',
    'data:',
    '  values: |',
    '    - name: V2_DRIVER_IMAGE',
    `      value: ${localDriver}`,
  ].join('\n');
  const stack = createTestStack(t, {
    runner: deploymentRunner(calls, { manifest }),
  });
  const headRoot = createRevisionRoot(stack, 'head');

  assert.throws(
    () =>
      stack.applyKfpManifests(headRoot, {
        imageOverrides: {
          deployments: [
            {
              container: 'ml-pipeline-api-server',
              deployment: 'ml-pipeline',
              image: localApi,
            },
          ],
          images: { apiserver: localApi, driver: localDriver },
          runtimeEnvironment: { V2_DRIVER_IMAGE: localDriver },
        },
        platform: 'linux/amd64',
        requireLocalFirstParty: true,
      }),
    /did not set ml-pipeline\/ml-pipeline-api-server V2_DRIVER_IMAGE.*driver:head-sha/,
  );
  assert.equal(
    calls.some((call) => call.command === 'kubectl' && call.args.includes('apply')),
    false,
  );
});

test('third-party preflight excludes legacy Kubeflow images reserved for local builds', (t) => {
  const calls = [];
  const legacyApi = 'gcr.io/ml-pipeline/api-server:master';
  const dependency = 'docker.io/library/alpine:3.23';
  const stack = createTestStack(t, {
    runner: deploymentRunner(calls, { manifest: renderedManifest([legacyApi, dependency]) }),
  });

  const result = stack.preflightThirdPartyImages('/head', { platform: 'linux/amd64' });

  assert.deepEqual(result.images, [dependency]);
  assert.equal(
    calls.some(
      ({ args, command }) => command === 'docker' && args[0] === 'pull' && args.includes(legacyApi),
    ),
    false,
  );
});

test('refuses to reuse the exact managed cluster before side effects', async (t) => {
  const calls = [];
  const stack = createTestStack(t, {
    runner: (command, args) => {
      calls.push({ args, command });
      if (command === 'kind' && args[0] === 'get') return success('ui-smoke-base-test');
      return success();
    },
  });

  await assert.rejects(stack.createCluster(), /Refusing to reuse.*destroy that exact stack/);
  assert.equal(
    calls.some((call) => call.command === 'kind' && call.args.includes('create')),
    false,
  );
  assert.equal(
    calls.some((call) => call.command === 'kind' && call.args.includes('delete')),
    false,
  );

  assert.deepEqual(stack.destroyOwnedCluster(), { skipped: true, success: true });
  assert.equal(
    calls.some((call) => call.command === 'kind' && call.args.includes('delete')),
    false,
  );
  assert.equal(stack.teardownCluster(), true);
  assert.equal(
    calls.filter((call) => call.command === 'kind' && call.args.includes('delete')).length,
    1,
  );
});

test('normal destruction deletes only a cluster created by this stack instance', async (t) => {
  const calls = [];
  const stack = createTestStack(t, {
    runner: (command, args) => {
      calls.push({ args, command });
      if (command === 'kind' && args[0] === 'get') return success('');
      return success();
    },
  });

  assert.deepEqual(stack.destroyOwnedCluster(), { skipped: true, success: true });
  await stack.createCluster();
  assert.equal(stack.destroyOwnedCluster().success, true);
  assert.deepEqual(stack.destroyOwnedCluster(), { skipped: true, success: true });
  assert.equal(
    calls.filter((call) => call.command === 'kind' && call.args.includes('delete')).length,
    1,
  );
});

test('rolls back the exact cluster when revision deployment fails', async (t) => {
  const calls = [];
  const stack = createTestStack(t, {
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      if (command === 'kind' && args[0] === 'get') return success('');
      if (command === 'kubectl' && args.includes('nodes')) return success('amd64');
      if (command === 'docker' && args[0] === 'pull') {
        return { success: false, error: 'registry unavailable', output: '' };
      }
      return success();
    },
  });

  await assert.rejects(stack.ensureCluster('/revision'), /cannot be pulled.*registry unavailable/);
  assert.equal(
    calls.filter(
      (call) =>
        call.command === 'kind' &&
        call.args.join(' ') ===
          `delete cluster --name ${stack.clusterName} --kubeconfig ${stack.kubeconfigPath}`,
    ).length,
    1,
  );
});

test('detects and removes a partial cluster left by failed Kind creation', async (t) => {
  const calls = [];
  let checks = 0;
  const stack = createTestStack(t, {
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      if (command === 'kind' && args[0] === 'get') {
        checks++;
        return success(checks === 1 ? '' : 'ui-smoke-base-test');
      }
      if (command === 'kind' && args[0] === 'create') {
        return { success: false, error: 'node created before failure', output: '' };
      }
      return success();
    },
  });

  await assert.rejects(stack.createCluster(), /Failed to create Kind cluster/);
  assert.equal(checks, 2);
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kind' &&
        call.args.join(' ') ===
          `delete cluster --name ${stack.clusterName} --kubeconfig ${stack.kubeconfigPath}`,
    ),
  );
});

test('preserves setup and rollback errors when exact cluster deletion fails', async (t) => {
  const stack = createTestStack(t, {
    runner: (command, args) => {
      if (command === 'kind' && args[0] === 'get') return success('');
      if (command === 'kubectl' && args.includes('nodes')) return success('amd64');
      if (command === 'docker' && args[0] === 'pull') {
        return { success: false, error: 'registry unavailable', output: '' };
      }
      if (command === 'kind' && args[0] === 'delete') {
        return { success: false, error: 'delete unavailable', output: '' };
      }
      return success();
    },
  });

  await assert.rejects(stack.ensureCluster('/revision'), (error) => {
    assert.ok(error instanceof AggregateError);
    assert.match(error.message, /setup and rollback both failed/);
    assert.equal(error.errors.length, 2);
    assert.match(error.errors[0].message, /cannot be pulled.*registry unavailable/);
    assert.match(error.errors[1].message, /Failed to roll back managed Kind cluster/);
    return true;
  });
});

test('component tags and image archives are scoped to the stack', async (t) => {
  const calls = [];
  const stack = createTestStack(t, {
    clusterName: 'ui-smoke-run-base',
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      if (command === 'kubectl' && args.includes('nodes')) return success('arm64');
      return success();
    },
  });
  const selected = [
    COMPONENTS.find((component) => component.name === 'apiserver'),
    COMPONENTS.find((component) => component.name === 'driver'),
    COMPONENTS.find((component) => component.name === 'launcher'),
  ];

  const result = await stack.buildAndDeployComponents(selected, '/repo', {
    buildMetadata: BUILD_METADATA,
    tagSuffix: 'test-sha',
  });
  assert.equal(result.images.apiserver, 'kfp-ui-smoke/apiserver:ui-smoke-run-base-test-sha');
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'docker' &&
        call.args.includes('kfp-ui-smoke/apiserver:ui-smoke-run-base-test-sha') &&
        call.args.includes(`COMMIT_SHA=${BUILD_METADATA.commitSha}`),
    ),
  );
  assert.ok(
    calls
      .filter((call) => call.command === 'docker' && call.args[0] === 'build')
      .every((call) => call.args.includes('linux/arm64')),
  );
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kind' &&
        call.args[0] === 'load' &&
        call.args[1] === 'image-archive' &&
        call.args.includes('ui-smoke-run-base'),
    ),
  );
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args.includes(
          'ml-pipeline-api-server=kfp-ui-smoke/apiserver:ui-smoke-run-base-test-sha',
        ),
    ),
  );
  const environmentCall = calls.find(
    (call) => call.command === 'kubectl' && call.args.includes('env'),
  );
  assert.ok(
    environmentCall.args.includes('V2_DRIVER_IMAGE=kfp-ui-smoke/driver:ui-smoke-run-base-test-sha'),
  );
  assert.ok(
    environmentCall.args.includes(
      'V2_LAUNCHER_IMAGE=kfp-ui-smoke/launcher:ui-smoke-run-base-test-sha',
    ),
  );
});

test('deployRevision patches exact component images before the first workload apply', async (t) => {
  const calls = [];
  let overlay;
  const apiserver = COMPONENTS.find((component) => component.name === 'apiserver');
  const stack = createTestStack(t, {
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      if (command === 'kubectl' && args.includes('nodes')) return success('amd64');
      if (command === 'kubectl' && args[0] === 'kustomize') {
        const kustomizationPath = path.join(args[1], 'kustomization.yaml');
        overlay = JSON.parse(fs.readFileSync(kustomizationPath, 'utf8'));
        const image = JSON.parse(overlay.patches[0].patch).spec.template.spec.containers[0].image;
        const outputPath = args[args.indexOf('--output') + 1];
        fs.writeFileSync(outputPath, renderedManifest([image]));
      }
      if (command === 'kubectl' && args.includes('deployments')) {
        return success('ml-pipeline');
      }
      return success();
    },
  });
  const headRoot = createRevisionRoot(stack, 'head');

  const result = await stack.deployRevision(headRoot, {
    buildMetadata: BUILD_METADATA,
    components: [apiserver],
    platform: 'linux/amd64',
    tagSuffix: 'head-sha',
  });

  assert.equal(result.images.apiserver, 'kfp-ui-smoke/apiserver:ui-smoke-base-test-head-sha');
  const patch = JSON.parse(overlay.patches[0].patch);
  assert.equal(patch.metadata.name, 'ml-pipeline');
  assert.equal(
    patch.spec.template.spec.containers[0].image,
    'kfp-ui-smoke/apiserver:ui-smoke-base-test-head-sha',
  );
  const buildIndex = calls.findIndex(
    (call) => call.command === 'docker' && call.args[0] === 'build',
  );
  const loadIndex = calls.findIndex(
    (call) =>
      call.command === 'kind' &&
      call.args[0] === 'load' &&
      call.args.some((argument) => argument.includes('component-apiserver')),
  );
  const applyIndex = calls.findIndex(
    (call) => call.command === 'kubectl' && call.args.includes('apply'),
  );
  assert.ok(buildIndex >= 0 && loadIndex > buildIndex && applyIndex > loadIndex);
  assert.deepEqual(
    calls[buildIndex].args.filter(
      (argument) => argument.startsWith('COMMIT_SHA=') || argument.startsWith('TAG_NAME='),
    ),
    [`COMMIT_SHA=${BUILD_METADATA.commitSha}`, `TAG_NAME=${BUILD_METADATA.tagName}`],
  );
  assert.equal(
    calls.some(
      (call) =>
        call.command === 'kubectl' && call.args.includes('set') && call.args.includes('image'),
    ),
    false,
  );
});

test('every live image deployment and manifest mutation failure remains fatal', async (t) => {
  const apiserver = COMPONENTS.find((component) => component.name === 'apiserver');
  const failures = [
    {
      matches: (command, args) => command === 'kind' && args[0] === 'load',
      message: /Failed to load/,
    },
    {
      matches: (command, args) =>
        command === 'kubectl' && args.includes('set') && args.includes('image'),
      message: /Failed to set image/,
    },
    {
      matches: (command, args) => command === 'kubectl' && args.includes('patch'),
      message: /Failed to set IfNotPresent/,
    },
    {
      matches: (command, args) => command === 'kubectl' && args.includes('restart'),
      message: /Failed to restart/,
    },
    {
      matches: (command, args) => command === 'kubectl' && args.includes('status'),
      message: /did not become ready/,
    },
  ];
  for (const [index, failure] of failures.entries()) {
    const stack = createTestStack(t, {
      clusterName: `ui-smoke-failure-${index}`,
      runner: (command, args) => {
        if (command === 'kubectl' && args.includes('nodes')) return success('amd64');
        return failure.matches(command, args)
          ? { success: false, error: 'injected failure', output: '' }
          : success();
      },
    });
    await assert.rejects(
      stack.buildAndDeployComponents([apiserver], '/repo', {
        buildMetadata: BUILD_METADATA,
        tagSuffix: 'failure',
      }),
      failure.message,
    );
  }
});

test('revision image builds require and forward exact frontend provenance', async (t) => {
  const calls = [];
  const frontend = COMPONENTS.find((component) => component.name === 'frontend');
  assert.ok(frontend, 'frontend image must be part of the exact revision stack');
  const stack = createTestStack(t, { runner: deploymentRunner(calls) });

  await assert.rejects(
    stack.buildComponentImages([frontend], '/head', { platform: 'linux/amd64' }),
    /frontend requires non-empty build metadata commitSha/,
  );
  await stack.buildComponentImages([frontend], '/head', {
    buildMetadata: BUILD_METADATA,
    platform: 'linux/amd64',
    tagSuffix: 'head-sha',
  });

  const build = calls.find((call) => call.command === 'docker' && call.args[0] === 'build');
  assert.ok(build);
  assert.equal(build.options.timeout, 30 * 60 * 1000);
  for (const expected of [
    `COMMIT_HASH=${BUILD_METADATA.commitSha}`,
    `TAG_NAME=${BUILD_METADATA.tagName}`,
    `DATE=${BUILD_METADATA.buildDate}`,
    `NODE_VERSION=${BUILD_METADATA.nodeVersion}`,
  ]) {
    assert.ok(build.args.includes(expected), `missing Docker build argument ${expected}`);
  }
});

test('managed child termination waits and escalates when SIGTERM is ignored', async () => {
  const child = new FakeChild();
  const signals = [];
  let closed = false;
  child.kill = (signal) => {
    signals.push(signal);
    if (signal === 'SIGKILL') {
      setTimeout(() => {
        closed = true;
        child.signalCode = signal;
        child.emit('close', null, signal);
      }, 10);
    }
    return true;
  };

  await cluster.terminateChild(child, 1);
  assert.deepEqual(signals, ['SIGTERM', 'SIGKILL']);
  assert.equal(closed, true);
});

test('port forwarding uses each stack context, kubeconfig, and optional service set', async (t) => {
  const base = createTestStack(t);
  const head = createTestStack(t, {
    clusterName: 'ui-smoke-head-forward',
    context: 'kind-ui-smoke-head-forward',
    kubeconfigPath: path.join(path.dirname(base.kubeconfigPath), 'head-kubeconfig'),
    ports: { api: 3502, frontendServer: 3501, metadata: null, objectStore: 9500 },
  });
  const calls = [];
  const spawnFn = (command, args, options) => {
    calls.push({ args, command, options });
    return new FakeChild(`${command}-${calls.length}`);
  };
  const ready = async (_port, _timeout, { child }) => child.exitCode === null;

  const baseChildren = await base.ensurePortForwarding({
    portInUse: async () => false,
    spawnFn,
    waitForTcpFn: ready,
  });
  const headChildren = await head.ensurePortForwarding({
    portInUse: async () => false,
    spawnFn,
    waitForTcpFn: ready,
  });

  assert.equal(baseChildren.length, 3);
  assert.equal(headChildren.length, 2);
  assert.ok(calls.slice(0, 3).every((call) => call.args.includes(base.kubeconfigPath)));
  assert.ok(calls.slice(3).every((call) => call.args.includes(head.kubeconfigPath)));
  assert.ok(calls.slice(0, 3).every((call) => call.options.env.KUBECONFIG === base.kubeconfigPath));
  assert.ok(calls.slice(3).every((call) => call.options.env.KUBECONFIG === head.kubeconfigPath));
  assert.equal(
    calls.slice(3).some((call) => call.args.includes('svc/metadata-envoy-service')),
    false,
  );

  await assert.rejects(
    base.ensurePortForwarding({ portInUse: async () => true }),
    /became occupied/,
  );
});

test('deployed UI forwarding targets the isolated stack service', async (t) => {
  const stack = createTestStack(t, {
    ports: { api: 3602, frontendServer: 3601, metadata: null, objectStore: 9600 },
  });
  const calls = [];
  const child = new FakeChild('deployed-ui');

  const children = await stack.ensureDeployedUiPortForwarding({
    portInUse: async () => false,
    spawnFn(command, args, options) {
      calls.push({ args, command, options });
      return child;
    },
    waitForTcpFn: async (port, _timeout, options) => port === 3601 && options.child === child,
  });

  assert.deepEqual(children, [child]);
  assert.equal(stack.deployedUiUrl, 'http://127.0.0.1:3601');
  assert.ok(calls[0].args.includes('svc/ml-pipeline-ui'));
  assert.ok(calls[0].args.includes('3601:80'));
  assert.equal(calls[0].options.env.KUBECONFIG, stack.kubeconfigPath);
});

test('stack diagnostics are bounded and always use the run-scoped kubeconfig and context', async (t) => {
  const calls = [];
  let clusterExists = false;
  const stack = createTestStack(t, {
    runner(command, args, options) {
      calls.push({ args, command, options });
      if (command === 'kind' && args[0] === 'get') {
        return success(clusterExists ? 'ui-smoke-base-test' : '');
      }
      if (command === 'kind' && args[0] === 'create') clusterExists = true;
      if (
        command === 'kubectl' &&
        args.includes('get') &&
        args.includes('pods') &&
        args.some((argument) => argument.startsWith('jsonpath='))
      ) {
        return success('workflow-task-z\nml-pipeline-a\nmysql-b\nml-pipeline-ui-z\nseaweedfs-a\n');
      }
      if (command === 'kubectl' && args.includes('df')) {
        return success(
          'Filesystem Size Used Available Use% Mounted on\n/dev/sda 100G 99G 1G 99% /data',
        );
      }
      if (command === 'kubectl' && args.includes('wget')) {
        return { success: false, output: '', error: 'master is unavailable' };
      }
      if (command === 'kubectl' && args.includes('logs')) {
        return success(
          `Authorization: Bearer secret-token\nDATABASE_PASSWORD=hunter2\n${'x'.repeat(4096)}`,
        );
      }
      return success('ready');
    },
  });
  await stack.createCluster();
  calls.length = 0;

  const diagnostic = stack.collectDiagnostics({
    maxOutputBytes: 1024,
    maxPods: 2,
    tailLines: 5,
  });

  assert.equal(diagnostic.collected, true);
  assert.equal(diagnostic.owned, true);
  assert.equal(diagnostic.logs.length, 2);
  assert.deepEqual(
    diagnostic.logs.map((entry) => entry.name),
    ['pod-ml-pipeline-a', 'pod-ml-pipeline-ui-z'],
  );
  assert.ok(diagnostic.logs.every((entry) => entry.truncated));
  assert.ok(diagnostic.logs.every((entry) => entry.bytes < 1600));
  const diskSpace = diagnostic.status.find((entry) => entry.name === 'seaweedfs-disk-space');
  assert.ok(diskSpace.success);
  assert.match(diskSpace.preview, /99% \/data/);
  assert.deepEqual(diskSpace.command.slice(-8), [
    'exec',
    'pod/seaweedfs-a',
    '-c',
    'seaweedfs',
    '--',
    'df',
    '-h',
    '/data',
  ]);
  const volumeStatus = diagnostic.status.find((entry) => entry.name === 'seaweedfs-volume-status');
  assert.equal(volumeStatus.success, false);
  assert.match(volumeStatus.preview, /master is unavailable/);
  assert.deepEqual(volumeStatus.command.slice(-3), [
    'wget',
    '-qO-',
    'http://127.0.0.1:9333/dir/status',
  ]);
  assert.ok(diagnostic.status.every((entry) => !Object.hasOwn(entry, 'diagnosticOutput')));
  assert.ok(
    calls
      .filter((call) => call.command === 'kubectl')
      .every(
        (call) =>
          call.args[0] === '--kubeconfig' &&
          call.args[1] === stack.kubeconfigPath &&
          call.args[2] === '--context' &&
          call.args[3] === stack.context &&
          call.options.env.KUBECONFIG === stack.kubeconfigPath,
      ),
  );
  assert.ok(
    diagnostic.logs.every(
      (entry) =>
        entry.command.includes('<run-scoped-kubeconfig>') &&
        !entry.command.includes(stack.kubeconfigPath),
    ),
  );
  assert.ok(diagnostic.logs.every((entry) => /^[a-f0-9]{64}$/.test(entry.sha256)));
  assert.ok(diagnostic.logs.every((entry) => entry.artifactPath.startsWith('diagnostics/')));
  assert.ok(diagnostic.logs.every((entry) => entry.preview.includes('<redacted>')));
  assert.ok(diagnostic.logs.every((entry) => !/secret-token|hunter2/.test(entry.preview)));
  for (const entry of diagnostic.logs) {
    const contents = fs.readFileSync(path.join(stack.archiveDir, entry.artifactPath), 'utf8');
    assert.doesNotMatch(contents, /secret-token|hunter2/);
    assert.match(contents, /<redacted>/);
  }
});

test('process cleanup is isolated between stack instances', async (t) => {
  const base = createTestStack(t);
  const head = createTestStack(t, {
    clusterName: 'ui-smoke-head-processes',
    context: 'kind-ui-smoke-head-processes',
    kubeconfigPath: path.join(path.dirname(base.kubeconfigPath), 'head-process-kubeconfig'),
  });
  const baseChild = base.spawnProcess('base-command', [], { spawnFn: () => new FakeChild('base') });
  const headChild = head.spawnProcess('head-command', [], { spawnFn: () => new FakeChild('head') });
  const terminated = [];

  await base.cleanup({ terminate: async (child) => terminated.push(child.id) });
  assert.deepEqual(terminated, ['base']);
  assert.equal(headChild.killedWith, null);
  await head.cleanup({ terminate: async (child) => terminated.push(child.id) });
  assert.deepEqual(terminated, ['base', 'head']);
  assert.equal(baseChild.killedWith, null);
});

test('process cleanup surfaces termination failures and retains stubborn children for retry', async (t) => {
  const stack = createTestStack(t);
  const child = stack.spawnProcess('stubborn-command', [], {
    spawnFn: () => new FakeChild('stubborn'),
  });
  let attempts = 0;

  await assert.rejects(
    stack.cleanup({
      async terminate(received) {
        attempts++;
        assert.equal(received, child);
        throw new Error('still running');
      },
    }),
    (error) => {
      assert.ok(error instanceof AggregateError);
      assert.match(error.message, /Failed to stop 1 process/);
      assert.match(error.errors[0].message, /still running/);
      return true;
    },
  );
  await stack.cleanup({
    async terminate(received) {
      attempts++;
      assert.equal(received, child);
      received.signalCode = 'SIGKILL';
    },
  });
  assert.equal(attempts, 2);
});

test('skipBuild starts a loopback-confined revision server with isolated endpoints', async (t) => {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'cluster-manager-server-'));
  t.after(() => fs.rmSync(repo, { recursive: true, force: true }));
  fs.mkdirSync(path.join(repo, 'frontend', 'server', 'dist'), { recursive: true });
  fs.mkdirSync(path.join(repo, 'frontend', 'build'), { recursive: true });
  fs.writeFileSync(path.join(repo, 'frontend', 'server', 'dist', 'server.js'), '');
  const stack = createTestStack(t);
  const child = new FakeChild();
  let spawnCall;

  const returned = await stack.startFrontendServer(repo, {
    env: {
      METADATA_ENVOY_SERVICE_SERVICE_HOST: 'ambient-metadata',
      METADATA_ENVOY_SERVICE_SERVICE_SCHEME: 'https',
      ML_PIPELINE_SERVICE_HOST: 'ambient-api',
      ML_PIPELINE_SERVICE_SCHEME: 'https',
    },
    skipBuild: true,
    spawnFn: (command, args, options) => {
      spawnCall = { args, command, options };
      return child;
    },
    waitForServiceFn: async () => true,
  });
  assert.equal(returned, child);
  assert.equal(spawnCall.command, 'node');
  assert.equal(spawnCall.args[0], '--require');
  assert.equal(spawnCall.args[2], 'dist/server.js');
  assert.equal(spawnCall.args.at(-1), '3101');
  const preloadPath = spawnCall.args[1];
  assert.equal(fs.statSync(preloadPath).isFile(), true);
  let listenArguments;
  const serverPrototype = {
    listen(...args) {
      listenArguments = args;
      return this;
    },
  };
  vm.runInNewContext(fs.readFileSync(preloadPath, 'utf8'), {
    require(moduleName) {
      assert.equal(moduleName, 'node:net');
      return { Server: { prototype: serverPrototype } };
    },
  });
  serverPrototype.listen(3101, () => {});
  assert.equal(listenArguments[0], 3101);
  assert.equal(listenArguments[1], '127.0.0.1');
  assert.equal(spawnCall.options.env.KUBECONFIG, stack.kubeconfigPath);
  assert.equal(spawnCall.options.env.MINIO_PORT, '9100');
  assert.equal(spawnCall.options.env.ML_PIPELINE_SERVICE_HOST, '127.0.0.1');
  assert.equal(spawnCall.options.env.METADATA_ENVOY_SERVICE_SERVICE_PORT, '9190');
  assert.equal(spawnCall.options.env.METADATA_ENVOY_SERVICE_SERVICE_HOST, '127.0.0.1');
  assert.equal(spawnCall.options.env.METADATA_ENVOY_SERVICE_SERVICE_SCHEME, 'http');
  assert.equal(spawnCall.options.env.ML_PIPELINE_SERVICE_PORT, '3102');
  assert.equal(spawnCall.options.env.ML_PIPELINE_SERVICE_SCHEME, 'http');
  stack.stopFrontendServer();
  assert.equal(child.killedWith, 'SIGTERM');
});

test('trusted server build installs locked dependencies before compiling', async (t) => {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'cluster-manager-server-build-'));
  t.after(() => fs.rmSync(repo, { recursive: true, force: true }));
  fs.mkdirSync(path.join(repo, 'frontend', 'server'), { recursive: true });
  fs.mkdirSync(path.join(repo, 'frontend', 'build'), { recursive: true });
  const calls = [];
  const child = new FakeChild();
  const stack = createTestStack(t);

  await stack.startFrontendServer(repo, {
    runner: (command, args, options) => {
      calls.push({ args, command, options });
      return success();
    },
    spawnFn: () => child,
    waitForServiceFn: async () => true,
  });
  assert.deepEqual(
    calls.map((call) => [call.command, ...call.args]),
    [
      ['npm', 'ci'],
      ['npm', 'run', 'build'],
    ],
  );
  assert.ok(calls.every((call) => call.options.cwd.endsWith('/frontend/server')));
  assert.ok(calls.every((call) => call.options.env.KUBECONFIG === stack.kubeconfigPath));
  stack.stopFrontendServer();
});
