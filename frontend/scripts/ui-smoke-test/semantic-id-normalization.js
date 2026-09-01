'use strict';

const SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION = 'ui-smoke-id-normalization/v1';
const SEMANTIC_ID_NORMALIZATION_MODES = Object.freeze({
  BROWSER_COMPATIBILITY: 'disabled-browser-compatibility',
  SEMANTIC_FULL_STACK: 'semantic-full-stack',
});
const SEMANTIC_ID_KINDS = Object.freeze([
  'artifact',
  'artifact-uri',
  'execution',
  'pod',
  'run',
  'task',
]);
const SEMANTIC_ID_PATH_PATTERN = /^[a-z0-9][a-z0-9.\/\[\]-]*$/;
const SEMANTIC_ID_TOKEN_PATTERN =
  /^\[ui-id:(?:artifact|artifact-uri|execution|pod|run|task):[a-z0-9][a-z0-9.:-]*\]$/;
const SEMANTIC_COLOR_PALETTE = Object.freeze([
  '#4285f4',
  '#2b9c1e',
  '#e00000',
  '#8026c0',
  '#9dafff',
  '#82c57a',
]);

function semanticIdNormalizationRenderingContract(mode) {
  if (!Object.values(SEMANTIC_ID_NORMALIZATION_MODES).includes(mode)) {
    throw new Error(`Unsupported semantic ID normalization mode ${mode || '(missing)'}.`);
  }
  return {
    derivedColorPalette: [...SEMANTIC_COLOR_PALETTE],
    failOnReplacementCountMismatch: true,
    mode,
    rawIdentifierPolicy: 'SHA-256 attestation only',
    schemaVersion: SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
    tokenFormat: '[ui-id:<kind>:<semantic-path>]',
  };
}

function semanticIdToken(kind, semanticId) {
  const compact = semanticId
    .replace(/^run\./, '')
    .replace(/\/(?:task|artifact|metric)\./g, '/')
    .replace(/\[(\d+)\]/g, '/$1')
    .replaceAll('/', ':');
  return `[ui-id:${kind}:${compact}]`;
}

module.exports = {
  SEMANTIC_COLOR_PALETTE,
  SEMANTIC_ID_KINDS,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
  SEMANTIC_ID_NORMALIZATION_MODES,
  SEMANTIC_ID_PATH_PATTERN,
  SEMANTIC_ID_TOKEN_PATTERN,
  semanticIdNormalizationRenderingContract,
  semanticIdToken,
};
