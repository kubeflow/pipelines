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

const CAPTURE_MANIFEST_FILENAME = 'manifest.json';
const CAPTURE_MANIFEST_SCHEMA_VERSION = 2;
const COMPARISON_SUMMARY_FILENAME = 'summary.json';
const COMPARISON_SUMMARY_SCHEMA_VERSION = 2;
const COMPARISON_REPORT_FILENAME = 'report.html';
const MANAGED_OUTPUTS_FILENAME = '.managed-outputs.json';
const MANAGED_OUTPUTS_SCHEMA_VERSION = 2;
const MANAGED_STATIC_OUTPUTS = [COMPARISON_REPORT_FILENAME, COMPARISON_SUMMARY_FILENAME];
const FRESHNESS_TOLERANCE_MS = 1000;
const CAPTURE_STATUSES = new Set(['success', 'degraded', 'skipped', 'failed']);
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

function deriveDiffPercent(looksSameResult, width, height) {
  if (
    Number.isFinite(looksSameResult?.differentPixels) &&
    Number.isFinite(looksSameResult?.totalPixels) &&
    looksSameResult.totalPixels > 0
  ) {
    return (looksSameResult.differentPixels / looksSameResult.totalPixels) * 100;
  }

  if (Number.isFinite(looksSameResult?.diffPercentage)) {
    return looksSameResult.diffPercentage;
  }

  if (looksSameResult?.equal === true) {
    return 0;
  }

  if (looksSameResult?.diffBounds && width > 0 && height > 0) {
    const bounds = normalizeRegion(looksSameResult.diffBounds, width, height);
    if (bounds) {
      return ((bounds.width * bounds.height) / (width * height)) * 100;
    }
  }

  return null;
}

