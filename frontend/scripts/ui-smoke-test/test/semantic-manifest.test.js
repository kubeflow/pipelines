const assert = require('node:assert/strict');
const test = require('node:test');
const { parse } = require('yaml');

const {
  ARTIFACT_FIXTURES,
  REVISION_FLAVORS,
  SEMANTIC_FIXTURE_SET,
  SEMANTIC_RESOURCE_DEFINITIONS,
  SEMANTIC_SCHEMA_VERSION,
  buildLogicalFixtures,
  buildSemanticDeployment,
  combineRevisionSemanticManifests,
  combineSemanticManifests,
  detectRevisionFlavor,
  extractRunBinding,
  semanticManifestForRevision,
  validateCombinedSemanticManifest,
} = require('../semantic-manifest');
const { RICH_PIPELINE_YAML } = require('../seed-data');
const { strictSemanticFixtureManifest } = require('./semantic-fixture');

const EXPECTED_ROC_POINTS = structuredClone(ARTIFACT_FIXTURES['artifact.roc-curve'].points);
const RICH_PIPELINE_SPEC = parse(RICH_PIPELINE_YAML);
const richRunObservation = (response) => ({
  pipelineSpec: RICH_PIPELINE_SPEC,
  response,
  semanticKey: 'run.rich',
});

test('native root identity requires an explicit parentless ROOT row and remains optional', () => {
  const response = nativeRichRun();
  assert.equal(extractRunBinding(response, 'run.rich').rootTask, undefined);
  response.tasks.push({ task_id: 'native-root', type: 'ROOT', name: 'Rich Visual Run' });
  assert.deepEqual(extractRunBinding(response, 'run.rich').rootTask, {
    taskId: 'native-root',
    type: 'ROOT',
  });
  response.tasks.at(-1).parent_task_id = 'other';
  assert.throws(
    () => extractRunBinding(response, 'run.rich'),
    /ROOT task must have an ID and no parent/,
  );
  delete response.tasks.at(-1).parent_task_id;
  response.tasks.push({ task_id: 'second-root', type: 'ROOT' });
  assert.throws(() => extractRunBinding(response, 'run.rich'), /multiple ROOT/);
});

test('combined semantic fixture validation recomputes required bindings instead of trusting flags', () => {
  const manifest = strictSemanticFixtureManifest();
  assert.deepEqual(manifest.logical, buildLogicalFixtures(SEMANTIC_RESOURCE_DEFINITIONS));
  assert.equal(validateCombinedSemanticManifest(manifest), manifest);

  const empty = structuredClone(manifest);
  empty.deployments.head.bindings = { resources: {}, runs: {} };
  assert.throws(
    () => validateCombinedSemanticManifest(empty),
    /missing resource binding run\.training-1/,
  );

  const missingUri = structuredClone(manifest);
  missingUri.deployments.head.bindings.runs['run.training-2'].artifacts[
    'artifact.html-report'
  ].records[0].uri = '';
  assert.throws(
    () => validateCombinedSemanticManifest(missingUri),
    /artifact\.html-report is missing a native artifact URI/,
  );

  const wrongLabel = structuredClone(manifest);
  wrongLabel.deployments.base.bindings.resources['run.evaluation'].displayName = 'Wrong Run';
  assert.throws(
    () => validateCombinedSemanticManifest(wrongLabel),
    /run\.evaluation has invalid kind or display name/,
  );
});

test('revision semantic manifests recombine into the exact validated fixture pair', () => {
  const combined = strictSemanticFixtureManifest();
  const seedManifest = (role) => ({
    defaults: combined.deployments[role].defaults,
    resources: combined.deployments[role].resources,
    semantic: {
      bindings: combined.deployments[role].bindings,
      fixtureSet: combined.fixtureSet,
      logical: combined.logical,
      revisionFlavor: combined.deployments[role].revisionFlavor,
      validation: combined.deployments[role].validation,
    },
  });
  const base = semanticManifestForRevision(seedManifest('base'), 'base', {
    commit: 'base-commit',
  });
  const head = semanticManifestForRevision(seedManifest('head'), 'head', {
    commit: 'head-commit',
  });
  const recombined = combineRevisionSemanticManifests(base, head);

  assert.deepEqual(recombined.deployments.base.bindings, combined.deployments.base.bindings);
  assert.deepEqual(recombined.deployments.head.bindings, combined.deployments.head.bindings);
  assert.equal(recombined.deployments.base.revision.commit, 'base-commit');
  assert.equal(recombined.deployments.head.revision.commit, 'head-commit');
  assert.equal(validateCombinedSemanticManifest(recombined), recombined);

  head.logical.resources['run.training-1'].displayName = 'Drifted';
  assert.throws(
    () => combineRevisionSemanticManifests(base, head),
    /logical\.resources does not match the deterministic fixture contract/,
  );
});

