const assert = require('node:assert/strict');
const test = require('node:test');

const {
  ARTIFACT_FIXTURES,
  REVISION_FLAVORS,
  SEMANTIC_FIXTURE_SET,
  SEMANTIC_SCHEMA_VERSION,
  buildLogicalFixtures,
  buildSemanticDeployment,
  combineSemanticManifests,
  detectRevisionFlavor,
  extractRunBinding,
} = require('../semantic-manifest');

const EXPECTED_ROC_POINTS = structuredClone(ARTIFACT_FIXTURES['artifact.roc-curve'].points);

function legacyMetricArtifacts() {
  return [
    {
      artifactId: '81',
      metadata: { accuracy: 0.92, loss: 0.08 },
    },
    {
      artifactId: '82',
      metadata: { confidenceMetrics: structuredClone(EXPECTED_ROC_POINTS) },
    },
  ];
}

function nativeRocArtifact(artifactId = 'roc-artifact') {
  return {
    artifact_id: artifactId,
    metadata: { confidenceMetrics: structuredClone(EXPECTED_ROC_POINTS) },
    name: 'roc_curve',
  };
}

const RESOURCE_DEFINITIONS = {
  runs: [{ displayName: 'Visual Run', semanticKey: 'run.visuals' }],
};
const RICH_RESOURCE_DEFINITIONS = {
  runs: [
    {
      displayName: 'Rich Visual Run',
      fixtureProfile: 'rich-topology',
      pipelineSemanticKey: 'pipeline.rich',
      semanticKey: 'run.rich',
    },
  ],
};

function legacyRichRun() {
  return {
    display_name: 'Rich Visual Run',
    run_details: {
      pipeline_context_id: '11',
      task_details: [
        {
          child_tasks: [{ task_id: 'legacy-consume' }],
          display_name: 'write-metrics',
          execution_id: '1',
          outputs: {
            roc_curve: { artifact_ids: [82] },
            scalar_metrics: { artifact_ids: [81] },
          },
          state: 'SUCCEEDED',
          task_id: 'legacy-write',
        },
        {
          child_tasks: [{ task_id: 'legacy-nested' }],
          display_name: 'consume-metrics',
          execution_id: '2',
          inputs: { metrics: { artifact_ids: [81] } },
          state: 'SUCCEEDED',
          task_id: 'legacy-consume',
        },
        {
          child_tasks: [{ task_id: 'legacy-nested' }],
          display_name: 'retry-once',
          execution_id: '3',
          executor_detail: {
            failed_main_jobs: ['retry-attempt-0'],
            main_job: 'retry-attempt-1',
          },
          state: 'SUCCEEDED',
          task_id: 'legacy-retry',
        },
        {
          child_tasks: [{ task_id: 'legacy-nested' }],
          display_name: 'parallel-loop',
          execution_id: '4',
          state: 'SUCCEEDED',
          task_id: 'legacy-loop',
        },
        {
          display_name: 'loop-worker',
          execution_id: '5',
          parent_task_id: 'legacy-loop',
          state: 'SUCCEEDED',
          task_id: 'legacy-loop-0',
        },
        {
          display_name: 'loop-worker',
          execution_id: '6',
          parent_task_id: 'legacy-loop',
          state: 'SUCCEEDED',
          task_id: 'legacy-loop-1',
        },
        {
          display_name: 'nested-dag',
          execution_id: '7',
          state: 'SUCCEEDED',
          task_id: 'legacy-nested',
        },
        {
          display_name: 'nested-worker',
          execution_id: '8',
          parent_task_id: 'legacy-nested',
          state: 'SUCCEEDED',
          task_id: 'legacy-nested-worker',
        },
      ],
    },
    run_id: 'legacy-rich-run',
    semanticArtifacts: legacyMetricArtifacts(),
  };
}

