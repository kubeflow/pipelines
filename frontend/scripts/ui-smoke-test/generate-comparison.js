#!/usr/bin/env node
/**
 * Side-by-Side Comparison Generator
 *
 * Takes screenshots from main and PR branches and creates side-by-side
 * comparison images with labels.
 *
 * Usage: node generate-comparison.js --main ./screenshots/main --pr ./screenshots/pr --output ./screenshots/comparison
 */

const sharp = require('sharp');
const path = require('path');
const fs = require('fs');

// Parse command line arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};

const MAIN_DIR = getArg('main', './screenshots/main');
const PR_DIR = getArg('pr', './screenshots/pr');
const OUTPUT_DIR = getArg('output', './screenshots/comparison');

// Colors for labels
const LABEL_HEIGHT = 40;
const LABEL_BACKGROUND = '#1a1a2e';
const LABEL_TEXT_COLOR = '#ffffff';
const DIVIDER_WIDTH = 4;
const DIVIDER_COLOR = '#4a4e69';

function escapeXml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

async function createLabeledImage(imagePath, label, width, height) {
  /**
   * Creates an image with a label header
   */
  const safeLabel = escapeXml(label);
  // Use smaller font for long labels (e.g. with SHA info)
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
    const metadata = await image.metadata();

    // Resize image to target dimensions if needed
    const resizedImage = image.resize(width, height, {
      fit: 'contain',
      background: { r: 245, g: 245, b: 245, alpha: 1 },
    });

    // Composite the label on top
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
    // Return a placeholder if image doesn't exist
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
  /**
   * Creates a vertical divider
   */
  return sharp({
    create: {
      width: DIVIDER_WIDTH,
      height,
      channels: 4,
      background: { r: 74, g: 78, b: 105, alpha: 1 }, // DIVIDER_COLOR
    },
  }).png();
}

async function generateComparison(pageName, mainPath, prPath, outputPath, mainLabel, prLabel) {
  /**
   * Generates a side-by-side comparison image
   */
  console.log(`Generating comparison for: ${pageName}`);

  // Get dimensions from PR image (or use default)
  let width = 640; // Half of 1280
  let height = 400; // Half of 800

  try {
    const metadata = await sharp(prPath).metadata();
    if (metadata.width && metadata.height) {
      width = Math.floor(metadata.width / 2);
      height = Math.floor(metadata.height / 2);
    }
  } catch (e) {
    // Use defaults
  }

  const totalHeight = height + LABEL_HEIGHT;

  // Create labeled images
  const mainImage = await createLabeledImage(mainPath, mainLabel, width, height);
  const prImage = await createLabeledImage(prPath, prLabel, width, height);
  const divider = await createDivider(totalHeight);

  // Combine side by side
  const combined = await sharp({
    create: {
      width: width * 2 + DIVIDER_WIDTH,
      height: totalHeight,
      channels: 4,
      background: { r: 255, g: 255, b: 255, alpha: 1 },
    },
  })
    .composite([
      { input: await mainImage.toBuffer(), top: 0, left: 0 },
      { input: await divider.toBuffer(), top: 0, left: width },
      { input: await prImage.toBuffer(), top: 0, left: width + DIVIDER_WIDTH },
    ])
    .png()
    .toFile(outputPath);

  console.log(`  ✓ Saved: ${path.basename(outputPath)}`);
  return outputPath;
}

async function calculateDiff(mainPath, prPath) {
  /**
   * Calculates pixel difference percentage between two images
   * Returns null if comparison not possible
   */
  try {
    const [mainBuffer, prBuffer] = await Promise.all([
      sharp(mainPath).raw().toBuffer({ resolveWithObject: true }),
      sharp(prPath).raw().toBuffer({ resolveWithObject: true }),
    ]);

    if (mainBuffer.info.width !== prBuffer.info.width ||
        mainBuffer.info.height !== prBuffer.info.height) {
      return null; // Different dimensions
    }

    const main = mainBuffer.data;
    const pr = prBuffer.data;

    let diffPixels = 0;
    const totalPixels = main.length / 4; // RGBA

    for (let i = 0; i < main.length; i += 4) {
      const rDiff = Math.abs(main[i] - pr[i]);
      const gDiff = Math.abs(main[i + 1] - pr[i + 1]);
      const bDiff = Math.abs(main[i + 2] - pr[i + 2]);

      // Consider pixel different if any channel differs by more than threshold
      if (rDiff > 5 || gDiff > 5 || bDiff > 5) {
        diffPixels++;
      }
    }

    return (diffPixels / totalPixels) * 100;
  } catch (e) {
    return null;
  }
}

