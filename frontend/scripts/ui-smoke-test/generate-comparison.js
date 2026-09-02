#!/usr/bin/env node
/**
 * Side-by-Side Comparison Generator
 *
 * Takes screenshots from base and head captures and creates side-by-side
 * comparison images with labels.
 *
 * Usage: node generate-comparison.js --main ./screenshots/main --pr ./screenshots/pr --output ./screenshots/comparison
 */

const looksSame = require('looks-same');
const sharp = require('sharp');
const crypto = require('crypto');
const path = require('path');
const fs = require('fs');
const {
  SCENARIO_CONTRACT_SCHEMA_VERSION,
  getGlobalVisualNormalizationContract,
  getSemanticIdNormalizationContract,
} = require('./semantic-capture-scenarios');
const {
  SEMANTIC_COLOR_PALETTE,
  SEMANTIC_ID_KINDS,
  SEMANTIC_ID_NORMALIZATION_MODES,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
  SEMANTIC_ID_PATH_PATTERN,
  SEMANTIC_ID_TOKEN_PATTERN,
  semanticIdNormalizationRenderingContract,
  semanticIdToken,
} = require('./semantic-id-normalization');
const { validateCombinedSemanticManifest } = require('./semantic-manifest');

const CAPTURE_MANIFEST_FILENAME = 'manifest.json';
const CAPTURE_MANIFEST_SCHEMA_VERSION = 3;
const COMPARISON_SUMMARY_FILENAME = 'summary.json';
const COMPARISON_SUMMARY_SCHEMA_VERSION = 2;
const COMPARISON_REPORT_FILENAME = 'report.html';
const MANAGED_OUTPUTS_FILENAME = '.managed-outputs.json';
const MANAGED_OUTPUTS_SCHEMA_VERSION = 2;
const MANAGED_STATIC_OUTPUTS = [COMPARISON_REPORT_FILENAME, COMPARISON_SUMMARY_FILENAME];
const SCENARIO_CONFIG_SCHEMA_VERSION = 'ui-smoke-comparison/v1';
const SCENARIO_POLICY_SCHEMA_VERSION = 'ui-smoke-comparison-policy/v1';
const MAX_SCENARIO_CONFIG_BYTES = 1024 * 1024;
const FRESHNESS_TOLERANCE_MS = 1000;
const CAPTURE_STATUSES = new Set(['success', 'degraded', 'skipped', 'failed']);
const CAPTURE_VALIDITIES = new Set([
  'valid',
  'missing_fixture',
  'expected_product_removal',
  'selector_drift',
  'ui_rendering_failure',
  'api_incompatibility',
  'seed_failure',
  'infrastructure_failure',
]);
const COMPARISON_ARGUMENT_NAMES = new Set([
  'diff-threshold',
  'fail-threshold',
  'looksame-cluster-size',
  'looksame-tolerance',
  'main',
  'main-label',
  'output',
  'pr',
  'pr-label',
  'scenario-config',
]);

const LABEL_HEIGHT = 40;
const LABEL_BACKGROUND = '#1a1a2e';
const LABEL_TEXT_COLOR = '#ffffff';
const DIVIDER_WIDTH = 4;
const REGION_BOX_PADDING = 4;
const DIFF_MARKER_COLOR = '#ff2b2b';
const DIFF_MARKER_WIDTH = 2;
const DIFF_MARKER_RADIUS = 4;
const MIN_REGION_AREA_PX = 12;
const MAX_HIGHLIGHT_REGIONS = 24;

class ComparisonError extends Error {
  constructor(message, failureType = 'comparison') {
    super(message);
    this.name = 'ComparisonError';
    this.failureType = failureType;
  }
}

function getArg(args, name, defaultValue) {
  const index = args.indexOf(`--${name}`);
  if (index === -1) {
    return defaultValue;
  }
  const value = args[index + 1];
  if (value === undefined || value.startsWith('--')) {
    throw new Error(`Missing value for --${name}`);
  }
  return value;
}

function validateCliArguments(args, allowedNames) {
  const seen = new Set();
  for (let index = 0; index < args.length; index += 2) {
    const token = args[index];
    if (typeof token !== 'string' || !token.startsWith('--')) {
      throw new Error(`Unexpected positional argument: ${token}`);
    }
    const name = token.slice(2);
    if (!allowedNames.has(name)) {
      throw new Error(`Unknown argument: ${token}`);
    }
    if (seen.has(name)) {
      throw new Error(`Duplicate argument: ${token}`);
    }
    const value = args[index + 1];
    if (value === undefined || value.startsWith('--')) {
      throw new Error(`Missing value for ${token}`);
    }
    seen.add(name);
  }
}

function parseComparisonOptions(args = process.argv.slice(2), env = process.env) {
  validateCliArguments(args, COMPARISON_ARGUMENT_NAMES);
  const failThresholdRaw = getArg(
    args,
    'fail-threshold',
    env.UI_SMOKE_FAIL_THRESHOLD === undefined ? '0' : env.UI_SMOKE_FAIL_THRESHOLD,
  );

  return {
    diffThreshold: Number(getArg(args, 'diff-threshold', env.UI_SMOKE_DIFF_THRESHOLD || '0')),
    failThreshold: failThresholdRaw === '' ? null : Number(failThresholdRaw),
    failThresholdRaw,
    looksSameClusterSize: Number(
      getArg(args, 'looksame-cluster-size', env.UI_SMOKE_LOOKSAME_CLUSTER_SIZE || '8'),
    ),
    looksSameTolerance: Number(
      getArg(args, 'looksame-tolerance', env.UI_SMOKE_LOOKSAME_TOLERANCE || '2.3'),
    ),
    mainDir: getArg(args, 'main', './screenshots/main'),
    mainLabel: getArg(args, 'main-label', null),
    outputDir: getArg(args, 'output', './screenshots/comparison'),
    prDir: getArg(args, 'pr', './screenshots/pr'),
    prLabel: getArg(args, 'pr-label', null),
    scenarioConfigPath: getArg(args, 'scenario-config', env.UI_SMOKE_SCENARIO_CONFIG || null),
  };
}

function validateComparisonOptions(options) {
  const errors = [];
  if (
    !Number.isFinite(options.diffThreshold) ||
    options.diffThreshold < 0 ||
    options.diffThreshold > 100
  ) {
    errors.push(`Invalid diff threshold: ${options.diffThreshold}`);
  }
  if (
    options.failThreshold !== null &&
    (!Number.isFinite(options.failThreshold) ||
      options.failThreshold < 0 ||
      options.failThreshold > 100)
  ) {
    errors.push(`Invalid fail threshold: ${options.failThresholdRaw}`);
  }
  if (
    !Number.isFinite(options.looksSameTolerance) ||
    options.looksSameTolerance < 0 ||
    options.looksSameTolerance > 100
  ) {
    errors.push(`Invalid looks-same tolerance: ${options.looksSameTolerance}`);
  }
  if (
    !Number.isSafeInteger(options.looksSameClusterSize) ||
    options.looksSameClusterSize <= 0 ||
    options.looksSameClusterSize > 1000
  ) {
    errors.push(`Invalid looks-same cluster size: ${options.looksSameClusterSize}`);
  }
  return errors;
}

function isPlainObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function requireBoundedString(value, label, maxLength = 2048) {
  if (typeof value !== 'string' || value.trim() === '' || value.length > maxLength) {
    throw new ComparisonError(
      `${label} must be a non-empty string of at most ${maxLength} characters.`,
      'config',
    );
  }
  return value;
}

function validateThreshold(value, label, allowNull = false) {
  if (allowNull && value === null) return value;
  if (!Number.isFinite(value) || value < 0 || value > 100) {
    throw new ComparisonError(
      `${label} must be a number from 0 through 100${allowNull ? ' or null' : ''}.`,
      'config',
    );
  }
  return value;
}

function canonicalizeJson(value) {
  if (Array.isArray(value)) {
    return value.map(canonicalizeJson);
  }
  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((key) => [key, canonicalizeJson(value[key])]),
    );
  }
  return value;
}

function normalizeExpectedChange(value, label) {
  if (value === undefined || value === null || value === '') return null;
  if (typeof value === 'string') {
    return requireBoundedString(value, label, 4000);
  }
  if (!isPlainObject(value) && !Array.isArray(value)) {
    throw new ComparisonError(`${label} must be a string, object, array, or null.`, 'config');
  }
  const normalized = canonicalizeJson(value);
  if (JSON.stringify(normalized).length > 16_000) {
    throw new ComparisonError(`${label} is too large.`, 'config');
  }
  return normalized;
}

function normalizeMask(mask, label) {
  if (!isPlainObject(mask)) {
    throw new ComparisonError(`${label} must be an object.`, 'config');
  }
  const allowedFields = new Set(['height', 'reason', 'width', 'x', 'y']);
  for (const key of Object.keys(mask)) {
    if (!allowedFields.has(key)) {
      throw new ComparisonError(`${label} has unknown field ${key}.`, 'config');
    }
  }
  for (const coordinate of ['x', 'y', 'width', 'height']) {
    if (!Number.isSafeInteger(mask[coordinate]) || mask[coordinate] < 0) {
      throw new ComparisonError(
        `${label}.${coordinate} must be a non-negative safe integer.`,
        'config',
      );
    }
  }
  if (mask.width === 0 || mask.height === 0) {
    throw new ComparisonError(`${label} width and height must be greater than zero.`, 'config');
  }
  return {
    x: mask.x,
    y: mask.y,
    width: mask.width,
    height: mask.height,
    ...(mask.reason === undefined
      ? {}
      : { reason: requireBoundedString(mask.reason, `${label}.reason`, 1000) }),
  };
}

function normalizeScenarioRule(rule, index) {
  const label = `Scenario config scenarios[${index}]`;
  if (!isPlainObject(rule)) {
    throw new ComparisonError(`${label} must be an object.`, 'config');
  }
  const allowedFields = new Set([
    'diffThreshold',
    'expectedChange',
    'failThreshold',
    'looksSameTolerance',
    'masks',
    'semanticScenario',
    'viewport',
  ]);
  for (const key of Object.keys(rule)) {
    if (!allowedFields.has(key)) {
      throw new ComparisonError(`${label} has unknown field ${key}.`, 'config');
    }
  }
  const semanticScenario = requireBoundedString(
    rule.semanticScenario,
    `${label}.semanticScenario`,
    200,
  );
  if (!/^[a-z0-9][a-z0-9-]*$/.test(semanticScenario)) {
    throw new ComparisonError(
      `${label}.semanticScenario must use lowercase letters, numbers, and hyphens.`,
      'config',
    );
  }
  let viewport = null;
  if (rule.viewport !== undefined && rule.viewport !== null) {
    if (
      !isPlainObject(rule.viewport) ||
      !Number.isSafeInteger(rule.viewport.width) ||
      rule.viewport.width <= 0 ||
      !Number.isSafeInteger(rule.viewport.height) ||
      rule.viewport.height <= 0 ||
      Object.keys(rule.viewport).some((key) => key !== 'width' && key !== 'height')
    ) {
      throw new ComparisonError(
        `${label}.viewport must contain positive integer width and height.`,
        'config',
      );
    }
    viewport = { width: rule.viewport.width, height: rule.viewport.height };
  }
  const masksProvided = Object.hasOwn(rule, 'masks');
  if (masksProvided && (!Array.isArray(rule.masks) || rule.masks.length > 100)) {
    throw new ComparisonError(`${label}.masks must be an array of at most 100 masks.`, 'config');
  }
  const masks = (masksProvided ? rule.masks : []).map((mask, maskIndex) =>
    normalizeMask(mask, `${label}.masks[${maskIndex}]`),
  );
  const maskKeys = new Set();
  for (const mask of masks) {
    const key = `${mask.x}:${mask.y}:${mask.width}:${mask.height}`;
    if (maskKeys.has(key)) {
      throw new ComparisonError(`${label} contains duplicate mask ${key}.`, 'config');
    }
    maskKeys.add(key);
  }
  masks.sort(
    (left, right) =>
      left.y - right.y ||
      left.x - right.x ||
      left.height - right.height ||
      left.width - right.width ||
      String(left.reason || '').localeCompare(String(right.reason || '')),
  );

  return {
    semanticScenario,
    viewport,
    ...(rule.diffThreshold === undefined
      ? {}
      : { diffThreshold: validateThreshold(rule.diffThreshold, `${label}.diffThreshold`) }),
    ...(rule.failThreshold === undefined
      ? {}
      : {
          failThreshold: validateThreshold(rule.failThreshold, `${label}.failThreshold`, true),
        }),
    ...(rule.looksSameTolerance === undefined
      ? {}
      : {
          looksSameTolerance: validateThreshold(
            rule.looksSameTolerance,
            `${label}.looksSameTolerance`,
          ),
        }),
    ...(masksProvided ? { masks } : {}),
    ...(Object.hasOwn(rule, 'expectedChange')
      ? {
          expectedChange: normalizeExpectedChange(rule.expectedChange, `${label}.expectedChange`),
        }
      : {}),
  };
}

function validateRevisionBinding(binding, actual, label) {
  if (!isPlainObject(binding)) {
    throw new ComparisonError(`${label} must be an object.`, 'config');
  }
  const allowedFields = new Set(['captureId', 'label', 'manifestSha256']);
  for (const key of Object.keys(binding)) {
    if (!allowedFields.has(key)) {
      throw new ComparisonError(`${label} has unknown field ${key}.`, 'config');
    }
  }
  if (Object.keys(binding).length === 0) {
    throw new ComparisonError(`${label} must bind at least one revision identity.`, 'config');
  }
  for (const key of Object.keys(binding)) {
    const expected = requireBoundedString(binding[key], `${label}.${key}`);
    if (key === 'manifestSha256' && !/^[a-f0-9]{64}$/.test(expected)) {
      throw new ComparisonError(
        `${label}.manifestSha256 must be a lowercase SHA-256 digest.`,
        'config',
      );
    }
    if (actual[key] !== expected) {
      throw new ComparisonError(
        `${label}.${key} does not match the ${label.endsWith('base') ? 'base' : 'head'} capture.`,
        'config',
      );
    }
  }
}

function normalizeOperatorPolicyAttestation(value) {
  if (value === undefined) return null;
  if (!isPlainObject(value) || typeof value.applied !== 'boolean') {
    throw new ComparisonError('Scenario config operatorPolicy must declare applied.', 'config');
  }
  const allowedFields = value.applied
    ? new Set(['applied', 'schemaVersion', 'sha256', 'sizeBytes'])
    : new Set(['applied']);
  if (Object.keys(value).some((key) => !allowedFields.has(key))) {
    throw new ComparisonError('Scenario config operatorPolicy has unknown fields.', 'config');
  }
  if (!value.applied) return { applied: false };
  if (
    value.schemaVersion !== SCENARIO_POLICY_SCHEMA_VERSION ||
    typeof value.sha256 !== 'string' ||
    !/^[a-f0-9]{64}$/.test(value.sha256) ||
    !Number.isSafeInteger(value.sizeBytes) ||
    value.sizeBytes < 1 ||
    value.sizeBytes > MAX_SCENARIO_CONFIG_BYTES
  ) {
    throw new ComparisonError(
      'Applied scenario config operatorPolicy must contain a valid schema, SHA-256, and size.',
      'config',
    );
  }
  return {
    applied: true,
    schemaVersion: value.schemaVersion,
    sha256: value.sha256,
    sizeBytes: value.sizeBytes,
  };
}

