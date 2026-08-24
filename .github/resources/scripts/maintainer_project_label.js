const DEPENDABOT_LOGIN = "dependabot[bot]";

function parseRootApprovers(ownersText) {
  const approvers = new Set();
  let inApprovers = false;
  let foundApprovers = false;

  for (const rawLine of ownersText.split(/\r?\n/)) {
    const line = rawLine.replace(/\s+#.*$/, "");
    const trimmed = line.trim();

    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    if (!/^\s/.test(line)) {
      if (/^approvers:\s*$/.test(trimmed)) {
        inApprovers = true;
        foundApprovers = true;
        continue;
      }
      if (inApprovers) {
        break;
      }
      continue;
    }

    if (!inApprovers) {
      continue;
    }

    const item = trimmed.match(/^-\s+([A-Za-z0-9](?:[A-Za-z0-9-]{0,38}))\s*$/);
    if (!item) {
      throw new Error(`Unsupported entry in root OWNERS approvers: ${trimmed}`);
    }
    approvers.add(item[1].toLowerCase());
  }

  if (!foundApprovers || approvers.size === 0) {
    throw new Error("Root OWNERS must contain a non-empty top-level approvers list");
  }

  return approvers;
}

function shouldTrackAuthor(login, approvers) {
  const normalizedLogin = login.toLowerCase();
  return normalizedLogin === DEPENDABOT_LOGIN || approvers.has(normalizedLogin);
}

async function reconcilePullRequest({
  github,
  owner,
  repo,
  pullRequest,
  approvers,
  dependabotLabel,
  trackingLabel,
}) {
  const labels = new Set(pullRequest.labels.map((label) => label.name));
  const shouldTrack = shouldTrackAuthor(pullRequest.user.login, approvers);
  const isDependabot = pullRequest.user.login.toLowerCase() === DEPENDABOT_LOGIN;
  const desiredLabels = new Map([
    [trackingLabel, shouldTrack],
    [dependabotLabel, isDependabot],
  ]);
  const changes = [];

  for (const [label, shouldHaveLabel] of desiredLabels) {
    const hasLabel = labels.has(label);
    if (shouldHaveLabel && !hasLabel) {
      await github.rest.issues.addLabels({
        owner,
        repo,
        issue_number: pullRequest.number,
        labels: [label],
      });
      changes.push(`added ${label}`);
    }

    if (!shouldHaveLabel && hasLabel) {
      await github.rest.issues.removeLabel({
        owner,
        repo,
        issue_number: pullRequest.number,
        name: label,
      });
      changes.push(`removed ${label}`);
    }
  }

  return changes.length > 0 ? changes.join(", ") : "unchanged";
}

module.exports = {
  DEPENDABOT_LOGIN,
  parseRootApprovers,
  reconcilePullRequest,
  shouldTrackAuthor,
};
