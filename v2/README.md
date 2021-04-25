# KFP V2 Launcher

TODO: Add design documents here.

## Development

### Develop kfp launcher in your own project

1. Configure your own container registry for LAUNCHER_DEV_IMAGE:

```bash
export PROJECT=<my-project>
# .env is a Makefile local config (ignored by git)
echo "LAUNCHER_DEV_IMAGE=gcr.io/${PROJECT}/dev/kfp-launcher" > .env
```

2. Build launcher image locally and push to your own registry:

```bash
make dev-push-launcher
```

3. Configure sample tests to use your dev image:

```bash
export KFP_LAUNCHER_IMAGE=gcr.io/${PROJECT}/dev/kfp-launcher
# consider putting this in your .bashrc or .zshrc to persist it.
```

4. Run one sample test:

```bash
python -m samples.path.to.sample_test
```

Read [v2 sample test documentation](./test/README.md) for more details.
