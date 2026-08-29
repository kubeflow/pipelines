const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const {
  METRICS_EXECUTOR_OUTPUT,
  MINIMAL_PIPELINE_YAML,
  RESOURCE_DEFINITIONS,
  SEED_IMAGE,
  resolveApiUrl,
  seedData,
  uploadPipeline,
  validateDetailRoutes,
  waitForRunsStable,
} = require('../seed-data');

function temporaryManifest(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'seed-data-test-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  return path.join(directory, 'seed-manifest.json');
}

test('uploads a valid v2 pipeline as multipart form data', async () => {
  let call;
  const result = await uploadPipeline(
    'ui-smoke-example',
    'description',
    async (...args) => {
      call = args;
      return { pipeline_id: 'pipeline-1' };
    },
    { displayName: 'UI Smoke Example' },
  );

  assert.equal(result.pipeline_id, 'pipeline-1');
  assert.equal(call[0], 'POST');
  assert.match(call[1], /^\/apis\/v2beta1\/pipelines\/upload\?/);
  assert.match(call[1], /name=ui-smoke-example/);
  assert.match(call[1], /display_name=UI\+Smoke\+Example/);
  assert.equal(call[2], null);
  assert.match(call[3].headers['Content-Type'], /^multipart\/form-data; boundary=/);
  const multipart = call[3].rawBody.toString('utf8');
  assert.match(multipart, /name="uploadfile"; filename="ui-smoke-pipeline.yaml"/);
  assert.ok(multipart.includes(MINIMAL_PIPELINE_YAML));
  assert.ok(multipart.includes(SEED_IMAGE));
  assert.match(multipart, /schemaTitle: system\.Metrics/);
  assert.match(multipart, /schemaTitle: system\.ClassificationMetrics/);
  assert.ok(multipart.includes(JSON.stringify(METRICS_EXECUTOR_OUTPUT)));
  assert.match(multipart, /metadata_path="\$\(dirname "\$1"\)\/output_metadata\.json"/);
  assert.doesNotMatch(multipart, /\/tmp\/kfp_outputs\/output_metadata\.json/);
  assert.doesNotMatch(multipart, /pip install|kfp\.dsl\.executor_main/);
  assert.deepEqual(METRICS_EXECUTOR_OUTPUT.artifacts.scalar_metrics.artifacts[0].metadata, {
    accuracy: 0.92,
    loss: 0.08,
  });
  assert.equal(
    METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata.confidenceMetrics.length,
    5,
  );

  await assert.rejects(
    uploadPipeline('ui-smoke-missing-id', 'description', async () => ({})),
    /did not contain a resource ID/,
  );
});

test('API URLs preserve a configured path prefix', () => {
  assert.equal(
    resolveApiUrl('https://example.test/kfp-api', '/apis/v2beta1/healthz').toString(),
    'https://example.test/kfp-api/apis/v2beta1/healthz',
  );
  assert.equal(
    resolveApiUrl('https://example.test/', '/apis/v2beta1/healthz').toString(),
    'https://example.test/apis/v2beta1/healthz',
  );
});