function loadScenarioConfig(configPath, captureContext) {
  if (!configPath) {
    return { attestation: null, rules: [] };
  }
  let stat;
  try {
    stat = fs.lstatSync(configPath);
  } catch (error) {
    throw new ComparisonError(
      `Unable to read scenario config ${configPath}: ${error.message}`,
      'config',
    );
  }
  if (!stat.isFile() || stat.isSymbolicLink() || stat.size > MAX_SCENARIO_CONFIG_BYTES) {
    throw new ComparisonError(
      `Scenario config must be a regular non-symlink file no larger than ${MAX_SCENARIO_CONFIG_BYTES} bytes.`,
      'config',
    );
  }
  const contents = fs.readFileSync(configPath);
  let config;
  try {
    config = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new ComparisonError(
      `Unable to parse scenario config ${configPath}: ${error.message}`,
      'config',
    );
  }
  if (!isPlainObject(config) || config.schemaVersion !== SCENARIO_CONFIG_SCHEMA_VERSION) {
    throw new ComparisonError(
      `Scenario config must use schema version ${SCENARIO_CONFIG_SCHEMA_VERSION}.`,
      'config',
    );
  }
  const allowedFields = new Set([
    'operatorPolicy',
    'revisionPair',
    'scenarioContractSchemaVersion',
    'scenarios',
    'schemaVersion',
    'viewports',
  ]);
  for (const key of Object.keys(config)) {
    if (!allowedFields.has(key)) {
      throw new ComparisonError(`Scenario config has unknown field ${key}.`, 'config');
    }
  }
  if (!isPlainObject(config.revisionPair)) {
    throw new ComparisonError('Scenario config revisionPair must be an object.', 'config');
  }
  if (Object.keys(config.revisionPair).some((key) => key !== 'base' && key !== 'head')) {
    throw new ComparisonError(
      'Scenario config revisionPair may contain only base and head.',
      'config',
    );
  }
  validateRevisionBinding(config.revisionPair.base, captureContext.base, 'revisionPair.base');
  validateRevisionBinding(config.revisionPair.head, captureContext.head, 'revisionPair.head');
  if (
    config.scenarioContractSchemaVersion !== undefined &&
    (config.scenarioContractSchemaVersion !== captureContext.base.scenarioContractSchemaVersion ||
      config.scenarioContractSchemaVersion !== captureContext.head.scenarioContractSchemaVersion)
  ) {
    throw new ComparisonError(
      'Scenario config semantic scenario contract does not match both captures.',
      'config',
    );
  }
  const viewports =
    config.viewports === undefined
      ? []
      : normalizeExpectedViewports(
          config.viewports,
          { viewports: config.viewports },
          { viewports: config.viewports },
        );
  const operatorPolicy = normalizeOperatorPolicyAttestation(config.operatorPolicy);
  if (
    !Array.isArray(config.scenarios) ||
    config.scenarios.length === 0 ||
    config.scenarios.length > 500
  ) {
    throw new ComparisonError(
      'Scenario config scenarios must contain between 1 and 500 rules.',
      'config',
    );
  }
  const rules = config.scenarios.map(normalizeScenarioRule);
  const ruleKeys = new Set();
  for (const rule of rules) {
    const key = `${rule.semanticScenario}@${rule.viewport ? `${rule.viewport.width}x${rule.viewport.height}` : '*'}`;
    if (ruleKeys.has(key)) {
      throw new ComparisonError(`Scenario config contains duplicate rule ${key}.`, 'config');
    }
    ruleKeys.add(key);
  }
  rules.sort((left, right) => {
    const leftKey = `${left.semanticScenario}@${left.viewport ? `${left.viewport.width}x${left.viewport.height}` : '*'}`;
    const rightKey = `${right.semanticScenario}@${right.viewport ? `${right.viewport.width}x${right.viewport.height}` : '*'}`;
    return leftKey.localeCompare(rightKey);
  });
  return {
    attestation: {
      revisionPair: canonicalizeJson(config.revisionPair),
      operatorPolicy,
      scenarioContractSchemaVersion: config.scenarioContractSchemaVersion ?? null,
      schemaVersion: SCENARIO_CONFIG_SCHEMA_VERSION,
      sha256: crypto.createHash('sha256').update(contents).digest('hex'),
      sizeBytes: contents.length,
      viewports,
    },
    rules,
  };
}

function readCaptureIdentity(directory, role) {
  const manifestPath = path.join(directory, CAPTURE_MANIFEST_FILENAME);
  let stat;
  try {
    stat = fs.lstatSync(manifestPath);
  } catch (error) {
    throw new ComparisonError(
      `Unable to read ${role} capture manifest: ${error.message}`,
      'manifest',
    );
  }
  if (!stat.isFile() || stat.isSymbolicLink() || stat.size > MAX_SCENARIO_CONFIG_BYTES * 10) {
    throw new ComparisonError(
      `${role} capture manifest must be a bounded regular non-symlink file.`,
      'manifest',
    );
  }
  const contents = fs.readFileSync(manifestPath);
  let manifest;
  try {
    manifest = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new ComparisonError(
      `Unable to parse ${role} capture manifest: ${error.message}`,
      'manifest',
    );
  }
  if (
    !isPlainObject(manifest) ||
    manifest.schemaVersion !== CAPTURE_MANIFEST_SCHEMA_VERSION ||
    typeof manifest.captureId !== 'string' ||
    manifest.captureId === '' ||
    !Array.isArray(manifest.results)
  ) {
    throw new ComparisonError(
      `${role} capture manifest cannot bind a scenario config.`,
      'manifest',
    );
  }
  validateCaptureManifest(manifest, manifestPath);
  return {
    captureId: manifest.captureId,
    contents,
    label: typeof manifest.label === 'string' && manifest.label ? manifest.label : null,
    manifest,
    manifestSha256: crypto.createHash('sha256').update(contents).digest('hex'),
  };
}

function loadScenarioPolicy(policyPath) {
  if (!policyPath) return { attestation: { applied: false }, rules: [] };
  let stat;
  try {
    stat = fs.lstatSync(policyPath);
  } catch (error) {
    throw new ComparisonError(
      `Unable to read scenario policy ${policyPath}: ${error.message}`,
      'config',
    );
  }
  if (!stat.isFile() || stat.isSymbolicLink() || stat.size > MAX_SCENARIO_CONFIG_BYTES) {
    throw new ComparisonError(
      'Scenario policy must be a bounded regular non-symlink file.',
      'config',
    );
  }
  const contents = fs.readFileSync(policyPath);
  let policy;
  try {
    policy = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new ComparisonError(
      `Unable to parse scenario policy ${policyPath}: ${error.message}`,
      'config',
    );
  }
  if (
    !isPlainObject(policy) ||
    policy.schemaVersion !== SCENARIO_POLICY_SCHEMA_VERSION ||
    !Array.isArray(policy.scenarios) ||
    Object.keys(policy).some((key) => key !== 'schemaVersion' && key !== 'scenarios')
  ) {
    throw new ComparisonError(
      `Scenario policy must use schema version ${SCENARIO_POLICY_SCHEMA_VERSION} and contain scenarios.`,
      'config',
    );
  }
  const rules = policy.scenarios.map(normalizeScenarioRule);
  const keys = new Set();
  for (const rule of rules) {
    const key = `${rule.semanticScenario}@${rule.viewport ? `${rule.viewport.width}x${rule.viewport.height}` : '*'}`;
    if (keys.has(key)) {
      throw new ComparisonError(`Scenario policy contains duplicate rule ${key}.`, 'config');
    }
    keys.add(key);
  }
  return {
    attestation: {
      applied: true,
      schemaVersion: SCENARIO_POLICY_SCHEMA_VERSION,
      sha256: crypto.createHash('sha256').update(contents).digest('hex'),
      sizeBytes: contents.length,
    },
    rules,
  };
}

function normalizeScenarioCatalog(scenarioCatalog) {
  const catalog = new Map();
  for (const [index, entry] of (scenarioCatalog || []).entries()) {
    if (!isPlainObject(entry)) {
      throw new ComparisonError(
        `Semantic scenario catalog entry ${index} must be an object.`,
        'config',
      );
    }
    const semanticScenario = requireBoundedString(
      entry.semanticScenario,
      `Semantic scenario catalog entry ${index}.semanticScenario`,
      200,
    );
    if (!/^[a-z0-9][a-z0-9-]*$/.test(semanticScenario)) {
      throw new ComparisonError(
        `Semantic scenario catalog entry ${index}.semanticScenario is invalid.`,
        'config',
      );
    }
    if (catalog.has(semanticScenario)) {
      throw new ComparisonError(
        `Semantic scenario catalog contains duplicate scenario ${semanticScenario}.`,
        'config',
      );
    }
    if (typeof entry.required !== 'boolean') {
      throw new ComparisonError(
        `Semantic scenario catalog entry ${index}.required must be a boolean.`,
        'config',
      );
    }
    catalog.set(semanticScenario, {
      expectedChange: normalizeExpectedChange(
        entry.expectedChange,
        `Semantic scenario catalog entry ${index}.expectedChange`,
      ),
      required: entry.required,
    });
  }
  return catalog;
}

function normalizeExpectedViewports(value, baseManifest, headManifest) {
  const source = value || baseManifest.viewports || headManifest.viewports;
  const entries =
    typeof source === 'string' ? source.split(',').map((item) => item.trim()) : source;
  if (!Array.isArray(entries) || entries.length === 0 || entries.length > 20) {
    throw new ComparisonError(
      'Expected viewports must contain between 1 and 20 entries.',
      'config',
    );
  }
  const viewports = entries.map((entry, index) => {
    if (typeof entry === 'string') {
      const match = /^([1-9]\d*)x([1-9]\d*)$/.exec(entry);
      if (match) return { width: Number(match[1]), height: Number(match[2]) };
    } else if (
      isPlainObject(entry) &&
      Number.isSafeInteger(entry.width) &&
      entry.width > 0 &&
      Number.isSafeInteger(entry.height) &&
      entry.height > 0
    ) {
      return { width: entry.width, height: entry.height };
    }
    throw new ComparisonError(`Expected viewport ${index} is invalid.`, 'config');
  });
  const keys = new Set();
  for (const viewport of viewports) {
    const key = `${viewport.width}x${viewport.height}`;
    if (keys.has(key)) {
      throw new ComparisonError(`Expected viewports contain duplicate ${key}.`, 'config');
    }
    keys.add(key);
  }
  return viewports.sort((left, right) => left.width - right.width || left.height - right.height);
}

function validateRequiredScenarioCoverage(baseIdentity, headIdentity, catalog, viewports) {
  const matches = (manifest, semanticScenario, viewport) =>
    manifest.results.find(
      (result) =>
        (result.semanticScenario || result.page) === semanticScenario &&
        result.viewport?.width === viewport.width &&
        result.viewport?.height === viewport.height,
    );
  for (const [semanticScenario, contract] of catalog) {
    if (!contract.required) continue;
    for (const viewport of viewports) {
      const key = `${viewport.width}x${viewport.height}`;
      const base = matches(baseIdentity.manifest, semanticScenario, viewport);
      const head = matches(headIdentity.manifest, semanticScenario, viewport);
      if (!base || !head) {
        throw new ComparisonError(
          `Required semantic scenario ${semanticScenario} is missing viewport ${key} from one or both capture manifests.`,
          'config',
        );
      }
      if (base.required !== true || head.required !== true) {
        throw new ComparisonError(
          `Required semantic scenario ${semanticScenario} viewport ${key} must remain required in both capture manifests.`,
          'config',
        );
      }
    }
  }
}

function manifestScenarioDefaults(baseIdentity, headIdentity, defaults, scenarioCatalog = []) {
  const records = new Map();
  const catalog = normalizeScenarioCatalog(scenarioCatalog);
  for (const [role, identity] of [
    ['base', baseIdentity],
    ['head', headIdentity],
  ]) {
    for (const result of identity.manifest.results) {
      if (!isPlainObject(result)) continue;
      const semanticScenario = result.semanticScenario || result.page;
      if (typeof semanticScenario !== 'string' || !/^[a-z0-9][a-z0-9-]*$/.test(semanticScenario)) {
        continue;
      }
      const current = records.get(semanticScenario) || { base: null, head: null };
      current[role] = result;
      records.set(semanticScenario, current);
    }
  }
  for (const [semanticScenario, contract] of catalog) {
    if (!contract.required) continue;
    const record = records.get(semanticScenario);
    if (!record?.base || !record?.head) {
      throw new ComparisonError(
        `Required semantic scenario ${semanticScenario} is missing from one or both capture manifests.`,
        'config',
      );
    }
    if (record.base.required !== true || record.head.required !== true) {
      throw new ComparisonError(
        `Required semantic scenario ${semanticScenario} must remain required in both capture manifests.`,
        'config',
      );
    }
  }
  return [...records.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([semanticScenario, record]) => {
      const claimProblems = [
        expectedRemovalClaimError(record.base, 'base'),
        expectedRemovalClaimError(record.head, 'head'),
      ].filter(Boolean);
      if (claimProblems.length > 0 && (record.base?.required || record.head?.required)) {
        throw new ComparisonError(`${semanticScenario}: ${claimProblems.join('; ')}`, 'config');
      }
      const expectedRemoval =
        record.head?.captureValidity === 'expected_product_removal' && claimProblems.length === 0;
      const catalogExpected = catalog.get(semanticScenario)?.expectedChange ?? null;
      const baseExpected = normalizeExpectedChange(
        record.base?.expectedChange ?? catalogExpected,
        `${semanticScenario} base expectedChange`,
      );
      const headExpected = normalizeExpectedChange(
        record.head?.expectedChange ?? catalogExpected,
        `${semanticScenario} head expectedChange`,
      );
      const expectedChange =
        JSON.stringify(baseExpected) === JSON.stringify(headExpected)
          ? baseExpected
          : { base: baseExpected, head: headExpected };
      return normalizeScenarioRule(
        {
          semanticScenario,
          diffThreshold: defaults.diffThreshold,
          failThreshold: expectedRemoval ? null : defaults.failThreshold,
          looksSameTolerance: defaults.looksSameTolerance,
          expectedChange,
          masks: [],
        },
        semanticScenario,
      );
    });
}

function mergeScenarioPolicies(defaultRules, overrideRules) {
  const byKey = new Map(defaultRules.map((rule) => [`${rule.semanticScenario}@*`, rule]));
  for (const override of overrideRules) {
    const key = `${override.semanticScenario}@${override.viewport ? `${override.viewport.width}x${override.viewport.height}` : '*'}`;
    if (override.viewport || !byKey.has(key)) {
      byKey.set(key, override);
      continue;
    }
    const baseline = byKey.get(key);
    byKey.set(key, {
      ...baseline,
      ...override,
      expectedChange: Object.hasOwn(override, 'expectedChange')
        ? override.expectedChange
        : baseline.expectedChange,
      masks: Object.hasOwn(override, 'masks') ? override.masks : baseline.masks,
    });
  }
  return [...byKey.values()].sort((left, right) => {
    const leftKey = `${left.semanticScenario}@${left.viewport ? `${left.viewport.width}x${left.viewport.height}` : '*'}`;
    const rightKey = `${right.semanticScenario}@${right.viewport ? `${right.viewport.width}x${right.viewport.height}` : '*'}`;
    return leftKey.localeCompare(rightKey);
  });
}

function writeBoundScenarioConfig({
  baseDir,
  defaults,
  headDir,
  outputPath,
  policyPath = null,
  scenarioCatalog = [],
  expectedViewports = null,
}) {
  const base = readCaptureIdentity(baseDir, 'Base');
  const head = readCaptureIdentity(headDir, 'Head');
  const provenance = validateCapturePairProvenance(base.manifest, head.manifest);
  const catalog = normalizeScenarioCatalog(scenarioCatalog);
  const viewports = normalizeExpectedViewports(expectedViewports, base.manifest, head.manifest);
  validateRequiredScenarioCoverage(base, head, catalog, viewports);
  const defaultRules = manifestScenarioDefaults(base, head, defaults, scenarioCatalog);
  if (defaultRules.length === 0) {
    throw new ComparisonError(
      'Capture manifests contain no semantic scenarios to configure.',
      'config',
    );
  }
  const operatorPolicy = loadScenarioPolicy(policyPath);
  const scenarios = mergeScenarioPolicies(defaultRules, operatorPolicy.rules);
  const binding = (identity) => ({
    captureId: identity.captureId,
    ...(identity.label ? { label: identity.label } : {}),
    manifestSha256: identity.manifestSha256,
  });
  const config = {
    schemaVersion: SCENARIO_CONFIG_SCHEMA_VERSION,
    operatorPolicy: operatorPolicy.attestation,
    revisionPair: {
      base: binding(base),
      head: binding(head),
    },
    scenarioContractSchemaVersion: provenance.scenarioContractSchemaVersion,
    scenarios,
    viewports,
  };
  const parent = path.dirname(outputPath);
  const parentStat = fs.lstatSync(parent);
  if (!parentStat.isDirectory() || parentStat.isSymbolicLink()) {
    throw new ComparisonError(
      'Scenario config output parent must be a non-symlink directory.',
      'config',
    );
  }
  const contents = Buffer.from(`${JSON.stringify(config, null, 2)}\n`);
  fs.writeFileSync(outputPath, contents, { flag: 'wx' });
  return {
    config,
    path: outputPath,
    sha256: crypto.createHash('sha256').update(contents).digest('hex'),
    sizeBytes: contents.length,
  };
}

function canonicalDirectoryPath(directory) {
  let existing = path.resolve(directory);
  const missing = [];
  while (!fs.existsSync(existing)) {
    const parent = path.dirname(existing);
    if (parent === existing) break;
    missing.unshift(path.basename(existing));
    existing = parent;
  }
  return path.join(fs.realpathSync(existing), ...missing);
}

