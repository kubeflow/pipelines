#!/usr/bin/env node
/**
 * Pure orchestration for an in-place UI upgrade comparison.
 *
 * This module deliberately knows nothing about Kind, kubectl, migration CLIs, or
 * screenshot implementations. Every external action is supplied by the caller,
 * which keeps the lifecycle testable and prevents an unsupported migration from
 * mutating a cluster before the capability check has completed.
 */

const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const CAPTURE_MANIFEST_SCHEMA_VERSION = 2;
const COMPARISON_SUMMARY_SCHEMA_VERSION = 2;
const CAPTURE_STATUSES = new Set(['success', 'degraded', 'skipped', 'failed']);
const PNG_SIGNATURE = Buffer.from('89504e470d0a1a0a', 'hex');
const FRESHNESS_TOLERANCE_MS = 1000;

const MIGRATION_REQUIREMENT = Object.freeze({
  capability: 'mlmd-to-native-migration',
  issueNumber: 14029,
  issueUrl: 'https://github.com/kubeflow/pipelines/issues/14029',
  required: true,
});

const CONTRACT_VERSION = 1;

const UPGRADE_CAPABILITY_CONTRACT = Object.freeze({
  migration: Object.freeze({
    available: 'boolean',
    required: true,
    version: 'non-empty string',
  }),
  schemaVersion: CONTRACT_VERSION,
  startupGate: Object.freeze({
    available: 'boolean',
    migrationVersion: 'non-empty string matching migration.version',
    required: true,
  }),
  trackingIssue: MIGRATION_REQUIREMENT,
});

const CAPTURE_VALIDITY = Object.freeze({
  VALID: 'valid',
  MIGRATION_UNAVAILABLE: 'migration_unavailable',
  MIGRATION_FAILED: 'migration_failed',
  STARTUP_GATE_FAILED: 'startup_gate_failed',
  PRESERVATION_FAILED: 'preservation_failed',
  CAPTURE_FAILED: 'capture_failed',
  COMPARISON_FAILED: 'comparison_failed',
  INFRASTRUCTURE_FAILURE: 'infrastructure_failure',
});

const PHASES = Object.freeze({
  CAPABILITY_CHECK: 'capability_check',
  CONFIGURATION_CHECK: 'configuration_check',
  DEPLOY_BASE: 'deploy_base',
  SEED_BASE: 'seed_base',
  CAPTURE_BASE: 'capture_base',
  FREEZE_BASE: 'freeze_base',
  READ_BASE_STATE: 'read_base_state',
  MIGRATE: 'migrate',
  VALIDATE_MIGRATION: 'validate_migration',
  DEPLOY_HEAD: 'deploy_head',
  VALIDATE_STARTUP_GATE: 'validate_startup_gate',
  READ_HEAD_STATE: 'read_head_state',
  VERIFY_PRESERVATION: 'verify_preservation',
  PRUNE_SAFE_REMOVED_RESOURCES: 'prune_safe_removed_resources',
  CAPTURE_HEAD: 'capture_head',
  COMPARE_CAPTURES: 'compare_captures',
  COMPLETE: 'complete',
  PERSIST_RESULT: 'persist_result',
  LOAD_ADAPTER: 'load_adapter',
  CREATE_OPERATIONS: 'create_operations',
  RUNTIME_PREFLIGHT: 'runtime_preflight',
  CLEANUP_ENVIRONMENT: 'cleanup_environment',
});

// Upgrade cleanup is intentionally narrower than Kubernetes' general prune
// support. Persistent or executable one-shot resources are never accepted.
const SAFE_PRUNE_KINDS = Object.freeze([
  'ClusterRole',
  'ClusterRoleBinding',
  'ConfigMap',
  'Deployment',
  'Role',
  'RoleBinding',
  'Service',
  'ServiceAccount',
]);

const SAFE_PRUNE_KIND_SET = new Set(SAFE_PRUNE_KINDS);
const CLUSTER_SCOPED_PRUNE_KINDS = new Set(['ClusterRole', 'ClusterRoleBinding']);

const REQUIRED_OPERATIONS = Object.freeze([
  'deployBase',
  'seedBase',
  'captureBase',
  'freezeBase',
  'readBaseState',
  'migrate',
  'validateMigration',
  'deployHead',
  'validateStartupGate',
  'readHeadState',
  'verifyPreservation',
  'pruneRemovedResources',
  'captureHead',
  'compareCaptures',
]);
const REQUIRED_CLEANUP_OPERATION = 'cleanupEnvironment';

function isRecord(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function immutableSnapshot(value) {
  if (Array.isArray(value)) {
    return Object.freeze(value.map(immutableSnapshot));
  }
  if (isRecord(value)) {
    return Object.freeze(
      Object.fromEntries(
        Object.entries(value).map(([key, entry]) => [key, immutableSnapshot(entry)]),
      ),
    );
  }
  return value;
}

function requireNonEmptyString(value, description) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`${description} must be a non-empty string.`);
  }
  return value;
}

function requireSuccessfulResult(result, operation) {
  if (!isRecord(result) || result.success !== true) {
    throw new Error(`${operation} did not return { success: true }.`);
  }
  return result;
}

function serializeError(error) {
  return {
    message: error instanceof Error ? error.message : String(error),
    name: error instanceof Error ? error.name : 'Error',
  };
}

function assessUpgradeCapabilities(capabilities = {}) {
  const declared = isRecord(capabilities) ? capabilities : {};
  const migration = isRecord(declared.migration) ? declared.migration : {};
  const startupGate = isRecord(declared.startupGate) ? declared.startupGate : {};
  const missing = [];

  if (migration.available !== true) missing.push('migration');
  if (startupGate.available !== true) missing.push('startup_gate');

  const migrationVersion = typeof migration.version === 'string' ? migration.version.trim() : '';
  const gateVersion =
    typeof startupGate.migrationVersion === 'string' ? startupGate.migrationVersion.trim() : '';

  if (migration.available === true && migrationVersion === '') {
    missing.push('migration_version');
  }
  if (startupGate.available === true && gateVersion === '') {
    missing.push('startup_gate_version');
  }
  if (migrationVersion && gateVersion && migrationVersion !== gateVersion) {
    missing.push('version_mismatch');
  }

  return {
    available: missing.length === 0,
    contract: UPGRADE_CAPABILITY_CONTRACT,
    gateVersion: gateVersion || null,
    migrationVersion: migrationVersion || null,
    missing,
    requirement: MIGRATION_REQUIREMENT,
  };
}

