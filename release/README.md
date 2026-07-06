# kfpr release CLI

`kfpr` is the Kubeflow Pipelines release automation CLI. It can run the full checkpointed release
flow or any individual release step.

## Quick start

From the repository root:

```bash
python3 -m pip install -e release
kfpr --help
kfpr doctor --release-type patch --version 3.2.1 --fork-remote USER
kfpr run --dry-run --state-file /tmp/kfp-release-state.json
```

Without installing, run from source:

```bash
PYTHONPATH=release python3 -m kfpr.cli --help
```

## Full release flow

```bash
kfpr run \
  --release-type patch \
  --version 3.2.1 \
  --fork-remote USER \
  --patch-prs 12345,12346 \
  --include-backend \
  --include-sdk
```

`run` stores prompt answers and completed steps in `release-state.json` by default. Pass
`--state-file` to use a different checkpoint file. If the checkpoint already exists, `run` prints
the saved status and asks before resuming; pass `--force` to resume without that prompt.

For major and minor releases, `run` asks which source branch to cut the release branch from and
defaults to `master`.

For `--fork-remote`, you can pass either a full remote URL or just your GitHub username. `kfpr`
expands `droctothorpe` to `git@github.com:droctothorpe/pipelines.git`.

## Individual steps

List the selected flow:

```bash
kfpr steps --release-type patch --include-backend --no-include-sdk
kfpr steps --diagram
```

Run one step without marking it complete:

```bash
kfpr update-version-tags \
  --release-type patch \
  --version 3.2.1 \
  --fork-remote USER \
  --patch-prs 12345,12346 \
  --dry-run
```

Mark a step complete after it succeeds:

```bash
kfpr update-version-tags \
  --release-type patch \
  --version 3.2.1 \
  --fork-remote USER \
  --patch-prs 12345,12346 \
  --mark-done
```

Single-step commands do not update the checkpoint unless `--mark-done` is passed.

If you complete a step outside `kfpr` (for example, manually creating an already-existing
release branch), mark that step done before resuming:

```bash
kfpr mark-done prepare-release-branch --state-file release-state.json
kfpr run --state-file release-state.json
```

Use the exact step ID from `kfpr steps` or the list below.

## Recovery helpers

```bash
kfpr status --state-file release-state.json
kfpr clear --state-file release-state.json
kfpr validate-state --state-file release-state.json
kfpr mark-done update-version-tags --state-file release-state.json
kfpr reset-step update-version-tags --state-file release-state.json
```

`status` prints saved answers, completed steps, the next step, the resume command, and the active
manual checklist when the next step needs external confirmation.

`confirm-rtd` prompts for a Read the Docs API token, keeps it only in process memory, and uses it
to activate release versions, trigger builds, wait for successful builds, and update project
defaults. If the API call fails, `kfpr` asks whether to fall back to the manual checklist.

## Commands

```text
run
doctor
status
clear
validate-state
mark-done
reset-step
steps
preflight
prepare-release-branch
prepare-patch-branch
cherry-pick-prs
merge-cherry-pick-pr
update-version-tags
merge-version-pr
publish-images
update-sdk-versions
merge-sdk-pr
create-sdk-release
publish-sdks
create-kfp-kubernetes-docs-branch
confirm-rtd
create-backend-release
sync-master
confirm-website-and-slack
```
