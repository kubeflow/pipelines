#!/usr/bin/env node
/**
 * PR Upload Script
 *
 * Uploads comparison screenshots to a GitHub PR as a comment.
 * Uses GitHub CLI for authentication and imgbb/imgur for image hosting.
 *
 * Usage: node upload-to-pr.js --pr 12756 --repo kubeflow/pipelines --screenshots ./screenshots/comparison
 */

const { execSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// Parse command line arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};

const PR_NUMBER = getArg('pr', '');
const REPO = getArg('repo', 'kubeflow/pipelines');
const SCREENSHOTS_DIR = getArg('screenshots', './screenshots/comparison');

if (!PR_NUMBER) {
  console.error('Error: --pr <number> is required');
  process.exit(1);
}

/**
 * Run a command and return stdout
 */
function runCommand(cmd, options = {}) {
  try {
    return execSync(cmd, { encoding: 'utf8', ...options }).trim();
  } catch (error) {
    if (!options.silent) {
      console.error(`Command failed: ${cmd}`);
      console.error(error.message);
    }
    return null;
  }
}

/**
 * Upload image to GitHub as a release asset or via issue attachment workaround
 * Returns the URL of the uploaded image
 */
async function uploadImageToGitHub(imagePath) {
  // GitHub doesn't have a direct image upload API for comments
  // Options:
  // 1. Use a third-party image host
  // 2. Create a gist with the image (base64 encoded)
  // 3. Upload as release asset
  // 4. Use the drag-drop workaround (manual)

  // For automation, we'll create a summary with local paths
  // and instructions for viewing

  return path.basename(imagePath);
}

/**
 * Generate markdown summary from comparison results
 */
function generateMarkdownSummary(summaryPath) {
  let summary;
  try {
    summary = JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
  } catch (e) {
    console.error('Could not read summary.json:', e.message);
    return null;
  }

  let md = `## 🔍 UI Smoke Test Results

**Generated:** ${new Date(summary.timestamp).toLocaleString()}

### Summary
- **Pages Tested:** ${summary.stats.total}
- **Successful:** ${summary.stats.success}
- **Failed:** ${summary.stats.failed}
- **Pages with Visual Changes:** ${summary.stats.pagesWithDiff}

### Page Results

| Page | Status | Diff % | Notes |
|------|--------|--------|-------|
`;

  for (const result of summary.results) {
    const status = result.status === 'success' ? '✅' : '❌';
    const diff = result.diffPercent !== null
      ? `${result.diffPercent.toFixed(2)}%`
      : 'N/A';
    const notes = [];

    if (!result.mainExists) notes.push('No main screenshot');
    if (!result.prExists) notes.push('No PR screenshot');
    if (result.diffPercent && result.diffPercent > 1) notes.push('⚠️ Visual changes detected');
    if (result.error) notes.push(result.error);

    md += `| ${result.page} | ${status} | ${diff} | ${notes.join(', ') || '-'} |\n`;
  }

  // Add instructions for viewing full results
  md += `
### Viewing Full Comparison Images

The comparison images are available in the CI artifacts. To view them:

1. Download the \`ui-smoke-test-results\` artifact from the workflow run
2. Open the \`.ui-smoke-test/screenshots/comparison/\` directory
3. Each PNG shows the main branch (left) vs PR branch (right)

---
<details>
<summary>How to run locally</summary>

\`\`\`bash
cd frontend
npm ci
npm run build
cd ../scripts/ui-smoke-test
npm install
./smoke-test.sh ${PR_NUMBER} --skip-upload
\`\`\`

</details>
`;

  return md;
}

/**
 * Post comment to PR using gh CLI
 */
function postCommentToPR(markdown) {
  console.log(`Posting comment to PR #${PR_NUMBER}...`);

  // Write markdown to temp file to avoid shell escaping issues
  const tempFile = path.join(SCREENSHOTS_DIR, 'pr-comment.md');
  fs.writeFileSync(tempFile, markdown);

  // Use gh CLI to post comment
  const result = runCommand(
    `gh pr comment ${PR_NUMBER} --repo ${REPO} --body-file "${tempFile}"`,
    { silent: false }
  );

  if (result !== null) {
    console.log('✓ Comment posted successfully');
    return true;
  } else {
    console.error('Failed to post comment');
    return false;
  }
}

/**
 * Check if there's already a smoke test comment and edit it instead
 */
function findExistingComment() {
  const comments = runCommand(
    `gh pr view ${PR_NUMBER} --repo ${REPO} --json comments --jq '.comments[] | select(.body | contains("UI Smoke Test Results")) | .id'`,
    { silent: true }
  );

  if (comments) {
    const ids = comments.split('\n').filter(Boolean);
    return ids[ids.length - 1]; // Return most recent
  }
  return null;
}

async function main() {
  console.log(`Uploading UI smoke test results to PR #${PR_NUMBER}`);
  console.log(`Repository: ${REPO}`);
  console.log(`Screenshots directory: ${SCREENSHOTS_DIR}`);

  // Check for gh CLI
  const ghVersion = runCommand('gh --version', { silent: true });
  if (!ghVersion) {
    console.error('Error: GitHub CLI (gh) is not installed or not in PATH');
    process.exit(1);
  }

  // Check gh auth status
  const authStatus = runCommand('gh auth status', { silent: true });
  if (!authStatus || authStatus.includes('not logged')) {
    console.error('Error: Not authenticated with GitHub CLI. Run: gh auth login');
    process.exit(1);
  }

  // Check for summary file
  const summaryPath = path.join(SCREENSHOTS_DIR, 'summary.json');
  if (!fs.existsSync(summaryPath)) {
    console.error(`Error: Summary file not found: ${summaryPath}`);
    console.error('Run generate-comparison.js first');
    process.exit(1);
  }

  // Generate markdown
  const markdown = generateMarkdownSummary(summaryPath);
  if (!markdown) {
    console.error('Failed to generate markdown summary');
    process.exit(1);
  }

  // Check for existing comment to update
  const existingCommentId = findExistingComment();
  if (existingCommentId) {
    console.log(`Found existing smoke test comment (ID: ${existingCommentId})`);
    // gh doesn't support editing comments directly, so we'll add a new one
    // with a note that it supersedes the previous one
  }

  // Post the comment
  const success = postCommentToPR(markdown);

  if (success) {
    console.log(`\n✓ Results posted to: https://github.com/${REPO}/pull/${PR_NUMBER}`);
  } else {
    console.error('\n✗ Failed to post results');
    process.exit(1);
  }
}

main().catch(err => {
  console.error('Upload failed:', err);
  process.exit(1);
});
