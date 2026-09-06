#!/usr/bin/env node
/**
 * Generate a UI smoke-test summary and post it to a GitHub pull request.
 */

const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const COMMENT_MARKER_PREFIX = 'kubeflow-pipelines-ui-smoke-test';
const MAX_SUMMARY_BYTES = 10 * 1024 * 1024;
const COMMENTS_PER_PAGE = 100;
const MAX_COMMENT_PAGES = 1000;
// One API page can contain 100 full comment bodies. Allow worst-case UTF-8 and
// JSON expansion while keeping child-process output bounded.
const GH_MAX_BUFFER_BYTES = 64 * 1024 * 1024;
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

function parseArgumentValues(argv, defaults = {}) {
  const values = { ...defaults };
  const seen = new Set();
  const knownArguments = new Set(['pr', 'repo', 'screenshots']);

  for (let index = 0; index < argv.length; index++) {
    const argument = argv[index];
    if (!argument.startsWith('--')) {
      throw new Error(`Unexpected argument: ${argument}`);
    }
    const name = argument.slice(2);
    if (!knownArguments.has(name)) {
      throw new Error(`Unknown argument: --${name}`);
    }
    if (seen.has(name)) {
      throw new Error(`Argument --${name} may only be provided once`);
    }

    const value = argv[index + 1];
    if (value === undefined || value.startsWith('--')) {
      throw new Error(`Argument --${name} requires a value`);
    }
    values[name] = value;
    seen.add(name);
    index++;
  }
  return values;
}

function validatePrNumber(value) {
  const prNumber = String(value);
  if (!/^[1-9]\d*$/.test(prNumber)) {
    throw new Error(`Invalid PR number: ${prNumber || '(empty)'}`);
  }
  return prNumber;
}

function validateRepository(value) {
  const repository = String(value);
  const parts = repository.split('/');
  if (parts.length !== 2) {
    throw new Error(`Repository must use owner/name format: ${repository}`);
  }
  const [owner, name] = parts;
  const validOwner = /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$/.test(owner);
  const validName = /^[A-Za-z0-9._-]{1,100}$/.test(name) && name !== '.' && name !== '..';
  if (!validOwner || !validName) {
    throw new Error(`Invalid GitHub repository: ${repository}`);
  }
  return repository;
}

function isPathInside(rootPath, candidatePath) {
  const relative = path.relative(rootPath, candidatePath);
  return (
    relative === '' ||
    (!relative.startsWith(`..${path.sep}`) && relative !== '..' && !path.isAbsolute(relative))
  );
}

function validateScreenshotsDirectory(value, cwd = process.cwd()) {
  const candidate = path.resolve(cwd, value);
  let directoryStat;
  try {
    directoryStat = fs.statSync(candidate);
  } catch (_error) {
    throw new Error(`Screenshots directory does not exist: ${candidate}`);
  }
  if (!directoryStat.isDirectory()) {
    throw new Error(`Screenshots path is not a directory: ${candidate}`);
  }

  const directory = fs.realpathSync(candidate);
  const summaryCandidate = path.join(directory, 'summary.json');
  let summaryPath;
  let summaryStat;
  try {
    summaryPath = fs.realpathSync(summaryCandidate);
    summaryStat = fs.statSync(summaryPath);
  } catch (_error) {
    throw new Error(`Summary file does not exist: ${summaryCandidate}`);
  }
  if (!isPathInside(directory, summaryPath) || !summaryStat.isFile()) {
    throw new Error('summary.json must be a regular file inside the screenshots directory');
  }
  if (summaryStat.size === 0 || summaryStat.size > MAX_SUMMARY_BYTES) {
    throw new Error(`summary.json must be between 1 and ${MAX_SUMMARY_BYTES} bytes`);
  }

  return { directory, summaryPath };
}

function parseCliArgs(argv, options = {}) {
  const values = parseArgumentValues(argv, {
    repo: 'kubeflow/pipelines',
    screenshots: './screenshots/comparison',
  });
  if (!values.pr) {
    throw new Error('--pr <number> is required');
  }

  const paths = validateScreenshotsDirectory(values.screenshots, options.cwd);
  return {
    ...paths,
    prNumber: validatePrNumber(values.pr),
    repo: validateRepository(values.repo),
  };
}