test('strict combined validation rejects missing and unexpected semantic keys', () => {
  const cases = [
    {
      mutate: (manifest) => delete manifest.logical.resources['experiment.image-classification'],
      name: 'missing logical resource',
      pattern: /logical\.resources does not match the deterministic fixture contract/,
    },
    {
      mutate: (manifest) => {
        manifest.logical.resources['experiment.unexpected'] = {
          displayName: 'Unexpected',
          kind: 'experiments',
        };
      },
      name: 'unexpected logical resource',
      pattern: /logical\.resources does not match the deterministic fixture contract/,
    },
    {
      mutate: (manifest) =>
        delete manifest.deployments.head.bindings.resources['pipeline.training.version'],
      name: 'missing deployment resource',
      pattern: /head deployment resource bindings.*missing pipeline\.training\.version/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.resources['pipeline.unexpected'] = {
          displayName: 'Unexpected',
          id: 'unexpected-pipeline-id',
          kind: 'pipeline',
        };
      },
      name: 'unexpected deployment resource',
      pattern: /head deployment resource bindings.*unexpected pipeline\.unexpected/,
    },
    {
      mutate: (manifest) => delete manifest.deployments.head.bindings.runs['run.inference'],
      name: 'missing deployment run',
      pattern: /head deployment run bindings.*missing run\.inference/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.unexpected'] = structuredClone(
          manifest.deployments.head.bindings.runs['run.inference'],
        );
      },
      name: 'unexpected deployment run',
      pattern: /head deployment run bindings.*unexpected run\.unexpected/,
    },
    {
      mutate: (manifest) =>
        delete manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
          'task.write-metrics'
        ],
      name: 'missing task key',
      pattern: /run\.training-2: taskInstances.*missing task\.write-metrics/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].taskInstances['task.unexpected'] =
          [];
      },
      name: 'unexpected task key',
      pattern: /run\.training-2: taskInstances.*unexpected task\.unexpected/,
    },
    {
      mutate: (manifest) =>
        delete manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.roc-curve'
        ],
      name: 'missing artifact key',
      pattern: /run\.training-2: artifacts.*missing artifact\.roc-curve/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts['artifact.unexpected'] =
          {};
      },
      name: 'unexpected artifact key',
      pattern: /run\.training-2: artifacts.*unexpected artifact\.unexpected/,
    },
    {
      mutate: (manifest) =>
        delete manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.scalar-metrics'
        ].members['metric.loss'],
      name: 'missing member key',
      pattern: /artifact\.scalar-metrics\.members.*missing metric\.loss/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.scalar-metrics'
        ].members['metric.unexpected'] = {};
      },
      name: 'unexpected member key',
      pattern: /artifact\.scalar-metrics\.members.*unexpected metric\.unexpected/,
    },
  ];

  for (const { mutate, name, pattern } of cases) {
    const manifest = strictSemanticFixtureManifest();
    mutate(manifest);
    assert.throws(() => validateCombinedSemanticManifest(manifest), pattern, name);
  }
});

test('strict combined validation rejects missing and unexpected relationship triples', () => {
  const missing = strictSemanticFixtureManifest();
  missing.deployments.head.bindings.runs['run.training-1'].relationships.shift();
  assert.throws(
    () => validateCombinedSemanticManifest(missing),
    /relationships do not match the semantic profile; missing artifact-consumer\|artifact\.scalar-metrics\|task\.consume-metrics/,
  );

  const unexpected = strictSemanticFixtureManifest();
  unexpected.deployments.head.bindings.runs['run.training-1'].relationships.push({
    kind: 'depends-on',
    source: 'task.nested-worker',
    target: 'task.write-metrics',
  });
  assert.throws(
    () => validateCombinedSemanticManifest(unexpected),
    /relationships do not match the semantic profile.*unexpected depends-on\|task\.nested-worker\|task\.write-metrics/,
  );
});

test('strict combined validation rejects malformed task-instance and relationship containers', () => {
  for (const [role, malformed] of [
    ['base', { length: 1 }],
    ['head', {}],
  ]) {
    const manifest = strictSemanticFixtureManifest();
    manifest.deployments[role].bindings.runs['run.training-2'].taskInstances['task.write-metrics'] =
      malformed;
    assert.throws(
      () => validateCombinedSemanticManifest(manifest),
      /taskInstances\.task\.write-metrics must be an array/,
      `${role} malformed taskInstances value`,
    );
  }

  const relationships = strictSemanticFixtureManifest();
  relationships.deployments.base.bindings.runs['run.training-2'].relationships = {};
  assert.throws(
    () => validateCombinedSemanticManifest(relationships),
    /run\.training-2: relationships must be an array/,
  );
});