function validateDistinctDirectories(options) {
  const directories = [
    ['--main', canonicalDirectoryPath(options.mainDir)],
    ['--pr', canonicalDirectoryPath(options.prDir)],
    ['--output', canonicalDirectoryPath(options.outputDir)],
  ];
  for (let left = 0; left < directories.length; left++) {
    for (let right = left + 1; right < directories.length; right++) {
      if (directories[left][1] === directories[right][1]) {
        throw new Error(
          `${directories[left][0]} and ${directories[right][0]} must resolve to distinct directories.`,
        );
      }
    }
  }
  return true;
}

function maxRegionsForDiff(diffPercent) {
  if (!Number.isFinite(diffPercent)) {
    return 10;
  }
  if (diffPercent < 1) {
    return 16;
  }
  if (diffPercent < 3) {
    return 20;
  }
  return MAX_HIGHLIGHT_REGIONS;
}

function escapeXml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function escapeHtml(value) {
  return escapeXml(value).replace(/'/g, '&#39;');
}

async function createLabeledImage(imagePath, label, width, height) {
  const safeLabel = escapeXml(label);
  const fontSize = label.length > 30 ? 13 : 16;
  const labelSvg = `
    <svg width="${width}" height="${LABEL_HEIGHT}">
      <rect width="100%" height="100%" fill="${LABEL_BACKGROUND}"/>
      <text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle"
            font-family="Arial, sans-serif" font-size="${fontSize}" font-weight="bold" fill="${LABEL_TEXT_COLOR}">
        ${safeLabel}
      </text>
    </svg>
  `;

  const resizedImage = sharp(imagePath).resize(width, height, {
    fit: 'contain',
    background: { r: 245, g: 245, b: 245, alpha: 1 },
  });

  return sharp({
    create: {
      width,
      height: height + LABEL_HEIGHT,
      channels: 4,
      background: { r: 255, g: 255, b: 255, alpha: 1 },
    },
  })
    .composite([
      { input: Buffer.from(labelSvg), top: 0, left: 0 },
      { input: await resizedImage.toBuffer(), top: LABEL_HEIGHT, left: 0 },
    ])
    .png();
}

async function createDivider(height) {
  return sharp({
    create: {
      width: DIVIDER_WIDTH,
      height,
      channels: 4,
      background: { r: 74, g: 78, b: 105, alpha: 1 },
    },
  }).png();
}

function normalizeRegion(region, imageWidth, imageHeight) {
  if (!region) {
    return null;
  }

  const left = Number(region.left);
  const top = Number(region.top);
  const right = Number(region.right);
  const bottom = Number(region.bottom);
  if (![left, top, right, bottom].every(Number.isFinite)) {
    return null;
  }

  const clampedLeft = Math.max(0, Math.min(imageWidth - 1, Math.floor(left)));
  const clampedTop = Math.max(0, Math.min(imageHeight - 1, Math.floor(top)));
  const clampedRight = Math.max(clampedLeft, Math.min(imageWidth - 1, Math.ceil(right)));
  const clampedBottom = Math.max(clampedTop, Math.min(imageHeight - 1, Math.ceil(bottom)));
  const width = clampedRight - clampedLeft + 1;
  const height = clampedBottom - clampedTop + 1;

  if (width <= 0 || height <= 0) {
    return null;
  }
  return { x: clampedLeft, y: clampedTop, width, height };
}

function extractDiffRegions(looksSameResult, imageWidth, imageHeight, diffPercent) {
  const clusters = Array.isArray(looksSameResult?.diffClusters) ? looksSameResult.diffClusters : [];
  let regions = clusters
    .map((cluster) => normalizeRegion(cluster, imageWidth, imageHeight))
    .filter(Boolean);

  if (regions.length === 0 && looksSameResult?.diffBounds) {
    const bounds = normalizeRegion(looksSameResult.diffBounds, imageWidth, imageHeight);
    if (bounds) {
      regions = [bounds];
    }
  }

  const filtered = regions.filter((region) => region.width * region.height >= MIN_REGION_AREA_PX);
  const withArea = (filtered.length > 0 ? filtered : regions).map((region) => ({
    ...region,
    area: region.width * region.height,
  }));

  return withArea
    .sort((a, b) => b.area - a.area)
    .slice(0, maxRegionsForDiff(diffPercent))
    .map(({ area, ...region }) => region);
}

function maskUnionArea(masks) {
  if (masks.length === 0) return 0;
  const xCoordinates = [...new Set(masks.flatMap((mask) => [mask.x, mask.x + mask.width]))].sort(
    (left, right) => left - right,
  );
  let area = 0;
  for (let index = 0; index < xCoordinates.length - 1; index += 1) {
    const left = xCoordinates[index];
    const right = xCoordinates[index + 1];
    const intervals = masks
      .filter((mask) => mask.x < right && mask.x + mask.width > left)
      .map((mask) => [mask.y, mask.y + mask.height])
      .sort((first, second) => first[0] - second[0] || first[1] - second[1]);
    let coveredHeight = 0;
    let currentStart = null;
    let currentEnd = null;
    for (const [start, end] of intervals) {
      if (currentStart === null) {
        currentStart = start;
        currentEnd = end;
      } else if (start <= currentEnd) {
        currentEnd = Math.max(currentEnd, end);
      } else {
        coveredHeight += currentEnd - currentStart;
        currentStart = start;
        currentEnd = end;
      }
    }
    if (currentStart !== null) coveredHeight += currentEnd - currentStart;
    area += (right - left) * coveredHeight;
  }
  return area;
}

function validateMasksForImage(masks, width, height) {
  for (const [index, mask] of masks.entries()) {
    if (mask.x + mask.width > width || mask.y + mask.height > height) {
      throw new ComparisonError(
        `Scenario mask ${index} (${mask.x},${mask.y},${mask.width},${mask.height}) exceeds the ${width}x${height} viewport.`,
        'config',
      );
    }
  }
  const maskedPixels = maskUnionArea(masks);
  if (maskedPixels >= width * height) {
    throw new ComparisonError('Scenario masks must leave at least one comparable pixel.', 'config');
  }
  return maskedPixels;
}

async function applyMasks(image, masks) {
  if (masks.length === 0) return image;
  return sharp(image)
    .composite(
      masks.map((mask) => ({
        input: {
          create: {
            width: mask.width,
            height: mask.height,
            channels: 4,
            background: { r: 17, g: 17, b: 17, alpha: 1 },
          },
        },
        left: mask.x,
        top: mask.y,
      })),
    )
    .png()
    .toBuffer();
}

function deriveDiffPercent(looksSameResult, width, height, maskedPixels = 0) {
  const comparablePixels = width * height - maskedPixels;
  if (
    Number.isFinite(looksSameResult?.differentPixels) &&
    Number.isFinite(looksSameResult?.totalPixels) &&
    looksSameResult.totalPixels > 0
  ) {
    if (looksSameResult.differentPixels > comparablePixels) return null;
    return (looksSameResult.differentPixels / comparablePixels) * 100;
  }

  if (maskedPixels === 0 && Number.isFinite(looksSameResult?.diffPercentage)) {
    return looksSameResult.diffPercentage;
  }

  if (looksSameResult?.equal === true) {
    return 0;
  }

  if (looksSameResult?.diffBounds && width > 0 && height > 0) {
    const bounds = normalizeRegion(looksSameResult.diffBounds, width, height);
    if (bounds) {
      return ((bounds.width * bounds.height) / comparablePixels) * 100;
    }
  }

  return null;
}

async function analyzeDiff(mainImage, prImage, options, compareImages = looksSame) {
  if (
    (typeof mainImage === 'string' && !fs.existsSync(mainImage)) ||
    (typeof prImage === 'string' && !fs.existsSync(prImage))
  ) {
    throw new ComparisonError('A required screenshot is missing.', 'missing');
  }

  let mainMeta;
  let prMeta;
  try {
    [mainMeta, prMeta] = await Promise.all([
      sharp(mainImage).metadata(),
      sharp(prImage).metadata(),
    ]);
  } catch (error) {
    throw new ComparisonError(`Unable to read screenshot: ${error.message}`, 'corrupt');
  }

  const mainWidth = mainMeta.width || 0;
  const mainHeight = mainMeta.height || 0;
  const prWidth = prMeta.width || 0;
  const prHeight = prMeta.height || 0;
  if (!mainWidth || !mainHeight || !prWidth || !prHeight) {
    throw new ComparisonError('Screenshot metadata has invalid dimensions.', 'corrupt');
  }
  if (mainWidth !== prWidth || mainHeight !== prHeight) {
    throw new ComparisonError(
      `Screenshot dimensions differ: base is ${mainWidth}x${mainHeight}, head is ${prWidth}x${prHeight}.`,
      'dimension-mismatch',
    );
  }

  const masks = Array.isArray(options.masks) ? options.masks : [];
  const maskedPixels = validateMasksForImage(masks, mainWidth, mainHeight);
  let maskedMainImage;
  let maskedPrImage;
  try {
    [maskedMainImage, maskedPrImage] = await Promise.all([
      applyMasks(mainImage, masks),
      applyMasks(prImage, masks),
    ]);
  } catch (error) {
    throw new ComparisonError(`Unable to apply scenario masks: ${error.message}`, 'analysis');
  }

  let looksSameResult;
  try {
    looksSameResult = await compareImages(maskedMainImage, maskedPrImage, {
      shouldCluster: true,
      clustersSize: options.looksSameClusterSize,
      tolerance: options.looksSameTolerance,
      // Capture hides carets in CSS, so ignoring caret-shaped regions here could mask real changes.
      ignoreCaret: false,
      ignoreAntialiasing: true,
      createDiffImage: true,
    });
  } catch (error) {
    throw new ComparisonError(`Image analysis failed: ${error.message}`, 'analysis');
  }

  const diffPercent = deriveDiffPercent(looksSameResult, mainWidth, mainHeight, maskedPixels);
  if (!Number.isFinite(diffPercent) || diffPercent < 0 || diffPercent > 100) {
    throw new ComparisonError('Image analysis did not return a valid diff percentage.', 'analysis');
  }

  return {
    comparablePixels: mainWidth * mainHeight - maskedPixels,
    diffPercent,
    maskedMainImage,
    maskedPixels,
    maskedPrImage,
    regions: extractDiffRegions(looksSameResult, mainWidth, mainHeight, diffPercent),
    width: mainWidth,
    height: mainHeight,
  };
}

function createDiffOverlay(diffAnalysis, renderWidth, renderHeight, totalHeight) {
  if (!diffAnalysis || !Array.isArray(diffAnalysis.regions) || diffAnalysis.regions.length === 0) {
    return null;
  }

  const scaleX = renderWidth / diffAnalysis.width;
  const scaleY = renderHeight / diffAnalysis.height;
  const rightOffset = renderWidth + DIVIDER_WIDTH;
  const boxes = diffAnalysis.regions.map((region) => {
    const baseX = region.x * scaleX;
    const baseY = LABEL_HEIGHT + region.y * scaleY;
    const baseWidth = region.width * scaleX;
    const baseHeight = region.height * scaleY;
    const x = Math.max(0, baseX - REGION_BOX_PADDING);
    const y = Math.max(LABEL_HEIGHT, baseY - REGION_BOX_PADDING);
    const width = Math.max(1, Math.min(renderWidth - x, baseWidth + REGION_BOX_PADDING * 2));
    const height = Math.max(1, Math.min(totalHeight - y, baseHeight + REGION_BOX_PADDING * 2));

    return `
      <rect x="${x}" y="${y}" width="${width}" height="${height}" rx="${DIFF_MARKER_RADIUS}" ry="${DIFF_MARKER_RADIUS}" fill="none" stroke="${DIFF_MARKER_COLOR}" stroke-width="${DIFF_MARKER_WIDTH}" />
      <rect x="${x + rightOffset}" y="${y}" width="${width}" height="${height}" rx="${DIFF_MARKER_RADIUS}" ry="${DIFF_MARKER_RADIUS}" fill="none" stroke="${DIFF_MARKER_COLOR}" stroke-width="${DIFF_MARKER_WIDTH}" />
    `;
  });

  return Buffer.from(`
    <svg width="${renderWidth * 2 + DIVIDER_WIDTH}" height="${totalHeight}">
      ${boxes.join('\n')}
    </svg>
  `);
}

async function generateComparison(
  pageName,
  mainPath,
  prPath,
  outputPath,
  mainLabel,
  prLabel,
  diffAnalysis,
  highlightDiff,
) {
  console.log(`Generating comparison for: ${pageName}`);
  const width = Math.max(1, Math.floor(diffAnalysis.width / 2));
  const height = Math.max(1, Math.floor(diffAnalysis.height / 2));
  const totalHeight = height + LABEL_HEIGHT;
  const [mainImage, prImage, divider] = await Promise.all([
    createLabeledImage(mainPath, mainLabel, width, height),
    createLabeledImage(prPath, prLabel, width, height),
    createDivider(totalHeight),
  ]);
  const composites = [
    { input: await mainImage.toBuffer(), top: 0, left: 0 },
    { input: await divider.toBuffer(), top: 0, left: width },
    { input: await prImage.toBuffer(), top: 0, left: width + DIVIDER_WIDTH },
  ];

  const diffOverlay = highlightDiff
    ? createDiffOverlay(diffAnalysis, width, height, totalHeight)
    : null;
  if (diffOverlay) {
    composites.push({ input: diffOverlay, top: 0, left: 0 });
  }

  const contents = await sharp({
    create: {
      width: width * 2 + DIVIDER_WIDTH,
      height: totalHeight,
      channels: 4,
      background: { r: 255, g: 255, b: 255, alpha: 1 },
    },
  })
    .composite(composites)
    .png()
    .toBuffer();
  fs.writeFileSync(outputPath, contents, { flag: 'wx' });

  console.log(`  ✓ Saved: ${path.basename(outputPath)}`);
  return outputPath;
}

function comparisonArtifactFilenames(filename) {
  if (
    typeof filename !== 'string' ||
    path.basename(filename) !== filename ||
    !filename.endsWith('.png')
  ) {
    throw new ComparisonError(`Invalid comparison filename ${filename}.`, 'manifest');
  }
  const stem = filename.slice(0, -4);
  return {
    base: `${stem}--base.png`,
    head: `${stem}--head.png`,
    overlay: `${stem}--overlay.png`,
    rawDiff: `${stem}--raw-diff.png`,
    // Keep the historical filename for compatibility with existing upgrade/report consumers.
    highlightedDiff: filename,
  };
}

function artifactAttestation(outputDir, filename) {
  const outputPath = path.join(outputDir, filename);
  const stat = fs.statSync(outputPath);
  return {
    filename,
    sha256: sha256File(outputPath),
    sizeBytes: stat.size,
  };
}

async function createOverlayImage(mainImage, prImage) {
  const translucentHead = await sharp(prImage).removeAlpha().ensureAlpha(0.5).png().toBuffer();
  return sharp(mainImage)
    .composite([{ input: translucentHead, blend: 'over' }])
    .png()
    .toBuffer();
}

async function createRawDiffImage(maskedMainImage, maskedPrImage) {
  return sharp(maskedMainImage)
    .composite([{ input: maskedPrImage, blend: 'difference' }])
    .png()
    .toBuffer();
}

async function generateArtifactSet({
  baseImage,
  diffAnalysis,
  filenames,
  hasVisualDiff,
  headImage,
  mainLabel,
  outputDir,
  page,
  prLabel,
}) {
  const outputPaths = Object.fromEntries(
    Object.entries(filenames).map(([name, filename]) => [name, path.join(outputDir, filename)]),
  );
  const [overlay, rawDiff] = await Promise.all([
    createOverlayImage(baseImage, headImage),
    createRawDiffImage(diffAnalysis.maskedMainImage, diffAnalysis.maskedPrImage),
  ]);
  fs.writeFileSync(outputPaths.base, baseImage, { flag: 'wx' });
  fs.writeFileSync(outputPaths.head, headImage, { flag: 'wx' });
  fs.writeFileSync(outputPaths.overlay, overlay, { flag: 'wx' });
  fs.writeFileSync(outputPaths.rawDiff, rawDiff, { flag: 'wx' });
  await generateComparison(
    page,
    baseImage,
    headImage,
    outputPaths.highlightedDiff,
    mainLabel,
    prLabel,
    diffAnalysis,
    hasVisualDiff,
  );
  return Object.fromEntries(
    Object.entries(filenames).map(([name, filename]) => [
      name,
      artifactAttestation(outputDir, filename),
    ]),
  );
}

function parseTimestamp(value, description) {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) {
    throw new ComparisonError(`${description} is missing or invalid.`, 'manifest');
  }
  return timestamp;
}