function nativeRichRun() {
  return {
    display_name: 'Rich Visual Run',
    run_id: 'native-rich-run',
    task_count: 10,
    tasks: [
      {
        child_tasks: [{ name: 'consume-metrics', task_id: 'native-consume' }],
        name: 'write-metrics',
        outputs: {
          artifacts: [
            {
              artifact_key: 'scalar_metrics',
              artifacts: [
                { artifact_id: 'metric-accuracy', name: 'accuracy', number_value: 0.92 },
                { artifact_id: 'metric-loss', name: 'loss', number_value: 0.08 },
              ],
            },
            {
              artifact_key: 'roc_curve',
              artifacts: [nativeRocArtifact()],
            },
          ],
        },
        scope_path: 'root.write-metrics',
        state: 'SUCCEEDED',
        task_id: 'native-write',
        type: 'RUNTIME',
      },
      {
        child_tasks: [{ name: 'nested-dag', task_id: 'native-nested' }],
        inputs: {
          artifacts: [
            {
              artifact_key: 'metrics',
              artifacts: [
                { artifact_id: 'metric-accuracy', name: 'accuracy', number_value: 0.92 },
                { artifact_id: 'metric-loss', name: 'loss', number_value: 0.08 },
              ],
            },
          ],
        },
        name: 'consume-metrics',
        scope_path: 'root.consume-metrics',
        state: 'SUCCEEDED',
        task_id: 'native-consume',
        type: 'RUNTIME',
      },
      {
        child_tasks: [{ name: 'nested-dag', task_id: 'native-nested' }],
        name: 'retry-once',
        pods: [
          { name: 'retry-attempt-0', type: 'EXECUTOR' },
          { name: 'retry-attempt-1', type: 'EXECUTOR' },
        ],
        scope_path: 'root.retry-once',
        state: 'SUCCEEDED',
        task_id: 'native-retry',
        type: 'RUNTIME',
      },
      {
        child_tasks: [{ name: 'nested-dag', task_id: 'native-nested' }],
        name: 'parallel-loop',
        scope_path: 'root.parallel-loop',
        state: 'SUCCEEDED',
        task_id: 'native-loop',
        type: 'LOOP',
        type_attributes: { iteration_count: 2 },
      },
      {
        display_name: 'parallel-loop',
        name: 'parallel-loop-0',
        parent_task_id: 'native-loop',
        scope_path: 'root.parallel-loop.parallel-loop-0',
        state: 'SUCCEEDED',
        task_id: 'native-loop-scope-0',
        type: 'DAG',
        type_attributes: { iteration_index: 0 },
      },
      {
        display_name: 'parallel-loop',
        name: 'parallel-loop-1',
        parent_task_id: 'native-loop',
        scope_path: 'root.parallel-loop.parallel-loop-1',
        state: 'SUCCEEDED',
        task_id: 'native-loop-scope-1',
        type: 'DAG',
        type_attributes: { iteration_index: 1 },
      },
      {
        name: 'loop-worker',
        parent_task_id: 'native-loop-scope-0',
        scope_path: 'root.parallel-loop.loop-worker',
        state: 'SUCCEEDED',
        task_id: 'native-loop-worker-0',
        type: 'RUNTIME',
        type_attributes: { iteration_index: 0 },
      },
      {
        name: 'loop-worker',
        parent_task_id: 'native-loop-scope-1',
        scope_path: 'root.parallel-loop.loop-worker',
        state: 'SUCCEEDED',
        task_id: 'native-loop-worker-1',
        type: 'RUNTIME',
        type_attributes: { iteration_index: 1 },
      },
      {
        name: 'nested-dag',
        scope_path: 'root.nested-dag',
        state: 'SUCCEEDED',
        task_id: 'native-nested',
        type: 'DAG',
      },
      {
        name: 'nested-worker',
        parent_task_id: 'native-nested',
        scope_path: 'root.nested-dag.nested-worker',
        state: 'SUCCEEDED',
        task_id: 'native-nested-worker',
        type: 'RUNTIME',
      },
    ],
  };
}