test('strict combined validation rejects malformed identity leaves and artifact cardinality drift', () => {
  const strict = strictSemanticFixtureManifest();
  assert.equal(
    strict.deployments.base.bindings.runs['run.training-2'].artifacts['artifact.scalar-metrics']
      .artifactIds.length,
    1,
  );
  assert.equal(
    strict.deployments.head.bindings.runs['run.training-2'].artifacts['artifact.scalar-metrics']
      .artifactIds.length,
    2,
  );

  const cases = [
    {
      mutate: (manifest) => {
        manifest.deployments.base.bindings.runs['run.training-2'].taskInstances[
          'task.write-metrics'
        ] = [null];
      },
      name: 'null legacy task instance',
      pattern: /taskInstances\.task\.write-metrics\[0\] must be an object/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.base.bindings.runs['run.training-2'].taskInstances[
          'task.write-metrics'
        ] = [{}];
      },
      name: 'empty legacy task instance',
      pattern: /taskInstances\.task\.write-metrics\[0\]\.taskId must be a nonempty string ID/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.resources['experiment.image-classification'].id = {};
      },
      name: 'object resource ID',
      pattern: /resource binding experiment\.image-classification has an invalid generated ID/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].runId = {};
      },
      name: 'object run ID',
      pattern: /run binding run\.training-2 does not match its logical resource/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.roc-curve'
        ].artifactIds = {};
      },
      name: 'non-array artifact IDs',
      pattern: /artifact\.roc-curve\.artifactIds must be an array of nonempty scalar IDs/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.roc-curve'
        ].artifactIds = [{}];
      },
      name: 'object artifact ID',
      pattern: /artifact\.roc-curve\.artifactIds\[0\] is not a valid nonempty scalar ID/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.roc-curve'
        ].records[0].artifactId = {};
      },
      name: 'object record artifact ID',
      pattern: /artifact\.roc-curve\.records\[0\]\.artifactId is not a valid nonempty scalar ID/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.roc-curve'
        ].records[0].uri = {};
      },
      name: 'object record URI',
      pattern: /artifact\.roc-curve\.records\[0\]\.uri must be a nonempty string/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.scalar-metrics'
        ].members['metric.accuracy'].artifactIds = {};
      },
      name: 'non-array member IDs',
      pattern: /members\.metric\.accuracy\.artifactIds must be an array of nonempty scalar IDs/,
    },
    {
      mutate: (manifest) => {
        const artifact =
          manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
            'artifact.html-report'
          ];
        artifact.artifactIds.push('injected-html-artifact');
        artifact.records.push({
          artifactId: 'injected-html-artifact',
          uri: 's3://ui-smoke/injected.html',
        });
        artifact.files.push({
          artifactId: 'injected-html-artifact',
          name: 'html_report',
          type: 'system.HTML',
          uri: 's3://ui-smoke/injected.html',
        });
      },
      name: 'second non-scalar artifact',
      pattern: /artifact\.html-report\.artifactIds must contain exactly 1 ID/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].artifacts[
          'artifact.scalar-metrics'
        ].members['metric.accuracy'].artifactIds = ['injected-metric-artifact'];
      },
      name: 'member ID outside parent group',
      pattern: /scalar metric member IDs do not exactly cover the parent artifact IDs/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-1'].artifacts[
          'artifact.scalar-metrics'
        ].consumers['task.consume-metrics'].artifactIds.push('injected-consumer-artifact');
      },
      name: 'consumer ID outside parent group',
      pattern: /scalar metric producer and consumer artifact bindings differ/,
    },
  ];

  for (const { mutate, name, pattern } of cases) {
    const manifest = strictSemanticFixtureManifest();
    mutate(manifest);
    assert.throws(() => validateCombinedSemanticManifest(manifest), pattern, name);
  }
});

test('strict combined validation canonicalizes numeric legacy artifact IDs only', () => {
  const legacy = strictSemanticFixtureManifest();
  const scalar =
    legacy.deployments.base.bindings.runs['run.training-2'].artifacts['artifact.scalar-metrics'];
  scalar.artifactIds = [81];
  scalar.records[0].artifactId = 81;
  scalar.members['metric.accuracy'].artifactIds = [81];
  scalar.members['metric.loss'].artifactIds = [81];
  const scalarOutput =
    legacy.deployments.base.bindings.runs['run.training-2'].taskInstances['task.write-metrics'][0]
      .artifactReferences.outputs;
  scalarOutput.find((group) => group.key === 'scalar_metrics').artifacts[0].artifactId = 81;
  assert.equal(validateCombinedSemanticManifest(legacy), legacy);

  const native = strictSemanticFixtureManifest();
  native.deployments.head.bindings.runs['run.training-2'].artifacts[
    'artifact.roc-curve'
  ].artifactIds[0] = 81;
  assert.throws(
    () => validateCombinedSemanticManifest(native),
    /artifact\.roc-curve\.artifactIds\[0\] is not a valid nonempty scalar ID/,
  );
});

test('strict combined validation requires an exact two-pod native retry', () => {
  const withDriver = strictSemanticFixtureManifest();
  withDriver.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].podBindings.unshift({
    name: 'retry-driver',
    type: 'DRIVER',
    uid: '00000000-aaaa-bbbb-cccc-000000000000',
  });
  assert.equal(validateCombinedSemanticManifest(withDriver), withDriver);

  const malformedExecutorPods = strictSemanticFixtureManifest();
  malformedExecutorPods.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].executorPods = { length: 2 };
  assert.throws(
    () => validateCombinedSemanticManifest(malformedExecutorPods),
    /executorPods must contain exactly 2 nonempty pod names/,
  );

  const malformedPodBindings = strictSemanticFixtureManifest();
  malformedPodBindings.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].podBindings = { length: 2 };
  assert.throws(
    () => validateCombinedSemanticManifest(malformedPodBindings),
    /podBindings must contain exactly 2 EXECUTOR records/,
  );

  const mismatchedNames = strictSemanticFixtureManifest();
  mismatchedNames.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].podBindings[0].name = 'different-pod';
  assert.throws(
    () => validateCombinedSemanticManifest(mismatchedNames),
    /pod binding names differ from executorPods/,
  );

  const thirdPod = strictSemanticFixtureManifest();
  const retry =
    thirdPod.deployments.head.bindings.runs['run.training-1'].taskInstances['task.retry-once'][0];
  retry.executorPods.push('retry-attempt-2');
  retry.podBindings.push({
    name: 'retry-attempt-2',
    type: 'EXECUTOR',
    uid: '33333333-aaaa-bbbb-cccc-333333333333',
  });
  assert.throws(
    () => validateCombinedSemanticManifest(thirdPod),
    /recorded 3 attempt\(s\), expected exactly 2/,
  );

  const twoDrivers = strictSemanticFixtureManifest();
  twoDrivers.deployments.head.bindings.runs['run.training-1'].taskInstances[
    'task.retry-once'
  ][0].podBindings.unshift(
    {
      name: 'retry-driver-0',
      type: 'DRIVER',
      uid: '00000000-aaaa-bbbb-cccc-000000000000',
    },
    {
      name: 'retry-driver-1',
      type: 'DRIVER',
      uid: '00000000-aaaa-bbbb-cccc-000000000001',
    },
  );
  assert.throws(
    () => validateCombinedSemanticManifest(twoDrivers),
    /may contain at most one DRIVER pod binding/,
  );
});

