const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const { buildSemanticIdentifierCatalog } = require('../capture-screenshots');
const { summarizeComparison } = require('../generate-comparison');
const {
  getGlobalVisualNormalizationContract,
  getSemanticIdNormalizationContract,
} = require('../semantic-capture-scenarios');
const {
  SEMANTIC_COLOR_PALETTE,
  SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
} = require('../semantic-id-normalization');
const { strictSemanticFixtureManifest } = require('./semantic-fixture');

const {
  CAPTURE_VALIDITY,
  CONTRACT_VERSION,
  MIGRATION_REQUIREMENT,
  PHASES,
  REQUIRED_OPERATIONS,
  SAFE_PRUNE_KINDS,
  UPGRADE_CAPABILITY_CONTRACT,
  assessUpgradeCapabilities,
  orchestrateUpgrade,
  validateSafeRemovedResources,
  writeUpgradeComparisonArtifacts,
} = require('../upgrade-orchestrator');

const artifactRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-upgrade-artifacts-'));
test.after(() => fs.rmSync(artifactRoot, { force: true, recursive: true }));

function writeJsonArtifact(name, value) {
  const artifactPath = path.join(artifactRoot, name);
  fs.writeFileSync(artifactPath, `${JSON.stringify(value)}\n`);
  return artifactPath;
}

function sha256(contents) {
  return crypto.createHash('sha256').update(contents).digest('hex');
}

function canonicalGlobalVisualNormalization(role) {
  const contract = getGlobalVisualNormalizationContract(role);
  return {
    complete: true,
    rules: contract.rules.map((rule) => ({
      actualMatches: rule.expectedMatches,
      applied: rule.operation === 'hide',
      expectedChange: rule.expectedChange,
      expectedMatches: rule.expectedMatches,
      hiddenMatches: rule.operation === 'hide' ? rule.expectedMatches : 0,
      key: rule.key,
      operation: rule.operation,
      selector: rule.selector,
    })),
    schemaVersion: contract.schemaVersion,
  };
}

const semanticManifestFixture = strictSemanticFixtureManifest();
const sourceProvenanceFixture = {
  fingerprint: `sha256:${'c'.repeat(64)}`,
  revision: {
    commit: 'e92f2e2982e22e28bcd491d7a8358071d8e51662',
    ref: 'HEAD',
    tree: 'd'.repeat(40),
  },
  schemaVersion: 'ui-smoke-source/v1',
};

function writeCaptureInput(directory, filename, value) {
  const inputPath = path.join(directory, filename);
  const contents = Buffer.from(`${JSON.stringify(value)}\n`);
  fs.writeFileSync(inputPath, contents);
  return {
    path: inputPath,
    schemaVersion:
      typeof value.schemaVersion === 'string' || typeof value.schemaVersion === 'number'
        ? value.schemaVersion
        : null,
    sha256: sha256(contents),
    sizeBytes: contents.length,
  };
}

function createCaptureFixture(role, captureId = `${role}-capture`, options = {}) {
  const directory = path.join(artifactRoot, role);
  fs.mkdirSync(directory, { recursive: true });
  const page = options.page || 'run-details-rich-graph';
  const filename = `${page}-1280x800.png`;
  const screenshotPath = path.join(directory, filename);
  const screenshot = Buffer.concat([
    Buffer.from('89504e470d0a1a0a', 'hex'),
    Buffer.from(`${role}-screenshot`),
  ]);
  fs.writeFileSync(screenshotPath, screenshot);
  const capturedAtMs = fs.statSync(screenshotPath).mtimeMs;
  const capturedAt = new Date(capturedAtMs).toISOString();
  const revisionRole = options.revisionRole || role;
  const inputs = {
    revisionRole,
    semanticManifest: writeCaptureInput(
      directory,
      'semantic-manifest.json',
      options.semanticManifest || semanticManifestFixture,
    ),
    sourceProvenance: writeCaptureInput(
      directory,
      'source-provenance.json',
      options.sourceProvenance || sourceProvenanceFixture,
    ),
  };
  const manifest = {
    schemaVersion: 3,
    captureId,
    startedAt: new Date(capturedAtMs - 2000).toISOString(),
    completedAt: new Date(capturedAtMs + 2000).toISOString(),
    complete: true,
    deterministicRendering: {
      animations: 'disabled',
      colorScheme: 'light',
      fixedTime: '2025-01-15T12:00:00.000Z',
      locale: 'en-US',
      semanticIdNormalization: {
        derivedColorPalette: SEMANTIC_COLOR_PALETTE,
        failOnReplacementCountMismatch: true,
        mode: 'semantic-full-stack',
        rawIdentifierPolicy: 'SHA-256 attestation only',
        schemaVersion: SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
        tokenFormat: '[ui-id:<kind>:<semantic-path>]',
      },
      timezone: 'UTC',
    },
    fatalErrors: [],
    inputs,
    scenarioContractSchemaVersion: 'ui-smoke-scenarios/v2',
    results: [
      {
        capturedAt,
        filename,
        page,
        path: screenshotPath,
        required: true,
        sha256: sha256(screenshot),
        sizeBytes: screenshot.length,
        status: 'success',
        globalVisualNormalization:
          options.globalVisualNormalization || canonicalGlobalVisualNormalization(revisionRole),
        semanticIdNormalization: options.semanticIdNormalization || {
          complete: true,
          derivedColorScopes: [],
          schemaVersion: SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
          scopes: [],
          totalReplacementCount: 0,
        },
        viewport: { height: 800, width: 1280 },
      },
    ],
    summary: { complete: true, requiredIncomplete: 0 },
  };
  const manifestPath = writeJsonArtifact(`${role}/manifest.json`, manifest);
  return {
    captureId,
    filename,
    inputs,
    manifest,
    manifestPath,
    manifestSha256: sha256(fs.readFileSync(manifestPath)),
    screenshotArtifacts: [
      {
        filename,
        required: true,
        sha256: sha256(screenshot),
        sizeBytes: screenshot.length,
        status: 'success',
      },
    ],
  };
}

