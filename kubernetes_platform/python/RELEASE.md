## kfp-kubernetes release instructions

Some steps require elevated permissions to push branches, publish the package, and release documentation. However, anyone can perform step 1 to begin the process.

1.  [No permissions required] Update the package version in
    [`__init__.py`](https://github.com/kubeflow/pipelines/blob/master/kubernetes_platform/python/kfp/kubernetes/__init__.py)
    and add a documentation version to the
    [`version_info` array](https://github.com/kubeflow/pipelines/blob/0907a1155b393516b4f8de8561467dbb1f9be5da/kubernetes_platform/python/docs/conf.py#L140).

    **Create and merge the PR into the `master` branch.**

1.  [Requires repo OWNER permissions]  Replace the `KFP_KUBERNETES_VERSION` value with the version in
    `__init__.py`, then run the following commands:

    ```
    KFP_KUBERNETES_VERSION=0.0.1 # replace with correct version
    cd kubernetes_platform/python
    source create_release_branch.sh
    ```

1.  Follow the instructions printed out by the script in Step 2, which explain how to push the branch to upstream.

    By the end, you
    should have pushed a modified `__init__.py`, `conf.py` (from Step 1), and `.gitignore`, `kubernetes_executor_config_pb2.py` and two modified `.readthedocs.yml` files (from Step 2) to the release branch.

1.  [Requires credentials] From the `kubernetes_platform/python` directory with
    `KFP_KUBERNETES_VERSION` set, run:

    ```
    source release.sh
    ```
    To upload packages, you need an [API token](https://packaging.python.org/en/latest/guides/distributing-packages-using-setuptools/#create-an-account) (contact @chensun for help).
    Visit https://pypi.org/project/kfp-kubernetes/ and confirm the package was published.
    
1.  [Requires credentials] Go to
    [readthedocs.org/projects/kfp-kubernetes/](https://readthedocs.org/projects/kfp-kubernetes/) (contact @chensun for help),
    click "Versions" in the menu panel, and search for the correct branch to activate the version. Make sure the docs build.
    Visit the settings page and set the "Default branch" and "Default version" to the version you just released.

