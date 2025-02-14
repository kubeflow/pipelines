# KFP -> DSP Rebase process

This document describes the process to upgrade Data Science Pipelines (DSP) code from a specific Kubeflow Pipelines (KFP) version tag. The following repositories must be updated, in the following order:

- https://github.com/opendatahub-io/data-science-pipelines
- https://github.com/opendatahub-io/argo-workflows
- https://github.com/opendatahub-io/data-science-pipelines-operator

## Checklist

- The new rebase branch has been created from the upstream tag
- The new rebase branch includes relevant carries from target branch
- The upstream tag is pushed to kubeflow/pipelines to ensure that build artifacts are versioned correctly

## Getting Started

### Data Science Pipelines repository

Preparing the local repo clone

Clone from a personal fork, and add the remote for upstream and opendatahub, fetching its branches:

 ```
git remote add --fetch kubeflow https://github.com/kubeflow/pipelines
git remote add --fetch opendatahub https://github.com/opendatahub-io/data-science-pipelines
```

### Creating a new local branch for the new rebase

Branch the target release to a new branch:

```
TAG=2.2.0
git checkout -b rebase-$TAG $TAG
```

Merge opendatahub(master) branch into the `rebase-\$TAG` branch with merge strategy ours. It discards all changes from the other branch (opendatahub/master) and create a merge commit. This leaves the content of your branch unchanged, and when you next merge with the other branch, Git will only consider changes made from this point forward. (Do not confuse this with ours conflict resolution strategy for recursive merge strategy, -X option.)

```
git merge opendatahub/master
```

This action will need to resolve some conflicts manually. Some recommentations when working in this task are:

* Dockerfiles are not expected to have any merge conflicts. We should have our dsp images stored in a separate path from the kfp ones.
* Any changes in generated files (go.mod, go.sum, package.json, package-lock.json) should always prioritize upstream changes.
* In case of changes in backend code that diverges completelly between kfp and dsp, you should use `git blame` to find the author(s) of the changes and work together to fix the conflicts. Do not try to fix by yourself, you are not alone in this.

After resolving all conflicts, remember to run a pre-flight check before going to the next task.

```
pre-commit
make unittest
make functest
```

### Create the Pull-Request in opendatahub-io/data-science-pipelines repository

Create a PR with the result of the previous tasks with the following description: `UPSTREAM <carry>: Rebase code to kfp x.y.z`

## Argo Workflows Repo

If the kfp code you are rebasing uses a newer Argo workflows version, you must update opendatahub-io/argo-workflows repository.

### Preparing the local repo clone

Clone from a personal fork, and add the remote for upstream and opendatahub, fetching its branches:

```
git clone git@github.com:<user id>/argo-workflows
git remote add --fetch argo https://github.com/argoproj/argo-workflows
git remote add --fetch opendatahub https://github.com/opendatahub-io/argo-workflows
```

### Creating a backup branch form the dsp repo

Argo Workflows git history diverges completely across versions, so it's important to create a backup branch from the current dsp repo in case we need to revert changes.

```
git fetch opendatahub main
git checkout -b dsp-backup opendatahub/main
```

**NOTE:** Keep this branch for as long time as possible after Argo rebase, just in case we need to revert some changes.

### Creating a new local branch for the new rebase

Branch the target release to a new branch:

```
TAG=v3.4.17
git checkout -b argo-upgrade $TAG
```

### Create the Pull-Request in opendatahub-io/argo-workflows repository

Create a PR with the result of the previous tasks with the following description: `Upgrade argo-workflows code to x.y.z`

## Data Science Pipelines Operator repository

### Apply the DataSciencePipelinesApplication CustomResource from the opendatahub-io/data-science-pipelines Pull-Request

With the Pull-Request opened in opendatahub-io/data-science-pipelines repository, you can get a DataSciencePipelinesApplication (DSPA)  CustomResource with the resulting image builds from the bot comment like this.

```
An OCP cluster where you are logged in as cluster admin is required.

The Data Science Pipelines team recommends testing this using the Data Science Pipelines Operator. Check here for more information on using the DSPO.

To use and deploy a DSP stack with these images (assuming the DSPO is deployed), first save the following YAML to a file named dspa.pr-76.yaml:

apiVersion: datasciencepipelinesapplications.opendatahub.io/v1alpha1
kind: DataSciencePipelinesApplication
metadata:
  name: pr-76
spec:
  dspVersion: v2
  apiServer:
    image: "quay.io/opendatahub/ds-pipelines-api-server:pr-76"
    argoDriverImage: "quay.io/opendatahub/ds-pipelines-driver:pr-76"
    argoLauncherImage: "quay.io/opendatahub/ds-pipelines-launcher:pr-76"
  persistenceAgent:
    image: "quay.io/opendatahub/ds-pipelines-persistenceagent:pr-76"
  scheduledWorkflow:
    image: "quay.io/opendatahub/ds-pipelines-scheduledworkflow:pr-76"
  mlmd:  
    deploy: true  # Optional component
    grpc:
      image: "quay.io/opendatahub/mlmd-grpc-server:latest"
    envoy:
      image: "registry.redhat.io/openshift-service-mesh/proxyv2-rhel8:2.3.9-2"
  mlpipelineUI:
    deploy: true  # Optional component 
    image: "quay.io/opendatahub/ds-pipelines-frontend:pr-76"
  objectStorage:
    minio:
      deploy: true
      image: 'quay.io/opendatahub/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance'
Then run the following:

cd $(mktemp -d)
git clone git@github.com:opendatahub-io/data-science-pipelines.git
cd data-science-pipelines/
git fetch origin pull/76/head
git checkout -b pullrequest f5a03d13022b1e1ba3ba09129e840633982522ac
oc apply -f dspa.pr-76.yaml
More instructions here on how to deploy and test a Data Science Pipelines Application.
```

### Fix the Data Science Pipelines Operator code

Check if there are any breaking changes, and fix the code whenever is needed
One obvious change would be the tag references in params.env file

### Create the Pull-Request in opendatahub-io/argo-workflows repository

Create a PR with the result of the previous tasks with the following description: `Upgrade argo-workflows code to x.y.z`

### Create a Draft/WIP Pull Request in opendatahub-io/argo-workflows repository

**NOTE**: This is only to show the diff/changeset and accept feedback from the team before proceeding. DO NOT ACTUALLY MERGE THIS

Create a PR with the result of the previous tasks with the following description: `Upgrade argo-workflows code to x.y.z`

### Force-push Version Upgrade branch to main

Upon acceptance of the Draft PR (again, do not actually merge this), force the `opendatahub/main` branch to now match the upgrade version 'feature' branch:

```bash
git push -f origin argo-upgrade:main
```

Obviously, this will completely overwrite the git history of the `opendatahub/main` remote branch so please ensure a backup branch (`dsp-backup`) was created as instructed above

### Disclaimer / Future Work

This process this obviously very heavy-handed and destructive, and depends on there being no carries or downstream-only commits.  We should adjust the procedure to account for this

## Followup work
QE team has a Jenkins Job that can help test some basic features. Ask Diego Lovison to trigger the Jenkins job. You might need to create a separate branch DSPO from the code rebase to change `params.env` file with the values of the generated images from the previous PRs to run this Jenkins job.

It is also good creating a follow-up task in JIRA to coordinate with QE to run some regression tests before merging the PRs.

## Updating with rebase.sh (experimental)
WIP