function canonicalSemanticIdNormalization(role, page = 'artifact-details') {
  const contract = getSemanticIdNormalizationContract(role, page);
  assert.ok(contract, `missing ${role} normalization contract for ${page}`);
  const catalog = buildSemanticIdentifierCatalog(semanticManifestFixture, role);
  const scopes = contract.scopes.map((scope) => {
    assert.ok(Array.isArray(scope.semanticIds), `${page} test contract must select semantic IDs`);
    const entries = scope.semanticIds.map((semanticId) => {
      const identifier = catalog.find((candidate) => candidate.semanticId === semanticId);
      assert.ok(identifier, `missing ${role} catalog identifier ${semanticId}`);
      return {
        ...(identifier.equivalenceClass ? { equivalenceClass: identifier.equivalenceClass } : {}),
        kind: identifier.kind,
        replacementCount: scope.minReplacementsPerIdentifier,
        semanticId: identifier.semanticId,
        sourceIdSha256: sha256(identifier.value),
        token: identifier.token,
        tokenKind: identifier.tokenKind,
        tokenSemanticId: identifier.tokenSemanticId,
      };
    });
    const replacementCount = entries.reduce((total, entry) => total + entry.replacementCount, 0);
    return {
      ...scope,
      entries,
      replacementCount,
      rootCount: replacementCount > 0 ? 1 : 0,
    };
  });
  return {
    complete: true,
    derivedColorScopes: [],
    schemaVersion: SEMANTIC_ID_NORMALIZATION_SCHEMA_VERSION,
    scopes,
    totalReplacementCount: scopes.reduce((total, scope) => total + scope.replacementCount, 0),
  };
}

const baseCaptureFixture = createCaptureFixture('base');
const headCaptureFixture = createCaptureFixture('head');

function comparisonSummary(overrides = {}) {
  return {
    schemaVersion: 2,
    captures: {
      base: {
        captureId: baseCaptureFixture.captureId,
        manifestSha256: baseCaptureFixture.manifestSha256,
        requiredFilenames: [baseCaptureFixture.filename],
      },
      head: {
        captureId: headCaptureFixture.captureId,
        manifestSha256: headCaptureFixture.manifestSha256,
        requiredFilenames: [headCaptureFixture.filename],
      },
    },
    fatalErrors: [],
    passed: true,
    results: [
      {
        filename: baseCaptureFixture.filename,
        page: 'run-details-rich-graph',
        status: 'success',
      },
    ],
    valid: true,
    ...overrides,
  };
}

const comparisonSummaryPath = writeJsonArtifact('comparison-summary.json', comparisonSummary());
const comparisonReportPath = path.join(artifactRoot, 'comparison-report.html');
fs.writeFileSync(comparisonReportPath, '<!doctype html><title>comparison</title>\n');

function request() {
  return {
    artifactRoot,
    baseRevision: '3487bc4c1ab141e76b2f8d9bba71f2b1d54a964f',
    headRevision: 'e92f2e2982e22e28bcd491d7a8358071d8e51662',
    runId: 'upgrade-test-run',
    sourceProvenance: sourceProvenanceFixture,
  };
}

function capabilities(overrides = {}) {
  return {
    migration: { available: true, version: 'mlmd-to-native/v1' },
    startupGate: { available: true, migrationVersion: 'mlmd-to-native/v1' },
    ...overrides,
  };
}

function removedResources() {
  return [
    {
      apiVersion: 'apps/v1',
      expectedRemoval: true,
      kind: 'Deployment',
      name: 'metadata-writer',
      namespace: 'kubeflow',
    },
    {
      apiVersion: 'v1',
      expectedRemoval: true,
      kind: 'Service',
      name: 'metadata-grpc-service',
      namespace: 'kubeflow',
    },
  ];
}

function harness(overrides = {}) {
  const calls = [];
  const written = [];
  const environmentId = 'kind-ui-smoke-upgrade';
  const pvcIdentities = {
    'kubeflow/mysql-pv-claim': 'mysql-uid',
    'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
  };
  const semanticFixtures = {
    'historical-artifact': { digest: 'artifact-digest', present: true },
    'historical-run': { digest: 'run-digest', present: true },
  };

  const operation = (name, implementation) => async (input) => {
    calls.push(name);
    return implementation(input);
  };

  const operations = {
    deployBase: operation('deployBase', async () => ({ success: true, environmentId })),
    seedBase: operation('seedBase', async () => ({
      success: true,
      semanticManifest: semanticManifestFixture,
    })),
    captureBase: operation('captureBase', async () => ({
      captureValidity: CAPTURE_VALIDITY.VALID,
      manifestPath: baseCaptureFixture.manifestPath,
      success: true,
    })),
    freezeBase: operation('freezeBase', async () => ({ success: true })),
    readBaseState: operation('readBaseState', async () => ({
      success: true,
      pvcIdentities,
      semanticFixtures,
    })),
    migrate: operation('migrate', async () => ({
      success: true,
      migrationVersion: 'mlmd-to-native/v1',
    })),
    validateMigration: operation('validateMigration', async () => ({
      success: true,
      durableMarker: {
        durable: true,
        status: 'complete',
        validated: true,
        version: 'mlmd-to-native/v1',
      },
    })),
    deployHead: operation('deployHead', async () => ({ success: true, environmentId })),
    validateStartupGate: operation('validateStartupGate', async () => ({
      accepted: true,
      durableMarkerObserved: true,
      enforced: true,
      migrationVersion: 'mlmd-to-native/v1',
      success: true,
    })),
    readHeadState: operation('readHeadState', async () => ({
      success: true,
      pvcIdentities: { ...pvcIdentities },
      semanticFixtures: { ...semanticFixtures },
    })),
    verifyPreservation: operation('verifyPreservation', async () => ({
      preserved: true,
      pvcIdentitiesPreserved: true,
      semanticFixturesPreserved: true,
      success: true,
    })),
    pruneRemovedResources: operation('pruneRemovedResources', async () => ({ success: true })),
    captureHead: operation('captureHead', async () => ({
      captureValidity: CAPTURE_VALIDITY.VALID,
      manifestPath: headCaptureFixture.manifestPath,
      success: true,
    })),
    compareCaptures: operation('compareCaptures', async () => ({
      captureValidity: CAPTURE_VALIDITY.VALID,
      reportPath: comparisonReportPath,
      success: true,
      summaryPath: comparisonSummaryPath,
    })),
    cleanupEnvironment: async () => ({ success: true }),
    async writeResult(result) {
      calls.push('writeResult');
      written.push(result);
    },
    ...overrides,
  };
  return { calls, operations, written };
}