function validateSafeRemovedResources(resources = []) {
  if (!Array.isArray(resources)) {
    throw new Error('removedResources must be an array.');
  }

  const seen = new Set();
  return Object.freeze(
    resources.map((resource, index) => {
      if (!isRecord(resource)) {
        throw new Error(`removedResources[${index}] must be an object.`);
      }
      const apiVersion = requireNonEmptyString(
        resource.apiVersion,
        `removedResources[${index}].apiVersion`,
      );
      const kind = requireNonEmptyString(resource.kind, `removedResources[${index}].kind`);
      const name = requireNonEmptyString(resource.name, `removedResources[${index}].name`);
      if (!SAFE_PRUNE_KIND_SET.has(kind)) {
        throw new Error(`Refusing to prune unsafe Kubernetes kind ${kind}.`);
      }
      if (resource.expectedRemoval !== true) {
        throw new Error(`${kind}/${name} is not explicitly marked expectedRemoval=true.`);
      }

      let namespace = null;
      if (!CLUSTER_SCOPED_PRUNE_KINDS.has(kind)) {
        namespace = requireNonEmptyString(
          resource.namespace,
          `removedResources[${index}].namespace`,
        );
      } else if (resource.namespace !== undefined && resource.namespace !== null) {
        throw new Error(`${kind}/${name} is cluster-scoped and must not specify a namespace.`);
      }

      const identity = `${apiVersion}|${kind}|${namespace || ''}|${name}`;
      if (seen.has(identity)) {
        throw new Error(`Duplicate removed resource ${kind}/${name}.`);
      }
      seen.add(identity);

      return Object.freeze({ apiVersion, expectedRemoval: true, kind, name, namespace });
    }),
  );
}

function validateRequest(request) {
  if (!isRecord(request)) throw new Error('request must be an object.');
  const artifactRoot = requireNonEmptyString(request.artifactRoot, 'request.artifactRoot');
  if (!path.isAbsolute(artifactRoot)) {
    throw new Error('request.artifactRoot must be an absolute path.');
  }
  if (!isRecord(request.sourceProvenance)) {
    throw new Error('request.sourceProvenance must be an object.');
  }
  return Object.freeze({
    ...request,
    artifactRoot,
    baseRevision: requireNonEmptyString(request.baseRevision, 'request.baseRevision'),
    headRevision: requireNonEmptyString(request.headRevision, 'request.headRevision'),
    runId: requireNonEmptyString(request.runId, 'request.runId'),
  });
}

function requestEvidence(request) {
  if (!isRecord(request)) return null;
  return immutableSnapshot({
    artifactRoot: typeof request.artifactRoot === 'string' ? request.artifactRoot : null,
    baseRevision: typeof request.baseRevision === 'string' ? request.baseRevision : null,
    headRevision: typeof request.headRevision === 'string' ? request.headRevision : null,
    runId: typeof request.runId === 'string' ? request.runId : null,
    sourceProvenance: isRecord(request.sourceProvenance) ? request.sourceProvenance : null,
  });
}

function isPathInside(parent, candidate) {
  const relative = path.relative(parent, candidate);
  return (
    relative !== '' &&
    !relative.startsWith(`..${path.sep}`) &&
    relative !== '..' &&
    !path.isAbsolute(relative)
  );
}

function readArtifact(filePath, artifactRoot, description, { json = false } = {}) {
  const declaredPath = requireNonEmptyString(filePath, description);
  if (!path.isAbsolute(declaredPath)) {
    throw new Error(`${description} must be an absolute path.`);
  }

  const rootStat = fs.lstatSync(artifactRoot);
  if (!rootStat.isDirectory() || rootStat.isSymbolicLink()) {
    throw new Error('request.artifactRoot must be a non-symlink directory.');
  }
  const realRoot = fs.realpathSync(artifactRoot);
  const resolvedPath = path.resolve(declaredPath);

  const artifactStat = fs.lstatSync(resolvedPath);
  if (!artifactStat.isFile() || artifactStat.isSymbolicLink()) {
    throw new Error(`${description} must be a non-symlink regular file.`);
  }
  const realPath = fs.realpathSync(resolvedPath);
  if (!isPathInside(realRoot, realPath)) {
    throw new Error(`${description} resolves outside request.artifactRoot.`);
  }
  if (artifactStat.size === 0) {
    throw new Error(`${description} must not be empty.`);
  }

  const contents = fs.readFileSync(realPath);
  const artifact = {
    contents,
    path: realPath,
    sha256: crypto.createHash('sha256').update(contents).digest('hex'),
    sizeBytes: artifactStat.size,
  };
  if (!json) return artifact;
  let value;
  try {
    value = JSON.parse(contents.toString('utf8'));
  } catch (error) {
    throw new Error(`${description} must contain valid JSON: ${error.message}`);
  }
  if (!isRecord(value)) {
    throw new Error(`${description} must contain a JSON object.`);
  }
  return { ...artifact, value };
}

function parseArtifactTimestamp(value, description) {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) {
    throw new Error(`${description} must be a valid timestamp.`);
  }
  return timestamp;
}

function sortedUniqueFilenames(values, description) {
  if (!Array.isArray(values)) {
    throw new Error(`${description} must be an array.`);
  }
  const filenames = values.map((filename) => requireNonEmptyString(filename, description));
  if (
    filenames.some(
      (filename) => path.basename(filename) !== filename || !filename.endsWith('.png'),
    ) ||
    new Set(filenames).size !== filenames.length
  ) {
    throw new Error(`${description} contains an invalid or duplicate PNG filename.`);
  }
  return filenames.sort();
}

function canonicalJson(value, description) {
  let jsonValue;
  try {
    const serialized = JSON.stringify(value);
    if (serialized === undefined) throw new Error('value is not JSON-serializable');
    jsonValue = JSON.parse(serialized);
  } catch (error) {
    throw new Error(`${description} must be JSON-serializable: ${error.message}`);
  }

  const sortValue = (entry) => {
    if (Array.isArray(entry)) return entry.map(sortValue);
    if (!isRecord(entry)) return entry;
    return Object.fromEntries(
      Object.keys(entry)
        .sort()
        .map((key) => [key, sortValue(entry[key])]),
    );
  };
  return JSON.stringify(sortValue(jsonValue));
}

