#!/usr/bin/env node
/**
 * Screenshot Capture Script for Kubeflow Pipelines UI
 *
 * Captures screenshots of key UI pages for visual comparison testing.
 * Uses Playwright for browser automation.
 *
 * Usage: node capture-screenshots.js --base-url http://localhost:4001 --output ./screenshots --label main
 */

const crypto = require('crypto');
const path = require('path');
const fs = require('fs');
const {
  SCENARIO_CONTRACT_SCHEMA_VERSION,
  getGlobalVisualNormalizationContract,
  globalExpectedChangeAnnotation,
  resolveSemanticScenarios,
} = require('./semantic-capture-scenarios.js');
const {
  SEMANTIC_COLOR_PALETTE,
  SEMANTIC_ID_KINDS,
  SEMANTIC_ID_NORMALIZATION_MODES,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
  SEMANTIC_ID_PATH_PATTERN,
  SEMANTIC_ID_TOKEN_PATTERN,
  semanticIdNormalizationRenderingContract,
  semanticIdToken,
} = require('./semantic-id-normalization.js');
const {
  COMPARISON_RUN_FIXTURES,
  REVISION_FLAVORS,
  SEMANTIC_FIXTURE_SET,
  SEMANTIC_SCHEMA_VERSION,
  TASK_FIXTURES,
  validateRevisionSemanticManifest,
} = require('./semantic-manifest.js');

const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_SEED_MANIFEST = path.join(REPO_ROOT, '.ui-smoke-test', 'seed-manifest.json');
const CAPTURE_MANIFEST_FILENAME = 'manifest.json';
const CAPTURE_MANIFEST_SCHEMA_VERSION = 3;
const CAPTURE_OWNER_FILENAME = '.ui-smoke-capture-managed.json';
const CAPTURE_OWNER_SCHEMA_VERSION = 1;
const CAPTURE_STATUSES = new Set(['success', 'degraded', 'skipped', 'failed']);
const CAPTURE_VALIDITIES = new Set([
  'valid',
  'ui_rendering_failure',
  'api_incompatibility',
  'seed_failure',
  'missing_fixture',
  'selector_drift',
  'expected_product_removal',
  'infrastructure_failure',
]);
const DETERMINISTIC_STYLE_ID = 'ui-smoke-test-deterministic-rendering';
const DETERMINISTIC_FONT_FAMILY = 'UI Smoke Roboto';
const DETERMINISTIC_FONT_PACKAGE = '@fontsource/roboto@5.3.0';
const DETERMINISTIC_FONT_ASSETS = Object.freeze(
  [400, 500, 700].map((weight) => {
    const filename = `roboto-latin-${weight}-normal.woff2`;
    const bytes = fs.readFileSync(require.resolve(`@fontsource/roboto/files/${filename}`));
    return Object.freeze({
      dataUrl: `data:font/woff2;base64,${bytes.toString('base64')}`,
      filename,
      sha256: crypto.createHash('sha256').update(bytes).digest('hex'),
      weight,
    });
  }),
);
const DETERMINISTIC_TIME_ISO = '2030-01-02T03:04:05.000Z';
const DETERMINISTIC_TIME_MS = Date.parse(DETERMINISTIC_TIME_ISO);
const DIAGNOSTIC_LIMIT = 20;
const DIAGNOSTIC_TEXT_LIMIT = 500;
const CAPTURE_ARGUMENT_NAMES = new Set([
  'base-url',
  'label',
  'normalization-mode',
  'output',
  'port',
  'revision-role',
  'seed-manifest',
  'semantic-manifest',
  'source-provenance',
]);
const DETERMINISTIC_CSS = `
  ${DETERMINISTIC_FONT_ASSETS.map(
    (asset) => `
      @font-face {
        font-family: "${DETERMINISTIC_FONT_FAMILY}";
        font-style: normal;
        font-weight: ${asset.weight};
        font-display: block;
        src: url("${asset.dataUrl}") format("woff2");
      }
    `,
  ).join('\n')}
  :root {
    color-scheme: light !important;
  }
  *, *::before, *::after {
    animation-delay: 0s !important;
    animation-duration: 0s !important;
    animation-iteration-count: 1 !important;
    caret-color: transparent !important;
    scroll-behavior: auto !important;
    transition-delay: 0s !important;
    transition-duration: 0s !important;
  }
  :where(
    html, body, button, input, select, textarea, div, p, a, dt, dd, h1, h2, h3, h4, h5, h6,
    label, li, table, text, span:not(.material-icons):not(.material-icons-outlined)
  ) {
    font-family: "${DETERMINISTIC_FONT_FAMILY}", sans-serif !important;
    font-synthesis: none !important;
    font-variant-ligatures: none !important;
  }
`;

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
  return seen;
}

function parseViewports(value) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error('At least one viewport is required.');
  }

  const seen = new Set();
  return value.split(',').map((raw) => {
    const trimmed = raw.trim();
    const match = /^([1-9]\d*)x([1-9]\d*)$/.exec(trimmed);
    if (!match) {
      throw new Error(`Invalid viewport "${trimmed}". Use WIDTHxHEIGHT (for example 1280x800).`);
    }
    const width = Number(match[1]);
    const height = Number(match[2]);
    if (!Number.isSafeInteger(width) || !Number.isSafeInteger(height)) {
      throw new Error(`Invalid viewport "${trimmed}": dimensions must be safe integers.`);
    }
    if (seen.has(trimmed)) {
      throw new Error(`Duplicate viewport "${trimmed}".`);
    }
    seen.add(trimmed);
    return { width, height };
  });
}

function normalizeBaseUrl(value) {
  let parsed;
  try {
    parsed = new URL(value);
  } catch (error) {
    throw new Error(`Invalid base URL "${value}": ${error.message}`);
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error(`Invalid base URL "${value}": protocol must be http or https.`);
  }

  parsed.hash = '';
  parsed.pathname = parsed.pathname.replace(/\/+$/, '') || '/';
  const normalized = parsed.toString();
  if (parsed.pathname !== '/') {
    return normalized;
  }
  const rootSuffix = `/${parsed.search}`;
  return `${normalized.slice(0, -rootSuffix.length)}${parsed.search}`;
}

function resolveCaptureUrl(baseUrl, routePath) {
  const resolved = new URL(normalizeBaseUrl(baseUrl));
  const hashIndex = routePath.indexOf('#');
  const routeBeforeHash = hashIndex === -1 ? routePath : routePath.slice(0, hashIndex);
  const normalizedRoutePath = routeBeforeHash.startsWith('/')
    ? routeBeforeHash
    : `/${routeBeforeHash}`;
  const basePath = resolved.pathname.replace(/\/+$/, '');
  resolved.pathname = `${basePath}${normalizedRoutePath}` || '/';
  if (hashIndex !== -1) {
    resolved.hash = routePath.slice(hashIndex + 1);
  }
  return resolved.toString();
}

function parseCaptureOptions(args = process.argv.slice(2), env = process.env) {
  const providedArguments = validateCliArguments(args, CAPTURE_ARGUMENT_NAMES);
  if (providedArguments.has('base-url') && providedArguments.has('port')) {
    throw new Error('--base-url and --port are mutually exclusive.');
  }
  const port = getArg(args, 'port', '4001');
  const baseUrl = normalizeBaseUrl(getArg(args, 'base-url', null) || `http://localhost:${port}`);
  const rawViewportEnv = env.UI_SMOKE_VIEWPORTS ?? env.UI_SMOKE_VIEWPORT;
  const viewportValue = rawViewportEnv && rawViewportEnv.trim() ? rawViewportEnv : '1280x800';
  const rawPageNames = env.UI_SMOKE_PAGES;

  const revisionRole = getArg(args, 'revision-role', null);
  if (revisionRole !== null && !['base', 'head', 'current'].includes(revisionRole)) {
    throw new Error('--revision-role must be base, head, or current.');
  }
  const semanticManifestPath = getArg(args, 'semantic-manifest', null);
  const sourceProvenancePath = getArg(args, 'source-provenance', null);
  const semanticIdNormalizationMode = getArg(args, 'normalization-mode', null);
  if (!Object.values(SEMANTIC_ID_NORMALIZATION_MODES).includes(semanticIdNormalizationMode)) {
    throw new Error(
      `--normalization-mode is required and must be ${Object.values(SEMANTIC_ID_NORMALIZATION_MODES).join(' or ')}.`,
    );
  }
  if ((semanticManifestPath || sourceProvenancePath) && !revisionRole) {
    throw new Error('--revision-role is required with capture provenance inputs.');
  }
  if (semanticIdNormalizationMode === SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK) {
    if (
      !semanticManifestPath ||
      !sourceProvenancePath ||
      !['base', 'head'].includes(revisionRole)
    ) {
      throw new Error(
        'semantic-full-stack normalization requires --revision-role base|head, --semantic-manifest, and --source-provenance.',
      );
    }
  } else if (semanticManifestPath || sourceProvenancePath) {
    throw new Error(
      'disabled-browser-compatibility normalization forbids semantic and source provenance inputs.',
    );
  }

  return {
    baseUrl,
    label: getArg(args, 'label', 'screenshot'),
    outputDir: getArg(args, 'output', './screenshots'),
    pageNames:
      rawPageNames && rawPageNames.trim()
        ? rawPageNames
            .split(',')
            .map((name) => name.trim())
            .filter(Boolean)
        : null,
    seedManifestPath: getArg(
      args,
      'seed-manifest',
      env.UI_SMOKE_SEED_MANIFEST || DEFAULT_SEED_MANIFEST,
    ),
    revisionRole,
    semanticIdNormalizationMode,
    semanticManifestPath,
    sourceProvenancePath,
    viewports: parseViewports(viewportValue),
  };
}

