const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const test = require('node:test');
const { parseDocument } = require('yaml');

const { buildLogicalFixtures, buildSemanticDeployment } = require('../semantic-manifest');
const {
  decodeGetArtifactsByIdResponse,
  decodeGetContextByTypeAndNameResponse,
  decodeGetEventsByExecutionIdsResponse,
  decodeGetExecutionsByContextResponse,
} = require('../mlmd-protobuf');

const {
  METRICS_EXECUTOR_OUTPUT,
  MINIMAL_PIPELINE_YAML,
  RICH_PIPELINE_YAML,
  RESOURCE_DEFINITIONS,
  SEED_FIXTURE_RUNTIME_REQUIREMENTS,
  SEED_IMAGE,
  SEMANTIC_MARKER,
  buildSeedManifest,
  fetchMlmdArtifactsByIds,
  fetchMlmdLineageByContext,
  fetchMlmdLineageForRun,
  fetchRunBindingResponse,
  fetchRunBindingResponses,
  resolveApiUrl,
  seedData,
  uploadPipeline,
  validateDetailRoutes,
  waitForSemanticBindings,
  waitForRunsStable,
} = require('../seed-data');

const MINIMAL_PIPELINE_SPEC = parseDocument(MINIMAL_PIPELINE_YAML).toJS();
const RICH_PIPELINE_SPEC = parseDocument(RICH_PIPELINE_YAML).toJS();

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

