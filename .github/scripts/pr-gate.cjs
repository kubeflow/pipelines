// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

const APPROVAL_LABEL = 'pr-gate/approved';
const MAINTAINER_PERMISSIONS = new Set(['write', 'maintain', 'admin']);
const BOT_AUTHORS = new Set(['dependabot[bot]', 'copybara-service[bot]']);
const TRUSTED_ASSOCIATIONS = new Set(['MEMBER', 'OWNER', 'COLLABORATOR']);
const CLOSURE_MARKER = '<!-- kfp-pr-gate-admission -->';
const CONTRIBUTING_URL = 'https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md';

function hasApproval(pull) {
  return pull.labels.some((label) => label.name === APPROVAL_LABEL);
}

async function ensureApprovalLabel(github, repository) {
  try {
    await github.rest.issues.getLabel({ ...repository, name: APPROVAL_LABEL });
  } catch (error) {
    if (error.status !== 404) throw error;
    try {
      await github.rest.issues.createLabel({
        ...repository,
        name: APPROVAL_LABEL,
        color: '0e8a16',
        description: 'Maintainer-approved exception to the PR linked-issue admission requirement',
      });
    } catch (createError) {
      if (createError.status !== 422) throw createError;
      // Another PR may have created the repository-wide label concurrently.
      await github.rest.issues.getLabel({ ...repository, name: APPROVAL_LABEL });
    }
  }
}

async function inspectAdmission(github, repository, number) {
  const { data: pull } = await github.rest.pulls.get({ ...repository, pull_number: number });
  if (pull.merged || pull.state !== 'open') return { status: 'ignored', pull };
  if (
    BOT_AUTHORS.has(pull.user.login) ||
    TRUSTED_ASSOCIATIONS.has(pull.author_association) ||
    hasApproval(pull)
  ) {
    return { status: 'exempt', pull };
  }
  const result = await github.graphql(
    `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        pullRequest(number: $number) {
          closingIssuesReferences(first: 1) {
            nodes { number repository { nameWithOwner } labels(first: 100) { nodes { name } } }
          }
        }
      }
    }`,
    { ...repository, number },
  );
  const issue = result.repository.pullRequest.closingIssuesReferences.nodes[0];
  const localIssue =
    issue &&
    issue.repository.nameWithOwner.toLowerCase() ===
      `${repository.owner}/${repository.repo}`.toLowerCase();
  if (localIssue && issue.labels.nodes.some((label) => label.name === 'ready')) {
    return { status: 'ready', pull };
  }
  const reason = !issue
    ? 'no linked issue was found'
    : !localIssue
      ? 'the linked issue is not in this repository'
      : `linked issue #${issue.number} does not have the \`ready\` label`;
  return { status: 'needs-admission', pull, reason };
}

async function admittedResult(github, repository, number, decision) {
  if (
    decision.status === 'ready' &&
    !decision.pull.labels.some((label) => label.name === 'ok-to-test')
  ) {
    await github.rest.issues.addLabels({
      ...repository,
      issue_number: number,
      labels: ['ok-to-test'],
    });
  }
  return { status: decision.status };
}

async function explainClosure(github, repository, number, reason) {
  const body =
    `${CLOSURE_MARKER}\nClosing this PR because ${reason}. ` +
    `Please follow [CONTRIBUTING.md](${CONTRIBUTING_URL}): discuss the contribution in an issue, ` +
    'wait for a maintainer to apply `ready`, and link it with a closing keyword such as `Fixes #1234` before reopening.\n\n' +
    'A repository maintainer can comment `/allow` to admit this contribution without the linked-issue requirement ' +
    'and reopen an unmerged PR. Normal CI authorization and review/merge requirements still apply.';
  const comments = await github.paginate(github.rest.issues.listComments, {
    ...repository,
    issue_number: number,
    per_page: 100,
  });
  const existing = comments.find(
    (comment) =>
      comment.user.login === 'github-actions[bot]' && comment.body?.startsWith(CLOSURE_MARKER),
  );
  if (!existing) {
    await github.rest.issues.createComment({ ...repository, issue_number: number, body });
  } else if (existing.body !== body) {
    await github.rest.issues.updateComment({ ...repository, comment_id: existing.id, body });
  }
}

