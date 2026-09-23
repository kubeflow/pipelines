// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

const assert = require('node:assert/strict');
const test = require('node:test');
const { handlePrGate } = require('./pr-gate.cjs');

function harness({ permission = 'write', state = 'open', labels = [] } = {}) {
  const pull = {
    number: 42,
    state,
    merged: false,
    user: { login: 'contributor' },
    author_association: 'NONE',
    labels: labels.map((name) => ({ name })),
  };
  const writes = [];
  const comments = [];
  const command = {
    id: 123,
    body: '/allow',
    user: { login: 'maintainer', type: 'User' },
    html_url: 'https://github.com/kubeflow/pipelines/pull/42#issuecomment-123',
  };
  let repositoryLabel = false;
  const github = {
    graphql: async () => ({
      repository: { pullRequest: { closingIssuesReferences: { nodes: [] } } },
    }),
    rest: {
      pulls: {
        get: async () => ({ data: structuredClone(pull) }),
        update: async (args) => {
          writes.push(['pull', args]);
          pull.state = args.state;
          return { data: structuredClone(pull) };
        },
      },
      repos: {
        getCollaboratorPermissionLevel: async () => ({ data: { permission } }),
      },
      issues: {
        getComment: async () => ({ data: structuredClone(command) }),
        getLabel: async () => {
          if (!repositoryLabel) throw Object.assign(new Error('Not found'), { status: 404 });
          return { data: { name: 'pr-gate/approved' } };
        },
        createLabel: async (args) => {
          writes.push(['create-label', args]);
          repositoryLabel = true;
          return { data: args };
        },
        addLabels: async (args) => {
          writes.push(['labels', args]);
          for (const name of args.labels) {
            if (!pull.labels.some((label) => label.name === name)) pull.labels.push({ name });
          }
          return { data: structuredClone(pull.labels) };
        },
        createComment: async (args) => {
          writes.push(['comment', args]);
          comments.push({
            id: comments.length + 1,
            user: { login: 'github-actions[bot]' },
            ...args,
          });
          return { data: comments.at(-1) };
        },
        updateComment: async (args) => {
          writes.push(['update-comment', args]);
          Object.assign(
            comments.find((comment) => comment.id === args.comment_id),
            args,
          );
          return { data: {} };
        },
        listComments: async () => ({ data: comments }),
      },
    },
    paginate: async (method, args) => (await method(args)).data,
  };
  const context = {
    repo: { owner: 'kubeflow', repo: 'pipelines' },
    eventName: 'issue_comment',
    payload: {
      action: 'created',
      issue: {
        number: 42,
        pull_request: { url: 'https://api.github.com/repos/kubeflow/pipelines/pulls/42' },
      },
      comment: structuredClone(command),
    },
  };
  return { github, context, pull, writes, comments, command };
}

function pullEvent(setup, action = 'synchronize') {
  setup.context.eventName = 'pull_request_target';
  setup.context.payload = { action, pull_request: structuredClone(setup.pull) };
  return setup;
}

test('pushes to an admitted issueless PR keep it open even with a stale event snapshot', async () => {
  const setup = pullEvent(harness());
  setup.pull.labels.push({ name: 'pr-gate/approved' });
  assert.equal((await handlePrGate(setup)).status, 'exempt');
  assert.equal(setup.pull.state, 'open');
  assert.deepEqual(setup.writes, []);
});

test('ordinary external PRs still close with contribution and /allow instructions', async () => {
  const setup = pullEvent(harness());
  assert.equal((await handlePrGate(setup)).status, 'closed');
  assert.equal(setup.pull.state, 'closed');
  assert.equal(setup.comments.length, 1);
  assert.match(setup.comments[0].body, /CONTRIBUTING\.md/);
  assert.match(setup.comments[0].body, /\/allow/);
});

test('/allow reopens a closed, unmerged PR after recording the exemption', async () => {
  const setup = harness({ state: 'closed', labels: ['do-not-merge/hold'] });
  const result = await handlePrGate(setup);
  assert.equal(result.status, 'allowed');
  assert.equal(setup.pull.state, 'open');
  assert.deepEqual(
    setup.pull.labels.map((label) => label.name),
    ['do-not-merge/hold', 'pr-gate/approved'],
  );
  assert.ok(
    setup.writes.findIndex(([kind]) => kind === 'labels') <
      setup.writes.findIndex(([kind]) => kind === 'pull'),
  );
  assert.match(setup.comments[0].body, /reopened/i);
});

test('/allow reopens an already admitted PR without duplicating its label', async () => {
  const setup = harness({ state: 'closed', labels: ['pr-gate/approved'] });
  await handlePrGate(setup);
  assert.equal(setup.pull.state, 'open');
  assert.equal(setup.writes.filter(([kind]) => kind === 'labels').length, 0);
});

test('/allow admits an issueless PR without granting CI authorization', async () => {
  const setup = harness();
  const result = await handlePrGate(setup);
  assert.equal(result.status, 'allowed');
  assert.equal(setup.pull.state, 'open');
  assert.deepEqual(setup.pull.labels, [{ name: 'pr-gate/approved' }]);
  assert.equal(setup.comments.length, 1);
  assert.match(setup.comments[0].body, /@maintainer/);
  assert.match(setup.comments[0].body, /CI authorization.*unchanged/s);
});

