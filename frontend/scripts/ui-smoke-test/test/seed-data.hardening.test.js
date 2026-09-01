const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');
const { parseDocument } = require('yaml');

const {
  METRICS_EXECUTOR_OUTPUT,
  MINIMAL_PIPELINE_YAML,
  RICH_PIPELINE_YAML,
  RESOURCE_DEFINITIONS,
  SEED_IMAGE,
  SEMANTIC_MARKER,
  buildSeedManifest,
  fetchMlmdArtifactsByIds,
  fetchRunBindingResponse,
  resolveApiUrl,
  seedData,
  uploadPipeline,
  validateDetailRoutes,
  waitForSemanticBindings,
  waitForRunsStable,
} = require('../seed-data');

test('projects revision-specific semantic IDs into capture defaults', () => {
  const manifest = buildSeedManifest(
    {
      experimentIds: ['experiment-1'],
      pipelineIds: ['pipeline-1'],
      recurringRunIds: ['recurring-1'],
      runIds: ['legacy-run-1'],
    },
    {
      apiBase: 'http://legacy.test',
      semantic: {
        bindings: {
          resources: {
            'run.evaluation': { id: 'semantic-evaluation' },
            'run.training-1': { id: 'semantic-training-1' },
            'run.training-2': { id: 'semantic-training-2' },
          },
          runs: {
            'run.training-1': {
              artifacts: {
                'artifact.scalar-metrics': {
                  artifactIds: ['81'],
                  members: { 'metric.accuracy': { artifactIds: ['81'] } },
                },
              },
              tasks: {
                'task.write-metrics': {
                  mlmdExecutionId: '73',
                  taskId: 'legacy-write-metrics',
                },
              },
            },
          },
        },
      },
    },
  );

  assert.equal(manifest.defaults.artifactId, '81');
  assert.equal(
    manifest.defaults.compareRunlist,
    'semantic-training-1,semantic-training-2,semantic-evaluation',
  );
  assert.equal(manifest.defaults.executionId, '73');
  assert.equal(manifest.defaults.runId, 'legacy-run-1');
  assert.equal(manifest.defaults.taskId, 'legacy-write-metrics');
});

