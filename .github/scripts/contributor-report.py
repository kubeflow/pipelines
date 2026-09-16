#!/usr/bin/env python3

from __future__ import annotations

import base64
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any

REPORT_MARKER = "<!-- kfp-contributor-report -->"
KUBEFLOW_ORG_YAML_PATH = "github-orgs/kubeflow/org.yaml"
KUBEFLOW_INTERNAL_ACLS_REPO = "kubeflow/internal-acls"
KFP_REPO = "kubeflow/pipelines"
ALL_TIME_FROM = "2008-01-01T00:00:00Z"

CONTRIBUTOR_QUERY = """
query ContributorReport(
  $username: String!
  $issueCountQuery: String!
  $mergedPrQuery: String!
  $issueCommentCursor: String
) {
  user(login: $username) {
    login
    createdAt
    issueComments(
      first: 100
      after: $issueCommentCursor
      orderBy: { field: UPDATED_AT, direction: DESC }
    ) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        issue {
          url
          repository {
            owner {
              login
            }
            name
          }
        }
      }
    }
  }
  issuesOpened: search(query: $issueCountQuery, type: ISSUE_ADVANCED, first: 1) {
    issueCount
  }
  mergedPrs: search(query: $mergedPrQuery, type: ISSUE_ADVANCED, first: 1) {
    issueCount
  }
}
"""

REVIEW_QUERY = """
query ContributorReviewComments(
  $username: String!
  $from: DateTime!
  $to: DateTime!
  $reviewCursor: String
) {
  user(login: $username) {
    contributionsCollection(from: $from, to: $to) {
      pullRequestReviewContributions(first: 100, after: $reviewCursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          pullRequestReview {
            bodyText
            comments(first: 1) {
              totalCount
            }
            pullRequest {
              repository {
                owner {
                  login
                }
                name
              }
            }
          }
        }
      }
    }
  }
}
"""


@dataclass
class ContributorStats:
    created_at: str
    issues_opened: int
    merged_prs: int
    pr_comments: int
    pr_thread_comments: int
    pr_review_comments: int


def require_env(name: str) -> str:
    value = os.environ.get(name)
    if not value:
        raise RuntimeError(f"Missing required environment variable: {name}")
    return value


def parse_repo(full_name: str) -> tuple[str, str]:
    parts = full_name.split("/", 1)
    if len(parts) != 2 or not parts[0] or not parts[1]:
        raise RuntimeError(f"Invalid repository name: {full_name}")
    return parts[0], parts[1]


