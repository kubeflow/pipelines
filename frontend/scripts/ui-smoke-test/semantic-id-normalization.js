'use strict';

const { createHash } = require('crypto');

const SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION = 'ui-smoke-id-normalization/v2';
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
  /^(?:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}|[1-9][0-9]{5}|[a-f0-9]{12,63}|s3:\/\/ui-smoke\/[a-f0-9]{16}\/artifact)$/;
const SEMANTIC_ID_SHAPE_PATTERN = /^(?:uuid|decimal|uri|text-(?:1[2-9]|[2-5][0-9]|6[0-3]))$/;
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
    tokenFormat: 'kind-shaped-sha256/v2',
  };
}

function semanticIdShape(kind, value) {
  if (kind === 'artifact-uri') return 'uri';
  if (/^[0-9]+$/.test(value)) return 'decimal';
  if (/^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/i.test(value)) return 'uuid';
  return `text-${Math.max(12, Math.min(63, String(value).length))}`;
}

function semanticIdToken(kind, semanticId, shape = kind === 'artifact-uri' ? 'uri' : 'uuid') {
  if (!SEMANTIC_ID_SHAPE_PATTERN.test(shape))
    throw new Error(`Invalid semantic token shape: ${shape}`);
  const digest = createHash('sha256').update(`${kind}\0${semanticId}`).digest('hex');
  if (shape === 'uri') return `s3://ui-smoke/${digest.slice(0, 16)}/artifact`;
  if (shape === 'decimal') return String(100000 + (parseInt(digest.slice(0, 12), 16) % 900000));
  if (shape.startsWith('text-')) return digest.slice(0, Number(shape.slice(5)));
  return `${digest.slice(0, 8)}-${digest.slice(8, 12)}-${digest.slice(12, 16)}-${digest.slice(16, 20)}-${digest.slice(20, 32)}`;
}

module.exports = {
  SEMANTIC_COLOR_PALETTE,
  SEMANTIC_ID_KINDS,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
  SEMANTIC_ID_NORMALIZATION_MODES,
  SEMANTIC_ID_PATH_PATTERN,
  SEMANTIC_ID_TOKEN_PATTERN,
  SEMANTIC_ID_SHAPE_PATTERN,
  semanticIdShape,
  semanticIdNormalizationRenderingContract,
  semanticIdToken,
};
