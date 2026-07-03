#!/usr/bin/env node
/**
 * PR Upload Script
 *
 * Generates a markdown summary of UI smoke test comparison results
 * and posts it as a comment on a GitHub PR using the GitHub CLI.
 *
 * Usage: node upload-to-pr.js --pr 12756 --repo kubeflow/pipelines --screenshots ./screenshots/comparison
 */

const { execSync } = require('child_process');
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
cd frontend/scripts/ui-smoke-test
npm ci
node smoke-test-runner.js --compare master --pr ${PR_NUMBER} --skip-upload
\`\`\`

</details>
`;

  return md;
}

/**
 * Post comment to PR using gh CLI.
 * If an existing smoke test comment is found, edit it instead of creating a new one.
 */
function postCommentToPR(markdown) {
  console.log(`Posting comment to PR #${PR_NUMBER}...`);

  // Write markdown to temp file to avoid shell escaping issues
  const tempFile = path.join(SCREENSHOTS_DIR, 'pr-comment.md');
  fs.writeFileSync(tempFile, markdown);

  const existingId = findExistingComment();

  if (existingId) {
    console.log(`Updating existing comment (ID: ${existingId})...`);
    const result = runCommand(
      `gh api repos/${REPO}/issues/comments/${existingId} -X PATCH -F body=@"${tempFile}"`,
      { silent: false }
    );

    if (result !== null) {
      console.log('✓ Comment updated successfully');
      return true;
    }
    console.log('Failed to update existing comment, creating new one...');
  }

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
 * Check if there's already a smoke test comment and return its ID for editing.
 */
function findExistingComment() {
  const result = runCommand(
    `gh api repos/${REPO}/issues/${PR_NUMBER}/comments --jq '.[] | select(.body | contains("UI Smoke Test Results")) | .id'`,
    { silent: true }
  );

  if (result) {
    const ids = result.split('\n').filter(Boolean);
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

  // Post (or update existing) comment
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
