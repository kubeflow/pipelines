const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const sharp = require('sharp');

const comparison = require('../generate-comparison');
const { capturePair, parseCli } = require('../smoke-test-runner');
const { generateMarkdownSummary, validateSummary } = require('../upload-to-pr');

function fixtureRoot(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-smoke-policy-'));
  t.after(() => fs.rmSync(root, { force: true, recursive: true }));
  return root;
}

function sha256(contents) {
  return crypto.createHash('sha256').update(contents).digest('hex');
}

async function writePng(filePath, color) {
  await sharp({
    create: {
      width: 10,
      height: 10,
      channels: 4,
      background: color,
    },
  })
    .png()
    .toFile(filePath);
}

function captureResult(filename, overrides = {}) {
  const match = /^(.*)-([1-9]\d*)x([1-9]\d*)\.png$/.exec(filename);
  assert.ok(match);
  return {
    captureValidity: 'valid',
    filename,
    page: match[1],
    required: true,
    semanticScenario: match[1],
    scenarioTitle: match[1],
    status: 'success',
    viewport: { width: Number(match[2]), height: Number(match[3]) },
    ...overrides,
  };
}

function writeCaptureManifest(directory, label, captureId, results, overrides = {}) {
  const now = Date.now();
  const normalized = results.map((result) => {
    if (result.status !== 'success' && result.status !== 'degraded') return result;
    const contents = fs.readFileSync(path.join(directory, result.filename));
    return {
      ...result,
      capturedAt: new Date(now).toISOString(),
      sha256: sha256(contents),
    };
  });
  const requiredIncomplete = normalized.some(
    (result) => result.required && result.status !== 'success',
  );
  const hasCaptured = normalized.some(
    (result) => result.status === 'success' || result.status === 'degraded',
  );
  const complete = !requiredIncomplete && hasCaptured;
  const revisionRole = path.basename(directory) === 'base' ? 'base' : 'head';
  const inputAttestation = (name, digest) => ({
    path: path.join(directory, `${name}.json`),
    schemaVersion: `ui-smoke-${name}/v1`,
    sha256: digest.repeat(64),
    sizeBytes: 1,
  });
  fs.writeFileSync(
    path.join(directory, 'manifest.json'),
    `${JSON.stringify(
      {
        schemaVersion: 2,
        captureId,
        label,
        browser: {
          engine: 'chromium',
          playwrightVersion: '1.55.0',
          version: 'test-chromium',
        },
        deterministicRendering: {
          colorScheme: 'light',
          locale: 'en-US',
          timezone: 'UTC',
        },
        startedAt: new Date(now - 2000).toISOString(),
        completedAt: new Date(now + 2000).toISOString(),
        fatalErrors: [],
        inputs: overrides.inputs || {
          revisionRole,
          seedManifest: inputAttestation('seed', revisionRole === 'base' ? 'a' : 'b'),
          semanticManifest: inputAttestation('semantic', 'c'),
          sourceProvenance: inputAttestation('source', 'd'),
        },
        scenarioContractSchemaVersion:
          overrides.scenarioContractSchemaVersion || 'ui-smoke-scenarios/v1',
        viewports: [
          ...new Map(
            normalized.map((result) => [
              `${result.viewport.width}x${result.viewport.height}`,
              result.viewport,
            ]),
          ).values(),
        ],
        results: normalized,
        complete,
        summary: { complete },
      },
      null,
      2,
    )}\n`,
  );
}

function options(root, overrides = {}) {
  return {
    diffThreshold: 0,
    failThreshold: 0,
    failThresholdRaw: '0',
    looksSameClusterSize: 8,
    looksSameTolerance: 2.3,
    mainDir: path.join(root, 'base'),
    outputDir: path.join(root, 'comparison'),
    prDir: path.join(root, 'head'),
    scenarioConfigPath: null,
    ...overrides,
  };
}

async function createPair(t, results) {
  const root = fixtureRoot(t);
  const baseDir = path.join(root, 'base');
  const headDir = path.join(root, 'head');
  fs.mkdirSync(baseDir);
  fs.mkdirSync(headDir);
  for (const result of results) {
    await Promise.all([
      writePng(path.join(baseDir, result.base.filename), result.baseColor || '#111111'),
      writePng(path.join(headDir, result.head.filename), result.headColor || '#eeeeee'),
    ]);
  }
  writeCaptureManifest(
    baseDir,
    'base-v1',
    'base-capture',
    results.map((result) => result.base),
  );
  writeCaptureManifest(
    headDir,
    'head-v2',
    'head-capture',
    results.map((result) => result.head),
  );
  return root;
}