test('fills each missing deterministic resource type instead of skipping on partial data', async (t) => {
  const manifestPath = temporaryManifest(t);
  const calls = [];
  const request = async (method, endpoint, body, options) => {
    calls.push({ method, endpoint, body, options });
    if (endpoint === '/apis/v2beta1/healthz') return { status: 'healthy' };
    if (endpoint.startsWith('/apis/v2beta1/pipelines?')) {
      return {
        pipelines: [
          {
            pipeline_id: 'pipeline-existing',
            name: RESOURCE_DEFINITIONS.pipelines[0].name,
            display_name: RESOURCE_DEFINITIONS.pipelines[0].displayName,
          },
        ],
      };
    }
    if (endpoint.startsWith('/apis/v2beta1/experiments?')) return { experiments: [] };
    if (endpoint.startsWith('/apis/v2beta1/runs?')) return { runs: [] };
    if (endpoint.startsWith('/apis/v2beta1/recurringruns?')) return { recurring_runs: [] };
    if (endpoint.startsWith('/apis/v2beta1/pipelines/pipeline-existing/versions?')) {
      return {
        pipeline_versions: [
          { pipeline_id: 'pipeline-existing', pipeline_version_id: 'version-existing' },
        ],
      };
    }
    if (method === 'POST' && endpoint === '/apis/v2beta1/experiments') {
      return { ...body, experiment_id: 'experiment-created' };
    }
    if (method === 'POST' && endpoint === '/apis/v2beta1/runs') {
      return { ...body, run_id: 'run-created' };
    }
    if (method === 'POST' && endpoint === '/apis/v2beta1/recurringruns') {
      return { ...body, recurring_run_id: 'recurring-created' };
    }
    if (method === 'GET') return {};
    throw new Error(`Unexpected request: ${method} ${endpoint}`);
  };

  const result = await seedData({
    pipelines: 1,
    experiments: 1,
    runs: 1,
    recurringRuns: 1,
    request,
    manifestPath,
    apiBase: 'http://seed.test',
    waitForRunsFn: async () => true,
  });

  assert.equal(result.success, true);
  assert.equal(result.skipped, false);
  assert.deepEqual(result.resources.pipelineIds, ['pipeline-existing']);
  assert.deepEqual(result.resources.experimentIds, ['experiment-created']);
  assert.deepEqual(result.resources.runIds, ['run-created']);
  assert.deepEqual(result.resources.recurringRunIds, ['recurring-created']);
  assert.ok(fs.existsSync(manifestPath));
  assert.ok(
    !calls.some(
      (call) =>
        call.method === 'POST' && call.endpoint.startsWith('/apis/v2beta1/pipelines/upload?'),
    ),
  );
  const runCall = calls.find(
    (call) => call.method === 'POST' && call.endpoint === '/apis/v2beta1/runs',
  );
  assert.equal(runCall.body.display_name, RESOURCE_DEFINITIONS.runs[0]);
  assert.deepEqual(runCall.body.pipeline_version_reference, {
    pipeline_id: 'pipeline-existing',
    pipeline_version_id: 'version-existing',
  });
});

test('propagates partial creation failures and does not write a manifest', async (t) => {
  const manifestPath = temporaryManifest(t);
  const request = async (method, endpoint, body) => {
    if (endpoint === '/apis/v2beta1/healthz') return {};
    if (method === 'GET' && endpoint.includes('?')) {
      if (endpoint.includes('/pipelines?')) return { pipelines: [] };
      if (endpoint.includes('/experiments?')) return { experiments: [] };
      if (endpoint.includes('/runs?')) return { runs: [] };
      if (endpoint.includes('/recurringruns?')) return { recurring_runs: [] };
    }
    if (method === 'POST' && endpoint === '/apis/v2beta1/experiments') {
      return { ...body, experiment_id: 'experiment-created' };
    }
    if (method === 'POST' && endpoint.startsWith('/apis/v2beta1/pipelines/upload?')) {
      throw new Error('pipeline upload rejected');
    }
    throw new Error(`Unexpected request: ${method} ${endpoint}`);
  };

  const result = await seedData({
    pipelines: 1,
    experiments: 1,
    runs: 1,
    recurringRuns: 1,
    request,
    manifestPath,
    waitForRunsFn: async () => true,
  });
  assert.equal(result.success, false);
  assert.match(result.error, /resource operation\(s\) failed/);
  assert.ok(result.failures.some((failure) => failure.error.includes('pipeline upload rejected')));
  assert.equal(fs.existsSync(manifestPath), false);
});

