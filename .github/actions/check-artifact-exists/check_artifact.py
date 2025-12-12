#!/usr/bin/env python3
"""
GitHub Action script to check if an artifact exists in the repository for a specific commit.
Uses the GitHub REST API to list artifacts by name and filters by head SHA.
Returns true if the artifact exists, false otherwise.
"""

import os
import sys
import requests
import json

def get_github_token() -> str:
    """Get GitHub token from environment variables."""
    token = os.getenv('GITHUB_TOKEN')
    if not token:
        print("::error::GITHUB_TOKEN environment variable is required")
        sys.exit(1)
    return token

def get_head_sha() -> str:
    """Get the head SHA from environment variables."""
    head_sha = os.getenv('GITHUB_SHA')
    if not head_sha:
        print("::error::GITHUB_SHA environment variable is required")
        sys.exit(1)
    return head_sha

def get_repository() -> str:
    """Get the repository name from environment variables."""
    repo = os.getenv('GITHUB_REPOSITORY')
    if not repo:
        print("::error::GITHUB_REPOSITORY environment variable is required")
        sys.exit(1)
    return repo

def set_output(name: str, value: str) -> None:
    """Set GitHub Actions output."""
    github_output = os.getenv('GITHUB_OUTPUT')
    if github_output:
        with open(github_output, 'a') as f:
            f.write(f"{name}={value}\n")
    else:
        # Fallback for older GitHub Actions runner versions
        print(f"::set-output name={name}::{value}")

def check_artifact_exists(artifact_name: str, token: str, repo: str, head_sha: str) -> bool:
    """
    Check if an artifact exists in the repository for a specific commit.

    Args:
        artifact_name: Name of the artifact to check
        token: GitHub token for authentication
        repo: Repository in format 'owner/repo'
        head_sha: Commit SHA to filter artifacts by

    Returns:
        True if artifact exists, False otherwise
    """
    headers = {
        'Authorization': f'token {token}',
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
    }

    # GitHub API endpoint to list artifacts for a repository
    url = f"https://api.github.com/repos/{repo}/actions/artifacts"

    # Add query parameters - API will filter by artifact name
    params = {
        'per_page': 100,
        'name': artifact_name
    }

    max_iterations = 100

    try:
        page = 1
        while max_iterations > 0:
            params['page'] = page
            response = requests.get(url, headers=headers, params=params)

            if response.status_code == requests.codes.forbidden:
                print(f"::warning::GitHub API returned 403 when listing artifacts "
                      f"(name={artifact_name}, page={page}); treating as not found.")
                return False

            response.raise_for_status()

            artifacts_data = response.json()
            artifacts = artifacts_data.get('artifacts', [])
            total_count = artifacts_data.get('total_count', 0)

            # No artifacts found at all
            if not artifacts:
                print(f"::debug::No artifacts found on page '{page}' with name {artifact_name} matching head_sha {head_sha}")
                break

            # Check each artifact's head_sha since API filters by name already
            for artifact in artifacts:
                workflow_run = artifact.get('workflow_run', {})
                artifact_head_sha = workflow_run.get('head_sha')

                if artifact_head_sha == head_sha:
                    print(f"::debug::Artifact '{artifact_name}' found with ID {artifact.get('id')}, matching head_sha {head_sha}")
                    return True
                else:
                    print(f"::debug::Artifact '{artifact_name}' found but head_sha doesn't match (expected: {head_sha}, found: {artifact_head_sha})")

            # Check if we've reached the last page
            # If we've retrieved all artifacts (page * per_page >= total_count) or
            # if this page has fewer artifacts than requested, we're done
            if page * params['per_page'] >= total_count or len(artifacts) < params['per_page']:
                break

            page += 1
            max_iterations -= 1

        print(f"::debug::Artifact '{artifact_name}' with head_sha '{head_sha}' not found")
        return False

    except requests.exceptions.RequestException as e:
        print(f"::error::Failed to check artifact existence: {e}")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"::error::Failed to parse GitHub API response: {e}")
        sys.exit(1)

def main():
    """Main function."""
    # Get inputs
    artifact_name = os.getenv('INPUT_ARTIFACT_NAME')
    if not artifact_name:
        print("::error::artifact_name input is required")
        sys.exit(1)

    # Get environment variables
    token = get_github_token()
    repo = get_repository()
    head_sha = get_head_sha()

    print(f"::debug::Checking for artifact '{artifact_name}' in repository '{repo}', head_sha '{head_sha}'")

    # Check if artifact exists
    exists = check_artifact_exists(artifact_name, token, repo, head_sha)

    # Set outputs
    set_output('exists', str(exists).lower())

    print(f"::debug::Artifact {artifact_name} exists: {exists}")

if __name__ == '__main__':
    main()