function requireObject(value, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must be an object`);
  }
}

function requireNonNegativeInteger(value, label) {
  if (!Number.isInteger(value) || value < 0) {
    throw new Error(`${label} must be a non-negative integer`);
  }
}

function requireThreshold(value, label, allowNull = false) {
  if (allowNull && value === null) return;
  if (!Number.isFinite(value) || value < 0 || value > 100) {
    throw new Error(`${label} must be a number from 0 through 100${allowNull ? ' or null' : ''}`);
  }
}

function validateSummary(summary) {
  requireObject(summary, 'summary');
  if (summary.schemaVersion !== 2) {
    throw new Error('summary.schemaVersion must be 2');
  }
  if (typeof summary.timestamp !== 'string' || Number.isNaN(Date.parse(summary.timestamp))) {
    throw new Error('summary.timestamp must be a valid timestamp');
  }
  if (
    !Array.isArray(summary.fatalErrors) ||
    summary.fatalErrors.some((error) => typeof error !== 'string')
  ) {
    throw new Error('summary.fatalErrors must be an array of strings');
  }
  if (typeof summary.valid !== 'boolean' || typeof summary.passed !== 'boolean') {
    throw new Error('summary.valid and summary.passed must be booleans');
  }
  for (const name of ['mainLabel', 'prLabel']) {
    if (
      typeof summary[name] !== 'string' ||
      summary[name].trim() === '' ||
      summary[name].length > 1000
    ) {
      throw new Error(`summary.${name} must be a non-empty string of at most 1000 characters`);
    }
  }

  requireObject(summary.analysis, 'summary.analysis');
  requireThreshold(summary.analysis.looksSameTolerance, 'summary.analysis.looksSameTolerance');
  if (
    !Number.isSafeInteger(summary.analysis.looksSameClusterSize) ||
    summary.analysis.looksSameClusterSize < 1 ||
    summary.analysis.looksSameClusterSize > 1000
  ) {
    throw new Error('summary.analysis.looksSameClusterSize must be an integer from 1 through 1000');
  }

  requireObject(summary.thresholds, 'summary.thresholds');
  requireThreshold(summary.thresholds.diffThreshold, 'summary.thresholds.diffThreshold');
  requireThreshold(summary.thresholds.failThreshold, 'summary.thresholds.failThreshold', true);

  if (summary.scenarioConfig !== undefined && summary.scenarioConfig !== null) {
    requireObject(summary.scenarioConfig, 'summary.scenarioConfig');
    const operatorPolicy = summary.scenarioConfig.operatorPolicy;
    if (operatorPolicy !== undefined && operatorPolicy !== null) {
      requireObject(operatorPolicy, 'summary.scenarioConfig.operatorPolicy');
      if (typeof operatorPolicy.applied !== 'boolean') {
        throw new Error('summary.scenarioConfig.operatorPolicy.applied must be a boolean');
      }
      if (operatorPolicy.applied) {
        if (
          operatorPolicy.schemaVersion !== 'ui-smoke-comparison-policy/v1' ||
          typeof operatorPolicy.sha256 !== 'string' ||
          !/^[a-f0-9]{64}$/.test(operatorPolicy.sha256) ||
          !Number.isSafeInteger(operatorPolicy.sizeBytes) ||
          operatorPolicy.sizeBytes < 1
        ) {
          throw new Error('summary.scenarioConfig.operatorPolicy attestation is invalid');
        }
      }
    }
  }

  requireObject(summary.stats, 'summary.stats');
  for (const name of [
    'total',
    'success',
    'failed',
    'skipped',
    'pagesWithDiff',
    'pagesExceedingFailThreshold',
  ]) {
    requireNonNegativeInteger(summary.stats[name], `summary.stats.${name}`);
  }
  for (const name of [
    'missing',
    'stale',
    'corrupt',
    'integrity',
    'dimensionMismatch',
    'analysisFailed',
  ]) {
    requireNonNegativeInteger(summary.stats[name], `summary.stats.${name}`);
  }

  if (!Array.isArray(summary.results)) {
    throw new Error('summary.results must be an array');
  }
  if (summary.stats.total !== summary.results.length) {
    throw new Error('summary.stats.total must equal summary.results.length');
  }

  summary.results.forEach((result, index) => {
    const label = `summary.results[${index}]`;
    requireObject(result, label);
    if (typeof result.page !== 'string' || result.page.trim() === '') {
      throw new Error(`${label}.page must be a non-empty string`);
    }
    if (!['success', 'failed', 'skipped'].includes(result.status)) {
      throw new Error(`${label}.status must be success, failed, or skipped`);
    }
    if (result.status === 'success') {
      requireThreshold(result.diffPercent, `${label}.diffPercent`);
      if (typeof result.hasVisualDiff !== 'boolean') {
        throw new Error(`${label}.hasVisualDiff must be a boolean for a successful comparison`);
      }
      if (typeof result.exceedsFailThreshold !== 'boolean') {
        throw new Error(
          `${label}.exceedsFailThreshold must be a boolean for a successful comparison`,
        );
      }
    } else if (result.diffPercent !== undefined && result.diffPercent !== null) {
      requireThreshold(result.diffPercent, `${label}.diffPercent`);
    }
    if (result.thresholdsEvaluated !== undefined) {
      if (typeof result.thresholdsEvaluated !== 'boolean') {
        throw new Error(`${label}.thresholdsEvaluated must be a boolean when provided`);
      }
      if (result.thresholdsEvaluated !== (result.status === 'success')) {
        throw new Error(
          `${label}.thresholdsEvaluated must be true only for successful comparisons`,
        );
      }
    }
    if (result.scenarioThresholds !== undefined) {
      requireObject(result.scenarioThresholds, `${label}.scenarioThresholds`);
      requireThreshold(
        result.scenarioThresholds.diffThreshold,
        `${label}.scenarioThresholds.diffThreshold`,
      );
      requireThreshold(
        result.scenarioThresholds.failThreshold,
        `${label}.scenarioThresholds.failThreshold`,
        true,
      );
      requireThreshold(
        result.scenarioThresholds.looksSameTolerance,
        `${label}.scenarioThresholds.looksSameTolerance`,
      );
      if (
        typeof result.scenarioThresholds.source !== 'string' ||
        result.scenarioThresholds.source.trim() === ''
      ) {
        throw new Error(`${label}.scenarioThresholds.source must be a non-empty string`);
      }
    }
    for (const name of [
      'semanticScenario',
      'scenarioTitle',
      'captureValidity',
      'comparisonValidity',
    ]) {
      if (result[name] !== undefined && (typeof result[name] !== 'string' || result[name] === '')) {
        throw new Error(`${label}.${name} must be a non-empty string when provided`);
      }
    }
    if (result.captureValidity !== undefined && !CAPTURE_VALIDITIES.has(result.captureValidity)) {
      throw new Error(`${label}.captureValidity is unsupported`);
    }
    if (result.captureValidityByRevision !== undefined) {
      requireObject(result.captureValidityByRevision, `${label}.captureValidityByRevision`);
      for (const role of ['base', 'head']) {
        if (!CAPTURE_VALIDITIES.has(result.captureValidityByRevision[role])) {
          throw new Error(`${label}.captureValidityByRevision.${role} is unsupported`);
        }
      }
    }
    if (result.captureEvidenceByRevision !== undefined) {
      requireObject(result.captureEvidenceByRevision, `${label}.captureEvidenceByRevision`);
      const serializedEvidence = JSON.stringify(result.captureEvidenceByRevision);
      if (serializedEvidence.length > 256_000) {
        throw new Error(`${label}.captureEvidenceByRevision is too large`);
      }
      for (const role of ['base', 'head']) {
        const evidence = result.captureEvidenceByRevision[role];
        if (evidence !== null) {
          requireObject(evidence, `${label}.captureEvidenceByRevision.${role}`);
          if (!['success', 'degraded', 'skipped', 'failed'].includes(evidence.status)) {
            throw new Error(`${label}.captureEvidenceByRevision.${role}.status is invalid`);
          }
        }
      }
    }
    for (const name of ['mainExists', 'prExists', 'hasVisualDiff', 'exceedsFailThreshold']) {
      if (result[name] !== undefined && typeof result[name] !== 'boolean') {
        throw new Error(`${label}.${name} must be a boolean when provided`);
      }
    }
    if (result.error !== undefined && typeof result.error !== 'string') {
      throw new Error(`${label}.error must be a string when provided`);
    }
    if (result.failureType !== undefined && typeof result.failureType !== 'string') {
      throw new Error(`${label}.failureType must be a string when provided`);
    }
    if (result.reason !== undefined && typeof result.reason !== 'string') {
      throw new Error(`${label}.reason must be a string when provided`);
    }
  });

  const successCount = summary.results.filter((result) => result.status === 'success').length;
  const failedCount = summary.results.filter((result) => result.status === 'failed').length;
  const skippedCount = summary.results.filter((result) => result.status === 'skipped').length;
  const successfulResults = summary.results.filter((result) => result.status === 'success');
  for (const result of successfulResults) {
    const resultThresholds = result.scenarioThresholds || summary.thresholds;
    const expectedVisualDiff = result.diffPercent > resultThresholds.diffThreshold;
    const expectedThresholdFailure =
      resultThresholds.failThreshold !== null &&
      result.diffPercent > resultThresholds.failThreshold;
    if (
      result.hasVisualDiff !== expectedVisualDiff ||
      result.exceedsFailThreshold !== expectedThresholdFailure
    ) {
      throw new Error(`summary result flags do not match thresholds for ${result.page}`);
    }
  }
  const expectedFailureStats = {
    analysisFailed: summary.results.filter((result) => result.failureType === 'analysis').length,
    corrupt: summary.results.filter((result) => result.failureType === 'corrupt').length,
    dimensionMismatch: summary.results.filter(
      (result) => result.failureType === 'dimension-mismatch',
    ).length,
    integrity: summary.results.filter((result) => result.failureType === 'integrity').length,
    missing: summary.results.filter((result) => result.failureType === 'missing').length,
    stale: summary.results.filter((result) => result.failureType === 'stale').length,
  };
  if (
    summary.stats.success !== successCount ||
    summary.stats.failed !== failedCount ||
    summary.stats.skipped !== skippedCount
  ) {
    throw new Error('summary success/failed/skipped statistics do not match summary.results');
  }
  for (const [name, expected] of Object.entries(expectedFailureStats)) {
    if (summary.stats[name] !== expected) {
      throw new Error(`summary.stats.${name} does not match summary.results`);
    }
  }
  const expectedPagesWithDiff = successfulResults.filter((result) => result.hasVisualDiff).length;
  const expectedPagesExceeding = successfulResults.filter(
    (result) => result.exceedsFailThreshold,
  ).length;
  if (
    summary.stats.pagesWithDiff !== expectedPagesWithDiff ||
    summary.stats.pagesExceedingFailThreshold !== expectedPagesExceeding
  ) {
    throw new Error('summary visual-difference statistics do not match summary.results');
  }
  const expectedValid = summary.fatalErrors.length === 0 && failedCount === 0 && successCount > 0;
  const expectedPassed = expectedValid && summary.stats.pagesExceedingFailThreshold === 0;
  if (summary.valid !== expectedValid || summary.passed !== expectedPassed) {
    throw new Error('summary valid/passed state is inconsistent with its results');
  }
  return summary;
}

function readSummary(summaryPath) {
  let summary;
  try {
    summary = JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
  } catch (error) {
    throw new Error(`Could not read summary.json: ${error.message}`);
  }
  return validateSummary(summary);
}

function commentMarker(repo, prNumber) {
  return `<!-- ${COMMENT_MARKER_PREFIX}:${repo}#${prNumber} -->`;
}

