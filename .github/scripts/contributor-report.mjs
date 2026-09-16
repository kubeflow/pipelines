#!/usr/bin/env node

import fs from 'node:fs/promises'

const REPORT_MARKER = '<!-- kfp-contributor-report -->'
const KUBEFLOW_ORG_YAML_PATH = 'github-orgs/kubeflow/org.yaml'
const KUBEFLOW_INTERNAL_ACLS_REPO = 'kubeflow/internal-acls'
const KFP_REPO = 'kubeflow/pipelines'
const ALL_TIME_FROM = '2008-01-01T00:00:00Z'

function requireEnv(name) {
  const value = process.env[name]
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`)
  }
  return value
}

function parseRepo(fullName) {
  const [owner, repo] = fullName.split('/')
  if (!owner || !repo) {
    throw new Error(`Invalid repository name: ${fullName}`)
  }
  return { owner, repo }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function githubRequest(path, { method = 'GET', body, accept = 'application/vnd.github+json' } = {}) {
  const token = requireEnv('GITHUB_TOKEN')
  const apiUrl = process.env.GITHUB_API_URL || 'https://api.github.com'
  const url = new URL(path, apiUrl)

  const headers = {
    Authorization: `Bearer ${token}`,
    Accept: accept,
    'User-Agent': 'kubeflow-pipelines-contributor-report',
  }

  const init = {
    method,
    headers,
  }

  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  let lastError
  for (let attempt = 1; attempt <= 4; attempt += 1) {
    try {
      const response = await fetch(url, init)
      if (response.ok) {
        const text = await response.text()
        return text ? JSON.parse(text) : null
      }

      if ([502, 503, 504].includes(response.status) && attempt < 4) {
        await sleep(500 * attempt)
        continue
      }

      const text = await response.text()
      throw new Error(`GitHub API ${method} ${url} failed: ${response.status} ${response.statusText} ${text}`)
    } catch (error) {
      lastError = error
      if (attempt >= 4) {
        break
      }
      await sleep(500 * attempt)
    }
  }

  throw lastError
}

async function githubGraphql(query, variables) {
  const result = await githubRequest('/graphql', {
    method: 'POST',
    body: { query, variables },
  })

  if (result.errors?.length) {
    throw new Error(`GitHub GraphQL failed: ${JSON.stringify(result.errors)}`)
  }

  return result.data
}

function parseKubeflowOrgMembers(yamlText) {
  const members = new Set()
  let inKubeflowOrg = false
  let currentList = null

  for (const rawLine of yamlText.split(/\r?\n/)) {
    const line = rawLine.replace(/\t/g, '    ')

    if (!inKubeflowOrg) {
      if (/^\s{4}kubeflow:\s*$/.test(line)) {
        inKubeflowOrg = true
      }
      continue
    }

    if (/^\s{8}teams:\s*$/.test(line)) {
      break
    }

    if (/^\s{8}admins:\s*$/.test(line)) {
      currentList = 'admins'
      continue
    }

    if (/^\s{8}members:\s*$/.test(line)) {
      currentList = 'members'
      continue
    }

    if (/^\s{8}[A-Za-z_][^:]*:\s*/.test(line)) {
      currentList = null
      continue
    }

    if (currentList && /^\s{8}-\s+(.+?)\s*$/.test(line)) {
      const match = line.match(/^\s{8}-\s+(.+?)\s*$/)
      members.add(match[1].trim().toLowerCase())
    }
  }

  return members
}

async function fetchKubeflowOrgMembers() {
  const { content, encoding } = await githubRequest(
    `/repos/${KUBEFLOW_INTERNAL_ACLS_REPO}/contents/${KUBEFLOW_ORG_YAML_PATH}`,
  )
  if (encoding !== 'base64' || !content) {
    throw new Error('Unexpected response when fetching kubeflow org.yaml')
  }
  const yamlText = Buffer.from(content, 'base64').toString('utf8')
  return parseKubeflowOrgMembers(yamlText)
}

const CONTRIBUTOR_QUERY = `
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
`

const REVIEW_QUERY = `
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
`

function* yearlyWindows(createdAtIso) {
  const createdAt = new Date(createdAtIso)
  const now = new Date()
  let year = createdAt.getUTCFullYear()
  const endYear = now.getUTCFullYear()

  while (year <= endYear) {
    const from = new Date(Date.UTC(year, 0, 1, 0, 0, 0))
    const to = new Date(Date.UTC(year, 11, 31, 23, 59, 59))
    if (from < createdAt) {
      from.setTime(createdAt.getTime())
    }
    if (to > now) {
      to.setTime(now.getTime())
    }
    yield {
      from: from.toISOString(),
      to: to.toISOString(),
    }
    year += 1
  }
}

async function countReviewComments(username, createdAtIso) {
  let reviewCommentCount = 0

  for (const window of yearlyWindows(createdAtIso)) {
    let reviewCursor = null
    while (true) {
      const data = await githubGraphql(REVIEW_QUERY, {
        username,
        from: window.from,
        to: window.to,
        reviewCursor,
      })

      for (const node of data.user.contributionsCollection.pullRequestReviewContributions.nodes) {
        const review = node.pullRequestReview
        if (
          review?.pullRequest?.repository?.owner?.login === 'kubeflow' &&
          review?.pullRequest?.repository?.name === 'pipelines'
        ) {
          reviewCommentCount += review.comments.totalCount
          if (review.bodyText?.trim()) {
            reviewCommentCount += 1
          }
        }
      }

      const pageInfo = data.user.contributionsCollection.pullRequestReviewContributions.pageInfo
      if (!pageInfo.hasNextPage) {
        break
      }
      reviewCursor = pageInfo.endCursor
    }
  }

  return reviewCommentCount
}

async function fetchContributorStats(username) {
  const issueCountQuery = `repo:${KFP_REPO} is:issue author:${username}`
  const mergedPrQuery = `repo:${KFP_REPO} is:pr is:merged author:${username}`

  let issueCommentCursor = null
  let issueCommentCount = 0
  let createdAt = null
  let issuesOpened = null
  let mergedPrs = null

  while (true) {
    const data = await githubGraphql(CONTRIBUTOR_QUERY, {
      username,
      issueCountQuery,
      mergedPrQuery,
      issueCommentCursor,
    })

    if (!data.user) {
      throw new Error(`GitHub user not found: ${username}`)
    }

    createdAt = createdAt ?? data.user.createdAt
    issuesOpened = issuesOpened ?? data.issuesOpened.issueCount
    mergedPrs = mergedPrs ?? data.mergedPrs.issueCount

    for (const node of data.user.issueComments.nodes) {
      const issue = node.issue
      if (
        issue?.repository?.owner?.login === 'kubeflow' &&
        issue?.repository?.name === 'pipelines' &&
        issue?.url?.includes('/pull/')
      ) {
        issueCommentCount += 1
      }
    }

    const issuePageInfo = data.user.issueComments.pageInfo
    if (!issuePageInfo.hasNextPage) {
      break
    }
    issueCommentCursor = issuePageInfo.endCursor
  }

  const reviewCommentCount = await countReviewComments(username, createdAt)

  return {
    createdAt,
    issuesOpened,
    mergedPrs,
    prComments: issueCommentCount + reviewCommentCount,
    prThreadComments: issueCommentCount,
    prReviewComments: reviewCommentCount,
  }
}

function formatAge(createdAtIso) {
  const createdAt = new Date(createdAtIso)
  const now = new Date()
  const ageDays = Math.floor((now.getTime() - createdAt.getTime()) / (1000 * 60 * 60 * 24))
  return {
    createdAt: createdAt.toISOString().slice(0, 10),
    ageDays,
  }
}

function buildComment({ username, isKubeflowMember, stats }) {
  const age = formatAge(stats.createdAt)
  const memberText = isKubeflowMember ? 'Yes' : 'No'

  return [
    REPORT_MARKER,
    '## Contributor Report',
    '',
    `**User:** @${username}`,
    '',
    '| Metric | Value | Notes |',
    '|---|---:|---|',
    `| Kubeflow org member | ${memberText} | Source: [kubeflow/internal-acls org.yaml](https://github.com/kubeflow/internal-acls/blob/master/github-orgs/kubeflow/org.yaml) |`,
    `| Issues opened in ${KFP_REPO} | ${stats.issuesOpened} | Authored GitHub issues only |`,
    `| Merged PRs in ${KFP_REPO} | ${stats.mergedPrs} | Authored PRs with merged state |`,
    `| GitHub account age | ${age.ageDays} days | Created ${age.createdAt} |`,
    `| PR comments in ${KFP_REPO} | ${stats.prComments} | ${stats.prThreadComments} PR thread comments + ${stats.prReviewComments} review comments |`,
    '',
    '---',
    '<sub>This report is generated by a repo-local workflow using GitHub API data only and is safe to run on fork PRs.</sub>',
  ].join('\n')
}

