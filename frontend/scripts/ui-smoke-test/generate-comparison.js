#!/usr/bin/env node
/**
 * Side-by-Side Comparison Generator
 *
 * Takes screenshots from main and PR branches and creates side-by-side
 * comparison images with labels.
 *
 * Usage: node generate-comparison.js --main ./screenshots/main --pr ./screenshots/pr --output ./screenshots/comparison
 */

const looksSame = require('looks-same');
const sharp = require('sharp');
const path = require('path');
const fs = require('fs');

const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};

const MAIN_DIR = getArg('main', './screenshots/main');
const PR_DIR = getArg('pr', './screenshots/pr');
const OUTPUT_DIR = getArg('output', './screenshots/comparison');

// Highlight any non-zero diff by default.
const DIFF_THRESHOLD = Number(getArg('diff-threshold', process.env.UI_SMOKE_DIFF_THRESHOLD || '0'));

// Fail on any non-zero diff by default.
const FAIL_THRESHOLD_RAW = getArg('fail-threshold', process.env.UI_SMOKE_FAIL_THRESHOLD || '0');
const FAIL_THRESHOLD = FAIL_THRESHOLD_RAW === '' ? null : Number(FAIL_THRESHOLD_RAW);

const LOOKSAME_TOLERANCE = Number(
  getArg('looksame-tolerance', process.env.UI_SMOKE_LOOKSAME_TOLERANCE || '2.3'),
);
const LOOKSAME_CLUSTER_SIZE = Number(
  getArg('looksame-cluster-size', process.env.UI_SMOKE_LOOKSAME_CLUSTER_SIZE || '8'),
);

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

function escapeXml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
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

  try {
    const image = sharp(imagePath);

    const resizedImage = image.resize(width, height, {
      fit: 'contain',
      background: { r: 245, g: 245, b: 245, alpha: 1 },
    });

    const labelBuffer = Buffer.from(labelSvg);

    return sharp({
      create: {
        width,
        height: height + LABEL_HEIGHT,
        channels: 4,
        background: { r: 255, g: 255, b: 255, alpha: 1 },
      },
    })
      .composite([
        { input: labelBuffer, top: 0, left: 0 },
        { input: await resizedImage.toBuffer(), top: LABEL_HEIGHT, left: 0 },
      ])
      .png();
  } catch (error) {
    const placeholderSvg = `
      <svg width="${width}" height="${height}">
        <rect width="100%" height="100%" fill="#f0f0f0"/>
        <text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle"
              font-family="Arial, sans-serif" font-size="18" fill="#888888">
          No screenshot available
        </text>
      </svg>
    `;

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
        { input: Buffer.from(placeholderSvg), top: LABEL_HEIGHT, left: 0 },
      ])
      .png();
  }
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

  return {
    x: clampedLeft,
    y: clampedTop,
    width,
    height,
  };
}

function extractDiffRegions(looksSameResult, imageWidth, imageHeight, diffPercent) {
  const clusters = Array.isArray(looksSameResult?.diffClusters)
    ? looksSameResult.diffClusters
    : [];

  let regions = clusters
    .map(cluster => normalizeRegion(cluster, imageWidth, imageHeight))
    .filter(Boolean);

  if (regions.length === 0 && looksSameResult?.diffBounds) {
    const bounds = normalizeRegion(looksSameResult.diffBounds, imageWidth, imageHeight);
    if (bounds) {
      regions = [bounds];
    }
  }

  const filtered = regions.filter(region => region.width * region.height >= MIN_REGION_AREA_PX);
  const withArea = (filtered.length > 0 ? filtered : regions).map(region => ({
    ...region,
    area: region.width * region.height,
  }));

  const maxRegions = maxRegionsForDiff(diffPercent);
  return withArea
    .sort((a, b) => b.area - a.area)
    .slice(0, maxRegions)
    .map(({ area, ...region }) => region);
}

function deriveDiffPercent(looksSameResult, width, height) {
  if (Number.isFinite(looksSameResult?.differentPixels) && Number.isFinite(looksSameResult?.totalPixels)) {
    const total = looksSameResult.totalPixels;
    if (total > 0) {
      return (looksSameResult.differentPixels / total) * 100;
    }
  }

  if (Number.isFinite(looksSameResult?.diffPercentage)) {
    const raw = looksSameResult.diffPercentage;
    return raw <= 1 ? raw * 100 : raw;
  }

  if (looksSameResult?.equal === true) {
    return 0;
  }

  if (looksSameResult?.diffBounds && width > 0 && height > 0) {
    const bounds = normalizeRegion(looksSameResult.diffBounds, width, height);
    if (bounds) {
      const area = bounds.width * bounds.height;
      return (area / (width * height)) * 100;
    }
  }

  return null;
}

