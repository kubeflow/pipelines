const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const {
  parseRootApprovers,
  reconcilePullRequest,
  shouldTrackAuthor,
} = require("./maintainer_project_label");

test("parses only root approvers and normalizes case", () => {
  const approvers = parseRootApprovers(`
reviewers:
  - reviewer-only
approvers:
  - chensun
  - HumairAK # mixed-case login
labels:
  - ignored
`);

  assert.deepEqual([...approvers], ["chensun", "humairak"]);
  assert.equal(shouldTrackAuthor("HumairAK", approvers), true);
  assert.equal(shouldTrackAuthor("reviewer-only", approvers), false);
});

test("always includes Dependabot", () => {
  const approvers = new Set(["chensun"]);
  assert.equal(shouldTrackAuthor("dependabot[bot]", approvers), true);
  assert.equal(shouldTrackAuthor("renovate[bot]", approvers), false);
});

test("fails closed for a missing or malformed approvers list", () => {
  assert.throws(() => parseRootApprovers("reviewers:\n  - chensun\n"), /non-empty/);
  assert.throws(() => parseRootApprovers("approvers:\n  chensun\n"), /Unsupported/);
  assert.throws(() => parseRootApprovers("approvers: [chensun]\n"), /non-empty/);
});

test("parses the repository root OWNERS file", () => {
  const owners = fs.readFileSync(path.join(__dirname, "../../../OWNERS"), "utf8");
  const approvers = parseRootApprovers(owners);
  assert.ok(approvers.size > 0);
  for (const login of approvers) {
    assert.match(login, /^[a-z0-9][a-z0-9-]{0,38}$/);
    assert.equal(login, login.toLowerCase());
  }
});

test("workflow uses a trusted event and least-privilege label permissions", () => {
  const workflow = fs.readFileSync(
    path.join(__dirname, "../../workflows/sync-maintainer-project-label.yml"),
    "utf8"
  );

  assert.match(workflow, /pull_request_target:\n    types:\n      - opened\n      - reopened\n      - synchronize/);
  const permissions = workflow.match(/^permissions:\n((?:  [^\n]+\n)+)/m);
  assert.ok(permissions, "workflow must declare its token permissions");
  assert.deepEqual(permissions[1].trim().split("\n").map((line) => line.trim()), [
    "contents: read",
    "issues: write",
    "pull-requests: read",
  ]);
  assert.match(workflow, /cron: '7,22,37,52 \* \* \* \*'/);
  assert.match(workflow, /uses: actions\/github-script@v9/);
  assert.match(workflow, /ref: \$\{\{ github\.event\.repository\.default_branch \}\}/);
  assert.doesNotMatch(workflow, /pull_request\.head|github\.head_ref/);
});

test("OWNERS changes run the parser and workflow tests", () => {
  const workflow = fs.readFileSync(
    path.join(__dirname, "../../workflows/ci-scripts-tests.yml"),
    "utf8"
  );
  const paths = workflow.match(/^  pull_request:\n    paths:\n((?:      - [^\n]+\n)+)/m);
  assert.ok(paths, "CI script tests must declare pull-request path filters");
  assert.match(paths[1], /^      - ['"]?OWNERS['"]?$/m);
});

test("reconciles maintainer tracking labels", async () => {
  for (const [eligible, labeled, expected] of [
    [true, false, "added project/maintainer-review"],
    [true, true, "unchanged"],
    [false, false, "unchanged"],
    [false, true, "removed project/maintainer-review"],
  ]) {
    const calls = [];
    const github = {
      rest: {
        issues: {
          addLabels: async (request) => calls.push(["add", request]),
          removeLabel: async (request) => calls.push(["remove", request]),
        },
      },
    };
    const result = await reconcilePullRequest({
      github,
      owner: "kubeflow",
      repo: "pipelines",
      pullRequest: {
        number: 42,
        user: {login: eligible ? "chensun" : "contributor"},
        labels: labeled ? [{name: "project/maintainer-review"}] : [],
      },
      approvers: new Set(["chensun"]),
      dependabotLabel: "project/dependabot-review",
      trackingLabel: "project/maintainer-review",
    });

    assert.equal(result, expected);
    assert.equal(calls.length, expected === "unchanged" ? 0 : 1);
    if (expected.startsWith("added")) assert.equal(calls[0][0], "add");
    if (expected.startsWith("removed")) assert.equal(calls[0][0], "remove");
  }
});

test("keeps the Dependabot-only label exclusive to Dependabot", async () => {
  const calls = [];
  const github = {
    rest: {
      issues: {
        addLabels: async (request) => calls.push(["add", request.labels[0]]),
        removeLabel: async (request) => calls.push(["remove", request.name]),
      },
    },
  };

  const added = await reconcilePullRequest({
    github,
    owner: "kubeflow",
    repo: "pipelines",
    pullRequest: {number: 1, user: {login: "dependabot[bot]"}, labels: []},
    approvers: new Set(),
    dependabotLabel: "project/dependabot-review",
    trackingLabel: "project/maintainer-review",
  });
  assert.equal(
    added,
    "added project/maintainer-review, added project/dependabot-review"
  );

  const removed = await reconcilePullRequest({
    github,
    owner: "kubeflow",
    repo: "pipelines",
    pullRequest: {
      number: 2,
      user: {login: "chensun"},
      labels: [
        {name: "project/maintainer-review"},
        {name: "project/dependabot-review"},
      ],
    },
    approvers: new Set(["chensun"]),
    dependabotLabel: "project/dependabot-review",
    trackingLabel: "project/maintainer-review",
  });
  assert.equal(removed, "removed project/dependabot-review");
});
