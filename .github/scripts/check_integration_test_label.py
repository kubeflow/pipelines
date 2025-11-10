#!/usr/bin/env python3
"""
Script to check if integration-tests-verified label is present for main -> stable merges.
Removes the label when new commits are pushed.
Uses PyGithub SDK for GitHub API interactions.
"""

import os
import sys
from github import Github


def has_integration_test_label(pull_request):
    """Check if the integration-tests-verified label is present."""
    labels = [label.name for label in pull_request.labels]
    return "integration-tests-verified" in labels


def remove_integration_test_label(pull_request):
    """Remove the integration-tests-verified label if present."""
    try:
        pull_request.remove_from_labels("integration-tests-verified")
        return True
    except Exception as e:
        print(f"⚠️ Failed to remove integration-tests-verified label: {e}")
        return False


def post_instruction_comment(pull_request):
    """Post instruction comment with detailed testing steps."""
    instruction_comment = """## 🚦 Integration Test Verification Required

This pull request is merging **main** → **stable** and requires integration test verification.

### ✅ Required Action:
Comment `/integration-tests-ok` on this PR **only after** running the integration tests.

### 📝 Steps:
1. Run integration tests in OpenShift cluster with latest ODH nightly
    1. Fetch nightly ODH build information from **#odh-nightlies-notifications** slack channel
    2. Follow the instructions on [this](https://spaces.redhat.com/spaces/RHODS/pages/512757017/02+-+Jira+testing+and+Verification) page to create a cluster and deploy latest ODH
    3. Once the deployment is DONE and your cluster is available:
         * Login to openshift console
         * Go to Operator > Installed Operators > Open Data Hub Operator > Data Science Cluster > default-dsc
         * Open the yaml spec
         * Update the `aipipelines` section with:
         ```
           aipipelines:
                devFlags:
                    manifests:
                      - uri: https://github.com/opendatahub-io/data-science-pipelines-operator/tarball/main
                        contextDir: config
                        sourcePath: base
                managementState: Managed
         ```
         * Save and wait for DSPO to update
         * Deploy DSPA
         * Run [Iris Pipeline](https://github.com/red-hat-data-services/ods-ci/blob/master/ods_ci/tests/Resources/Files/pipeline-samples/v2/cache-disabled/iris_pipeline_compiled.yaml), [Flip Coin](https://github.com/red-hat-data-services/ods-ci/blob/master/ods_ci/tests/Resources/Files/pipeline-samples/v2/cache-disabled/flip_coin_compiled.yaml) pipelines
         * Make sure the pipeline runs Succeeds
2. Comment `/integration-tests-ok` on this PR to add the verification label
3. This workflow check will automatically pass once the label is added

### 🔒 Authorization:
Only organization members and owners can use the `/integration-tests-ok` command.

---
*This requirement ensures production stability by verifying integration tests against the latest ODH nightly build.*"""

    # Check if instruction comment already exists
    comments = pull_request.get_issue_comments()
    for comment in comments:
        if ("Integration Test Verification Required" in comment.body and
            comment.user.type == "Bot"):
            print("ℹ️ Instruction comment already exists")
            return

    # Post new comment
    try:
        pull_request.create_issue_comment(instruction_comment)
        print("✅ Posted instruction comment")
    except Exception as e:
        print(f"⚠️ Failed to post comment: {e}")


def main():
    """Main function to check integration test requirement."""
    # Get environment variables
    token = os.getenv("GITHUB_TOKEN")
    pr_number = os.getenv("PR_NUMBER")
    repo_owner = os.getenv("REPO_OWNER")
    repo_name = os.getenv("REPO_NAME")
    github_event_name = os.getenv("GITHUB_EVENT_NAME")
    github_event_action = os.getenv("GITHUB_EVENT_ACTION")

    if not all([token, pr_number, repo_owner, repo_name]):
        print("❌ Missing required environment variables")
        sys.exit(1)

    try:
        pr_number = int(pr_number)
    except ValueError:
        print(f"❌ Invalid PR number: {pr_number}")
        sys.exit(1)

    print(f"🔍 Checking PR #{pr_number} in {repo_owner}/{repo_name}")
    print(f"📝 Event: {github_event_name}, Action: {github_event_action}")

    # Initialize GitHub client
    try:
        github_client = Github(token)
        repo = github_client.get_repo(f"{repo_owner}/{repo_name}")
        pull_request = repo.get_pull(pr_number)
    except Exception as e:
        print(f"❌ Error accessing GitHub API: {e}")
        sys.exit(1)

    # If this is a synchronize event (new commits), remove the integration test label
    label_was_removed = False
    if github_event_action == "synchronize":
        print("🔄 New commits detected - checking for integration-tests-verified label")

        if has_integration_test_label(pull_request):
            print("🗑️ Removing integration-tests-verified label due to new commits")

            if remove_integration_test_label(pull_request):
                print("✅ Successfully removed integration-tests-verified label")
                label_was_removed = True

                # Post a comment explaining the removal
                removal_comment = """## 🔄 Integration Test Label Removed

New commits have been pushed to this PR. The `integration-tests-verified` label has been automatically removed to ensure tests are re-run with the latest changes.

**⚠️ This workflow check will now FAIL until you complete the steps below:**

**Next Steps to Pass This Check:**
1. Re-run integration tests in OpenShift cluster with latest ODH nightly
2. Fetch nightly build information from **#odh-nightlies-notifications** Slack channel
3. Comment `/integration-tests-ok` on this PR after confirming tests pass with the new commits

Once you comment `/integration-tests-ok`, the label will be re-added and this workflow will automatically pass."""

                try:
                    pull_request.create_issue_comment(removal_comment)
                    print("✅ Posted label removal notification")
                except Exception as e:
                    print(f"⚠️ Failed to post removal comment: {e}")
            else:
                print("⚠️ Failed to remove label")
        else:
            print("ℹ️ No integration-tests-verified label found")

    # Refresh PR to get updated labels if label was removed
    if label_was_removed:
        try:
            pull_request = repo.get_pull(pr_number)
        except Exception as e:
            print(f"⚠️ Failed to refresh PR: {e}")

    # Check for integration test label
    has_label = has_integration_test_label(pull_request)

    if has_label:
        print("✅ Integration test label verified - merge can proceed")
        print("✅ Found integration-tests-verified label on PR")
        sys.exit(0)
    else:
        print("❌ Integration test verification required to merge to stable")

        if label_was_removed:
            print("\n🚫 WORKFLOW FAILED: Label was automatically removed due to new commits")
            print("📋 TO PASS THIS CHECK:")
            print("   1. Re-run integration tests in OpenShift cluster with latest ODH nightly")
            print("   2. Fetch nightly build info from #odh-nightlies-notifications Slack channel")
            print("   3. Comment '/integration-tests-ok' on this PR after tests pass")
            print("   4. This workflow will automatically re-run and pass")
        else:
            # Post instruction comment for normal cases
            post_instruction_comment(pull_request)

            print("\n📋 Required: Comment '/integration-tests-ok' on this PR after running integration tests")
            print("\n⚠️ Important: Only comment this after actually running the integration tests!")

        sys.exit(1)


if __name__ == "__main__":
    main()