test('capabilities are unavailable unless both matching migration and startup-gate versions exist', () => {
  assert.deepEqual(assessUpgradeCapabilities(), {
    available: false,
    contract: UPGRADE_CAPABILITY_CONTRACT,
    gateVersion: null,
    migrationVersion: null,
    missing: ['migration', 'startup_gate'],
    requirement: MIGRATION_REQUIREMENT,
  });
  assert.deepEqual(
    assessUpgradeCapabilities({
      migration: { available: true, version: 'v1' },
      startupGate: { available: true, migrationVersion: 'v2' },
    }).missing,
    ['version_mismatch'],
  );
  assert.equal(assessUpgradeCapabilities(capabilities()).available, true);
  assert.equal(UPGRADE_CAPABILITY_CONTRACT.schemaVersion, CONTRACT_VERSION);
  assert.equal(Object.hasOwn(MIGRATION_REQUIREMENT, 'available'), false);
  assert.equal(Object.isFrozen(SAFE_PRUNE_KINDS), true);
  assert.equal(assessUpgradeCapabilities(null).available, false);
});

test('missing migration writes a structured result before any cluster or head mutation', async () => {
  const { calls, operations, written } = harness();
  const result = await orchestrateUpgrade({
    capabilities: {
      migration: { available: false },
      startupGate: { available: false },
    },
    operations,
    removedResources: removedResources(),
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.MIGRATION_UNAVAILABLE);
  assert.equal(result.complete, false);
  assert.equal(result.contractVersion, CONTRACT_VERSION);
  assert.equal(result.baseCaptured, false);
  assert.equal(result.headCaptured, false);
  assert.deepEqual(result.migration.missing, ['migration', 'startup_gate']);
  assert.equal(result.migration.requirement.issueNumber, 14029);
  assert.deepEqual(calls, ['writeResult']);
  assert.equal(written.length, 1);
  assert.equal(written[0].captureValidity, CAPTURE_VALIDITY.MIGRATION_UNAVAILABLE);
});

test('a missing startup gate is also fail-closed before base deployment', async () => {
  const { calls, operations } = harness();
  const result = await orchestrateUpgrade({
    capabilities: {
      migration: { available: true, version: 'mlmd-to-native/v1' },
      startupGate: { available: false },
    },
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.MIGRATION_UNAVAILABLE);
  assert.deepEqual(calls, ['writeResult']);
});

test('runs the complete upgrade lifecycle in exact fail-closed order', async () => {
  const { calls, operations } = harness({
    pruneRemovedResources: async ({ resources, state }) => {
      calls.push('pruneRemovedResources');
      assert.equal(state.preservation.preserved, true);
      assert.deepEqual(
        resources.map((resource) => `${resource.kind}/${resource.name}`),
        ['Deployment/metadata-writer', 'Service/metadata-grpc-service'],
      );
      return { success: true };
    },
  });

  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    removedResources: removedResources(),
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.VALID);
  assert.equal(result.complete, true);
  assert.equal(result.comparisonPassed, true);
  assert.equal(result.environmentId, 'kind-ui-smoke-upgrade');
  assert.equal(result.baseCapture.captureId, baseCaptureFixture.captureId);
  assert.equal(result.baseCapture.manifestSha256, baseCaptureFixture.manifestSha256);
  assert.deepEqual(result.baseCapture.requiredFilenames, [baseCaptureFixture.filename]);
  assert.equal(result.headCapture.captureId, headCaptureFixture.captureId);
  assert.equal(result.comparison.summarySha256.length, 64);
  assert.equal(result.comparison.reportSha256.length, 64);
  assert.equal(result.request.baseRevision, request().baseRevision);
  assert.equal(result.request.headRevision, request().headRevision);
  assert.deepEqual(calls, [
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
    'writeResult',
  ]);
  assert.deepEqual(
    result.phaseHistory.filter(({ status }) => status === 'completed').map(({ phase }) => phase),
    [
      PHASES.CONFIGURATION_CHECK,
      PHASES.DEPLOY_BASE,
      PHASES.SEED_BASE,
      PHASES.CAPTURE_BASE,
      PHASES.FREEZE_BASE,
      PHASES.READ_BASE_STATE,
      PHASES.MIGRATE,
      PHASES.VALIDATE_MIGRATION,
      PHASES.DEPLOY_HEAD,
      PHASES.VALIDATE_STARTUP_GATE,
      PHASES.READ_HEAD_STATE,
      PHASES.VERIFY_PRESERVATION,
      PHASES.PRUNE_SAFE_REMOVED_RESOURCES,
      PHASES.CAPTURE_HEAD,
      PHASES.COMPARE_CAPTURES,
      PHASES.COMPLETE,
    ],
  );
});

test('capture manifests must be complete, non-degraded artifacts inside the run root', async (t) => {
  await t.test('degraded capture', async () => {
    const degradedPath = writeJsonArtifact('degraded-manifest.json', {
      ...baseCaptureFixture.manifest,
      captureId: 'degraded-capture',
      results: baseCaptureFixture.manifest.results.map((capture) => ({
        ...capture,
        status: 'degraded',
      })),
    });
    const { calls, operations } = harness({
      captureBase: async () => {
        calls.push('captureBase');
        return {
          captureValidity: CAPTURE_VALIDITY.VALID,
          manifestPath: degradedPath,
          success: true,
        };
      },
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(result.error.message, /incomplete or degraded/);
    assert.equal(result.baseCaptured, false);
    assert.equal(calls.includes('freezeBase'), false);
  });

  await t.test('path traversal', async () => {
    const outsideRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-outside-artifact-'));
    t.after(() => fs.rmSync(outsideRoot, { force: true, recursive: true }));
    const outsideManifest = path.join(outsideRoot, 'manifest.json');
    fs.writeFileSync(outsideManifest, JSON.stringify(baseCaptureFixture.manifest));
    const { operations } = harness({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: outsideManifest,
        success: true,
      }),
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /outside request\.artifactRoot/);
  });

  await t.test('symlink', async () => {
    const linkPath = path.join(artifactRoot, 'linked-manifest.json');
    fs.symlinkSync(baseCaptureFixture.manifestPath, linkPath);
    t.after(() => fs.rmSync(linkPath, { force: true }));
    const { operations } = harness({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: linkPath,
        success: true,
      }),
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /non-symlink regular file/);
  });
});

test('capture artifact validation binds every successful PNG to its manifest', async (t) => {
  const runWithBaseCapture = async (manifestPath) => {
    const { operations } = harness({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath,
        success: true,
      }),
    });
    return orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });
  };

  await t.test('unsupported manifest schema', async () => {
    const fixture = createCaptureFixture('bad-schema', undefined, { revisionRole: 'base' });
    const invalidManifest = { ...fixture.manifest, schemaVersion: 99 };
    const manifestPath = writeJsonArtifact('bad-schema/invalid-manifest.json', invalidManifest);
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /unsupported schema version/);
  });

  await t.test('missing deterministic semantic normalization contract', async () => {
    const fixture = createCaptureFixture('missing-normalization-contract', undefined, {
      revisionRole: 'base',
    });
    const invalidManifest = { ...fixture.manifest };
    delete invalidManifest.deterministicRendering;
    const manifestPath = writeJsonArtifact(
      'missing-normalization-contract/invalid-manifest.json',
      invalidManifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /no deterministic rendering contract/);
  });

  await t.test('wrong deterministic semantic normalization palette', async () => {
    const fixture = createCaptureFixture('wrong-normalization-palette', undefined, {
      revisionRole: 'base',
    });
    const invalidManifest = {
      ...fixture.manifest,
      deterministicRendering: {
        ...fixture.manifest.deterministicRendering,
        semanticIdNormalization: {
          ...fixture.manifest.deterministicRendering.semanticIdNormalization,
          derivedColorPalette: ['#000000'],
        },
      },
    };
    const manifestPath = writeJsonArtifact(
      'wrong-normalization-palette/invalid-manifest.json',
      invalidManifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /must use semantic ID normalization contract/);
  });

  await t.test('browser-compatibility normalization mode', async () => {
    const fixture = createCaptureFixture('wrong-normalization-mode', undefined, {
      revisionRole: 'base',
    });
    fixture.manifest.deterministicRendering.semanticIdNormalization.mode =
      'disabled-browser-compatibility';
    const manifestPath = writeJsonArtifact(
      'wrong-normalization-mode/invalid-manifest.json',
      fixture.manifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /must use semantic ID normalization contract/);
  });

  await t.test('weakened deterministic normalization contract', async () => {
    const fixture = createCaptureFixture('weakened-normalization-contract', undefined, {
      revisionRole: 'base',
    });
    fixture.manifest.deterministicRendering.semanticIdNormalization.failOnReplacementCountMismatch = false;
    const manifestPath = writeJsonArtifact(
      'weakened-normalization-contract/invalid-manifest.json',
      fixture.manifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /must use semantic ID normalization contract/);
  });

  await t.test('stale semantic scenario contract', async () => {
    const fixture = createCaptureFixture('wrong-scenario-contract', undefined, {
      revisionRole: 'base',
    });
    fixture.manifest.scenarioContractSchemaVersion = 'ui-smoke-scenarios/v1';
    const manifestPath = writeJsonArtifact(
      'wrong-scenario-contract/invalid-manifest.json',
      fixture.manifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /must use scenario contract ui-smoke-scenarios\/v2/);
  });

  await t.test('missing global visual normalization evidence', async () => {
    const fixture = createCaptureFixture('missing-global-normalization', undefined, {
      revisionRole: 'base',
    });
    delete fixture.manifest.results[0].globalVisualNormalization;
    const manifestPath = writeJsonArtifact(
      'missing-global-normalization/invalid-manifest.json',
      fixture.manifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(result.error.message, /invalid global visual normalization evidence.*missing/);
  });

  await t.test('wrong global visual normalization evidence', async () => {
    const fixture = createCaptureFixture('wrong-global-normalization', undefined, {
      revisionRole: 'base',
    });
    fixture.manifest.results[0].globalVisualNormalization.rules[0].actualMatches = 0;
    const manifestPath = writeJsonArtifact(
      'wrong-global-normalization/invalid-manifest.json',
      fixture.manifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(
      result.error.message,
      /invalid global visual normalization evidence.*does not match its contract/,
    );
  });

  await t.test('empty canonical scenario normalization evidence', async () => {
    const fixture = createCaptureFixture('empty-canonical-normalization', undefined, {
      page: 'artifact-details',
      revisionRole: 'base',
    });
    const result = await runWithBaseCapture(fixture.manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(
      result.error.message,
      /does not attest the base semantic ID normalization contract/,
    );
  });

  await t.test('unknown semantic scenario with empty normalization evidence', async () => {
    const fixture = createCaptureFixture('unknown-semantic-scenario', undefined, {
      page: 'runs',
      revisionRole: 'base',
    });
    const result = await runWithBaseCapture(fixture.manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /does not bind canonical semantic scenario runs/);
  });

  await t.test('missing semantic normalization attestation', async () => {
    const fixture = createCaptureFixture('missing-normalization-evidence', undefined, {
      revisionRole: 'base',
    });
    const [capture] = fixture.manifest.results;
    const invalidCapture = { ...capture };
    delete invalidCapture.semanticIdNormalization;
    const invalidManifest = { ...fixture.manifest, results: [invalidCapture] };
    const manifestPath = writeJsonArtifact(
      'missing-normalization-evidence/invalid-manifest.json',
      invalidManifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /invalid semantic ID normalization evidence/);
  });

  await t.test('inconsistent semantic normalization attestation', async () => {
    const fixture = createCaptureFixture('bad-normalization-evidence', undefined, {
      revisionRole: 'base',
    });
    const [capture] = fixture.manifest.results;
    const invalidManifest = {
      ...fixture.manifest,
      results: [
        {
          ...capture,
          semanticIdNormalization: {
            ...capture.semanticIdNormalization,
            totalReplacementCount: 1,
          },
        },
      ],
    };
    const manifestPath = writeJsonArtifact(
      'bad-normalization-evidence/invalid-manifest.json',
      invalidManifest,
    );
    const result = await runWithBaseCapture(manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /inconsistent total replacement count/);
  });

  await t.test('missing screenshot', async () => {
    const fixture = createCaptureFixture('missing-png', undefined, { revisionRole: 'base' });
    fs.unlinkSync(path.join(path.dirname(fixture.manifestPath), fixture.filename));
    const result = await runWithBaseCapture(fixture.manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /ENOENT|unreadable path/);
  });

  await t.test('corrupt screenshot hash', async () => {
    const fixture = createCaptureFixture('corrupt-png', undefined, { revisionRole: 'base' });
    fs.appendFileSync(path.join(path.dirname(fixture.manifestPath), fixture.filename), 'tampered');
    const result = await runWithBaseCapture(fixture.manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /declared hash|declared size/);
  });

  await t.test('symlinked screenshot', async () => {
    const fixture = createCaptureFixture('symlink-png', undefined, { revisionRole: 'base' });
    const screenshotPath = path.join(path.dirname(fixture.manifestPath), fixture.filename);
    const targetPath = path.join(path.dirname(fixture.manifestPath), 'target.png');
    fs.renameSync(screenshotPath, targetPath);
    fs.symlinkSync(targetPath, screenshotPath);
    const result = await runWithBaseCapture(fixture.manifestPath);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.match(result.error.message, /non-symlink regular file/);
  });
});

test('upgrade capture validation binds normalization evidence to the semantic catalog', async (t) => {
  const canonicalBase = createCaptureFixture('catalog-canonical-base', undefined, {
    page: 'artifact-details',
    revisionRole: 'base',
    semanticIdNormalization: canonicalSemanticIdNormalization('base'),
  });
  const runWithBaseCapture = async (manifestPath) => {
    const { operations } = harness({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath,
        success: true,
      }),
    });
    return orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });
  };

  await t.test('accepts canonical catalog-backed normalization evidence', async () => {
    const canonicalHead = createCaptureFixture('catalog-canonical-head', undefined, {
      page: 'artifact-details',
      revisionRole: 'head',
      semanticIdNormalization: canonicalSemanticIdNormalization('head'),
    });
    const summaryPath = writeJsonArtifact('catalog-canonical-summary.json', {
      captures: {
        base: {
          captureId: canonicalBase.captureId,
          manifestSha256: canonicalBase.manifestSha256,
          requiredFilenames: [canonicalBase.filename],
        },
        head: {
          captureId: canonicalHead.captureId,
          manifestSha256: canonicalHead.manifestSha256,
          requiredFilenames: [canonicalHead.filename],
        },
      },
      fatalErrors: [],
      passed: true,
      results: [
        {
          filename: canonicalBase.filename,
          page: 'artifact-details',
          status: 'success',
        },
      ],
      schemaVersion: 2,
      valid: true,
    });
    const { operations } = harness({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: canonicalBase.manifestPath,
        success: true,
      }),
      captureHead: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: canonicalHead.manifestPath,
        success: true,
      }),
      compareCaptures: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        reportPath: comparisonReportPath,
        success: true,
        summaryPath,
      }),
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.complete, true);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.VALID);
    assert.equal(result.phase, PHASES.COMPLETE);
  });

  await t.test('rejects a shape-valid source hash that is absent from the catalog', async () => {
    const manifest = structuredClone(canonicalBase.manifest);
    manifest.results[0].semanticIdNormalization.scopes[0].entries[0].sourceIdSha256 = '0'.repeat(
      64,
    );
    const manifestPath = writeJsonArtifact(
      'catalog-canonical-base/tampered-hash-manifest.json',
      manifest,
    );

    const result = await runWithBaseCapture(manifestPath);

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(result.error.message, /does not match its attested semantic fixture manifest/);
  });

  await t.test('rejects a shape-valid token identity that is absent from the catalog', async () => {
    const manifest = structuredClone(canonicalBase.manifest);
    const [artifactScope, uriScope] = manifest.results[0].semanticIdNormalization.scopes;
    const [artifactEntry] = artifactScope.entries;
    const [uriEntry] = uriScope.entries;
    artifactEntry.token = uriEntry.token;
    artifactEntry.tokenKind = uriEntry.tokenKind;
    artifactEntry.tokenSemanticId = uriEntry.tokenSemanticId;
    const manifestPath = writeJsonArtifact(
      'catalog-canonical-base/tampered-token-manifest.json',
      manifest,
    );

    const result = await runWithBaseCapture(manifestPath);

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(result.error.message, /does not match its attested semantic fixture manifest/);
  });
});