function requireCaptureInputAttestation(attestation, expectedValue, artifactRoot, description) {
  if (!isRecord(attestation)) {
    throw new Error(`${description} attestation must be an object.`);
  }
  const artifact = readArtifact(attestation.path, artifactRoot, `${description}.path`, {
    json: true,
  });
  const schemaVersion =
    typeof artifact.value.schemaVersion === 'string' ||
    typeof artifact.value.schemaVersion === 'number'
      ? artifact.value.schemaVersion
      : null;
  if (
    attestation.sha256 !== artifact.sha256 ||
    attestation.sizeBytes !== artifact.sizeBytes ||
    attestation.schemaVersion !== schemaVersion
  ) {
    throw new Error(`${description} attestation does not match its declared JSON artifact.`);
  }
  if (
    canonicalJson(artifact.value, `${description} artifact`) !==
    canonicalJson(expectedValue, `${description} expected value`)
  ) {
    throw new Error(`${description} does not match the expected provenance input.`);
  }
  return {
    path: artifact.path,
    schemaVersion,
    sha256: artifact.sha256,
    sizeBytes: artifact.sizeBytes,
  };
}

function requireCaptureInputs(manifest, expectedInputs, artifactRoot, operation) {
  if (!isRecord(expectedInputs)) {
    throw new Error(`${operation} expected capture inputs are missing.`);
  }
  const expectedRevisionRole = requireNonEmptyString(
    expectedInputs.revisionRole,
    `${operation} expected revisionRole`,
  );
  if (!isRecord(manifest.inputs)) {
    throw new Error(`${operation} capture manifest inputs must be an object.`);
  }
  if (manifest.inputs.revisionRole !== expectedRevisionRole) {
    throw new Error(`${operation} capture manifest revisionRole must be ${expectedRevisionRole}.`);
  }
  return {
    revisionRole: expectedRevisionRole,
    semanticManifest: requireCaptureInputAttestation(
      manifest.inputs.semanticManifest,
      expectedInputs.semanticManifest,
      artifactRoot,
      `${operation} semanticManifest`,
    ),
    sourceProvenance: requireCaptureInputAttestation(
      manifest.inputs.sourceProvenance,
      expectedInputs.sourceProvenance,
      artifactRoot,
      `${operation} sourceProvenance`,
    ),
  };
}

function requireValidCaptureArtifact(result, operation, artifactRoot, expectedInputs) {
  requireSuccessfulResult(result, operation);
  if (result.captureValidity !== CAPTURE_VALIDITY.VALID) {
    throw new Error(`${operation}.captureValidity must be ${CAPTURE_VALIDITY.VALID}.`);
  }
  const {
    path: manifestPath,
    sha256: manifestSha256,
    sizeBytes: manifestSizeBytes,
    value: manifest,
  } = readArtifact(result.manifestPath, artifactRoot, `${operation}.manifestPath`, { json: true });
  if (manifest.schemaVersion !== CAPTURE_MANIFEST_SCHEMA_VERSION) {
    throw new Error(
      `${operation} capture manifest uses unsupported schema version ${manifest.schemaVersion}.`,
    );
  }
  const inputs = requireCaptureInputs(manifest, expectedInputs, artifactRoot, operation);
  const captureId = requireNonEmptyString(manifest.captureId, `${operation} captureId`);
  const startedAt = parseArtifactTimestamp(manifest.startedAt, `${operation} startedAt`);
  const completedAt = parseArtifactTimestamp(manifest.completedAt, `${operation} completedAt`);
  if (completedAt < startedAt) {
    throw new Error(`${operation} capture manifest completed before it started.`);
  }
  if (!Array.isArray(manifest.fatalErrors) || manifest.fatalErrors.length > 0) {
    throw new Error(`${operation} capture manifest reports fatal errors.`);
  }
  if (!Array.isArray(manifest.results) || manifest.results.length === 0) {
    throw new Error(`${operation} capture manifest has no captured scenarios.`);
  }
  const records = [];
  const filenames = new Set();
  for (const [index, capture] of manifest.results.entries()) {
    if (!isRecord(capture)) {
      throw new Error(`${operation} capture result ${index} is invalid.`);
    }
    const page = requireNonEmptyString(capture.page, `${operation} capture result page`);
    if (typeof capture.required !== 'boolean' || !CAPTURE_STATUSES.has(capture.status)) {
      throw new Error(`${operation} capture result ${page} has invalid metadata.`);
    }
    if (
      !isRecord(capture.viewport) ||
      !Number.isSafeInteger(capture.viewport.width) ||
      capture.viewport.width <= 0 ||
      !Number.isSafeInteger(capture.viewport.height) ||
      capture.viewport.height <= 0
    ) {
      throw new Error(`${operation} capture result ${page} has an invalid viewport.`);
    }
    const expectedFilename = `${page}-${capture.viewport.width}x${capture.viewport.height}.png`;
    if (
      capture.filename !== expectedFilename ||
      path.basename(capture.filename) !== capture.filename ||
      filenames.has(capture.filename)
    ) {
      throw new Error(`${operation} capture result ${page} has an invalid or duplicate filename.`);
    }
    filenames.add(capture.filename);
    if (
      capture.status === 'degraded' ||
      capture.status === 'failed' ||
      (capture.required && capture.status !== 'success')
    ) {
      throw new Error(`${operation} capture manifest contains incomplete or degraded scenarios.`);
    }
    if (capture.status !== 'success') {
      records.push({
        filename: capture.filename,
        required: capture.required,
        status: capture.status,
      });
      continue;
    }

    const capturedAt = parseArtifactTimestamp(
      capture.capturedAt,
      `${operation} capture result ${capture.filename} capturedAt`,
    );
    if (
      capturedAt < startedAt - FRESHNESS_TOLERANCE_MS ||
      capturedAt > completedAt + FRESHNESS_TOLERANCE_MS
    ) {
      throw new Error(`${operation} capture result ${capture.filename} is outside its window.`);
    }
    if (typeof capture.sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(capture.sha256)) {
      throw new Error(`${operation} capture result ${capture.filename} has an invalid sha256.`);
    }
    const screenshotPath = path.join(path.dirname(manifestPath), capture.filename);
    if (capture.path !== undefined) {
      let declaredScreenshotPath;
      try {
        const resolvedDeclaredPath = path.resolve(capture.path);
        declaredScreenshotPath = path.join(
          fs.realpathSync(path.dirname(resolvedDeclaredPath)),
          path.basename(resolvedDeclaredPath),
        );
      } catch (error) {
        throw new Error(
          `${operation} capture result ${capture.filename} declares an unreadable path.`,
        );
      }
      if (declaredScreenshotPath !== screenshotPath) {
        throw new Error(`${operation} capture result ${capture.filename} declares the wrong path.`);
      }
    }
    const screenshot = readArtifact(
      screenshotPath,
      artifactRoot,
      `${operation} screenshot ${capture.filename}`,
    );
    if (
      screenshot.contents.length < PNG_SIGNATURE.length ||
      !screenshot.contents.subarray(0, PNG_SIGNATURE.length).equals(PNG_SIGNATURE)
    ) {
      throw new Error(`${operation} screenshot ${capture.filename} is not a PNG file.`);
    }
    if (screenshot.sha256 !== capture.sha256) {
      throw new Error(
        `${operation} screenshot ${capture.filename} does not match its declared hash.`,
      );
    }
    if (
      capture.sizeBytes !== undefined &&
      (!Number.isSafeInteger(capture.sizeBytes) || capture.sizeBytes !== screenshot.sizeBytes)
    ) {
      throw new Error(
        `${operation} screenshot ${capture.filename} does not match its declared size.`,
      );
    }
    const screenshotStat = fs.statSync(screenshot.path);
    if (Math.abs(screenshotStat.mtimeMs - capturedAt) > FRESHNESS_TOLERANCE_MS) {
      throw new Error(
        `${operation} screenshot ${capture.filename} timestamp does not match its manifest.`,
      );
    }
    records.push({
      filename: capture.filename,
      required: capture.required,
      sha256: screenshot.sha256,
      sizeBytes: screenshot.sizeBytes,
      status: capture.status,
    });
  }

  const requiredFilenames = records
    .filter((capture) => capture.required)
    .map((capture) => capture.filename)
    .sort();
  if (
    manifest.complete !== true ||
    !isRecord(manifest.summary) ||
    manifest.summary.complete !== true ||
    manifest.summary.requiredIncomplete !== 0 ||
    requiredFilenames.length === 0
  ) {
    throw new Error(`${operation} capture manifest is incomplete.`);
  }
  return {
    ...result,
    captureId,
    inputs,
    manifestPath,
    manifestSha256,
    manifestSizeBytes,
    requiredFilenames,
    screenshotArtifacts: records.filter((capture) => capture.status === 'success'),
  };
}