test('strict combined validation requires exact native ParallelFor runtime evidence', () => {
  const cases = [
    {
      mutate: (run) => {
        delete run.taskInstances['task.parallel-loop'][0].iterationCount;
      },
      name: 'missing loop iteration count',
      pattern: /iteration count undefined did not match 2/,
    },
    {
      mutate: (run) => {
        run.taskInstances['task.parallel-loop'][0].iterationCount = 3;
      },
      name: 'wrong loop iteration count',
      pattern: /iteration count 3 did not match 2/,
    },
    {
      mutate: (run) => {
        run.taskInstances['task.loop-worker'][1].iterationIndex = 0;
      },
      name: 'duplicate worker iteration index',
      pattern: /task\.loop-worker iteration indexes \[0,0\] did not match \[0,1\]/,
    },
    {
      mutate: (run) => {
        run.taskInstances['task.loop-worker'][1].parentTaskId = 'wrong-loop';
      },
      name: 'worker with wrong loop parent',
      pattern: /tasks must be direct children of the native loop task/,
    },
  ];

  for (const { mutate, name, pattern } of cases) {
    const manifest = strictSemanticFixtureManifest();
    mutate(manifest.deployments.head.bindings.runs['run.training-1']);
    assert.throws(() => validateCombinedSemanticManifest(manifest), pattern, name);
  }
});