test('capture manifests are bound to their revision role and exact provenance inputs', async (t) => {
  const runWithOverrides = async (operationOverrides) => {
    const { calls, operations } = harness(operationOverrides);
    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });
    return { calls, result };
  };

  await t.test('swapped base/head capture', async () => {
    const { calls, result } = await runWithOverrides({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: headCaptureFixture.manifestPath,
        success: true,
      }),
    });

    assert.equal(result.complete, false);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(result.error.message, /revisionRole must be base/);
    assert.equal(calls.includes('freezeBase'), false);
  });

  await t.test('base capture reused for head', async () => {
    const { calls, result } = await runWithOverrides({
      captureHead: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: baseCaptureFixture.manifestPath,
        success: true,
      }),
    });

    assert.equal(result.complete, false);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_HEAD);
    assert.match(result.error.message, /revisionRole must be head/);
    assert.equal(calls.includes('compareCaptures'), false);
  });

  await t.test('wrong semantic manifest', async () => {
    const wrongSemantic = createCaptureFixture('wrong-semantic', undefined, {
      revisionRole: 'base',
      semanticManifest: {
        ...semanticManifestFixture,
        fixtures: ['unrelated-fixture'],
      },
    });
    const { result } = await runWithOverrides({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: wrongSemantic.manifestPath,
        success: true,
      }),
    });

    assert.equal(result.complete, false);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_BASE);
    assert.match(result.error.message, /semanticManifest does not match the expected provenance/);
  });

  await t.test('wrong source provenance', async () => {
    const wrongSource = createCaptureFixture('wrong-source', undefined, {
      revisionRole: 'head',
      sourceProvenance: {
        ...sourceProvenanceFixture,
        fingerprint: `sha256:${'f'.repeat(64)}`,
      },
    });
    const { calls, result } = await runWithOverrides({
      captureHead: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: wrongSource.manifestPath,
        success: true,
      }),
    });

    assert.equal(result.complete, false);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.CAPTURE_FAILED);
    assert.equal(result.phase, PHASES.CAPTURE_HEAD);
    assert.match(result.error.message, /sourceProvenance does not match the expected provenance/);
    assert.equal(calls.includes('compareCaptures'), false);
  });
});