function requireCaptureArtifactsUnchanged(capture, artifactRoot, role) {
  const manifest = readArtifact(capture.manifestPath, artifactRoot, `${role}Capture.manifestPath`, {
    json: true,
  });
  if (
    manifest.sha256 !== capture.manifestSha256 ||
    manifest.value.captureId !== capture.captureId
  ) {
    throw new Error(`${role} capture manifest changed after validation.`);
  }
  if (!Array.isArray(capture.screenshotArtifacts) || capture.screenshotArtifacts.length === 0) {
    throw new Error(`${role} capture has no retained screenshot artifacts.`);
  }
  if (!isRecord(capture.inputs)) {
    throw new Error(`${role} capture has no retained provenance inputs.`);
  }
  for (const inputName of ['semanticManifest', 'sourceProvenance']) {
    const input = capture.inputs[inputName];
    if (!isRecord(input)) {
      throw new Error(`${role} capture has no retained ${inputName} input.`);
    }
    const artifact = readArtifact(input.path, artifactRoot, `${role} capture ${inputName}`, {
      json: true,
    });
    if (artifact.sha256 !== input.sha256 || artifact.sizeBytes !== input.sizeBytes) {
      throw new Error(`${role} capture ${inputName} changed after validation.`);
    }
  }
  for (const screenshotArtifact of capture.screenshotArtifacts) {
    const screenshot = readArtifact(
      path.join(path.dirname(capture.manifestPath), screenshotArtifact.filename),
      artifactRoot,
      `${role} capture screenshot ${screenshotArtifact.filename}`,
    );
    if (
      screenshot.sha256 !== screenshotArtifact.sha256 ||
      screenshot.sizeBytes !== screenshotArtifact.sizeBytes
    ) {
      throw new Error(`${role} capture screenshot ${screenshotArtifact.filename} changed.`);
    }
  }
}