function nativeRichRunResponse(runId = 'run-created') {
  const rocMetadata = structuredClone(
    METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
  );
  return {
    displayName: RESOURCE_DEFINITIONS.runs[0].displayName,
    runId,
    taskCount: 8,
    tasks: [
      {
        childTasks: [{ name: 'consume-metrics', taskId: 'task-consume-metrics' }],
        name: 'write-metrics',
        outputs: {
          artifacts: [
            {
              artifactKey: 'scalar_metrics',
              artifacts: [
                { artifactId: 'metric-loss', name: 'loss', numberValue: 0.08 },
                { artifactId: 'metric-accuracy', name: 'accuracy', numberValue: 0.92 },
              ],
            },
            {
              artifactKey: 'roc_curve',
              artifacts: [
                {
                  artifactId: 'roc-artifact',
                  metadata: rocMetadata,
                  name: 'roc_curve',
                },
              ],
            },
            {
              artifactKey: 'html_report',
              artifacts: [
                {
                  artifactId: 'html-artifact',
                  name: 'html_report',
                  type: 'HTML',
                  uri: 's3://fixtures/report.html',
                },
              ],
            },
            {
              artifactKey: 'markdown_report',
              artifacts: [
                {
                  artifactId: 'markdown-artifact',
                  name: 'markdown_report',
                  type: 'Markdown',
                  uri: 's3://fixtures/report.md',
                },
              ],
            },
          ],
        },
        scopePath: 'root.write-metrics',
        state: 'SUCCEEDED',
        taskId: 'task-write-metrics',
        type: 'RUNTIME',
      },
      {
        inputs: {
          artifacts: [
            {
              artifactKey: 'metrics',
              artifacts: [
                { artifactId: 'metric-loss', name: 'loss', numberValue: 0.08 },
                { artifactId: 'metric-accuracy', name: 'accuracy', numberValue: 0.92 },
              ],
            },
          ],
        },
        name: 'consume-metrics',
        scopePath: 'root.consume-metrics',
        state: 'SUCCEEDED',
        taskId: 'task-consume-metrics',
        type: 'RUNTIME',
      },
      {
        name: 'retry-once',
        pods: [
          { name: 'retry-0', type: 'EXECUTOR' },
          { name: 'retry-1', type: 'EXECUTOR' },
        ],
        scopePath: 'root.retry-once',
        state: 'SUCCEEDED',
        taskId: 'task-retry',
        type: 'RUNTIME',
      },
      {
        name: 'parallel-loop',
        scopePath: 'root.parallel-loop',
        state: 'SUCCEEDED',
        taskId: 'task-loop',
        type: 'LOOP',
        typeAttributes: { iterationCount: 2 },
      },
      {
        displayName: 'parallel-loop',
        name: 'parallel-loop-0',
        parentTaskId: 'task-loop',
        scopePath: 'root.parallel-loop.parallel-loop-0',
        state: 'SUCCEEDED',
        taskId: 'task-loop-scope-0',
        type: 'DAG',
        typeAttributes: { iterationIndex: 0 },
      },
      {
        displayName: 'parallel-loop',
        name: 'parallel-loop-1',
        parentTaskId: 'task-loop',
        scopePath: 'root.parallel-loop.parallel-loop-1',
        state: 'SUCCEEDED',
        taskId: 'task-loop-scope-1',
        type: 'DAG',
        typeAttributes: { iterationIndex: 1 },
      },
      {
        name: 'loop-worker',
        parentTaskId: 'task-loop-scope-0',
        scopePath: 'root.parallel-loop.loop-worker',
        state: 'SUCCEEDED',
        taskId: 'task-loop-worker-0',
        type: 'RUNTIME',
        typeAttributes: { iterationIndex: 0 },
      },
      {
        name: 'loop-worker',
        parentTaskId: 'task-loop-scope-1',
        scopePath: 'root.parallel-loop.loop-worker',
        state: 'SUCCEEDED',
        taskId: 'task-loop-worker-1',
        type: 'RUNTIME',
        typeAttributes: { iterationIndex: 1 },
      },
      {
        name: 'nested-dag',
        scopePath: 'root.nested-dag',
        state: 'SUCCEEDED',
        taskId: 'task-nested',
        type: 'DAG',
      },
      {
        name: 'nested-worker',
        parentTaskId: 'task-nested',
        scopePath: 'root.nested-dag.nested-worker',
        state: 'SUCCEEDED',
        taskId: 'task-nested-worker',
        type: 'RUNTIME',
      },
    ],
  };
}

test('fixture pipeline specs are valid YAML with intact deterministic report commands', () => {
  for (const [profile, source] of [
    ['minimal', MINIMAL_PIPELINE_YAML],
    ['rich-topology', RICH_PIPELINE_YAML],
  ]) {
    const document = parseDocument(source);
    assert.deepEqual(
      document.errors.map((error) => error.message),
      [],
      `${profile} fixture must parse as YAML`,
    );
    const pipeline = document.toJS();
    const args = pipeline.deploymentSpec.executors['exec-write-metrics'].container.args;
    assert.match(args[0], /printf '%s\\n' '<h1>UI Smoke HTML Report<\/h1>/);
    assert.match(args[0], /printf '%s\\n' '# UI Smoke Markdown Report'/);
    assert.deepEqual(args.slice(-2), [
      "{{$.outputs.artifacts['html_report'].path}}",
      "{{$.outputs.artifacts['markdown_report'].path}}",
    ]);
  }
});

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
  assert.match(multipart, /schemaTitle: system\.HTML/);
  assert.match(multipart, /schemaTitle: system\.Markdown/);
  assert.match(multipart, /UI Smoke HTML Report/);
  assert.match(multipart, /UI Smoke Markdown Report/);
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
  assert.deepEqual(METRICS_EXECUTOR_OUTPUT.artifacts.html_report.artifacts[0].metadata, {});
  assert.deepEqual(METRICS_EXECUTOR_OUTPUT.artifacts.markdown_report.artifacts[0].metadata, {});

  await assert.rejects(
    uploadPipeline('ui-smoke-missing-id', 'description', async () => ({})),
    /did not contain a resource ID/,
  );
});