async function analyzeDiff(mainPath, prPath) {
  try {
    if (!fs.existsSync(mainPath) || !fs.existsSync(prPath)) {
      return null;
    }

    const [mainMeta, prMeta] = await Promise.all([
      sharp(mainPath).metadata(),
      sharp(prPath).metadata(),
    ]);

    const imageWidth = prMeta.width || mainMeta.width || 0;
    const imageHeight = prMeta.height || mainMeta.height || 0;

    if (!imageWidth || !imageHeight) {
      return null;
    }

    const looksSameResult = await looksSame(mainPath, prPath, {
      shouldCluster: true,
      clustersSize: Math.max(1, Math.floor(LOOKSAME_CLUSTER_SIZE)),
      tolerance: LOOKSAME_TOLERANCE,
      ignoreCaret: true,
      ignoreAntialiasing: true,
      createDiffImage: true,
    });

    const diffPercent = deriveDiffPercent(looksSameResult, imageWidth, imageHeight);
    const regions = extractDiffRegions(looksSameResult, imageWidth, imageHeight, diffPercent || 0);

    return {
      diffPercent,
      regions,
      width: imageWidth,
      height: imageHeight,
    };
  } catch (error) {
    return null;
  }
}

function createDiffOverlay(diffAnalysis, renderWidth, renderHeight, totalHeight) {
  if (!diffAnalysis || !Array.isArray(diffAnalysis.regions) || diffAnalysis.regions.length === 0) {
    return null;
  }

  const scaleX = renderWidth / diffAnalysis.width;
  const scaleY = renderHeight / diffAnalysis.height;
  const rightOffset = renderWidth + DIVIDER_WIDTH;

  const boxes = diffAnalysis.regions.map(region => {
    const baseX = region.x * scaleX;
    const baseY = LABEL_HEIGHT + region.y * scaleY;
    const baseWidth = region.width * scaleX;
    const baseHeight = region.height * scaleY;

    const x = Math.max(0, baseX - REGION_BOX_PADDING);
    const y = Math.max(LABEL_HEIGHT, baseY - REGION_BOX_PADDING);
    const w = Math.max(1, Math.min(renderWidth - x, baseWidth + REGION_BOX_PADDING * 2));
    const h = Math.max(1, Math.min(totalHeight - y, baseHeight + REGION_BOX_PADDING * 2));

    return `
      <rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${DIFF_MARKER_RADIUS}" ry="${DIFF_MARKER_RADIUS}" fill="none" stroke="${DIFF_MARKER_COLOR}" stroke-width="${DIFF_MARKER_WIDTH}" />
      <rect x="${x + rightOffset}" y="${y}" width="${w}" height="${h}" rx="${DIFF_MARKER_RADIUS}" ry="${DIFF_MARKER_RADIUS}" fill="none" stroke="${DIFF_MARKER_COLOR}" stroke-width="${DIFF_MARKER_WIDTH}" />
    `;
  });

  const overlaySvg = `
    <svg width="${renderWidth * 2 + DIVIDER_WIDTH}" height="${totalHeight}">
      ${boxes.join('\n')}
    </svg>
  `;

  return Buffer.from(overlaySvg);
}