async function upsertComment({ owner, repo, issueNumber, body }) {
  const comments = await githubRequest(`/repos/${owner}/${repo}/issues/${issueNumber}/comments?per_page=100`)
  const existing = comments.find((comment) => comment.body?.includes(REPORT_MARKER))

  if (existing) {
    await githubRequest(`/repos/${owner}/${repo}/issues/comments/${existing.id}`, {
      method: 'PATCH',
      body: { body },
    })
    return 'updated'
  }

  await githubRequest(`/repos/${owner}/${repo}/issues/${issueNumber}/comments`, {
    method: 'POST',
    body: { body },
  })
  return 'created'
}

async function main() {
  const eventPath = requireEnv('GITHUB_EVENT_PATH')
  const event = JSON.parse(await fs.readFile(eventPath, 'utf8'))
  const pullRequest = event.pull_request
  if (!pullRequest) {
    throw new Error('This workflow requires a pull_request or pull_request_target event payload')
  }

  const username = pullRequest.user?.login
  if (!username) {
    throw new Error('Could not determine pull request author login')
  }

  const { owner, repo } = parseRepo(requireEnv('GITHUB_REPOSITORY'))
  const orgMembers = await fetchKubeflowOrgMembers()
  const isKubeflowMember = orgMembers.has(username.toLowerCase())
  const stats = await fetchContributorStats(username)
  const comment = buildComment({ username, isKubeflowMember, stats })

  if (process.env.CONTRIBUTOR_REPORT_DRY_RUN === 'true') {
    console.log(comment)
    return
  }

  const action = await upsertComment({
    owner,
    repo,
    issueNumber: pullRequest.number,
    body: comment,
  })
  console.log(`Contributor report ${action} for #${pullRequest.number}`)
}

main().catch((error) => {
  console.error(error.stack || String(error))
  process.exit(1)
})