test('strict combined validation permits only deterministic native executor-log outputs', () => {
  const ordinary = strictSemanticFixtureManifest();
  assert.equal(validateCombinedSemanticManifest(ordinary), ordinary);

  const cases = [
    {
      mutate: (manifest) => {
        delete manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
          'task.write-metrics'
        ][0].artifactReferences;
      },
      name: 'missing ordinary runtime artifact references',
      pattern: /must contain exactly one native runtime executor-logs output group/,
    },
    ...['task.write-metrics', 'task.consume-metrics'].map((taskKey) => ({
      mutate: (manifest) => {
        delete manifest.deployments.base.bindings.runs['run.training-1'].taskInstances[taskKey][0]
          .artifactReferences;
      },
      name: `missing legacy declared artifact references for ${taskKey}`,
      pattern: /must contain the declared semantic artifact groups/,
    })),
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
          'task.write-metrics'
        ][0].artifactReferences.outputs = [];
      },
      name: 'missing ordinary runtime executor-log group',
      pattern: /must contain exactly 1 native runtime executor-logs output group.*found 0/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.base.bindings.runs['run.training-1'].taskInstances[
          'task.retry-once'
        ][0].artifactReferences = structuredClone(
          manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
            'task.retry-once'
          ][0].artifactReferences,
        );
      },
      name: 'legacy executor logs',
      pattern: /contains forbidden executor-logs/,
    },
    {
      mutate: (manifest) => {
        const references =
          manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
            'task.retry-once'
          ][0].artifactReferences;
        references.inputs = references.outputs;
        references.outputs = [];
      },
      name: 'executor logs as inputs',
      pattern: /contains forbidden executor-logs/,
    },
    {
      mutate: (manifest) => {
        const references =
          manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
            'task.retry-once'
          ][0].artifactReferences;
        manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
          'task.nested-dag'
        ][0].artifactReferences = {
          inputs: [],
          outputs: [structuredClone(references.outputs[0])],
        };
      },
      name: 'executor logs on DAG task',
      pattern: /contains forbidden executor-logs/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        run.taskInstances['task.parallel-loop'][0].artifactReferences = structuredClone(
          run.taskInstances['task.write-metrics'][0].artifactReferences,
        );
      },
      name: 'executor logs on loop controller',
      pattern: /contains forbidden executor-logs/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
          'task.write-metrics'
        ][0].artifactReferences = {
          inputs: [],
          outputs: [
            {
              artifacts: [{ artifactId: 'undeclared', uri: 's3://ui-smoke/undeclared' }],
              key: 'undeclared-output',
            },
          ],
        };
      },
      name: 'arbitrary undeclared output',
      pattern: /artifactId is not declared by a semantic artifact binding/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const artifact = run.artifacts['artifact.html-report'];
        run.taskInstances['task.write-metrics'][0].artifactReferences.outputs.push({
          artifacts: [structuredClone(artifact.records[0])],
          key: 'undeclared-alias',
        });
      },
      name: 'invented output port reusing a declared artifact',
      pattern: /invalid artifact group keys; unexpected undeclared-alias/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const artifact = run.artifacts['artifact.html-report'];
        run.taskInstances['task.write-metrics'][0].artifactReferences.outputs.push({
          artifacts: [structuredClone(artifact.records[0])],
        });
      },
      name: 'missing output port reusing a declared artifact',
      pattern: /invalid artifact group keys; unexpected artifact/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const outputs = run.taskInstances['task.write-metrics'][0].artifactReferences.outputs;
        const html = outputs.find((group) => group.key === 'html_report');
        const roc = outputs.find((group) => group.key === 'roc_curve');
        [html.artifacts, roc.artifacts] = [roc.artifacts, html.artifacts];
      },
      name: 'declared output ports swapped between logical artifacts',
      pattern: /artifact IDs do not match artifact\.(html-report|roc-curve)/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const outputs = run.taskInstances['task.write-metrics'][0].artifactReferences.outputs;
        outputs.find((group) => group.key === 'html_report').artifacts[0].uri =
          run.artifacts['artifact.roc-curve'].records[0].uri;
      },
      name: 'declared output ID paired with another logical artifact URI',
      pattern: /uri does not match artifact\.html-report record/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const inputs = run.taskInstances['task.consume-metrics'][0].artifactReferences.inputs;
        inputs.find((group) => group.key === 'metrics').artifacts = [
          structuredClone(run.artifacts['artifact.html-report'].records[0]),
          structuredClone(run.artifacts['artifact.roc-curve'].records[0]),
        ];
      },
      name: 'consumer port references unrelated declared artifacts',
      pattern: /artifact IDs do not match artifact\.scalar-metrics/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
          'task.retry-once'
        ][0].artifactReferences.outputs[0].artifacts.pop();
      },
      name: 'missing retry executor log',
      pattern: /artifacts must contain exactly 2 record\(s\), found 1/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
          'task.retry-once'
        ][0].artifactReferences.outputs[0].artifacts.reverse();
      },
      name: 'out-of-order retry executor logs',
      pattern: /uri must end exactly in deterministic executor-log leaf executor-logs-0/,
    },
    {
      mutate: (manifest) => {
        const artifacts =
          manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
            'task.retry-once'
          ][0].artifactReferences.outputs[0].artifacts;
        artifacts[1].artifactId = artifacts[0].artifactId;
        artifacts[1].uri = artifacts[0].uri;
      },
      name: 'duplicate retry executor logs',
      pattern: /contains duplicate executor-log IDs/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        run.taskInstances[
          'task.retry-once'
        ][0].artifactReferences.outputs[0].artifacts[0].artifactId =
          run.artifacts['artifact.roc-curve'].artifactIds[0];
      },
      name: 'executor log collides with declared artifact',
      pattern: /artifactId collides with a declared semantic artifact/,
    },
    {
      mutate: (manifest) => {
        manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
          'task.retry-once'
        ][0].artifactReferences.outputs[0].artifacts[0].uri =
          's3://ui-smoke/head/run.training-1/executor-logs';
      },
      name: 'unsuffixed executor log URI',
      pattern: /uri must end exactly in deterministic executor-log leaf executor-logs-0/,
    },
    ...['?download=true', '#fragment', '/'].map((suffix) => ({
      mutate: (manifest) => {
        const outputs =
          manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
            'task.write-metrics'
          ][0].artifactReferences.outputs;
        outputs.find((group) => group.key === 'executor-logs').artifacts[0].uri += suffix;
      },
      name: `executor log URI suffix ${suffix}`,
      pattern: /uri must end exactly in deterministic executor-log leaf executor-logs-0/,
    })),
    {
      mutate: (manifest) => {
        const outputs =
          manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
            'task.write-metrics'
          ][0].artifactReferences.outputs;
        delete outputs.find((group) => group.key === 'executor-logs').artifacts[0].name;
      },
      name: 'missing executor log name',
      pattern: /missing name/,
    },
    {
      mutate: (manifest) => {
        const outputs =
          manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
            'task.write-metrics'
          ][0].artifactReferences.outputs;
        outputs.find((group) => group.key === 'executor-logs').artifacts[0].type =
          'system.Artifact';
      },
      name: 'wrong executor log REST type',
      pattern: /type must be Artifact/,
    },
    {
      mutate: (manifest) => {
        const outputs =
          manifest.deployments.head.bindings.runs['run.training-2'].taskInstances[
            'task.write-metrics'
          ][0].artifactReferences.outputs;
        outputs.find((group) => group.key === 'executor-logs').artifacts[0].description =
          'injected';
      },
      name: 'extra executor log record field',
      pattern: /unexpected description/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const retryGroup = run.taskInstances['task.retry-once'][0].artifactReferences.outputs[0];
        retryGroup.artifacts[0].uri = run.artifacts['artifact.html-report'].records[0].uri;
      },
      name: 'executor log URI collides with declared artifact',
      pattern: /uri collides with a declared semantic artifact/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-1'];
        const retryGroup = run.taskInstances['task.retry-once'][0].artifactReferences.outputs[0];
        run.taskInstances['task.write-metrics'][0].artifactReferences = {
          inputs: [],
          outputs: [
            {
              artifacts: [structuredClone(retryGroup.artifacts[0])],
              key: 'executor-logs',
            },
          ],
        };
      },
      name: 'executor log reused across tasks',
      pattern: /contains duplicate executor-log IDs/,
    },
    {
      mutate: (manifest) => {
        const references =
          manifest.deployments.head.bindings.runs['run.training-1'].taskInstances[
            'task.retry-once'
          ][0].artifactReferences;
        const duplicate = structuredClone(references.outputs[0]);
        duplicate.artifacts = duplicate.artifacts.map((record, index) => ({
          artifactId: `${record.artifactId}-duplicate`,
          uri: `s3://ui-smoke/duplicate/executor-logs-${index}`,
        }));
        references.outputs.push(duplicate);
      },
      name: 'second executor-log group',
      pattern: /duplicate normalized group key executor-logs|found 2/,
    },
    {
      mutate: (manifest) => {
        const run = manifest.deployments.head.bindings.runs['run.training-2'];
        const artifact = run.artifacts['artifact.roc-curve'];
        run.taskInstances['task.write-metrics'][0].artifactReferences = {
          inputs: [],
          outputs: [
            {
              artifacts: [structuredClone(artifact.records[0])],
              key: 'duplicate_group',
            },
            {
              artifacts: [structuredClone(artifact.records[0])],
              key: 'duplicate-group',
            },
          ],
        };
      },
      name: 'duplicate normalized output group keys',
      pattern: /contains duplicate normalized group key duplicate-group/,
    },
  ];

  for (const { mutate, name, pattern } of cases) {
    const manifest = strictSemanticFixtureManifest();
    mutate(manifest);
    assert.throws(() => validateCombinedSemanticManifest(manifest), pattern, name);
  }
});