def github_request(path: str, method: str = "GET", body: Any | None = None) -> Any:
    token = require_env("GITHUB_TOKEN")
    api_url = os.environ.get("GITHUB_API_URL", "https://api.github.com")
    url = urllib.parse.urljoin(api_url.rstrip("/") + "/", path.lstrip("/"))
    data = None
    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
        "User-Agent": "kubeflow-pipelines-contributor-report",
    }
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode("utf-8")

    last_error: Exception | None = None
    for attempt in range(1, 5):
        request = urllib.request.Request(url, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(request) as response:
                payload = response.read().decode("utf-8")
                return json.loads(payload) if payload else None
        except urllib.error.HTTPError as error:
            if error.code in {502, 503, 504} and attempt < 4:
                time.sleep(0.5 * attempt)
                continue
            detail = error.read().decode("utf-8", errors="replace")
            raise RuntimeError(
                f"GitHub API {method} {url} failed: {error.code} {error.reason} {detail}"
            ) from error
        except Exception as error:  # network/transient errors
            last_error = error
            if attempt >= 4:
                break
            time.sleep(0.5 * attempt)

    if last_error is None:
        raise RuntimeError(f"GitHub API {method} {url} failed for an unknown reason")
    raise last_error


def github_graphql(query: str, variables: dict[str, Any]) -> dict[str, Any]:
    result = github_request("/graphql", method="POST", body={"query": query, "variables": variables})
    if result.get("errors"):
        raise RuntimeError(f"GitHub GraphQL failed: {json.dumps(result['errors'])}")
    return result["data"]


def parse_kubeflow_org_members(yaml_text: str) -> set[str]:
    members: set[str] = set()
    in_kubeflow_org = False
    current_list: str | None = None

    for raw_line in yaml_text.splitlines():
        line = raw_line.replace("\t", "    ")

        if not in_kubeflow_org:
            if line == "    kubeflow:":
                in_kubeflow_org = True
            continue

        if line == "        teams:":
            break
        if line == "        admins:":
            current_list = "admins"
            continue
        if line == "        members:":
            current_list = "members"
            continue
        if line.startswith("        ") and ":" in line and not line.startswith("        - "):
            current_list = None
            continue
        if current_list and line.startswith("        - "):
            members.add(line[len("        - ") :].strip().lower())

    return members


def fetch_kubeflow_org_members() -> set[str]:
    payload = github_request(f"/repos/{KUBEFLOW_INTERNAL_ACLS_REPO}/contents/{KUBEFLOW_ORG_YAML_PATH}")
    if payload.get("encoding") != "base64" or not payload.get("content"):
        raise RuntimeError("Unexpected response when fetching kubeflow org.yaml")
    yaml_text = base64.b64decode(payload["content"]).decode("utf-8")
    return parse_kubeflow_org_members(yaml_text)


def yearly_windows(created_at_iso: str) -> list[tuple[str, str]]:
    created_at = datetime.fromisoformat(created_at_iso.replace("Z", "+00:00"))
    now = datetime.now(timezone.utc)
    windows: list[tuple[str, str]] = []
    for year in range(created_at.year, now.year + 1):
        start = datetime(year, 1, 1, tzinfo=timezone.utc)
        end = datetime(year, 12, 31, 23, 59, 59, tzinfo=timezone.utc)
        if start < created_at:
            start = created_at
        if end > now:
            end = now
        windows.append((start.isoformat().replace("+00:00", "Z"), end.isoformat().replace("+00:00", "Z")))
    return windows


def count_review_comments(username: str, created_at_iso: str) -> int:
    review_comment_count = 0
    for start, end in yearly_windows(created_at_iso):
        review_cursor = None
        while True:
            data = github_graphql(
                REVIEW_QUERY,
                {
                    "username": username,
                    "from": start,
                    "to": end,
                    "reviewCursor": review_cursor,
                },
            )
            nodes = data["user"]["contributionsCollection"]["pullRequestReviewContributions"]["nodes"]
            for node in nodes:
                review = node.get("pullRequestReview")
                if not review:
                    continue
                repo = review["pullRequest"]["repository"]
                if repo["owner"]["login"] == "kubeflow" and repo["name"] == "pipelines":
                    review_comment_count += review["comments"]["totalCount"]
                    if review.get("bodyText", "").strip():
                        review_comment_count += 1
            page_info = data["user"]["contributionsCollection"]["pullRequestReviewContributions"]["pageInfo"]
            if not page_info["hasNextPage"]:
                break
            review_cursor = page_info["endCursor"]
    return review_comment_count


def fetch_contributor_stats(username: str) -> ContributorStats:
    issue_count_query = f"repo:{KFP_REPO} is:issue author:{username}"
    merged_pr_query = f"repo:{KFP_REPO} is:pr is:merged author:{username}"

    issue_comment_cursor = None
    issue_comment_count = 0
    created_at = None
    issues_opened = None
    merged_prs = None

    while True:
        data = github_graphql(
            CONTRIBUTOR_QUERY,
            {
                "username": username,
                "issueCountQuery": issue_count_query,
                "mergedPrQuery": merged_pr_query,
                "issueCommentCursor": issue_comment_cursor,
            },
        )
        user = data.get("user")
        if not user:
            raise RuntimeError(f"GitHub user not found: {username}")

        if created_at is None:
            created_at = user["createdAt"]
            issues_opened = data["issuesOpened"]["issueCount"]
            merged_prs = data["mergedPrs"]["issueCount"]

        for node in user["issueComments"]["nodes"]:
            issue = node.get("issue")
            if not issue:
                continue
            repo = issue["repository"]
            if repo["owner"]["login"] == "kubeflow" and repo["name"] == "pipelines" and "/pull/" in issue["url"]:
                issue_comment_count += 1

        page_info = user["issueComments"]["pageInfo"]
        if not page_info["hasNextPage"]:
            break
        issue_comment_cursor = page_info["endCursor"]

    assert created_at is not None
    assert issues_opened is not None
    assert merged_prs is not None
    review_comment_count = count_review_comments(username, created_at)

    return ContributorStats(
        created_at=created_at,
        issues_opened=issues_opened,
        merged_prs=merged_prs,
        pr_comments=issue_comment_count + review_comment_count,
        pr_thread_comments=issue_comment_count,
        pr_review_comments=review_comment_count,
    )


def format_age(created_at_iso: str) -> tuple[str, int]:
    created_at = datetime.fromisoformat(created_at_iso.replace("Z", "+00:00"))
    now = datetime.now(timezone.utc)
    age_days = (now - created_at).days
    return created_at.date().isoformat(), age_days


def build_comment(username: str, is_kubeflow_member: bool, stats: ContributorStats) -> str:
    created_date, age_days = format_age(stats.created_at)
    member_text = "Yes" if is_kubeflow_member else "No"
    return "\n".join(
        [
            REPORT_MARKER,
            "## Contributor Report",
            "",
            f"**User:** @{username}",
            "",
            "| Metric | Value | Notes |",
            "|---|---:|---|",
            (
                "| Kubeflow org member | "
                f"{member_text} | Source: "
                "[kubeflow/internal-acls org.yaml](https://github.com/kubeflow/internal-acls/blob/master/github-orgs/kubeflow/org.yaml) |"
            ),
            f"| Issues opened in {KFP_REPO} | {stats.issues_opened} | Authored GitHub issues only |",
            f"| Merged PRs in {KFP_REPO} | {stats.merged_prs} | Authored PRs with merged state |",
            f"| GitHub account age | {age_days} days | Created {created_date} |",
            (
                f"| PR comments in {KFP_REPO} | {stats.pr_comments} | "
                f"{stats.pr_thread_comments} PR thread comments + {stats.pr_review_comments} review comments |"
            ),
            "",
            "---",
            "<sub>This report is generated by a repo-local workflow using GitHub API data only and is safe to run on fork PRs.</sub>",
        ]
    )


def upsert_comment(owner: str, repo: str, issue_number: int, body: str) -> str:
    comments = github_request(f"/repos/{owner}/{repo}/issues/{issue_number}/comments?per_page=100")
    existing = next((comment for comment in comments if REPORT_MARKER in (comment.get("body") or "")), None)
    if existing:
        github_request(
            f"/repos/{owner}/{repo}/issues/comments/{existing['id']}",
            method="PATCH",
            body={"body": body},
        )
        return "updated"
    github_request(
        f"/repos/{owner}/{repo}/issues/{issue_number}/comments",
        method="POST",
        body={"body": body},
    )
    return "created"


def main() -> int:
    event_path = require_env("GITHUB_EVENT_PATH")
    with open(event_path, "r", encoding="utf-8") as handle:
        event = json.load(handle)
    pull_request = event.get("pull_request")
    if not pull_request:
        raise RuntimeError("This workflow requires a pull_request or pull_request_target event payload")

    username = pull_request.get("user", {}).get("login")
    if not username:
        raise RuntimeError("Could not determine pull request author login")

    owner, repo = parse_repo(require_env("GITHUB_REPOSITORY"))
    org_members = fetch_kubeflow_org_members()
    is_kubeflow_member = username.lower() in org_members
    stats = fetch_contributor_stats(username)
    comment = build_comment(username, is_kubeflow_member, stats)

    if os.environ.get("CONTRIBUTOR_REPORT_DRY_RUN") == "true":
        print(comment)
        return 0

    action = upsert_comment(owner, repo, pull_request["number"], comment)
    print(f"Contributor report {action} for #{pull_request['number']}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as error:  # pragma: no cover - workflow script entrypoint
        print(getattr(error, "stack", None) or str(error), file=sys.stderr)
        raise
