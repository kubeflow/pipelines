# Testing and formatting

Run the full retained SDK suite, not only the small deployment corpus:

```sh
uv sync --frozen --extra test --extra lint --extra codegen
uv run python -m pytest -q sdk/python/kfp test samples/argo-native/deploy_test.py
uv run ruff check sdk/python/kfp samples/argo-native hack/generate-python.py test
uv run ruff format --check sdk/python/kfp samples/argo-native hack/generate-python.py test
```

Repeat SDK validation on Python 3.11 and 3.13. Build a wheel and sdist; verify the
wheel includes the native runtime and both schema bindings, and excludes retired
client/runner packages. Generated bindings are committed so installing a wheel
requires neither protoc nor a backend.

Frontend, with the pinned Node/npm versions:

```sh
cd frontend
npm ci
npm run test:ci
```

The native production-bundle test checks both entrypoints, API boundaries and
memory-only credentials. The real standalone viewer smoke covers offline and
sandboxed rendering, graph/inspector state and watch recovery. Keep tests at those
boundaries rather than restoring old RPC-model fixtures.

`test/native-deployment.sh` builds an isolated Kind stack and runs the actual
compatibility corpus and authenticated browser acceptance test. For an existing
OrbStack stack use the explicit-context commands in the sample README. Expected
failure cases must fail for the expected reason; an arbitrary terminal state is
not a passing result. Do not reset unrelated clusters or artifact stores.