test('strict legacy lineage requires the complete ParallelFor execution topology', () => {
  const cases = [
    {
      mutate: (run) => run.executionInstances['task.parallel-loop'].pop(),
      name: 'missing loop iteration execution',
      pattern: /task\.parallel-loop must contain exactly 3 execution\(s\)/,
    },
    {
      mutate: (run) => {
        run.executionInstances['task.parallel-loop'][1].parentDagId = 'wrong-controller';
      },
      name: 'iteration with wrong controller',
      pattern: /loop-iteration executions must be children of the loop controller/,
    },
    {
      mutate: (run) => {
        run.executionInstances['task.loop-worker'][1].parentDagId = 'wrong-iteration';
      },
      name: 'worker with wrong iteration parent',
      pattern: /task\.loop-worker executions must be children of their matching loop iterations/,
    },
    {
      mutate: (run) => {
        const workers = run.executionInstances['task.loop-worker'];
        workers[1].iterationIndex = workers[0].iterationIndex;
        workers[1].parentDagId = workers[0].parentDagId;
      },
      name: 'duplicate worker iteration index',
      pattern: /task\.loop-worker execution indexes must exactly match 0, 1/,
    },
    {
      mutate: (run) => {
        run.taskInstances['task.parallel-loop'][0].mlmdExecutionId =
          run.executionInstances['task.parallel-loop'][1].executionId;
      },
      name: 'loop task bound to iteration instead of controller',
      pattern: /taskInstances\.task\.parallel-loop MLMD execution IDs do not exactly match/,
    },
    {
      mutate: (run) => {
        run.relationships.find((relationship) => relationship.kind === 'depends-on').evidence =
          'runtime-observed';
      },
      name: 'unattributed declared dependency',
      pattern: /legacy depends-on relationships must declare pipeline-version-spec evidence/,
    },
  ];

  for (const { mutate, name, pattern } of cases) {
    const manifest = strictSemanticFixtureManifest();
    mutate(manifest.deployments.base.bindings.runs['run.training-1']);
    assert.throws(() => validateCombinedSemanticManifest(manifest), pattern, name);
  }
});

function legacyMetricArtifacts() {
  return [
    {
      artifactId: '81',
      metadata: { accuracy: 0.92, loss: 0.08 },
      uri: 's3://fixtures/scalar-metrics',
    },
    {
      artifactId: '82',
      metadata: { confidenceMetrics: structuredClone(EXPECTED_ROC_POINTS) },
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
            html_report: { artifact_ids: [83] },
            markdown_report: { artifact_ids: [84] },
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
  const response = {
    display_name: 'Rich Visual Run',
    run_id: 'native-rich-run',
    task_count: 8,
    tasks: [
      {
        name: 'write-metrics',
        outputs: {
          artifacts: [
            {
              artifact_key: 'scalar_metrics',
              artifacts: [
                {
                  artifact_id: 'metric-accuracy',
                  name: 'accuracy',
                  number_value: 0.92,
                },
                {
                  artifact_id: 'metric-loss',
                  name: 'loss',
                  number_value: 0.08,
                },
              ],
            },
            {
              artifact_key: 'roc_curve',
              artifacts: [nativeRocArtifact()],
            },
            {
              artifact_key: 'html_report',
              artifacts: [
                {
                  artifact_id: 'html-artifact',
                  name: 'html_report',
                  type: 'HTML',
                  uri: 's3://fixtures/report.html',
                },
              ],
            },
            {
              artifact_key: 'markdown_report',
              artifacts: [
                {
                  artifact_id: 'markdown-artifact',
                  name: 'markdown_report',
                  type: 'Markdown',
                  uri: 's3://fixtures/report.md',
                },
              ],
            },
          ],
        },
        scope_path: 'root.write-metrics',
        state: 'SUCCEEDED',
        task_id: 'native-write',
        type: 'RUNTIME',
      },
      {
        inputs: {
          artifacts: [
            {
              artifact_key: 'metrics',
              artifacts: [
                {
                  artifact_id: 'metric-accuracy',
                  name: 'accuracy',
                  number_value: 0.92,
                },
                {
                  artifact_id: 'metric-loss',
                  name: 'loss',
                  number_value: 0.08,
                },
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
        name: 'retry-once',
        pods: [
          { name: 'retry-driver', type: 1, uid: 'retry-driver-uid' },
          { name: 'retry-attempt-0', type: 'EXECUTOR', uid: 'retry-uid-0' },
          { name: 'retry-attempt-1', type: 'EXECUTOR', uid: 'retry-uid-1' },
        ],
        scope_path: 'root.retry-once',
        state: 'SUCCEEDED',
        task_id: 'native-retry',
        type: 'RUNTIME',
      },
      {
        child_tasks: [
          { name: 'loop-worker', task_id: 'native-loop-worker-0' },
          { name: 'loop-worker', task_id: 'native-loop-worker-1' },
        ],
        name: 'parallel-loop',
        scope_path: 'root.parallel-loop',
        state: 'SUCCEEDED',
        task_id: 'native-loop',
        type: 'LOOP',
        type_attributes: { iteration_count: 2 },
      },
      {
        name: 'loop-worker',
        parent_task_id: 'native-loop',
        scope_path: 'root.parallel-loop.loop-worker',
        state: 'SUCCEEDED',
        task_id: 'native-loop-worker-0',
        type: 'RUNTIME',
        type_attributes: { iteration_index: 0 },
      },
      {
        name: 'loop-worker',
        parent_task_id: 'native-loop',
        scope_path: 'root.parallel-loop.loop-worker',
        state: 'SUCCEEDED',
        task_id: 'native-loop-worker-1',
        type: 'RUNTIME',
        type_attributes: { iteration_index: 1 },
      },
      {
        child_tasks: [{ name: 'nested-worker', task_id: 'native-nested-worker' }],
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
  for (const task of response.tasks.filter((task) => task.type === 'RUNTIME')) {
    const attemptCount = task.name === 'retry-once' ? 2 : 1;
    task.outputs ||= {};
    task.outputs.artifacts ||= [];
    task.outputs.artifacts.push({
      artifact_key: 'executor-logs',
      artifacts: Array.from({ length: attemptCount }, (_, attemptIndex) => ({
        artifact_id: `${task.task_id}-executor-log-${attemptIndex}`,
        name: 'executor-logs',
        type: 'Artifact',
        uri: `s3://fixtures/${task.task_id}/executor-logs-${attemptIndex}`,
      })),
    });
  }
  return response;
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
    runResponses: [richRunObservation(response)],
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
    runResponses: [richRunObservation(wrong)],
  });
  const missingSemantic = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(missing)],
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
        runResponses: [richRunObservation(candidate)],
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

test('requires deterministic HTML and Markdown artifacts in each revision', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const legacy = legacyRichRun();
  delete legacy.run_details.task_details[0].outputs.html_report;
  const native = nativeRichRun();
  native.tasks[0].outputs.artifacts = native.tasks[0].outputs.artifacts.filter(
    (group) => group.artifact_key !== 'markdown_report',
  );

  const legacySemantic = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(legacy)],
  });
  const nativeSemantic = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(native)],
  });

  assert.equal(legacySemantic.validation.valid, false);
  assert.match(
    legacySemantic.validation.errors.join('\n'),
    /artifact\.html-report\.artifactIds must contain exactly 1 ID/,
  );
  assert.equal(nativeSemantic.validation.valid, false);
  assert.match(
    nativeSemantic.validation.errors.join('\n'),
    /artifact\.markdown-report\.artifactIds must contain exactly 1 ID/,
  );
});

