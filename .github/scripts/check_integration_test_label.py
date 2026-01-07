#!/usr/bin/env python3
"""Script to check if integration-tests-verified label is present for main ->
stable merges.

Removes the label when new commits are pushed. Uses PyGithub SDK for
GitHub API interactions.
"""

import os
import sys

from github import Github


def has_integration_test_label(pull_request):
    """Check if the integration-tests-verified label is present."""
    labels = [label.name for label in pull_request.labels]
    return 'integration-tests-verified' in labels


def remove_integration_test_label(pull_request):
    """Remove the integration-tests-verified label if present."""
    try:
        pull_request.remove_from_labels('integration-tests-verified')
        return True
    except Exception as e:
        print(f'⚠️ Failed to remove integration-tests-verified label: {e}')
        return False


def post_instruction_comment(pull_request):
    """Post instruction comment with detailed testing steps."""
    instruction_comment = """## 🚦 Integration Test Verification Required

This pull request is merging **main** → **stable** and requires integration test verification.

### ✅ Required Action:
Comment `/integration-tests-ok` on this PR **only after** running the integration tests.

### 📝 Steps:

1. Run integration tests in OpenShift cluster with latest build
    1. Go to [this](https://jenkins-csb-rhods-opendatascience.dno.corp.redhat.com/job/devops/job/rhoai-test-flow/build?delay=0sec) Jenkins job
    2. Provide Cluster Name
    3. PRODUCT=RHODS (for RHOAI deployment) or ODS (for ODH deployment)
    4. CLUSTER_TYPE=selfmanaged
    4. TEST_ENVIRONMENT=AWS
    3. Check Install_Cluster checkbox
    4. Check "DEPROVISION_AFTER_INSTALL_FAILURE" if you want to destroy the cluster on job failure automatically
    5. Check "ADD_ICSP" is not checked
    6. Enter ODS_BUILD_URL=odh-nightly (for ODH deployment) or "latest build url from #rhoai-build-notifications w/o the https://" (for RHOAI deployment)
    7. Enter UPDATE_CHANNEL=odh-nightlies (for ODH deployment) or fast (for RHOAI deployment)
    8. Uncheck RUN_TESTS
    10. Run the job
    11. Once the deployment is DONE and your cluster is available, update the DSPO version by following the instructions [here](https://github.com/opendatahub-io/opendatahub-operator/blob/main/hack/component-dev/README.md)
    12. Deploy DSPA using the instructions provided in the comments
    13. On your local, run the Smoke tests via the following docker (or podman equivalent command):
        ```
        docker run -it -v /home/nsingla/.kube/config:/dspa/backend/test/.kube/config --rm --workdir=/dspa/backend/test --env AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" --env AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" --env NAMESPACE=$NAMESPACE_TO_CREATE --env DSPA_NAME=$SERVICE_NAME_OF_YOUR_CHOICE quay.io/opendatahub/ds-pipelines-tests::master --label-filter=smoke
        ```
    14. Now go back to Jenkins to run tests:
        1. This time uncheck "Install Cluster" checkbox
        2. Check RUN_TESTS checkbox
        3. Enter RUN_DASHBOARD_TESTS=SmokeSet1 (this will run cypress Dashboard Tests Job as a downstream job)
        4. Under "Components Tests Configuration", select "smoke" as quality gate and enable "ai-pipelines" tests (update the test image if you have any changes in the test code in this PR)
    15. Repeat the above steps but this time check `ENABLE_FIPS_IN_CLUSTER` checkbox to create a FIPS cluster
2. Comment `/integration-tests-ok` on this PR to add the verification label
3. This workflow check will automatically pass once the label is added

### 🔒 Authorization:
Only organization members and owners can use the `/integration-tests-ok` command.

---
*This requirement ensures production stability by verifying integration tests against the latest ODH nightly build.*"""

    # Check if instruction comment already exists
    comments = pull_request.get_issue_comments()
    for comment in comments:
        if ('Integration Test Verification Required' in comment.body and
                comment.user.type == 'Bot'):
            print('ℹ️ Instruction comment already exists')
            return

    # Post new comment
    try:
        pull_request.create_issue_comment(instruction_comment)
        print('✅ Posted instruction comment')
    except Exception as e:
        print(f'⚠️ Failed to post comment: {e}')


def main():
    """Main function to check integration test requirement."""
    # Get environment variables
    token = os.getenv('GITHUB_TOKEN')
    pr_number = os.getenv('PR_NUMBER')
    repo_owner = os.getenv('REPO_OWNER')
    repo_name = os.getenv('REPO_NAME')
    github_event_name = os.getenv('GITHUB_EVENT_NAME')
    github_event_action = os.getenv('GITHUB_EVENT_ACTION')

    if not all([token, pr_number, repo_owner, repo_name]):
        print('❌ Missing required environment variables')
        sys.exit(1)

    try:
        pr_number = int(pr_number)
    except ValueError:
        print(f'❌ Invalid PR number: {pr_number}')
        sys.exit(1)

    print(f'🔍 Checking PR #{pr_number} in {repo_owner}/{repo_name}')
    print(f'📝 Event: {github_event_name}, Action: {github_event_action}')

    # Initialize GitHub client
    try:
        github_client = Github(token)
        repo = github_client.get_repo(f'{repo_owner}/{repo_name}')
        pull_request = repo.get_pull(pr_number)
    except Exception as e:
        print(f'❌ Error accessing GitHub API: {e}')
        sys.exit(1)

    # If this is a synchronize event (new commits), remove the integration test label
    label_was_removed = False
    if github_event_action == 'synchronize':
        print(
            '🔄 New commits detected - checking for integration-tests-verified label'
        )

        if has_integration_test_label(pull_request):
            print(
                '🗑️ Removing integration-tests-verified label due to new commits'
            )

            if remove_integration_test_label(pull_request):
                print('✅ Successfully removed integration-tests-verified label')
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
                    print('✅ Posted label removal notification')
                except Exception as e:
                    print(f'⚠️ Failed to post removal comment: {e}')
            else:
                print('⚠️ Failed to remove label')
        else:
            print('ℹ️ No integration-tests-verified label found')

    # Refresh PR to get updated labels if label was removed
    if label_was_removed:
        try:
            pull_request = repo.get_pull(pr_number)
        except Exception as e:
            print(f'⚠️ Failed to refresh PR: {e}')

    # Check for integration test label
    has_label = has_integration_test_label(pull_request)

    if has_label:
        print('✅ Integration test label verified - merge can proceed')
        print('✅ Found integration-tests-verified label on PR')
        sys.exit(0)
    else:
        print('❌ Integration test verification required to merge to stable')

        if label_was_removed:
            print(
                '\n🚫 WORKFLOW FAILED: Label was automatically removed due to new commits'
            )
            print('📋 TO PASS THIS CHECK:')
            print(
                '   1. Re-run integration tests in OpenShift cluster with latest ODH nightly'
            )
            print(
                "   3. Comment '/integration-tests-ok' on this PR after tests pass"
            )
            print('   4. This workflow will automatically re-run and pass')
        else:
            # Post instruction comment for normal cases
            post_instruction_comment(pull_request)

            print(
                "\n📋 Required: Comment '/integration-tests-ok' on this PR after running integration tests"
            )
            print(
                '\n⚠️ Important: Only comment this after actually running the integration tests!'
            )

        sys.exit(1)


if __name__ == '__main__':
    main()