function loadAttestedJsonInput(filePath, description) {
  if (!filePath) return null;
  const resolvedPath = path.resolve(filePath);
  const stat = fs.lstatSync(resolvedPath);
  if (!stat.isFile() || stat.isSymbolicLink()) {
    throw new Error(`${description} must be a non-symlink regular file.`);
  }
  const contents = fs.readFileSync(resolvedPath);
  let value;
  try {
    value = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new Error(`${description} must contain valid JSON: ${error.message}`);
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${description} must contain a JSON object.`);
  }
  return {
    attestation: {
      path: resolvedPath,
      schemaVersion:
        typeof value.schemaVersion === 'string' || typeof value.schemaVersion === 'number'
          ? value.schemaVersion
          : null,
      sha256: crypto.createHash('sha256').update(contents).digest('hex'),
      sizeBytes: contents.length,
    },
    value,
  };
}

function attestJsonInput(filePath, description) {
  return loadAttestedJsonInput(filePath, description)?.attestation || null;
}

function seedValuesFromManifest(manifest) {
  const defaults = manifest.defaults || {};
  const resources = manifest.resources || {};
  const runIds = Array.isArray(resources.runIds) ? resources.runIds : [];
  const semanticBindings = manifest.semantic?.bindings || {};
  const semanticResources = semanticBindings.resources || {};
  const semanticRuns = semanticBindings.runs || {};
  const richRun = semanticRuns['run.training-1'] || {};
  const taskId = (semanticKey) => richRun.taskInstances?.[semanticKey]?.[0]?.taskId || null;
  const artifactId = (semanticKey) => richRun.artifacts?.[semanticKey]?.artifactIds?.[0] || null;
  const artifactMemberId = (semanticKey, memberKey) =>
    richRun.artifacts?.[semanticKey]?.members?.[memberKey]?.artifactIds?.[0] || null;
  const semanticComparisonRunIds = COMPARISON_RUN_FIXTURES.map(
    (semanticKey) => semanticResources[semanticKey]?.id,
  ).filter(Boolean);
  const semanticCompareRunlist =
    semanticComparisonRunIds.length === COMPARISON_RUN_FIXTURES.length
      ? semanticComparisonRunIds.join(',')
      : null;

  return {
    artifactId:
      defaults.artifactId ||
      artifactId('artifact.html-report') ||
      artifactId('artifact.scalar-metrics'),
    compareRunlist:
      semanticCompareRunlist || defaults.compareRunlist || runIds.slice(0, 3).join(','),
    consumeMetricsTaskId: taskId('task.consume-metrics'),
    executionId:
      defaults.executionId ||
      richRun.taskInstances?.['task.write-metrics']?.[0]?.mlmdExecutionId ||
      null,
    experimentId: defaults.experimentId || (resources.experimentIds || [])[0],
    htmlArtifactId: artifactId('artifact.html-report'),
    markdownArtifactId: artifactId('artifact.markdown-report'),
    nestedDagTaskId: taskId('task.nested-dag'),
    parallelTaskId: taskId('task.parallel-loop'),
    pipelineId: defaults.pipelineId || (resources.pipelineIds || [])[0],
    recurringRunId: defaults.recurringRunId || (resources.recurringRunIds || [])[0],
    relatedArtifactId:
      artifactMemberId('artifact.scalar-metrics', 'metric.accuracy') ||
      artifactId('artifact.scalar-metrics'),
    runId: defaults.runId || runIds[0],
    richRunId:
      semanticResources['run.training-1']?.id || richRun.runId || defaults.runId || runIds[0],
    retryTaskId: taskId('task.retry-once'),
    rocArtifactId: artifactId('artifact.roc-curve'),
    scalarArtifactId: artifactId('artifact.scalar-metrics'),
    taskId: defaults.taskId || taskId('task.write-metrics'),
    writeMetricsTaskId: taskId('task.write-metrics'),
  };
}

function loadSeedManifestInput(manifestPath, options = {}) {
  const required = options.required === true;
  if (!manifestPath || !fs.existsSync(manifestPath)) {
    if (required) {
      throw new Error(
        `Seed manifest is required and was not found: ${manifestPath || '(missing path)'}`,
      );
    }
    return null;
  }

  try {
    const input = loadAttestedJsonInput(manifestPath, 'Seed manifest');
    return { ...input, seedValues: seedValuesFromManifest(input.value) };
  } catch (error) {
    if (required) {
      throw new Error(`Failed to load seed manifest ${manifestPath}: ${error.message}`, {
        cause: error,
      });
    }
    console.log(`Warning: failed to parse seed manifest ${manifestPath}: ${error.message}`);
    return null;
  }
}

function loadSeedValues(manifestPath, options = {}) {
  return loadSeedManifestInput(manifestPath, options)?.seedValues || null;
}

function semanticIdNormalizationError(message, captureValidity = 'seed_failure') {
  const error = new Error(message);
  error.captureValidity = captureValidity;
  return error;
}

function validateSemanticIdNormalizationMode(options) {
  const mode = options?.semanticIdNormalizationMode;
  if (!Object.values(SEMANTIC_ID_NORMALIZATION_MODES).includes(mode)) {
    throw semanticIdNormalizationError(
      `Capture requires an explicit semantic ID normalization mode; received ${mode || '(missing)'}.`,
    );
  }
  if (mode === SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK) {
    if (
      !options.semanticManifestPath ||
      !options.sourceProvenancePath ||
      !['base', 'head'].includes(options.revisionRole)
    ) {
      throw semanticIdNormalizationError(
        'semantic-full-stack normalization requires base|head revision role plus semantic and source provenance.',
      );
    }
  } else if (options.semanticManifestPath || options.sourceProvenancePath) {
    throw semanticIdNormalizationError(
      'disabled-browser-compatibility normalization forbids semantic and source provenance.',
    );
  }
  return mode;
}

function identifierValues(value) {
  const values = Array.isArray(value) ? value : [];
  return [...new Set(values.map((entry) => String(entry || '')).filter(Boolean))].sort();
}

function semanticIdentifierSegment(value, fallback) {
  return (
    String(value || '')
      .toLowerCase()
      .replace(/[^a-z0-9.-]+/g, '-')
      .replace(/^[^a-z0-9]+|[^a-z0-9]+$/g, '') || fallback
  );
}

function executorLogAttemptIndex(uri) {
  if (typeof uri !== 'string' || uri.length === 0 || uri.trim() !== uri) return null;
  const match = uri.match(/(?:^|\/)executor-logs-(0|[1-9]\d*)$/);
  return match ? Number(match[1]) : null;
}

function legacyExecutionSemanticScope(runKey, executionKey, instance, index) {
  if (instance?.executionRole === 'run-root') return `${runKey}/execution.root[${index}]`;
  if (instance?.executionRole === 'loop-controller') {
    return `${runKey}/${executionKey}/controller`;
  }
  if (instance?.executionRole === 'loop-iteration') {
    return `${runKey}/${executionKey}/iteration[${instance.iterationIndex}]`;
  }
  return `${runKey}/${executionKey}[${index}]`;
}

function taskVisualSemanticId(runKey, taskKey, index) {
  return `${runKey}/${taskKey}[${index}]`;
}

function unjoinableLegacyTaskEquivalence(manifest, runKey, taskKey) {
  const baseRun = manifest.deployments?.base?.bindings?.runs?.[runKey];
  const runProfile = manifest.logical?.runProfiles?.[baseRun?.fixtureProfile];
  const instances = baseRun?.taskInstances?.[taskKey];
  if (
    baseRun?.revisionFlavor !== REVISION_FLAVORS.LEGACY ||
    baseRun?.lineageComplete !== true ||
    taskKey !== runProfile?.loop?.worker ||
    !Array.isArray(instances) ||
    instances.length !== runProfile.loop.iterations
  ) {
    return '';
  }
  if (
    instances.some(
      (instance) => instance?.mlmdExecutionId || Number.isSafeInteger(instance?.iterationIndex),
    )
  ) {
    return '';
  }
  return `${runKey}/${taskKey}/equivalent`;
}

function legacyExecutionVisualIdentity(runKey, executionKey, instance, index) {
  const executionScope = legacyExecutionSemanticScope(runKey, executionKey, instance, index);
  if (instance?.executionRole === 'run-root') {
    return { tokenKind: 'execution', tokenSemanticId: `${executionScope}/execution` };
  }
  if (instance?.executionRole === 'loop-controller') {
    return { tokenKind: 'task', tokenSemanticId: taskVisualSemanticId(runKey, executionKey, 0) };
  }
  if (instance?.executionRole === 'loop-iteration') {
    return { tokenKind: 'task', tokenSemanticId: executionScope };
  }
  const taskIndex = Number.isSafeInteger(instance?.iterationIndex)
    ? instance.iterationIndex
    : index;
  return {
    tokenKind: 'task',
    tokenSemanticId: taskVisualSemanticId(runKey, executionKey, taskIndex),
  };
}

function buildSemanticIdentifierCatalog(manifest, revisionRole) {
  if (revisionRole !== 'base' && revisionRole !== 'head') {
    throw semanticIdNormalizationError(
      `Semantic identifier bindings require revision role base or head, received ${revisionRole || '(missing)'}.`,
      'missing_fixture',
    );
  }
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) {
    throw semanticIdNormalizationError('Semantic fixture manifest must contain an object.');
  }
  if (manifest.schemaVersion !== SEMANTIC_SCHEMA_VERSION) {
    throw semanticIdNormalizationError(
      `Semantic fixture manifest must use schema ${SEMANTIC_SCHEMA_VERSION}.`,
    );
  }
  if (manifest.fixtureSet !== SEMANTIC_FIXTURE_SET) {
    throw semanticIdNormalizationError(
      `Semantic fixture manifest must use fixture set ${SEMANTIC_FIXTURE_SET}.`,
    );
  }
  const deployment = manifest.deployments?.[revisionRole];
  const bindings = deployment?.bindings;
  if (!bindings || typeof bindings !== 'object' || Array.isArray(bindings)) {
    throw semanticIdNormalizationError(
      `Semantic fixture manifest has no ${revisionRole} deployment bindings.`,
      'missing_fixture',
    );
  }
  if (
    !manifest.logical?.resources ||
    typeof manifest.logical.resources !== 'object' ||
    Array.isArray(manifest.logical.resources) ||
    Object.keys(manifest.logical.resources).length === 0 ||
    !bindings.resources ||
    !bindings.runs ||
    Object.keys(bindings.resources).length === 0 ||
    Object.keys(bindings.runs).length === 0
  ) {
    throw semanticIdNormalizationError(
      `Semantic fixture manifest ${revisionRole} deployment has incomplete logical or generated bindings.`,
      'missing_fixture',
    );
  }
  const expectedFlavor =
    revisionRole === 'base' ? REVISION_FLAVORS.LEGACY : REVISION_FLAVORS.NATIVE;
  if (deployment.revisionFlavor !== expectedFlavor) {
    throw semanticIdNormalizationError(
      `Semantic fixture manifest ${revisionRole} deployment must use ${expectedFlavor}, received ${deployment.revisionFlavor || '(missing)'}.`,
      'missing_fixture',
    );
  }
  if (
    deployment.validation?.valid !== true ||
    !Array.isArray(deployment.validation?.errors) ||
    deployment.validation.errors.length !== 0
  ) {
    throw semanticIdNormalizationError(
      `Semantic fixture manifest ${revisionRole} deployment has not passed fixture validation.`,
      'seed_failure',
    );
  }

  const identifiers = [];
  const identitiesByKindAndValue = new Map();
  const identitiesByToken = new Map();
  const valuesByKindAndSemanticId = new Map();
  const add = (kind, semanticId, rawValue, metadata = {}) => {
    if (rawValue === undefined || rawValue === null || rawValue === '') return;
    const value = String(rawValue);
    if (!SEMANTIC_ID_KINDS.includes(kind)) {
      throw semanticIdNormalizationError(`Unsupported semantic identifier kind ${kind}.`);
    }
    if (!SEMANTIC_ID_PATH_PATTERN.test(semanticId)) {
      throw semanticIdNormalizationError(`Invalid semantic identifier path ${semanticId}.`);
    }
    const displayLabel =
      typeof metadata.displayLabel === 'string'
        ? metadata.displayLabel.replace(/\s+/g, ' ').trim()
        : '';
    const observedDisplayLabel =
      typeof metadata.observedDisplayLabel === 'string'
        ? metadata.observedDisplayLabel.replace(/\s+/g, ' ').trim()
        : '';
    if (
      displayLabel &&
      (kind !== 'run' || displayLabel.length > 1000 || /[\u0000-\u001f]/.test(displayLabel))
    ) {
      throw semanticIdNormalizationError(
        `Semantic identifier ${semanticId} has an invalid deterministic display label.`,
      );
    }
    if (observedDisplayLabel && displayLabel && observedDisplayLabel !== displayLabel) {
      throw semanticIdNormalizationError(
        `Semantic run ${semanticId} display label ${observedDisplayLabel} does not match fixture label ${displayLabel}.`,
        'missing_fixture',
      );
    }
    const equivalenceClass = metadata.equivalenceClass || '';
    if (equivalenceClass && (kind !== 'task' || !SEMANTIC_ID_PATH_PATTERN.test(equivalenceClass))) {
      throw semanticIdNormalizationError(
        `Semantic identifier ${semanticId} has an invalid visual equivalence class.`,
      );
    }
    const tokenKind = metadata.tokenKind || kind;
    const tokenSemanticId = metadata.tokenSemanticId || equivalenceClass || semanticId;
    if (
      !SEMANTIC_ID_KINDS.includes(tokenKind) ||
      !SEMANTIC_ID_PATH_PATTERN.test(tokenSemanticId) ||
      (equivalenceClass && tokenSemanticId !== equivalenceClass)
    ) {
      throw semanticIdNormalizationError(
        `Semantic identifier ${semanticId} has an invalid visual identity.`,
      );
    }
    const token = semanticIdToken(tokenKind, tokenSemanticId);
    if (!SEMANTIC_ID_TOKEN_PATTERN.test(token)) {
      throw semanticIdNormalizationError(`Invalid semantic identifier token ${token}.`);
    }
    const tokenIdentity = identitiesByToken.get(token);
    if (
      tokenIdentity &&
      (tokenIdentity.tokenKind !== tokenKind || tokenIdentity.tokenSemanticId !== tokenSemanticId)
    ) {
      throw semanticIdNormalizationError(
        `Semantic ${kind} identifiers ${tokenIdentity.semanticId} and ${semanticId} produce the same visual token.`,
      );
    }
    const collisionKey = `${kind}\u0000${value}`;
    const existing = identitiesByKindAndValue.get(collisionKey);
    if (existing) {
      if (existing.semanticId !== semanticId || existing.token !== token) {
        throw semanticIdNormalizationError(
          `Generated ${kind} identifier is ambiguously bound to ${existing.semanticId} and ${semanticId}.`,
        );
      }
      if (displayLabel && existing.displayLabel && existing.displayLabel !== displayLabel) {
        throw semanticIdNormalizationError(
          `Semantic ${kind} identifier ${semanticId} has conflicting deterministic display labels.`,
        );
      }
      if (displayLabel && !existing.displayLabel) existing.displayLabel = displayLabel;
      return existing;
    }
    const semanticKey = `${kind}\u0000${semanticId}`;
    const existingValue = valuesByKindAndSemanticId.get(semanticKey);
    if (existingValue !== undefined && existingValue !== value) {
      throw semanticIdNormalizationError(
        `Semantic ${kind} identifier ${semanticId} is bound to multiple generated values.`,
      );
    }
    const identifier = {
      kind,
      semanticId,
      token,
      tokenKind,
      tokenSemanticId,
      value,
      ...(displayLabel ? { displayLabel } : {}),
      ...(equivalenceClass ? { equivalenceClass } : {}),
    };
    identitiesByKindAndValue.set(collisionKey, identifier);
    identitiesByToken.set(token, identifier);
    valuesByKindAndSemanticId.set(semanticKey, value);
    identifiers.push(identifier);
    return identifier;
  };

  const artifactSemanticIdsByValue = new Map();

  for (const [semanticKey, resource] of Object.entries(bindings.resources || {}).sort(
    ([left], [right]) => left.localeCompare(right),
  )) {
    if (semanticKey.startsWith('run.')) {
      add('run', semanticKey, resource?.id, {
        displayLabel: manifest.logical?.resources?.[semanticKey]?.displayName,
        observedDisplayLabel: resource?.displayName,
      });
    }
  }

  for (const [runKey, run] of Object.entries(bindings.runs || {}).sort(([left], [right]) =>
    left.localeCompare(right),
  )) {
    add('run', runKey, run?.runId, {
      displayLabel: manifest.logical?.resources?.[runKey]?.displayName,
      observedDisplayLabel: run?.displayName,
    });
    const taskArtifactReferences = [];
    const runProfile = manifest.logical?.runProfiles?.[run.fixtureProfile];

    for (const [taskKey, instances] of Object.entries(run?.taskInstances || {}).sort(
      ([left], [right]) => left.localeCompare(right),
    )) {
      for (const [index, instance] of (Array.isArray(instances) ? instances : []).entries()) {
        const taskSemanticId = taskVisualSemanticId(runKey, taskKey, index);
        const equivalenceClass = unjoinableLegacyTaskEquivalence(manifest, runKey, taskKey);
        const taskExecutions = run?.executionInstances?.[taskKey] || [];
        const matchingLegacyExecution = instance?.mlmdExecutionId
          ? taskExecutions.find((execution) => execution?.executionId === instance.mlmdExecutionId)
          : undefined;
        add('task', taskSemanticId, instance?.taskId, {
          ...(equivalenceClass ? { equivalenceClass } : {}),
          tokenKind: 'task',
          tokenSemanticId: equivalenceClass || taskSemanticId,
        });
        if (instance?.mlmdExecutionId) {
          const legacyExecutionIndex = taskExecutions.findIndex(
            (execution) => execution?.executionId === instance.mlmdExecutionId,
          );
          if (legacyExecutionIndex < 0) {
            if (run.lineageComplete === true) {
              throw semanticIdNormalizationError(
                `Task ${taskSemanticId} references legacy execution ${instance.mlmdExecutionId} outside its semantic execution group.`,
                'missing_fixture',
              );
            }
            add('execution', `${taskSemanticId}/execution`, instance.mlmdExecutionId, {
              tokenKind: 'task',
              tokenSemanticId: taskSemanticId,
            });
          } else {
            const executionScope = legacyExecutionSemanticScope(
              runKey,
              taskKey,
              taskExecutions[legacyExecutionIndex],
              legacyExecutionIndex,
            );
            add('execution', `${executionScope}/execution`, instance.mlmdExecutionId, {
              tokenKind: 'task',
              tokenSemanticId: taskSemanticId,
            });
          }
        }
        const failedMainJobs = Array.isArray(instance?.failedMainJobs)
          ? instance.failedMainJobs
          : [];
        for (const [attemptIndex, podName] of failedMainJobs.entries()) {
          const podSemanticId = `${taskSemanticId}/pod.executor[${attemptIndex}]`;
          add('pod', `${podSemanticId}/name`, podName, {
            tokenKind: 'pod',
            tokenSemanticId: `${podSemanticId}/name`,
          });
        }
        const taskPodRole = TASK_FIXTURES[taskKey]?.kind === 'runtime' ? 'executor' : 'driver';
        const legacyRetryAttemptIndex =
          run.revisionFlavor === REVISION_FLAVORS.LEGACY
            ? Math.max(
                0,
                ...(matchingLegacyExecution?.executorLogs || [])
                  .map((record) => executorLogAttemptIndex(record?.uri))
                  .filter(Number.isSafeInteger),
              )
            : 0;
        const taskPodIndex =
          taskKey === runProfile?.retry?.task
            ? Math.max(failedMainJobs.length, legacyRetryAttemptIndex)
            : 0;
        const taskPodSemanticId = `${taskSemanticId}/pod.${taskPodRole}[${taskPodIndex}]`;
        add('pod', `${taskPodSemanticId}/name`, instance?.podName, {
          tokenKind: 'pod',
          tokenSemanticId: `${taskPodSemanticId}/name`,
        });
        const podIndexesByRole = new Map();
        for (const pod of instance?.podBindings || []) {
          const role = String(pod?.type || '').toLowerCase();
          if (role !== 'driver' && role !== 'executor') {
            throw semanticIdNormalizationError(
              `Task ${taskSemanticId} contains a pod without a stable DRIVER/EXECUTOR role.`,
              'missing_fixture',
            );
          }
          const podIndex = podIndexesByRole.get(role) || 0;
          podIndexesByRole.set(role, podIndex + 1);
          const podSemanticId = `${taskSemanticId}/pod.${role}[${podIndex}]`;
          add('pod', `${podSemanticId}/name`, pod?.name, {
            tokenKind: 'pod',
            tokenSemanticId: `${podSemanticId}/name`,
          });
          add('pod', `${podSemanticId}/uid`, pod?.uid, {
            tokenKind: 'pod',
            tokenSemanticId: `${podSemanticId}/uid`,
          });
        }
        if (instance?.artifactReferences) {
          taskArtifactReferences.push({
            references: instance.artifactReferences,
            taskKey,
            taskSemanticId,
          });
        }
      }
    }

    if (run.revisionFlavor === REVISION_FLAVORS.NATIVE && runProfile?.loop) {
      const loopTask = run.taskInstances?.[runProfile.loop.task]?.[0];
      for (const iterationIndex of runProfile.loop.iterationIndexes || []) {
        const scopeSemanticId = `${runKey}/${runProfile.loop.task}/iteration[${iterationIndex}]`;
        add('task', `${scopeSemanticId}/task`, `task.${loopTask?.name}.${iterationIndex}`, {
          tokenKind: 'task',
          tokenSemanticId: scopeSemanticId,
        });
      }
    }

    for (const [scopeKey, instances] of Object.entries(run?.scopeInstances || {}).sort(
      ([left], [right]) => left.localeCompare(right),
    )) {
      for (const [index, instance] of (Array.isArray(instances) ? instances : []).entries()) {
        if (!Number.isSafeInteger(instance?.iterationIndex)) {
          throw semanticIdNormalizationError(
            `Native scope ${runKey}/${scopeKey}[${index}] is missing an iteration index.`,
            'missing_fixture',
          );
        }
        const scopeSemanticId = `${runKey}/${scopeKey}/iteration[${instance.iterationIndex}]`;
        add('task', `${scopeSemanticId}/task`, instance?.taskId, {
          tokenKind: 'task',
          tokenSemanticId: scopeSemanticId,
        });
        const podIndexesByRole = new Map();
        for (const pod of instance?.podBindings || []) {
          const role = String(pod?.type || '').toLowerCase();
          if (role !== 'driver' && role !== 'executor') {
            throw semanticIdNormalizationError(
              `Native scope ${scopeSemanticId} contains a pod without a stable role.`,
              'missing_fixture',
            );
          }
          const podIndex = podIndexesByRole.get(role) || 0;
          podIndexesByRole.set(role, podIndex + 1);
          const podSemanticId = `${scopeSemanticId}/pod.${role}[${podIndex}]`;
          add('pod', `${podSemanticId}/name`, pod?.name, {
            tokenKind: 'pod',
            tokenSemanticId: `${podSemanticId}/name`,
          });
          add('pod', `${podSemanticId}/uid`, pod?.uid, {
            tokenKind: 'pod',
            tokenSemanticId: `${podSemanticId}/uid`,
          });
        }
      }
    }

    for (const [executionKey, instances] of Object.entries(run?.executionInstances || {}).sort(
      ([left], [right]) => left.localeCompare(right),
    )) {
      const retryExecutorLogs = [];
      for (const [index, instance] of (Array.isArray(instances) ? instances : []).entries()) {
        const executionScope = legacyExecutionSemanticScope(runKey, executionKey, instance, index);
        add(
          'execution',
          `${executionScope}/execution`,
          instance?.executionId,
          legacyExecutionVisualIdentity(runKey, executionKey, instance, index),
        );
        const executionVisualIdentity = legacyExecutionVisualIdentity(
          runKey,
          executionKey,
          instance,
          index,
        );
        const executionPodRole =
          TASK_FIXTURES[executionKey]?.kind === 'runtime' ? 'executor' : 'driver';
        const executionPodIndex =
          executionKey === runProfile?.retry?.task ? runProfile.retry.attempts - 1 : 0;
        const executionPodSemanticId = `${executionVisualIdentity.tokenSemanticId}/pod.${executionPodRole}[${executionPodIndex}]`;
        add('pod', `${executionPodSemanticId}/name`, instance?.podName, {
          tokenKind: 'pod',
          tokenSemanticId: `${executionPodSemanticId}/name`,
        });
        add('pod', `${executionPodSemanticId}/uid`, instance?.podUid, {
          tokenKind: 'pod',
          tokenSemanticId: `${executionPodSemanticId}/uid`,
        });
        for (const record of instance?.executorLogs || []) {
          if (executionKey === 'task.retry-once') {
            retryExecutorLogs.push(record);
            continue;
          }
          const attemptIndex = executorLogAttemptIndex(record?.uri);
          const artifactSemanticId = `${executionScope}/artifact.executor-logs[${attemptIndex}]`;
          add('artifact', artifactSemanticId, record?.artifactId);
          add('artifact-uri', `${artifactSemanticId}/uri`, record?.uri);
        }
      }
      if (executionKey === 'task.retry-once') {
        retryExecutorLogs.sort(
          (left, right) => executorLogAttemptIndex(left?.uri) - executorLogAttemptIndex(right?.uri),
        );
        for (const record of retryExecutorLogs) {
          const attemptIndex = executorLogAttemptIndex(record?.uri);
          const artifactSemanticId = `${runKey}/${executionKey}[0]/artifact.executor-logs[${attemptIndex}]`;
          add('artifact', artifactSemanticId, record?.artifactId);
          add('artifact-uri', `${artifactSemanticId}/uri`, record?.uri);
        }
      }
    }

    for (const [artifactKey, artifact] of Object.entries(run?.artifacts || {}).sort(
      ([left], [right]) => left.localeCompare(right),
    )) {
      const groupIds = identifierValues(artifact?.artifactIds);
      const memberReferencesByValue = new Map();
      for (const [memberKey, member] of Object.entries(artifact?.members || {}).sort(
        ([left], [right]) => left.localeCompare(right),
      )) {
        for (const [index, value] of identifierValues(member?.artifactIds).entries()) {
          const references = memberReferencesByValue.get(value) || [];
          references.push(`${runKey}/${artifactKey}/${memberKey}[${index}]`);
          memberReferencesByValue.set(value, references);
        }
      }

      const emitted = new Set();
      for (const [index, value] of groupIds.entries()) {
        const memberReferences = memberReferencesByValue.get(value) || [];
        const semanticId =
          memberReferences.length === 1
            ? memberReferences[0]
            : `${runKey}/${artifactKey}[${index}]`;
        add('artifact', semanticId, value);
        artifactSemanticIdsByValue.set(value, semanticId);
        emitted.add(value);
      }
      for (const [value, memberReferences] of memberReferencesByValue) {
        if (emitted.has(value)) continue;
        const semanticId =
          memberReferences.length === 1
            ? memberReferences[0]
            : `${runKey}/${artifactKey}[${groupIds.length + emitted.size}]`;
        add('artifact', semanticId, value);
        artifactSemanticIdsByValue.set(value, semanticId);
        emitted.add(value);
      }

      for (const record of artifact?.records || artifact?.files || []) {
        const artifactId = String(record?.artifactId || '');
        const semanticId = artifactSemanticIdsByValue.get(artifactId);
        if (semanticId) add('artifact-uri', `${semanticId}/uri`, record?.uri);
      }
    }

    for (const { references, taskKey, taskSemanticId } of taskArtifactReferences) {
      let executorLogGroupCount = 0;
      for (const direction of ['outputs', 'inputs']) {
        for (const [groupIndex, group] of (references?.[direction] || []).entries()) {
          const groupKey = semanticIdentifierSegment(group?.key, 'artifact');
          const groupSemanticId = `${taskSemanticId}/${direction}.${groupKey}[${groupIndex}]`;
          const isNativeExecutorLogs =
            revisionRole === 'head' && direction === 'outputs' && group?.key === 'executor-logs';
          let artifactEntries = (group?.artifacts || []).map((record, artifactIndex) => ({
            artifactIndex,
            record,
          }));
          if (isNativeExecutorLogs) {
            executorLogGroupCount += 1;
            if (executorLogGroupCount !== 1 || TASK_FIXTURES[taskKey]?.kind !== 'runtime') {
              throw semanticIdNormalizationError(
                `Task artifact reference ${groupSemanticId} is not a valid native runtime executor-log group.`,
                'missing_fixture',
              );
            }
            const expectedCount = taskKey === 'task.retry-once' ? 2 : 1;
            if (!Array.isArray(group?.artifacts) || group.artifacts.length !== expectedCount) {
              throw semanticIdNormalizationError(
                `Task artifact reference ${groupSemanticId} must contain exactly ${expectedCount} executor-log artifact(s).`,
                'missing_fixture',
              );
            }
            artifactEntries = group.artifacts
              .map((record) => ({
                artifactIndex: executorLogAttemptIndex(record?.uri),
                record,
              }))
              .sort((left, right) => left.artifactIndex - right.artifactIndex);
            if (
              artifactEntries.some(
                (entry, index) => entry.artifactIndex === null || entry.artifactIndex !== index,
              )
            ) {
              throw semanticIdNormalizationError(
                `Task artifact reference ${groupSemanticId} must use contiguous executor-log attempt URIs starting at 0.`,
                'missing_fixture',
              );
            }
          }
          for (const { artifactIndex, record } of artifactEntries) {
            const rawArtifactId = String(record?.artifactId || '');
            if (isNativeExecutorLogs) {
              const rawUri = String(record?.uri || '');
              const recordKeys = Object.keys(record || {}).sort();
              if (
                !rawArtifactId ||
                record?.name !== 'executor-logs' ||
                record?.type !== 'Artifact' ||
                JSON.stringify(recordKeys) !==
                  JSON.stringify(['artifactId', 'name', 'type', 'uri']) ||
                executorLogAttemptIndex(rawUri) !== artifactIndex
              ) {
                throw semanticIdNormalizationError(
                  `Task artifact reference ${groupSemanticId}/artifact[${artifactIndex}] has an invalid executor-log record.`,
                  'missing_fixture',
                );
              }
              const artifactSemanticId = `${taskSemanticId}/artifact.executor-logs[${artifactIndex}]`;
              add('artifact', artifactSemanticId, rawArtifactId);
              add('artifact-uri', `${artifactSemanticId}/uri`, rawUri);
              continue;
            }
            const artifactSemanticId = artifactSemanticIdsByValue.get(rawArtifactId);
            if (!artifactSemanticId) {
              throw semanticIdNormalizationError(
                `Task artifact reference ${groupSemanticId}/artifact[${artifactIndex}] is not bound to a declared semantic artifact.`,
                'missing_fixture',
              );
            }
            add('artifact-uri', `${artifactSemanticId}/uri`, record?.uri);
          }
        }
      }
    }
  }

  return identifiers.sort((left, right) => {
    const kind = left.kind.localeCompare(right.kind);
    return kind || left.semanticId.localeCompare(right.semanticId);
  });
}

function loadSemanticIdentifierCatalog(manifestPath, revisionRole) {
  if (!manifestPath) {
    throw semanticIdNormalizationError(
      'Semantic fixture manifest is required for semantic ID normalization.',
      'missing_fixture',
    );
  }
  try {
    const manifest = loadAttestedJsonInput(manifestPath, 'Semantic fixture manifest').value;
    validateRevisionSemanticManifest(manifest, revisionRole);
    return buildSemanticIdentifierCatalog(manifest, revisionRole);
  } catch (error) {
    if (CAPTURE_VALIDITIES.has(error?.captureValidity)) throw error;
    throw semanticIdNormalizationError(
      `Failed to load semantic identifier bindings from ${manifestPath}: ${error.message}`,
    );
  }
}

function prepareSemanticIdNormalization(config, catalog) {
  const scopes = config?.scopes || [];
  if (!Array.isArray(scopes)) {
    throw semanticIdNormalizationError('Semantic ID normalization scopes must be an array.');
  }
  const supportedKinds = new Set(SEMANTIC_ID_KINDS);
  const identifiersBySemanticId = new Map(catalog.map((entry) => [entry.semanticId, entry]));

  return scopes.map((scope, scopeIndex) => {
    if (!scope || typeof scope !== 'object' || Array.isArray(scope)) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} is invalid.`,
      );
    }
    const allowedFields = new Set([
      'kinds',
      'match',
      'maxReplacementsPerIdentifier',
      'maxReplacements',
      'minReplacementsPerIdentifier',
      'minReplacements',
      'selector',
      'semanticIds',
      'semanticIdPrefixes',
    ]);
    const unknownField = Object.keys(scope).find((field) => !allowedFields.has(field));
    if (unknownField) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} has unknown field ${unknownField}.`,
      );
    }
    if (typeof scope.selector !== 'string' || !scope.selector || scope.selector.length > 1024) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} has an invalid selector.`,
      );
    }
    if (scope.match !== 'exact' && scope.match !== 'substring') {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} must use exact or substring matching.`,
      );
    }
    const hasKinds = Array.isArray(scope.kinds) && scope.kinds.length > 0;
    const hasSemanticIds = Array.isArray(scope.semanticIds) && scope.semanticIds.length > 0;
    const hasSemanticIdPrefixes =
      Array.isArray(scope.semanticIdPrefixes) && scope.semanticIdPrefixes.length > 0;
    if (Number(hasKinds) + Number(hasSemanticIds) + Number(hasSemanticIdPrefixes) !== 1) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} must select kinds, semanticIds, or semanticIdPrefixes.`,
      );
    }

    let candidates;
    if (hasKinds) {
      const kinds = [...new Set(scope.kinds)];
      if (kinds.length !== scope.kinds.length || kinds.some((kind) => !supportedKinds.has(kind))) {
        throw semanticIdNormalizationError(
          `Semantic ID normalization scope ${scopeIndex} has invalid or duplicate kinds.`,
        );
      }
      candidates = catalog.filter((entry) => kinds.includes(entry.kind));
    } else if (hasSemanticIds) {
      const semanticIds = [...new Set(scope.semanticIds)];
      if (semanticIds.length !== scope.semanticIds.length) {
        throw semanticIdNormalizationError(
          `Semantic ID normalization scope ${scopeIndex} has duplicate semanticIds.`,
        );
      }
      candidates = semanticIds.map((semanticId) => {
        const identifier = identifiersBySemanticId.get(semanticId);
        if (!identifier) {
          throw semanticIdNormalizationError(
            `Semantic ID normalization scope ${scopeIndex} is missing fixture ${semanticId}.`,
            'missing_fixture',
          );
        }
        return identifier;
      });
    } else {
      const semanticIdPrefixes = [...new Set(scope.semanticIdPrefixes)];
      if (
        semanticIdPrefixes.length !== scope.semanticIdPrefixes.length ||
        semanticIdPrefixes.some(
          (prefix) => typeof prefix !== 'string' || !SEMANTIC_ID_PATH_PATTERN.test(prefix),
        )
      ) {
        throw semanticIdNormalizationError(
          `Semantic ID normalization scope ${scopeIndex} has invalid or duplicate semanticIdPrefixes.`,
        );
      }
      candidates = catalog.filter((entry) =>
        semanticIdPrefixes.some((prefix) => entry.semanticId.startsWith(prefix)),
      );
    }
    if (candidates.length === 0 && !(hasSemanticIdPrefixes && (scope.minReplacements ?? 0) === 0)) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} selected no fixture identifiers.`,
        'missing_fixture',
      );
    }
    if (
      scope.match === 'substring' &&
      candidates.some((candidate) => candidate.value.length < 8 || /^\d+$/.test(candidate.value))
    ) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} cannot substring-match short or numeric identifiers.`,
      );
    }

    const minReplacements = scope.minReplacements ?? 0;
    const maxReplacements = scope.maxReplacements ?? null;
    if (!Number.isSafeInteger(minReplacements) || minReplacements < 0) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} has invalid minReplacements.`,
      );
    }
    if (
      maxReplacements !== null &&
      (!Number.isSafeInteger(maxReplacements) || maxReplacements < minReplacements)
    ) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} has invalid maxReplacements.`,
      );
    }
    const minReplacementsPerIdentifier = scope.minReplacementsPerIdentifier ?? 0;
    const maxReplacementsPerIdentifier = scope.maxReplacementsPerIdentifier ?? null;
    if (
      !Number.isSafeInteger(minReplacementsPerIdentifier) ||
      minReplacementsPerIdentifier < 0 ||
      (maxReplacementsPerIdentifier !== null &&
        (!Number.isSafeInteger(maxReplacementsPerIdentifier) ||
          maxReplacementsPerIdentifier < minReplacementsPerIdentifier))
    ) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} has invalid per-identifier replacement bounds.`,
      );
    }
    if (
      (hasKinds || hasSemanticIdPrefixes) &&
      (minReplacementsPerIdentifier > 0 || maxReplacementsPerIdentifier !== null)
    ) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scopeIndex} can only use per-identifier bounds with semanticIds.`,
      );
    }

    const bindingsByRawValue = new Map();
    for (const candidate of candidates) {
      const existing = bindingsByRawValue.get(candidate.value);
      if (existing && existing.token !== candidate.token) {
        throw semanticIdNormalizationError(
          `Semantic ID normalization scope ${scopeIndex} maps one generated ID to both ${existing.semanticId} and ${candidate.semanticId}.`,
        );
      }
      bindingsByRawValue.set(candidate.value, candidate);
    }

    return {
      candidates: [...bindingsByRawValue.values()].sort(
        (left, right) =>
          right.value.length - left.value.length || left.semanticId.localeCompare(right.semanticId),
      ),
      match: scope.match,
      maxReplacements,
      maxReplacementsPerIdentifier,
      minReplacements,
      minReplacementsPerIdentifier,
      selector: scope.selector,
      selectedBy: hasKinds
        ? { kinds: [...new Set(scope.kinds)].sort() }
        : hasSemanticIds
          ? { semanticIds: [...scope.semanticIds].sort() }
          : { semanticIdPrefixes: [...new Set(scope.semanticIdPrefixes)].sort() },
    };
  });
}