function sha256File(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function defaultCaptureValidity(status) {
  if (status === 'success') return 'valid';
  if (status === 'degraded') return 'selector_drift';
  if (status === 'skipped') return 'missing_fixture';
  return 'ui_rendering_failure';
}

function sanitizeCaptureDiagnosticText(value) {
  return String(value || '')
    .replace(/[\u0000-\u001f\u007f]+/g, ' ')
    .replace(/\bBearer\s+\S+/gi, 'Bearer <redacted>')
    .replace(/([?&][a-zA-Z0-9_.-]+)=([^&\s]+)/g, '$1=<redacted>')
    .replace(
      /\b(authorization|cookie|token|secret|password|api[-_]?key)\s*[:=]\s*\S+/gi,
      '$1=<redacted>',
    )
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 2000);
}

function normalizeCaptureDiagnostics(value, label) {
  if (value === undefined || value === null) return null;
  if (!isPlainObject(value)) {
    throw new ComparisonError(`${label} must be an object.`, 'manifest');
  }
  const normalizeRecords = (records, recordLabel) => {
    if (!Array.isArray(records) || records.length > 50) {
      throw new ComparisonError(
        `${recordLabel} must be an array of at most 50 records.`,
        'manifest',
      );
    }
    return records.map((record, index) => {
      if (!isPlainObject(record) || Object.keys(record).length > 12) {
        throw new ComparisonError(`${recordLabel}[${index}] must be a bounded object.`, 'manifest');
      }
      return Object.fromEntries(
        Object.entries(record).map(([key, entry]) => {
          if (typeof entry === 'string') return [key, sanitizeCaptureDiagnosticText(entry)];
          if (
            entry === null ||
            typeof entry === 'boolean' ||
            (Number.isSafeInteger(entry) && Math.abs(entry) <= Number.MAX_SAFE_INTEGER)
          ) {
            return [key, entry];
          }
          throw new ComparisonError(
            `${recordLabel}[${index}].${key} has an unsupported value.`,
            'manifest',
          );
        }),
      );
    });
  };
  if (!isPlainObject(value.dropped)) {
    throw new ComparisonError(`${label}.dropped must be an object.`, 'manifest');
  }
  const dropped = {};
  for (const name of ['consoleErrors', 'failedRequests']) {
    if (!Number.isSafeInteger(value.dropped[name]) || value.dropped[name] < 0) {
      throw new ComparisonError(
        `${label}.dropped.${name} must be a non-negative integer.`,
        'manifest',
      );
    }
    dropped[name] = value.dropped[name];
  }
  return {
    consoleErrors: normalizeRecords(value.consoleErrors, `${label}.consoleErrors`),
    failedRequests: normalizeRecords(value.failedRequests, `${label}.failedRequests`),
    dropped,
  };
}

function normalizeSemanticIdNormalizationAttestation(value, label) {
  if (!isPlainObject(value) || value.schemaVersion !== SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION) {
    throw new ComparisonError(
      `${label} must use semantic ID normalization schema ${SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION}.`,
      'manifest',
    );
  }
  if (value.complete !== true || !Array.isArray(value.scopes)) {
    throw new ComparisonError(`${label} is incomplete or has invalid scopes.`, 'manifest');
  }
  const allowedTopFields = new Set([
    'complete',
    'derivedColorScopes',
    'schemaVersion',
    'scopes',
    'totalReplacementCount',
  ]);
  if (Object.keys(value).some((field) => !allowedTopFields.has(field))) {
    throw new ComparisonError(`${label} has an unknown field.`, 'manifest');
  }
  if (!Array.isArray(value.derivedColorScopes)) {
    throw new ComparisonError(`${label} has invalid derivedColorScopes.`, 'manifest');
  }
  const derivedColorKeys = new Set();
  const derivedColorScopes = value.derivedColorScopes.map((scope, scopeIndex) => {
    if (!isPlainObject(scope)) {
      throw new ComparisonError(
        `${label} derived-color scope ${scopeIndex} is invalid.`,
        'manifest',
      );
    }
    const allowedFields = new Set([
      'companionCount',
      'containerSelector',
      'elementCount',
      'key',
      'labelItemSelector',
      'mappingStrategy',
      'maxElements',
      'mappings',
      'minElements',
      'selector',
      'semanticIds',
    ]);
    if (Object.keys(scope).some((field) => !allowedFields.has(field))) {
      throw new ComparisonError(
        `${label} derived-color scope ${scopeIndex} has an unknown field.`,
        'manifest',
      );
    }
    if (
      typeof scope.key !== 'string' ||
      !/^[a-z0-9][a-z0-9-]*$/.test(scope.key) ||
      derivedColorKeys.has(scope.key) ||
      typeof scope.selector !== 'string' ||
      !scope.selector ||
      scope.selector.length > 1024 ||
      typeof scope.containerSelector !== 'string' ||
      !scope.containerSelector ||
      scope.containerSelector.length > 1024 ||
      typeof scope.labelItemSelector !== 'string' ||
      !scope.labelItemSelector ||
      scope.labelItemSelector.length > 1024 ||
      !['color-backed-labels', 'ordered-label-cards'].includes(scope.mappingStrategy) ||
      !Number.isSafeInteger(scope.minElements) ||
      scope.minElements < 1 ||
      !(
        scope.maxElements === null ||
        (Number.isSafeInteger(scope.maxElements) && scope.maxElements >= scope.minElements)
      ) ||
      !Number.isSafeInteger(scope.elementCount) ||
      scope.elementCount < scope.minElements ||
      (scope.maxElements !== null && scope.elementCount > scope.maxElements) ||
      !Number.isSafeInteger(scope.companionCount) ||
      scope.companionCount < scope.elementCount ||
      !Array.isArray(scope.semanticIds) ||
      scope.semanticIds.length === 0 ||
      new Set(scope.semanticIds).size !== scope.semanticIds.length ||
      scope.semanticIds.some(
        (semanticId) =>
          typeof semanticId !== 'string' || !SEMANTIC_ID_PATH_PATTERN.test(semanticId),
      ) ||
      scope.semanticIds.length !== scope.elementCount ||
      !Array.isArray(scope.mappings) ||
      scope.mappings.length !== scope.elementCount
    ) {
      throw new ComparisonError(
        `${label} derived-color scope ${scopeIndex} is malformed.`,
        'manifest',
      );
    }
    const semanticIds = new Set();
    const mappings = scope.mappings.map((mapping, mappingIndex) => {
      if (
        !isPlainObject(mapping) ||
        Object.keys(mapping).some(
          (field) =>
            field !== 'semanticId' && field !== 'paletteColor' && field !== 'sourceColorSha256',
        ) ||
        typeof mapping.semanticId !== 'string' ||
        !SEMANTIC_ID_PATH_PATTERN.test(mapping.semanticId) ||
        !scope.semanticIds.includes(mapping.semanticId) ||
        semanticIds.has(mapping.semanticId) ||
        mapping.paletteColor !==
          SEMANTIC_COLOR_PALETTE[
            scope.semanticIds.indexOf(mapping.semanticId) % SEMANTIC_COLOR_PALETTE.length
          ] ||
        typeof mapping.sourceColorSha256 !== 'string' ||
        !/^[a-f0-9]{64}$/.test(mapping.sourceColorSha256)
      ) {
        throw new ComparisonError(
          `${label} derived-color scope ${scopeIndex} mapping ${mappingIndex} is invalid.`,
          'manifest',
        );
      }
      semanticIds.add(mapping.semanticId);
      return canonicalizeJson(mapping);
    });
    if (
      mappings.some(
        (mapping, index) =>
          index > 0 && mappings[index - 1].semanticId.localeCompare(mapping.semanticId) >= 0,
      )
    ) {
      throw new ComparisonError(
        `${label} derived-color scope ${scopeIndex} mappings are not ordered by semantic ID.`,
        'manifest',
      );
    }
    derivedColorKeys.add(scope.key);
    return canonicalizeJson({ ...scope, mappings });
  });
  const supportedKinds = new Set(SEMANTIC_ID_KINDS);
  const scopes = value.scopes.map((scope, scopeIndex) => {
    if (!isPlainObject(scope)) {
      throw new ComparisonError(`${label} scope ${scopeIndex} is invalid.`, 'manifest');
    }
    const allowedScopeFields = new Set([
      'entries',
      'kinds',
      'match',
      'maxReplacementsPerIdentifier',
      'maxReplacements',
      'minReplacementsPerIdentifier',
      'minReplacements',
      'replacementCount',
      'rootCount',
      'selector',
      'semanticIdPrefixes',
      'semanticIds',
    ]);
    if (Object.keys(scope).some((field) => !allowedScopeFields.has(field))) {
      throw new ComparisonError(`${label} scope ${scopeIndex} has an unknown field.`, 'manifest');
    }
    if (
      typeof scope.selector !== 'string' ||
      !scope.selector ||
      scope.selector.length > 1024 ||
      (scope.match !== 'exact' && scope.match !== 'substring') ||
      !Array.isArray(scope.entries)
    ) {
      throw new ComparisonError(`${label} scope ${scopeIndex} is malformed.`, 'manifest');
    }
    const hasKinds = Array.isArray(scope.kinds) && scope.kinds.length > 0;
    const hasSemanticIds = Array.isArray(scope.semanticIds) && scope.semanticIds.length > 0;
    const hasSemanticIdPrefixes =
      Array.isArray(scope.semanticIdPrefixes) && scope.semanticIdPrefixes.length > 0;
    if (Number(hasKinds) + Number(hasSemanticIds) + Number(hasSemanticIdPrefixes) !== 1) {
      throw new ComparisonError(
        `${label} scope ${scopeIndex} must select kinds, semanticIds, or semanticIdPrefixes.`,
        'manifest',
      );
    }
    if (
      hasKinds &&
      (new Set(scope.kinds).size !== scope.kinds.length ||
        scope.kinds.some((kind) => !supportedKinds.has(kind)))
    ) {
      throw new ComparisonError(`${label} scope ${scopeIndex} has invalid kinds.`, 'manifest');
    }
    if (
      hasSemanticIds &&
      (new Set(scope.semanticIds).size !== scope.semanticIds.length ||
        scope.semanticIds.some(
          (semanticId) =>
            typeof semanticId !== 'string' || !SEMANTIC_ID_PATH_PATTERN.test(semanticId),
        ))
    ) {
      throw new ComparisonError(
        `${label} scope ${scopeIndex} has invalid semanticIds.`,
        'manifest',
      );
    }
    if (
      hasSemanticIdPrefixes &&
      (new Set(scope.semanticIdPrefixes).size !== scope.semanticIdPrefixes.length ||
        scope.semanticIdPrefixes.some(
          (prefix) => typeof prefix !== 'string' || !SEMANTIC_ID_PATH_PATTERN.test(prefix),
        ))
    ) {
      throw new ComparisonError(
        `${label} scope ${scopeIndex} has invalid semanticIdPrefixes.`,
        'manifest',
      );
    }
    if (
      !Number.isSafeInteger(scope.minReplacements) ||
      scope.minReplacements < 0 ||
      !(
        scope.maxReplacements === null ||
        (Number.isSafeInteger(scope.maxReplacements) &&
          scope.maxReplacements >= scope.minReplacements)
      ) ||
      !Number.isSafeInteger(scope.replacementCount) ||
      scope.replacementCount < 0 ||
      !Number.isSafeInteger(scope.rootCount) ||
      scope.rootCount < 0 ||
      !Number.isSafeInteger(scope.minReplacementsPerIdentifier) ||
      scope.minReplacementsPerIdentifier < 0 ||
      !(
        scope.maxReplacementsPerIdentifier === null ||
        (Number.isSafeInteger(scope.maxReplacementsPerIdentifier) &&
          scope.maxReplacementsPerIdentifier >= scope.minReplacementsPerIdentifier)
      ) ||
      ((hasKinds || hasSemanticIdPrefixes) &&
        (scope.minReplacementsPerIdentifier > 0 || scope.maxReplacementsPerIdentifier !== null))
    ) {
      throw new ComparisonError(`${label} scope ${scopeIndex} has invalid counts.`, 'manifest');
    }

    const semanticIds = new Set();
    const entries = scope.entries.map((entry, entryIndex) => {
      const equivalenceClass = entry?.equivalenceClass;
      const hasEquivalenceClass = equivalenceClass !== undefined;
      if (
        !isPlainObject(entry) ||
        !supportedKinds.has(entry.kind) ||
        typeof entry.semanticId !== 'string' ||
        !SEMANTIC_ID_PATH_PATTERN.test(entry.semanticId) ||
        typeof entry.token !== 'string' ||
        !SEMANTIC_ID_TOKEN_PATTERN.test(entry.token) ||
        !SEMANTIC_ID_KINDS.includes(entry.tokenKind) ||
        typeof entry.tokenSemanticId !== 'string' ||
        !SEMANTIC_ID_PATH_PATTERN.test(entry.tokenSemanticId) ||
        entry.token !== semanticIdToken(entry.tokenKind, entry.tokenSemanticId) ||
        (hasEquivalenceClass &&
          (entry.kind !== 'task' ||
            typeof equivalenceClass !== 'string' ||
            !SEMANTIC_ID_PATH_PATTERN.test(equivalenceClass) ||
            entry.tokenSemanticId !== equivalenceClass)) ||
        typeof entry.sourceIdSha256 !== 'string' ||
        !/^[a-f0-9]{64}$/.test(entry.sourceIdSha256) ||
        !Number.isSafeInteger(entry.replacementCount) ||
        entry.replacementCount < 0
      ) {
        throw new ComparisonError(
          `${label} scope ${scopeIndex} entry ${entryIndex} is invalid.`,
          'manifest',
        );
      }
      const allowedEntryFields = new Set([
        'equivalenceClass',
        'kind',
        'replacementCount',
        'semanticId',
        'sourceIdSha256',
        'token',
        'tokenKind',
        'tokenSemanticId',
      ]);
      if (Object.keys(entry).some((field) => !allowedEntryFields.has(field))) {
        throw new ComparisonError(
          `${label} scope ${scopeIndex} entry ${entryIndex} has an unknown field.`,
          'manifest',
        );
      }
      if (semanticIds.has(entry.semanticId)) {
        throw new ComparisonError(
          `${label} scope ${scopeIndex} contains duplicate semantic ID ${entry.semanticId}.`,
          'manifest',
        );
      }
      semanticIds.add(entry.semanticId);
      if (
        (hasKinds && !scope.kinds.includes(entry.kind)) ||
        (hasSemanticIds && !scope.semanticIds.includes(entry.semanticId)) ||
        (hasSemanticIdPrefixes &&
          !scope.semanticIdPrefixes.some((prefix) => entry.semanticId.startsWith(prefix)))
      ) {
        throw new ComparisonError(
          `${label} scope ${scopeIndex} entry ${entryIndex} is outside its declared selection.`,
          'manifest',
        );
      }
      if (
        entry.replacementCount < scope.minReplacementsPerIdentifier ||
        (scope.maxReplacementsPerIdentifier !== null &&
          entry.replacementCount > scope.maxReplacementsPerIdentifier)
      ) {
        throw new ComparisonError(
          `${label} scope ${scopeIndex} entry ${entryIndex} violates its per-identifier replacement bounds.`,
          'manifest',
        );
      }
      return canonicalizeJson(entry);
    });
    if (hasSemanticIds && scope.semanticIds.some((semanticId) => !semanticIds.has(semanticId))) {
      throw new ComparisonError(
        `${label} scope ${scopeIndex} omits an explicitly selected semantic ID.`,
        'manifest',
      );
    }
    const replacementCount = entries.reduce((sum, entry) => sum + entry.replacementCount, 0);
    if (
      replacementCount !== scope.replacementCount ||
      (replacementCount > 0 && scope.rootCount === 0) ||
      replacementCount < scope.minReplacements ||
      (scope.maxReplacements !== null && replacementCount > scope.maxReplacements)
    ) {
      throw new ComparisonError(
        `${label} scope ${scopeIndex} has inconsistent replacement counts.`,
        'manifest',
      );
    }
    return canonicalizeJson({ ...scope, entries });
  });
  const totalReplacementCount = scopes.reduce((sum, scope) => sum + scope.replacementCount, 0);
  if (
    !Number.isSafeInteger(value.totalReplacementCount) ||
    value.totalReplacementCount < 0 ||
    value.totalReplacementCount !== totalReplacementCount
  ) {
    throw new ComparisonError(`${label} has an inconsistent total replacement count.`, 'manifest');
  }
  return canonicalizeJson({ ...value, derivedColorScopes, scopes });
}