test('rejects native tasks without stable IDs and file artifacts with wrong metadata', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const missingTaskId = nativeRichRun();
  delete missingTaskId.tasks[0].task_id;
  const invalidFiles = nativeRichRun();
  const html = invalidFiles.tasks[0].outputs.artifacts.find(
    (group) => group.artifact_key === 'html_report',
  ).artifacts[0];
  html.type = 'Dataset';
  const markdown = invalidFiles.tasks[0].outputs.artifacts.find(
    (group) => group.artifact_key === 'markdown_report',
  ).artifacts[0];
  markdown.uri = '';

  const missingTaskIdSemantic = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(missingTaskId)],
  });
  const invalidFileSemantic = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(invalidFiles)],
  });

  assert.equal(missingTaskIdSemantic.validation.valid, false);
  assert.match(missingTaskIdSemantic.validation.errors.join('\n'), /missing a native task ID/);
  assert.equal(invalidFileSemantic.validation.valid, false);
  assert.match(
    invalidFileSemantic.validation.errors.join('\n'),
    /artifact\.html-report has native type Dataset, expected html/,
  );
  assert.match(
    invalidFileSemantic.validation.errors.join('\n'),
    /artifact\.markdown-report is missing a native artifact URI/,
  );
});

test('rejects legacy MLMD file artifacts with unusable metadata', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const artifacts = legacyMetricArtifacts();
  const html = artifacts.find((artifact) => artifact.artifactId === '83');
  html.type = 'system.Dataset';
  const markdown = artifacts.find((artifact) => artifact.artifactId === '84');
  markdown.uri = '';

  const semantic = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation({ ...legacyRichRun(), semantic_artifacts: artifacts })],
  });

  assert.equal(semantic.validation.valid, false);
  assert.match(
    semantic.validation.errors.join('\n'),
    /artifact\.html-report has legacy MLMD type system\.Dataset, expected html/,
  );
  assert.match(
    semantic.validation.errors.join('\n'),
    /artifact\.markdown-report is missing a legacy MLMD artifact URI/,
  );
});