function prepareSemanticDerivedColorNormalization(config, catalog) {
  const scopes = config?.derivedColorScopes || [];
  if (!Array.isArray(scopes)) {
    throw semanticIdNormalizationError(
      'Semantic derived-color normalization scopes must be an array.',
    );
  }
  const keys = new Set();
  const identifiersBySemanticId = new Map(
    (catalog || []).map((entry) => [entry.semanticId, entry]),
  );
  return scopes.map((scope, scopeIndex) => {
    if (!scope || typeof scope !== 'object' || Array.isArray(scope)) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scopeIndex} is invalid.`,
      );
    }
    const allowedFields = new Set([
      'containerSelector',
      'key',
      'labelItemSelector',
      'mappingStrategy',
      'maxElements',
      'minElements',
      'selector',
      'semanticIds',
    ]);
    const unknownField = Object.keys(scope).find((field) => !allowedFields.has(field));
    if (unknownField) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scopeIndex} has unknown field ${unknownField}.`,
      );
    }
    if (
      typeof scope.key !== 'string' ||
      !/^[a-z0-9][a-z0-9-]*$/.test(scope.key) ||
      keys.has(scope.key)
    ) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scopeIndex} has an invalid or duplicate key.`,
      );
    }
    keys.add(scope.key);
    for (const selectorField of ['containerSelector', 'labelItemSelector', 'selector']) {
      if (
        typeof scope[selectorField] !== 'string' ||
        !scope[selectorField] ||
        scope[selectorField].length > 1024
      ) {
        throw semanticIdNormalizationError(
          `Semantic derived-color normalization scope ${scope.key} has an invalid ${selectorField}.`,
        );
      }
    }
    if (!['color-backed-labels', 'ordered-label-cards'].includes(scope.mappingStrategy)) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scope.key} has an invalid mappingStrategy.`,
      );
    }
    const minElements = scope.minElements ?? 1;
    const maxElements = scope.maxElements ?? null;
    if (
      !Number.isSafeInteger(minElements) ||
      minElements < 1 ||
      (maxElements !== null && (!Number.isSafeInteger(maxElements) || maxElements < minElements))
    ) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scope.key} has invalid element bounds.`,
      );
    }
    if (
      !Array.isArray(scope.semanticIds) ||
      scope.semanticIds.length === 0 ||
      new Set(scope.semanticIds).size !== scope.semanticIds.length ||
      scope.semanticIds.some(
        (semanticId) =>
          typeof semanticId !== 'string' || !SEMANTIC_ID_PATH_PATTERN.test(semanticId),
      )
    ) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scope.key} has invalid semanticIds.`,
      );
    }
    const series = [...scope.semanticIds].sort().map((semanticId) => {
      const identifier = identifiersBySemanticId.get(semanticId);
      if (!identifier || identifier.kind !== 'run') {
        throw semanticIdNormalizationError(
          `Semantic derived-color normalization scope ${scope.key} is missing run fixture ${semanticId}.`,
          'missing_fixture',
        );
      }
      if (
        typeof identifier.displayLabel !== 'string' ||
        !identifier.displayLabel ||
        identifier.displayLabel.length > 1000 ||
        /[\u0000-\u001f]/.test(identifier.displayLabel)
      ) {
        throw semanticIdNormalizationError(
          `Semantic derived-color normalization scope ${scope.key} is missing a deterministic display label for ${semanticId}.`,
          'missing_fixture',
        );
      }
      return { displayLabel: identifier.displayLabel, semanticId };
    });
    const displayLabels = series.map((entry) => entry.displayLabel);
    if (new Set(displayLabels).size !== displayLabels.length) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scope.key} has duplicate deterministic display labels.`,
        'missing_fixture',
      );
    }
    if (minElements !== series.length || maxElements !== series.length) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization scope ${scope.key} must require exactly ${series.length} element(s), one for each semanticId.`,
      );
    }
    return {
      containerSelector: scope.containerSelector,
      key: scope.key,
      labelItemSelector: scope.labelItemSelector,
      mappingStrategy: scope.mappingStrategy,
      maxElements,
      minElements,
      selector: scope.selector,
      semanticIds: series.map((entry) => entry.semanticId),
      series,
    };
  });
}

