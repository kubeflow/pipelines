const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const cluster = require('../cluster-manager');
const { COMPONENTS } = require('../detect-changes');

class FakeChild extends EventEmitter {
  constructor() {
    super();
    this.exitCode = null;
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

test('uses SeaweedFS and the same object-store rewrite environment as the dev script', () => {
  assert.equal(cluster.PORT_FORWARDS[2].service, 'seaweedfs');
  const environment = cluster.frontendServerEnvironment({ KEEP: 'yes' });
  assert.equal(environment.KEEP, 'yes');
  assert.equal(environment.MINIO_HOST, 'localhost');
  assert.equal(environment.MINIO_NAMESPACE, '');
  assert.equal(environment.FRONTEND_SERVER_NAMESPACE, 'kubeflow');
  assert.match(environment.MINIO_ENDPOINT_REWRITE, /seaweedfs\.kubeflow\.svc\.cluster\.local:80/);
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

test('cluster creation restores context and applies both manifest layers explicitly', async () => {
  const calls = [];
  const runner = (command, args, options) => {
    calls.push({ command, args, options });
    if (command === 'kind' && args[0] === 'get') return success('');
    if (command === 'kubectl' && args.join(' ') === 'config current-context') {
      return success('developer-context');
    }
    if (
      command === 'kubectl' &&
      args.includes('get') &&
      args.includes('deployment') &&
      args.includes('ml-pipeline')
    ) {
      return { success: false, error: 'not found', output: '' };
    }
    return success();
  };

  const result = await cluster.ensureCluster('/repo', { runner });
  assert.deepEqual(result, { created: true, context: 'kind-ui-smoke-test' });
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args.join(' ') === 'config use-context developer-context',
    ),
  );
  assert.ok(
    !calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args.join(' ').includes('config use-context kind-ui-smoke-test'),
    ),
  );
  const applyCalls = calls.filter(
    (call) => call.command === 'kubectl' && call.args.includes('apply'),
  );
  assert.equal(applyCalls.length, 2);
  assert.ok(applyCalls[0].args.at(-1).endsWith('manifests/kustomize/cluster-scoped-resources'));
  assert.ok(applyCalls[1].args.at(-1).endsWith('manifests/kustomize/env/platform-agnostic'));
  assert.ok(applyCalls.every((call) => call.args.includes('kind-ui-smoke-test')));

  const pullCall = calls.find(
    (call) =>
      call.command === 'docker' && call.args.join(' ') === `pull ${cluster.SEED_RUNTIME_IMAGE}`,
  );
  assert.ok(pullCall);
  const loadCall = calls.find(
    (call) =>
      call.command === 'kind' &&
      call.args.join(' ') ===
        `load docker-image ${cluster.SEED_RUNTIME_IMAGE} --name ${cluster.CLUSTER_NAME}`,
  );
  assert.ok(loadCall);
  assert.ok(calls.indexOf(pullCall) < calls.indexOf(loadCall));
  assert.ok(calls.indexOf(loadCall) < calls.indexOf(applyCalls[0]));

  const workloadWait = calls.find(
    (call) => call.command === 'kubectl' && call.args.includes('--for=condition=Available'),
  );
  assert.ok(workloadWait);
  assert.ok(workloadWait.args.includes('--timeout=10m'));
  assert.deepEqual(
    workloadWait.args.filter((argument) => argument.startsWith('deployment/')).sort(),
    cluster.PLATFORM_DEPLOYMENTS.map((deployment) => `deployment/${deployment}`).sort(),
  );
});

test('refuses to reuse a managed cluster with potentially stale images or data', async () => {
  const calls = [];
  const runner = (command, args) => {
    calls.push({ command, args });
    if (command === 'kind' && args[0] === 'get') return success('ui-smoke-test');
    return success();
  };

  await assert.rejects(cluster.ensureCluster('/repo', { runner }), /Refusing to reuse.*--teardown/);
  assert.equal(
    calls.some((call) => call.command === 'kind' && argsInclude(call, 'create')),
    false,
  );
  assert.equal(
    calls.some((call) => call.command === 'kind' && argsInclude(call, 'delete')),
    false,
  );
});

test('rolls back the exact managed cluster when setup fails after creation', async () => {
  const failures = [
    {
      description: 'context restore',
      matches: (command, args) =>
        command === 'kubectl' && args.join(' ') === 'config use-context developer-context',
      message: /Failed to restore the previous kubectl context/,
    },
    {
      description: 'seed image pull',
      matches: (command, args) => command === 'docker' && args[0] === 'pull',
      message: /Failed to pull deterministic seed image/,
    },
    {
      description: 'seed image load',
      matches: (command, args) => command === 'kind' && args[0] === 'load',
      message: /Failed to load deterministic seed image/,
    },
    {
      description: 'manifest apply',
      matches: (command, args) => command === 'kubectl' && args.includes('apply'),
      message: /Failed to apply cluster-scoped KFP manifests/,
    },
    {
      description: 'workload readiness',
      matches: (command, args) =>
        command === 'kubectl' && args.includes('--for=condition=Available'),
      message: /deployments did not all become available/,
    },
  ];

  for (const failure of failures) {
    const calls = [];
    const runner = (command, args, options) => {
      calls.push({ command, args, options });
      if (command === 'kind' && args[0] === 'get') return success('');
      if (command === 'kubectl' && args.join(' ') === 'config current-context') {
        return success('developer-context');
      }
      if (failure.matches(command, args)) {
        return { success: false, error: `${failure.description} failed`, output: '' };
      }
      return success();
    };

    await assert.rejects(cluster.ensureCluster('/repo', { runner }), failure.message);
    assert.equal(
      calls.filter(
        (call) =>
          call.command === 'kind' &&
          call.args.join(' ') === `delete cluster --name ${cluster.CLUSTER_NAME}`,
      ).length,
      1,
      `${failure.description} should roll back the managed cluster`,
    );
  }
});

test('detects and removes a partial cluster left by a failed kind create', async () => {
  const calls = [];
  let clusterChecks = 0;
  const runner = (command, args, options) => {
    calls.push({ command, args, options });
    if (command === 'kind' && args[0] === 'get') {
      clusterChecks++;
      return success(clusterChecks === 1 ? '' : cluster.CLUSTER_NAME);
    }
    if (command === 'kind' && args[0] === 'create') {
      return { success: false, error: 'create failed after starting a node', output: '' };
    }
    if (command === 'kubectl' && args.join(' ') === 'config current-context') {
      return success('developer-context');
    }
    return success();
  };

  await assert.rejects(cluster.ensureCluster('/repo', { runner }), /Failed to create Kind cluster/);
  assert.equal(clusterChecks, 2);
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kind' &&
        call.args.join(' ') === `delete cluster --name ${cluster.CLUSTER_NAME}`,
    ),
  );
});

