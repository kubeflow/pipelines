## kfp-kubernetes release instructions

Some steps require elevated permissions to push branches, publish the package, and release documentation. However, anyone can perform step 1 to begin the process.

1.  [No permissions required] Update the package version in
    [`__init__.py`](https://github.com/kubeflow/pipelines/blob/master/kubernetes_platform/python/kfp/kubernetes/__init__.py)
    and add a documentation version to the
    [`version_info` array](https://github.com/kubeflow/pipelines/blob/0907a1155b393516b4f8de8561467dbb1f9be5da/kubernetes_platform/python/docs/conf.py#L140).

    **Create and merge the PR into the `master` branch.**

2.  [Requires repo OWNER permissions]  Replace the `KFP_KUBERNETES_VERSION` value with the version in
    `__init__.py`, then run the following commands:

    ```
    KFP_KUBERNETES_VERSION=0.0.1 # replace with correct version
    cd kubernetes_platform/python
    source create_release_branch.sh
    ```

    Follow the instructions printed out by the script in Step 2, which explain how to push the branch to upstream.

    By the end, you
    should have pushed a modified `__init__.py`, `conf.py`, `.gitignore`, and
    two modified `.readthedocs.yml` files to the release branch.

4.  [Requires credentials] Go to
    [readthedocs.org/projects/kfp-kubernetes/](https://readthedocs.org/projects/kfp-kubernetes/),
    click "Versions" in the menu panel, and
    search for the correct branch to activate the version. Make sure the docs
    build.

5.  [Requires credentials] From the `kubernetes_platform/python` directory with
    `KFP_KUBERNETES_VERSION` set, run:

    ```
    source release.sh
    ```
