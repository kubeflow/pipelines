const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const cluster = require('../cluster-manager');
const { COMPONENTS } = require('../detect-changes');

const BUILD_METADATA = Object.freeze({
  buildDate: '2026-09-01T12:00:00Z',
  commitSha: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
  nodeVersion: '24.14.0',
  tagName: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
});

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
      ['pull', '--platform', 'linux/amd64'],
      ['save', '--platform', 'linux/amd64'],
    ],
  );
  assert.equal(
    calls.some(({ command }) => command === 'kind'),
    false,
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
    /cannot be pulled.*linux\/arm64.*amd64 Kind node with emulation/,
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
    /cannot be exported.*linux\/arm64.*amd64 Kind node with emulation/,
  );
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

  assert.throws(
    () =>
      stack.applyKfpManifests('/head', {
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
        const kustomizationPath = path.join(args[1], 'kustomization.json');
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

  const result = await stack.deployRevision('/head', {
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