test('comparison artifacts are cryptographically bound to the exact capture pair', async (t) => {
  const runWithOverrides = async (operationOverrides) => {
    const { operations } = harness(operationOverrides);
    return orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });
  };

  await t.test('same capture ID', async () => {
    const sameIdHead = createCaptureFixture('same-id-head', baseCaptureFixture.captureId, {
      revisionRole: 'head',
    });
    const result = await runWithOverrides({
      captureHead: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: sameIdHead.manifestPath,
        success: true,
      }),
    });
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
    assert.match(result.error.message, /distinct capture IDs/);
  });

  await t.test('different required filename sets', async () => {
    const differentHead = createCaptureFixture('different-head', 'different-head-capture', {
      page: 'run-details-task-logs',
      revisionRole: 'head',
    });
    const result = await runWithOverrides({
      captureHead: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: differentHead.manifestPath,
        success: true,
      }),
    });
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
    assert.match(result.error.message, /required filename sets differ/);
  });

  await t.test('wrong manifest hash attestation', async () => {
    const summaryPath = writeJsonArtifact(
      'wrong-hash-summary.json',
      comparisonSummary({
        captures: {
          ...comparisonSummary().captures,
          head: {
            ...comparisonSummary().captures.head,
            manifestSha256: '0'.repeat(64),
          },
        },
      }),
    );
    const result = await runWithOverrides({
      compareCaptures: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        reportPath: comparisonReportPath,
        success: true,
        summaryPath,
      }),
    });
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
    assert.match(result.error.message, /head capture attestation is invalid/);
  });

  await t.test('capture mutated after validation', async () => {
    const mutableBase = createCaptureFixture('mutated-after-capture', undefined, {
      revisionRole: 'base',
    });
    const summaryPath = writeJsonArtifact(
      'mutated-after-capture-summary.json',
      comparisonSummary({
        captures: {
          ...comparisonSummary().captures,
          base: {
            captureId: mutableBase.captureId,
            manifestSha256: mutableBase.manifestSha256,
            requiredFilenames: [mutableBase.filename],
          },
        },
      }),
    );
    const result = await runWithOverrides({
      captureBase: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        manifestPath: mutableBase.manifestPath,
        success: true,
      }),
      compareCaptures: async () => {
        fs.appendFileSync(
          path.join(path.dirname(mutableBase.manifestPath), mutableBase.filename),
          'mutated',
        );
        return {
          captureValidity: CAPTURE_VALIDITY.VALID,
          reportPath: comparisonReportPath,
          success: true,
          summaryPath,
        };
      },
    });
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
    assert.match(result.error.message, /base capture screenshot .* changed/);
  });

  await t.test('unrelated or omitted result filenames', async () => {
    const summaryPath = writeJsonArtifact(
      'unrelated-summary.json',
      comparisonSummary({
        results: [
          { filename: baseCaptureFixture.filename, page: 'runs', status: 'success' },
          { filename: 'unrelated-1280x800.png', page: 'unrelated', status: 'success' },
        ],
      }),
    );
    const result = await runWithOverrides({
      compareCaptures: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        reportPath: comparisonReportPath,
        success: true,
        summaryPath,
      }),
    });
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
    assert.match(result.error.message, /exact required filename set/);
  });

  await t.test('non-HTML report', async () => {
    const reportPath = path.join(artifactRoot, 'not-a-report.html');
    fs.writeFileSync(reportPath, 'not html');
    const result = await runWithOverrides({
      compareCaptures: async () => ({
        captureValidity: CAPTURE_VALIDITY.VALID,
        reportPath,
        success: true,
        summaryPath: comparisonSummaryPath,
      }),
    });
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
    assert.match(result.error.message, /does not contain an HTML report/);
  });
});

