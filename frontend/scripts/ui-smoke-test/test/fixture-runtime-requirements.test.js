const assert = require('assert/strict');
const { test } = require('node:test');
const { parseAllDocuments, parseDocument } = require('yaml');

const { applyFixtureRuntimeRequirements } = require('../fixture-runtime-requirements');

function renderedManifest(workflowDefaults = null) {
  const defaults =
    workflowDefaults ||
    `spec:
  ttlStrategy:
    secondsAfterCompletion: 3600
  templateDefaults:
    retryStrategy:
      limit: '2'
      retryPolicy: OnError
`;
  const indented = defaults
    .trimEnd()
    .split('\n')
    .map((line) => `    ${line}`)
    .join('\n');
  return `apiVersion: v1
kind: ConfigMap
metadata:
  name: unrelated
data:
  keep: unchanged
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: kubeflow
data:
  artifactRepository: |
    archiveLogs: true
  workflowDefaults: |
${indented}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
`;
}

function parsedDefaults(manifestContents) {
  const documents = parseAllDocuments(manifestContents);
  const configMap = documents.find(
    (document) => document.getIn(['metadata', 'name']) === 'workflow-controller-configmap',
  );
  return {
    documents,
    workflowDefaults: parseDocument(configMap.getIn(['data', 'workflowDefaults'])),
  };
}

test('fixture retry policy rewrites only the rendered Argo workflow default', () => {
  const input = renderedManifest();
  const output = applyFixtureRuntimeRequirements(input, { argoRetryPolicy: 'OnFailure' });
  const { documents, workflowDefaults } = parsedDefaults(output);

  assert.equal(documents.length, 3);
  assert.equal(documents[0].getIn(['data', 'keep']), 'unchanged');
  assert.equal(documents[1].getIn(['data', 'artifactRepository']), 'archiveLogs: true\n');
  assert.equal(documents[2].getIn(['metadata', 'name']), 'workflow-controller');
  assert.equal(workflowDefaults.getIn(['spec', 'ttlStrategy', 'secondsAfterCompletion']), 3600);
  assert.equal(workflowDefaults.getIn(['spec', 'templateDefaults', 'retryStrategy', 'limit']), '2');
  assert.equal(
    workflowDefaults.getIn(['spec', 'templateDefaults', 'retryStrategy', 'retryPolicy']),
    'OnFailure',
  );
  assert.equal(applyFixtureRuntimeRequirements(output, { argoRetryPolicy: 'OnFailure' }), output);
});

test('empty fixture requirements preserve the rendered manifest byte-for-byte', () => {
  const input = renderedManifest();
  assert.equal(applyFixtureRuntimeRequirements(input), input);
  assert.equal(applyFixtureRuntimeRequirements(input, {}), input);
});

test('fixture requirements fail closed for missing, duplicate, or malformed Argo defaults', () => {
  const requirement = { argoRetryPolicy: 'OnFailure' };
  assert.throws(
    () =>
      applyFixtureRuntimeRequirements(
        renderedManifest().replace('workflow-controller-configmap', 'other-configmap'),
        requirement,
      ),
    /exactly one workflow-controller-configmap; found 0/,
  );
  assert.throws(
    () =>
      applyFixtureRuntimeRequirements(
        `${renderedManifest()}---\n${renderedManifest()}`,
        requirement,
      ),
    /exactly one workflow-controller-configmap; found 2/,
  );
  assert.throws(
    () =>
      applyFixtureRuntimeRequirements(
        renderedManifest(`spec:\n  templateDefaults: [not-a-map\n`),
        requirement,
      ),
    /data\.workflowDefaults is invalid YAML/,
  );
  assert.throws(
    () =>
      applyFixtureRuntimeRequirements(
        renderedManifest(`spec:\n  ttlStrategy:\n    secondsAfterCompletion: 3600\n`),
        requirement,
      ),
    /missing spec\.templateDefaults\.retryStrategy/,
  );
});

test('fixture requirements reject unknown keys and invalid retry policies', () => {
  const input = renderedManifest();
  assert.throws(
    () => applyFixtureRuntimeRequirements(input, { unknown: true }),
    /Unsupported fixture runtime requirements: unknown/,
  );
  assert.throws(
    () => applyFixtureRuntimeRequirements(input, { argoRetryPolicy: 'Sometimes' }),
    /Unsupported Argo retry policy fixture requirement/,
  );
});