function semanticIdNormalizationScopeContract(scope) {
  return canonicalizeJson({
    match: scope.match,
    maxReplacements: scope.maxReplacements ?? null,
    maxReplacementsPerIdentifier: scope.maxReplacementsPerIdentifier ?? null,
    minReplacements: scope.minReplacements ?? 0,
    minReplacementsPerIdentifier: scope.minReplacementsPerIdentifier ?? 0,
    selector: scope.selector,
    ...(Array.isArray(scope.kinds) ? { kinds: [...scope.kinds].sort() } : {}),
    ...(Array.isArray(scope.semanticIds) ? { semanticIds: [...scope.semanticIds].sort() } : {}),
    ...(Array.isArray(scope.semanticIdPrefixes)
      ? { semanticIdPrefixes: [...scope.semanticIdPrefixes].sort() }
      : {}),
  });
}

function semanticDerivedColorScopeContract(scope) {
  return canonicalizeJson({
    containerSelector: scope.containerSelector,
    key: scope.key,
    labelItemSelector: scope.labelItemSelector,
    mappingStrategy: scope.mappingStrategy,
    maxElements: scope.maxElements ?? null,
    minElements: scope.minElements ?? 1,
    selector: scope.selector,
    semanticIds: [...scope.semanticIds].sort(),
  });
}

function semanticIdNormalizationContractProjection(value) {
  return canonicalizeJson({
    derivedColorScopes: (value.derivedColorScopes || []).map(semanticDerivedColorScopeContract),
    scopes: (value.scopes || []).map(semanticIdNormalizationScopeContract),
  });
}

function validateSemanticIdNormalizationScenarioContracts(manifest, role) {
  if (!manifest.inputs?.semanticManifest) return;
  for (const result of manifest.results) {
    if (result.status !== 'success' && result.status !== 'degraded') continue;
    const pageContract = getSemanticIdNormalizationContract(role, result.page);
    const semanticScenario = result.semanticScenario || result.page;
    const semanticContract = getSemanticIdNormalizationContract(role, semanticScenario);
    if (!pageContract || !semanticContract || semanticScenario !== result.page) {
      throw new ComparisonError(
        `Capture result ${result.filename} does not bind canonical semantic scenario ${result.page}.`,
        'manifest',
      );
    }
    const actualContract = semanticIdNormalizationContractProjection(
      result.semanticIdNormalization,
    );
    const expectedContract = semanticIdNormalizationContractProjection(pageContract);
    if (JSON.stringify(actualContract) !== JSON.stringify(expectedContract)) {
      throw new ComparisonError(
        `Capture result ${result.filename} does not attest the ${role} semantic ID normalization contract for ${result.page}.`,
        'manifest',
      );
    }
  }
}

function validateGlobalVisualNormalizationEvidence(manifest, role, required) {
  const capturedResults = manifest.results.filter(
    (result) => result.status === 'success' || result.status === 'degraded',
  );
  if (!required) {
    for (const result of capturedResults) {
      if (
        result.globalVisualNormalization !== undefined &&
        result.globalVisualNormalization !== null
      ) {
        throw new ComparisonError(
          `Browser-compatibility capture result ${result.filename} cannot contain global visual normalization evidence.`,
          'manifest',
        );
      }
    }
    return;
  }
  if (role !== 'base' && role !== 'head') {
    throw new ComparisonError(
      'Semantic capture global visual normalization evidence requires revisionRole base or head.',
      'manifest',
    );
  }
  const contract = getGlobalVisualNormalizationContract(role);
  for (const result of capturedResults) {
    const label = `Capture result ${result.filename} globalVisualNormalization`;
    const evidence = result.globalVisualNormalization;
    if (
      !isPlainObject(evidence) ||
      evidence.schemaVersion !== contract.schemaVersion ||
      evidence.complete !== true ||
      !Array.isArray(evidence.rules) ||
      evidence.rules.length !== contract.rules.length ||
      Object.keys(evidence).some((field) => !['complete', 'rules', 'schemaVersion'].includes(field))
    ) {
      throw new ComparisonError(`${label} is missing or invalid.`, 'manifest');
    }
    for (const [index, expected] of contract.rules.entries()) {
      const actual = evidence.rules[index];
      const allowedFields = new Set([
        'actualMatches',
        'applied',
        'expectedChange',
        'expectedMatches',
        'hiddenMatches',
        'key',
        'operation',
        'selector',
      ]);
      if (
        !isPlainObject(actual) ||
        Object.keys(actual).some((field) => !allowedFields.has(field)) ||
        actual.actualMatches !== expected.expectedMatches ||
        actual.applied !== (expected.operation === 'hide') ||
        actual.expectedChange !== expected.expectedChange ||
        actual.expectedMatches !== expected.expectedMatches ||
        actual.hiddenMatches !== (expected.operation === 'hide' ? expected.expectedMatches : 0) ||
        actual.key !== expected.key ||
        actual.operation !== expected.operation ||
        actual.selector !== expected.selector
      ) {
        throw new ComparisonError(
          `${label}.rules[${index}] does not match its contract.`,
          'manifest',
        );
      }
    }
  }
}

function captureRecordEvidence(record, label) {
  if (!record) return null;
  return {
    status: record.status,
    error:
      record.error === undefined || record.error === null
        ? null
        : sanitizeCaptureDiagnosticText(record.error),
    reason:
      record.reason === undefined || record.reason === null
        ? null
        : sanitizeCaptureDiagnosticText(record.reason),
    diagnostics: normalizeCaptureDiagnostics(record.diagnostics, `${label} diagnostics`),
    globalVisualNormalization: record.globalVisualNormalization || null,
    semanticIdNormalization: record.semanticIdNormalization || null,
  };
}

function validateCaptureManifest(manifest, manifestPath) {
  if (!manifest || typeof manifest !== 'object') {
    throw new ComparisonError(`Capture manifest ${manifestPath} is not an object.`, 'manifest');
  }
  if (manifest.schemaVersion !== CAPTURE_MANIFEST_SCHEMA_VERSION) {
    throw new ComparisonError(
      `Capture manifest ${manifestPath} uses unsupported schema version ${manifest.schemaVersion}.`,
      'manifest',
    );
  }
  if (typeof manifest.captureId !== 'string' || manifest.captureId.length === 0) {
    throw new ComparisonError(`Capture manifest ${manifestPath} has no captureId.`, 'manifest');
  }
  if (!Array.isArray(manifest.results) || manifest.results.length === 0) {
    throw new ComparisonError(`Capture manifest ${manifestPath} has no results.`, 'manifest');
  }
  if (!Array.isArray(manifest.fatalErrors)) {
    throw new ComparisonError(
      `Capture manifest ${manifestPath} has an invalid fatalErrors field.`,
      'manifest',
    );
  }
  if (manifest.fatalErrors.length > 0) {
    throw new ComparisonError(
      `Capture manifest ${manifestPath} reports fatal errors: ${manifest.fatalErrors.join('; ')}`,
      'manifest',
    );
  }

  const startedAtMs = parseTimestamp(manifest.startedAt, `${manifestPath} startedAt`);
  const completedAtMs = parseTimestamp(manifest.completedAt, `${manifestPath} completedAt`);
  if (completedAtMs < startedAtMs) {
    throw new ComparisonError(
      `Capture manifest ${manifestPath} completed before it started.`,
      'manifest',
    );
  }

  const records = new Map();
  for (const result of manifest.results) {
    if (!result || typeof result !== 'object' || typeof result.page !== 'string') {
      throw new ComparisonError(
        `Capture manifest ${manifestPath} has an invalid result.`,
        'manifest',
      );
    }
    if (typeof result.required !== 'boolean') {
      throw new ComparisonError(
        `Capture result ${result.page} in ${manifestPath} does not declare required/optional.`,
        'manifest',
      );
    }
    if (!CAPTURE_STATUSES.has(result.status)) {
      throw new ComparisonError(
        `Capture result ${result.page} in ${manifestPath} has invalid status ${result.status}.`,
        'manifest',
      );
    }
    if (result.captureValidity !== undefined && !CAPTURE_VALIDITIES.has(result.captureValidity)) {
      throw new ComparisonError(
        `Capture result ${result.page} in ${manifestPath} has invalid captureValidity ${result.captureValidity}.`,
        'manifest',
      );
    }
    const captureValidity = result.captureValidity || defaultCaptureValidity(result.status);
    if (
      result.status === 'success' &&
      captureValidity !== 'valid' &&
      captureValidity !== 'expected_product_removal'
    ) {
      throw new ComparisonError(
        `Successful capture result ${result.page} in ${manifestPath} must be valid or an expected product removal.`,
        'manifest',
      );
    }
    if (result.status !== 'success' && captureValidity === 'valid') {
      throw new ComparisonError(
        `Incomplete capture result ${result.page} in ${manifestPath} cannot declare captureValidity valid.`,
        'manifest',
      );
    }
    if (
      result.semanticScenario !== undefined &&
      (typeof result.semanticScenario !== 'string' ||
        !/^[a-z0-9][a-z0-9-]*$/.test(result.semanticScenario))
    ) {
      throw new ComparisonError(
        `Capture result ${result.page} in ${manifestPath} has invalid semanticScenario.`,
        'manifest',
      );
    }
    for (const routeField of ['requestedRoute', 'resolvedRoute']) {
      if (
        result[routeField] !== undefined &&
        (typeof result[routeField] !== 'string' || result[routeField].length > 4096)
      ) {
        throw new ComparisonError(
          `Capture result ${result.page} in ${manifestPath} has invalid ${routeField}.`,
          'manifest',
        );
      }
    }
    if (
      !result.viewport ||
      !Number.isSafeInteger(result.viewport.width) ||
      result.viewport.width <= 0 ||
      !Number.isSafeInteger(result.viewport.height) ||
      result.viewport.height <= 0
    ) {
      throw new ComparisonError(
        `Capture result ${result.page} in ${manifestPath} has an invalid viewport.`,
        'manifest',
      );
    }

    const expectedFilename = `${result.page}-${result.viewport.width}x${result.viewport.height}.png`;
    if (
      result.filename !== expectedFilename ||
      path.basename(result.filename) !== result.filename
    ) {
      throw new ComparisonError(
        `Capture result ${result.page} in ${manifestPath} has invalid filename ${result.filename}.`,
        'manifest',
      );
    }
    if (records.has(result.filename)) {
      throw new ComparisonError(
        `Capture manifest ${manifestPath} contains duplicate result ${result.filename}.`,
        'manifest',
      );
    }

    let capturedAtMs = null;
    let semanticIdNormalization = null;
    if (result.status === 'success' || result.status === 'degraded') {
      capturedAtMs = parseTimestamp(
        result.capturedAt,
        `${manifestPath} result ${result.filename} capturedAt`,
      );
      if (
        capturedAtMs < startedAtMs - FRESHNESS_TOLERANCE_MS ||
        capturedAtMs > completedAtMs + FRESHNESS_TOLERANCE_MS
      ) {
        throw new ComparisonError(
          `Capture result ${result.filename} in ${manifestPath} is outside its capture window.`,
          'stale',
        );
      }
      if (typeof result.sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(result.sha256)) {
        throw new ComparisonError(
          `Capture result ${result.filename} in ${manifestPath} has an invalid sha256.`,
          'manifest',
        );
      }
      semanticIdNormalization = normalizeSemanticIdNormalizationAttestation(
        result.semanticIdNormalization,
        `Capture result ${result.filename} in ${manifestPath} semanticIdNormalization`,
      );
    } else if (result.semanticIdNormalization !== undefined) {
      semanticIdNormalization = normalizeSemanticIdNormalizationAttestation(
        result.semanticIdNormalization,
        `Capture result ${result.filename} in ${manifestPath} semanticIdNormalization`,
      );
    }

    const evidence = captureRecordEvidence(
      { ...result, semanticIdNormalization },
      `Capture result ${result.page} in ${manifestPath}`,
    );
    records.set(result.filename, {
      ...result,
      captureValidity,
      capturedAtMs,
      completedAtMs,
      diagnostics: evidence.diagnostics,
      error: evidence.error,
      reason: evidence.reason,
      semanticIdNormalization,
      startedAtMs,
    });
  }

  const requiredIncomplete = [...records.values()].some(
    (result) => result.required && result.status !== 'success',
  );
  const hasCapturedScreenshot = [...records.values()].some(
    (result) => result.status === 'success' || result.status === 'degraded',
  );
  const derivedComplete = !requiredIncomplete && hasCapturedScreenshot;
  if (
    typeof manifest.complete !== 'boolean' ||
    !manifest.summary ||
    typeof manifest.summary.complete !== 'boolean' ||
    manifest.complete !== derivedComplete ||
    manifest.summary.complete !== derivedComplete
  ) {
    throw new ComparisonError(
      `Capture manifest ${manifestPath} has inconsistent completion metadata.`,
      'manifest',
    );
  }
  return {
    label: typeof manifest.label === 'string' && manifest.label ? manifest.label : null,
    manifest,
    records,
  };
}

function normalizeCaptureInputAttestation(value, label, required = false) {
  if (value === null || value === undefined) {
    if (required) {
      throw new ComparisonError(`${label} is required.`, 'manifest');
    }
    return null;
  }
  if (
    !isPlainObject(value) ||
    typeof value.path !== 'string' ||
    value.path === '' ||
    typeof value.sha256 !== 'string' ||
    !/^[a-f0-9]{64}$/.test(value.sha256) ||
    !Number.isSafeInteger(value.sizeBytes) ||
    value.sizeBytes <= 0 ||
    !(
      value.schemaVersion === null ||
      typeof value.schemaVersion === 'string' ||
      typeof value.schemaVersion === 'number'
    )
  ) {
    throw new ComparisonError(`${label} is not a valid input attestation.`, 'manifest');
  }
  return {
    schemaVersion: value.schemaVersion,
    sha256: value.sha256,
    sizeBytes: value.sizeBytes,
  };
}

function captureProvenance(manifest, role) {
  if (!isPlainObject(manifest.inputs) || manifest.inputs.revisionRole !== role) {
    throw new ComparisonError(
      `${role === 'base' ? 'Base' : 'Head'} capture manifest inputs.revisionRole must be ${role}.`,
      'manifest',
    );
  }
  return {
    revisionRole: role,
    seedManifest: normalizeCaptureInputAttestation(
      manifest.inputs.seedManifest,
      `${role} seedManifest`,
      true,
    ),
    semanticManifest: normalizeCaptureInputAttestation(
      manifest.inputs.semanticManifest,
      `${role} semanticManifest`,
    ),
    sourceProvenance: normalizeCaptureInputAttestation(
      manifest.inputs.sourceProvenance,
      `${role} sourceProvenance`,
    ),
  };
}

function assertMatchingPairInput(base, head, name) {
  if (Boolean(base) !== Boolean(head)) {
    throw new ComparisonError(
      `Base and head capture manifests must either both bind ${name} or both omit it.`,
      'manifest',
    );
  }
  if (
    base &&
    (base.sha256 !== head.sha256 ||
      base.sizeBytes !== head.sizeBytes ||
      base.schemaVersion !== head.schemaVersion)
  ) {
    throw new ComparisonError(
      `Base and head capture manifests bind different ${name} inputs.`,
      'manifest',
    );
  }
}

function loadAttestedJsonArtifact(attestation, description) {
  const resolvedPath = path.resolve(attestation.path);
  let stat;
  let contents;
  try {
    stat = fs.lstatSync(resolvedPath);
    if (!stat.isFile() || stat.isSymbolicLink()) {
      throw new Error('path is not a non-symlink regular file');
    }
    contents = fs.readFileSync(resolvedPath);
  } catch (error) {
    throw new ComparisonError(`${description} cannot be read: ${error.message}.`, 'manifest');
  }
  let value;
  try {
    value = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new ComparisonError(`${description} is not valid JSON: ${error.message}.`, 'manifest');
  }
  const schemaVersion =
    typeof value?.schemaVersion === 'string' || typeof value?.schemaVersion === 'number'
      ? value.schemaVersion
      : null;
  if (
    contents.length !== attestation.sizeBytes ||
    crypto.createHash('sha256').update(contents).digest('hex') !== attestation.sha256 ||
    schemaVersion !== attestation.schemaVersion
  ) {
    throw new ComparisonError(`${description} does not match its capture attestation.`, 'manifest');
  }
  return value;
}