test('preserves setup and rollback errors when managed cluster deletion fails', async () => {
  const runner = (command, args) => {
    if (command === 'kind' && args[0] === 'get') return success('');
    if (command === 'kubectl' && args.join(' ') === 'config current-context') {
      return success('developer-context');
    }
    if (command === 'docker' && args[0] === 'pull') {
      return { success: false, error: 'pull unavailable', output: '' };
    }
    if (command === 'kind' && args[0] === 'delete') {
      return { success: false, error: 'delete unavailable', output: '' };
    }
    return success();
  };

  await assert.rejects(cluster.ensureCluster('/repo', { runner }), (error) => {
    assert.ok(error instanceof AggregateError);
    assert.match(error.message, /setup and rollback both failed/);
    assert.equal(error.errors.length, 2);
    assert.match(error.errors[0].message, /Failed to pull deterministic seed image/);
    assert.match(error.errors[1].message, /Failed to roll back managed Kind cluster/);
    return true;
  });
});

function argsInclude(call, value) {
  return call.args.includes(value);
}

test('builds exact local tags, deploys standing images, and configures runtime images', async () => {
  const calls = [];
  const runner = (command, args, options) => {
    calls.push({ command, args, options });
    if (command === 'kubectl' && args.includes('nodes')) return success('arm64');
    return success();
  };
  const selected = [
    COMPONENTS.find((component) => component.name === 'apiserver'),
    COMPONENTS.find((component) => component.name === 'driver'),
    COMPONENTS.find((component) => component.name === 'launcher'),
  ];

  const result = await cluster.buildAndDeployComponents(selected, '/repo', {
    runner,
    tagSuffix: 'test-sha',
  });
  assert.equal(result.images.apiserver, 'kfp-ui-smoke/apiserver:test-sha');
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'docker' &&
        call.args.join(' ') ===
          'build --platform linux/arm64 --tag kfp-ui-smoke/apiserver:test-sha --file backend/Dockerfile .',
    ),
  );
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kind' &&
        call.args.join(' ') ===
          'load docker-image kfp-ui-smoke/driver:test-sha --name ui-smoke-test',
    ),
  );
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args.includes('ml-pipeline-api-server=kfp-ui-smoke/apiserver:test-sha'),
    ),
  );
  const patch = calls.find((call) => call.command === 'kubectl' && call.args.includes('patch'));
  assert.match(patch.args.at(-1), /"imagePullPolicy":"IfNotPresent"/);
  const environmentCall = calls.find(
    (call) => call.command === 'kubectl' && call.args.includes('env'),
  );
  assert.ok(environmentCall.args.includes('V2_DRIVER_IMAGE=kfp-ui-smoke/driver:test-sha'));
  assert.ok(environmentCall.args.includes('V2_LAUNCHER_IMAGE=kfp-ui-smoke/launcher:test-sha'));
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args.includes('restart') &&
        call.args.includes('deployment/ml-pipeline'),
    ),
  );
  assert.ok(
    calls.some(
      (call) =>
        call.command === 'kubectl' &&
        call.args.includes('status') &&
        call.args.includes('deployment/ml-pipeline'),
    ),
  );
});

