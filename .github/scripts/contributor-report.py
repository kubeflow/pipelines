#!/usr/bin/env python3

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from datetime import timezone
import json
import os
import sys
import time
from typing import Any
import urllib.error
import urllib.parse
import urllib.request

from kubeflow_membership import is_kubeflow_member

REPORT_MARKER = "<!-- kfp-contributor-report -->"
KFP_REPO = "kubeflow/pipelines"
TRANSIENT_HTTP_STATUSES = {502, 503, 504}
TRANSIENT_ATTEMPTS = 4

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


@dataclass(frozen=True)
class ContributorStats:
    created_at: str
    issues_opened: int
    merged_prs: int
    pr_comments: int


@dataclass(frozen=True)
class MarkdownRow:
    metric: str
    value: str


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


def is_human_user(author: dict[str, Any]) -> bool:
    return author.get("type", "User") == "User"


def github_request(path: str,
                   method: str = "GET",
                   body: Any | None = None) -> Any:
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
    for attempt in range(1, TRANSIENT_ATTEMPTS + 1):
        request = urllib.request.Request(
            url, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(request) as response:
                payload = response.read().decode("utf-8")
                return json.loads(payload) if payload else None
        except urllib.error.HTTPError as error:
            if error.code in TRANSIENT_HTTP_STATUSES and attempt < TRANSIENT_ATTEMPTS:
                time.sleep(0.5 * attempt)
                continue
            detail = error.read().decode("utf-8", errors="replace")
            raise RuntimeError(
                f"GitHub API {method} {url} failed: {error.code} {error.reason} {detail}"
            ) from error
        except urllib.error.URLError as error:
            last_error = error
            if attempt >= TRANSIENT_ATTEMPTS:
                break
            time.sleep(0.5 * attempt)

    if last_error is None:
        raise RuntimeError(
            f"GitHub API {method} {url} failed for an unknown reason")
    raise last_error


def github_graphql(query: str, variables: dict[str, Any]) -> dict[str, Any]:
    result = github_request(
        "/graphql",
        method="POST",
        body={
            "query": query,
            "variables": variables,
        },
    )
    if result.get("errors"):
        raise RuntimeError(
            f"GitHub GraphQL failed: {json.dumps(result['errors'])}")
    return result["data"]


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
            if repo["owner"]["login"] == "kubeflow" and repo[
                    "name"] == "pipelines" and "/pull/" in issue["url"]:
                issue_comment_count += 1

        page_info = user["issueComments"]["pageInfo"]
        if not page_info["hasNextPage"]:
            break
        issue_comment_cursor = page_info["endCursor"]

    assert created_at is not None
    assert issues_opened is not None
    assert merged_prs is not None

    return ContributorStats(
        created_at=created_at,
        issues_opened=issues_opened,
        merged_prs=merged_prs,
        pr_comments=issue_comment_count,
    )


def format_age(created_at_iso: str) -> tuple[str, int]:
    created_at = datetime.fromisoformat(created_at_iso.replace("Z", "+00:00"))
    now = datetime.now(timezone.utc)
    age_days = (now - created_at).days
    return created_at.date().isoformat(), age_days


def render_table(rows: list[MarkdownRow]) -> list[str]:
    return [
        "| Metric | Value |",
        "|---|---:|",
        *[f"| {row.metric} | {row.value} |" for row in rows],
    ]


def build_user_rows(is_kubeflow_member: bool,
                    stats: ContributorStats) -> list[MarkdownRow]:
    created_date, age_days = format_age(stats.created_at)
    member_text = "Yes" if is_kubeflow_member else "No"
    return [
        MarkdownRow(
            metric="Kubeflow org member",
            value=member_text,
        ),
        MarkdownRow(
            metric=f"Issues opened in {KFP_REPO}",
            value=str(stats.issues_opened),
        ),
        MarkdownRow(
            metric=f"Merged PRs in {KFP_REPO}",
            value=str(stats.merged_prs),
        ),
        MarkdownRow(
            metric="GitHub account age",
            value=f"{age_days} days (created {created_date})",
        ),
        MarkdownRow(
            metric=f"PR comments in {KFP_REPO}",
            value=str(stats.pr_comments),
        ),
    ]


def build_non_user_rows(author_type: str) -> list[MarkdownRow]:
    reason = "N/A (non-human GitHub author)"
    return [
        MarkdownRow(metric="GitHub author type", value=author_type),
        MarkdownRow(metric="Kubeflow org member", value=reason),
        MarkdownRow(metric=f"Issues opened in {KFP_REPO}", value=reason),
        MarkdownRow(metric=f"Merged PRs in {KFP_REPO}", value=reason),
        MarkdownRow(metric="GitHub account age", value=reason),
        MarkdownRow(metric=f"PR comments in {KFP_REPO}", value=reason),
    ]


def build_comment(username: str,
                  rows: list[MarkdownRow],
                  title: str | None = None) -> str:
    resolved_title = title or f"#### Contributor Report for @{username}"
    return "\n".join([
        REPORT_MARKER,
        resolved_title,
        "",
        *render_table(rows),
    ])


def list_issue_comments(owner: str, repo: str,
                        issue_number: int) -> list[dict[str, Any]]:
    comments: list[dict[str, Any]] = []
    page = 1
    while True:
        page_comments = github_request(
            f"/repos/{owner}/{repo}/issues/{issue_number}/comments?per_page=100&page={page}"
        )
        comments.extend(page_comments)
        if len(page_comments) < 100:
            break
        page += 1
    return comments


def is_report_comment(comment: dict[str, Any]) -> bool:
    return (REPORT_MARKER in (comment.get("body") or "") and
            comment.get("user", {}).get("type") == "Bot")


def upsert_comment(owner: str, repo: str, issue_number: int, body: str) -> str:
    comments = list_issue_comments(owner, repo, issue_number)
    existing = next(
        (comment for comment in comments if is_report_comment(comment)), None)
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
        raise RuntimeError(
            "This workflow requires a pull_request or pull_request_target event payload"
        )

    author = pull_request.get("user", {})
    username = author.get("login")
    if not username:
        raise RuntimeError("Could not determine pull request author login")

    if is_human_user(author):
        rows = build_user_rows(
            is_kubeflow_member(username), fetch_contributor_stats(username))
    else:
        rows = build_non_user_rows(author.get("type", "Unknown"))

    comment = build_comment(username, rows)

    if os.environ.get("CONTRIBUTOR_REPORT_DRY_RUN") == "true":
        print(comment)
        return 0

    owner, repo = parse_repo(require_env("GITHUB_REPOSITORY"))
    action = upsert_comment(owner, repo, pull_request["number"], comment)
    print(f"Contributor report {action} for #{pull_request['number']}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as error:  # pragma: no cover - workflow script entrypoint
        print(str(error), file=sys.stderr)
        raise