function requireValidComparisonArtifacts(result, artifactRoot, baseCapture, headCapture) {
  requireSuccessfulResult(result, 'compareCaptures');
  if (result.captureValidity !== CAPTURE_VALIDITY.VALID) {
    throw new Error(`compareCaptures.captureValidity must be ${CAPTURE_VALIDITY.VALID}.`);
  }
  requireCaptureArtifactsUnchanged(baseCapture, artifactRoot, 'base');
  requireCaptureArtifactsUnchanged(headCapture, artifactRoot, 'head');
  const {
    path: summaryPath,
    sha256: summarySha256,
    sizeBytes: summarySizeBytes,
    value: summary,
  } = readArtifact(result.summaryPath, artifactRoot, 'compareCaptures.summaryPath', { json: true });
  const {
    contents: reportContents,
    path: reportPath,
    sha256: reportSha256,
    sizeBytes: reportSizeBytes,
  } = readArtifact(result.reportPath, artifactRoot, 'compareCaptures.reportPath');
  if (path.extname(reportPath).toLowerCase() !== '.html') {
    throw new Error('compareCaptures.reportPath must identify an HTML report.');
  }
  if (!/<(?:!doctype\s+html|html)(?:\s|>)/i.test(reportContents.toString('utf8'))) {
    throw new Error('compareCaptures.reportPath does not contain an HTML report.');
  }
  if (
    summary.schemaVersion !== COMPARISON_SUMMARY_SCHEMA_VERSION ||
    summary.valid !== true ||
    typeof summary.passed !== 'boolean' ||
    !Array.isArray(summary.fatalErrors) ||
    summary.fatalErrors.length > 0 ||
    !Array.isArray(summary.results) ||
    summary.results.length === 0 ||
    summary.results.some((comparison) => comparison?.status !== 'success')
  ) {
    throw new Error('compareCaptures summary is not a valid visual comparison.');
  }
  if (baseCapture.captureId === headCapture.captureId) {
    throw new Error('Base and head captures must have distinct capture IDs.');
  }
  const baseRequired = sortedUniqueFilenames(
    baseCapture.requiredFilenames,
    'Base capture requiredFilenames',
  );
  const headRequired = sortedUniqueFilenames(
    headCapture.requiredFilenames,
    'Head capture requiredFilenames',
  );
  if (JSON.stringify(baseRequired) !== JSON.stringify(headRequired)) {
    throw new Error('Base and head capture required filename sets differ.');
  }
  const summaryFilenames = sortedUniqueFilenames(
    summary.results.map((comparison) => comparison?.filename),
    'Comparison summary results',
  );
  if (JSON.stringify(summaryFilenames) !== JSON.stringify(baseRequired)) {
    throw new Error('Comparison summary does not contain the exact required filename set.');
  }
  for (const [role, capture] of [
    ['base', baseCapture],
    ['head', headCapture],
  ]) {
    const attestation = summary.captures?.[role];
    if (
      !isRecord(attestation) ||
      attestation.captureId !== capture.captureId ||
      attestation.manifestSha256 !== capture.manifestSha256 ||
      JSON.stringify(
        sortedUniqueFilenames(
          attestation.requiredFilenames,
          `Comparison summary ${role} requiredFilenames`,
        ),
      ) !== JSON.stringify(baseRequired)
    ) {
      throw new Error(`Comparison summary ${role} capture attestation is invalid.`);
    }
  }
  return {
    ...result,
    comparisonPassed: summary.passed,
    reportPath,
    reportSha256,
    reportSizeBytes,
    summaryPath,
    summarySha256,
    summarySizeBytes,
  };
}