for (const permission of ['write', 'maintain', 'admin']) {
  test(`${permission} permission can admit a contribution`, async () => {
    const setup = harness({ permission });
    assert.equal((await handlePrGate(setup)).status, 'allowed');
  });
}

for (const permission of ['read', 'triage', 'none']) {
  test(`${permission} permission cannot use /allow, even with a trusted association`, async () => {
    const setup = harness({ permission, state: 'closed' });
    setup.context.payload.comment.author_association = 'OWNER';
    assert.equal((await handlePrGate(setup)).status, 'denied');
    assert.deepEqual(setup.writes, []);
    assert.equal(setup.pull.state, 'closed');
  });
}

for (const body of [
  '/allow please',
  '/allow\nthanks',
  '> /allow',
  '```\n/allow\n```',
  '/ALLOW',
  '/allowed',
]) {
  test(`does not interpret ${JSON.stringify(body)} as the command`, async () => {
    const setup = harness();
    setup.context.payload.comment.body = body;
    assert.equal((await handlePrGate(setup)).status, 'ignored');
    assert.deepEqual(setup.writes, []);
  });
}

test('surrounding whitespace is allowed on the standalone command', async () => {
  const setup = harness();
  setup.command.body = setup.context.payload.comment.body = '  /allow\r\n';
  assert.equal((await handlePrGate(setup)).status, 'allowed');
});

test('uses the current command author permission, not the event actor', async () => {
  const setup = harness();
  setup.context.actor = 'administrator';
  setup.context.payload.sender = { login: 'administrator' };
  setup.github.rest.repos.getCollaboratorPermissionLevel = async (args) => {
    assert.deepEqual(args, { owner: 'kubeflow', repo: 'pipelines', username: 'maintainer' });
    return { data: { permission: 'read' } };
  };
  assert.equal((await handlePrGate(setup)).status, 'denied');
  assert.deepEqual(setup.writes, []);
});

test('an edited-away command is not replayed from an old event', async () => {
  const setup = harness();
  setup.command.body = 'Never mind';
  assert.equal((await handlePrGate(setup)).status, 'ignored');
  assert.deepEqual(setup.writes, []);
});

test('ordinary issue comments, edited events, and bot comments are ignored', async () => {
  for (const variant of ['issue', 'edited', 'bot']) {
    const setup = harness();
    if (variant === 'issue') delete setup.context.payload.issue.pull_request;
    if (variant === 'edited') setup.context.payload.action = 'edited';
    if (variant === 'bot') setup.command.user.type = 'Bot';
    assert.equal((await handlePrGate(setup)).status, 'ignored');
    assert.deepEqual(setup.writes, []);
  }
});

test('never reopens a merged PR or labels it as admitted', async () => {
  const setup = harness({ state: 'closed' });
  setup.pull.merged = true;
  assert.equal((await handlePrGate(setup)).status, 'ignored');
  assert.deepEqual(setup.writes, []);
});

test('repeated commands do not duplicate labels or acknowledgement comments', async () => {
  const setup = harness({ state: 'closed' });
  await handlePrGate(setup);
  const writes = setup.writes.length;
  assert.equal((await handlePrGate(setup)).status, 'already-allowed');
  setup.command.id++;
  setup.context.payload.comment.id++;
  assert.equal((await handlePrGate(setup)).status, 'already-allowed');
  assert.equal(setup.writes.length, writes);
  assert.equal(setup.comments.length, 1);
});

test('permission lookup failure cannot grant approval or reopen a PR', async () => {
  const setup = harness({ state: 'closed' });
  setup.github.rest.repos.getCollaboratorPermissionLevel = async () => {
    throw new Error('GitHub unavailable');
  };
  await assert.rejects(handlePrGate(setup), /GitHub unavailable/);
  assert.deepEqual(setup.writes, []);
});

test('failure to record the exemption cannot reopen the PR', async () => {
  const setup = harness({ state: 'closed' });
  setup.github.rest.issues.addLabels = async () => {
    throw new Error('Label write denied');
  };
  await assert.rejects(handlePrGate(setup), /Label write denied/);
  assert.equal(setup.pull.state, 'closed');
  assert.equal(setup.comments.length, 0);
});

test('label creation tolerates another PR creating the same label concurrently', async () => {
  const setup = harness();
  let reads = 0;
  setup.github.rest.issues.getLabel = async () => {
    if (reads++ === 0) throw Object.assign(new Error('Not found'), { status: 404 });
    return { data: { name: 'pr-gate/approved' } };
  };
  setup.github.rest.issues.createLabel = async () => {
    throw Object.assign(new Error('Already exists'), { status: 422 });
  };
  assert.equal((await handlePrGate(setup)).status, 'allowed');
});

function withLinkedIssue(setup, labels, repository = 'kubeflow/pipelines') {
  setup.github.graphql = async () => ({
    repository: {
      pullRequest: {
        closingIssuesReferences: {
          nodes: [
            {
              number: 17,
              repository: { nameWithOwner: repository },
              labels: { nodes: labels.map((name) => ({ name })) },
            },
          ],
        },
      },
    },
  });
  return setup;
}

