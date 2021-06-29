# KFP V2 Launcher

TODO: Add design documents here.

## Develop kfp launcher in your own project

1. Configure your own container registry for LAUNCHER_IMAGE_DEV:

    ```bash
    export PROJECT=<my-project>
    # .env is a Makefile local config (ignored by git)
    echo "LAUNCHER_IMAGE_DEV=gcr.io/${PROJECT}/dev/kfp-launcher" > .env
    ```

1. Configure sample tests to use your dev image:

    ```bash
    export KFP_LAUNCHER_IMAGE=gcr.io/${PROJECT}/dev/kfp-launcher
    # consider putting this in your .bashrc or .zshrc to persist it.
    ```

1. Build launcher image locally and push to your own registry:

    ```bash
    make dev-push-launcher
    ```

1. Run one sample test:

    ```bash
    python -m samples.path.to.sample_test
    ```

    Read [v2 sample test documentation](./test/README.md) for more details.