test('requires every seeded detail route to load before reporting success', async (t) => {
  const manifestPath = temporaryManifest(t);
  const request = async (method, endpoint) => {
    if (endpoint === '/apis/v2beta1/healthz') return {};
    if (endpoint.startsWith('/apis/v2beta1/pipelines?')) {
      return {
        pipelines: [
          {
            pipeline_id: 'pipeline-1',
            name: RESOURCE_DEFINITIONS.pipelines[0].name,
            display_name: RESOURCE_DEFINITIONS.pipelines[0].displayName,
          },
        ],
      };
    }
    if (endpoint.startsWith('/apis/v2beta1/experiments?')) {
      return {
        experiments: [
          {
            experiment_id: 'experiment-1',
            display_name: RESOURCE_DEFINITIONS.experiments[0].displayName,
          },
        ],
      };
    }
    if (endpoint.startsWith('/apis/v2beta1/runs?')) {
      return { runs: [{ run_id: 'run-1', display_name: RESOURCE_DEFINITIONS.runs[0] }] };
    }
    if (endpoint.startsWith('/apis/v2beta1/recurringruns?')) {
      return {
        recurring_runs: [
          {
            recurring_run_id: 'recurring-1',
            display_name: RESOURCE_DEFINITIONS.recurringRuns[0],
          },
        ],
      };
    }
    if (endpoint.startsWith('/apis/v2beta1/pipelines/pipeline-1/versions?')) {
      return { pipeline_versions: [{ pipeline_version_id: 'version-1' }] };
    }
    if (method === 'GET' && endpoint === '/apis/v2beta1/runs/run-1') {
      throw new Error('run detail unavailable');
    }
    if (method === 'GET') return {};
    throw new Error(`Unexpected request: ${method} ${endpoint}`);
  };

  const result = await seedData({
    pipelines: 1,
    experiments: 1,
    runs: 1,
    recurringRuns: 1,
    request,
    manifestPath,
    waitForRunsFn: async () => true,
  });
  assert.equal(result.success, false);
  assert.match(result.error, /detail route validation failed.*run detail unavailable/);
  assert.equal(fs.existsSync(manifestPath), false);
});

test('validates every run used by run-list and detail screenshots', async () => {
  const endpoints = [];
  const resources = {
    experimentIds: ['experiment-1'],
    pipelineIds: ['pipeline-1'],
    pipelineVersionIds: ['version-1'],
    recurringRunIds: ['recurring-1'],
    runIds: ['run-1', 'run-2', 'run-3'],
  };
  await assert.rejects(
    validateDetailRoutes(resources, async (_method, endpoint) => {
      endpoints.push(endpoint);
      if (endpoint.endsWith('/runs/run-2')) throw new Error('secondary run unavailable');
      return {};
    }),
    /secondary run unavailable/,
  );
  assert.ok(endpoints.includes('/apis/v2beta1/runs/run-1'));
  assert.ok(endpoints.includes('/apis/v2beta1/runs/run-2'));
  assert.ok(endpoints.includes('/apis/v2beta1/runs/run-3'));
});

test('waits for every seeded run to succeed and rejects terminal failures', async () => {
  let clock = 0;
  let requests = 0;
  const request = async (_method, endpoint) => {
    requests++;
    const runId = endpoint.split('/').at(-1);
    if (runId === 'run-1' && requests < 3) return { state: 'RUNNING' };
    return { state: 'SUCCEEDED' };
  };

  await waitForRunsStable(['run-1', 'run-2'], request, {
    interval: 10,
    now: () => clock,
    sleep: async (milliseconds) => {
      clock += milliseconds;
    },
    timeout: 100,
  });
  assert.ok(requests >= 3);

  await assert.rejects(
    waitForRunsStable(
      ['failed-run'],
      async () => ({ state: 'FAILED', error: 'container image could not be pulled' }),
      {
        interval: 10,
        now: () => clock,
        sleep: async (milliseconds) => {
          clock += milliseconds;
        },
        timeout: 20,
      },
    ),
    /failed-run reached FAILED: container image could not be pulled/,
  );

  await assert.rejects(
    waitForRunsStable(['stuck-run'], async () => ({ state: 'RUNNING' }), {
      interval: 10,
      now: () => clock,
      sleep: async (milliseconds) => {
        clock += milliseconds;
      },
      timeout: 20,
    }),
    /Timed out.*stuck-run=RUNNING/,
  );
});