test('every image deployment and manifest mutation failure is fatal', async () => {
  const apiserver = COMPONENTS.find((component) => component.name === 'apiserver');
  await assert.rejects(
    cluster.buildAndDeployComponents([apiserver], '/repo', {
      tagSuffix: 'failure',
      runner: (command, args) => {
        if (command === 'kubectl' && args.includes('nodes')) return success('amd64');
        return command === 'kind' && args[0] === 'load'
          ? { success: false, error: 'load failed' }
          : success();
      },
    }),
    /Failed to load/,
  );
  const mutationFailures = [
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
  for (const failure of mutationFailures) {
    await assert.rejects(
      cluster.buildAndDeployComponents([apiserver], '/repo', {
        tagSuffix: 'failure',
        runner: (command, args) => {
          if (command === 'kubectl' && args.includes('nodes')) return success('amd64');
          return failure.matches(command, args)
            ? { success: false, error: 'injected mutation failure' }
            : success();
        },
      }),
      failure.message,
    );
  }
  assert.throws(
    () =>
      cluster.reapplyManifests('/repo', {
        runner: (command, args) =>
          command === 'kubectl' && args.includes('apply')
            ? { success: false, error: 'apply failed' }
            : success(),
      }),
    /Failed to apply cluster-scoped/,
  );
});

test('managed child termination waits and escalates when SIGTERM is ignored', async () => {
  const child = new FakeChild();
  const signals = [];
  child.kill = (signal) => {
    signals.push(signal);
    if (signal === 'SIGKILL') {
      child.signalCode = signal;
      child.emit('close', null, signal);
    }
    return true;
  };

  await cluster.terminateChild(child, 1);
  assert.deepEqual(signals, ['SIGTERM', 'SIGKILL']);
});

test('waits for every port-forward process to survive and accept TCP connections', async () => {
  const calls = [];
  const children = [];
  const started = await cluster.ensurePortForwarding({
    portInUse: async () => false,
    spawnFn: (command, args) => {
      calls.push({ command, args });
      const child = new FakeChild();
      children.push(child);
      return child;
    },
    waitForTcpFn: async (_port, _timeout, { child }) => child.exitCode === null,
  });
  assert.equal(started.length, 3);
  assert.ok(calls.every((call) => call.args.includes('--context')));
  assert.ok(calls.some((call) => call.args.includes('svc/seaweedfs')));

  await assert.rejects(
    cluster.ensurePortForwarding({
      portInUse: async () => false,
      spawnFn: () => new FakeChild(),
      waitForTcpFn: async (_port, _timeout, { child }) => {
        child.exitCode = 1;
        child.emit('exit', 1, null);
        return false;
      },
    }),
    /exited before readiness/,
  );

  await assert.rejects(
    cluster.ensurePortForwarding({ portInUse: async () => true }),
    /became occupied/,
  );
});

test('skipBuild starts only an existing server artifact and does not scale cluster UI', async (t) => {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'cluster-manager-server-'));
  t.after(() => fs.rmSync(repo, { recursive: true, force: true }));
  fs.mkdirSync(path.join(repo, 'frontend', 'server', 'dist'), { recursive: true });
  fs.mkdirSync(path.join(repo, 'frontend', 'build'), { recursive: true });
  fs.writeFileSync(path.join(repo, 'frontend', 'server', 'dist', 'server.js'), '');
  const child = new FakeChild();
  const runnerCalls = [];
  let spawnCall;

  const returned = await cluster.startFrontendServer(repo, {
    skipBuild: true,
    runner: (...args) => {
      runnerCalls.push(args);
      return success();
    },
    spawnFn: (command, args, options) => {
      spawnCall = { command, args, options };
      return child;
    },
    waitForServiceFn: async () => true,
  });
  assert.equal(returned, child);
  assert.deepEqual(runnerCalls, []);
  assert.equal(spawnCall.command, 'node');
  assert.equal(spawnCall.options.env.MINIO_HOST, 'localhost');
  assert.equal(spawnCall.options.env.ML_PIPELINE_SERVICE_PORT, '3002');
  cluster.stopFrontendServer();
  assert.equal(child.killedWith, 'SIGTERM');
});

test('trusted server build installs its locked dependencies before compiling', async (t) => {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'cluster-manager-server-build-'));
  t.after(() => fs.rmSync(repo, { recursive: true, force: true }));
  fs.mkdirSync(path.join(repo, 'frontend', 'server'), { recursive: true });
  fs.mkdirSync(path.join(repo, 'frontend', 'build'), { recursive: true });
  const calls = [];
  const child = new FakeChild();

  await cluster.startFrontendServer(repo, {
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
  cluster.stopFrontendServer();
});