test('extracts legacy MLMD task and grouped metric bindings from snake/camel detail fields', () => {
  const response = {
    run: {
      display_name: 'Visual Run',
      run_details: {
        pipeline_context_id: '11',
        task_details: [
          {
            displayName: 'write-metrics',
            execution_id: '73',
            outputs: {
              roc_curve: { artifact_ids: [82] },
              scalarMetrics: { artifactIds: [81] },
            },
            state: 'SUCCEEDED',
            task_id: 'legacy-task',
          },
        ],
      },
      run_id: 'legacy-run',
    },
    semanticArtifacts: legacyMetricArtifacts(),
  };

  assert.equal(detectRevisionFlavor(response), REVISION_FLAVORS.LEGACY);
  const binding = extractRunBinding(response, 'run.visuals');
  assert.equal(binding.runId, 'legacy-run');
  assert.equal(binding.tasks['task.write-metrics'].mlmdExecutionId, '73');
  assert.deepEqual(binding.artifacts['artifact.scalar-metrics'].artifactIds, ['81']);
  assert.deepEqual(
    binding.artifacts['artifact.scalar-metrics'].members['metric.accuracy'].artifactIds,
    ['81'],
  );
  assert.deepEqual(binding.artifacts['artifact.roc-curve'].artifactIds, ['82']);
  assert.deepEqual(binding.artifacts['artifact.roc-curve'].points, EXPECTED_ROC_POINTS);
});

test('keeps native scalar metrics as one logical group with semantic member keys', () => {
  const response = {
    displayName: 'Visual Run',
    runId: 'native-run',
    taskCount: 1,
    tasks: [
      {
        display_name: 'Write Metrics',
        outputs: {
          artifacts: [
            {
              artifact_key: 'scalar_metrics',
              artifacts: [
                { artifact_id: 'z-loss', name: 'loss', number_value: 0.08 },
                { artifactId: 'a-accuracy', name: 'accuracy', numberValue: 0.92 },
              ],
            },
            {
              artifactKey: 'roc_curve',
              artifacts: [
                {
                  artifactId: 'roc-1',
                  metadata: { confidenceMetrics: structuredClone(EXPECTED_ROC_POINTS) },
                  name: 'roc',
                },
              ],
            },
          ],
        },
        scope_path: 'root.write-metrics',
        state: 'SUCCEEDED',
        taskId: 'native-task',
        type: 'RUNTIME',
      },
    ],
  };

  assert.equal(detectRevisionFlavor(response), REVISION_FLAVORS.NATIVE);
  const binding = extractRunBinding(response, 'run.visuals');
  const metricGroup = binding.artifacts['artifact.scalar-metrics'];
  assert.equal(metricGroup.storage, REVISION_FLAVORS.NATIVE);
  assert.deepEqual(metricGroup.artifactIds, ['a-accuracy', 'z-loss']);
  assert.deepEqual(Object.keys(metricGroup.members), ['metric.accuracy', 'metric.loss']);
  assert.deepEqual(metricGroup.members['metric.accuracy'].artifactIds, ['a-accuracy']);
  assert.deepEqual(metricGroup.members['metric.loss'].artifactIds, ['z-loss']);
  assert.deepEqual(binding.artifacts['artifact.roc-curve'].artifactIds, ['roc-1']);
  assert.deepEqual(binding.artifacts['artifact.roc-curve'].points, EXPECTED_ROC_POINTS);
});

test('does not infer legacy scalar metric members from a shared MLMD artifact ID', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const response = legacyRichRun();
  delete response.semanticArtifacts;

  const semantic = buildSemanticDeployment({
    logical,
    runResponses: [{ response, semanticKey: 'run.rich' }],
  });

  assert.equal(semantic.validation.valid, false);
  assert.deepEqual(
    semantic.bindings.runs['run.rich'].artifacts['artifact.scalar-metrics'].members,
    {},
  );
  assert.match(semantic.validation.errors.join('\n'), /missing metric\.accuracy/);
  assert.match(semantic.validation.errors.join('\n'), /missing metric\.loss/);
});