function escapeTableCell(value) {
  return String(value)
    .replace(/\|/g, '\\|')
    .replace(/[\r\n]+/g, ' ')
    .trim();
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function formatThreshold(value) {
  return value === null ? 'disabled' : `${value}%`;
}

function replayViewports(summary) {
  const configured = summary.scenarioConfig?.viewports;
  const candidates =
    Array.isArray(configured) && configured.length > 0
      ? configured
      : summary.results.map((result) => result.viewport).filter(Boolean);
  const keys = new Set();
  for (const viewport of candidates) {
    if (
      !viewport ||
      !Number.isSafeInteger(viewport.width) ||
      viewport.width < 1 ||
      !Number.isSafeInteger(viewport.height) ||
      viewport.height < 1
    ) {
      throw new Error('summary replay viewport metadata is invalid');
    }
    keys.add(`${viewport.width}x${viewport.height}`);
  }
  return [...keys].sort().join(',');
}

function generateMarkdownSummary(summary, options) {
  validateSummary(summary);
  const marker = commentMarker(options.repo, options.prNumber);
  const diffThreshold = summary.thresholds.diffThreshold;
  const failThreshold = summary.thresholds.failThreshold;
  const baseSha = /\(([0-9a-f]{40,64})\)/i.exec(summary.mainLabel)?.[1] || '<base-ref>';
  const browserOnlyFlag = summary.prLabel.includes('[browser-only;') ? ' --browser-only' : '';
  const isolatedFullStack =
    summary.mainLabel.includes('[isolated full stack]') ||
    summary.prLabel.includes('[isolated full stack]');
  const isolatedBaseRef =
    /^base:\s+([^\s(]+)/.exec(summary.mainLabel)?.[1] || '<exact-release-ref>';
  const capturedViewports = replayViewports(summary);
  const replayViewportFlag = capturedViewports
    ? ` --viewports ${capturedViewports}`
    : ' --viewports CAPTURED_WIDTHxHEIGHT';
  const replayThresholdFlags = ` --diff-threshold ${summary.thresholds.diffThreshold} --fail-threshold ${summary.thresholds.failThreshold}`;
  const operatorPolicy = summary.scenarioConfig?.operatorPolicy;
  const replayPolicyFlag = operatorPolicy?.applied
    ? ' --scenario-policy /path/to/reviewed/scenario-policy.json'
    : '';
  const boundConfigAttestation = summary.scenarioConfig?.sha256
    ? `\n- **Bound Scenario Config SHA-256:** \`${summary.scenarioConfig.sha256}\``
    : '';
  const operatorPolicyAttestation = operatorPolicy?.applied
    ? `\n- **Reviewed Scenario Policy SHA-256:** \`${operatorPolicy.sha256}\``
    : '';
  let markdown = `${marker}
## 🔍 UI Smoke Test Results

**Generated:** ${new Date(summary.timestamp).toISOString()}

**Overall:** ${summary.passed ? '✅ Passed' : summary.valid ? '❌ Visual threshold exceeded' : '❌ Invalid comparison'}

**Base:** <code>${escapeHtml(summary.mainLabel)}</code>

**Head:** <code>${escapeHtml(summary.prLabel)}</code>

### Summary

- **Pages Tested:** ${summary.stats.total}
- **Successful:** ${summary.stats.success}
- **Failed:** ${summary.stats.failed}
- **Skipped:** ${summary.stats.skipped}
- **Pages with Visual Changes:** ${summary.stats.pagesWithDiff}
- **Pages Above Failure Threshold:** ${summary.stats.pagesExceedingFailThreshold}
- **Default Diff Marker Threshold:** ${formatThreshold(diffThreshold)}
- **Default Failure Threshold:** ${formatThreshold(failThreshold)}
- **Looks-same Color Tolerance (ΔE):** ${summary.analysis.looksSameTolerance}
- **Looks-same Cluster Size:** ${summary.analysis.looksSameClusterSize}px
- **Captured Viewports:** ${capturedViewports || 'not recorded'}${boundConfigAttestation}${operatorPolicyAttestation}
`;

  if (summary.fatalErrors.length > 0) {
    markdown += `
### Fatal errors

${summary.fatalErrors.map((error) => `- ${escapeTableCell(error)}`).join('\n')}

`;
  }

  markdown += `
### Page Results

| Scenario | Capture validity | Status | Diff % | Thresholds | Revision routes | Notes |
|----------|------------------|--------|--------|------------|-----------------|-------|
`;

  for (const result of summary.results) {
    const diffIsFinite = Number.isFinite(result.diffPercent);
    const diff = diffIsFinite ? `${result.diffPercent.toFixed(2)}%` : 'N/A';
    const resultThresholds = result.scenarioThresholds || {
      diffThreshold,
      failThreshold,
      looksSameTolerance: summary.analysis.looksSameTolerance,
      source: 'global-defaults',
    };
    const notes = [];
    if (result.mainExists === false) notes.push('No base screenshot');
    if (result.prExists === false) notes.push('No PR screenshot');

    const hasVisualDiff =
      result.hasVisualDiff === true ||
      (diffIsFinite && result.diffPercent > resultThresholds.diffThreshold);
    if (hasVisualDiff)
      notes.push(`Visual change above ${resultThresholds.diffThreshold}% marker threshold`);

    const exceedsFailThreshold =
      result.exceedsFailThreshold === true ||
      (resultThresholds.failThreshold !== null &&
        diffIsFinite &&
        result.diffPercent > resultThresholds.failThreshold);
    if (exceedsFailThreshold)
      notes.push(`Above ${resultThresholds.failThreshold}% failure threshold`);
    if (result.status === 'failed') notes.push(result.error || 'Comparison failed');
    if (result.status === 'skipped')
      notes.push(result.reason || result.error || 'Comparison skipped');
    if (result.expectedChange) {
      notes.push(
        `Expected change: ${typeof result.expectedChange === 'string' ? result.expectedChange : JSON.stringify(result.expectedChange)}`,
      );
    }

    const status = result.status === 'success' ? '✅' : result.status === 'skipped' ? '⏭️' : '❌';
    const formatRoute = (route) => `${route?.requested || 'N/A'} → ${route?.resolved || 'N/A'}`;
    const routes = `base ${formatRoute(result.routes?.base)}; head ${formatRoute(result.routes?.head)}`;
    const thresholds = `diff ${formatThreshold(resultThresholds.diffThreshold)}; fail ${formatThreshold(resultThresholds.failThreshold)}; ΔE ${resultThresholds.looksSameTolerance}`;
    markdown += `| ${escapeTableCell(result.semanticScenario || result.page)} | ${escapeTableCell(result.captureValidity || (result.status === 'success' ? 'valid' : 'unknown'))} | ${status} | ${diff} | ${escapeTableCell(thresholds)} | ${escapeTableCell(routes)} | ${escapeTableCell(notes.join(', ') || '-')} |\n`;
  }

  const rerunCommand = isolatedFullStack
    ? `node smoke-test-runner.js --compare ${isolatedBaseRef} --full-stack --head-checkout /path/to/reviewed/head --trust-local-head --pr-number ${options.prNumber}${replayViewportFlag}${replayThresholdFlags}${replayPolicyFlag}`
    : `node smoke-test-runner.js --compare ${baseSha} --pr ${options.prNumber} --repo ${options.repo} --trust-pr-code${browserOnlyFlag}${replayViewportFlag}${replayThresholdFlags}${replayPolicyFlag}`;
  markdown += `
This comment contains the comparison summary only; images are not attached by this tool.

---
<details>
<summary>Local replay starter (restore reviewed inputs first)</summary>

The command preserves the recorded global thresholds and viewports. Replace any checkout${
    operatorPolicy?.applied ? ' and scenario-policy' : ''
  } placeholders with the reviewed inputs from this run. The generated bound scenario config
is capture-pair-specific; verify it against the SHA-256 attestation above.

\`\`\`bash
cd frontend/scripts/ui-smoke-test
npm ci
${rerunCommand}
\`\`\`

</details>
`;
  return markdown;
}

function runGh(args, options = {}, spawnSyncImpl = spawnSync) {
  if (!Array.isArray(args) || args.some((argument) => typeof argument !== 'string')) {
    throw new Error('gh arguments must be provided as a string array');
  }
  const result = spawnSyncImpl('gh', args, {
    encoding: 'utf8',
    input: options.input,
    maxBuffer: GH_MAX_BUFFER_BYTES,
  });
  const stdout = typeof result.stdout === 'string' ? result.stdout.trim() : '';
  const stderr = typeof result.stderr === 'string' ? result.stderr.trim() : '';
  if (result.error || result.status !== 0) {
    if (!options.silent) {
      console.error(`gh ${args.join(' ')} failed`);
      console.error(stderr || result.error?.message || `exit status ${result.status}`);
    }
    return { error: result.error, status: result.status, stderr, stdout, success: false };
  }
  return { status: result.status, stderr, stdout, success: true };
}

function getAuthenticatedLogin(runGhImpl = runGh) {
  const result = runGhImpl(['api', 'user'], { silent: true });
  if (!result.success) {
    throw new Error('Not authenticated with GitHub CLI. Run: gh auth login');
  }
  let user;
  try {
    user = JSON.parse(result.stdout);
  } catch (_error) {
    throw new Error('GitHub CLI returned an invalid authenticated-user response');
  }
  if (!user || typeof user.login !== 'string' || user.login === '') {
    throw new Error('GitHub CLI did not return the authenticated user login');
  }
  return user.login;
}

function findExistingComment(repo, prNumber, viewerLogin, marker, runGhImpl = runGh) {
  let existing = null;
  for (let page = 1; page <= MAX_COMMENT_PAGES; page++) {
    const endpoint = `repos/${repo}/issues/${prNumber}/comments?per_page=${COMMENTS_PER_PAGE}&page=${page}`;
    const result = runGhImpl(['api', endpoint], { silent: true });
    if (!result.success) {
      throw new Error(`Failed to list existing PR comments (page ${page})`);
    }

    let pageComments;
    try {
      pageComments = JSON.parse(result.stdout);
    } catch (_error) {
      throw new Error(`GitHub CLI returned invalid comments JSON on page ${page}`);
    }
    if (!Array.isArray(pageComments)) {
      throw new Error(`GitHub comments response on page ${page} was not an array`);
    }

    for (const comment of pageComments) {
      if (
        comment &&
        (typeof comment.id === 'number' || typeof comment.id === 'string') &&
        comment.user?.login === viewerLogin &&
        typeof comment.body === 'string' &&
        comment.body.split(/\r?\n/, 1)[0] === marker
      ) {
        existing = comment;
      }
    }

    if (pageComments.length < COMMENTS_PER_PAGE) {
      return existing;
    }
  }
  throw new Error(`PR comment pagination exceeded ${MAX_COMMENT_PAGES} pages`);
}

function postCommentToPR(markdown, options, runGhImpl = runGh) {
  const marker = commentMarker(options.repo, options.prNumber);
  const existing = findExistingComment(
    options.repo,
    options.prNumber,
    options.viewerLogin,
    marker,
    runGhImpl,
  );

  if (existing) {
    const commentId = String(existing.id);
    if (!/^\d+$/.test(commentId)) {
      throw new Error(`GitHub returned an invalid comment ID: ${commentId}`);
    }
    const result = runGhImpl(
      [
        'api',
        `repos/${options.repo}/issues/comments/${commentId}`,
        '--method',
        'PATCH',
        '--input',
        '-',
      ],
      { input: JSON.stringify({ body: markdown }) },
    );
    if (!result.success) {
      throw new Error(`Failed to update UI smoke-test comment ${commentId}`);
    }
    return { action: 'updated', commentId };
  }

  const result = runGhImpl(
    ['pr', 'comment', options.prNumber, '--repo', options.repo, '--body-file', '-'],
    { input: markdown },
  );
  if (!result.success) {
    throw new Error('Failed to create UI smoke-test comment');
  }
  return { action: 'created' };
}

function uploadResults(options, runGhImpl = runGh) {
  const version = runGhImpl(['--version'], { silent: true });
  if (!version.success) {
    throw new Error('GitHub CLI (gh) is not installed or not in PATH');
  }

  const viewerLogin = getAuthenticatedLogin(runGhImpl);
  const summary = readSummary(options.summaryPath);
  const markdown = generateMarkdownSummary(summary, options);
  return postCommentToPR(markdown, { ...options, viewerLogin }, runGhImpl);
}

function main(argv = process.argv.slice(2)) {
  try {
    const options = parseCliArgs(argv);
    console.log(`Uploading UI smoke test results to PR #${options.prNumber}`);
    console.log(`Repository: ${options.repo}`);
    console.log(`Screenshots directory: ${options.directory}`);

    const result = uploadResults(options);
    console.log(`✓ Comment ${result.action} successfully`);
    console.log(`Results posted to: https://github.com/${options.repo}/pull/${options.prNumber}`);
  } catch (error) {
    console.error(`Upload failed: ${error.message}`);
    process.exitCode = 1;
  }
}

if (require.main === module) {
  main();
}

module.exports = {
  COMMENT_MARKER_PREFIX,
  COMMENTS_PER_PAGE,
  GH_MAX_BUFFER_BYTES,
  MAX_COMMENT_PAGES,
  MAX_SUMMARY_BYTES,
  commentMarker,
  escapeTableCell,
  findExistingComment,
  formatThreshold,
  generateMarkdownSummary,
  getAuthenticatedLogin,
  isPathInside,
  main,
  parseCliArgs,
  postCommentToPR,
  readSummary,
  runGh,
  uploadResults,
  validatePrNumber,
  validateRepository,
  validateScreenshotsDirectory,
  validateSummary,
};