function validateSemanticNormalizationAgainstCatalog(manifest, role, semanticManifest) {
  let catalog;
  try {
    validateCombinedSemanticManifest(semanticManifest);
    catalog = require('./capture-screenshots').buildSemanticIdentifierCatalog(
      semanticManifest,
      role,
    );
  } catch (error) {
    throw new ComparisonError(
      `Attested semantic fixture manifest is invalid: ${error.message}`,
      'manifest',
    );
  }
  const expectedByIdentity = new Map(
    catalog.map((entry) => [`${entry.kind}\u0000${entry.semanticId}`, entry]),
  );
  const observedHashes = new Map();
  for (const result of manifest.results) {
    if (result.status !== 'success' && result.status !== 'degraded') continue;
    for (const scope of result.semanticIdNormalization.scopes) {
      for (const entry of scope.entries) {
        const identityKey = `${entry.kind}\u0000${entry.semanticId}`;
        const expected = expectedByIdentity.get(identityKey);
        const expectedHash = expected
          ? crypto.createHash('sha256').update(expected.value).digest('hex')
          : null;
        if (
          !expected ||
          expected.token !== entry.token ||
          expected.tokenKind !== entry.tokenKind ||
          expected.tokenSemanticId !== entry.tokenSemanticId ||
          (expected.equivalenceClass || undefined) !== entry.equivalenceClass ||
          expectedHash !== entry.sourceIdSha256
        ) {
          throw new ComparisonError(
            `${role} capture result ${result.filename} normalization entry ${entry.semanticId} does not match the attested semantic fixture manifest.`,
            'manifest',
          );
        }
        const observed = observedHashes.get(identityKey);
        if (observed && observed !== entry.sourceIdSha256) {
          throw new ComparisonError(
            `${role} capture binds inconsistent source hashes for ${entry.semanticId}.`,
            'manifest',
          );
        }
        observedHashes.set(identityKey, entry.sourceIdSha256);
      }
    }
    for (const scope of result.semanticIdNormalization.derivedColorScopes) {
      for (const semanticId of scope.semanticIds) {
        if (!expectedByIdentity.has(`run\u0000${semanticId}`)) {
          throw new ComparisonError(
            `${role} capture result ${result.filename} derived-color identity ${semanticId} is absent from the attested semantic fixture manifest.`,
            'manifest',
          );
        }
      }
    }
  }
}

function matchingManifestContract(baseManifest, headManifest, field) {
  const base = baseManifest[field];
  const head = headManifest[field];
  if (
    !isPlainObject(base) ||
    !isPlainObject(head) ||
    JSON.stringify(canonicalizeJson(base)) !== JSON.stringify(canonicalizeJson(head))
  ) {
    throw new ComparisonError(
      `Base and head capture manifests must use the same ${field} contract.`,
      'manifest',
    );
  }
  return canonicalizeJson(base);
}

function validateCapturePairProvenance(baseManifest, headManifest) {
  const base = captureProvenance(baseManifest, 'base');
  const head = captureProvenance(headManifest, 'head');
  if (base.seedManifest.schemaVersion !== head.seedManifest.schemaVersion) {
    throw new ComparisonError(
      'Base and head seed manifests use different schema versions.',
      'manifest',
    );
  }
  assertMatchingPairInput(base.semanticManifest, head.semanticManifest, 'semanticManifest');
  assertMatchingPairInput(base.sourceProvenance, head.sourceProvenance, 'sourceProvenance');
  if (Boolean(base.semanticManifest) !== Boolean(base.sourceProvenance)) {
    throw new ComparisonError(
      'Semantic full-stack captures require both semanticManifest and sourceProvenance; browser compatibility captures require neither.',
      'manifest',
    );
  }
  validateGlobalVisualNormalizationEvidence(baseManifest, 'base', Boolean(base.semanticManifest));
  validateGlobalVisualNormalizationEvidence(headManifest, 'head', Boolean(head.semanticManifest));
  if (base.semanticManifest) {
    const baseSemanticManifest = loadAttestedJsonArtifact(
      baseManifest.inputs.semanticManifest,
      'Base semanticManifest',
    );
    const headSemanticManifest = loadAttestedJsonArtifact(
      headManifest.inputs.semanticManifest,
      'Head semanticManifest',
    );
    if (
      JSON.stringify(canonicalizeJson(baseSemanticManifest)) !==
      JSON.stringify(canonicalizeJson(headSemanticManifest))
    ) {
      throw new ComparisonError(
        'Base and head semanticManifest artifacts do not contain the same fixture mapping.',
        'manifest',
      );
    }
    validateSemanticNormalizationAgainstCatalog(baseManifest, 'base', baseSemanticManifest);
    validateSemanticNormalizationAgainstCatalog(headManifest, 'head', headSemanticManifest);
  }
  if (baseManifest.scenarioContractSchemaVersion !== headManifest.scenarioContractSchemaVersion) {
    throw new ComparisonError(
      'Base and head capture manifests use different semantic scenario contracts.',
      'manifest',
    );
  }
  if (
    base.semanticManifest &&
    baseManifest.scenarioContractSchemaVersion !== SCENARIO_CONTRACT_SCHEMA_VERSION
  ) {
    throw new ComparisonError(
      `Semantic captures must use scenario contract ${SCENARIO_CONTRACT_SCHEMA_VERSION}.`,
      'manifest',
    );
  }
  validateSemanticIdNormalizationScenarioContracts(baseManifest, 'base');
  validateSemanticIdNormalizationScenarioContracts(headManifest, 'head');
  const browser = matchingManifestContract(baseManifest, headManifest, 'browser');
  const deterministicRendering = matchingManifestContract(
    baseManifest,
    headManifest,
    'deterministicRendering',
  );
  const expectedNormalizationMode = base.semanticManifest
    ? SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK
    : SEMANTIC_ID_NORMALIZATION_MODES.BROWSER_COMPATIBILITY;
  const expectedNormalizationContract =
    semanticIdNormalizationRenderingContract(expectedNormalizationMode);
  if (
    !isPlainObject(deterministicRendering.semanticIdNormalization) ||
    JSON.stringify(canonicalizeJson(deterministicRendering.semanticIdNormalization)) !==
      JSON.stringify(canonicalizeJson(expectedNormalizationContract))
  ) {
    throw new ComparisonError(
      `Capture manifests must use the exact ${expectedNormalizationMode} semantic ID normalization contract ${SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION}.`,
      'manifest',
    );
  }
  if (!base.semanticManifest) {
    for (const [role, manifest] of [
      ['base', baseManifest],
      ['head', headManifest],
    ]) {
      for (const result of manifest.results) {
        if (result.status !== 'success' && result.status !== 'degraded') continue;
        if (
          result.semanticIdNormalization.totalReplacementCount !== 0 ||
          result.semanticIdNormalization.scopes.length !== 0 ||
          result.semanticIdNormalization.derivedColorScopes.length !== 0
        ) {
          throw new ComparisonError(
            `${role} browser-compatibility capture ${result.filename} cannot contain semantic normalization evidence.`,
            'manifest',
          );
        }
      }
    }
  }
  return {
    base,
    browser,
    deterministicRendering,
    head,
    scenarioContractSchemaVersion: baseManifest.scenarioContractSchemaVersion ?? null,
  };
}

function loadCaptureManifest(directory) {
  const manifestPath = path.join(directory, CAPTURE_MANIFEST_FILENAME);
  if (!fs.existsSync(manifestPath)) {
    return null;
  }

  let manifestContents;
  let manifest;
  try {
    manifestContents = fs.readFileSync(manifestPath);
    manifest = JSON.parse(manifestContents.toString('utf8'));
  } catch (error) {
    throw new ComparisonError(
      `Unable to parse capture manifest ${manifestPath}: ${error.message}`,
      'manifest',
    );
  }
  const validated = validateCaptureManifest(manifest, manifestPath);
  return {
    ...validated,
    attestation: {
      captureId: manifest.captureId,
      manifestSha256: crypto.createHash('sha256').update(manifestContents).digest('hex'),
      manifestSizeBytes: manifestContents.length,
      requiredFilenames: [...validated.records.values()]
        .filter((result) => result.required && result.status === 'success')
        .map((result) => result.filename)
        .sort(),
    },
  };
}

function captureAnnotation(value, label) {
  try {
    return normalizeExpectedChange(value, label);
  } catch (error) {
    throw new ComparisonError(error.message, 'manifest');
  }
}

function scenarioMetadata(mainRecord, prRecord, filename) {
  const fallbackPage = (mainRecord || prRecord)?.page || filename.replace(/\.png$/, '');
  const baseScenario = mainRecord?.semanticScenario || mainRecord?.page || null;
  const headScenario = prRecord?.semanticScenario || prRecord?.page || null;
  const semanticScenario = baseScenario || headScenario || fallbackPage;
  const semanticMismatch = Boolean(baseScenario && headScenario && baseScenario !== headScenario);
  const baseTitle = mainRecord?.scenarioTitle || null;
  const headTitle = prRecord?.scenarioTitle || null;
  const scenarioTitle =
    baseTitle && headTitle && baseTitle !== headTitle
      ? `${baseTitle} → ${headTitle}`
      : baseTitle || headTitle || semanticScenario;
  const expectedChanges = {
    base: captureAnnotation(mainRecord?.expectedChange, `Base ${semanticScenario} expectedChange`),
    head: captureAnnotation(prRecord?.expectedChange, `Head ${semanticScenario} expectedChange`),
  };
  const expectedChange =
    JSON.stringify(expectedChanges.base) === JSON.stringify(expectedChanges.head)
      ? expectedChanges.base
      : expectedChanges;
  return {
    semanticMismatch,
    semanticScenario,
    scenarioTitle,
    expectedChange,
    routes: {
      base: {
        requested: mainRecord?.requestedRoute || null,
        resolved: mainRecord?.resolvedRoute || null,
        expectation: captureAnnotation(
          mainRecord?.routeExpectation,
          `Base ${semanticScenario} routeExpectation`,
        ),
      },
      head: {
        requested: prRecord?.requestedRoute || null,
        resolved: prRecord?.resolvedRoute || null,
        expectation: captureAnnotation(
          prRecord?.routeExpectation,
          `Head ${semanticScenario} routeExpectation`,
        ),
      },
    },
    captureValidityByRevision: {
      base: mainRecord?.captureValidity || 'missing_fixture',
      head: prRecord?.captureValidity || 'missing_fixture',
    },
    captureEvidenceByRevision: {
      base: captureRecordEvidence(mainRecord, `Base ${semanticScenario}`),
      head: captureRecordEvidence(prRecord, `Head ${semanticScenario}`),
    },
    viewport: mainRecord?.viewport || prRecord?.viewport || null,
  };
}

function aggregateCaptureValidity(captureValidityByRevision) {
  const values = Object.values(captureValidityByRevision);
  return (
    values.find((value) => value !== 'valid' && value !== 'expected_product_removal') ||
    (values.includes('expected_product_removal') ? 'expected_product_removal' : 'valid')
  );
}

function routeMatchesExpectation(actualRoute, expectedRoute) {
  if (typeof actualRoute !== 'string' || typeof expectedRoute !== 'string') return false;
  try {
    const parseRoute = (value) => {
      const parsed = new URL(value, 'http://ui-smoke.invalid');
      const route = parsed.hash.startsWith('#/')
        ? parsed.hash.slice(1)
        : `${parsed.pathname}${parsed.search}`;
      return new URL(route, 'http://ui-smoke.invalid');
    };
    const actual = parseRoute(actualRoute);
    const expected = parseRoute(expectedRoute);
    if (actual.pathname !== expected.pathname) return false;
    for (const [key, value] of expected.searchParams) {
      if (actual.searchParams.get(key) !== value) return false;
    }
    return true;
  } catch (_error) {
    return false;
  }
}

function expectedRemovalClaimError(record, role) {
  if (record?.captureValidity !== 'expected_product_removal') return null;
  if (role !== 'head') {
    return 'expected_product_removal may only be asserted by the head capture';
  }
  if (record.status !== 'success') {
    return 'head expected_product_removal must have a successful screenshot';
  }
  if (
    !isPlainObject(record.routeExpectation) ||
    record.routeExpectation.kind !== 'expected-removal' ||
    typeof record.routeExpectation.path !== 'string' ||
    record.routeExpectation.path === ''
  ) {
    return 'head expected_product_removal must declare routeExpectation.kind expected-removal and a path';
  }
  if (!routeMatchesExpectation(record.resolvedRoute, record.routeExpectation.path)) {
    return `head expected_product_removal resolved route ${record.resolvedRoute || '(missing)'} does not match ${record.routeExpectation.path}`;
  }
  return null;
}

function failedPlanResult(filename, required, message, failureType = 'capture', metadata = {}) {
  return {
    filename,
    page: filename.replace(/\.png$/, ''),
    required,
    status: 'failed',
    error: message,
    failureType,
    captureValidity: metadata.captureValidity || 'missing_fixture',
    comparisonValidity: 'not-compared',
    thresholdsEvaluated: false,
    ...metadata,
  };
}

function buildManifestComparisonPlan(mainManifest, prManifest, mainDir, prDir) {
  const filenames = [
    ...new Set([...mainManifest.records.keys(), ...prManifest.records.keys()]),
  ].sort();
  const pairs = [];
  const results = [];

  for (const filename of filenames) {
    const mainRecord = mainManifest.records.get(filename);
    const prRecord = prManifest.records.get(filename);
    const required = Boolean(mainRecord?.required || prRecord?.required);
    const metadata = scenarioMetadata(mainRecord, prRecord, filename);
    const problems = [];

    if (metadata.semanticMismatch) {
      problems.push('semantic scenario differs between captures');
    }

    if (
      !required &&
      (!mainRecord || !prRecord || mainRecord.status !== 'success' || prRecord.status !== 'success')
    ) {
      results.push({
        filename,
        page: filename.replace(/\.png$/, ''),
        required: false,
        status: 'skipped',
        reason: 'Optional capture was not successful in both revisions.',
        captureValidity: aggregateCaptureValidity(metadata.captureValidityByRevision),
        comparisonValidity: 'not-compared',
        thresholdsEvaluated: false,
        ...metadata,
      });
      continue;
    }

    const expectedRemovalProblems = [
      expectedRemovalClaimError(mainRecord, 'base'),
      expectedRemovalClaimError(prRecord, 'head'),
    ].filter(Boolean);
    if (expectedRemovalProblems.length > 0) {
      results.push(
        failedPlanResult(filename, required, expectedRemovalProblems.join('; '), 'capture', {
          ...metadata,
          captureValidity: 'ui_rendering_failure',
        }),
      );
      continue;
    }
    const expectedRemoval = prRecord?.captureValidity === 'expected_product_removal';

    if (required) {
      if (!mainRecord) problems.push('missing from base capture manifest');
      if (!prRecord) problems.push('missing from head capture manifest');
      if (mainRecord && prRecord && mainRecord.required !== prRecord.required) {
        problems.push('required/optional classification differs between captures');
      }
      if (mainRecord && mainRecord.status !== 'success') {
        problems.push(`base capture status is ${mainRecord.status}`);
      }
      if (prRecord && prRecord.status !== 'success') {
        problems.push(`head capture status is ${prRecord.status}`);
      }
    }

    if (problems.length > 0) {
      results.push(
        failedPlanResult(filename, required, problems.join('; '), 'capture', {
          ...metadata,
          captureValidity: aggregateCaptureValidity(metadata.captureValidityByRevision),
        }),
      );
      continue;
    }

    pairs.push({
      filename,
      mainPath: path.join(mainDir, filename),
      mainRecord,
      page: (mainRecord || prRecord).page,
      prPath: path.join(prDir, filename),
      prRecord,
      required,
      expectedRemoval,
      ...metadata,
    });
  }

  return { filenames, pairs, results };
}

function buildComparisonPlan(mainDir, prDir, scenarioConfigPath = null) {
  const mainManifest = loadCaptureManifest(mainDir);
  const prManifest = loadCaptureManifest(prDir);
  if (!mainManifest && !prManifest) {
    throw new ComparisonError(
      'Both capture manifests are required; refusing to compare untracked PNG directories.',
      'manifest',
    );
  }
  if (Boolean(mainManifest) !== Boolean(prManifest)) {
    throw new ComparisonError(
      'Capture manifest is present for only one revision; refusing to mix manifest and directory discovery.',
      'manifest',
    );
  }
  if (mainManifest.manifest.captureId === prManifest.manifest.captureId) {
    throw new ComparisonError(
      'Base and head manifests have the same captureId; refusing to compare a capture with itself.',
      'manifest',
    );
  }

  const manifestPlan = buildManifestComparisonPlan(mainManifest, prManifest, mainDir, prDir);
  const provenance = validateCapturePairProvenance(mainManifest.manifest, prManifest.manifest);
  mainManifest.attestation.inputs = provenance.base;
  prManifest.attestation.inputs = provenance.head;
  mainManifest.attestation.browser = provenance.browser;
  prManifest.attestation.browser = provenance.browser;
  mainManifest.attestation.deterministicRendering = provenance.deterministicRendering;
  prManifest.attestation.deterministicRendering = provenance.deterministicRendering;
  mainManifest.attestation.scenarioContractSchemaVersion = provenance.scenarioContractSchemaVersion;
  prManifest.attestation.scenarioContractSchemaVersion = provenance.scenarioContractSchemaVersion;
  const captures = {
    base: mainManifest.attestation,
    head: prManifest.attestation,
  };
  const scenarioConfig = loadScenarioConfig(scenarioConfigPath, {
    base: { ...captures.base, label: mainManifest.label || 'main (base)' },
    head: { ...captures.head, label: prManifest.label || 'PR (head)' },
  });
  const outputFilenames = [];
  const outputFilenameOwners = new Map();
  for (const pair of manifestPlan.pairs) {
    pair.artifactFilenames = comparisonArtifactFilenames(pair.filename);
    for (const filename of Object.values(pair.artifactFilenames)) {
      const previousOwner = outputFilenameOwners.get(filename);
      if (previousOwner) {
        throw new ComparisonError(
          `Comparison artifact filename ${filename} collides between ${previousOwner} and ${pair.filename}.`,
          'manifest',
        );
      }
      outputFilenameOwners.set(filename, pair.filename);
      outputFilenames.push(filename);
    }
  }

  return {
    ...manifestPlan,
    captures: {
      base: mainManifest.attestation,
      head: prManifest.attestation,
    },
    mainLabel: mainManifest.label || 'main (base)',
    outputFilenames: outputFilenames.sort(),
    prLabel: prManifest.label || 'PR (head)',
    scenarioConfig,
    sourceMode: 'manifest',
  };
}

