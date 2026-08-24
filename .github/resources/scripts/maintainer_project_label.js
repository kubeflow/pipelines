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
  trackingLabel,
}) {
  const labels = new Set(pullRequest.labels.map((label) => label.name));
  const shouldTrack = shouldTrackAuthor(pullRequest.user.login, approvers);
  const isTracked = labels.has(trackingLabel);

  if (shouldTrack && !isTracked) {
    await github.rest.issues.addLabels({
      owner,
      repo,
      issue_number: pullRequest.number,
      labels: [trackingLabel],
    });
    return "added";
  }

  if (!shouldTrack && isTracked) {
    await github.rest.issues.removeLabel({
      owner,
      repo,
      issue_number: pullRequest.number,
      name: trackingLabel,
    });
    return "removed";
  }

  return "unchanged";
}

module.exports = {
  DEPENDABOT_LOGIN,
  parseRootApprovers,
  reconcilePullRequest,
  shouldTrackAuthor,
};