test('the adapter helper converts built-in comparison output into attested artifacts', async () => {
  const builtinDirectory = path.join(artifactRoot, 'builtin-comparison');
  fs.mkdirSync(builtinDirectory, { recursive: true });
  const comparisonImage = Buffer.concat([
    Buffer.from('89504e470d0a1a0a', 'hex'),
    Buffer.from('comparison-image'),
  ]);
  fs.writeFileSync(path.join(builtinDirectory, baseCaptureFixture.filename), comparisonImage);
  const sourceSummaryPath = path.join(builtinDirectory, 'summary.json');
  const builtinSummary = summarizeComparison({
    fatalErrors: [],
    mainLabel: 'base',
    options: {
      diffThreshold: 0,
      failThreshold: 0,
      looksSameClusterSize: 8,
      looksSameTolerance: 2.3,
    },
    prLabel: 'head',
    results: [
      {
        diffPercent: 0,
        exceedsFailThreshold: false,
        filename: baseCaptureFixture.filename,
        hasVisualDiff: false,
        page: 'runs',
        required: true,
        status: 'success',
      },
    ],
    sourceMode: 'manifest',
  });
  fs.writeFileSync(sourceSummaryPath, `${JSON.stringify(builtinSummary)}\n`);
  const operationResult = writeUpgradeComparisonArtifacts({
    artifactRoot,
    baseCapture: {
      captureId: baseCaptureFixture.captureId,
      inputs: baseCaptureFixture.inputs,
      manifestPath: baseCaptureFixture.manifestPath,
      manifestSha256: baseCaptureFixture.manifestSha256,
      requiredFilenames: [baseCaptureFixture.filename],
      screenshotArtifacts: baseCaptureFixture.screenshotArtifacts,
    },
    headCapture: {
      captureId: headCaptureFixture.captureId,
      inputs: headCaptureFixture.inputs,
      manifestPath: headCaptureFixture.manifestPath,
      manifestSha256: headCaptureFixture.manifestSha256,
      requiredFilenames: [headCaptureFixture.filename],
      screenshotArtifacts: headCaptureFixture.screenshotArtifacts,
    },
    reportPath: path.join(artifactRoot, 'generated-report.html'),
    sourceSummaryPath,
    summaryPath: path.join(artifactRoot, 'generated-summary.json'),
  });

  const attested = JSON.parse(fs.readFileSync(operationResult.summaryPath, 'utf8'));
  assert.equal(attested.captures.base.captureId, baseCaptureFixture.captureId);
  assert.equal(attested.captures.head.manifestSha256, headCaptureFixture.manifestSha256);
  assert.deepEqual(
    attested.results.map(({ filename }) => filename),
    [baseCaptureFixture.filename],
  );
  assert.match(
    fs.readFileSync(operationResult.reportPath, 'utf8'),
    /Comparison and highlighted diff/,
  );

  const { operations } = harness({
    compareCaptures: async () => operationResult,
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });
  assert.equal(result.complete, true);
  assert.equal(result.comparisonPassed, true);
});