test('maps rich legacy and native topology to instance groups and semantic relationships', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const legacy = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(legacyRichRun())],
  });
  const native = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(nativeRichRun())],
  });

  assert.equal(legacy.validation.valid, true, legacy.validation.errors.join('; '));
  assert.equal(native.validation.valid, true, native.validation.errors.join('; '));
  assert.deepEqual(legacy.bindings.runs['run.rich'].artifacts['artifact.html-report'].artifactIds, [
    '83',
  ]);
  assert.deepEqual(
    native.bindings.runs['run.rich'].artifacts['artifact.markdown-report'].artifactIds,
    ['markdown-artifact'],
  );
  assert.equal(legacy.bindings.runs['run.rich'].taskInstances['task.loop-worker'].length, 2);
  assert.deepEqual(
    native.bindings.runs['run.rich'].taskInstances['task.loop-worker'].map(
      (instance) => instance.iterationIndex,
    ),
    [0, 1],
  );
  assert.equal(legacy.bindings.runs['run.rich'].tasks['task.retry-once'].failedMainJobs.length, 1);
  assert.equal(native.bindings.runs['run.rich'].tasks['task.retry-once'].executorPods.length, 2);
  assert.deepEqual(native.bindings.runs['run.rich'].tasks['task.retry-once'].podBindings, [
    { name: 'retry-driver', type: 'DRIVER', uid: 'retry-driver-uid' },
    { name: 'retry-attempt-0', type: 'EXECUTOR', uid: 'retry-uid-0' },
    { name: 'retry-attempt-1', type: 'EXECUTOR', uid: 'retry-uid-1' },
  ]);
  assert.deepEqual(native.bindings.runs['run.rich'].scopeInstances, {});
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
  const expectedDependencies = [
    {
      evidence: 'pipeline-version-spec',
      kind: 'depends-on',
      occurrences: 1,
      source: 'task.consume-metrics',
      target: 'task.nested-dag',
    },
    {
      evidence: 'pipeline-version-spec',
      kind: 'depends-on',
      occurrences: 1,
      source: 'task.parallel-loop',
      target: 'task.nested-dag',
    },
    {
      evidence: 'pipeline-version-spec',
      kind: 'depends-on',
      occurrences: 1,
      source: 'task.retry-once',
      target: 'task.nested-dag',
    },
    {
      evidence: 'pipeline-version-spec',
      kind: 'depends-on',
      occurrences: 1,
      source: 'task.write-metrics',
      target: 'task.consume-metrics',
    },
  ];
  for (const deployment of [legacy, native]) {
    assert.deepEqual(
      deployment.bindings.runs['run.rich'].relationships.filter(
        (relationship) => relationship.kind === 'depends-on',
      ),
      expectedDependencies,
    );
  }
});

test('does not backfill missing legacy dependencies from the expected semantic profile', () => {
  const pipelineSpec = structuredClone(RICH_PIPELINE_SPEC);
  pipelineSpec.root.dag.tasks['consume-metrics'].dependentTasks = [];

  const semantic = buildSemanticDeployment({
    logical: buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS),
    runResponses: [
      {
        pipelineSpec,
        response: legacyRichRun(),
        semanticKey: 'run.rich',
      },
    ],
  });
  const relationships = semantic.bindings.runs['run.rich'].relationships;

  assert.equal(semantic.validation.valid, false);
  assert.equal(
    relationships.some(
      (relationship) =>
        relationship.kind === 'depends-on' &&
        relationship.source === 'task.write-metrics' &&
        relationship.target === 'task.consume-metrics',
    ),
    false,
  );
  assert.match(
    semantic.validation.errors.join('\n'),
    /missing depends-on\|task\.write-metrics\|task\.consume-metrics/,
  );
});

test('normalizes task Artifact reference order independently of native API ordering', () => {
  const original = nativeRichRun();
  const reordered = nativeRichRun();
  const reorderedOutputs = reordered.tasks.find((task) => task.name === 'write-metrics').outputs
    .artifacts;
  reorderedOutputs.reverse();
  reorderedOutputs.find((group) => group.artifact_key === 'scalar_metrics').artifacts.reverse();

  const originalBinding = extractRunBinding(original, 'run.rich');
  const reorderedBinding = extractRunBinding(reordered, 'run.rich');
  assert.deepEqual(
    originalBinding.taskInstances['task.write-metrics'][0].artifactReferences,
    reorderedBinding.taskInstances['task.write-metrics'][0].artifactReferences,
  );
});

test('normalizes executor-log references by deterministic URI attempt suffix', () => {
  const response = nativeRichRun();
  response.tasks.find((task) => task.name === 'retry-once').outputs = {
    artifacts: [
      {
        artifact_key: 'executor-logs',
        artifacts: [
          {
            artifact_id: 'executor-log-1',
            name: 'executor-logs',
            uri: 's3://fixtures/executor-logs-1',
          },
          {
            artifact_id: 'executor-log-0',
            name: 'executor-logs',
            uri: 's3://fixtures/executor-logs-0',
          },
        ],
      },
    ],
  };

  const retry = extractRunBinding(response, 'run.rich').taskInstances['task.retry-once'][0];
  assert.deepEqual(
    retry.artifactReferences.outputs[0].artifacts.map((artifact) => artifact.uri),
    ['s3://fixtures/executor-logs-0', 's3://fixtures/executor-logs-1'],
  );
});

test('requires legacy metric URIs while accepting native value-only metrics', () => {
  const logical = buildLogicalFixtures(RICH_RESOURCE_DEFINITIONS);
  const legacyResponse = legacyRichRun();
  delete legacyResponse.semanticArtifacts.find((artifact) => artifact.artifactId === '81').uri;
  const legacy = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(legacyResponse)],
  });
  assert.equal(legacy.validation.valid, false);
  assert.match(
    legacy.validation.errors.join('\n'),
    /artifact\.scalar-metrics is missing a legacy MLMD artifact URI/,
  );

  const nativeResponse = nativeRichRun();
  const native = buildSemanticDeployment({
    logical,
    runResponses: [richRunObservation(nativeResponse)],
  });
  assert.equal(native.validation.valid, true, native.validation.errors.join('; '));
  assert.deepEqual(
    native.bindings.runs['run.rich'].artifacts['artifact.scalar-metrics'].records.map(
      (record) => record.uri,
    ),
    [null, null],
  );
  assert.equal(
    native.bindings.runs['run.rich'].artifacts['artifact.roc-curve'].records[0].uri,
    null,
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
    runResponses: [richRunObservation(response)],
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
  assert.equal(combined.schemaVersion, 'ui-smoke-semantic/v3');
  assert.equal(combined.fixtureSet, SEMANTIC_FIXTURE_SET);
  assert.equal(combined.fixtureSet, 'ui-smoke-deterministic-v3');
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