function writeScenarioConfig(root, scenarios, revisionPair = null) {
  const configPath = path.join(root, 'scenario-config.json');
  fs.writeFileSync(
    configPath,
    `${JSON.stringify(
      {
        schemaVersion: comparison.SCENARIO_CONFIG_SCHEMA_VERSION,
        revisionPair: revisionPair || {
          base: { label: 'base-v1' },
          head: { label: 'head-v2' },
        },
        scenarios,
      },
      null,
      2,
    )}\n`,
  );
  return configPath;
}

test('scenario policy binds revisions and emits five attested managed artifacts', async (t) => {
  const filename = 'executions-to-runs-10x10.png';
  const root = await createPair(t, [
    {
      base: captureResult(filename, {
        requestedRoute: '/#/executions',
        resolvedRoute: '/#/executions',
        routeExpectation: { kind: 'direct', path: '/executions' },
        scenarioTitle: 'Legacy Executions to Runs',
      }),
      head: captureResult(filename, {
        requestedRoute: '/#/executions',
        resolvedRoute: '/#/runs',
        routeExpectation: { kind: 'redirect', path: '/runs' },
        scenarioTitle: 'Legacy Executions to Runs',
      }),
    },
  ]);
  const configPath = writeScenarioConfig(root, [
    {
      semanticScenario: 'executions-to-runs',
      diffThreshold: 19,
      failThreshold: 20,
      looksSameTolerance: 0.75,
      expectedChange: '<removed> redirects to Runs',
      masks: [{ x: 0, y: 0, width: 5, height: 10, reason: 'clock' }],
    },
  ]);
  let compareOptions;
  const run = await comparison.runComparison(options(root, { scenarioConfigPath: configPath }), {
    looksSame: async (_base, _head, receivedOptions) => {
      compareOptions = receivedOptions;
      return {
        equal: false,
        differentPixels: 10,
        totalPixels: 100,
        diffBounds: { left: 5, top: 0, right: 5, bottom: 9 },
        diffClusters: [{ left: 5, top: 0, right: 5, bottom: 9 }],
      };
    },
  });

  assert.equal(run.exitCode, 0);
  assert.equal(compareOptions.tolerance, 0.75);
  const result = run.summary.results[0];
  assert.equal(result.semanticScenario, 'executions-to-runs');
  assert.deepEqual(result.routes, {
    base: {
      expectation: { kind: 'direct', path: '/executions' },
      requested: '/#/executions',
      resolved: '/#/executions',
    },
    head: {
      expectation: { kind: 'redirect', path: '/runs' },
      requested: '/#/executions',
      resolved: '/#/runs',
    },
  });
  assert.equal(result.diffPercent, 20);
  assert.equal(result.comparablePixels, 50);
  assert.equal(result.hasVisualDiff, true);
  assert.equal(result.exceedsFailThreshold, false);
  assert.equal(result.thresholdsEvaluated, true);
  assert.deepEqual(result.scenarioThresholds, {
    diffThreshold: 19,
    failThreshold: 20,
    looksSameTolerance: 0.75,
    source: 'scenario-config',
  });
  assert.deepEqual(result.masks, [{ x: 0, y: 0, width: 5, height: 10, reason: 'clock' }]);
  assert.equal(result.expectedChange, '<removed> redirects to Runs');
  assert.equal(run.summary.stats.thresholdEvaluations, 1);
  assert.equal(run.summary.stats.incompleteCaptures, 0);
  assert.equal(run.summary.scenarioConfig.sha256, sha256(fs.readFileSync(configPath)));

  const artifactFilenames = Object.values(result.artifacts).map((artifact) => artifact.filename);
  assert.equal(artifactFilenames.length, 5);
  for (const artifact of Object.values(result.artifacts)) {
    const artifactPath = path.join(options(root).outputDir, artifact.filename);
    const contents = fs.readFileSync(artifactPath);
    assert.equal(artifact.sha256, sha256(contents));
    assert.equal(artifact.sizeBytes, contents.length);
  }
  const marker = JSON.parse(
    fs.readFileSync(path.join(options(root).outputDir, '.managed-outputs.json'), 'utf8'),
  );
  assert.deepEqual(marker.filenames, [...artifactFilenames].sort());

  const report = fs.readFileSync(run.reportPath, 'utf8');
  assert.equal((report.match(/<img src="data:image\/png;base64,/g) || []).length, 5);
  assert.match(report, /Base route/);
  assert.match(report, /#\/executions/);
  assert.match(report, /#\/runs/);
  assert.match(report, /&lt;removed&gt; redirects to Runs/);
  assert.match(report, /x=0, y=0, width=5, height=10/);
  assert.match(report, /20\.0000% visual difference/);
  assert.doesNotMatch(report, /(?:href|src)="https?:/);

  assert.equal(validateSummary(run.summary), run.summary);
  const markdown = generateMarkdownSummary(run.summary, {
    prNumber: '13986',
    repo: 'kubeflow/pipelines',
  });
  assert.match(markdown, /executions-to-runs/);
  assert.match(markdown, /diff 19%; fail 20%; ΔE 0\.75/);
  assert.match(markdown, /valid/);
  assert.match(
    markdown,
    /base \/#\/executions → \/#\/executions; head \/#\/executions → \/#\/runs/,
  );
  const isolatedMarkdown = generateMarkdownSummary(
    {
      ...run.summary,
      mainLabel: `base: 2.17.1 (${'a'.repeat(40)}) [isolated full stack]`,
      prLabel: `HEAD (${'b'.repeat(40)}) [isolated full stack]`,
    },
    { prNumber: '13986', repo: 'kubeflow/pipelines' },
  );
  assert.match(isolatedMarkdown, /--compare 2\.17\.1 --full-stack/);
  assert.match(isolatedMarkdown, /--head-checkout \/path\/to\/reviewed\/head/);
  assert.match(isolatedMarkdown, /--viewports 10x10/);
  assert.match(isolatedMarkdown, /--diff-threshold 0 --fail-threshold 0/);
  assert.doesNotMatch(isolatedMarkdown, /--scenario-policy/);
  assert.match(isolatedMarkdown, /Local replay starter \(restore reviewed inputs first\)/);
  assert.match(isolatedMarkdown, /Bound Scenario Config SHA-256/);
  assert.doesNotMatch(isolatedMarkdown, /--pr 13986 --repo/);
});

test('scenario policy refuses a stale revision binding before comparison', async (t) => {
  const filename = 'runs-10x10.png';
  const root = await createPair(t, [
    { base: captureResult(filename), head: captureResult(filename) },
  ]);
  const configPath = writeScenarioConfig(root, [{ semanticScenario: 'runs' }], {
    base: { label: 'wrong-base' },
    head: { label: 'head-v2' },
  });
  const run = await comparison.runComparison(options(root, { scenarioConfigPath: configPath }));

  assert.equal(run.exitCode, 1);
  assert.match(run.summary.fatalErrors[0], /revisionPair\.base\.label does not match/);
  assert.equal(run.summary.stats.thresholdEvaluations, 0);
  assert.deepEqual(
    JSON.parse(fs.readFileSync(path.join(options(root).outputDir, '.managed-outputs.json'), 'utf8'))
      .filenames,
    [],
  );
});

test('comparison rejects swapped roles, provenance drift, and browser instability', async (t) => {
  const filename = 'runs-10x10.png';
  for (const scenario of [
    {
      name: 'swapped role',
      mutate: (manifest) => {
        manifest.inputs.revisionRole = 'head';
      },
      expected: /inputs\.revisionRole must be base/,
    },
    {
      name: 'semantic provenance drift',
      mutate: (manifest) => {
        manifest.inputs.semanticManifest.sha256 = 'e'.repeat(64);
      },
      expected: /different semanticManifest inputs/,
    },
    {
      name: 'browser version drift',
      mutate: (manifest) => {
        manifest.browser.version = 'different-chromium';
      },
      expected: /same browser contract/,
    },
    {
      name: 'rendering contract drift',
      mutate: (manifest) => {
        manifest.deterministicRendering.timezone = 'America\/New_York';
      },
      expected: /same deterministicRendering contract/,
    },
  ]) {
    await t.test(scenario.name, async () => {
      const root = await createPair(t, [
        { base: captureResult(filename), head: captureResult(filename) },
      ]);
      const manifestPath = path.join(
        root,
        scenario.name === 'swapped role' ? 'base' : 'head',
        'manifest.json',
      );
      const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
      scenario.mutate(manifest);
      fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
      const run = await comparison.runComparison(options(root));
      assert.equal(run.exitCode, 1);
      assert.match(run.summary.fatalErrors[0], scenario.expected);
      assert.equal(run.summary.stats.thresholdEvaluations, 0);
    });
  }
});

test('incomplete capture records cannot claim valid capture validity', async (t) => {
  const filename = 'runs-10x10.png';
  for (const status of ['degraded', 'failed']) {
    await t.test(status, async () => {
      const root = await createPair(t, [
        {
          base: captureResult(filename),
          head: captureResult(filename, { captureValidity: 'valid', status }),
        },
      ]);
      const run = await comparison.runComparison(options(root));
      assert.equal(run.exitCode, 1);
      assert.match(
        run.summary.fatalErrors[0],
        /Incomplete capture result runs .* cannot declare captureValidity valid/,
      );
      assert.equal(run.summary.stats.incompleteCaptures, 0);
      assert.equal(run.summary.stats.thresholdEvaluations, 0);
    });
  }
});

test('semantic captures require the current semantic scenario contract', async (t) => {
  const filename = 'runs-10x10.png';
  for (const contract of [null, 'ui-smoke-scenarios/obsolete']) {
    const root = await createPair(t, [
      { base: captureResult(filename), head: captureResult(filename) },
    ]);
    for (const role of ['base', 'head']) {
      const manifestPath = path.join(root, role, 'manifest.json');
      const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
      manifest.scenarioContractSchemaVersion = contract;
      fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
    }
    const run = await comparison.runComparison(options(root));
    assert.equal(run.exitCode, 1);
    assert.match(
      run.summary.fatalErrors[0],
      /Semantic captures must use scenario contract ui-smoke-scenarios\/v1/,
    );
  }
});

test('bound policy enforces required semantic catalog completeness and classification', async (t) => {
  const filename = 'runs-10x10.png';
  const catalog = [
    {
      semanticScenario: 'runs',
      scenarioTitle: 'Runs',
      expectedChange: null,
      required: true,
    },
    {
      semanticScenario: 'required-details',
      scenarioTitle: 'Required details',
      expectedChange: null,
      required: true,
    },
  ];
  const missingRoot = await createPair(t, [
    { base: captureResult(filename), head: captureResult(filename) },
  ]);
  assert.throws(
    () =>
      comparison.writeBoundScenarioConfig({
        baseDir: path.join(missingRoot, 'base'),
        defaults: { diffThreshold: 0, failThreshold: 0, looksSameTolerance: 2.3 },
        headDir: path.join(missingRoot, 'head'),
        outputPath: path.join(missingRoot, 'bound.json'),
        scenarioCatalog: catalog,
      }),
    /Required semantic scenario required-details is missing/,
  );
  assert.throws(
    () =>
      comparison.writeBoundScenarioConfig({
        baseDir: path.join(missingRoot, 'base'),
        defaults: { diffThreshold: 0, failThreshold: 0, looksSameTolerance: 2.3 },
        expectedViewports: '10x10,20x10',
        headDir: path.join(missingRoot, 'head'),
        outputPath: path.join(missingRoot, 'viewport-bound.json'),
        scenarioCatalog: [catalog[0]],
      }),
    /Required semantic scenario runs is missing viewport 20x10/,
  );

  const optionalRoot = await createPair(t, [
    {
      base: captureResult(filename, { required: false }),
      head: captureResult(filename, { required: false }),
    },
  ]);
  assert.throws(
    () =>
      comparison.writeBoundScenarioConfig({
        baseDir: path.join(optionalRoot, 'base'),
        defaults: { diffThreshold: 0, failThreshold: 0, looksSameTolerance: 2.3 },
        headDir: path.join(optionalRoot, 'head'),
        outputPath: path.join(optionalRoot, 'bound.json'),
        scenarioCatalog: [catalog[0]],
      }),
    /must remain required in both capture manifests/,
  );

  const policyRoot = missingRoot;
  const writePolicy = (name, scenarios) => {
    const policyPath = path.join(policyRoot, name);
    fs.writeFileSync(
      policyPath,
      `${JSON.stringify({
        schemaVersion: comparison.SCENARIO_POLICY_SCHEMA_VERSION,
        scenarios,
      })}\n`,
    );
    return policyPath;
  };
  const defaults = { diffThreshold: 0, failThreshold: 0, looksSameTolerance: 2.3 };
  const catalogWithAnnotation = [{ ...catalog[0], expectedChange: 'Catalog annotation' }];
  assert.throws(
    () =>
      comparison.writeBoundScenarioConfig({
        baseDir: path.join(policyRoot, 'base'),
        defaults,
        headDir: path.join(policyRoot, 'head'),
        outputPath: path.join(policyRoot, 'duplicate-bound.json'),
        policyPath: writePolicy('duplicate-policy.json', [
          { semanticScenario: 'runs' },
          { semanticScenario: 'runs' },
        ]),
        scenarioCatalog: catalogWithAnnotation,
      }),
    /duplicate rule runs@\*/,
  );
  assert.throws(
    () =>
      comparison.writeBoundScenarioConfig({
        baseDir: path.join(policyRoot, 'base'),
        defaults,
        headDir: path.join(policyRoot, 'head'),
        outputPath: path.join(policyRoot, 'bad-mask-bound.json'),
        policyPath: writePolicy('bad-mask-policy.json', [
          { semanticScenario: 'runs', masks: 'not-an-array' },
        ]),
        scenarioCatalog: catalogWithAnnotation,
      }),
    /masks must be an array/,
  );
  const cleared = comparison.writeBoundScenarioConfig({
    baseDir: path.join(policyRoot, 'base'),
    defaults,
    headDir: path.join(policyRoot, 'head'),
    outputPath: path.join(policyRoot, 'cleared-bound.json'),
    policyPath: writePolicy('clear-policy.json', [
      { semanticScenario: 'runs', expectedChange: null },
    ]),
    scenarioCatalog: catalogWithAnnotation,
  });
  assert.equal(cleared.config.scenarios[0].expectedChange, null);
});

test('expected removals are successful analyzed pairs with an explicitly disabled fail threshold', async (t) => {
  const removalFilename = 'executions-to-runs-10x10.png';
  const normalFilename = 'runs-10x10.png';
  const root = await createPair(t, [
    {
      base: captureResult(removalFilename, {
        expectedChange: 'Executions redirects to Runs',
        requestedRoute: '/#/executions',
        resolvedRoute: '/#/executions',
      }),
      head: captureResult(removalFilename, {
        captureValidity: 'expected_product_removal',
        expectedChange: 'Executions redirects to Runs',
        requestedRoute: '/#/executions',
        resolvedRoute: '/runs',
        routeExpectation: { kind: 'expected-removal', path: '/runs' },
      }),
    },
    {
      base: captureResult(normalFilename),
      head: captureResult(normalFilename),
      baseColor: '#224466',
      headColor: '#224466',
    },
  ]);
  let analysisCalls = 0;
  const run = await comparison.runComparison(options(root, { failThreshold: 1 }), {
    looksSame: async () => {
      analysisCalls += 1;
      if (analysisCalls === 1) {
        return {
          equal: false,
          differentPixels: 15,
          totalPixels: 100,
          diffBounds: { left: 0, top: 0, right: 1, bottom: 9 },
          diffClusters: [{ left: 0, top: 0, right: 1, bottom: 9 }],
        };
      }
      return {
        equal: true,
        differentPixels: 0,
        totalPixels: 100,
        diffBounds: null,
        diffClusters: [],
      };
    },
  });

  assert.equal(run.exitCode, 0);
  assert.equal(analysisCalls, 2);
  const removal = run.summary.results.find(
    (result) => result.semanticScenario === 'executions-to-runs',
  );
  assert.equal(removal.status, 'success');
  assert.equal(removal.captureValidity, 'expected_product_removal');
  assert.equal(removal.comparisonValidity, 'expected-change');
  assert.equal(removal.thresholdsEvaluated, true);
  assert.equal(removal.diffPercent, 15);
  assert.equal(removal.hasVisualDiff, true);
  assert.equal(removal.exceedsFailThreshold, false);
  assert.equal(removal.scenarioThresholds.failThreshold, null);
  assert.equal(Object.keys(removal.artifacts).length, 5);
  assert.equal(run.summary.stats.expectedProductRemovals, 1);
  assert.equal(run.summary.stats.incompleteCaptures, 0);
  assert.equal(run.summary.stats.thresholdEvaluations, 2);
  const report = fs.readFileSync(run.reportPath, 'utf8');
  assert.equal((report.match(/<img src="data:image\/png;base64,/g) || []).length, 10);
  assert.match(report, /failure threshold disabled/);
  assert.match(report, /Executions redirects to Runs/);
  validateSummary(run.summary);

  const reviewedConfig = writeScenarioConfig(root, [
    {
      semanticScenario: 'executions-to-runs',
      diffThreshold: 0,
      failThreshold: 10,
      looksSameTolerance: 2.3,
      masks: [],
    },
    {
      semanticScenario: 'runs',
      diffThreshold: 0,
      failThreshold: 1,
      looksSameTolerance: 2.3,
      masks: [],
    },
  ]);
  let reviewedAnalysisCalls = 0;
  const reviewedRun = await comparison.runComparison(
    options(root, {
      outputDir: path.join(root, 'comparison-reviewed-threshold'),
      scenarioConfigPath: reviewedConfig,
    }),
    {
      looksSame: async () => {
        reviewedAnalysisCalls += 1;
        return reviewedAnalysisCalls === 1
          ? {
              equal: false,
              differentPixels: 15,
              totalPixels: 100,
              diffBounds: { left: 0, top: 0, right: 1, bottom: 9 },
              diffClusters: [{ left: 0, top: 0, right: 1, bottom: 9 }],
            }
          : {
              equal: true,
              differentPixels: 0,
              totalPixels: 100,
              diffBounds: null,
              diffClusters: [],
            };
      },
    },
  );
  const reviewedRemoval = reviewedRun.summary.results.find(
    (result) => result.semanticScenario === 'executions-to-runs',
  );
  assert.equal(reviewedRun.exitCode, 1);
  assert.equal(reviewedRemoval.status, 'success');
  assert.equal(reviewedRemoval.exceedsFailThreshold, true);
  assert.equal(reviewedRemoval.scenarioThresholds.failThreshold, 10);
  validateSummary(reviewedRun.summary);
});

test('expected-removal claims fail closed unless a successful head proves the resolved route', async (t) => {
  const filename = 'executions-to-runs-10x10.png';
  const cases = [
    {
      name: 'base-side claim',
      base: {
        captureValidity: 'expected_product_removal',
        resolvedRoute: '/runs',
        routeExpectation: { kind: 'expected-removal', path: '/runs' },
      },
      head: {},
      expected: /may only be asserted by the head capture/,
    },
    {
      name: 'unsuccessful head claim',
      base: {},
      head: {
        captureValidity: 'expected_product_removal',
        diagnostics: {
          consoleErrors: [
            { kind: 'console', message: 'Authorization: Bearer secret-value', url: null },
          ],
          failedRequests: [],
          dropped: { consoleErrors: 0, failedRequests: 0 },
        },
        reason: 'control missing token=secret-value',
        resolvedRoute: '/runs',
        routeExpectation: { kind: 'expected-removal', path: '/runs' },
        status: 'skipped',
      },
      expected: /must have a successful screenshot/,
    },
    {
      name: 'unverified route kind',
      base: {},
      head: {
        captureValidity: 'expected_product_removal',
        resolvedRoute: '/runs',
        routeExpectation: { kind: 'redirect', path: '/runs' },
      },
      expected: /routeExpectation\.kind expected-removal/,
    },
    {
      name: 'wrong resolved route',
      base: {},
      head: {
        captureValidity: 'expected_product_removal',
        resolvedRoute: '/artifacts',
        routeExpectation: { kind: 'expected-removal', path: '/runs' },
      },
      expected: /resolved route \/artifacts does not match \/runs/,
    },
  ];

  for (const scenario of cases) {
    await t.test(scenario.name, async () => {
      const root = await createPair(t, [
        {
          base: captureResult(filename, scenario.base),
          head: captureResult(filename, scenario.head),
        },
      ]);
      let analysisCalls = 0;
      const run = await comparison.runComparison(options(root), {
        looksSame: async () => {
          analysisCalls++;
          return { equal: true, differentPixels: 0, totalPixels: 100 };
        },
      });
      assert.equal(run.exitCode, 1);
      assert.equal(analysisCalls, 0);
      assert.equal(run.summary.results[0].status, 'failed');
      assert.equal(run.summary.results[0].captureValidity, 'ui_rendering_failure');
      assert.match(run.summary.results[0].error, scenario.expected);
      assert.equal(run.summary.results[0].thresholdsEvaluated, false);
      assert.equal(run.summary.results[0].artifacts, undefined);
      if (scenario.name === 'unsuccessful head claim') {
        const evidence = run.summary.results[0].captureEvidenceByRevision.head;
        assert.doesNotMatch(JSON.stringify(evidence), /secret-value/);
        assert.match(JSON.stringify(evidence), /redacted/);
        const report = fs.readFileSync(run.reportPath, 'utf8');
        assert.match(report, /Head capture evidence/);
        assert.doesNotMatch(report, /secret-value/);
      }
    });
  }
});

test('degraded captures and invalid masks remain separate from pixel failures', async (t) => {
  const validFilename = 'runs-10x10.png';
  const degradedFilename = 'run-details-10x10.png';
  const root = await createPair(t, [
    {
      base: captureResult(validFilename),
      head: captureResult(validFilename),
      baseColor: '#123456',
      headColor: '#123456',
    },
    {
      base: captureResult(degradedFilename, { required: false }),
      head: captureResult(degradedFilename, {
        captureValidity: 'selector_drift',
        diagnostics: {
          consoleErrors: [{ kind: 'console', message: 'selector failed', url: '/runs' }],
          failedRequests: [
            {
              error: 'HTTP 500',
              method: 'GET',
              resourceType: 'fetch',
              sameOrigin: true,
              status: 500,
              url: '/apis/v2beta1/runs',
            },
          ],
          dropped: { consoleErrors: 0, failedRequests: 0 },
        },
        error: 'Task panel selector drifted',
        required: false,
        status: 'degraded',
      }),
    },
  ]);
  const configPath = writeScenarioConfig(root, [
    {
      semanticScenario: 'runs',
      masks: [{ x: 9, y: 9, width: 2, height: 2 }],
    },
  ]);
  const run = await comparison.runComparison(options(root, { scenarioConfigPath: configPath }), {
    looksSame: async () => ({
      equal: true,
      differentPixels: 0,
      totalPixels: 100,
      diffBounds: null,
      diffClusters: [],
    }),
  });

  assert.equal(run.exitCode, 1);
  assert.equal(run.summary.stats.thresholdEvaluations, 0);
  assert.equal(run.summary.stats.degradedCaptures, 1);
  assert.equal(run.summary.stats.incompleteCaptures, 1);
  assert.equal(run.summary.stats.pixelComparisonFailures, 0);
  const maskFailure = run.summary.results.find((result) => result.semanticScenario === 'runs');
  assert.equal(maskFailure.captureValidity, 'valid');
  assert.equal(maskFailure.comparisonValidity, 'config-invalid');
  assert.equal(maskFailure.failureType, 'config');
  assert.equal(maskFailure.thresholdsEvaluated, false);
  assert.equal(maskFailure.artifacts, undefined);
  const degraded = run.summary.results.find((result) => result.semanticScenario === 'run-details');
  assert.equal(degraded.captureValidity, 'selector_drift');
  assert.equal(degraded.comparisonValidity, 'not-compared');
  assert.equal(degraded.captureEvidenceByRevision.head.error, 'Task panel selector drifted');
  assert.equal(degraded.captureEvidenceByRevision.head.diagnostics.failedRequests[0].status, 500);
  const report = fs.readFileSync(run.reportPath, 'utf8');
  assert.match(report, /Task panel selector drifted/);
  assert.match(report, /apis\/v2beta1\/runs/);
});

test('capturePair binds reviewed policy and viewport rules to the exact capture pair', async (t) => {
  const root = fixtureRoot(t);
  const screenshotsDir = path.join(root, 'screenshots');
  fs.mkdirSync(screenshotsDir);
  const policyPath = path.join(root, 'reviewed-policy.json');
  fs.writeFileSync(
    policyPath,
    `${JSON.stringify({
      schemaVersion: comparison.SCENARIO_POLICY_SCHEMA_VERSION,
      scenarios: [
        {
          semanticScenario: 'runs',
          diffThreshold: 4,
          failThreshold: 9,
          looksSameTolerance: 0.4,
          expectedChange: 'Reviewed Runs evolution',
          masks: [{ x: 0, y: 0, width: 1, height: 1 }],
        },
        {
          semanticScenario: 'runs',
          viewport: { width: 10, height: 10 },
          masks: [{ x: 0, y: 0, width: 2, height: 2, reason: 'viewport clock' }],
        },
      ],
    })}\n`,
  );
  const childCalls = [];
  const runChildImpl = async (_command, args) => {
    childCalls.push(args);
    if (args[0].endsWith('capture-screenshots.js')) {
      const outputDir = args[args.indexOf('--output') + 1];
      const label = args[args.indexOf('--label') + 1];
      const revisionRole = args[args.indexOf('--revision-role') + 1];
      fs.mkdirSync(outputDir);
      const filename = 'runs-10x10.png';
      await writePng(path.join(outputDir, filename), '#123456');
      writeCaptureManifest(outputDir, label, `${revisionRole}-capture`, [
        captureResult(filename, { revisionRole }),
      ]);
      return { success: true };
    }
    return { success: true };
  };

  const result = await capturePair({
    baseUrl: 'http://127.0.0.1:3101',
    headUrl: 'http://127.0.0.1:3201',
    screenshotsDir,
    labels: { base: 'base-v1', head: 'head-v2' },
    options: {
      diffThreshold: 1,
      failThreshold: 2,
      scenarioPolicyPath: policyPath,
      viewports: '10x10',
    },
    baseSeedManifestPath: path.join(root, 'base-seed.json'),
    headSeedManifestPath: path.join(root, 'head-seed.json'),
    runChildImpl,
    scenarioCatalog: [
      {
        semanticScenario: 'runs',
        scenarioTitle: 'Runs',
        expectedChange: null,
        required: true,
      },
    ],
  });

  assert.equal(result.scenarioConfigPath, path.join(screenshotsDir, 'scenario-config.json'));
  const configContents = fs.readFileSync(result.scenarioConfigPath);
  const config = JSON.parse(configContents);
  assert.equal(config.schemaVersion, comparison.SCENARIO_CONFIG_SCHEMA_VERSION);
  assert.equal(config.revisionPair.base.captureId, 'base-capture');
  assert.equal(config.revisionPair.head.captureId, 'head-capture');
  assert.equal(config.revisionPair.base.manifestSha256.length, 64);
  assert.equal(config.revisionPair.head.manifestSha256.length, 64);
  assert.deepEqual(config.operatorPolicy, {
    applied: true,
    schemaVersion: comparison.SCENARIO_POLICY_SCHEMA_VERSION,
    sha256: sha256(fs.readFileSync(policyPath)),
    sizeBytes: fs.statSync(policyPath).size,
  });
  assert.deepEqual(config.scenarios, [
    {
      semanticScenario: 'runs',
      viewport: null,
      diffThreshold: 4,
      failThreshold: 9,
      looksSameTolerance: 0.4,
      masks: [{ x: 0, y: 0, width: 1, height: 1 }],
      expectedChange: 'Reviewed Runs evolution',
    },
    {
      semanticScenario: 'runs',
      viewport: { width: 10, height: 10 },
      masks: [{ x: 0, y: 0, width: 2, height: 2, reason: 'viewport clock' }],
    },
  ]);
  const comparisonArgs = childCalls.at(-1);
  assert.equal(comparisonArgs[0].endsWith('generate-comparison.js'), true);
  assert.equal(
    comparisonArgs[comparisonArgs.indexOf('--scenario-config') + 1],
    result.scenarioConfigPath,
  );

  let compareOptions;
  const comparisonRun = await comparison.runComparison(
    options(screenshotsDir, {
      failThreshold: null,
      failThresholdRaw: '',
      scenarioConfigPath: result.scenarioConfigPath,
    }),
    {
      looksSame: async (_base, _head, receivedOptions) => {
        compareOptions = receivedOptions;
        return {
          equal: false,
          differentPixels: 10,
          totalPixels: 100,
          diffBounds: { left: 2, top: 0, right: 2, bottom: 9 },
          diffClusters: [{ left: 2, top: 0, right: 2, bottom: 9 }],
        };
      },
    },
  );
  assert.equal(comparisonRun.exitCode, 1);
  assert.equal(compareOptions.tolerance, 0.4);
  const compared = comparisonRun.summary.results[0];
  assert.equal(compared.diffPercent, (10 / 96) * 100);
  assert.equal(compared.exceedsFailThreshold, true);
  assert.equal(compared.scenarioThresholds.failThreshold, 9);
  assert.deepEqual(compared.masks, [{ x: 0, y: 0, width: 2, height: 2, reason: 'viewport clock' }]);
  assert.equal(compared.expectedChange, 'Reviewed Runs evolution');
  assert.equal(comparisonRun.summary.stats.pagesExceedingFailThreshold, 1);
  assert.equal(validateSummary(comparisonRun.summary), comparisonRun.summary);
  const replay = generateMarkdownSummary(comparisonRun.summary, {
    prNumber: '13986',
    repo: 'kubeflow/pipelines',
  });
  assert.match(replay, /--scenario-policy \/path\/to\/reviewed\/scenario-policy\.json/);
  assert.match(
    replay,
    new RegExp(`Reviewed Scenario Policy SHA-256.*${config.operatorPolicy.sha256}`),
  );
});

test('parseCli scopes reviewed scenario policy input to comparison workflows', () => {
  const policyPath = path.resolve('reviewed-policy.json');
  assert.equal(
    parseCli(['--compare', 'origin/master', '--scenario-policy', 'reviewed-policy.json'])
      .scenarioPolicyPath,
    policyPath,
  );
  assert.throws(
    () =>
      parseCli([
        '--current-only',
        '--use-existing',
        '--url',
        'http://127.0.0.1:3000',
        '--scenario-policy',
        'reviewed-policy.json',
      ]),
    /--scenario-policy is only valid with --compare/,
  );
});