async function analyzeDiff(mainPath, prPath, options, compareImages = looksSame) {
  if (!fs.existsSync(mainPath) || !fs.existsSync(prPath)) {
    throw new ComparisonError('A required screenshot is missing.', 'missing');
  }

  let mainMeta;
  let prMeta;
  try {
    [mainMeta, prMeta] = await Promise.all([sharp(mainPath).metadata(), sharp(prPath).metadata()]);
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

  let looksSameResult;
  try {
    looksSameResult = await compareImages(mainPath, prPath, {
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

  const diffPercent = deriveDiffPercent(looksSameResult, mainWidth, mainHeight);
  if (!Number.isFinite(diffPercent) || diffPercent < 0 || diffPercent > 100) {
    throw new ComparisonError('Image analysis did not return a valid diff percentage.', 'analysis');
  }

  return {
    diffPercent,
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

  await sharp({
    create: {
      width: width * 2 + DIVIDER_WIDTH,
      height: totalHeight,
      channels: 4,
      background: { r: 255, g: 255, b: 255, alpha: 1 },
    },
  })
    .composite(composites)
    .png()
    .toFile(outputPath);

  console.log(`  ✓ Saved: ${path.basename(outputPath)}`);
  return outputPath;
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
    }

    records.set(result.filename, {
      ...result,
      capturedAtMs,
      completedAtMs,
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

function failedPlanResult(filename, required, message, failureType = 'capture') {
  return {
    filename,
    page: filename.replace(/\.png$/, ''),
    required,
    status: 'failed',
    error: message,
    failureType,
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
    const problems = [];

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
    } else if (
      !mainRecord ||
      !prRecord ||
      mainRecord.status !== 'success' ||
      prRecord.status !== 'success'
    ) {
      results.push({
        filename,
        page: filename.replace(/\.png$/, ''),
        required: false,
        status: 'skipped',
        reason: 'Optional capture was not successful in both revisions.',
      });
      continue;
    }

    if (problems.length > 0) {
      results.push(failedPlanResult(filename, required, problems.join('; '), 'capture'));
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
    });
  }

  return { filenames, pairs, results };
}

function buildComparisonPlan(mainDir, prDir) {
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

  return {
    ...buildManifestComparisonPlan(mainManifest, prManifest, mainDir, prDir),
    captures: {
      base: mainManifest.attestation,
      head: prManifest.attestation,
    },
    mainLabel: mainManifest.label || 'main (base)',
    prLabel: prManifest.label || 'PR (head)',
    sourceMode: 'manifest',
  };
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
  sourceMode,
}) {
  const orderedResults = [...results].sort((left, right) => {
    if (left.filename === right.filename) return 0;
    return left.filename < right.filename ? -1 : 1;
  });
  const failed = orderedResults.filter((result) => result.status === 'failed');
  const success = orderedResults.filter((result) => result.status === 'success');
  const pagesExceedingFailThreshold =
    options.failThreshold === null ? [] : success.filter((result) => result.exceedsFailThreshold);
  const valid = fatalErrors.length === 0 && failed.length === 0 && success.length > 0;
  const passed = valid && pagesExceedingFailThreshold.length === 0;

  return {
    schemaVersion: COMPARISON_SUMMARY_SCHEMA_VERSION,
    timestamp: new Date().toISOString(),
    mainLabel,
    prLabel,
    sourceMode,
    captures,
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

function renderImageFigure(caption, alt, contents) {
  const dataUrl = imageDataUrl(contents);
  return `<figure><figcaption>${escapeHtml(caption)}</figcaption><a href="${dataUrl}" title="Open the full-size embedded image"><img src="${dataUrl}" alt="${escapeHtml(alt)}" loading="lazy"></a></figure>`;
}

function renderComparisonResult(result, embeddedImages) {
  let statusDetail;
  if (result.status === 'success') {
    statusDetail = `${result.diffPercent.toFixed(4)}% visual difference; ${result.diffRegionCount} highlighted region(s); ${result.exceedsFailThreshold ? 'above' : 'within'} the failure threshold.`;
  } else if (result.status === 'skipped') {
    statusDetail = result.reason;
  } else {
    statusDetail = `${result.failureType || 'comparison'}: ${result.error}`;
  }

  let images = '';
  if (result.status === 'success') {
    const imageSet = embeddedImages.get(result.filename);
    if (!imageSet) {
      throw new Error(`Missing embedded report images for ${result.filename}.`);
    }
    images = `<div class="images">${renderImageFigure('Base', `Base capture for ${result.page}`, imageSet.base)}${renderImageFigure('Head', `Head capture for ${result.page}`, imageSet.head)}${renderImageFigure('Highlighted comparison', `Side-by-side highlighted comparison for ${result.page}`, imageSet.comparison)}</div>`;
  }

  return `<section class="result status-${escapeHtml(result.status)}"><h2>${escapeHtml(result.page)}</h2><p class="metadata"><span>Status: ${escapeHtml(result.status)}</span><span>Capture validity: ${result.status === 'success' ? 'valid' : 'invalid'}</span><span>${result.required ? 'Required' : 'Optional'}</span></p><p>${escapeHtml(statusDetail)}</p>${images}</section>`;
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

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'">
  <title>UI smoke comparison: ${escapeHtml(overallStatus)}</title>
  <style>
    :root{color-scheme:light;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f5f6fa;color:#171821}body{max-width:1600px;margin:0 auto;padding:24px}h1,h2{line-height:1.25}code{overflow-wrap:anywhere}.overview,.captures,.metadata,.images{display:grid;gap:12px}.overview{grid-template-columns:repeat(auto-fit,minmax(190px,1fr));margin:20px 0}.overview div,.capture,.result,.errors,.empty{background:#fff;border:1px solid #d9dce7;border-radius:8px;padding:16px}.overview strong,.metadata span{display:block}.captures{grid-template-columns:repeat(auto-fit,minmax(320px,1fr));margin-bottom:20px}.capture h2,.result h2{margin-top:0}.capture dl{display:grid;grid-template-columns:max-content 1fr;gap:6px 12px;margin:0}.capture dd{margin:0;min-width:0}.metadata{grid-template-columns:repeat(auto-fit,minmax(160px,max-content));color:#4b5063}.images{grid-template-columns:repeat(auto-fit,minmax(280px,1fr));align-items:start}figure{margin:0}figcaption{font-weight:650;margin-bottom:8px}img{display:block;width:100%;height:auto;border:1px solid #c8cbd7;background:#fff}a:focus{outline:3px solid #3157d5;outline-offset:3px}.status-failed{border-left:5px solid #b42318}.status-skipped{border-left:5px solid #b7791f}.status-success{border-left:5px solid #16803c}.errors{border-left:5px solid #b42318}.errors li+li{margin-top:8px}@media(max-width:700px){body{padding:12px}.capture dl{grid-template-columns:1fr}.capture dt{font-weight:650}}
  </style>
</head>
<body>
  <header>
    <h1>UI smoke comparison</h1>
    <div class="overview"><div><strong>Overall status</strong>${escapeHtml(overallStatus)}</div><div><strong>Capture validity</strong>${summary.valid ? 'valid' : 'invalid'}</div><div><strong>Base revision</strong>${escapeHtml(summary.mainLabel)}</div><div><strong>Head revision</strong>${escapeHtml(summary.prLabel)}</div><div><strong>Successful pairs</strong>${summary.stats.success}</div><div><strong>Pairs with visual differences</strong>${summary.stats.pagesWithDiff}</div></div>
  </header>
  <div class="captures">${baseCapture}${headCapture}</div>
  ${fatalErrors}
  <main>${results}</main>
</body>
</html>
`;
}

function logSummary(summary, summaryPath) {
  console.log('\n--- Summary ---');
  console.log(
    `Visual changes above ${summary.thresholds.diffThreshold}% are boxed in red (looks-same clusters).`,
  );
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
  let sourceMode = 'unknown';

  console.log('Generating side-by-side comparisons');
  console.log(`Main screenshots: ${options.mainDir}`);
  console.log(`PR screenshots: ${options.prDir}`);
  console.log(`Output: ${options.outputDir}`);

  fs.mkdirSync(options.outputDir, { recursive: true });
  cleanComparisonOutputs(options.outputDir);

  if (fatalErrors.length === 0) {
    try {
      const plan = buildComparisonPlan(options.mainDir, options.prDir);
      mainLabel = options.mainLabel || plan.mainLabel;
      prLabel = options.prLabel || plan.prLabel;
      sourceMode = plan.sourceMode;
      captures = plan.captures;
      cleanComparisonOutputs(options.outputDir, plan.filenames);
      results.push(...plan.results);

      if (plan.filenames.length === 0) {
        fatalErrors.push('No screenshots found to compare.');
      }

      for (const pair of plan.pairs) {
        const outputPath = path.join(options.outputDir, pair.filename);
        try {
          const baseImage = validateFreshCapture(pair.mainRecord, pair.mainPath, 'Base');
          const headImage = validateFreshCapture(pair.prRecord, pair.prPath, 'Head');
          const diffAnalysis = await analyzeDiff(
            pair.mainPath,
            pair.prPath,
            options,
            compareImages,
          );
          const hasVisualDiff = diffAnalysis.diffPercent > options.diffThreshold;
          const exceedsFailThreshold =
            options.failThreshold !== null && diffAnalysis.diffPercent > options.failThreshold;
          await generateComparison(
            pair.page,
            pair.mainPath,
            pair.prPath,
            outputPath,
            mainLabel,
            prLabel,
            diffAnalysis,
            hasVisualDiff,
          );
          const comparisonImage = fs.readFileSync(outputPath);
          embeddedImages.set(pair.filename, {
            base: baseImage,
            head: headImage,
            comparison: comparisonImage,
          });
          results.push({
            filename: pair.filename,
            page: pair.page,
            required: pair.required,
            outputPath,
            mainExists: true,
            prExists: true,
            diffPercent: diffAnalysis.diffPercent,
            diffRegionCount: hasVisualDiff ? diffAnalysis.regions.length : 0,
            hasVisualDiff,
            exceedsFailThreshold,
            status: 'success',
          });
        } catch (error) {
          console.error(`  ✗ Failed: ${error.message}`);
          try {
            if (fs.existsSync(outputPath)) {
              fs.unlinkSync(outputPath);
            }
          } catch (cleanupError) {
            console.error(`  ✗ Failed to remove incomplete output: ${cleanupError.message}`);
          }
          results.push({
            filename: pair.filename,
            page: pair.page,
            required: pair.required,
            status: 'failed',
            error: error.message,
            failureType: error.failureType || 'comparison',
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
    console.error(
      `\nVisual diff threshold exceeded: ${summary.stats.pagesExceedingFailThreshold} page(s) above ${options.failThreshold}%`,
    );
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
  COMPARISON_REPORT_FILENAME,
  COMPARISON_SUMMARY_SCHEMA_VERSION,
  ComparisonError,
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
  validateComparisonOptions,
  validateDistinctDirectories,
  validateFreshCapture,
};

if (require.main === module) {
  void main();
}