async function enforceAdmission(github, repository, number) {
  let decision = await inspectAdmission(github, repository, number);
  if (decision.status !== 'needs-admission')
    return admittedResult(github, repository, number, decision);
  await explainClosure(github, repository, number, decision.reason);
  // A queued event or a concurrently granted exemption must not authorize closure.
  decision = await inspectAdmission(github, repository, number);
  if (decision.status !== 'needs-admission')
    return admittedResult(github, repository, number, decision);
  let current;
  try {
    await github.rest.pulls.update({ ...repository, pull_number: number, state: 'closed' });
  } finally {
    // A close can succeed even if its response is lost. Reconcile concurrent /allow grants.
    ({ data: current } = await github.rest.pulls.get({ ...repository, pull_number: number }));
    if (hasApproval(current) && !current.merged && current.state === 'closed') {
      await github.rest.pulls.update({ ...repository, pull_number: number, state: 'open' });
    }
  }
  if (hasApproval(current) && !current.merged) return { status: 'exempt' };
  return { status: current.state === 'closed' && !current.merged ? 'closed' : 'ignored' };
}

async function handlePrGate({ github, context }) {
  const { payload, repo: repository } = context;
  if (context.eventName === 'pull_request_target') {
    if (
      !['opened', 'labeled', 'unlabeled', 'synchronize', 'reopened'].includes(payload.action) ||
      !Number.isSafeInteger(payload.pull_request?.number)
    )
      return { status: 'ignored' };
    return enforceAdmission(github, repository, payload.pull_request.number);
  }
  if (
    context.eventName !== 'issue_comment' ||
    payload.action !== 'created' ||
    !payload.issue?.pull_request ||
    payload.comment?.body?.trim() !== '/allow'
  ) {
    return { status: 'ignored' };
  }
  const number = payload.issue.number;
  const { data: comment } = await github.rest.issues.getComment({
    ...repository,
    comment_id: payload.comment.id,
  });
  if (comment.body.trim() !== '/allow' || comment.user.type !== 'User') {
    return { status: 'ignored' };
  }
  const { data: permission } = await github.rest.repos.getCollaboratorPermissionLevel({
    ...repository,
    username: comment.user.login,
  });
  if (!MAINTAINER_PERMISSIONS.has(permission.permission)) return { status: 'denied' };
  const { data: pull } = await github.rest.pulls.get({ ...repository, pull_number: number });
  if (pull.merged) return { status: 'ignored' };
  const alreadyApproved = hasApproval(pull);
  if (!alreadyApproved) {
    await ensureApprovalLabel(github, repository);
    await github.rest.issues.addLabels({
      ...repository,
      issue_number: number,
      labels: [APPROVAL_LABEL],
    });
  }
  const { data: current } = await github.rest.pulls.get({ ...repository, pull_number: number });
  if (current.merged || !hasApproval(current)) return { status: 'ignored' };
  const reopened = current.state === 'closed';
  if (reopened) {
    await github.rest.pulls.update({ ...repository, pull_number: number, state: 'open' });
  }
  if (alreadyApproved && !reopened) return { status: 'already-allowed' };
  await github.rest.issues.createComment({
    ...repository,
    issue_number: number,
    body:
      `@${comment.user.login} approved this contribution without the linked-issue requirement ` +
      `([request](${comment.html_url})). ${reopened ? 'This PR has been reopened. ' : ''}PR admission is satisfied. ` +
      'CI authorization, required checks, and review/merge requirements remain unchanged.',
  });
  return { status: 'allowed', reopened };
}

module.exports = { handlePrGate };