function ruleMatchesResult(rule, result) {
  if (rule.semanticScenario !== result.semanticScenario) return false;
  if (!rule.viewport) return true;
  return (
    result.viewport?.width === rule.viewport.width &&
    result.viewport?.height === rule.viewport.height
  );
}

function scenarioPolicy(result, rules, options) {
  const matching = rules.filter((rule) => ruleMatchesResult(rule, result));
  const exact = matching.find((rule) => rule.viewport);
  const generic = matching.find((rule) => !rule.viewport);
  const configured = Boolean(exact || generic);
  const configuredValue = (name) => exact?.[name] ?? generic?.[name];
  const configuredNullableValue = (name, fallback) => {
    if (exact && Object.hasOwn(exact, name)) return exact[name];
    if (generic && Object.hasOwn(generic, name)) return generic[name];
    return fallback;
  };
  return {
    diffThreshold: configuredValue('diffThreshold') ?? options.diffThreshold,
    failThreshold: configuredNullableValue(
      'failThreshold',
      result.expectedRemoval ? null : options.failThreshold,
    ),
    looksSameTolerance: configuredValue('looksSameTolerance') ?? options.looksSameTolerance,
    // A viewport-specific rule owns its full mask set; an empty array intentionally clears
    // masks inherited from the scenario-wide rule.
    masks: exact && Object.hasOwn(exact, 'masks') ? exact.masks : generic?.masks || [],
    expectedChange: configuredNullableValue('expectedChange', result.expectedChange ?? null),
    source: configured
      ? 'scenario-config'
      : result.expectedRemoval
        ? 'expected-removal-contract'
        : 'global-defaults',
  };
}

function applyScenarioPolicies(plan, options) {
  const allResults = [...plan.pairs, ...plan.results];
  for (const rule of plan.scenarioConfig.rules) {
    if (!allResults.some((result) => ruleMatchesResult(rule, result))) {
      const viewport = rule.viewport ? ` at ${rule.viewport.width}x${rule.viewport.height}` : '';
      throw new ComparisonError(
        `Scenario config rule ${rule.semanticScenario}${viewport} did not match any capture result.`,
        'config',
      );
    }
  }
  for (const result of allResults) {
    result.policy = scenarioPolicy(result, plan.scenarioConfig.rules, options);
    result.expectedChange = result.policy.expectedChange;
  }
  return plan;
}

function classifyPairFailure(failureType) {
  if (failureType === 'config') {
    return { captureValidity: 'valid', comparisonValidity: 'config-invalid' };
  }
  if (failureType === 'missing') {
    return { captureValidity: 'infrastructure_failure', comparisonValidity: 'not-compared' };
  }
  if (['stale', 'integrity', 'corrupt'].includes(failureType)) {
    return { captureValidity: 'infrastructure_failure', comparisonValidity: 'not-compared' };
  }
  return { captureValidity: 'valid', comparisonValidity: 'pixel-comparison-failure' };
}

function validateFreshCapture(record, filePath, side) {
  if (!record) {
    return null;
  }
  let stat;
  try {
    stat = fs.statSync(filePath);
  } catch (error) {
    throw new ComparisonError(`${side} screenshot is missing: ${filePath}`, 'missing');
  }
  if (!stat.isFile()) {
    throw new ComparisonError(`${side} screenshot is not a file: ${filePath}`, 'missing');
  }
  if (stat.mtimeMs < record.startedAtMs - FRESHNESS_TOLERANCE_MS) {
    throw new ComparisonError(
      `${side} screenshot predates its capture manifest: ${filePath}`,
      'stale',
    );
  }
  if (Math.abs(stat.mtimeMs - record.capturedAtMs) > FRESHNESS_TOLERANCE_MS) {
    throw new ComparisonError(
      `${side} screenshot timestamp does not match its capture manifest: ${filePath}`,
      'stale',
    );
  }
  let contents;
  try {
    contents = fs.readFileSync(filePath);
  } catch (error) {
    throw new ComparisonError(
      `${side} screenshot could not be hashed: ${error.message}`,
      'integrity',
    );
  }
  const actualSha256 = crypto.createHash('sha256').update(contents).digest('hex');
  if (actualSha256 !== record.sha256) {
    throw new ComparisonError(
      `${side} screenshot content does not match its capture manifest: ${filePath}`,
      'integrity',
    );
  }
  return contents;
}

function validateManagedOutputFilenames(filenames, label) {
  if (!Array.isArray(filenames) || filenames.length > 1000) {
    throw new Error(`${label} must contain an array of at most 1000 PNG filenames.`);
  }
  const unique = new Set();
  for (const filename of filenames) {
    if (
      typeof filename !== 'string' ||
      !filename.endsWith('.png') ||
      path.basename(filename) !== filename ||
      unique.has(filename)
    ) {
      throw new Error(`${label} contains an invalid or duplicate filename: ${filename}`);
    }
    unique.add(filename);
  }
  return [...unique];
}

function writeManagedOutputMarker(outputDir, filenames) {
  fs.writeFileSync(
    path.join(outputDir, MANAGED_OUTPUTS_FILENAME),
    `${JSON.stringify(
      {
        schemaVersion: MANAGED_OUTPUTS_SCHEMA_VERSION,
        artifacts: MANAGED_STATIC_OUTPUTS,
        filenames,
      },
      null,
      2,
    )}\n`,
  );
}

