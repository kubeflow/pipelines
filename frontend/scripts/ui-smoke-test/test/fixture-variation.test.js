'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const { parseDocument } = require('yaml');
const { COMPARISON_RUN_FIXTURES } = require('../semantic-manifest');
const {
  metricsExecutorOutputForRun,
  createRun,
  MINIMAL_PIPELINE_YAML,
  RICH_PIPELINE_YAML,
} = require('../seed-data');
const { strictSemanticFixtureManifest } = require('./semantic-fixture');
const { buildSemanticIdentifierCatalog } = require('../capture-screenshots');
const {
  semanticIdShape,
  semanticIdToken,
  SEMANTIC_ID_TOKEN_PATTERN,
} = require('../semantic-id-normalization');

test('classification fixture includes a nonempty deterministic confusion matrix', () => {
  for (const [index, runKey] of COMPARISON_RUN_FIXTURES.entries()) {
    assert.deepEqual(
      metricsExecutorOutputForRun(runKey).artifacts.roc_curve.artifacts[0].metadata.confusionMatrix,
      {
        annotationSpecs: [
          { displayName: 'predicted-negative' },
          { displayName: 'predicted-positive' },
        ],
        rows: [{ row: [42 - index * 4, 8 + index * 4] }, { row: [3 + index * 4, 47 - index * 4] }],
      },
    );
  }
});

test('comparison runs have distinct deterministic curves/scalars and equivalent base/head values', async () => {
  const manifest = strictSemanticFixtureManifest();
  const curves = new Set();
  const scalars = new Set();
  for (const runKey of COMPARISON_RUN_FIXTURES) {
    const output = metricsExecutorOutputForRun(runKey);
    const metrics = output.artifacts.scalar_metrics.artifacts[0].metadata;
    const points = output.artifacts.roc_curve.artifacts[0].metadata.confidenceMetrics;
    curves.add(JSON.stringify(points));
    scalars.add(JSON.stringify(metrics));
    for (const role of ['base', 'head']) {
      const artifacts = manifest.deployments[role].bindings.runs[runKey].artifacts;
      assert.deepEqual(artifacts['artifact.roc-curve'].points, points);
      assert.equal(
        artifacts['artifact.scalar-metrics'].members['metric.accuracy'].numberValue,
        metrics.accuracy,
      );
      assert.equal(
        artifacts['artifact.scalar-metrics'].members['metric.loss'].numberValue,
        metrics.loss,
      );
    }
    let body;
    await createRun(
      'fixture',
      'pipeline',
      'experiment',
      async (_method, _url, request) => {
        body = request;
        return { run_id: 'created' };
      },
      'version',
      { semanticKey: runKey },
    );
    assert.deepEqual(body.runtime_config.parameters, { fixture_run: runKey });
    for (const yaml of [MINIMAL_PIPELINE_YAML, RICH_PIPELINE_YAML]) {
      const doc = parseDocument(yaml);
      assert.deepEqual(doc.errors, []);
      const pipeline = doc.toJS();
      assert.equal(pipeline.root.inputDefinitions.parameters.fixture_run.parameterType, 'STRING');
      assert.equal(
        pipeline.root.dag.tasks['write-metrics'].inputs.parameters.fixture_run
          .componentInputParameter,
        'fixture_run',
      );
      const args = pipeline.deploymentSpec.executors['exec-write-metrics'].container.args;
      assert.equal(args.at(-1), "{{$.inputs.parameters['fixture_run']}}");
      assert.ok(args[0].includes(`${runKey}) printf '%s' '${JSON.stringify(output)}'`));
    }
  }
  assert.equal(curves.size, 3);
  assert.equal(scalars.size, 3);
});

test('identity tokens retain realistic bounded shapes and unambiguous semantic mappings', () => {
  for (const [kind, raw, shape, length] of [
    ['run', '11223344-1234-1234-1234-112233445566', 'uuid', 36],
    ['artifact', '42', 'decimal', 6],
    ['pod', 'training-worker-ab123', 'text-21', 21],
    ['artifact-uri', 's3://very-long-generated-bucket/run/artifact', 'uri', 39],
  ]) {
    assert.equal(semanticIdShape(kind, raw), shape);
    const token = semanticIdToken(kind, 'run.training-1/task.write-metrics[0]', shape);
    assert.match(token, SEMANTIC_ID_TOKEN_PATTERN);
    assert.equal(token.length, length);
    assert.equal(token, semanticIdToken(kind, 'run.training-1/task.write-metrics[0]', shape));
  }
  const manifest = strictSemanticFixtureManifest();
  for (const role of ['base', 'head']) {
    const identities = new Map();
    for (const entry of buildSemanticIdentifierCatalog(manifest, role)) {
      const identity = `${entry.tokenKind}/${entry.tokenSemanticId}`;
      if (identities.has(entry.token)) assert.equal(identities.get(entry.token), identity);
      identities.set(entry.token, identity);
      assert.equal(
        entry.token,
        semanticIdToken(entry.tokenKind, entry.tokenSemanticId, entry.tokenShape),
      );
      assert.ok(entry.token.length <= 63);
    }
  }
  assert.notEqual(
    semanticIdToken('task', 'run.training-1/task.foo[0]'),
    semanticIdToken('task', 'run.training-1/foo[0]'),
  );
  assert.throws(
    () => semanticIdToken('run', 'run.training-1', 'text-999'),
    /Invalid semantic token shape/,
  );
});