test('rejects wrong and missing legacy scalar metric values read from MLMD', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const wrong = legacyRichRun();
  wrong.semanticArtifacts.find((artifact) => artifact.artifactId === '81').metadata.accuracy = 0.91;
  const missing = legacyRichRun();
  delete missing.semanticArtifacts.find((artifact) => artifact.artifactId === '81').metadata.loss;

  const wrongSemantic = buildSemanticDeployment({
    logical,
    runResponses: [{ response: wrong, semanticKey: 'run.rich' }],
  });
  const missingSemantic = buildSemanticDeployment({
    logical,
    runResponses: [{ response: missing, semanticKey: 'run.rich' }],
  });

  assert.equal(wrongSemantic.validation.valid, false);
  assert.match(wrongSemantic.validation.errors.join('\n'), /metric\.accuracy value 0\.91/);
  assert.equal(missingSemantic.validation.valid, false);
  assert.match(missingSemantic.validation.errors.join('\n'), /missing metric\.loss/);
});

test('rejects wrong or missing deterministic ROC payloads in either revision', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  for (const [flavor, response, findArtifact] of [
    [
      'legacy',
      legacyRichRun(),
      (run) => run.semanticArtifacts.find((artifact) => artifact.artifactId === '82'),
    ],
    [
      'native',
      nativeRichRun(),
      (run) =>
        run.tasks
          .find((task) => task.name === 'write-metrics')
          .outputs.artifacts.find((group) => group.artifact_key === 'roc_curve').artifacts[0],
    ],
  ]) {
    const wrong = structuredClone(response);
    findArtifact(wrong).metadata.confidenceMetrics[1].recall = 0.36;
    const missing = structuredClone(response);
    delete findArtifact(missing).metadata.confidenceMetrics;

    for (const [condition, candidate] of [
      ['wrong', wrong],
      ['missing', missing],
    ]) {
      const semantic = buildSemanticDeployment({
        logical,
        runResponses: [{ response: candidate, semanticKey: 'run.rich' }],
      });
      assert.equal(semantic.validation.valid, false, `${flavor} ${condition} ROC payload`);
      assert.match(
        semantic.validation.errors.join('\n'),
        /artifact\.roc-curve points .* did not match/,
        `${flavor} ${condition} ROC payload`,
      );
    }
  }
});

test('maps rich legacy and native topology to instance groups and semantic relationships', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const legacy = buildSemanticDeployment({
    logical,
    runResponses: [{ response: legacyRichRun(), semanticKey: 'run.rich' }],
  });
  const native = buildSemanticDeployment({
    logical,
    runResponses: [{ response: nativeRichRun(), semanticKey: 'run.rich' }],
  });

  assert.equal(legacy.validation.valid, true, legacy.validation.errors.join('; '));
  assert.equal(native.validation.valid, true, native.validation.errors.join('; '));
  assert.equal(legacy.bindings.runs['run.rich'].taskInstances['task.loop-worker'].length, 2);
  assert.deepEqual(
    native.bindings.runs['run.rich'].taskInstances['task.loop-worker'].map(
      (instance) => instance.iterationIndex,
    ),
    [0, 1],
  );
  assert.equal(legacy.bindings.runs['run.rich'].tasks['task.retry-once'].failedMainJobs.length, 1);
  assert.equal(native.bindings.runs['run.rich'].tasks['task.retry-once'].executorPods.length, 2);
  assert.ok(
    native.bindings.runs['run.rich'].relationships.some(
      (relationship) =>
        relationship.kind === 'artifact-consumer' &&
        relationship.source === 'artifact.scalar-metrics' &&
        relationship.target === 'task.consume-metrics',
    ),
  );
  assert.equal(
    native.bindings.runs['run.rich'].relationships.find(
      (relationship) =>
        relationship.kind === 'contains' && relationship.source === 'task.parallel-loop',
    ).occurrences,
    2,
  );
});