test('rich fixture uses pinned revision-compatible topology without runtime installs', async () => {
  let call;
  await uploadPipeline(
    'ui-smoke-rich',
    'rich topology',
    async (...args) => {
      call = args;
      return { pipeline_id: 'pipeline-rich' };
    },
    { pipelineYaml: RICH_PIPELINE_YAML },
  );

  const multipart = call[3].rawBody.toString('utf8');
  assert.ok(multipart.includes(RICH_PIPELINE_YAML));
  assert.match(multipart, /retryPolicy:[\s\S]*maxRetryCount: 1/);
  assert.match(multipart, /KFP_RETRY_INDEX/);
  assert.match(multipart, /parameterIterator:[\s\S]*raw: '\["alpha", "beta"\]'/);
  assert.match(multipart, /comp-nested-dag:[\s\S]*nested-worker:/);
  assert.match(multipart, /taskOutputArtifact:[\s\S]*producerTask: write-metrics/);
  assert.equal((multipart.match(new RegExp(SEED_IMAGE, 'g')) || []).length, 5);
  assert.doesNotMatch(multipart, /pip install|kfp\.dsl\.executor_main|python:/);
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

test('resource definitions carry stable semantic keys', () => {
  for (const definitions of Object.values(RESOURCE_DEFINITIONS)) {
    for (const definition of definitions) {
      assert.match(definition.semanticKey, /^[a-z][a-z0-9.-]+$/);
      assert.ok(definition.displayName || definition.name);
    }
  }
  assert.deepEqual(
    RESOURCE_DEFINITIONS.runs
      .filter((definition) => definition.fixtureProfile === 'rich-topology')
      .map((definition) => definition.semanticKey),
    ['run.training-1'],
  );
  assert.equal(RESOURCE_DEFINITIONS.runs[0].pipelineSemanticKey, 'pipeline.training');
  assert.ok(
    RESOURCE_DEFINITIONS.runs
      .slice(1)
      .every((definition) => definition.fixtureProfile === 'metrics'),
  );
});

test('falls back from an unsupported FULL query to the legacy detail response', async () => {
  const endpoints = [];
  const response = await fetchRunBindingResponse('legacy/run', async (_method, endpoint) => {
    endpoints.push(endpoint);
    if (endpoint.endsWith('?view=FULL')) throw new Error('unknown query parameter view');
    return {
      run_details: { pipeline_context_id: '1', task_details: [] },
      run_id: 'legacy/run',
    };
  });

  assert.deepEqual(endpoints, [
    '/apis/v2beta1/runs/legacy%2Frun?view=FULL',
    '/apis/v2beta1/runs/legacy%2Frun',
  ]);
  assert.equal(response.run_id, 'legacy/run');
});

test('hydrates legacy run artifact IDs with actual MLMD metric and ROC values', async () => {
  const requestedArtifactIds = [];
  const response = await fetchRunBindingResponse(
    'legacy-run',
    async (_method, endpoint) => {
      assert.equal(endpoint, '/apis/v2beta1/runs/legacy-run?view=FULL');
      return {
        run_details: {
          pipeline_context_id: '1',
          task_details: [
            {
              display_name: 'write-metrics',
              outputs: {
                roc_curve: { artifact_ids: ['82'] },
                scalar_metrics: { artifact_ids: ['81'] },
              },
            },
          ],
        },
        run_id: 'legacy-run',
      };
    },
    {
      fetchLegacyArtifacts: async (artifactIds) => {
        requestedArtifactIds.push(...artifactIds);
        return [
          { artifactId: '81', metadata: { accuracy: 0.92, loss: 0.08 } },
          {
            artifactId: '82',
            metadata: structuredClone(
              METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
            ),
          },
        ];
      },
    },
  );

  assert.deepEqual(requestedArtifactIds, ['81', '82']);
  assert.deepEqual(response.semanticArtifacts[0].metadata, { accuracy: 0.92, loss: 0.08 });
  assert.equal(response.semanticArtifacts[1].metadata.confidenceMetrics.length, 5);
});

function testVarint(value) {
  let remaining = BigInt(value);
  const bytes = [];
  do {
    let byte = Number(remaining & 0x7fn);
    remaining >>= 7n;
    if (remaining !== 0n) byte |= 0x80;
    bytes.push(byte);
  } while (remaining !== 0n);
  return Buffer.from(bytes);
}

function testField(fieldNumber, wireType, payload) {
  return Buffer.concat([testVarint((fieldNumber << 3) | wireType), Buffer.from(payload)]);
}

function testMessageField(fieldNumber, message) {
  return testField(fieldNumber, 2, Buffer.concat([testVarint(message.length), message]));
}

function testStringField(fieldNumber, value) {
  return testMessageField(fieldNumber, Buffer.from(value));
}

function testDoubleField(fieldNumber, value) {
  const bytes = Buffer.alloc(8);
  bytes.writeDoubleLE(value);
  return testField(fieldNumber, 1, bytes);
}

function testGoogleValue(value) {
  if (value === null) return testField(1, 0, testVarint(0));
  if (typeof value === 'number') return testDoubleField(2, value);
  if (typeof value === 'string') return testStringField(3, value);
  if (typeof value === 'boolean') return testField(4, 0, testVarint(value ? 1 : 0));
  if (Array.isArray(value)) {
    return testMessageField(
      6,
      Buffer.concat(value.map((entry) => testMessageField(1, testGoogleValue(entry)))),
    );
  }
  return testMessageField(
    5,
    Buffer.concat(
      Object.entries(value).map(([key, entry]) =>
        testMessageField(
          1,
          Buffer.concat([testStringField(1, key), testMessageField(2, testGoogleValue(entry))]),
        ),
      ),
    ),
  );
}

function testMlmdValue(value) {
  if (typeof value === 'number') return testDoubleField(2, value);
  return testMessageField(
    4,
    testMessageField(
      1,
      Buffer.concat([testStringField(1, 'list'), testMessageField(2, testGoogleValue(value))]),
    ),
  );
}

function testMlmdArtifact(artifactId, metadata, details = {}) {
  return Buffer.concat([
    testField(1, 0, testVarint(artifactId)),
    ...(details.uri ? [testStringField(3, details.uri)] : []),
    ...Object.entries(metadata).map(([key, value]) =>
      testMessageField(
        5,
        Buffer.concat([testStringField(1, key), testMessageField(2, testMlmdValue(value))]),
      ),
    ),
    ...(details.name ? [testStringField(7, details.name)] : []),
    ...(details.type ? [testStringField(8, details.type)] : []),
  ]);
}

function testMlmdResponse(artifacts) {
  return Buffer.concat(
    artifacts.map(({ artifactId, metadata, ...details }) =>
      testMessageField(1, testMlmdArtifact(artifactId, metadata, details)),
    ),
  );
}

function grpcFrame(type, payload) {
  const frame = Buffer.alloc(5 + payload.length);
  frame[0] = type;
  frame.writeUInt32BE(payload.length, 1);
  Buffer.from(payload).copy(frame, 5);
  return frame;
}

test('decodes scalar and ROC values from the MLMD gRPC-web artifact response', async () => {
  const responseBytes = testMlmdResponse([
    { artifactId: 81, metadata: { accuracy: 0.92, loss: 0.08 } },
    {
      artifactId: 82,
      metadata: METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
    },
    {
      artifactId: 83,
      metadata: { display_name: 'html_report' },
      name: 'html-output',
      type: 'system.HTML',
      uri: 's3://fixtures/report.html',
    },
  ]);

  const artifacts = await fetchMlmdArtifactsByIds(
    ['81', '82', '83'],
    async (method, endpoint, body, options) => {
      assert.equal(method, 'POST');
      assert.equal(endpoint, '/ml_metadata.MetadataStoreService/GetArtifactsByID');
      assert.equal(body, null);
      assert.equal(options.responseType, 'buffer');
      assert.equal(options.headers['Content-Type'], 'application/grpc-web+proto');
      const requestFrame = options.rawBody;
      const requestLength = requestFrame.readUInt32BE(1);
      assert.equal(requestLength, 6);
      assert.equal(requestFrame.subarray(5, 5 + requestLength).toString('hex'), '085108520853');
      return Buffer.concat([
        grpcFrame(0x00, responseBytes),
        grpcFrame(0x80, Buffer.from('grpc-status: 0\r\n')),
      ]);
    },
  );

  assert.deepEqual(artifacts, [
    { artifactId: '81', metadata: { accuracy: 0.92, loss: 0.08 } },
    {
      artifactId: '82',
      metadata: structuredClone(METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata),
    },
    {
      artifactId: '83',
      metadata: { display_name: 'html_report' },
      name: 'html-output',
      type: 'system.HTML',
      uri: 's3://fixtures/report.html',
    },
  ]);
});

test('legacy MLMD hydration runs when generated protobuf modules are unavailable', () => {
  const responseBytes = Buffer.concat([
    grpcFrame(
      0x00,
      testMlmdResponse([{ artifactId: 81, metadata: { accuracy: 0.92, loss: 0.08 } }]),
    ),
    grpcFrame(0x80, Buffer.from('grpc-status: 0\r\n')),
  ]);
  const seedDataPath = path.resolve(__dirname, '../seed-data.js');
  const script = String.raw`
    const Module = require('node:module');
    const originalLoad = Module._load;
    Module._load = function(request, parent, isMain) {
      if (/google-protobuf|third_party[\\/]mlmd[\\/]generated/.test(request)) {
        throw new Error('blocked generated protobuf dependency: ' + request);
      }
      return originalLoad.call(this, request, parent, isMain);
    };
    const seedDataPath = process.argv[1];
    const responseBody = Buffer.from(process.argv[2], 'base64');
    const { fetchMlmdArtifactsByIds } = require(seedDataPath);
    fetchMlmdArtifactsByIds(['81'], async (_method, _endpoint, _body, options) => {
      process.stderr.write(options.rawBody.toString('hex'));
      return responseBody;
    }).then((artifacts) => process.stdout.write(JSON.stringify(artifacts)));
  `;

  const child = spawnSync(
    process.execPath,
    ['-e', script, seedDataPath, responseBytes.toString('base64')],
    { encoding: 'utf8' },
  );
  assert.equal(child.status, 0, child.stderr);
  assert.equal(child.stderr, '00000000020851');
  assert.deepEqual(JSON.parse(child.stdout), [
    { artifactId: '81', metadata: { accuracy: 0.92, loss: 0.08 } },
  ]);
});

test('hydrates native semantic bindings through the paginated Task/Artifact API', async () => {
  const native = nativeRichRunResponse('native/run');
  const tasks = native.tasks;
  const calls = [];
  const response = await fetchRunBindingResponse('native/run', async (_method, endpoint) => {
    calls.push(endpoint);
    if (endpoint === '/apis/v2beta1/runs/native%2Frun?view=FULL') {
      return { displayName: native.displayName, runId: native.runId, taskCount: tasks.length };
    }
    if (endpoint === '/apis/v2beta1/runs/native%2Frun/tasks?page_size=100') {
      return { tasks: tasks.slice(0, 4), next_page_token: 'next page' };
    }
    if (endpoint === '/apis/v2beta1/runs/native%2Frun/tasks?page_size=100&page_token=next+page') {
      return { tasks: tasks.slice(4) };
    }
    throw new Error(`Unexpected endpoint: ${endpoint}`);
  });

  assert.equal(response.runId, 'native/run');
  assert.deepEqual(response.tasks, tasks);
  assert.deepEqual(calls, [
    '/apis/v2beta1/runs/native%2Frun?view=FULL',
    '/apis/v2beta1/runs/native%2Frun/tasks?page_size=100',
    '/apis/v2beta1/runs/native%2Frun/tasks?page_size=100&page_token=next+page',
  ]);
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
    if (method === 'GET' && endpoint === '/apis/v2beta1/runs/run-created?view=FULL') {
      return nativeRichRunResponse();
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
  assert.equal(runCall.body.display_name, RESOURCE_DEFINITIONS.runs[0].displayName);
  assert.match(runCall.body.description, new RegExp(`${SEMANTIC_MARKER}=run\\.training-1`));
  assert.deepEqual(runCall.body.pipeline_version_reference, {
    pipeline_id: 'pipeline-existing',
    pipeline_version_id: 'version-existing',
  });
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  assert.equal(manifest.defaults.runId, 'run-created');
  assert.equal(manifest.defaults.artifactId, 'metric-accuracy');
  assert.equal(manifest.defaults.executionId, null);
  assert.equal(manifest.defaults.taskId, 'task-write-metrics');
  assert.deepEqual(manifest.resources.runIds, ['run-created']);
  assert.equal(manifest.semantic.revisionFlavor, 'native-task-artifact');
  assert.equal(manifest.semantic.validation.valid, true);
  const metricBinding =
    manifest.semantic.bindings.runs['run.training-1'].artifacts['artifact.scalar-metrics'];
  assert.deepEqual(metricBinding.artifactIds, ['metric-accuracy', 'metric-loss']);
  assert.deepEqual(Object.keys(metricBinding.members), ['metric.accuracy', 'metric.loss']);
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
      return {
        runs: [
          {
            run_id: 'run-1',
            display_name: RESOURCE_DEFINITIONS.runs[0].displayName,
          },
        ],
      };
    }
    if (endpoint.startsWith('/apis/v2beta1/recurringruns?')) {
      return {
        recurring_runs: [
          {
            recurring_run_id: 'recurring-1',
            display_name: RESOURCE_DEFINITIONS.recurringRuns[0].displayName,
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

test('polls semantic bindings until eventually consistent task and artifact data is valid', async () => {
  let clock = 0;
  let detailRequests = 0;
  let taskRequests = 0;
  const selections = {
    experiments: {},
    pipelines: {},
    pipelineVersions: {},
    recurringRuns: {},
    runs: {
      'run.training-2': {
        definition: RESOURCE_DEFINITIONS.runs[1],
        resource: { run_id: 'run-1' },
      },
    },
  };

  const semantic = await waitForSemanticBindings(
    selections,
    async (_method, endpoint) => {
      if (endpoint === '/apis/v2beta1/runs/run-1?view=FULL') {
        detailRequests++;
        return { runId: 'run-1', taskCount: taskRequests > 0 ? 1 : 0 };
      }
      assert.equal(endpoint, '/apis/v2beta1/runs/run-1/tasks?page_size=100');
      taskRequests++;
      if (taskRequests === 1) return { tasks: [] };
      return {
        tasks: [
          {
            name: 'write-metrics',
            outputs: {
              artifacts: [
                {
                  artifactKey: 'scalar_metrics',
                  artifacts: [
                    { artifactId: 'accuracy-1', name: 'accuracy', numberValue: 0.92 },
                    { artifactId: 'loss-1', name: 'loss', numberValue: 0.08 },
                  ],
                },
                {
                  artifactKey: 'roc_curve',
                  artifacts: [
                    {
                      artifactId: 'roc-1',
                      metadata: structuredClone(
                        METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
                      ),
                      name: 'roc_curve',
                    },
                  ],
                },
                {
                  artifactKey: 'html_report',
                  artifacts: [
                    {
                      artifactId: 'html-1',
                      name: 'html_report',
                      type: 'HTML',
                      uri: 's3://fixtures/report.html',
                    },
                  ],
                },
                {
                  artifactKey: 'markdown_report',
                  artifacts: [
                    {
                      artifactId: 'markdown-1',
                      name: 'markdown_report',
                      type: 'Markdown',
                      uri: 's3://fixtures/report.md',
                    },
                  ],
                },
              ],
            },
            taskId: 'task-1',
            type: 'RUNTIME',
          },
        ],
      };
    },
    {
      interval: 10,
      now: () => clock,
      sleep: async (milliseconds) => {
        clock += milliseconds;
      },
      timeout: 100,
    },
  );

  assert.equal(detailRequests, 2);
  assert.equal(taskRequests, 2);
  assert.equal(semantic.validation.valid, true);
});

test('attributes semantic discovery timeouts to missing fixtures or API incompatibility', async () => {
  const selections = {
    experiments: {},
    pipelines: {},
    pipelineVersions: {},
    recurringRuns: {},
    runs: {
      'run.training-2': {
        definition: RESOURCE_DEFINITIONS.runs[1],
        resource: { run_id: 'run-1' },
      },
    },
  };

  await assert.rejects(
    waitForSemanticBindings(selections, async () => ({ runId: 'run-1', taskCount: 0, tasks: [] }), {
      timeout: 0,
    }),
    (error) =>
      error.code === 'MISSING_FIXTURE' &&
      /expected 1 task\.write-metrics instance/.test(error.message),
  );

  await assert.rejects(
    waitForSemanticBindings(
      selections,
      async () => {
        throw new Error('run detail API rejected view=FULL');
      },
      { timeout: 0 },
    ),
    (error) =>
      error.code === 'API_INCOMPATIBILITY' && /run detail API rejected/.test(error.message),
  );
});