async function main() {
  console.log('Generating side-by-side comparisons');
  console.log(`Main screenshots: ${MAIN_DIR}`);
  console.log(`PR screenshots: ${PR_DIR}`);
  console.log(`Output: ${OUTPUT_DIR}`);

  // Ensure output directory exists
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });

  // Get all screenshots from both directories
  const mainFiles = fs.existsSync(MAIN_DIR)
    ? fs.readdirSync(MAIN_DIR).filter(f => f.endsWith('.png') && f !== 'manifest.json')
    : [];
  const prFiles = fs.existsSync(PR_DIR)
    ? fs.readdirSync(PR_DIR).filter(f => f.endsWith('.png') && f !== 'manifest.json')
    : [];

  // Combine unique page names
  const allPages = new Set([...mainFiles, ...prFiles]);

  if (allPages.size === 0) {
    console.error('No screenshots found to compare');
    process.exit(1);
  }

  // Read labels from manifest files if available, otherwise use defaults
  let mainLabel = 'main (base)';
  let prLabel = 'PR (head)';
  try {
    const mainManifest = JSON.parse(fs.readFileSync(path.join(MAIN_DIR, 'manifest.json'), 'utf8'));
    if (mainManifest.label) mainLabel = mainManifest.label;
  } catch (e) { /* use default */ }
  try {
    const prManifest = JSON.parse(fs.readFileSync(path.join(PR_DIR, 'manifest.json'), 'utf8'));
    if (prManifest.label) prLabel = prManifest.label;
  } catch (e) { /* use default */ }

  console.log(`Main label: ${mainLabel}`);
  console.log(`PR label: ${prLabel}`);

  const results = [];

  for (const filename of allPages) {
    const pageName = filename.replace('.png', '');
    const mainPath = path.join(MAIN_DIR, filename);
    const prPath = path.join(PR_DIR, filename);
    const outputPath = path.join(OUTPUT_DIR, filename);

    try {
      await generateComparison(pageName, mainPath, prPath, outputPath, mainLabel, prLabel);

      // Calculate diff percentage
      const diffPercent = await calculateDiff(mainPath, prPath);

      results.push({
        page: pageName,
        outputPath,
        mainExists: fs.existsSync(mainPath),
        prExists: fs.existsSync(prPath),
        diffPercent,
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

  // Generate summary
  const summaryPath = path.join(OUTPUT_DIR, 'summary.json');
  fs.writeFileSync(summaryPath, JSON.stringify({
    timestamp: new Date().toISOString(),
    mainLabel,
    prLabel,
    results,
    stats: {
      total: results.length,
      success: results.filter(r => r.status === 'success').length,
      failed: results.filter(r => r.status === 'failed').length,
      pagesWithDiff: results.filter(r => r.diffPercent && r.diffPercent > 0.1).length,
    },
  }, null, 2));

  // Print summary
  console.log('\n--- Summary ---');
  for (const result of results) {
    if (result.status === 'success') {
      const diffStr = result.diffPercent !== null
        ? `(${result.diffPercent.toFixed(2)}% diff)`
        : '(diff not calculated)';
      console.log(`  ${result.page}: ✓ ${diffStr}`);
    } else {
      console.log(`  ${result.page}: ✗ ${result.error}`);
    }
  }

  console.log(`\nSummary saved to: ${summaryPath}`);
}

main().catch(err => {
  console.error('Comparison generation failed:', err);
  process.exit(1);
});
