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
  assert.deepEqual([...parseRootApprovers(owners)], [
    "chensun",
    "droctothorpe",
    "humairak",
    "jeffspahr",
    "mprahl",
    "zazulam",
  ]);
});

test("workflow uses a trusted event and least-privilege label permissions", () => {
  const workflow = fs.readFileSync(
    path.join(__dirname, "../../workflows/sync-maintainer-project-label.yml"),
    "utf8"
  );

  assert.match(workflow, /pull_request_target:\n    types:\n      - opened\n      - reopened\n      - synchronize/);
  assert.match(workflow, /permissions:\n  contents: read\n  issues: write/);
  assert.match(workflow, /uses: actions\/github-script@v9/);
  assert.match(workflow, /ref: \$\{\{ github\.event\.repository\.default_branch \}\}/);
  assert.doesNotMatch(workflow, /pull_request\.head|github\.head_ref/);
});

test("reconciles each eligibility and current-label state", async () => {
  for (const [eligible, labeled, expected] of [
    [true, false, "added"],
    [true, true, "unchanged"],
    [false, false, "unchanged"],
    [false, true, "removed"],
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
      trackingLabel: "project/maintainer-review",
    });

    assert.equal(result, expected);
    assert.equal(calls.length, expected === "unchanged" ? 0 : 1);
    if (expected === "added") assert.equal(calls[0][0], "add");
    if (expected === "removed") assert.equal(calls[0][0], "remove");
  }
});