test('a local ready issue preserves the existing ok-to-test path', async () => {
  const setup = withLinkedIssue(pullEvent(harness()), ['ready']);
  assert.equal((await handlePrGate(setup)).status, 'ready');
  assert.deepEqual(setup.pull.labels, [{ name: 'ok-to-test' }]);
  assert.equal(setup.pull.state, 'open');
});

for (const [labels, repository] of [
  [['not-ready'], 'kubeflow/pipelines'],
  [['ready'], 'someone/else'],
  [[], 'kubeflow/pipelines'],
]) {
  test(`linked issue ${repository} with labels ${labels} does not satisfy admission`, async () => {
    const setup = withLinkedIssue(pullEvent(harness()), labels, repository);
    assert.equal((await handlePrGate(setup)).status, 'closed');
    assert.equal(setup.pull.state, 'closed');
  });
}

test('trusted members and the two exempt bot authors retain their exemptions', async () => {
  for (const association of ['MEMBER', 'OWNER', 'COLLABORATOR']) {
    const setup = pullEvent(harness());
    setup.pull.author_association = association;
    assert.equal((await handlePrGate(setup)).status, 'exempt');
    assert.deepEqual(setup.writes, []);
  }
  for (const author of ['dependabot[bot]', 'copybara-service[bot]']) {
    const setup = pullEvent(harness());
    setup.pull.user.login = author;
    assert.equal((await handlePrGate(setup)).status, 'exempt');
    assert.deepEqual(setup.writes, []);
  }
});

test('other bots and similar author names remain subject to admission', async () => {
  for (const author of ['renovate[bot]', 'dependabot-helper', 'copybara-service']) {
    const setup = pullEvent(harness());
    setup.pull.user.login = author;
    assert.equal((await handlePrGate(setup)).status, 'closed');
  }
});

test('removing the exemption re-enables the normal admission requirement', async () => {
  const setup = harness();
  await handlePrGate(setup);
  pullEvent(setup, 'unlabeled');
  setup.pull.labels = [];
  assert.equal((await handlePrGate(setup)).status, 'closed');
});

test('already closed PRs are not reopened by ordinary gate events', async () => {
  const setup = pullEvent(harness({ state: 'closed', labels: ['pr-gate/approved'] }));
  assert.equal((await handlePrGate(setup)).status, 'ignored');
  assert.deepEqual(setup.writes, []);
});

test('repeated admission checks reuse the bot closure comment', async () => {
  const setup = pullEvent(harness());
  await handlePrGate(setup);
  setup.pull.state = 'open';
  await handlePrGate(setup);
  assert.equal(setup.comments.length, 1);
});

test('a contributor cannot impersonate the bot closure comment with a marker', async () => {
  const setup = pullEvent(harness());
  setup.comments.push({
    id: 7,
    user: { login: 'contributor' },
    body: '<!-- kfp-pr-gate-admission --> spoofed',
  });
  await handlePrGate(setup);
  assert.equal(setup.comments[0].body, '<!-- kfp-pr-gate-admission --> spoofed');
  assert.equal(setup.comments.length, 2);
  assert.equal(
    setup.writes.some(([kind]) => kind === 'update-comment'),
    false,
  );
});

test('a lost closure response still reconciles a concurrent approval', async () => {
  const setup = pullEvent(harness());
  const updatePull = setup.github.rest.pulls.update;
  setup.github.rest.pulls.update = async (args) => {
    const result = await updatePull(args);
    if (args.state === 'closed') {
      setup.pull.labels.push({ name: 'pr-gate/approved' });
      throw new Error('Lost closure response');
    }
    return result;
  };
  await assert.rejects(handlePrGate(setup), /Lost closure response/);
  assert.equal(setup.pull.state, 'open');
});

test('an exemption granted during policy evaluation prevents closure', async () => {
  const setup = pullEvent(harness());
  const createComment = setup.github.rest.issues.createComment;
  setup.github.rest.issues.createComment = async (args) => {
    const result = await createComment(args);
    setup.pull.labels.push({ name: 'pr-gate/approved' });
    return result;
  };
  assert.equal((await handlePrGate(setup)).status, 'exempt');
  assert.equal(setup.pull.state, 'open');
  assert.equal(
    setup.writes.some(([kind]) => kind === 'pull'),
    false,
  );
});

test('a concurrent /allow between the final check and closure leaves the PR open', async () => {
  const setup = harness();
  const commandContext = structuredClone(setup.context);
  pullEvent(setup);
  const updatePull = setup.github.rest.pulls.update;
  setup.github.rest.pulls.update = async (args) => {
    if (args.state === 'closed') {
      assert.equal(
        (await handlePrGate({ github: setup.github, context: commandContext })).status,
        'allowed',
      );
    }
    return updatePull(args);
  };
  assert.equal((await handlePrGate(setup)).status, 'exempt');
  assert.equal(setup.pull.state, 'open');
  assert.deepEqual(setup.pull.labels, [{ name: 'pr-gate/approved' }]);
});
