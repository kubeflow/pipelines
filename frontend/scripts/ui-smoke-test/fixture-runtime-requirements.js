/**
 * Apply fixture-declared runtime requirements to a rendered, disposable KFP stack.
 *
 * The source manifests are never changed. This module rewrites only the embedded Argo workflow
 * defaults in the rendered manifest that is about to be applied by the smoke-test utility.
 */

const SUPPORTED_ARGO_RETRY_POLICIES = new Set([
  'Always',
  'OnError',
  'OnFailure',
  'OnTransientError',
]);
const SUPPORTED_REQUIREMENTS = new Set(['argoRetryPolicy']);

function validateRequirements(requirements) {
  if (requirements === undefined || requirements === null) return null;
  if (
    typeof requirements !== 'object' ||
    Array.isArray(requirements) ||
    Object.getPrototypeOf(requirements) !== Object.prototype
  ) {
    throw new Error('Fixture runtime requirements must be a plain object.');
  }

  const keys = Object.keys(requirements);
  const unsupported = keys.filter((key) => !SUPPORTED_REQUIREMENTS.has(key));
  if (unsupported.length > 0) {
    throw new Error(`Unsupported fixture runtime requirements: ${unsupported.join(', ')}.`);
  }
  if (keys.length === 0) return null;
  if (!SUPPORTED_ARGO_RETRY_POLICIES.has(requirements.argoRetryPolicy)) {
    throw new Error(
      `Unsupported Argo retry policy fixture requirement: ${JSON.stringify(
        requirements.argoRetryPolicy,
      )}.`,
    );
  }
  return requirements;
}

function parseDocuments(manifestContents) {
  // Keep YAML out of the CLI startup path. Browser-only and teardown operations do not need it.
  const { parseAllDocuments } = require('yaml');
  const documents = parseAllDocuments(manifestContents, {
    keepSourceTokens: true,
    maxAliasCount: 100,
  });
  const errors = documents.flatMap((document) => document.errors || []);
  if (errors.length > 0) {
    throw new Error(`Rendered revision manifests are invalid YAML: ${errors[0].message}`);
  }
  return documents;
}

function findWorkflowControllerConfigMap(documents) {
  const matches = documents.filter(
    (document) =>
      document.getIn(['apiVersion']) === 'v1' &&
      document.getIn(['kind']) === 'ConfigMap' &&
      document.getIn(['metadata', 'name']) === 'workflow-controller-configmap',
  );
  if (matches.length !== 1) {
    throw new Error(
      `Rendered revision manifests must contain exactly one workflow-controller-configmap; found ${matches.length}.`,
    );
  }
  return matches[0];
}

function serializeBlockScalar(node, value) {
  if (
    node?.type !== 'BLOCK_LITERAL' ||
    node.srcToken?.type !== 'block-scalar' ||
    !Array.isArray(node.srcToken.props) ||
    !Array.isArray(node.range) ||
    node.range.length < 2
  ) {
    throw new Error('workflow-controller-configmap data.workflowDefaults must be a YAML block.');
  }
  const indentation = node.srcToken.source.match(/(?:^|\n)( +)\S/)?.[1];
  if (!indentation) {
    throw new Error('Could not determine workflowDefaults block indentation.');
  }
  const header = node.srcToken.props.map((token) => token.source || '').join('');
  if (!header.startsWith('|') || !header.endsWith('\n')) {
    throw new Error('workflowDefaults has an unsupported YAML block header.');
  }
  const body = value
    .trimEnd()
    .split('\n')
    .map((line) => (line ? `${indentation}${line}` : ''))
    .join('\n');
  return `${header}${body}\n`;
}

function applyFixtureRuntimeRequirements(manifestContents, requirements) {
  if (typeof manifestContents !== 'string' || manifestContents.length === 0) {
    throw new Error('Rendered revision manifests must be a non-empty string.');
  }
  const validated = validateRequirements(requirements);
  if (!validated) return manifestContents;

  const { isMap, parseDocument } = require('yaml');
  const documents = parseDocuments(manifestContents);
  const configMap = findWorkflowControllerConfigMap(documents);
  const workflowDefaultsNode = configMap.getIn(['data', 'workflowDefaults'], true);
  if (typeof workflowDefaultsNode?.value !== 'string') {
    throw new Error('workflow-controller-configmap is missing string data.workflowDefaults.');
  }

  const workflowDefaults = parseDocument(workflowDefaultsNode.value, { maxAliasCount: 100 });
  if (workflowDefaults.errors.length > 0) {
    throw new Error(
      `workflow-controller-configmap data.workflowDefaults is invalid YAML: ${workflowDefaults.errors[0].message}`,
    );
  }
  const retryStrategy = workflowDefaults.getIn(['spec', 'templateDefaults', 'retryStrategy'], true);
  if (!isMap(retryStrategy)) {
    throw new Error(
      'workflow-controller-configmap data.workflowDefaults is missing spec.templateDefaults.retryStrategy.',
    );
  }
  workflowDefaults.setIn(
    ['spec', 'templateDefaults', 'retryStrategy', 'retryPolicy'],
    validated.argoRetryPolicy,
  );

  const replacement = serializeBlockScalar(workflowDefaultsNode, String(workflowDefaults));
  return `${manifestContents.slice(0, workflowDefaultsNode.range[0])}${replacement}${manifestContents.slice(
    workflowDefaultsNode.range[1],
  )}`;
}

module.exports = {
  applyFixtureRuntimeRequirements,
};