function prepareArtifactOutput(filePath, artifactRoot, description, extension) {
  const declaredPath = requireNonEmptyString(filePath, description);
  if (!path.isAbsolute(declaredPath) || path.extname(declaredPath).toLowerCase() !== extension) {
    throw new Error(`${description} must be an absolute ${extension} path.`);
  }
  const rootStat = fs.lstatSync(artifactRoot);
  if (!rootStat.isDirectory() || rootStat.isSymbolicLink()) {
    throw new Error('artifactRoot must be a non-symlink directory.');
  }
  const realRoot = fs.realpathSync(artifactRoot);
  const resolvedRoot = path.resolve(artifactRoot);
  const resolvedOutput = path.resolve(declaredPath);
  if (!isPathInside(resolvedRoot, resolvedOutput)) {
    throw new Error(`${description} must be contained by artifactRoot.`);
  }
  const parent = path.dirname(resolvedOutput);
  fs.mkdirSync(parent, { recursive: true });
  const realParent = fs.realpathSync(parent);
  if (!isPathInside(realRoot, realParent) && realParent !== realRoot) {
    throw new Error(`${description} resolves outside artifactRoot.`);
  }
  try {
    fs.lstatSync(declaredPath);
    throw new Error(`${description} already exists.`);
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
  return path.join(realParent, path.basename(declaredPath));
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function relativeArtifactUrl(reportPath, artifactPath) {
  return encodeURI(path.relative(path.dirname(reportPath), artifactPath).split(path.sep).join('/'));
}

/**
 * Convert the built-in comparison summary into the attested summary and HTML
 * report required by compareCaptures. Upgrade adapters can call this helper
 * after generate-comparison.js completes; the returned object is the complete
 * compareCaptures operation result.
 */
function writeUpgradeComparisonArtifacts({
  artifactRoot,
  baseCapture,
  headCapture,
  reportPath,
  sourceSummaryPath,
  summaryPath,
  title = 'Upgrade UI comparison',
}) {
  if (!isRecord(baseCapture) || !isRecord(headCapture)) {
    throw new Error('baseCapture and headCapture must be validated capture results.');
  }
  if (baseCapture.captureId === headCapture.captureId) {
    throw new Error('Base and head captures must have distinct capture IDs.');
  }
  const baseRequired = sortedUniqueFilenames(
    baseCapture.requiredFilenames,
    'Base capture requiredFilenames',
  );
  const headRequired = sortedUniqueFilenames(
    headCapture.requiredFilenames,
    'Head capture requiredFilenames',
  );
  if (JSON.stringify(baseRequired) !== JSON.stringify(headRequired)) {
    throw new Error('Base and head capture required filename sets differ.');
  }
  for (const [role, capture] of [
    ['base', baseCapture],
    ['head', headCapture],
  ]) {
    requireNonEmptyString(capture.captureId, `${role}Capture.captureId`);
    if (!/^[a-f0-9]{64}$/.test(capture.manifestSha256 || '')) {
      throw new Error(`${role}Capture.manifestSha256 is invalid.`);
    }
    requireCaptureArtifactsUnchanged(capture, artifactRoot, role);
  }

  const sourceSummary = readArtifact(sourceSummaryPath, artifactRoot, 'sourceSummaryPath', {
    json: true,
  });
  if (
    sourceSummary.value.schemaVersion !== COMPARISON_SUMMARY_SCHEMA_VERSION ||
    sourceSummary.value.valid !== true ||
    typeof sourceSummary.value.passed !== 'boolean' ||
    !Array.isArray(sourceSummary.value.fatalErrors) ||
    sourceSummary.value.fatalErrors.length > 0 ||
    !Array.isArray(sourceSummary.value.results)
  ) {
    throw new Error('The built-in comparison summary is invalid.');
  }
  const resultsByFilename = new Map();
  for (const comparison of sourceSummary.value.results) {
    if (
      !isRecord(comparison) ||
      typeof comparison.filename !== 'string' ||
      resultsByFilename.has(comparison.filename)
    ) {
      throw new Error('The built-in comparison summary has invalid or duplicate results.');
    }
    resultsByFilename.set(comparison.filename, comparison);
  }
  const requiredResults = baseRequired.map((filename) => {
    const comparison = resultsByFilename.get(filename);
    if (!comparison || comparison.status !== 'success') {
      throw new Error(`The built-in comparison omitted successful required result ${filename}.`);
    }
    const comparisonImage = readArtifact(
      path.join(path.dirname(sourceSummary.path), filename),
      artifactRoot,
      `comparison image ${filename}`,
    );
    if (
      comparisonImage.contents.length < PNG_SIGNATURE.length ||
      !comparisonImage.contents.subarray(0, PNG_SIGNATURE.length).equals(PNG_SIGNATURE)
    ) {
      throw new Error(`Comparison image ${filename} is not a PNG file.`);
    }
    return comparison;
  });

  const outputSummaryPath = prepareArtifactOutput(
    summaryPath,
    artifactRoot,
    'summaryPath',
    '.json',
  );
  const outputReportPath = prepareArtifactOutput(reportPath, artifactRoot, 'reportPath', '.html');
  if (outputSummaryPath === outputReportPath || outputSummaryPath === sourceSummary.path) {
    throw new Error('Attested summary, source summary, and report paths must be distinct.');
  }
  const attestedSummary = {
    ...sourceSummary.value,
    captures: {
      base: {
        captureId: baseCapture.captureId,
        manifestSha256: baseCapture.manifestSha256,
        requiredFilenames: baseRequired,
      },
      head: {
        captureId: headCapture.captureId,
        manifestSha256: headCapture.manifestSha256,
        requiredFilenames: headRequired,
      },
    },
    results: requiredResults,
    sourceSummary: {
      sha256: sourceSummary.sha256,
      sizeBytes: sourceSummary.sizeBytes,
    },
  };
  fs.writeFileSync(outputSummaryPath, `${JSON.stringify(attestedSummary, null, 2)}\n`);

  const sections = baseRequired.map((filename) => {
    const basePath = path.join(path.dirname(baseCapture.manifestPath), filename);
    const headPath = path.join(path.dirname(headCapture.manifestPath), filename);
    const comparisonPath = path.join(path.dirname(sourceSummary.path), filename);
    return `<section><h2>${escapeHtml(filename)}</h2><div class="images"><figure><figcaption>Base</figcaption><img src="${escapeHtml(relativeArtifactUrl(outputReportPath, basePath))}"></figure><figure><figcaption>Head</figcaption><img src="${escapeHtml(relativeArtifactUrl(outputReportPath, headPath))}"></figure><figure><figcaption>Comparison and highlighted diff</figcaption><img src="${escapeHtml(relativeArtifactUrl(outputReportPath, comparisonPath))}"></figure></div></section>`;
  });
  const html = `<!doctype html><html><head><meta charset="utf-8"><title>${escapeHtml(title)}</title><style>body{font:14px system-ui,sans-serif;margin:24px;background:#fff;color:#111}.images{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}img{max-width:100%;border:1px solid #ccc}figure{margin:0}figcaption{font-weight:600;margin-bottom:8px}</style></head><body><h1>${escapeHtml(title)}</h1><p>Base capture: ${escapeHtml(baseCapture.captureId)} · Head capture: ${escapeHtml(headCapture.captureId)}</p>${sections.join('')}</body></html>\n`;
  fs.writeFileSync(outputReportPath, html);
  return {
    captureValidity: CAPTURE_VALIDITY.VALID,
    reportPath: outputReportPath,
    success: true,
    summaryPath: outputSummaryPath,
  };
}

function validateOperations(operations) {
  if (!isRecord(operations)) throw new Error('operations must be an object.');
  for (const operation of REQUIRED_OPERATIONS) {
    if (typeof operations[operation] !== 'function') {
      throw new Error(`operations.${operation} must be a function.`);
    }
  }
  if (typeof operations[REQUIRED_CLEANUP_OPERATION] !== 'function') {
    throw new Error(`operations.${REQUIRED_CLEANUP_OPERATION} must be a function.`);
  }
}

function requirePersistentState(result, operation) {
  requireSuccessfulResult(result, operation);
  if (!isRecord(result.pvcIdentities) || Object.keys(result.pvcIdentities).length === 0) {
    throw new Error(`${operation} must return non-empty pvcIdentities.`);
  }
  if (!isRecord(result.semanticFixtures) || Object.keys(result.semanticFixtures).length === 0) {
    throw new Error(`${operation} must return non-empty semanticFixtures.`);
  }
  for (const [name, uid] of Object.entries(result.pvcIdentities)) {
    requireNonEmptyString(name, `${operation}.pvcIdentities key`);
    requireNonEmptyString(uid, `${operation}.pvcIdentities[${name}]`);
  }
  for (const [key, fixture] of Object.entries(result.semanticFixtures)) {
    requireNonEmptyString(key, `${operation}.semanticFixtures key`);
    if (!isRecord(fixture) || fixture.present !== true) {
      throw new Error(`${operation}.semanticFixtures[${key}] must prove present=true.`);
    }
    semanticFixtureEvidence(fixture, `${operation}.semanticFixtures[${key}]`);
  }
  return result;
}

function requireSamePvcIdentities(baseState, headState) {
  const base = baseState.pvcIdentities;
  const head = headState.pvcIdentities;
  const baseNames = Object.keys(base).sort();
  const headNames = Object.keys(head).sort();
  if (JSON.stringify(baseNames) !== JSON.stringify(headNames)) {
    throw new Error('PVC identity set changed during the in-place upgrade.');
  }
  for (const name of baseNames) {
    if (base[name] !== head[name]) {
      throw new Error(`PVC ${name} was replaced during the in-place upgrade.`);
    }
  }
}

function semanticFixtureEvidence(fixture, description) {
  const digest = typeof fixture.digest === 'string' ? fixture.digest.trim() : '';
  const evidence = typeof fixture.evidence === 'string' ? fixture.evidence.trim() : '';
  if (!digest && !evidence) {
    throw new Error(`${description} must contain a revision-independent digest or evidence.`);
  }
  return digest || evidence;
}

function requireSameSemanticFixtures(baseState, headState) {
  const baseKeys = Object.keys(baseState.semanticFixtures).sort();
  const headKeys = Object.keys(headState.semanticFixtures).sort();
  if (JSON.stringify(baseKeys) !== JSON.stringify(headKeys)) {
    throw new Error('Semantic fixture identity set changed during the in-place upgrade.');
  }
  for (const key of baseKeys) {
    const baseFixture = baseState.semanticFixtures[key];
    const headFixture = headState.semanticFixtures[key];
    if (!isRecord(baseFixture) || baseFixture.present !== true) {
      throw new Error(`Base semantic fixture ${key} is not present.`);
    }
    if (!isRecord(headFixture) || headFixture.present !== true) {
      throw new Error(`Head semantic fixture ${key} is not present.`);
    }
    const baseEvidence = semanticFixtureEvidence(baseFixture, `Base semantic fixture ${key}`);
    const headEvidence = semanticFixtureEvidence(headFixture, `Head semantic fixture ${key}`);
    if (baseEvidence !== headEvidence) {
      throw new Error(`Semantic fixture ${key} changed during the in-place upgrade.`);
    }
  }
}

function failureValidity(phase) {
  if (phase === PHASES.MIGRATE || phase === PHASES.VALIDATE_MIGRATION) {
    return CAPTURE_VALIDITY.MIGRATION_FAILED;
  }
  if (phase === PHASES.VALIDATE_STARTUP_GATE) {
    return CAPTURE_VALIDITY.STARTUP_GATE_FAILED;
  }
  if (
    phase === PHASES.READ_BASE_STATE ||
    phase === PHASES.READ_HEAD_STATE ||
    phase === PHASES.VERIFY_PRESERVATION
  ) {
    return CAPTURE_VALIDITY.PRESERVATION_FAILED;
  }
  if (phase === PHASES.CAPTURE_BASE || phase === PHASES.CAPTURE_HEAD) {
    return CAPTURE_VALIDITY.CAPTURE_FAILED;
  }
  if (phase === PHASES.COMPARE_CAPTURES) {
    return CAPTURE_VALIDITY.COMPARISON_FAILED;
  }
  return CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE;
}

function createResultWriteFailure(result, error) {
  const writeError = serializeError(error);
  const phaseHistory = Array.isArray(result.phaseHistory)
    ? [...result.phaseHistory, { phase: PHASES.PERSIST_RESULT, status: 'failed' }]
    : [{ phase: PHASES.PERSIST_RESULT, status: 'failed' }];
  return {
    ...result,
    captureValidity: CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE,
    complete: false,
    error: {
      category: CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE,
      message: `Failed to persist upgrade result: ${writeError.message}`,
      name: writeError.name,
    },
    phase: PHASES.PERSIST_RESULT,
    phaseHistory,
    resultWriteError: writeError,
  };
}

async function persistResult(operations, result) {
  if (typeof operations?.writeResult !== 'function') {
    return createResultWriteFailure(
      result,
      new Error('operations.writeResult must be a function.'),
    );
  }
  try {
    await operations.writeResult(result);
    return result;
  } catch (error) {
    if (
      isRecord(error?.persistedResult) &&
      error.persistedResult.complete === false &&
      isRecord(error.persistedResult.resultWriteError)
    ) {
      return error.persistedResult;
    }
    return createResultWriteFailure(result, error);
  }
}

/**
 * Run a fail-closed, in-place upgrade comparison.
 *
 * Capability inspection and configuration validation happen before deployBase,
 * so an unavailable migration or unsafe prune inventory cannot mutate a cluster.
 */
async function orchestrateUpgrade({
  capabilities = {},
  operations = {},
  removedResources = [],
  request,
} = {}) {
  const phaseHistory = [];
  let currentPhase = PHASES.CAPABILITY_CHECK;
  let normalizedRequest = null;
  const suppliedRequestEvidence = requestEvidence(request);
  const capabilityAssessment = immutableSnapshot(assessUpgradeCapabilities(capabilities));

  if (!capabilityAssessment.available) {
    phaseHistory.push({ phase: PHASES.CAPABILITY_CHECK, status: 'failed' });
    return persistResult(operations, {
      baseCaptured: false,
      captureValidity: CAPTURE_VALIDITY.MIGRATION_UNAVAILABLE,
      complete: false,
      contractVersion: CONTRACT_VERSION,
      headCaptured: false,
      migration: capabilityAssessment,
      mode: 'upgrade-in-place',
      phase: PHASES.CAPABILITY_CHECK,
      phaseHistory,
      request: suppliedRequestEvidence,
    });
  }

  const state = {
    baseCapture: null,
    baseDeployment: null,
    baseState: null,
    freeze: null,
    headCapture: null,
    comparison: null,
    headDeployment: null,
    headState: null,
    migration: null,
    migrationValidation: null,
    preservation: null,
    seed: null,
    startupGate: null,
  };

  const runPhase = async (phase, operation, extra = {}) => {
    currentPhase = phase;
    phaseHistory.push({ phase, status: 'started' });
    const result = await operations[operation](
      immutableSnapshot({
        ...extra,
        capabilities: capabilityAssessment,
        request: normalizedRequest,
        state,
      }),
    );
    const successfulResult = requireSuccessfulResult(result, operation);
    phaseHistory[phaseHistory.length - 1] = { phase, status: 'completed' };
    return immutableSnapshot(successfulResult);
  };

  try {
    currentPhase = PHASES.CONFIGURATION_CHECK;
    phaseHistory.push({ phase: currentPhase, status: 'started' });
    normalizedRequest = immutableSnapshot(validateRequest(request));
    const safeRemovedResources = validateSafeRemovedResources(removedResources);
    validateOperations(operations);
    phaseHistory[phaseHistory.length - 1] = { phase: currentPhase, status: 'completed' };

    state.baseDeployment = await runPhase(PHASES.DEPLOY_BASE, 'deployBase');
    const environmentId = requireNonEmptyString(
      state.baseDeployment.environmentId,
      'deployBase.environmentId',
    );

    state.seed = await runPhase(PHASES.SEED_BASE, 'seedBase', {
      environmentId,
    });
    if (!isRecord(state.seed.semanticManifest)) {
      throw new Error('seedBase must return a semanticManifest object.');
    }

    state.baseCapture = requireValidCaptureArtifact(
      await runPhase(PHASES.CAPTURE_BASE, 'captureBase', {
        environmentId,
        semanticManifest: state.seed.semanticManifest,
      }),
      'captureBase',
      normalizedRequest.artifactRoot,
      {
        revisionRole: 'base',
        semanticManifest: state.seed.semanticManifest,
        sourceProvenance: normalizedRequest.sourceProvenance,
      },
    );

    state.freeze = await runPhase(PHASES.FREEZE_BASE, 'freezeBase', {
      environmentId,
    });

    state.baseState = requirePersistentState(
      await runPhase(PHASES.READ_BASE_STATE, 'readBaseState', {
        environmentId,
        semanticManifest: state.seed.semanticManifest,
      }),
      'readBaseState',
    );

    state.migration = await runPhase(PHASES.MIGRATE, 'migrate', {
      baseState: state.baseState,
      environmentId,
      semanticManifest: state.seed.semanticManifest,
    });
    if (state.migration.migrationVersion !== capabilityAssessment.migrationVersion) {
      throw new Error(
        'migrate returned a migration version different from the declared capability.',
      );
    }

    state.migrationValidation = await runPhase(PHASES.VALIDATE_MIGRATION, 'validateMigration', {
      environmentId,
      migration: state.migration,
    });
    const marker = state.migrationValidation.durableMarker;
    if (
      !isRecord(marker) ||
      marker.durable !== true ||
      marker.validated !== true ||
      marker.status !== 'complete' ||
      marker.version !== capabilityAssessment.migrationVersion
    ) {
      throw new Error('Migration validation did not prove a durable, validated completion marker.');
    }

    state.headDeployment = await runPhase(PHASES.DEPLOY_HEAD, 'deployHead', {
      environmentId,
      migrationValidation: state.migrationValidation,
    });
    if (state.headDeployment.environmentId !== environmentId) {
      throw new Error('deployHead did not update the same environment used by deployBase.');
    }

    state.startupGate = await runPhase(PHASES.VALIDATE_STARTUP_GATE, 'validateStartupGate', {
      environmentId,
      migrationValidation: state.migrationValidation,
    });
    if (
      state.startupGate.enforced !== true ||
      state.startupGate.accepted !== true ||
      state.startupGate.durableMarkerObserved !== true ||
      state.startupGate.migrationVersion !== capabilityAssessment.gateVersion
    ) {
      throw new Error('Startup-gate validation did not prove enforcement of the migrated state.');
    }

    state.headState = requirePersistentState(
      await runPhase(PHASES.READ_HEAD_STATE, 'readHeadState', {
        environmentId,
        semanticManifest: state.seed.semanticManifest,
      }),
      'readHeadState',
    );
    requireSamePvcIdentities(state.baseState, state.headState);
    requireSameSemanticFixtures(state.baseState, state.headState);

    state.preservation = await runPhase(PHASES.VERIFY_PRESERVATION, 'verifyPreservation', {
      baseState: state.baseState,
      environmentId,
      headState: state.headState,
      semanticManifest: state.seed.semanticManifest,
    });
    if (
      state.preservation.preserved !== true ||
      state.preservation.pvcIdentitiesPreserved !== true ||
      state.preservation.semanticFixturesPreserved !== true
    ) {
      throw new Error(
        'Preservation verification did not prove PVC and semantic fixture continuity.',
      );
    }

    await runPhase(PHASES.PRUNE_SAFE_REMOVED_RESOURCES, 'pruneRemovedResources', {
      environmentId,
      resources: safeRemovedResources,
    });

    state.headCapture = requireValidCaptureArtifact(
      await runPhase(PHASES.CAPTURE_HEAD, 'captureHead', {
        environmentId,
        preservation: state.preservation,
        semanticManifest: state.seed.semanticManifest,
      }),
      'captureHead',
      normalizedRequest.artifactRoot,
      {
        revisionRole: 'head',
        semanticManifest: state.seed.semanticManifest,
        sourceProvenance: normalizedRequest.sourceProvenance,
      },
    );
    if (state.baseCapture.manifestPath === state.headCapture.manifestPath) {
      throw new Error('Base and head captures must use distinct manifests.');
    }

    state.comparison = requireValidComparisonArtifacts(
      await runPhase(PHASES.COMPARE_CAPTURES, 'compareCaptures', {
        baseCapture: state.baseCapture,
        environmentId,
        headCapture: state.headCapture,
        semanticManifest: state.seed.semanticManifest,
      }),
      normalizedRequest.artifactRoot,
      state.baseCapture,
      state.headCapture,
    );

    currentPhase = PHASES.COMPLETE;
    phaseHistory.push({ phase: currentPhase, status: 'completed' });
    return persistResult(operations, {
      baseCapture: state.baseCapture,
      baseCaptured: true,
      captureValidity: CAPTURE_VALIDITY.VALID,
      comparison: state.comparison,
      comparisonPassed: state.comparison.comparisonPassed,
      complete: true,
      contractVersion: CONTRACT_VERSION,
      environmentId,
      headCapture: state.headCapture,
      headCaptured: true,
      migration: {
        requirement: MIGRATION_REQUIREMENT,
        version: capabilityAssessment.migrationVersion,
      },
      mode: 'upgrade-in-place',
      phase: PHASES.COMPLETE,
      phaseHistory,
      preservation: state.preservation,
      request: normalizedRequest,
    });
  } catch (error) {
    const last = phaseHistory[phaseHistory.length - 1];
    if (last?.phase === currentPhase && last.status !== 'failed') {
      phaseHistory[phaseHistory.length - 1] = { phase: currentPhase, status: 'failed' };
    }
    return persistResult(operations, {
      baseCaptured: state.baseCapture !== null,
      captureValidity: failureValidity(currentPhase),
      complete: false,
      contractVersion: CONTRACT_VERSION,
      error: { category: failureValidity(currentPhase), ...serializeError(error) },
      headCaptured: state.headCapture !== null,
      migration: {
        requirement: MIGRATION_REQUIREMENT,
        version: capabilityAssessment.migrationVersion,
      },
      mode: 'upgrade-in-place',
      phase: currentPhase,
      phaseHistory,
      request: normalizedRequest || suppliedRequestEvidence,
    });
  }
}

module.exports = {
  CAPTURE_VALIDITY,
  CONTRACT_VERSION,
  MIGRATION_REQUIREMENT,
  PHASES,
  REQUIRED_OPERATIONS,
  REQUIRED_CLEANUP_OPERATION,
  SAFE_PRUNE_KINDS,
  UPGRADE_CAPABILITY_CONTRACT,
  assessUpgradeCapabilities,
  createResultWriteFailure,
  orchestrateUpgrade,
  validateOperations,
  validateRequest,
  validateSafeRemovedResources,
  writeUpgradeComparisonArtifacts,
};