test('comparison requires a valid summary and HTML report before completion', async () => {
  const invalidSummaryPath = writeJsonArtifact(
    'invalid-comparison-summary.json',
    comparisonSummary({
      fatalErrors: ['missing semantic pair'],
      passed: false,
      results: [],
      valid: false,
    }),
  );
  const { calls, operations } = harness({
    compareCaptures: async () => {
      calls.push('compareCaptures');
      return {
        captureValidity: CAPTURE_VALIDITY.VALID,
        reportPath: comparisonReportPath,
        success: true,
        summaryPath: invalidSummaryPath,
      };
    },
  });

  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.complete, false);
  assert.equal(result.captureValidity, CAPTURE_VALIDITY.COMPARISON_FAILED);
  assert.equal(result.phase, PHASES.COMPARE_CAPTURES);
  assert.match(result.error.message, /not a valid visual comparison/);
  assert.deepEqual(calls.slice(-2), ['compareCaptures', 'writeResult']);
});

test('a valid visual diff failure stays distinct from capture validity', async () => {
  const failedThresholdSummaryPath = writeJsonArtifact(
    'threshold-comparison-summary.json',
    comparisonSummary({ passed: false }),
  );
  const { operations } = harness({
    compareCaptures: async () => ({
      captureValidity: CAPTURE_VALIDITY.VALID,
      reportPath: comparisonReportPath,
      success: true,
      summaryPath: failedThresholdSummaryPath,
    }),
  });

  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.complete, true);
  assert.equal(result.captureValidity, CAPTURE_VALIDITY.VALID);
  assert.equal(result.comparisonPassed, false);
});

test('result persistence failure can never return an optimistic success', async () => {
  const { operations } = harness({
    async writeResult() {
      throw new Error('disk unavailable');
    },
  });

  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.complete, false);
  assert.equal(result.captureValidity, CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE);
  assert.equal(result.phase, PHASES.PERSIST_RESULT);
  assert.match(result.resultWriteError.message, /disk unavailable/);
});