test('rejects rich topology with collapsed loops, missing retry evidence, or broken artifact links', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const response = nativeRichRun();
  response.tasks = response.tasks.filter((task) => task.task_id !== 'native-loop-worker-1');
  response.tasks.find((task) => task.name === 'retry-once').pods = [
    { name: 'retry-attempt-0', type: 'EXECUTOR' },
  ];
  response.tasks.find((task) => task.name === 'consume-metrics').inputs.artifacts[0].artifacts = [
    { artifact_id: 'different-artifact', name: 'accuracy', number_value: 0.92 },
  ];

  const semantic = buildSemanticDeployment({
    logical,
    runResponses: [{ response, semanticKey: 'run.rich' }],
  });

  assert.equal(semantic.validation.valid, false);
  assert.match(semantic.validation.errors.join('\n'), /expected 2 task\.loop-worker/);
  assert.match(semantic.validation.errors.join('\n'), /recorded 1 attempt/);
  assert.match(
    semantic.validation.errors.join('\n'),
    /producer and consumer artifact bindings differ/,
  );
});

test('builds and combines revision deployments by semantic key with provenance and capture projection', () => {
  const logical = buildLogicalFixtures(RESOURCE_DEFINITIONS);
  const baseSemantic = buildSemanticDeployment({
    logical,
    resourceBindings: {
      'run.visuals': { id: 'base-generated-run', kind: 'run' },
    },
    runResponses: [
      {
        response: {
          run_details: {
            task_details: [
              {
                display_name: 'write-metrics',
                execution_id: 7,
                outputs: { scalar_metrics: { artifact_ids: [9] } },
                task_id: 'base-task',
              },
            ],
          },
          run_id: 'base-generated-run',
        },
        semanticKey: 'run.visuals',
      },
    ],
  });
  const headSemantic = buildSemanticDeployment({
    logical,
    resourceBindings: {
      'run.visuals': { id: 'head-generated-run', kind: 'run' },
    },
    runResponses: [
      {
        response: {
          runId: 'head-generated-run',
          taskCount: 1,
          tasks: [{ name: 'write-metrics', taskId: 'head-task' }],
        },
        semanticKey: 'run.visuals',
      },
    ],
  });
  const base = {
    apiBase: 'http://base.test',
    defaults: { runId: 'base-generated-run' },
    resources: { runIds: ['base-generated-run'] },
    semantic: baseSemantic,
  };
  const head = {
    apiBase: 'http://head.test',
    defaults: { runId: 'head-generated-run' },
    resources: { runIds: ['head-generated-run'] },
    semantic: headSemantic,
  };

  const combined = combineSemanticManifests(
    { base, head },
    {
      revisions: {
        base: { commit: 'base-sha', ref: '2.17.1' },
        head: { commit: 'head-sha', ref: 'pull/13986/head' },
      },
    },
  );

  assert.equal(combined.schemaVersion, SEMANTIC_SCHEMA_VERSION);
  assert.equal(combined.schemaVersion, 'ui-smoke-semantic/v2');
  assert.equal(combined.fixtureSet, SEMANTIC_FIXTURE_SET);
  assert.equal(combined.fixtureSet, 'ui-smoke-deterministic-v2');
  assert.equal(combined.deployments.base.revision.role, 'base');
  assert.equal(combined.deployments.base.revision.ref, '2.17.1');
  assert.equal(combined.deployments.head.revision.commit, 'head-sha');
  assert.equal(combined.deployments.base.defaults.runId, 'base-generated-run');
  assert.deepEqual(combined.deployments.head.resources.runIds, ['head-generated-run']);
  assert.equal(combined.deployments.base.bindings.runs['run.visuals'].runId, 'base-generated-run');
  assert.equal(combined.deployments.head.bindings.runs['run.visuals'].runId, 'head-generated-run');

  const incompatibleHead = structuredClone(head);
  incompatibleHead.semantic.logical.resources['run.visuals'].displayName = 'Different Run';
  assert.throws(
    () => combineSemanticManifests({ base, head: incompatibleHead }),
    /logical fixtures do not match/,
  );
});