async function normalizeSemanticDerivedColors(page, config, catalog) {
  const plan = prepareSemanticDerivedColorNormalization(config, catalog);
  if (plan.length === 0) return [];
  const evaluated = await page.evaluate(
    ({ palette, scopes }) => {
      const normalizedColor = (value) =>
        String(value || '')
          .replace(/\s+/g, '')
          .toLowerCase();
      const normalizedLabel = (value) =>
        String(value || '')
          .replace(/\s+/g, ' ')
          .trim();
      return scopes.map((scope) => {
        const elements = Array.from(document.querySelectorAll(scope.selector));
        const sourceColors = elements.map((element) => {
          const computed = getComputedStyle(element);
          return computed.stroke || element.getAttribute('stroke') || '';
        });
        const sourceColorIndexes = new Map();
        sourceColors.forEach((color, index) => {
          const normalized = normalizedColor(color);
          if (normalized) sourceColorIndexes.set(normalized, index);
        });
        const mappingBySourceColor = new Map();
        const orderedMappings = [];
        let ambiguous =
          scope.mappingStrategy === 'ordered-label-cards'
            ? false
            : sourceColorIndexes.size !== sourceColors.length;
        const seenCompanions = new Set();
        const labelItems = Array.from(document.querySelectorAll(scope.labelItemSelector));
        const matchingSeriesFor = (item) => {
          const titledElement = item.querySelector?.('[title]');
          const visibleLabel = normalizedLabel(
            titledElement?.getAttribute('title') || item.textContent,
          );
          if (!visibleLabel) return [];
          return scope.series.filter((series) =>
            visibleLabel.includes(normalizedLabel(series.displayLabel)),
          );
        };
        const bindSourceColor = (sourceColor, matchingSeries, companion = null) => {
          if (!sourceColorIndexes.has(sourceColor) || matchingSeries.length !== 1) {
            ambiguous = true;
            return;
          }
          const semanticId = matchingSeries[0].semanticId;
          const existing = mappingBySourceColor.get(sourceColor);
          if (existing && existing.semanticId !== semanticId) ambiguous = true;
          if (!existing) mappingBySourceColor.set(sourceColor, { elements: [], semanticId });
          if (companion) mappingBySourceColor.get(sourceColor).elements.push(companion);
        };
        if (scope.mappingStrategy === 'ordered-label-cards') {
          if (labelItems.length !== sourceColors.length) ambiguous = true;
          labelItems.forEach((item, index) => {
            // The visible label proves identity and component order pairs the card with its curve.
            // Inline-styled descendants are presentation-only; recolor all that exist without
            // treating their incidental count as semantic evidence.
            const styledElements = [
              ...(item.matches('[style]') ? [item] : []),
              ...item.querySelectorAll('[style]'),
            ].filter((element) => {
              const color = normalizedColor(getComputedStyle(element).backgroundColor);
              return color && color !== 'rgba(0,0,0,0)' && color !== 'transparent';
            });
            const matchingSeries = matchingSeriesFor(item);
            if (matchingSeries.length !== 1) {
              ambiguous = true;
              return;
            }
            seenCompanions.add(item);
            orderedMappings.push({
              elementIndex: index,
              elements: styledElements,
              semanticId: matchingSeries[0].semanticId,
              sourceColor: normalizedColor(sourceColors[index]),
            });
          });
        } else {
          for (const item of labelItems) {
            const matchingSeries = matchingSeriesFor(item);
            const styledElements = [
              ...(item.matches('[style]') ? [item] : []),
              ...item.querySelectorAll('[style]'),
            ];
            for (const element of styledElements) {
              const sourceColor = normalizedColor(getComputedStyle(element).backgroundColor);
              if (!sourceColorIndexes.has(sourceColor)) continue;
              bindSourceColor(sourceColor, matchingSeries, element);
            }
          }
        }

        const seriesOrder = new Map(
          scope.series.map((series, index) => [series.semanticId, index]),
        );
        const mappings = (
          scope.mappingStrategy === 'ordered-label-cards'
            ? orderedMappings
            : [...mappingBySourceColor.entries()].map(([sourceColor, mapping]) => ({
                ...mapping,
                sourceColor,
              }))
        ).sort(
          (left, right) => seriesOrder.get(left.semanticId) - seriesOrder.get(right.semanticId),
        );
        if (new Set(mappings.map((mapping) => mapping.semanticId)).size !== mappings.length) {
          ambiguous = true;
        }
        const paletteBySourceColor = new Map();
        mappings.forEach((mapping) => {
          const color = palette[seriesOrder.get(mapping.semanticId) % palette.length];
          paletteBySourceColor.set(mapping.sourceColor, color);
          for (const element of mapping.elements) {
            seenCompanions.add(element);
            element.style.setProperty('background-color', color, 'important');
          }
        });
        elements.forEach((element, index) => {
          const orderedMapping = orderedMappings.find((mapping) => mapping.elementIndex === index);
          const color = orderedMapping
            ? palette[seriesOrder.get(orderedMapping.semanticId) % palette.length]
            : paletteBySourceColor.get(normalizedColor(sourceColors[index]));
          if (!color) return;
          element.setAttribute('stroke', color);
          element.style.setProperty('stroke', color, 'important');
        });
        for (const container of document.querySelectorAll(scope.containerSelector)) {
          for (const element of container.querySelectorAll('span[style]')) {
            if (!/^Series #\d+$/.test(normalizedLabel(element.parentElement?.textContent)))
              continue;
            const color = paletteBySourceColor.get(
              normalizedColor(getComputedStyle(element).backgroundColor),
            );
            if (!color) continue;
            seenCompanions.add(element);
            element.style.setProperty('background-color', color, 'important');
          }
        }
        return {
          ambiguous,
          companionCount: seenCompanions.size,
          elementCount: elements.length,
          labelCount: labelItems.length,
          mappings: mappings.map(({ semanticId, sourceColor }) => ({ semanticId, sourceColor })),
        };
      });
    },
    { palette: SEMANTIC_COLOR_PALETTE, scopes: plan },
  );

  return plan.map((scope, index) => {
    const result = evaluated[index] || {
      ambiguous: true,
      companionCount: 0,
      elementCount: 0,
      labelCount: 0,
      mappings: [],
    };
    if (
      result.elementCount < scope.minElements ||
      (scope.maxElements !== null && result.elementCount > scope.maxElements)
    ) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization ${scope.key} found ${result.elementCount} element(s); expected ${scope.minElements}${
          scope.maxElements === null ? ' or more' : `-${scope.maxElements}`
        }.`,
        'selector_drift',
      );
    }
    if (
      result.ambiguous ||
      result.mappings.length !== result.elementCount ||
      result.companionCount < result.elementCount
    ) {
      throw semanticIdNormalizationError(
        `Semantic derived-color normalization ${scope.key} could not map each curve to one visible semantic label ` +
          `(curves=${result.elementCount}, labels=${result.labelCount}, mappings=${result.mappings.length}, companions=${result.companionCount}).`,
        'selector_drift',
      );
    }
    return {
      companionCount: result.companionCount,
      containerSelector: scope.containerSelector,
      elementCount: result.elementCount,
      key: scope.key,
      labelItemSelector: scope.labelItemSelector,
      mappingStrategy: scope.mappingStrategy,
      maxElements: scope.maxElements,
      mappings: result.mappings.map((mapping) => ({
        paletteColor:
          SEMANTIC_COLOR_PALETTE[
            scope.semanticIds.indexOf(mapping.semanticId) % SEMANTIC_COLOR_PALETTE.length
          ],
        semanticId: mapping.semanticId,
        sourceColorSha256: crypto.createHash('sha256').update(mapping.sourceColor).digest('hex'),
      })),
      minElements: scope.minElements,
      selector: scope.selector,
      semanticIds: scope.semanticIds,
    };
  });
}

async function normalizeSemanticIds(page, config, catalog) {
  if (
    config !== undefined &&
    config !== null &&
    (!config ||
      typeof config !== 'object' ||
      Array.isArray(config) ||
      Object.keys(config).some((field) => field !== 'scopes' && field !== 'derivedColorScopes'))
  ) {
    throw semanticIdNormalizationError('Semantic ID normalization config is invalid.');
  }
  const plan = prepareSemanticIdNormalization(config || { scopes: [] }, catalog || []);
  const evaluated = await page.evaluate((scopes) => {
    const results = [];
    for (const scope of scopes) {
      const roots = Array.from(document.querySelectorAll(scope.selector));
      const counts = Object.fromEntries(
        scope.candidates.map((candidate) => [candidate.semanticId, 0]),
      );
      const seenTextNodes = new Set();
      for (const root of roots) {
        const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
          if (seenTextNodes.has(node)) continue;
          seenTextNodes.add(node);
          const parentName = node.parentElement?.tagName;
          if (parentName === 'SCRIPT' || parentName === 'STYLE') continue;
          let value = node.nodeValue || '';
          if (scope.match === 'exact') {
            const trimmed = value.trim();
            const candidate = scope.candidates.find((entry) => entry.value === trimmed);
            if (!candidate) continue;
            const start = value.indexOf(trimmed);
            node.nodeValue = `${value.slice(0, start)}${candidate.token}${value.slice(start + trimmed.length)}`;
            counts[candidate.semanticId] += 1;
            continue;
          }
          const originalValue = value;
          const parts = [];
          let cursor = 0;
          while (cursor < originalValue.length) {
            let nextCandidate = null;
            let nextIndex = -1;
            for (const candidate of scope.candidates) {
              const candidateIndex = originalValue.indexOf(candidate.value, cursor);
              if (candidateIndex === -1) continue;
              if (nextIndex === -1 || candidateIndex < nextIndex) {
                nextCandidate = candidate;
                nextIndex = candidateIndex;
              }
            }
            if (!nextCandidate) break;
            parts.push(originalValue.slice(cursor, nextIndex), nextCandidate.token);
            counts[nextCandidate.semanticId] += 1;
            cursor = nextIndex + nextCandidate.value.length;
          }
          if (cursor > 0) {
            parts.push(originalValue.slice(cursor));
            node.nodeValue = parts.join('');
          }
        }
      }
      results.push({ counts, rootCount: roots.length });
    }
    return results;
  }, plan);

  const scopes = plan.map((scope, index) => {
    const result = evaluated[index] || { counts: {}, rootCount: 0 };
    const replacementCount = Object.values(result.counts).reduce((sum, count) => sum + count, 0);
    if (
      replacementCount < scope.minReplacements ||
      (scope.maxReplacements !== null && replacementCount > scope.maxReplacements)
    ) {
      throw semanticIdNormalizationError(
        `Semantic ID normalization scope ${scope.selector} replaced ${replacementCount} identifier(s); expected ${scope.minReplacements}${scope.maxReplacements === null ? ' or more' : `-${scope.maxReplacements}`}.`,
        'selector_drift',
      );
    }
    for (const candidate of scope.candidates) {
      const count = result.counts[candidate.semanticId] || 0;
      if (
        count < scope.minReplacementsPerIdentifier ||
        (scope.maxReplacementsPerIdentifier !== null && count > scope.maxReplacementsPerIdentifier)
      ) {
        throw semanticIdNormalizationError(
          `Semantic ID normalization scope ${scope.selector} replaced ${candidate.semanticId} ${count} time(s); expected ${scope.minReplacementsPerIdentifier}${
            scope.maxReplacementsPerIdentifier === null
              ? ' or more'
              : `-${scope.maxReplacementsPerIdentifier}`
          }.`,
          'selector_drift',
        );
      }
    }
    const explicitlySelected = new Set(scope.selectedBy.semanticIds || []);
    const entries = scope.candidates
      .map((candidate) => ({
        ...(candidate.equivalenceClass ? { equivalenceClass: candidate.equivalenceClass } : {}),
        kind: candidate.kind,
        replacementCount: result.counts[candidate.semanticId] || 0,
        semanticId: candidate.semanticId,
        sourceIdSha256: crypto.createHash('sha256').update(candidate.value).digest('hex'),
        token: candidate.token,
        tokenKind: candidate.tokenKind,
        tokenSemanticId: candidate.tokenSemanticId,
      }))
      .filter((entry) => entry.replacementCount > 0 || explicitlySelected.has(entry.semanticId));
    return {
      ...scope.selectedBy,
      entries,
      match: scope.match,
      maxReplacements: scope.maxReplacements,
      maxReplacementsPerIdentifier: scope.maxReplacementsPerIdentifier,
      minReplacements: scope.minReplacements,
      minReplacementsPerIdentifier: scope.minReplacementsPerIdentifier,
      replacementCount,
      rootCount: result.rootCount,
      selector: scope.selector,
    };
  });

  const derivedColorScopes = await normalizeSemanticDerivedColors(
    page,
    config || {},
    catalog || [],
  );
  return {
    complete: true,
    derivedColorScopes,
    schemaVersion: SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
    scopes,
    totalReplacementCount: scopes.reduce((sum, scope) => sum + scope.replacementCount, 0),
  };
}

function resolvePathTemplate(routePath, seedValues) {
  const missing = [];
  const resolved = routePath.replace(/\{seed\.([a-zA-Z0-9_]+)\}/g, (_match, key) => {
    const value = seedValues && seedValues[key];
    if (value === undefined || value === null || value === '') {
      missing.push(key);
      return '';
    }
    return String(value);
  });

  if (missing.length > 0) {
    return { missing, resolvedPath: null };
  }

  return { missing: [], resolvedPath: resolved };
}

class SkipCaptureError extends Error {
  constructor(message, captureValidity = 'missing_fixture') {
    super(message);
    this.name = 'SkipCaptureError';
    this.captureValidity = captureValidity;
  }
}

async function executeActions(page, actions) {
  if (!Array.isArray(actions) || actions.length === 0) {
    return;
  }

  for (const action of actions) {
    const timeout = action.timeoutMs || 10000;
    const descriptor = action.selector ? `${action.type}(${action.selector})` : action.type;
    const locator = action.selector ? page.locator(action.selector) : null;
    const target =
      locator && Number.isSafeInteger(action.index) ? locator.nth(action.index) : locator?.first();

    try {
      switch (action.type) {
        case 'click':
          await target.click({ timeout });
          break;
        case 'dispatchClick':
          await page
            .locator(action.selector)
            .first()
            .evaluate((element) => element.click());
          break;
        case 'waitForSelector':
          await page.waitForSelector(action.selector, { timeout });
          break;
        case 'waitForFunction':
          if (typeof action.predicate !== 'function') {
            throw new Error('waitForFunction requires a predicate function');
          }
          await page.waitForFunction(action.predicate, undefined, { timeout });
          break;
        case 'waitForText':
          await page
            .getByText(action.text, { exact: false })
            .nth(Math.max(0, (action.minCount || 1) - 1))
            .waitFor({ timeout });
          break;
        case 'waitForFrameText':
          {
            const deadline = Date.now() + timeout;
            let foundCount = 0;
            const minCount = action.minCount || 1;
            do {
              foundCount = 0;
              for (const frame of page.frames()) {
                foundCount += await frame.getByText(action.text, { exact: false }).count();
              }
              if (foundCount < minCount) await page.waitForTimeout(100);
            } while (foundCount < minCount && Date.now() < deadline);
            if (foundCount < minCount) {
              throw new Error(
                `expected ${minCount} occurrence(s) in frames, found ${foundCount}: ${action.text}`,
              );
            }
          }
          break;
        case 'assertAbsent': {
          const count = await page.locator(action.selector).count();
          if (count !== 0) {
            throw new Error(`expected selector to be absent, found ${count} match(es)`);
          }
          break;
        }
        case 'skipIf': {
          if (typeof action.predicate !== 'function') {
            throw new Error('skipIf requires a predicate function');
          }
          const shouldSkip = await page.evaluate(action.predicate);
          if (shouldSkip) {
            throw new SkipCaptureError(
              action.reason || `Skip condition met: ${descriptor}`,
              action.captureValidity,
            );
          }
          break;
        }
        case 'scrollIntoView':
          await page.locator(action.selector).first().scrollIntoViewIfNeeded({ timeout });
          break;
        case 'hover':
          await target.hover({ timeout });
          break;
        case 'moveMouse':
          await page.mouse.move(action.x || 0, action.y || 0);
          break;
        case 'press':
          await page.keyboard.press(action.key);
          break;
        case 'waitForTimeout':
          await page.waitForTimeout(action.ms || 500);
          break;
        default:
          throw new Error(`Unsupported action type "${action.type}"`);
      }
    } catch (error) {
      if (error instanceof SkipCaptureError) {
        throw error;
      }
      if (action.optional) {
        console.log(`  Warning: optional action failed: ${descriptor}: ${error.message}`);
        continue;
      }
      const actionError = new Error(`Action failed: ${descriptor}: ${error.message}`);
      if (action.failureValidity) actionError.captureValidity = action.failureValidity;
      throw actionError;
    }
  }
}

function comparePageReadyPredicate() {
  const bodyText = document.body.innerText;
  const hasError =
    !!document.querySelector('[role="alert"]') ||
    bodyText.includes('Error: failed loading') ||
    bodyText.includes('An error is preventing');
  const isLoading =
    document.querySelectorAll('[role="circularprogress"], .MuiCircularProgress-root').length > 0;
  const hasRunResults =
    document.querySelectorAll('[data-testid="table-row"]').length >= 2 ||
    document.querySelectorAll('table tbody tr').length > 0;
  return (
    !hasError &&
    !isLoading &&
    hasRunResults &&
    bodyText.includes('Parameters') &&
    bodyText.includes('Scalar Metrics')
  );
}

function scalarMetricsReadyPredicate() {
  const pageText = document.body.innerText;
  if (
    pageText.includes('There are no Scalar Metrics artifacts available on the selected runs.') ||
    pageText.includes('An error is preventing the Scalar Metrics from being displayed.')
  ) {
    return false;
  }
  const metricLabels = new Set(
    Array.from(document.querySelectorAll('table tbody tr > td:first-child[title]')).map((cell) =>
      cell.getAttribute('title'),
    ),
  );
  return metricLabels.has('accuracy') && metricLabels.has('loss');
}

function rocCurveReadyPredicate() {
  const pageText = document.body.innerText;
  if (
    pageText.includes('There are no ROC Curve artifacts available on the selected runs.') ||
    pageText.includes('An error is preventing the ROC Curve from being displayed.')
  ) {
    return false;
  }
  const heading = Array.from(document.querySelectorAll('h3')).find((candidate) =>
    candidate.textContent?.trim().startsWith('ROC Curve:'),
  );
  return (
    !!heading &&
    !heading.textContent.includes('no artifacts') &&
    !!document.querySelector('.recharts-wrapper .recharts-line-curve')
  );
}

const PIPELINE_DETAILS_ROOT_SELECTOR =
  '[data-testid="pipeline-detail-v1"], [data-testid="pipeline-detail-v2"]';
const PIPELINE_DETAILS_GRAPH_SELECTOR =
  '[data-testid="pipeline-detail-v1"] .graphNode, ' +
  '[data-testid="pipeline-detail-v2"] [data-testid="DagCanvas"] .react-flow__node';
const PIPELINE_DETAILS_WRITE_METRICS_SELECTOR =
  '[data-testid="pipeline-detail-v1"] .graphNode:has-text("write-metrics"), ' +
  '[data-testid="pipeline-detail-v2"] [data-testid="DagCanvas"] ' +
  '.react-flow__node:has-text("write-metrics")';

function pipelineDetailsGraphReadyPredicate() {
  if (document.querySelector('[role="alert"]')) return false;
  const root = document.querySelector(
    '[data-testid="pipeline-detail-v1"], [data-testid="pipeline-detail-v2"]',
  );
  if (!root) return false;
  const nodes = Array.from(
    root.querySelectorAll('.graphNode, [data-testid="DagCanvas"] .react-flow__node'),
  );
  return (
    nodes.length > 0 &&
    nodes.every((node) => {
      const style = getComputedStyle(node);
      return style.display !== 'none' && style.visibility !== 'hidden';
    })
  );
}

// Pages to capture - these are the main UI routes (using hash-based routing)
// waitFor: selector to wait for before capturing (indicates page is loaded)
// waitForData: additional selector that indicates data has loaded (optional)
const PAGES = [
  {
    name: 'pipelines',
    path: '/#/pipelines',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes(
              'No pipelines found. Click "Upload pipeline" to start.',
            )),
      },
    ],
  },
  {
    name: 'pipeline-details-seeded',
    path: '/#/pipelines/details/{seed.pipelineId}',
    waitFor: PIPELINE_DETAILS_ROOT_SELECTOR,
    actions: [{ type: 'waitForFunction', predicate: pipelineDetailsGraphReadyPredicate }],
  },
  {
    name: 'pipeline-details-seeded-sidepanel',
    path: '/#/pipelines/details/{seed.pipelineId}',
    waitFor: PIPELINE_DETAILS_ROOT_SELECTOR,
    actions: [
      { type: 'waitForFunction', predicate: pipelineDetailsGraphReadyPredicate },
      { type: 'click', selector: PIPELINE_DETAILS_WRITE_METRICS_SELECTOR },
      { type: 'waitForSelector', selector: '[aria-label="close"]' },
    ],
  },
  {
    name: 'experiments',
    path: '/#/experiments',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes(
              'No experiments found. Click "Create experiment" to start.',
            )),
      },
    ],
  },
  {
    name: 'runs',
    path: '/#/runs',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes('No available runs found.')),
      },
    ],
  },
  {
    name: 'run-details-seeded',
    path: '/#/runs/details/{seed.runId}',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () => {
          const legacyNodes = document.querySelectorAll('.graphNode');
          if (legacyNodes.length > 0) {
            return true;
          }
          const flowNodes = Array.from(document.querySelectorAll('.react-flow__node'));
          return (
            flowNodes.length > 0 &&
            flowNodes.every((node) => getComputedStyle(node).visibility !== 'hidden')
          );
        },
      },
      { type: 'waitForTimeout', ms: 1000, optional: true },
    ],
  },
  {
    name: 'run-details-seeded-sidepanel',
    path: '/#/runs/details/{seed.runId}',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () => {
          const legacyNodes = document.querySelectorAll('.graphNode');
          if (legacyNodes.length > 0) {
            return true;
          }
          const flowNodes = Array.from(document.querySelectorAll('.react-flow__node'));
          return (
            flowNodes.length > 0 &&
            flowNodes.every((node) => getComputedStyle(node).visibility !== 'hidden')
          );
        },
      },
      {
        type: 'click',
        selector: '[role="tab"]:has-text("Graph"), button:has-text("Graph")',
        optional: true,
      },
      { type: 'click', selector: '.react-flow__node:visible, .graphNode:visible' },
      { type: 'waitForSelector', selector: '[aria-label="close"]' },
      { type: 'waitForTimeout', ms: 750, optional: true },
    ],
  },
  {
    name: 'compare-seeded',
    path: '/#/compare?runlist={seed.compareRunlist}',
    waitFor: '#root',
    actions: [
      { type: 'waitForFunction', predicate: comparePageReadyPredicate },
      { type: 'waitForFunction', predicate: scalarMetricsReadyPredicate },
    ],
  },
  {
    name: 'compare-seeded-roc',
    path: '/#/compare?runlist={seed.compareRunlist}',
    waitFor: '#root',
    actions: [
      { type: 'waitForFunction', predicate: comparePageReadyPredicate },
      {
        type: 'waitForSelector',
        selector: '[role="tab"]:has-text("ROC Curve"), button:has-text("ROC Curve")',
      },
      {
        type: 'click',
        selector: '[role="tab"]:has-text("ROC Curve"), button:has-text("ROC Curve")',
      },
      {
        type: 'waitForFunction',
        predicate: rocCurveReadyPredicate,
      },
      { type: 'scrollIntoView', selector: '.recharts-wrapper, .rv-xy-plot', optional: true },
      { type: 'moveMouse', x: 8, y: 8, optional: true },
      { type: 'waitForTimeout', ms: 250 },
      { type: 'waitForTimeout', ms: 1000, optional: true },
    ],
  },
  { name: 'runs-new', path: '/#/runs/new', waitFor: '#choosePipelineBtn' },
  {
    name: 'runs-new-pipeline-dialog',
    path: '/#/runs/new',
    waitFor: '#choosePipelineBtn',
    actions: [
      { type: 'click', selector: '#choosePipelineBtn' },
      { type: 'waitForSelector', selector: '#pipelineSelectorDialog' },
    ],
  },
  {
    name: 'runs-new-upload-dialog',
    path: '/#/runs/new',
    required: false,
    waitFor: '#choosePipelineBtn',
    actions: [
      { type: 'dispatchClick', selector: '#choosePipelineBtn' },
      { type: 'waitForSelector', selector: '#pipelineSelectorDialog' },
      {
        type: 'skipIf',
        captureValidity: 'expected_product_removal',
        predicate: () => !document.body.innerText.includes('Upload pipeline'),
        reason: 'Current new-run selector does not expose an upload option from this dialog.',
      },
      { type: 'dispatchClick', selector: 'text=Upload pipeline' },
      { type: 'waitForSelector', selector: '#dropZone' },
    ],
  },
  {
    name: 'recurring-runs',
    path: '/#/recurringruns',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes('No available recurring runs found.')),
      },
    ],
  },
  {
    name: 'artifacts',
    path: '/#/artifacts',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          document.querySelectorAll('[class*="tableRow"]').length > 0 ||
          document.body.innerText.includes('No artifacts found.'),
      },
    ],
  },
  {
    name: 'artifact-lineage-from-list',
    path: '/#/artifacts',
    required: false,
    waitFor: '[class*="tableRow"]',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !!document.querySelector('a[href*="#/artifacts/"], a[href*="/artifacts/"]') ||
          document.body.innerText.includes('No artifacts found.'),
      },
      {
        type: 'skipIf',
        predicate: () => document.body.innerText.includes('No artifacts found.'),
        reason: 'Artifact list is empty; cannot open a lineage view from the list page.',
      },
      { type: 'click', selector: 'a[href*="#/artifacts/"], a[href*="/artifacts/"]' },
      {
        type: 'waitForSelector',
        selector: '[role="tab"]:has-text("Lineage Explorer"), button:has-text("Lineage Explorer")',
      },
      {
        type: 'click',
        selector: '[role="tab"]:has-text("Lineage Explorer"), button:has-text("Lineage Explorer")',
      },
      { type: 'waitForTimeout', ms: 1000, optional: true },
    ],
  },
  {
    name: 'executions',
    path: '/#/executions',
    waitFor: '#root',
    actions: [
      {
        type: 'waitForFunction',
        predicate: () =>
          !document.querySelector('[role="alert"]') &&
          (document.querySelectorAll('[class*="tableRow"]').length > 0 ||
            document.body.innerText.includes('No executions found.')),
      },
    ],
  },
  {
    name: 'pipeline-create',
    path: '/#/pipeline_versions/new',
    waitFor: '#dropZone',
    waitForData: '#pipelinePackageUrl, [data-testid="uploadFileInput"]',
  },
  { name: 'experiment-create', path: '/#/experiments/new', waitFor: 'input' },
];

const SEMANTICALLY_REPLACED_PAGE_NAMES = new Set([
  'artifact-lineage-from-list',
  'artifacts',
  'compare-seeded',
  'compare-seeded-roc',
  'executions',
  'run-details-seeded',
  'run-details-seeded-sidepanel',
]);
const REVISION_AWARE_PAGE_ALIASES = Object.freeze({
  'artifact-lineage-from-list': 'artifact-related-tasks',
  artifacts: 'artifact-list-evolution',
  'compare-seeded': 'compare-runs',
  'compare-seeded-roc': 'compare-roc-selection',
  executions: 'executions-to-runs',
  'run-details-seeded': 'run-details-rich-graph',
  'run-details-seeded-sidepanel': 'run-details-task-panel',
});

function buildRevisionAwarePages(revisionRole, seedValues, legacyPages = PAGES) {
  const semanticPages = resolveSemanticScenarios(revisionRole, seedValues);
  const retainedPages = legacyPages
    .filter((page) => !SEMANTICALLY_REPLACED_PAGE_NAMES.has(page.name))
    .map((page) => ({
      ...page,
      expectedChange: globalExpectedChangeAnnotation(page.expectedChange || null),
    }));
  const names = new Set(semanticPages.map((page) => page.name));
  for (const page of retainedPages) {
    if (names.has(page.name)) {
      throw new Error(`Duplicate capture page name after scenario merge: ${page.name}`);
    }
    names.add(page.name);
  }
  return [...semanticPages, ...retainedPages];
}

function revisionAwarePageNames(pageNames) {
  if (!pageNames) return pageNames;
  return pageNames.map((name) => REVISION_AWARE_PAGE_ALIASES[name] || name);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function captureFilename(pageName, viewport) {
  return `${pageName}-${viewport.width}x${viewport.height}.png`;
}

function isManagedCaptureFilename(filename) {
  return /^[a-z0-9][a-z0-9-]*-[1-9]\d*x[1-9]\d*\.png$/.test(filename);
}

function validateManagedFilenames(filenames, label) {
  if (!Array.isArray(filenames) || filenames.length > 1000) {
    throw new Error(`${label} must contain an array of at most 1000 capture filenames.`);
  }
  const unique = new Set();
  for (const filename of filenames) {
    if (
      typeof filename !== 'string' ||
      path.basename(filename) !== filename ||
      !isManagedCaptureFilename(filename) ||
      unique.has(filename)
    ) {
      throw new Error(`${label} contains an invalid or duplicate filename: ${filename}`);
    }
    unique.add(filename);
  }
  return [...unique];
}

function cleanCaptureOutputs(outputDir, managedFilenames) {
  fs.mkdirSync(outputDir, { recursive: true });
  const expectedFilenames = validateManagedFilenames(managedFilenames, 'Capture output list');
  const ownerPath = path.join(outputDir, CAPTURE_OWNER_FILENAME);
  const existingEntries = fs.readdirSync(outputDir, { withFileTypes: true });
  const removed = [];

  if (existingEntries.length > 0) {
    let ownerStat;
    try {
      ownerStat = fs.lstatSync(ownerPath);
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`Refusing to clean non-empty unowned capture directory: ${outputDir}`);
      }
      throw error;
    }
    if (!ownerStat.isFile() || ownerStat.isSymbolicLink()) {
      throw new Error(`Capture ownership marker must be a regular file: ${ownerPath}`);
    }

    let owner;
    try {
      owner = JSON.parse(fs.readFileSync(ownerPath, 'utf8'));
    } catch (error) {
      throw new Error(`Capture ownership marker is invalid: ${error.message}`);
    }
    if (owner?.schemaVersion !== CAPTURE_OWNER_SCHEMA_VERSION) {
      throw new Error(`Unsupported capture ownership marker in ${ownerPath}.`);
    }
    const previousFilenames = validateManagedFilenames(owner.files, 'Capture ownership marker');
    for (const filename of [...previousFilenames, CAPTURE_MANIFEST_FILENAME]) {
      const target = path.join(outputDir, filename);
      try {
        const stat = fs.lstatSync(target);
        if (!stat.isFile() && !stat.isSymbolicLink()) {
          throw new Error(`Managed capture output is not a file: ${target}`);
        }
        fs.unlinkSync(target);
        removed.push(filename);
      } catch (error) {
        if (error.code !== 'ENOENT') throw error;
      }
    }
  }

  for (const filename of expectedFilenames) {
    if (fs.existsSync(path.join(outputDir, filename))) {
      throw new Error(`Refusing to overwrite an unmanaged capture output: ${filename}`);
    }
  }

  fs.writeFileSync(
    ownerPath,
    `${JSON.stringify(
      {
        schemaVersion: CAPTURE_OWNER_SCHEMA_VERSION,
        files: expectedFilenames,
      },
      null,
      2,
    )}\n`,
    { flag: 'w' },
  );

  return removed;
}

function assertNavigationResponse(response, url) {
  if (!response) {
    throw new Error(`Navigation to ${url} did not return an HTTP response.`);
  }

  const status = typeof response.status === 'function' ? response.status() : Number.NaN;
  const ok = typeof response.ok === 'function' ? response.ok() : status >= 200 && status < 300;
  if (!ok) {
    const statusText =
      typeof response.statusText === 'function' && response.statusText()
        ? ` ${response.statusText()}`
        : '';
    throw new Error(`Navigation to ${url} failed with HTTP ${status}${statusText}.`);
  }
}

async function installDeterministicRendering(page) {
  await page.emulateMedia({ colorScheme: 'light', reducedMotion: 'reduce' });
  await page.addInitScript(
    ({ css, fixedTimeMs, pollingDelayMs, styleId }) => {
      const NativeDate = Date;
      function FrozenDate(...args) {
        if (new.target) {
          return new NativeDate(...(args.length > 0 ? args : [fixedTimeMs]));
        }
        return new NativeDate(fixedTimeMs).toString();
      }
      FrozenDate.prototype = NativeDate.prototype;
      Object.setPrototypeOf(FrozenDate, NativeDate);
      FrozenDate.now = () => fixedTimeMs;
      FrozenDate.parse = NativeDate.parse;
      FrozenDate.UTC = NativeDate.UTC;
      globalThis.Date = FrozenDate;

      const nativeSetTimeout = globalThis.setTimeout.bind(globalThis);
      globalThis.setInterval = () => 0;
      globalThis.setTimeout = (callback, delay = 0, ...args) =>
        Number(delay) >= pollingDelayMs ? 0 : nativeSetTimeout(callback, delay, ...args);

      if (document.getElementById(styleId)) {
        return;
      }
      const install = () => {
        if (document.getElementById(styleId)) {
          return;
        }
        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = css;
        (document.head || document.documentElement).appendChild(style);
      };
      if (document.head || document.documentElement) {
        install();
      } else {
        document.addEventListener('DOMContentLoaded', install, { once: true });
      }
    },
    {
      css: DETERMINISTIC_CSS,
      fixedTimeMs: DETERMINISTIC_TIME_MS,
      pollingDelayMs: 5000,
      styleId: DETERMINISTIC_STYLE_ID,
    },
  );
}

async function ensureDeterministicRendering(page) {
  await page.evaluate(
    ({ css, styleId }) => {
      if (!document.getElementById(styleId)) {
        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = css;
        (document.head || document.documentElement).appendChild(style);
      }
    },
    { css: DETERMINISTIC_CSS, styleId: DETERMINISTIC_STYLE_ID },
  );
}

async function waitForFonts(page) {
  await page.evaluate(async () => {
    if (document.fonts && document.fonts.ready) {
      await document.fonts.ready;
    }
  });
}

async function assertDeterministicFont(page) {
  const status = await page.evaluate((fontFamily) => {
    const available = document.fonts.check(`16px "${fontFamily}"`);
    return {
      available,
      computedBodyFont: getComputedStyle(document.body).fontFamily,
      reason: available ? null : `${fontFamily} did not load from the pinned capture asset`,
    };
  }, DETERMINISTIC_FONT_FAMILY);
  if (status && !status.available) {
    const error = new Error(`Deterministic capture font check failed: ${status.reason}`);
    error.captureValidity = 'infrastructure_failure';
    throw error;
  }
  return status || null;
}

async function stabilizeChildFrames(page) {
  if (typeof page.frames !== 'function') return [];
  const mainFrame = typeof page.mainFrame === 'function' ? page.mainFrame() : null;
  const statuses = [];
  for (const frame of page.frames()) {
    if (frame === mainFrame || typeof frame.addStyleTag !== 'function') continue;
    await frame.addStyleTag({ content: DETERMINISTIC_CSS });
    await waitForFonts(frame);
    statuses.push(await assertDeterministicFont(frame));
    await normalizeDynamicText(frame);
  }
  return statuses;
}

async function normalizeDynamicText(page) {
  await page.evaluate(
    ({ fixedDate, fixedDateTime, fixedDuration }) => {
      const dateTimePatterns = [
        /\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z\b/g,
        /\b\d{1,2}\/\d{1,2}\/\d{4}, \d{1,2}:\d{2}:\d{2} [AP]M\b/g,
      ];
      const durationPattern = /(?<![\d:])-?\d+:\d{2}:\d{2}(?![\d:])/g;
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        const parentName = node.parentElement?.tagName;
        if (parentName === 'SCRIPT' || parentName === 'STYLE') continue;
        let value = node.nodeValue || '';
        value = value.replace(dateTimePatterns[0], fixedDateTime);
        value = value.replace(dateTimePatterns[1], fixedDate);
        value = value.replace(durationPattern, fixedDuration);
        node.nodeValue = value;
      }
    },
    {
      fixedDate: '1/2/2030, 3:04:05 AM',
      fixedDateTime: DETERMINISTIC_TIME_ISO,
      fixedDuration: '00:00:42',
    },
  );
}

function summarizeCaptureResults(results, fatalErrors = []) {
  const countStatus = (status) => results.filter((result) => result.status === status).length;
  const requiredResults = results.filter((result) => result.required);
  const optionalResults = results.filter((result) => !result.required);
  const requiredIncomplete = requiredResults.filter((result) => result.status !== 'success').length;
  const captured = countStatus('success') + countStatus('degraded');

  return {
    total: results.length,
    required: requiredResults.length,
    optional: optionalResults.length,
    success: countStatus('success'),
    degraded: countStatus('degraded'),
    skipped: countStatus('skipped'),
    failed: countStatus('failed'),
    requiredIncomplete,
    complete:
      results.length > 0 && captured > 0 && requiredIncomplete === 0 && fatalErrors.length === 0,
  };
}

function selectPages(pageNames, pages = PAGES) {
  if (!pageNames) {
    return { pages, unknownPageNames: [] };
  }

  const requested = new Set(pageNames);
  const selected = pages.filter((page) => requested.has(page.name));
  const known = new Set(pages.map((page) => page.name));
  const unknownPageNames = [...requested].filter((name) => !known.has(name));
  return { pages: selected, unknownPageNames };
}

function isAllowedCaptureNetworkUrl(urlValue, baseUrl) {
  let candidate;
  let base;
  try {
    candidate = new URL(urlValue);
    base = new URL(baseUrl);
  } catch (error) {
    return false;
  }
  if (candidate.protocol === 'data:' || candidate.protocol === 'blob:') return true;
  if (candidate.protocol === 'http:' || candidate.protocol === 'https:') {
    return candidate.origin === base.origin;
  }
  if (candidate.protocol === 'ws:' || candidate.protocol === 'wss:') {
    const expectedProtocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
    return candidate.protocol === expectedProtocol && candidate.host === base.host;
  }
  return false;
}

async function installNetworkIsolation(context, baseUrl) {
  await context.route('**/*', async (route) => {
    if (isAllowedCaptureNetworkUrl(route.request().url(), baseUrl)) {
      await route.continue();
    } else {
      await route.abort('blockedbyclient');
    }
  });
  if (typeof context.routeWebSocket === 'function') {
    await context.routeWebSocket(/^(?:ws|wss):\/\//, (webSocket) => {
      if (isAllowedCaptureNetworkUrl(webSocket.url(), baseUrl)) {
        webSocket.connectToServer();
      }
      // Leaving a cross-origin socket unconnected creates an inert mocked socket in the page.
    });
  }
}

function sanitizeDiagnosticText(value) {
  return String(value || '')
    .replace(/[\u0000-\u001f\u007f]+/g, ' ')
    .replace(/:\/\/([^\s/:@]+):([^\s/@]+)@/g, '://<redacted>:<redacted>@')
    .replace(/\bBearer\s+\S+/gi, 'Bearer <redacted>')
    .replace(/([?&][a-zA-Z0-9_.-]+)=([^&\s]+)/g, '$1=<redacted>')
    .replace(
      /(["']?(?:access_token|api[_-]?key|auth|authorization|cookie|credential|password|secret|set-cookie|token|x-api-key)["']?\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;}]+)/gi,
      '$1<redacted>',
    )
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, DIAGNOSTIC_TEXT_LIMIT);
}

function sanitizeDiagnosticUrl(value, baseUrl) {
  try {
    const url = new URL(value, baseUrl);
    const base = new URL(baseUrl);
    const queryKeys = [...new Set(url.searchParams.keys())].sort();
    const query =
      queryKeys.length > 0 ? `?${queryKeys.map((key) => `${key}=<redacted>`).join('&')}` : '';
    let hash = url.hash;
    if (hash.startsWith('#/')) {
      const hashUrl = new URL(hash.slice(1), base.origin);
      const hashQueryKeys = [...new Set(hashUrl.searchParams.keys())].sort();
      const hashQuery =
        hashQueryKeys.length > 0
          ? `?${hashQueryKeys.map((key) => `${key}=<redacted>`).join('&')}`
          : '';
      hash = `#${hashUrl.pathname}${hashQuery}`;
    }
    const prefix = url.origin === base.origin ? '' : url.origin;
    return sanitizeDiagnosticText(`${prefix}${url.pathname}${query}${hash}`);
  } catch (_error) {
    return '[invalid URL]';
  }
}

function createPageDiagnostics(page, baseUrl, limit = DIAGNOSTIC_LIMIT) {
  const consoleErrors = [];
  const failedRequests = [];
  const dropped = { consoleErrors: 0, failedRequests: 0 };
  const append = (collection, key, record) => {
    if (collection.length >= limit) {
      dropped[key]++;
      return;
    }
    collection.push(record);
  };
  const isSameOrigin = (urlValue) => {
    try {
      return new URL(urlValue, baseUrl).origin === new URL(baseUrl).origin;
    } catch (_error) {
      return false;
    }
  };

  if (typeof page.on === 'function') {
    page.on('console', (message) => {
      if (typeof message.type === 'function' && message.type() !== 'error') return;
      const location = typeof message.location === 'function' ? message.location() || {} : {};
      append(consoleErrors, 'consoleErrors', {
        kind: 'console',
        column: Number.isSafeInteger(location.columnNumber) ? location.columnNumber : null,
        line: Number.isSafeInteger(location.lineNumber) ? location.lineNumber : null,
        message: sanitizeDiagnosticText(
          typeof message.text === 'function' ? message.text() : String(message),
        ),
        url: location.url ? sanitizeDiagnosticUrl(location.url, baseUrl) : null,
      });
    });
    page.on('pageerror', (error) => {
      append(consoleErrors, 'consoleErrors', {
        kind: 'pageerror',
        column: null,
        line: null,
        message: sanitizeDiagnosticText(error?.message || error),
        url: null,
      });
    });
    page.on('requestfailed', (request) => {
      const requestUrl = request.url?.() || '';
      append(failedRequests, 'failedRequests', {
        error: sanitizeDiagnosticText(request.failure?.()?.errorText || 'request failed'),
        method: sanitizeDiagnosticText(request.method?.() || 'GET'),
        resourceType: sanitizeDiagnosticText(request.resourceType?.() || 'unknown'),
        sameOrigin: isSameOrigin(requestUrl),
        status: null,
        url: sanitizeDiagnosticUrl(requestUrl, baseUrl),
      });
    });
    page.on('response', (response) => {
      const status = Number(response.status?.());
      if (!Number.isFinite(status) || status < 400) return;
      const request = response.request?.();
      const responseUrl = response.url?.() || '';
      append(failedRequests, 'failedRequests', {
        error: sanitizeDiagnosticText(response.statusText?.() || `HTTP ${status}`),
        method: sanitizeDiagnosticText(request?.method?.() || 'GET'),
        resourceType: sanitizeDiagnosticText(request?.resourceType?.() || 'unknown'),
        sameOrigin: isSameOrigin(responseUrl),
        status,
        url: sanitizeDiagnosticUrl(responseUrl, baseUrl),
      });
    });
  }

  return { consoleErrors, failedRequests, dropped };
}

function routeFromUrl(value) {
  const parsed = new URL(value);
  if (parsed.hash.startsWith('#/')) return parsed.hash.slice(1);
  return `${parsed.pathname}${parsed.search}`;
}

function routeMatches(actualRoute, expectedRoute) {
  const actual = new URL(actualRoute, 'http://ui-smoke.invalid');
  const expected = new URL(expectedRoute, 'http://ui-smoke.invalid');
  if (actual.pathname !== expected.pathname) return false;
  for (const [key, value] of expected.searchParams) {
    if (actual.searchParams.get(key) !== value) return false;
  }
  return true;
}

async function assertRouteExpectation(page, routeExpectation) {
  if (!routeExpectation?.path) return null;
  try {
    await page.waitForFunction(
      (expectedRoute) => {
        const actualRoute = location.hash.startsWith('#/')
          ? location.hash.slice(1)
          : `${location.pathname}${location.search}`;
        const actual = new URL(actualRoute, location.origin);
        const expected = new URL(expectedRoute, location.origin);
        if (actual.pathname !== expected.pathname) return false;
        for (const [key, value] of expected.searchParams) {
          if (actual.searchParams.get(key) !== value) return false;
        }
        return true;
      },
      routeExpectation.path,
      { timeout: routeExpectation.timeoutMs || 10000 },
    );
  } catch (cause) {
    const error = new Error(
      `Route expectation failed: expected ${routeExpectation.path}: ${cause.message}`,
    );
    error.captureValidity = 'ui_rendering_failure';
    throw error;
  }
  const resolvedRoute = typeof page.url === 'function' ? routeFromUrl(page.url()) : null;
  if (resolvedRoute && !routeMatches(resolvedRoute, routeExpectation.path)) {
    throw new Error(
      `Route expectation failed: expected ${routeExpectation.path}, resolved ${resolvedRoute}`,
    );
  }
  return resolvedRoute;
}

function classifyCaptureFailure(error, diagnostics) {
  if (CAPTURE_VALIDITIES.has(error?.captureValidity)) return error.captureValidity;
  const message = String(error?.message || error || '');
  if (/missing fixture|missing seed key/i.test(message)) return 'missing_fixture';
  if (/seed|provenance/i.test(message)) return 'seed_failure';
  if (/Navigation .*HTTP|browser|context|page crashed/i.test(message)) {
    return 'infrastructure_failure';
  }
  if (
    diagnostics?.failedRequests?.some(
      (request) =>
        request.sameOrigin &&
        (['fetch', 'xhr', 'websocket'].includes(request.resourceType) ||
          /(?:^|\/)apis?\/|(?:^|\/)ml_metadata\//.test(request.url)) &&
        (request.status === null || request.status >= 400),
    )
  ) {
    return 'api_incompatibility';
  }
  if (/selector|Action failed|waitFor|locator|text/i.test(message)) return 'selector_drift';
  return 'ui_rendering_failure';
}

async function applyGlobalVisualNormalizations(page, semanticIdNormalizationMode, revisionRole) {
  if (semanticIdNormalizationMode !== SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK) {
    return null;
  }
  const contract = getGlobalVisualNormalizationContract(revisionRole);
  const evidence = {
    complete: false,
    rules: [],
    schemaVersion: contract.schemaVersion,
  };
  try {
    for (const rule of contract.rules) {
      const locator = page.locator(rule.selector);
      const actualMatches = await locator.count();
      const ruleEvidence = {
        actualMatches,
        applied: false,
        expectedChange: rule.expectedChange,
        expectedMatches: rule.expectedMatches,
        hiddenMatches: 0,
        key: rule.key,
        operation: rule.operation,
        selector: rule.selector,
      };
      evidence.rules.push(ruleEvidence);
      if (actualMatches !== rule.expectedMatches) {
        const error = new Error(
          `Global visual normalization selector ${rule.selector} matched ${actualMatches} element(s); expected ${rule.expectedMatches} for ${revisionRole}.`,
        );
        error.captureValidity = 'selector_drift';
        throw error;
      }
      if (rule.operation === 'hide') {
        await page.addStyleTag({
          content: `${rule.selector} { display: none !important; }`,
        });
        ruleEvidence.hiddenMatches = await locator.evaluateAll(
          (elements) =>
            elements.filter((element) => getComputedStyle(element).display === 'none').length,
        );
        if (ruleEvidence.hiddenMatches !== rule.expectedMatches) {
          const error = new Error(
            `Global visual normalization selector ${rule.selector} hid ${ruleEvidence.hiddenMatches} element(s); expected ${rule.expectedMatches} for ${revisionRole}.`,
          );
          error.captureValidity = 'selector_drift';
          throw error;
        }
        ruleEvidence.applied = true;
      } else if (rule.operation !== 'assert-absent') {
        throw new Error(`Unsupported global visual normalization operation ${rule.operation}.`);
      }
    }
    evidence.complete = true;
    return evidence;
  } catch (error) {
    error.globalVisualNormalization = evidence;
    throw error;
  }
}

async function captureScreenshots(options, dependencies = {}) {
  const semanticIdNormalizationMode = validateSemanticIdNormalizationMode(options);
  const chromium = dependencies.chromium || require('playwright').chromium;
  const revisionAware = options.revisionRole === 'base' || options.revisionRole === 'head';
  let seedLoadError = null;
  let seedManifestInput = null;
  let seedValues = null;
  try {
    seedManifestInput = loadSeedManifestInput(options.seedManifestPath, {
      required: revisionAware,
    });
    seedValues = seedManifestInput?.seedValues || null;
  } catch (error) {
    seedLoadError = error;
  }
  const pages =
    options.pages ||
    (revisionAware ? buildRevisionAwarePages(options.revisionRole, seedValues || {}) : PAGES);
  const selectedPageNames =
    options.pages || !revisionAware ? options.pageNames : revisionAwarePageNames(options.pageNames);
  const { pages: filteredPages, unknownPageNames } = selectPages(selectedPageNames, pages);
  let semanticIdentifierCatalog = [];
  let semanticIdentifierLoadError = null;
  let semanticManifestInput = null;
  const semanticIdNormalizationEnabled =
    semanticIdNormalizationMode === SEMANTIC_ID_NORMALIZATION_MODES.SEMANTIC_FULL_STACK;
  if (semanticIdNormalizationEnabled) {
    try {
      semanticManifestInput = loadAttestedJsonInput(
        options.semanticManifestPath,
        'Semantic fixture manifest',
      );
      validateRevisionSemanticManifest(semanticManifestInput.value, options.revisionRole);
      if (filteredPages.some((page) => page.semanticIdNormalization)) {
        semanticIdentifierCatalog = buildSemanticIdentifierCatalog(
          semanticManifestInput.value,
          options.revisionRole,
        );
      }
    } catch (error) {
      semanticIdentifierLoadError = error;
    }
  }
  const startedAt = new Date().toISOString();
  const captureId = crypto.randomUUID();
  const fatalErrors = [];
  const results = [];
  const completedFilenames = new Set();
  const inputs = {
    revisionRole: options.revisionRole || null,
    seedManifest: null,
    semanticManifest: null,
    sourceProvenance: null,
  };
  let browser;
  let browserVersion = null;

  if (seedLoadError) {
    fatalErrors.push(`Seed manifest is invalid: ${seedLoadError.message}`);
  }
  if (semanticIdentifierLoadError) {
    fatalErrors.push(
      `Semantic identifier bindings are invalid: ${semanticIdentifierLoadError.message}`,
    );
  }

  try {
    if (seedValues && !seedLoadError) {
      inputs.seedManifest = seedManifestInput.attestation;
    }
    inputs.semanticManifest = semanticManifestInput?.attestation || null;
    inputs.sourceProvenance = attestJsonInput(options.sourceProvenancePath, 'Source provenance');
  } catch (error) {
    fatalErrors.push(`Capture provenance is invalid: ${error.message}`);
  }

  console.log(`Starting screenshot capture for ${options.label}`);
  console.log(`Base URL: ${options.baseUrl}`);
  console.log(`Output directory: ${options.outputDir}`);
  console.log(
    `Viewports: ${options.viewports.map((viewport) => `${viewport.width}x${viewport.height}`).join(', ')}`,
  );
  console.log(
    seedValues
      ? `Seed manifest: ${options.seedManifestPath}`
      : revisionAware
        ? `Seed manifest: invalid (${seedLoadError?.message || 'required input unavailable'})`
        : 'Seed manifest: not found (seeded routes will be skipped)',
  );
  console.log(`Pages to capture: ${filteredPages.map((page) => page.name).join(', ') || '(none)'}`);

  const managedFilenames = filteredPages.flatMap((pageDefinition) =>
    options.viewports.map((viewport) => captureFilename(pageDefinition.name, viewport)),
  );
  cleanCaptureOutputs(options.outputDir, managedFilenames);

  if (unknownPageNames.length > 0) {
    fatalErrors.push(`Unknown page name(s): ${unknownPageNames.join(', ')}`);
  }
  if (filteredPages.length === 0) {
    fatalErrors.push('No pages selected for capture.');
  }

  const addResult = (result) => {
    if (!CAPTURE_STATUSES.has(result.status)) {
      throw new Error(`Invalid capture status: ${result.status}`);
    }
    if (!CAPTURE_VALIDITIES.has(result.captureValidity)) {
      throw new Error(`Invalid capture validity: ${result.captureValidity}`);
    }
    completedFilenames.add(result.filename);
    results.push(result);
  };

  try {
    if (filteredPages.length > 0 && fatalErrors.length === 0) {
      browser = await chromium.launch({
        headless: true,
      });
      browserVersion = typeof browser.version === 'function' ? browser.version() : null;
    }

    for (const viewport of options.viewports) {
      if (!browser) {
        break;
      }
      let context;
      try {
        context = await browser.newContext({
          viewport,
          colorScheme: 'light',
          deviceScaleFactor: 2,
          ignoreHTTPSErrors: true,
          locale: 'en-US',
          reducedMotion: 'reduce',
          serviceWorkers: 'block',
          timezoneId: 'UTC',
        });
        await installNetworkIsolation(context, options.baseUrl);

        for (const pageConfig of filteredPages) {
          const required = pageConfig.required !== false;
          const filename = captureFilename(pageConfig.name, viewport);
          const filepath = path.join(options.outputDir, filename);
          const { resolvedPath, missing } = resolvePathTemplate(pageConfig.path, seedValues);
          const missingFixtures = [...new Set([...(pageConfig.missingFixtures || []), ...missing])];

          if (!resolvedPath || missingFixtures.length > 0) {
            const reason = `missing fixture key(s): ${missingFixtures.join(', ')}`;
            console.log(
              `Skipping ${pageConfig.name} (${viewport.width}x${viewport.height}): ${reason}`,
            );
            addResult({
              captureValidity: 'missing_fixture',
              expectedChange: pageConfig.expectedChange || null,
              filename,
              page: pageConfig.name,
              reason,
              requestedRoute: pageConfig.path,
              required,
              revisionRole: options.revisionRole || null,
              routeExpectation: pageConfig.routeExpectation || null,
              scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
              semanticScenario: pageConfig.semanticScenario || pageConfig.name,
              status: 'skipped',
              viewport,
            });
            continue;
          }

          const url = resolveCaptureUrl(options.baseUrl, resolvedPath);
          console.log(
            `Capturing ${pageConfig.name} (${viewport.width}x${viewport.height}): ${url}`,
          );

          let page;
          let diagnostics = { consoleErrors: [], failedRequests: [], dropped: {} };
          let fontStatus = null;
          let globalVisualNormalization = null;
          let resolvedRoute = null;
          let semanticIdNormalization = null;
          try {
            // Hash-only navigation on a reused page is a same-document navigation and returns no
            // HTTP response. A fresh page guarantees that every route performs a network request
            // whose status can be validated before capture.
            page = await context.newPage();
            diagnostics = createPageDiagnostics(page, options.baseUrl);
            await installDeterministicRendering(page);
            const response = await page.goto(url, {
              waitUntil: 'networkidle',
              timeout: 30000,
            });
            assertNavigationResponse(response, url);
            await ensureDeterministicRendering(page);
            resolvedRoute = await assertRouteExpectation(page, pageConfig.routeExpectation);

            let selectorFailed = false;
            if (pageConfig.waitFor) {
              try {
                await page.waitForSelector(pageConfig.waitFor, { timeout: 10000 });
              } catch (error) {
                console.log(`  Warning: waitFor selector '${pageConfig.waitFor}' not found.`);
                selectorFailed = true;
              }
            }

            await executeActions(page, pageConfig.actions);

            if (pageConfig.waitForData) {
              try {
                await page.waitForSelector(pageConfig.waitForData, { timeout: 10000 });
                console.log(`  Data loaded: found '${pageConfig.waitForData}'`);
              } catch (error) {
                console.log(`  Warning: data selector '${pageConfig.waitForData}' not found.`);
                selectorFailed = true;
              }
            }

            globalVisualNormalization = await applyGlobalVisualNormalizations(
              page,
              semanticIdNormalizationMode,
              options.revisionRole,
            );

            await waitForFonts(page);
            fontStatus = {
              childFrames: await stabilizeChildFrames(page),
              main: await assertDeterministicFont(page),
            };
            await page.waitForTimeout(pageConfig.waitForTimeoutMs || 2000);
            await normalizeDynamicText(page);
            semanticIdNormalization = await normalizeSemanticIds(
              page,
              semanticIdNormalizationEnabled ? pageConfig.semanticIdNormalization : null,
              semanticIdentifierCatalog,
            );
            await page.screenshot({
              animations: 'disabled',
              fullPage: false,
              path: filepath,
            });

            const status = selectorFailed ? 'degraded' : 'success';
            const captureValidity = selectorFailed
              ? classifyCaptureFailure(new Error('selector readiness failed'), diagnostics)
              : pageConfig.routeExpectation?.kind === 'expected-removal'
                ? 'expected_product_removal'
                : 'valid';
            const statusIcon = selectorFailed ? '⚠' : '✓';
            const capturedAt = fs.statSync(filepath).mtime.toISOString();
            const sha256 = crypto
              .createHash('sha256')
              .update(fs.readFileSync(filepath))
              .digest('hex');
            console.log(`  ${statusIcon} Saved: ${filename}${selectorFailed ? ' (degraded)' : ''}`);
            addResult({
              captureValidity,
              capturedAt,
              diagnostics,
              expectedChange: pageConfig.expectedChange || null,
              filename,
              font: fontStatus,
              globalVisualNormalization,
              page: pageConfig.name,
              path: filepath,
              requestedRoute: resolvedPath,
              required,
              resolvedRoute:
                resolvedRoute ||
                (typeof page.url === 'function' ? routeFromUrl(page.url()) : undefined),
              revisionRole: options.revisionRole || null,
              routeExpectation: pageConfig.routeExpectation || null,
              scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
              semanticScenario: pageConfig.semanticScenario || pageConfig.name,
              semanticIdNormalization,
              sha256,
              status,
              viewport,
            });
          } catch (error) {
            globalVisualNormalization =
              error.globalVisualNormalization || globalVisualNormalization;
            if (error instanceof SkipCaptureError) {
              console.log(`  ↷ Skipped: ${error.message}`);
              addResult({
                captureValidity: error.captureValidity || 'missing_fixture',
                diagnostics,
                expectedChange: pageConfig.expectedChange || null,
                filename,
                globalVisualNormalization,
                page: pageConfig.name,
                reason: error.message,
                requestedRoute: resolvedPath,
                required,
                resolvedRoute:
                  resolvedRoute ||
                  (typeof page?.url === 'function' ? routeFromUrl(page.url()) : undefined),
                revisionRole: options.revisionRole || null,
                routeExpectation: pageConfig.routeExpectation || null,
                scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
                semanticScenario: pageConfig.semanticScenario || pageConfig.name,
                status: 'skipped',
                viewport,
              });
              continue;
            }
            console.log(`  ✗ Failed: ${error.message}`);
            addResult({
              captureValidity: classifyCaptureFailure(error, diagnostics),
              diagnostics,
              error: error.message,
              expectedChange: pageConfig.expectedChange || null,
              filename,
              globalVisualNormalization,
              page: pageConfig.name,
              requestedRoute: resolvedPath,
              required,
              resolvedRoute:
                resolvedRoute ||
                (typeof page?.url === 'function' ? routeFromUrl(page.url()) : undefined),
              revisionRole: options.revisionRole || null,
              routeExpectation: pageConfig.routeExpectation || null,
              scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
              semanticScenario: pageConfig.semanticScenario || pageConfig.name,
              status: 'failed',
              viewport,
            });
          } finally {
            if (page) {
              await page
                .close()
                .catch((error) => fatalErrors.push(`Failed to close page: ${error.message}`));
            }
          }
        }
      } catch (error) {
        fatalErrors.push(
          `Capture aborted for viewport ${viewport.width}x${viewport.height}: ${error.message}`,
        );
      } finally {
        if (context) {
          await context
            .close()
            .catch((error) =>
              fatalErrors.push(`Failed to close browser context: ${error.message}`),
            );
        }
      }
    }
  } catch (error) {
    fatalErrors.push(`Capture aborted: ${error.message}`);
  } finally {
    if (browser) {
      await browser
        .close()
        .catch((error) => fatalErrors.push(`Failed to close browser: ${error.message}`));
    }
  }

  for (const viewport of options.viewports) {
    for (const pageConfig of filteredPages) {
      const filename = captureFilename(pageConfig.name, viewport);
      if (!completedFilenames.has(filename)) {
        addResult({
          captureValidity: fatalErrors.some((error) =>
            /seed|fixture|provenance|semantic identifier/i.test(error),
          )
            ? 'seed_failure'
            : 'infrastructure_failure',
          error: fatalErrors.at(-1) || 'Capture did not complete.',
          expectedChange: pageConfig.expectedChange || null,
          filename,
          page: pageConfig.name,
          requestedRoute: pageConfig.path,
          required: pageConfig.required !== false,
          revisionRole: options.revisionRole || null,
          routeExpectation: pageConfig.routeExpectation || null,
          scenarioTitle: pageConfig.scenarioTitle || pageConfig.name,
          semanticScenario: pageConfig.semanticScenario || pageConfig.name,
          status: 'failed',
          viewport,
        });
      }
    }
  }

  const completedAt = new Date().toISOString();
  const summary = summarizeCaptureResults(results, fatalErrors);
  const manifest = {
    schemaVersion: CAPTURE_MANIFEST_SCHEMA_VERSION,
    captureId,
    label: options.label,
    startedAt,
    completedAt,
    timestamp: completedAt,
    baseUrl: options.baseUrl,
    browser: {
      engine: 'chromium',
      playwrightVersion: require('playwright/package.json').version,
      version: browserVersion,
    },
    deterministicRendering: {
      animations: 'disabled',
      colorScheme: 'light',
      fixedTime: DETERMINISTIC_TIME_ISO,
      fontFamily: DETERMINISTIC_FONT_FAMILY,
      fontPackage: DETERMINISTIC_FONT_PACKAGE,
      fontPolicy: 'embedded WOFF2 assets required; synthesis and ligatures disabled',
      fonts: DETERMINISTIC_FONT_ASSETS.map(({ filename, sha256, weight }) => ({
        filename,
        sha256,
        weight,
      })),
      locale: 'en-US',
      polling: 'timers at or above 5000ms disabled',
      reducedMotion: 'reduce',
      semanticIdNormalization: semanticIdNormalizationRenderingContract(
        semanticIdNormalizationMode,
      ),
      timezone: 'UTC',
    },
    inputs,
    scenarioContractSchemaVersion:
      pages.some((page) => page.scenarioContractSchemaVersion) && SCENARIO_CONTRACT_SCHEMA_VERSION,
    seedManifestPath: seedValues ? options.seedManifestPath : null,
    viewports: options.viewports,
    results,
    fatalErrors,
    summary,
    complete: summary.complete,
  };
  const manifestPath = path.join(options.outputDir, CAPTURE_MANIFEST_FILENAME);
  fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);

  console.log(`\nCapture complete. Results saved to ${manifestPath}`);
  console.log(
    `Required captures: ${summary.required - summary.requiredIncomplete}/${summary.required} successful`,
  );
  if (summary.degraded > 0 || summary.skipped > 0 || summary.failed > 0) {
    console.warn(
      `Incomplete captures: ${summary.degraded} degraded, ${summary.skipped} skipped, ${summary.failed} failed`,
    );
  }
  for (const error of fatalErrors) {
    console.error(`  ✗ ${error}`);
  }

  return { exitCode: summary.complete ? 0 : 1, manifest, manifestPath };
}

async function main(args = process.argv.slice(2), env = process.env) {
  try {
    const options = parseCaptureOptions(args, env);
    const result = await captureScreenshots(options);
    process.exitCode = result.exitCode;
    return result;
  } catch (error) {
    console.error('Screenshot capture failed:', error.message);
    process.exitCode = 1;
    return { exitCode: 1, error };
  }
}

module.exports = {
  CAPTURE_MANIFEST_SCHEMA_VERSION,
  CAPTURE_OWNER_FILENAME,
  CAPTURE_VALIDITIES,
  DETERMINISTIC_TIME_ISO,
  DETERMINISTIC_FONT_ASSETS,
  PIPELINE_DETAILS_GRAPH_SELECTOR,
  PIPELINE_DETAILS_ROOT_SELECTOR,
  PIPELINE_DETAILS_WRITE_METRICS_SELECTOR,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
  PAGES,
  applyGlobalVisualNormalizations,
  assertDeterministicFont,
  assertRouteExpectation,
  assertNavigationResponse,
  buildRevisionAwarePages,
  buildSemanticIdentifierCatalog,
  captureFilename,
  captureScreenshots,
  classifyCaptureFailure,
  cleanCaptureOutputs,
  comparePageReadyPredicate,
  createPageDiagnostics,
  executeActions,
  installNetworkIsolation,
  isAllowedCaptureNetworkUrl,
  loadSeedValues,
  loadSemanticIdentifierCatalog,
  normalizeDynamicText,
  normalizeSemanticDerivedColors,
  normalizeSemanticIds,
  normalizeBaseUrl,
  parseCaptureOptions,
  parseViewports,
  pipelineDetailsGraphReadyPredicate,
  prepareSemanticIdNormalization,
  prepareSemanticDerivedColorNormalization,
  rocCurveReadyPredicate,
  resolveCaptureUrl,
  resolvePathTemplate,
  revisionAwarePageNames,
  routeFromUrl,
  routeMatches,
  sanitizeDiagnosticText,
  sanitizeDiagnosticUrl,
  scalarMetricsReadyPredicate,
  selectPages,
  stabilizeChildFrames,
  summarizeCaptureResults,
};

if (require.main === module) {
  void main();
}