function existingOutputStat(outputPath) {
  try {
    return fs.lstatSync(outputPath);
  } catch (error) {
    if (error.code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

function cleanComparisonOutputs(outputDir, currentFilenames = []) {
  fs.mkdirSync(outputDir, { recursive: true });
  const nextFilenames = validateManagedOutputFilenames(currentFilenames, 'Comparison output list');
  const markerPath = path.join(outputDir, MANAGED_OUTPUTS_FILENAME);
  const entries = fs.readdirSync(outputDir, { withFileTypes: true });
  const removed = [];

  let previousFilenames = [];
  let previousStaticOutputs = [];
  if (entries.length > 0) {
    let markerStat;
    try {
      markerStat = fs.lstatSync(markerPath);
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`Refusing to clean non-empty unowned comparison directory: ${outputDir}`);
      }
      throw error;
    }
    if (!markerStat.isFile() || markerStat.isSymbolicLink()) {
      throw new Error(`Comparison ownership marker must be a regular file: ${markerPath}`);
    }
    let marker;
    try {
      marker = JSON.parse(fs.readFileSync(markerPath, 'utf8'));
    } catch (error) {
      throw new Error(`Comparison ownership marker is invalid: ${error.message}`);
    }
    if (marker?.schemaVersion !== 1 && marker?.schemaVersion !== MANAGED_OUTPUTS_SCHEMA_VERSION) {
      throw new Error(`Unsupported comparison ownership marker in ${markerPath}.`);
    }
    previousFilenames = validateManagedOutputFilenames(
      marker.filenames,
      'Comparison ownership marker',
    );
    if (marker.schemaVersion === 1) {
      previousStaticOutputs = [COMPARISON_SUMMARY_FILENAME];
    } else {
      if (
        !Array.isArray(marker.artifacts) ||
        marker.artifacts.length !== MANAGED_STATIC_OUTPUTS.length ||
        marker.artifacts.some((filename, index) => filename !== MANAGED_STATIC_OUTPUTS[index])
      ) {
        throw new Error(`Comparison ownership marker has invalid managed artifacts: ${markerPath}`);
      }
      previousStaticOutputs = MANAGED_STATIC_OUTPUTS;
    }
  }

  const previouslyManaged = new Set([...previousFilenames, ...previousStaticOutputs]);
  for (const filename of [...nextFilenames, ...MANAGED_STATIC_OUTPUTS]) {
    const target = path.join(outputDir, filename);
    if (existingOutputStat(target) && !previouslyManaged.has(filename)) {
      throw new Error(`Refusing to overwrite an unmanaged comparison output: ${filename}`);
    }
  }

  for (const filename of previouslyManaged) {
    const target = path.join(outputDir, filename);
    try {
      const stat = fs.lstatSync(target);
      if (!stat.isFile() && !stat.isSymbolicLink()) {
        throw new Error(`Managed comparison output is not a file: ${target}`);
      }
      fs.unlinkSync(target);
      removed.push(filename);
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
    }
  }
  writeManagedOutputMarker(outputDir, nextFilenames);
  return removed;
}

function summarizeComparison({
  captures,
  fatalErrors,
  mainLabel,
  options,
  prLabel,
  results,
  scenarioConfig,
  sourceMode,
}) {
  const normalizedResults = results.map((result) => {
    const { policy, ...rest } = result;
    return {
      ...rest,
      expectedChange: policy?.expectedChange ?? result.expectedChange ?? null,
      masks: policy?.masks || result.masks || [],
      scenarioThresholds: {
        diffThreshold: policy?.diffThreshold ?? options.diffThreshold,
        failThreshold:
          policy && Object.hasOwn(policy, 'failThreshold')
            ? policy.failThreshold
            : options.failThreshold,
        looksSameTolerance: policy?.looksSameTolerance ?? options.looksSameTolerance,
        source: policy?.source || 'global-defaults',
      },
    };
  });
  const orderedResults = normalizedResults.sort((left, right) => {
    if (left.filename === right.filename) return 0;
    return left.filename < right.filename ? -1 : 1;
  });
  const failed = orderedResults.filter((result) => result.status === 'failed');
  const success = orderedResults.filter((result) => result.status === 'success');
  const pagesExceedingFailThreshold = success.filter((result) => result.exceedsFailThreshold);
  const valid = fatalErrors.length === 0 && failed.length === 0 && success.length > 0;
  const passed = valid && pagesExceedingFailThreshold.length === 0;

  return {
    schemaVersion: COMPARISON_SUMMARY_SCHEMA_VERSION,
    timestamp: new Date().toISOString(),
    mainLabel,
    prLabel,
    sourceMode,
    captures,
    scenarioConfig,
    thresholds: {
      diffThreshold: options.diffThreshold,
      failThreshold: options.failThreshold,
    },
    analysis: {
      looksSameClusterSize: options.looksSameClusterSize,
      looksSameTolerance: options.looksSameTolerance,
    },
    fatalErrors,
    results: orderedResults,
    stats: {
      total: orderedResults.length,
      success: success.length,
      skipped: orderedResults.filter((result) => result.status === 'skipped').length,
      failed: failed.length,
      missing: failed.filter((result) => result.failureType === 'missing').length,
      stale: failed.filter((result) => result.failureType === 'stale').length,
      corrupt: failed.filter((result) => result.failureType === 'corrupt').length,
      integrity: failed.filter((result) => result.failureType === 'integrity').length,
      dimensionMismatch: failed.filter((result) => result.failureType === 'dimension-mismatch')
        .length,
      analysisFailed: failed.filter((result) => result.failureType === 'analysis').length,
      pagesWithDiff: success.filter((result) => result.hasVisualDiff).length,
      pagesExceedingFailThreshold: pagesExceedingFailThreshold.length,
      validSemanticPairs: success.length,
      thresholdEvaluations: success.filter((result) => result.thresholdsEvaluated).length,
      incompleteCaptures: orderedResults.filter(
        (result) =>
          result.captureValidity !== 'valid' &&
          result.captureValidity !== 'expected_product_removal',
      ).length,
      degradedCaptures: orderedResults.filter(
        (result) => result.captureValidity === 'selector_drift',
      ).length,
      expectedProductRemovals: orderedResults.filter(
        (result) => result.captureValidity === 'expected_product_removal',
      ).length,
      pixelComparisonFailures: failed.filter(
        (result) => result.comparisonValidity === 'pixel-comparison-failure',
      ).length,
    },
    valid,
    passed,
  };
}

function imageDataUrl(contents) {
  return `data:image/png;base64,${contents.toString('base64')}`;
}

function renderCaptureAttestation(role, attestation) {
  if (!attestation) {
    return `<section class="capture"><h2>${escapeHtml(role)}</h2><p>Capture manifest unavailable.</p></section>`;
  }
  return `<section class="capture"><h2>${escapeHtml(role)}</h2><dl><dt>Capture ID</dt><dd><code>${escapeHtml(attestation.captureId)}</code></dd><dt>Manifest SHA-256</dt><dd><code>${escapeHtml(attestation.manifestSha256)}</code></dd><dt>Manifest size</dt><dd>${attestation.manifestSizeBytes} bytes</dd><dt>Required successful screenshots</dt><dd>${attestation.requiredFilenames.length}</dd></dl></section>`;
}

function renderImageFigure(caption, alt, contents, artifact) {
  const dataUrl = imageDataUrl(contents);
  const attestation = artifact
    ? `<small><code>${escapeHtml(artifact.filename)}</code><br>SHA-256: <code>${escapeHtml(artifact.sha256)}</code> · ${artifact.sizeBytes} bytes</small>`
    : '';
  return `<figure><figcaption>${escapeHtml(caption)}</figcaption>${attestation}<a href="${dataUrl}" title="Open the full-size embedded image"><img src="${dataUrl}" alt="${escapeHtml(alt)}" loading="lazy"></a></figure>`;
}

function annotationText(value) {
  if (value === null || value === undefined || value === '') return 'None';
  return typeof value === 'string' ? value : JSON.stringify(canonicalizeJson(value));
}

function renderRoute(role, route) {
  const requested = route?.requested || 'not recorded';
  const resolved = route?.resolved || 'not recorded';
  const expectation = annotationText(route?.expectation);
  return `<div><strong>${escapeHtml(role)} route</strong><code>${escapeHtml(requested)}</code><br><span>Resolved: <code>${escapeHtml(resolved)}</code></span><br><span>Expectation: ${escapeHtml(expectation)}</span></div>`;
}

function renderCaptureEvidence(role, evidence) {
  if (!evidence) {
    return `<details><summary>${escapeHtml(role)} capture evidence</summary><p>No capture record.</p></details>`;
  }
  return `<details><summary>${escapeHtml(role)} capture evidence (${escapeHtml(evidence.status)})</summary><pre>${escapeHtml(JSON.stringify(evidence, null, 2))}</pre></details>`;
}

function renderComparisonResult(result, embeddedImages) {
  let statusDetail;
  if (result.status === 'success') {
    const thresholdState =
      result.scenarioThresholds.failThreshold === null
        ? 'failure threshold disabled'
        : `${result.exceedsFailThreshold ? 'above' : 'within'} the ${result.scenarioThresholds.failThreshold}% failure threshold`;
    statusDetail = `${result.diffPercent.toFixed(4)}% visual difference across ${result.comparablePixels} unmasked pixel(s); ${result.diffRegionCount} highlighted region(s); ${thresholdState}.`;
  } else if (result.status === 'skipped') {
    statusDetail = result.reason;
  } else {
    statusDetail = `${result.failureType || 'comparison'}: ${result.error}`;
  }

  let images = '';
  if (result.artifacts) {
    const imageSet = embeddedImages.get(result.filename);
    if (!imageSet) {
      throw new Error(`Missing embedded report images for ${result.filename}.`);
    }
    images = `<div class="images">${renderImageFigure('Base copy', `Base capture for ${result.semanticScenario}`, imageSet.base, result.artifacts.base)}${renderImageFigure('Head copy', `Head capture for ${result.semanticScenario}`, imageSet.head, result.artifacts.head)}${renderImageFigure('50/50 overlay', `Overlay for ${result.semanticScenario}`, imageSet.overlay, result.artifacts.overlay)}${renderImageFigure('Raw masked diff', `Raw masked diff for ${result.semanticScenario}`, imageSet.rawDiff, result.artifacts.rawDiff)}${renderImageFigure('Highlighted comparison', `Side-by-side highlighted comparison for ${result.semanticScenario}`, imageSet.highlightedDiff, result.artifacts.highlightedDiff)}</div>`;
  }

  const masks = result.masks.length
    ? `<ol class="masks">${result.masks.map((mask) => `<li><code>x=${mask.x}, y=${mask.y}, width=${mask.width}, height=${mask.height}</code>${mask.reason ? ` — ${escapeHtml(mask.reason)}` : ''}</li>`).join('')}</ol>`
    : '<p>No masked regions.</p>';
  const baseCaptureValidity = result.captureValidityByRevision?.base || 'unknown';
  const headCaptureValidity = result.captureValidityByRevision?.head || 'unknown';
  return `<section class="result status-${escapeHtml(result.status)}"><h2>${escapeHtml(result.scenarioTitle || result.semanticScenario || result.page)}</h2><p><code>${escapeHtml(result.semanticScenario || result.page)}</code></p><p class="metadata"><span>Status: ${escapeHtml(result.status)}</span><span>Pair capture validity: ${escapeHtml(result.captureValidity || 'unknown')}</span><span>Base capture validity: ${escapeHtml(baseCaptureValidity)}</span><span>Head capture validity: ${escapeHtml(headCaptureValidity)}</span><span>Comparison validity: ${escapeHtml(result.comparisonValidity || 'unknown')}</span><span>${result.required ? 'Required' : 'Optional'}</span><span>Thresholds evaluated: ${result.thresholdsEvaluated ? 'yes' : 'no'}</span></p><div class="routes">${renderRoute('Base', result.routes?.base)}${renderRoute('Head', result.routes?.head)}</div><div class="capture-evidence">${renderCaptureEvidence('Base', result.captureEvidenceByRevision?.base)}${renderCaptureEvidence('Head', result.captureEvidenceByRevision?.head)}</div><p>${escapeHtml(statusDetail)}</p><p><strong>Expected change:</strong> ${escapeHtml(annotationText(result.expectedChange))}</p><p><strong>Scenario tolerances:</strong> diff ${result.scenarioThresholds.diffThreshold}%; fail ${result.scenarioThresholds.failThreshold === null ? 'disabled' : `${result.scenarioThresholds.failThreshold}%`}; pixel tolerance ${result.scenarioThresholds.looksSameTolerance}; source ${escapeHtml(result.scenarioThresholds.source)}.</p><div><strong>Masks (${result.masks.length})</strong>${masks}</div>${images}</section>`;
}

function createComparisonReport(summary, embeddedImages) {
  const overallStatus = summary.passed ? 'passed' : 'failed';
  const fatalErrors = summary.fatalErrors.length
    ? `<section class="errors"><h2>Fatal errors</h2><ul>${summary.fatalErrors.map((error) => `<li>${escapeHtml(error)}</li>`).join('')}</ul></section>`
    : '';
  const results = summary.results.length
    ? summary.results.map((result) => renderComparisonResult(result, embeddedImages)).join('')
    : '<section class="empty"><h2>No comparison results</h2><p>No valid screenshot pairs were available.</p></section>';
  const baseCapture = renderCaptureAttestation('Base capture', summary.captures?.base);
  const headCapture = renderCaptureAttestation('Head capture', summary.captures?.head);
  const scenarioConfig = summary.scenarioConfig
    ? `<section class="capture"><h2>Scenario policy</h2><dl><dt>Schema</dt><dd><code>${escapeHtml(summary.scenarioConfig.schemaVersion)}</code></dd><dt>SHA-256</dt><dd><code>${escapeHtml(summary.scenarioConfig.sha256)}</code></dd><dt>Size</dt><dd>${summary.scenarioConfig.sizeBytes} bytes</dd><dt>Base binding</dt><dd><code>${escapeHtml(annotationText(summary.scenarioConfig.revisionPair?.base))}</code></dd><dt>Head binding</dt><dd><code>${escapeHtml(annotationText(summary.scenarioConfig.revisionPair?.head))}</code></dd></dl></section>`
    : '<section class="capture"><h2>Scenario policy</h2><p>Global defaults; no external scenario config.</p></section>';

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'">
  <title>UI smoke comparison: ${escapeHtml(overallStatus)}</title>
  <style>
    :root{color-scheme:light;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f5f6fa;color:#171821}body{max-width:1800px;margin:0 auto;padding:24px}h1,h2{line-height:1.25}code{overflow-wrap:anywhere}.overview,.captures,.metadata,.images,.routes,.capture-evidence{display:grid;gap:12px}.overview{grid-template-columns:repeat(auto-fit,minmax(190px,1fr));margin:20px 0}.overview div,.capture,.result,.errors,.empty,.routes>div{background:#fff;border:1px solid #d9dce7;border-radius:8px;padding:16px}.overview strong,.metadata span,.routes strong{display:block}.captures{grid-template-columns:repeat(auto-fit,minmax(320px,1fr));margin-bottom:20px}.capture h2,.result h2{margin-top:0}.capture dl{display:grid;grid-template-columns:max-content 1fr;gap:6px 12px;margin:0}.capture dd{margin:0;min-width:0}.metadata{grid-template-columns:repeat(auto-fit,minmax(160px,max-content));color:#4b5063}.routes,.capture-evidence{grid-template-columns:repeat(auto-fit,minmax(280px,1fr))}.capture-evidence{margin:12px 0}.capture-evidence pre{white-space:pre-wrap;overflow-wrap:anywhere}.images{grid-template-columns:repeat(auto-fit,minmax(300px,1fr));align-items:start}figure{margin:0}figcaption{font-weight:650;margin-bottom:4px}figure small{display:block;min-height:3.5em;margin-bottom:8px;color:#4b5063}img{display:block;width:100%;height:auto;border:1px solid #c8cbd7;background:#fff}a:focus{outline:3px solid #3157d5;outline-offset:3px}.status-failed{border-left:5px solid #b42318}.status-skipped{border-left:5px solid #b7791f}.status-success{border-left:5px solid #16803c}.errors{border-left:5px solid #b42318}.errors li+li{margin-top:8px}.masks{margin-top:6px}@media(max-width:700px){body{padding:12px}.capture dl{grid-template-columns:1fr}.capture dt{font-weight:650}}
  </style>
</head>
<body>
  <header>
    <h1>UI smoke comparison</h1>
    <div class="overview"><div><strong>Overall status</strong>${escapeHtml(overallStatus)}</div><div><strong>Comparison validity</strong>${summary.valid ? 'valid' : 'invalid'}</div><div><strong>Base revision</strong>${escapeHtml(summary.mainLabel)}</div><div><strong>Head revision</strong>${escapeHtml(summary.prLabel)}</div><div><strong>Valid semantic pairs</strong>${summary.stats.validSemanticPairs}</div><div><strong>Threshold evaluations</strong>${summary.stats.thresholdEvaluations}</div><div><strong>Incomplete captures</strong>${summary.stats.incompleteCaptures}</div><div><strong>Expected removals</strong>${summary.stats.expectedProductRemovals}</div><div><strong>Pairs with visual differences</strong>${summary.stats.pagesWithDiff}</div><div><strong>Pairs above failure threshold</strong>${summary.stats.pagesExceedingFailThreshold}</div></div>
  </header>
  <div class="captures">${baseCapture}${headCapture}${scenarioConfig}</div>
  ${fatalErrors}
  <main>${results}</main>
</body>
</html>
`;
}

function logSummary(summary, summaryPath) {
  console.log('\n--- Summary ---');
  console.log('Scenario-specific visual changes are boxed in red (looks-same clusters).');
  for (const result of summary.results) {
    if (result.status === 'success') {
      const visualMarker = result.hasVisualDiff ? ' [diff]' : '';
      const failMarker = result.exceedsFailThreshold ? ' [above fail-threshold]' : '';
      console.log(
        `  ${result.page}: ✓ (${result.diffPercent.toFixed(2)}% diff)${visualMarker}${failMarker}`,
      );
    } else if (result.status === 'skipped') {
      console.log(`  ${result.page}: ↷ ${result.reason}`);
    } else {
      console.log(`  ${result.page}: ✗ ${result.error}`);
    }
  }
  for (const error of summary.fatalErrors) {
    console.error(`  ✗ ${error}`);
  }
  console.log(`\nSummary saved to: ${summaryPath}`);
}

async function runComparison(options, dependencies = {}) {
  validateDistinctDirectories(options);
  const compareImages = dependencies.looksSame || looksSame;
  const fatalErrors = validateComparisonOptions(options);
  const results = [];
  const embeddedImages = new Map();
  let captures = { base: null, head: null };
  let mainLabel = 'main (base)';
  let prLabel = 'PR (head)';
  let scenarioConfig = null;
  let sourceMode = 'unknown';

  console.log('Generating side-by-side comparisons');
  console.log(`Main screenshots: ${options.mainDir}`);
  console.log(`PR screenshots: ${options.prDir}`);
  console.log(`Output: ${options.outputDir}`);

  fs.mkdirSync(options.outputDir, { recursive: true });
  cleanComparisonOutputs(options.outputDir);

  if (fatalErrors.length === 0) {
    try {
      const plan = buildComparisonPlan(
        options.mainDir,
        options.prDir,
        options.scenarioConfigPath || null,
      );
      mainLabel = options.mainLabel || plan.mainLabel;
      prLabel = options.prLabel || plan.prLabel;
      sourceMode = plan.sourceMode;
      captures = plan.captures;
      scenarioConfig = plan.scenarioConfig.attestation;
      applyScenarioPolicies(plan, options);
      cleanComparisonOutputs(options.outputDir, plan.outputFilenames);
      results.push(...plan.results);

      if (plan.filenames.length === 0) {
        fatalErrors.push('No screenshots found to compare.');
      }

      for (const pair of plan.pairs) {
        const outputPath = path.join(options.outputDir, pair.artifactFilenames.highlightedDiff);
        try {
          const baseImage = validateFreshCapture(pair.mainRecord, pair.mainPath, 'Base');
          const headImage = validateFreshCapture(pair.prRecord, pair.prPath, 'Head');
          const diffAnalysis = await analyzeDiff(
            baseImage,
            headImage,
            {
              ...options,
              looksSameTolerance: pair.policy.looksSameTolerance,
              masks: pair.policy.masks,
            },
            compareImages,
          );
          const hasVisualDiff = diffAnalysis.diffPercent > pair.policy.diffThreshold;
          const exceedsFailThreshold =
            pair.policy.failThreshold !== null &&
            diffAnalysis.diffPercent > pair.policy.failThreshold;
          const artifacts = await generateArtifactSet({
            baseImage,
            diffAnalysis,
            filenames: pair.artifactFilenames,
            hasVisualDiff,
            headImage,
            mainLabel,
            outputDir: options.outputDir,
            page: pair.scenarioTitle,
            prLabel,
          });
          embeddedImages.set(pair.filename, {
            ...Object.fromEntries(
              Object.entries(artifacts).map(([name, artifact]) => [
                name,
                fs.readFileSync(path.join(options.outputDir, artifact.filename)),
              ]),
            ),
          });
          const commonResult = {
            artifacts,
            comparablePixels: diffAnalysis.comparablePixels,
            filename: pair.filename,
            page: pair.page,
            required: pair.required,
            outputPath,
            mainExists: true,
            prExists: true,
            diffPercent: diffAnalysis.diffPercent,
            diffRegionCount: hasVisualDiff ? diffAnalysis.regions.length : 0,
            policy: pair.policy,
            semanticScenario: pair.semanticScenario,
            scenarioTitle: pair.scenarioTitle,
            captureValidityByRevision: pair.captureValidityByRevision,
            captureEvidenceByRevision: pair.captureEvidenceByRevision,
            routes: pair.routes,
            viewport: pair.viewport,
          };
          if (pair.expectedRemoval) {
            results.push({
              ...commonResult,
              captureValidity: 'expected_product_removal',
              comparisonValidity: 'expected-change',
              hasVisualDiff,
              exceedsFailThreshold,
              reason: 'Semantic scenario records an expected product removal.',
              status: 'success',
              thresholdsEvaluated: true,
            });
          } else {
            results.push({
              ...commonResult,
              captureValidity: 'valid',
              comparisonValidity: 'valid',
              hasVisualDiff,
              exceedsFailThreshold,
              status: 'success',
              thresholdsEvaluated: true,
            });
          }
        } catch (error) {
          console.error(`  ✗ Failed: ${error.message}`);
          for (const filename of Object.values(pair.artifactFilenames)) {
            try {
              const artifactPath = path.join(options.outputDir, filename);
              if (fs.existsSync(artifactPath)) {
                fs.unlinkSync(artifactPath);
              }
            } catch (cleanupError) {
              console.error(
                `  ✗ Failed to remove incomplete output ${filename}: ${cleanupError.message}`,
              );
            }
          }
          const failureClassification = classifyPairFailure(error.failureType || 'comparison');
          const pairCaptureValidity = aggregateCaptureValidity(pair.captureValidityByRevision);
          results.push({
            ...failureClassification,
            captureValidity:
              failureClassification.captureValidity === 'valid'
                ? pairCaptureValidity
                : failureClassification.captureValidity,
            filename: pair.filename,
            page: pair.page,
            required: pair.required,
            status: 'failed',
            error: error.message,
            failureType: error.failureType || 'comparison',
            thresholdsEvaluated: false,
            policy: pair.policy,
            semanticScenario: pair.semanticScenario,
            scenarioTitle: pair.scenarioTitle,
            captureValidityByRevision: pair.captureValidityByRevision,
            captureEvidenceByRevision: pair.captureEvidenceByRevision,
            routes: pair.routes,
            viewport: pair.viewport,
          });
        }
      }
    } catch (error) {
      fatalErrors.push(error.message);
    }
  }

  const summary = summarizeComparison({
    captures,
    fatalErrors,
    mainLabel,
    options,
    prLabel,
    results,
    scenarioConfig,
    sourceMode,
  });
  const summaryPath = path.join(options.outputDir, COMPARISON_SUMMARY_FILENAME);
  const reportPath = path.join(options.outputDir, COMPARISON_REPORT_FILENAME);
  const report = createComparisonReport(summary, embeddedImages);
  fs.writeFileSync(summaryPath, `${JSON.stringify(summary, null, 2)}\n`, { flag: 'wx' });
  fs.writeFileSync(reportPath, report, { flag: 'wx' });
  logSummary(summary, summaryPath);
  console.log(`Report saved to: ${reportPath}`);

  if (summary.stats.pagesExceedingFailThreshold > 0) {
    const failures = summary.results
      .filter((result) => result.status === 'success' && result.exceedsFailThreshold)
      .map(
        (result) =>
          `${result.semanticScenario} (${result.diffPercent.toFixed(4)}% > ${result.scenarioThresholds.failThreshold}%)`,
      );
    console.error(`\nVisual diff threshold exceeded: ${failures.join(', ')}`);
  }

  return { exitCode: summary.passed ? 0 : 1, reportPath, summary, summaryPath };
}

async function main(args = process.argv.slice(2), env = process.env) {
  let options;
  try {
    options = parseComparisonOptions(args, env);
  } catch (error) {
    console.error('Comparison generation failed:', error.message);
    process.exitCode = 1;
    return { exitCode: 1, error };
  }

  try {
    const result = await runComparison(options);
    process.exitCode = result.exitCode;
    return result;
  } catch (error) {
    console.error('Comparison generation failed:', error.message);
    process.exitCode = 1;
    return { exitCode: 1, error };
  }
}

module.exports = {
  CAPTURE_MANIFEST_SCHEMA_VERSION,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
  COMPARISON_REPORT_FILENAME,
  COMPARISON_SUMMARY_SCHEMA_VERSION,
  ComparisonError,
  SCENARIO_CONFIG_SCHEMA_VERSION,
  SCENARIO_POLICY_SCHEMA_VERSION,
  analyzeDiff,
  buildComparisonPlan,
  cleanComparisonOutputs,
  createComparisonReport,
  createDiffOverlay,
  deriveDiffPercent,
  extractDiffRegions,
  normalizeRegion,
  parseComparisonOptions,
  runComparison,
  summarizeComparison,
  validateCaptureManifest,
  normalizeSemanticIdNormalizationAttestation,
  validateSemanticNormalizationAgainstCatalog,
  validateSemanticIdNormalizationScenarioContracts,
  validateComparisonOptions,
  validateDistinctDirectories,
  validateFreshCapture,
  validateGlobalVisualNormalizationEvidence,
  writeBoundScenarioConfig,
};

if (require.main === module) {
  void main();
}
