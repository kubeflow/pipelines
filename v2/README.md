# KFP V2 Launcher

Design: [bit.ly/kfp-v2-compatible](https://bit.ly/kfp-v2-compatible)

## Develop KFP v2 compatible in your own project

1. Configure your own container registry for LAUNCHER_IMAGE_DEV:

    ```bash
    export PROJECT=<my-project>
    # .env is a Makefile local config (ignored by git)
    echo "DEV_IMAGE_PREFIX=gcr.io/${PROJECT}/dev/kfp-" > .env
    ```

1. Configure sample tests to use your dev image:

    ```bash
    export KFP_LAUNCHER_IMAGE=gcr.io/${PROJECT}/dev/kfp-launcher
    # consider putting this in your .bashrc or .zshrc to persist it.
    ```

1. Build launcher image locally and push to your own registry:

    ```bash
    make push-launcher-dev
    ```

1. Run one sample test:

    ```bash
    python -m samples.path.to.sample_test
    ```

    Read [v2 sample test documentation](./test/README.md) for more details.

## Develop KFP v2 in your own project

TODO(Bobgy): avoid hard coding image names and document how to do this.

### Update licenses

Download the license tool binary from <https://github.com/Bobgy/go-licenses/releases> and put it into $PATH.

Update licenses info by:

```bash
make license-launcher
```

or run the following to enable verbose output:

```bash
GO_LICENSES_FLAGS=-v4 make license-launcher
```

After the update, check generated third_party/licenses/launcher.csv file to
make sure licenses of new dependencies are correctly identified.

If something is unexpected, examine the unexpected dependencies by yourself and add
overrides to [go-licenses.yaml](./go-licenses.yaml).

For detailed documentation about the tool: <https://github.com/Bobgy/go-licenses/tree/main/v2>.