async function generateComparison(pageName, mainPath, prPath, outputPath, mainLabel, prLabel, diffAnalysis) {
  console.log(`Generating comparison for: ${pageName}`);

  let width = 640;
  let height = 400;

  try {
    const metadata = await sharp(prPath).metadata();
    if (metadata.width && metadata.height) {
      width = Math.floor(metadata.width / 2);
      height = Math.floor(metadata.height / 2);
    }
  } catch (e) {
    // Keep defaults.
  }

  const totalHeight = height + LABEL_HEIGHT;

  const mainImage = await createLabeledImage(mainPath, mainLabel, width, height);
  const prImage = await createLabeledImage(prPath, prLabel, width, height);
  const divider = await createDivider(totalHeight);

  const composites = [
    { input: await mainImage.toBuffer(), top: 0, left: 0 },
    { input: await divider.toBuffer(), top: 0, left: width },
    { input: await prImage.toBuffer(), top: 0, left: width + DIVIDER_WIDTH },
  ];

  const diffOverlay = createDiffOverlay(diffAnalysis, width, height, totalHeight);
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

async function main() {
  console.log('Generating side-by-side comparisons');
  console.log(`Main screenshots: ${MAIN_DIR}`);
  console.log(`PR screenshots: ${PR_DIR}`);
  console.log(`Output: ${OUTPUT_DIR}`);

  if (!Number.isFinite(DIFF_THRESHOLD) || DIFF_THRESHOLD < 0) {
    console.error(`Invalid diff threshold: ${DIFF_THRESHOLD}`);
    process.exit(1);
  }

  if (FAIL_THRESHOLD !== null && (!Number.isFinite(FAIL_THRESHOLD) || FAIL_THRESHOLD < 0)) {
    console.error(`Invalid fail threshold: ${FAIL_THRESHOLD_RAW}`);
    process.exit(1);
  }

  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  const mainFiles = fs.existsSync(MAIN_DIR)
    ? fs.readdirSync(MAIN_DIR).filter(f => f.endsWith('.png'))
    : [];
  const prFiles = fs.existsSync(PR_DIR)
    ? fs.readdirSync(PR_DIR).filter(f => f.endsWith('.png'))
    : [];

  const allPages = new Set([...mainFiles, ...prFiles]);

  if (allPages.size === 0) {
    console.error('No screenshots found to compare');
    process.exit(1);
  }

  let mainLabel = 'main (base)';
  let prLabel = 'PR (head)';
  try {
    const mainManifest = JSON.parse(fs.readFileSync(path.join(MAIN_DIR, 'manifest.json'), 'utf8'));
    if (mainManifest.label) mainLabel = mainManifest.label;
  } catch (e) {
    // Use default.
  }
  try {
    const prManifest = JSON.parse(fs.readFileSync(path.join(PR_DIR, 'manifest.json'), 'utf8'));
    if (prManifest.label) prLabel = prManifest.label;
  } catch (e) {
    // Use default.
  }

  console.log(`Main label: ${mainLabel}`);
  console.log(`PR label: ${prLabel}`);

  const results = [];

  for (const filename of allPages) {
    const pageName = filename.replace('.png', '');
    const mainPath = path.join(MAIN_DIR, filename);
    const prPath = path.join(PR_DIR, filename);
    const outputPath = path.join(OUTPUT_DIR, filename);

    try {
      const diffAnalysis = await analyzeDiff(mainPath, prPath);
      await generateComparison(pageName, mainPath, prPath, outputPath, mainLabel, prLabel, diffAnalysis);

      const hasVisualDiff =
        diffAnalysis !== null &&
        Number.isFinite(diffAnalysis.diffPercent) &&
        diffAnalysis.diffPercent > DIFF_THRESHOLD;
      const exceedsFailThreshold =
        FAIL_THRESHOLD !== null &&
        diffAnalysis !== null &&
        Number.isFinite(diffAnalysis.diffPercent) &&
        diffAnalysis.diffPercent > FAIL_THRESHOLD;

      results.push({
        page: pageName,
        outputPath,
        mainExists: fs.existsSync(mainPath),
        prExists: fs.existsSync(prPath),
        diffPercent: diffAnalysis ? diffAnalysis.diffPercent : null,
        diffRegionCount: diffAnalysis ? diffAnalysis.regions.length : 0,
        hasVisualDiff,
        exceedsFailThreshold,
        status: 'success',
      });
    } catch (error) {
      console.error(`  ✗ Failed: ${error.message}`);
      results.push({
        page: pageName,
        status: 'failed',
        error: error.message,
      });
    }
  }

  const summaryPath = path.join(OUTPUT_DIR, 'summary.json');
  fs.writeFileSync(
    summaryPath,
    JSON.stringify(
      {
        timestamp: new Date().toISOString(),
        mainLabel,
        prLabel,
        thresholds: {
          diffThreshold: DIFF_THRESHOLD,
          failThreshold: FAIL_THRESHOLD,
        },
        results,
        stats: {
          total: results.length,
          success: results.filter(r => r.status === 'success').length,
          failed: results.filter(r => r.status === 'failed').length,
          pagesExceedingFailThreshold:
            FAIL_THRESHOLD === null ? 0 : results.filter(r => r.exceedsFailThreshold).length,
          pagesWithDiff: results.filter(r => r.hasVisualDiff).length,
        },
      },
      null,
      2,
    ),
  );

  console.log('\n--- Summary ---');
  console.log(
    `Visual changes above ${DIFF_THRESHOLD}% are boxed in red (looks-same clusters).`,
  );
  for (const result of results) {
    if (result.status === 'success') {
      const diffStr = result.diffPercent !== null
        ? `(${result.diffPercent.toFixed(2)}% diff)`
        : '(diff not calculated)';
      const visualMarker = result.hasVisualDiff ? ' [diff]' : '';
      const failMarker = result.exceedsFailThreshold ? ' [above fail-threshold]' : '';
      console.log(`  ${result.page}: ✓ ${diffStr}${visualMarker}${failMarker}`);
    } else {
      console.log(`  ${result.page}: ✗ ${result.error}`);
    }
  }

  console.log(`\nSummary saved to: ${summaryPath}`);

  if (FAIL_THRESHOLD !== null) {
    const exceeded = results.filter(r => r.exceedsFailThreshold);
    if (exceeded.length > 0) {
      console.error(
        `\nVisual diff threshold exceeded: ${exceeded.length} page(s) above ${FAIL_THRESHOLD}%`,
      );
      for (const item of exceeded) {
        if (item.diffPercent !== null) {
          console.error(`  ${item.page}: ${item.diffPercent.toFixed(2)}%`);
        } else {
          console.error(`  ${item.page}: diff not calculated`);
        }
      }
      process.exit(1);
    }
  }
}

main().catch(err => {
  console.error('Comparison generation failed:', err);
  process.exit(1);
});