test('migration failure stops before validation and head deployment', async () => {
  const { calls, operations } = harness({
    migrate: async () => {
      calls.push('migrate');
      throw new Error('migration job failed');
    },
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.MIGRATION_FAILED);
  assert.equal(result.phase, PHASES.MIGRATE);
  assert.match(result.error.message, /migration job failed/);
  assert.deepEqual(result.phaseHistory.at(-1), {
    phase: PHASES.MIGRATE,
    status: 'failed',
  });
  assert.deepEqual(calls, [
    'deployBase',
    'seedBase',
    'captureBase',
    'freezeBase',
    'readBaseState',
    'migrate',
    'writeResult',
  ]);
});

test('a non-durable migration marker blocks head deployment', async () => {
  const { calls, operations } = harness({
    validateMigration: async () => {
      calls.push('validateMigration');
      return {
        success: true,
        durableMarker: {
          durable: false,
          status: 'complete',
          validated: true,
          version: 'mlmd-to-native/v1',
        },
      };
    },
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.MIGRATION_FAILED);
  assert.equal(result.phase, PHASES.VALIDATE_MIGRATION);
  assert.deepEqual(result.phaseHistory.at(-1), {
    phase: PHASES.VALIDATE_MIGRATION,
    status: 'failed',
  });
  assert.equal(calls.includes('deployHead'), false);
});

test('head deployment must target the exact base environment', async () => {
  const { calls, operations } = harness({
    deployHead: async () => {
      calls.push('deployHead');
      return { success: true, environmentId: 'different-cluster' };
    },
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE);
  assert.equal(result.phase, PHASES.DEPLOY_HEAD);
  assert.equal(calls.includes('validateStartupGate'), false);
});

test('startup-gate failure stops state verification, pruning, and capture', async () => {
  const { calls, operations } = harness({
    validateStartupGate: async () => {
      calls.push('validateStartupGate');
      return {
        accepted: true,
        durableMarkerObserved: false,
        enforced: true,
        migrationVersion: 'mlmd-to-native/v1',
        success: true,
      };
    },
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.STARTUP_GATE_FAILED);
  assert.equal(result.phase, PHASES.VALIDATE_STARTUP_GATE);
  assert.equal(calls.includes('readHeadState'), false);
  assert.equal(calls.includes('pruneRemovedResources'), false);
  assert.equal(calls.includes('captureHead'), false);
});

test('PVC replacement is a preservation failure and blocks pruning', async () => {
  const { calls, operations } = harness({
    readHeadState: async () => {
      calls.push('readHeadState');
      return {
        success: true,
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'replacement-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-artifact': { digest: 'artifact-digest', present: true },
          'historical-run': { digest: 'run-digest', present: true },
        },
      };
    },
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.PRESERVATION_FAILED);
  assert.equal(result.phase, PHASES.READ_HEAD_STATE);
  assert.match(result.error.message, /PVC kubeflow\/mysql-pv-claim was replaced/);
  assert.equal(calls.includes('verifyPreservation'), false);
  assert.equal(calls.includes('pruneRemovedResources'), false);
});

test('adapter callbacks cannot mutate the frozen preservation baseline', async () => {
  const { calls, operations } = harness({
    migrate: async ({ baseState, capabilities: receivedCapabilities }) => {
      calls.push('migrate');
      assert.equal(Object.isFrozen(baseState), true);
      assert.equal(Object.isFrozen(baseState.pvcIdentities), true);
      assert.equal(Object.isFrozen(receivedCapabilities), true);
      assert.equal(Object.isFrozen(receivedCapabilities.contract), true);
      baseState.pvcIdentities['kubeflow/mysql-pv-claim'] = 'replacement-uid';
      receivedCapabilities.migrationVersion = 'forged-version';
      return { migrationVersion: 'mlmd-to-native/v1', success: true };
    },
    readHeadState: async () => {
      calls.push('readHeadState');
      return {
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'replacement-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-artifact': { digest: 'artifact-digest', present: true },
          'historical-run': { digest: 'run-digest', present: true },
        },
        success: true,
      };
    },
  });

  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.PRESERVATION_FAILED);
  assert.match(result.error.message, /PVC kubeflow\/mysql-pv-claim was replaced/);
  assert.equal(calls.includes('verifyPreservation'), false);
});

test('semantic continuity requires presence and matching revision-independent evidence', async (t) => {
  await t.test('present=false', async () => {
    const { operations } = harness({
      readHeadState: async () => ({
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'mysql-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-artifact': { digest: 'artifact-digest', present: false },
          'historical-run': { digest: 'run-digest', present: true },
        },
        success: true,
      }),
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.PRESERVATION_FAILED);
    assert.match(result.error.message, /must prove present=true/);
  });

  await t.test('changed digest', async () => {
    const { operations } = harness({
      readHeadState: async () => ({
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'mysql-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-artifact': { digest: 'changed-digest', present: true },
          'historical-run': { digest: 'run-digest', present: true },
        },
        success: true,
      }),
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.captureValidity, CAPTURE_VALIDITY.PRESERVATION_FAILED);
    assert.match(result.error.message, /Semantic fixture historical-artifact changed/);
  });

  await t.test('revision-specific IDs may differ', async () => {
    const { operations } = harness({
      readBaseState: async () => ({
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'mysql-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-artifact': {
            digest: 'artifact-digest',
            id: 'legacy-mlmd-id',
            present: true,
          },
          'historical-run': { digest: 'run-digest', id: 'legacy-run-id', present: true },
        },
        success: true,
      }),
      readHeadState: async () => ({
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'mysql-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-artifact': {
            digest: 'artifact-digest',
            id: 'native-artifact-id',
            present: true,
          },
          'historical-run': { digest: 'run-digest', id: 'native-run-id', present: true },
        },
        success: true,
      }),
    });

    const result = await orchestrateUpgrade({
      capabilities: capabilities(),
      operations,
      request: request(),
    });

    assert.equal(result.complete, true);
    assert.equal(result.captureValidity, CAPTURE_VALIDITY.VALID);
  });
});

test('missing semantic fixture identities block delegated preservation and pruning', async () => {
  const { calls, operations } = harness({
    readHeadState: async () => {
      calls.push('readHeadState');
      return {
        success: true,
        pvcIdentities: {
          'kubeflow/mysql-pv-claim': 'mysql-uid',
          'kubeflow/seaweedfs-pvc': 'seaweedfs-uid',
        },
        semanticFixtures: {
          'historical-run': { digest: 'run-digest', present: true },
        },
      };
    },
  });
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.PRESERVATION_FAILED);
  assert.equal(result.phase, PHASES.READ_HEAD_STATE);
  assert.match(result.error.message, /Semantic fixture identity set changed/);
  assert.equal(calls.includes('verifyPreservation'), false);
  assert.equal(calls.includes('pruneRemovedResources'), false);
});

test('unsafe or implicit prune resources are rejected before base deployment', async () => {
  assert.throws(
    () =>
      validateSafeRemovedResources([
        {
          apiVersion: 'v1',
          expectedRemoval: true,
          kind: 'PersistentVolumeClaim',
          name: 'mysql-pv-claim',
          namespace: 'kubeflow',
        },
      ]),
    /Refusing to prune unsafe Kubernetes kind PersistentVolumeClaim/,
  );

  const { calls, operations } = harness();
  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    removedResources: [
      {
        apiVersion: 'apps/v1',
        expectedRemoval: false,
        kind: 'Deployment',
        name: 'metadata-writer',
        namespace: 'kubeflow',
      },
    ],
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE);
  assert.equal(result.phase, PHASES.CONFIGURATION_CHECK);
  assert.deepEqual(calls, ['writeResult']);
});

test('an owned-environment cleanup hook is required before base deployment', async () => {
  const { calls, operations } = harness();
  delete operations.cleanupEnvironment;

  const result = await orchestrateUpgrade({
    capabilities: capabilities(),
    operations,
    request: request(),
  });

  assert.equal(result.captureValidity, CAPTURE_VALIDITY.INFRASTRUCTURE_FAILURE);
  assert.equal(result.phase, PHASES.CONFIGURATION_CHECK);
  assert.match(result.error.message, /operations\.cleanupEnvironment must be a function/);
  assert.deepEqual(calls, ['writeResult']);
});

test('every injected lifecycle failure stops before the next operation', async (t) => {
  for (const [index, failingOperation] of REQUIRED_OPERATIONS.entries()) {
    await t.test(failingOperation, async () => {
      const { calls, operations } = harness();
      operations[failingOperation] = async () => {
        calls.push(failingOperation);
        throw new Error(`${failingOperation} failed`);
      };

      const result = await orchestrateUpgrade({
        capabilities: capabilities(),
        operations,
        removedResources: removedResources(),
        request: request(),
      });

      assert.equal(result.complete, false);
      assert.match(result.error.message, new RegExp(`${failingOperation} failed`));
      assert.deepEqual(calls, [...REQUIRED_OPERATIONS.slice(0, index + 1), 'writeResult']);
    });
  }
});