function nativeRichRunResponse(
  runId = 'run-created',
  pipelineId = 'pipeline-existing',
  pipelineVersionId = 'version-existing',
) {
  const rocMetadata = structuredClone(
    METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
  );
  const response = {
    displayName: RESOURCE_DEFINITIONS.runs[0].displayName,
    pipelineVersionReference: {
      pipelineId,
      pipelineVersionId,
    },
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
                {
                  artifactId: 'metric-loss',
                  name: 'loss',
                  numberValue: 0.08,
                  uri: 's3://fixtures/scalar-metrics/loss',
                },
                {
                  artifactId: 'metric-accuracy',
                  name: 'accuracy',
                  numberValue: 0.92,
                  uri: 's3://fixtures/scalar-metrics/accuracy',
                },
              ],
            },
            {
              artifactKey: 'roc_curve',
              artifacts: [
                {
                  artifactId: 'roc-artifact',
                  metadata: rocMetadata,
                  name: 'roc_curve',
                  uri: 's3://fixtures/roc-curve',
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
        childTasks: [{ name: 'nested-dag', taskId: 'task-nested' }],
        inputs: {
          artifacts: [
            {
              artifactKey: 'metrics',
              artifacts: [
                {
                  artifactId: 'metric-loss',
                  name: 'loss',
                  numberValue: 0.08,
                  uri: 's3://fixtures/scalar-metrics/loss',
                },
                {
                  artifactId: 'metric-accuracy',
                  name: 'accuracy',
                  numberValue: 0.92,
                  uri: 's3://fixtures/scalar-metrics/accuracy',
                },
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
        childTasks: [{ name: 'nested-dag', taskId: 'task-nested' }],
        name: 'retry-once',
        pods: [
          { name: 'retry-0', type: 'EXECUTOR', uid: 'retry-0-uid' },
          { name: 'retry-1', type: 'EXECUTOR', uid: 'retry-1-uid' },
        ],
        scopePath: 'root.retry-once',
        state: 'SUCCEEDED',
        taskId: 'task-retry',
        type: 'RUNTIME',
      },
      {
        childTasks: [{ name: 'nested-dag', taskId: 'task-nested' }],
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
  for (const task of response.tasks.filter((task) => task.type === 'RUNTIME')) {
    const attemptCount = task.name === 'retry-once' ? 2 : 1;
    task.outputs ||= {};
    task.outputs.artifacts ||= [];
    task.outputs.artifacts.push({
      artifactKey: 'executor-logs',
      artifacts: Array.from({ length: attemptCount }, (_, attemptIndex) => ({
        artifactId: `${task.taskId}-executor-log-${attemptIndex}`,
        name: 'executor-logs',
        type: 'Artifact',
        uri: `s3://fixtures/${task.taskId}/executor-logs-${attemptIndex}`,
      })),
    });
  }
  return response;
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
  assert.match(multipart, /printf 'metrics consumed\\n'/);
  assert.doesNotMatch(multipart, /test -f .*inputs\.artifacts\['metrics'\]\.path/);
  assert.equal((multipart.match(new RegExp(SEED_IMAGE, 'g')) || []).length, 5);
  assert.doesNotMatch(multipart, /pip install|kfp\.dsl\.executor_main|python:/);
});

test('rich retry fixture declares its required Argo failure predicate', () => {
  assert.deepEqual(SEED_FIXTURE_RUNTIME_REQUIREMENTS, {
    argoRetryPolicy: 'OnFailure',
  });
  assert.equal(Object.isFrozen(SEED_FIXTURE_RUNTIME_REQUIREMENTS), true);
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

test('hydrates complete legacy GetRun projections with full MLMD lineage', async () => {
  const lineageSelections = [];
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
              execution_id: '73',
              outputs: {
                html_report: { artifact_ids: ['83'] },
                markdown_report: { artifact_ids: ['84'] },
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
      fetchLegacyLineage: async (selection) => {
        lineageSelections.push(selection);
        return {
          artifacts: [
            { artifactId: '81', metadata: { accuracy: 0.92, loss: 0.08 } },
            {
              artifactId: '82',
              metadata: structuredClone(
                METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
              ),
            },
            {
              artifactId: '83',
              metadata: { display_name: 'html_report' },
              type: 'system.HTML',
              uri: 's3://fixtures/report.html',
            },
            {
              artifactId: '84',
              metadata: { display_name: 'markdown_report' },
              type: 'system.Markdown',
              uri: 's3://fixtures/report.md',
            },
            {
              artifactId: '85',
              metadata: { display_name: 'executor-logs' },
              type: 'system.Artifact',
              uri: 's3://fixtures/executor-logs-0',
            },
          ],
          events: [
            ['81', 'scalar_metrics'],
            ['82', 'roc_curve'],
            ['83', 'html_report'],
            ['84', 'markdown_report'],
            ['85', 'executor-logs'],
          ].map(([artifactId, key]) => ({
            artifactId,
            executionId: '73',
            path: [{ key }],
            type: 'OUTPUT',
          })),
          executions: [
            {
              executionId: '73',
              metadata: { display_name: 'write-metrics' },
              state: 'COMPLETE',
            },
          ],
        };
      },
    },
  );

  assert.deepEqual(lineageSelections, [{ expectedContextId: null, runId: 'legacy-run' }]);
  assert.deepEqual(response.semanticArtifacts[0].metadata, { accuracy: 0.92, loss: 0.08 });
  assert.equal(response.semanticArtifacts[1].metadata.confidenceMetrics.length, 5);
  assert.deepEqual(response.semanticExecutions[0].executorLogs, [
    {
      artifactId: '85',
      name: 'executor-logs',
      type: 'Artifact',
      uri: 's3://fixtures/executor-logs-0',
    },
  ]);
});

test('rejects executor-log Events whose MLMD Artifact identity is not exact', async () => {
  const cases = [
    {
      artifact: { metadata: { display_name: 'executor-logs' } },
      label: 'missing type',
      pattern: /has type undefined; expected "system\.Artifact"/,
    },
    {
      artifact: { metadata: { display_name: 'executor-logs' }, type: 'system.HTML' },
      label: 'wrong type',
      pattern: /has type "system\.HTML"; expected "system\.Artifact"/,
    },
    {
      artifact: { metadata: {}, type: 'system.Artifact' },
      label: 'missing display name',
      pattern: /metadata\.display_name undefined; expected "executor-logs"/,
    },
    {
      artifact: { metadata: { displayName: 'executor-logs' }, type: 'system.Artifact' },
      label: 'camel-case display name alias',
      pattern: /metadata\.display_name undefined; expected "executor-logs"/,
    },
  ];

  for (const { artifact, label, pattern } of cases) {
    await assert.rejects(
      fetchRunBindingResponse(
        'legacy-run',
        async () => ({
          run_details: {
            task_details: [
              { display_name: 'write-metrics', execution_id: '73', task_id: 'legacy-write' },
            ],
          },
          run_id: 'legacy-run',
        }),
        {
          fetchLegacyLineage: async () => ({
            artifacts: [
              {
                artifactId: '85',
                uri: 's3://fixtures/executor-logs-0',
                ...artifact,
              },
            ],
            events: [
              {
                artifactId: '85',
                executionId: '73',
                path: [{ key: 'executor-logs' }],
                type: 'OUTPUT',
              },
            ],
            executions: [
              {
                executionId: '73',
                metadata: { display_name: 'write-metrics' },
                state: 'COMPLETE',
              },
            ],
          }),
        },
      ),
      pattern,
      label,
    );
  }
});

test('hydrates real 2.17.1 task artifacts and execution IDs from MLMD Events', async () => {
  const lineageSelections = [];
  const response = await fetchRunBindingResponse(
    'legacy-run',
    async (_method, endpoint) => {
      assert.equal(endpoint, '/apis/v2beta1/runs/legacy-run?view=FULL');
      return {
        run_details: {
          task_details: [
            {
              display_name: 'write-metrics',
              task_id: 'legacy-write',
            },
            {
              display_name: 'consume-metrics',
              task_id: 'legacy-consume',
            },
          ],
        },
        run_id: 'legacy-run',
      };
    },
    {
      fetchLegacyLineage: async (selection) => {
        lineageSelections.push(selection);
        return {
          artifacts: [
            {
              artifactId: '81',
              metadata: { accuracy: 0.92, loss: 0.08 },
              uri: 's3://fixtures/scalar-metrics',
            },
            {
              artifactId: '82',
              metadata: structuredClone(
                METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
              ),
              uri: 's3://fixtures/roc-curve',
            },
            {
              artifactId: '83',
              metadata: { display_name: 'html_report' },
              type: 'system.HTML',
              uri: 's3://fixtures/report.html',
            },
            {
              artifactId: '84',
              metadata: { display_name: 'markdown_report' },
              type: 'system.Markdown',
              uri: 's3://fixtures/report.md',
            },
          ],
          events: [
            {
              artifactId: '81',
              executionId: '73',
              path: [{ key: 'scalar_metrics' }],
              type: 'OUTPUT',
            },
            { artifactId: '82', executionId: '73', path: [{ key: 'roc_curve' }], type: 'OUTPUT' },
            { artifactId: '83', executionId: '73', path: [{ key: 'html_report' }], type: 'OUTPUT' },
            {
              artifactId: '84',
              executionId: '73',
              path: [{ key: 'markdown_report' }],
              type: 'OUTPUT',
            },
            { artifactId: '81', executionId: '74', path: [{ key: 'metrics' }], type: 'INPUT' },
          ],
          executions: [
            {
              executionId: '73',
              metadata: { display_name: 'write-metrics', pod_name: 'write-pod' },
              state: 'COMPLETE',
            },
            {
              executionId: '74',
              metadata: { display_name: 'consume-metrics', pod_name: 'consume-pod' },
              state: 'COMPLETE',
            },
          ],
        };
      },
    },
  );

  assert.deepEqual(lineageSelections, [{ expectedContextId: null, runId: 'legacy-run' }]);
  const [write, consume] = response.run_details.task_details;
  assert.equal(write.execution_id, '73');
  assert.equal(consume.execution_id, '74');
  assert.deepEqual(write.outputs, {
    html_report: { artifact_ids: ['83'] },
    markdown_report: { artifact_ids: ['84'] },
    roc_curve: { artifact_ids: ['82'] },
    scalar_metrics: { artifact_ids: ['81'] },
  });
  assert.deepEqual(consume.inputs, { metrics: { artifact_ids: ['81'] } });
  assert.deepEqual(
    response.semanticArtifacts.map((artifact) => artifact.artifactId),
    ['81', '82', '83', '84'],
  );
});

test('builds a production-shaped rich legacy binding from MLMD execution lineage', async () => {
  const definition = RESOURCE_DEFINITIONS.runs.find(
    (candidate) => candidate.semanticKey === 'run.training-1',
  );
  const task = (displayName, taskId, children = []) => ({
    child_tasks: children.map((podName) => ({ pod_name: podName })),
    display_name: displayName,
    state: 'SUCCEEDED',
    task_id: taskId,
  });
  const response = await fetchRunBindingResponse(
    'legacy-run',
    async () => ({
      display_name: definition.displayName,
      run_details: {
        task_details: [
          task('write-metrics', 'task-uuid-write', ['argo-node-consume']),
          task('consume-metrics', 'task-uuid-consume', ['argo-node-nested']),
          task('nested-dag', 'task-uuid-nested', ['argo-node-nested-worker']),
          task('nested-worker', 'task-uuid-nested-worker'),
          task('retry-once', 'task-uuid-retry', ['argo-node-nested']),
          task('parallel-loop', 'task-uuid-parallel', [
            'argo-node-nested',
            'argo-node-loop-0',
            'argo-node-loop-1',
          ]),
          task('loop-worker', 'task-uuid-loop-1'),
          task('loop-worker', 'task-uuid-loop-0'),
        ],
      },
      run_id: 'legacy-run',
    }),
    {
      fetchLegacyLineage: async () => {
        const regularArtifacts = [
          {
            artifactId: '81',
            metadata: { accuracy: 0.92, loss: 0.08 },
            uri: 's3://fixtures/scalar-metrics',
          },
          {
            artifactId: '82',
            metadata: structuredClone(
              METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
            ),
            uri: 's3://fixtures/roc-curve',
          },
          {
            artifactId: '83',
            metadata: { display_name: 'html_report' },
            type: 'system.HTML',
            uri: 's3://fixtures/report.html',
          },
          {
            artifactId: '84',
            metadata: { display_name: 'markdown_report' },
            type: 'system.Markdown',
            uri: 's3://fixtures/report.md',
          },
        ];
        const logArtifacts = [
          ['91', 'write', 0],
          ['92', 'consume', 0],
          ['93', 'retry', 0],
          ['94', 'retry', 1],
          ['95', 'loop-0', 0],
          ['96', 'loop-1', 0],
          ['97', 'nested-worker', 0],
        ].map(([artifactId, owner, attempt]) => ({
          artifactId,
          metadata: { display_name: 'executor-logs' },
          type: 'system.Artifact',
          uri: `s3://fixtures/${owner}/executor-logs-${attempt}`,
        }));
        const executions = [
          {
            executionId: '70',
            metadata: {},
            name: 'run/legacy-run',
            state: 'COMPLETE',
            type: 'system.DAGExecution',
          },
          {
            executionId: '71',
            metadata: { display_name: 'nested-dag', parent_dag_id: 70 },
            state: 'COMPLETE',
            type: 'system.DAGExecution',
          },
          {
            executionId: '72',
            metadata: { display_name: 'parallel-loop', parent_dag_id: 70 },
            state: 'COMPLETE',
            type: 'system.DAGExecution',
          },
          {
            executionId: '80',
            metadata: {
              display_name: 'parallel-loop',
              iteration_index: 1,
              parent_dag_id: 72,
            },
            state: 'COMPLETE',
            type: 'system.DAGExecution',
          },
          {
            executionId: '79',
            metadata: {
              display_name: 'parallel-loop',
              iteration_index: 0,
              parent_dag_id: 72,
            },
            state: 'COMPLETE',
            type: 'system.DAGExecution',
          },
          {
            executionId: '73',
            metadata: { display_name: 'write-metrics', parent_dag_id: 70, pod_name: 'write-pod' },
            state: 'COMPLETE',
            type: 'system.ContainerExecution',
          },
          {
            executionId: '74',
            metadata: {
              display_name: 'consume-metrics',
              parent_dag_id: 70,
              pod_name: 'consume-pod',
            },
            state: 'COMPLETE',
            type: 'system.ContainerExecution',
          },
          {
            executionId: '75',
            metadata: { display_name: 'retry-once', parent_dag_id: 70 },
            state: 'COMPLETE',
            type: 'system.ContainerExecution',
          },
          {
            executionId: '77',
            metadata: { display_name: 'loop-worker', parent_dag_id: 80 },
            state: 'COMPLETE',
            type: 'system.ContainerExecution',
          },
          {
            executionId: '76',
            metadata: { display_name: 'loop-worker', parent_dag_id: 79 },
            state: 'COMPLETE',
            type: 'system.ContainerExecution',
          },
          {
            executionId: '78',
            metadata: { display_name: 'nested-worker', parent_dag_id: 71 },
            state: 'COMPLETE',
            type: 'system.ContainerExecution',
          },
        ];
        const outputEvents = [
          ['81', '73', 'scalar_metrics'],
          ['82', '73', 'roc_curve'],
          ['83', '73', 'html_report'],
          ['84', '73', 'markdown_report'],
          ['91', '73', 'executor-logs'],
          ['92', '74', 'executor-logs'],
          ['93', '75', 'executor-logs'],
          ['94', '75', 'executor-logs'],
          ['95', '76', 'executor-logs'],
          ['96', '77', 'executor-logs'],
          ['97', '78', 'executor-logs'],
        ].map(([artifactId, executionId, key]) => ({
          artifactId,
          executionId,
          path: [{ key }],
          type: 'OUTPUT',
        }));
        return {
          artifacts: [...regularArtifacts, ...logArtifacts],
          events: [
            ...outputEvents,
            {
              artifactId: '81',
              executionId: '74',
              path: [{ key: 'metrics' }],
              type: 'INPUT',
            },
          ],
          executions,
        };
      },
    },
  );

  const semantic = buildSemanticDeployment({
    logical: buildLogicalFixtures(RESOURCE_DEFINITIONS),
    runResponses: [
      {
        pipelineSpec: RICH_PIPELINE_SPEC,
        response,
        semanticKey: definition.semanticKey,
      },
    ],
  });
  assert.equal(semantic.validation.valid, true, semantic.validation.errors.join('; '));
  const binding = semantic.bindings.runs[definition.semanticKey];
  assert.deepEqual(
    Object.values(binding.executionInstances)
      .flat()
      .map((execution) => execution.executionId)
      .sort((left, right) => left.localeCompare(right, 'en', { numeric: true })),
    ['70', '71', '72', '73', '74', '75', '76', '77', '78', '79', '80'],
  );
  assert.deepEqual(
    binding.executionInstances['task.parallel-loop'].map((execution) => ({
      executionId: execution.executionId,
      executionRole: execution.executionRole,
      iterationIndex: execution.iterationIndex,
    })),
    [
      { executionId: '72', executionRole: 'loop-controller', iterationIndex: undefined },
      { executionId: '79', executionRole: 'loop-iteration', iterationIndex: 0 },
      { executionId: '80', executionRole: 'loop-iteration', iterationIndex: 1 },
    ],
  );
  assert.equal(binding.taskInstances['task.parallel-loop'][0].mlmdExecutionId, '72');
  assert.equal(
    binding.taskInstances['task.loop-worker'].every(
      (taskBinding) => !Object.hasOwn(taskBinding, 'mlmdExecutionId'),
    ),
    true,
  );
  assert.deepEqual(
    binding.executionInstances['task.loop-worker'].map((execution) => ({
      iterationIndex: execution.iterationIndex,
      iterationIndexEvidence: execution.iterationIndexEvidence,
    })),
    [
      { iterationIndex: 0, iterationIndexEvidence: 'mlmd-parent-dag' },
      { iterationIndex: 1, iterationIndexEvidence: 'mlmd-parent-dag' },
    ],
  );
  assert.deepEqual(
    binding.executionInstances['task.retry-once'][0].executorLogs.map((record) => record.uri),
    ['s3://fixtures/retry/executor-logs-0', 's3://fixtures/retry/executor-logs-1'],
  );
  assert.equal(binding.taskInstances['task.retry-once'][0].failedMainJobs.length, 0);
  assert.deepEqual(
    binding.relationships.map(({ kind, source, target }) => ({ kind, source, target })),
    structuredClone(require('../semantic-manifest').RUN_PROFILES['rich-topology'].relationships),
  );
});

test('builds a valid legacy semantic run from an empty GetRun artifact projection', async () => {
  const definition = RESOURCE_DEFINITIONS.runs.find(
    (candidate) => candidate.semanticKey === 'run.training-2',
  );
  const artifacts = [
    {
      artifactId: '81',
      metadata: { accuracy: 0.92, loss: 0.08 },
      uri: 's3://fixtures/scalar-metrics',
    },
    {
      artifactId: '82',
      metadata: structuredClone(METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata),
      uri: 's3://fixtures/roc-curve',
    },
    {
      artifactId: '83',
      metadata: { display_name: 'html_report' },
      type: 'system.HTML',
      uri: 's3://fixtures/report.html',
    },
    {
      artifactId: '84',
      metadata: { display_name: 'markdown_report' },
      type: 'system.Markdown',
      uri: 's3://fixtures/report.md',
    },
    {
      artifactId: '85',
      metadata: { display_name: 'executor-logs' },
      type: 'system.Artifact',
      uri: 's3://fixtures/executor-logs-0',
    },
  ];
  const events = [
    ['81', 'scalar_metrics'],
    ['82', 'roc_curve'],
    ['83', 'html_report'],
    ['84', 'markdown_report'],
  ].map(([artifactId, key]) => ({
    artifactId,
    executionId: '73',
    path: [{ key }],
    type: 'OUTPUT',
  }));
  events.push({
    artifactId: '85',
    executionId: '73',
    path: [{ key: 'executor-logs' }],
    type: 'OUTPUT',
  });
  const response = await fetchRunBindingResponse(
    'legacy-run',
    async () => ({
      display_name: definition.displayName,
      run_details: {
        task_details: [
          {
            display_name: 'write-metrics',
            state: 'SUCCEEDED',
            task_id: 'legacy-write',
          },
        ],
      },
      run_id: 'legacy-run',
    }),
    {
      fetchLegacyLineage: async () => ({
        artifacts,
        events,
        executions: [
          {
            executionId: '70',
            metadata: {},
            name: 'run/legacy-run',
            state: 'COMPLETE',
            type: 'system.DAGExecution',
          },
          {
            executionId: '73',
            metadata: { display_name: 'write-metrics', pod_name: 'write-pod' },
            state: 'COMPLETE',
          },
        ],
      }),
    },
  );
  const semantic = buildSemanticDeployment({
    logical: buildLogicalFixtures(RESOURCE_DEFINITIONS),
    runResponses: [{ response, semanticKey: definition.semanticKey }],
  });

  assert.equal(semantic.validation.valid, true, semantic.validation.errors.join('; '));
  const run = semantic.bindings.runs[definition.semanticKey];
  assert.equal(run.tasks['task.write-metrics'].mlmdExecutionId, '73');
  assert.deepEqual(
    run.tasks['task.write-metrics'].artifactReferences.outputs.map((group) => group.key),
    ['html_report', 'markdown_report', 'roc_curve', 'scalar_metrics'],
  );

  await assert.rejects(
    fetchRunBindingResponse(
      'legacy-run',
      async () => {
        const raw = structuredClone(response);
        delete raw.semanticArtifacts;
        delete raw.semanticExecutions;
        delete raw.run_details.task_details[0].execution_id;
        delete raw.run_details.task_details[0].outputs;
        return raw;
      },
      {
        fetchLegacyLineage: async () => ({
          artifacts,
          events: events.map((event) => ({ ...event, type: 'DECLARED_OUTPUT' })),
          executions: [
            {
              executionId: '73',
              metadata: { display_name: 'write-metrics', pod_name: 'write-pod' },
              state: 'COMPLETE',
            },
          ],
        }),
      },
    ),
    /event 0 is not a declared fixture or runtime executor-log event/,
  );
});

test('fails closed on ambiguous legacy execution mappings and incomplete context artifacts', async () => {
  const runResponse = {
    run_details: {
      pipeline_run_context_id: '17',
      task_details: [{ display_name: 'write-metrics', task_id: 'legacy-write' }],
    },
    run_id: 'legacy-run',
  };
  const executions = [73, 74].map((executionId) => ({
    executionId: String(executionId),
    metadata: { display_name: 'write-metrics' },
    state: 'COMPLETE',
  }));
  const getRun = async () => structuredClone(runResponse);

  let queriedLineageForWrongRun = false;
  const wrongRun = structuredClone(runResponse);
  wrongRun.run_id = 'different-run';
  await assert.rejects(
    fetchRunBindingResponse('legacy-run', async () => wrongRun, {
      fetchLegacyLineage: async () => {
        queriedLineageForWrongRun = true;
        return { artifacts: [], events: [], executions: [] };
      },
    }),
    /returned run "different-run" for requested run "legacy-run"/,
  );
  assert.equal(queriedLineageForWrongRun, false);

  await assert.rejects(
    fetchRunBindingResponse('legacy-run', getRun, {
      fetchLegacyLineage: async () => ({ artifacts: [], events: [], executions }),
    }),
    /ambiguously matches 2 MLMD executions/,
  );

  await assert.rejects(
    fetchRunBindingResponse('legacy-run', getRun, {
      fetchLegacyLineage: async () => ({
        artifacts: [],
        events: [
          {
            artifactId: '81',
            executionId: '73',
            path: [{ key: 'scalar_metrics' }],
            type: 'OUTPUT',
          },
        ],
        executions: [executions[0]],
      }),
    }),
    /event 0 references artifact 81 outside its run context/,
  );

  await assert.rejects(
    fetchRunBindingResponse('legacy-run', getRun, {
      fetchLegacyLineage: async () => ({
        artifacts: [{ artifactId: '81', metadata: { accuracy: 0.92, loss: 0.08 } }],
        events: [
          {
            artifactId: '81',
            executionId: '73',
            path: [{ key: 'scalar_metrics' }],
            type: 'OUTPUT',
          },
          {
            artifactId: '81',
            executionId: '73',
            path: [{ key: 'invented_alias' }],
            type: 'OUTPUT',
          },
        ],
        executions: [executions[0]],
      }),
    }),
    /event 1 is not a declared fixture or runtime executor-log event/,
  );

  const podMismatch = structuredClone(runResponse);
  podMismatch.run_details.task_details[0].pod_name = 'getrun-pod';
  await assert.rejects(
    fetchRunBindingResponse('legacy-run', async () => podMismatch, {
      fetchLegacyLineage: async () => ({
        artifacts: [],
        events: [],
        executions: [
          {
            executionId: '73',
            metadata: { display_name: 'write-metrics', pod_name: 'different-mlmd-pod' },
            state: 'COMPLETE',
          },
        ],
      }),
    }),
    /has no matching MLMD execution/,
  );

  const unexpectedGroup = structuredClone(runResponse);
  unexpectedGroup.run_details.task_details[0].outputs = {
    invented_alias: { artifact_ids: ['999'] },
  };
  await assert.rejects(
    fetchRunBindingResponse('legacy-run', async () => unexpectedGroup, {
      fetchLegacyLineage: async () => ({
        artifacts: [],
        events: [],
        executions: [executions[0]],
      }),
    }),
    /unexpected outputs artifact group "invented_alias"/,
  );
});

test('does not guess execution IDs for repeated non-artifact legacy tasks', async () => {
  const response = await fetchRunBindingResponse(
    'legacy-run',
    async () => ({
      run_details: {
        task_details: [
          { display_name: 'write-metrics', task_id: 'legacy-write' },
          { display_name: 'loop-worker', task_id: 'legacy-loop-0' },
          { display_name: 'loop-worker', task_id: 'legacy-loop-1' },
        ],
      },
      run_id: 'legacy-run',
    }),
    {
      fetchLegacyLineage: async () => ({
        artifacts: [
          { artifactId: '81', metadata: { accuracy: 0.92, loss: 0.08 } },
          {
            artifactId: '82',
            metadata: structuredClone(
              METRICS_EXECUTOR_OUTPUT.artifacts.roc_curve.artifacts[0].metadata,
            ),
          },
          { artifactId: '83', metadata: {}, type: 'system.HTML', uri: 's3://report.html' },
          {
            artifactId: '84',
            metadata: {},
            type: 'system.Markdown',
            uri: 's3://report.md',
          },
        ],
        events: [
          ['81', 'scalar_metrics'],
          ['82', 'roc_curve'],
          ['83', 'html_report'],
          ['84', 'markdown_report'],
        ].map(([artifactId, key]) => ({
          artifactId,
          executionId: '73',
          path: [{ key }],
          type: 'OUTPUT',
        })),
        executions: [
          { executionId: '73', metadata: { display_name: 'write-metrics' }, state: 'COMPLETE' },
          { executionId: '74', metadata: { display_name: 'loop-worker' }, state: 'COMPLETE' },
          { executionId: '75', metadata: { display_name: 'loop-worker' }, state: 'FAILED' },
        ],
      }),
    },
  );

  assert.equal(response.run_details.task_details[0].execution_id, '73');
  assert.equal(response.run_details.task_details[1].execution_id, undefined);
  assert.equal(response.run_details.task_details[2].execution_id, undefined);
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

function testMlmdExecution(executionId, metadata, details = {}) {
  return Buffer.concat([
    testField(1, 0, testVarint(executionId)),
    testField(3, 0, testVarint(details.state || 3)),
    ...Object.entries(metadata).map(([key, value]) =>
      testMessageField(
        5,
        Buffer.concat([testStringField(1, key), testMessageField(2, testStringField(3, value))]),
      ),
    ),
    ...(details.name ? [testStringField(6, details.name)] : []),
    ...(details.type ? [testStringField(7, details.type)] : []),
  ]);
}

function testMlmdContext(contextId, name, type) {
  return Buffer.concat([
    testField(1, 0, testVarint(contextId)),
    testStringField(3, name),
    testStringField(6, type),
  ]);
}

function testMlmdEvent(artifactId, executionId, key, type) {
  const path = testMessageField(1, testStringField(2, key));
  return Buffer.concat([
    testField(1, 0, testVarint(artifactId)),
    testField(2, 0, testVarint(executionId)),
    testMessageField(3, path),
    testField(4, 0, testVarint(type)),
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

test('rejects incomplete gRPC-web responses and MLMD records without positive IDs', async () => {
  const artifactResponse = testMlmdResponse([{ artifactId: 81, metadata: {} }]);
  await assert.rejects(
    fetchMlmdArtifactsByIds(['81'], async () => grpcFrame(0x00, artifactResponse)),
    /missing its terminal grpc-status trailer/,
  );
  await assert.rejects(
    fetchMlmdArtifactsByIds(['81'], async () =>
      Buffer.concat([
        grpcFrame(0x00, artifactResponse),
        grpcFrame(0x80, Buffer.from('grpc-status: 0\r\n')),
        grpcFrame(0x00, artifactResponse),
      ]),
    ),
    /frame after its terminal trailer/,
  );

  assert.throws(
    () => decodeGetArtifactsByIdResponse(testMessageField(1, Buffer.alloc(0))),
    /Artifact is missing its required positive ID/,
  );
  assert.throws(
    () => decodeGetExecutionsByContextResponse(testMessageField(1, Buffer.alloc(0))),
    /Execution is missing its required positive ID/,
  );
  assert.throws(
    () => decodeGetEventsByExecutionIdsResponse(testMessageField(1, Buffer.alloc(0))),
    /Event is missing a required positive artifact or execution ID/,
  );
  assert.throws(
    () => decodeGetContextByTypeAndNameResponse(testMessageField(1, Buffer.alloc(0))),
    /Context is missing its required positive ID/,
  );
  assert.throws(
    () => decodeGetArtifactsByIdResponse(testMlmdResponse([{ artifactId: 0, metadata: {} }])),
    /MLMD artifact ID must be a positive int64/,
  );
});

test('queries and decodes complete MLMD context lineage without generated protobufs', async () => {
  const contextResponse = testMessageField(
    1,
    testMlmdContext(17, 'legacy-run', 'system.PipelineRun'),
  );
  const executionResponse = Buffer.concat([
    testMessageField(
      1,
      testMlmdExecution(
        73,
        { display_name: 'write-metrics', pod_name: 'write-pod' },
        {
          type: 'system.ContainerExecution',
        },
      ),
    ),
    testMessageField(
      1,
      testMlmdExecution(74, { display_name: 'consume-metrics', pod_name: 'consume-pod' }),
    ),
  ]);
  const artifactResponse = testMlmdResponse([
    {
      artifactId: 81,
      metadata: { accuracy: 0.92, loss: 0.08 },
      uri: 's3://fixtures/scalar-metrics',
    },
  ]);
  const eventResponse = Buffer.concat([
    testMessageField(1, testMlmdEvent(81, 73, 'scalar_metrics', 4)),
    testMessageField(1, testMlmdEvent(81, 74, 'metrics', 3)),
  ]);
  const calls = [];
  const lineage = await fetchMlmdLineageForRun(
    'legacy-run',
    async (method, endpoint, body, options) => {
      assert.equal(method, 'POST');
      assert.equal(body, null);
      assert.equal(options.responseType, 'buffer');
      const payloadLength = options.rawBody.readUInt32BE(1);
      const payload = options.rawBody.subarray(5, 5 + payloadLength);
      calls.push({ endpoint, payload: payload.toString('hex') });
      const message = {
        '/ml_metadata.MetadataStoreService/GetArtifactsByContext': artifactResponse,
        '/ml_metadata.MetadataStoreService/GetContextByTypeAndName': contextResponse,
        '/ml_metadata.MetadataStoreService/GetEventsByExecutionIDs': eventResponse,
        '/ml_metadata.MetadataStoreService/GetExecutionsByContext': executionResponse,
      }[endpoint];
      assert.ok(message, `Unexpected MLMD endpoint: ${endpoint}`);
      return Buffer.concat([
        grpcFrame(0x00, message),
        grpcFrame(0x80, Buffer.from('grpc-status: 0\r\n')),
      ]);
    },
    { expectedContextId: '17' },
  );

  assert.deepEqual(
    calls.sort((left, right) => left.endpoint.localeCompare(right.endpoint)),
    [
      {
        endpoint: '/ml_metadata.MetadataStoreService/GetArtifactsByContext',
        payload: '0811',
      },
      {
        endpoint: '/ml_metadata.MetadataStoreService/GetContextByTypeAndName',
        payload: '0a1273797374656d2e506970656c696e6552756e120a6c65676163792d72756e',
      },
      {
        endpoint: '/ml_metadata.MetadataStoreService/GetEventsByExecutionIDs',
        payload: '0849084a',
      },
      {
        endpoint: '/ml_metadata.MetadataStoreService/GetExecutionsByContext',
        payload: '0811',
      },
    ],
  );
  assert.deepEqual(lineage.executions, [
    {
      executionId: '73',
      metadata: { display_name: 'write-metrics', pod_name: 'write-pod' },
      state: 'COMPLETE',
      type: 'system.ContainerExecution',
    },
    {
      executionId: '74',
      metadata: { display_name: 'consume-metrics', pod_name: 'consume-pod' },
      state: 'COMPLETE',
    },
  ]);
  assert.deepEqual(lineage.artifacts, [
    {
      artifactId: '81',
      metadata: { accuracy: 0.92, loss: 0.08 },
      uri: 's3://fixtures/scalar-metrics',
    },
  ]);
  assert.deepEqual(lineage.events, [
    {
      artifactId: '81',
      executionId: '73',
      path: [{ key: 'scalar_metrics' }],
      type: 'OUTPUT',
    },
    {
      artifactId: '81',
      executionId: '74',
      path: [{ key: 'metrics' }],
      type: 'INPUT',
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

test('binds run observations to the selected pipeline spec and rejects missing provenance', async () => {
  const pipelineSemanticKey = 'pipeline.data-ingestion';
  const selections = {
    pipelines: {
      [pipelineSemanticKey]: {
        definition: RESOURCE_DEFINITIONS.pipelines[1],
        resource: { pipeline_id: 'pipeline-1' },
      },
    },
    pipelineVersions: {
      [pipelineSemanticKey]: {
        definition: RESOURCE_DEFINITIONS.pipelines[1],
        resource: {
          pipeline_id: 'pipeline-1',
          pipeline_spec: structuredClone(MINIMAL_PIPELINE_SPEC),
          pipeline_version_id: 'version-1',
        },
      },
    },
    runs: {
      'run.training-2': {
        definition: RESOURCE_DEFINITIONS.runs[1],
        resource: { run_id: 'run-1' },
      },
    },
  };
  const runResponse = {
    pipeline_version_reference: {
      pipeline_id: 'pipeline-1',
      pipeline_version_id: 'version-1',
    },
    run_id: 'run-1',
    task_count: 1,
    tasks: [{ name: 'write-metrics', task_id: 'task-1', type: 'RUNTIME' }],
  };
  const request = async (_method, endpoint) => {
    assert.equal(endpoint, '/apis/v2beta1/runs/run-1?view=FULL');
    return structuredClone(runResponse);
  };

  const observations = await fetchRunBindingResponses(selections, request);
  assert.deepEqual(observations, [
    {
      pipelineSpec: MINIMAL_PIPELINE_SPEC,
      response: runResponse,
      semanticKey: 'run.training-2',
    },
  ]);

  for (const [name, mutate, pattern] of [
    [
      'missing pipeline spec',
      ({ selected }) =>
        delete selected.pipelineVersions[pipelineSemanticKey].resource.pipeline_spec,
      /missing pipeline_spec/,
    ],
    [
      'malformed pipeline spec',
      ({ selected }) => {
        selected.pipelineVersions[pipelineSemanticKey].resource.pipeline_spec = [];
      },
      /missing pipeline_spec/,
    ],
    [
      'missing run reference',
      ({ response }) => delete response.pipeline_version_reference,
      /does not reference selected pipeline version pipeline-1\/version-1/,
    ],
    [
      'mismatched pipeline reference',
      ({ response }) => {
        response.pipeline_version_reference.pipeline_id = 'other-pipeline';
      },
      /does not reference selected pipeline version pipeline-1\/version-1/,
    ],
    [
      'mismatched version reference',
      ({ response }) => {
        response.pipeline_version_reference.pipeline_version_id = 'other-version';
      },
      /does not reference selected pipeline version pipeline-1\/version-1/,
    ],
  ]) {
    const selected = structuredClone(selections);
    const response = structuredClone(runResponse);
    mutate({ response, selected });
    await assert.rejects(
      fetchRunBindingResponses(selected, async () => response),
      pattern,
      name,
    );
  }
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
          {
            pipeline_id: 'pipeline-existing',
            pipeline_spec: structuredClone(RICH_PIPELINE_SPEC),
            pipeline_version_id: 'version-existing',
          },
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

  const waitCalls = [];
  const result = await seedData({
    pipelines: 1,
    experiments: 1,
    runs: 1,
    recurringRuns: 1,
    request,
    manifestPath,
    apiBase: 'http://seed.test',
    semanticTimeout: 0,
    waitForCreatedRuns: true,
    waitForRunsFn: async (runIds) => {
      waitCalls.push([...runIds]);
      return true;
    },
  });

  assert.equal(result.success, true, result.error);
  assert.equal(result.skipped, false);
  assert.deepEqual(result.resources.pipelineIds, ['pipeline-existing']);
  assert.deepEqual(result.resources.experimentIds, ['experiment-created']);
  assert.deepEqual(result.resources.runIds, ['run-created']);
  assert.deepEqual(result.resources.recurringRunIds, ['recurring-created']);
  assert.deepEqual(waitCalls, [['run-created'], ['run-created']]);
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
    pipelines: {
      'pipeline.data-ingestion': {
        definition: RESOURCE_DEFINITIONS.pipelines[1],
        resource: { pipeline_id: 'pipeline-1' },
      },
    },
    pipelineVersions: {
      'pipeline.data-ingestion': {
        definition: RESOURCE_DEFINITIONS.pipelines[1],
        resource: {
          pipeline_id: 'pipeline-1',
          pipeline_spec: structuredClone(MINIMAL_PIPELINE_SPEC),
          pipeline_version_id: 'version-1',
        },
      },
    },
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
        return {
          pipelineVersionReference: {
            pipelineId: 'pipeline-1',
            pipelineVersionId: 'version-1',
          },
          runId: 'run-1',
          taskCount: taskRequests > 0 ? 1 : 0,
        };
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
                    {
                      artifactId: 'accuracy-1',
                      name: 'accuracy',
                      numberValue: 0.92,
                      uri: 's3://fixtures/scalar-metrics/accuracy',
                    },
                    {
                      artifactId: 'loss-1',
                      name: 'loss',
                      numberValue: 0.08,
                      uri: 's3://fixtures/scalar-metrics/loss',
                    },
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
                      uri: 's3://fixtures/roc-curve',
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
                {
                  artifactKey: 'executor-logs',
                  artifacts: [
                    {
                      artifactId: 'task-1-executor-log-0',
                      name: 'executor-logs',
                      type: 'Artifact',
                      uri: 's3://fixtures/task-1/executor-logs-0',
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
    pipelines: {
      'pipeline.data-ingestion': {
        definition: RESOURCE_DEFINITIONS.pipelines[1],
        resource: { pipeline_id: 'pipeline-1' },
      },
    },
    pipelineVersions: {
      'pipeline.data-ingestion': {
        definition: RESOURCE_DEFINITIONS.pipelines[1],
        resource: {
          pipeline_id: 'pipeline-1',
          pipeline_spec: structuredClone(MINIMAL_PIPELINE_SPEC),
          pipeline_version_id: 'version-1',
        },
      },
    },
    recurringRuns: {},
    runs: {
      'run.training-2': {
        definition: RESOURCE_DEFINITIONS.runs[1],
        resource: { run_id: 'run-1' },
      },
    },
  };

  await assert.rejects(
    waitForSemanticBindings(
      selections,
      async () => ({
        pipelineVersionReference: {
          pipelineId: 'pipeline-1',
          pipelineVersionId: 'version-1',
        },
        runId: 'run-1',
        taskCount: 0,
        tasks: [],
      }),
      { timeout: 0 },
    ),
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
