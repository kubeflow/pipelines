.PHONY: test sdk-test frontend-test generate-python build

generate-python:
	uv run --frozen --extra codegen python hack/generate-python.py

sdk-test:
	uv run --frozen --extra test python -m pytest -q sdk/python/kfp test samples/argo-native/deploy_test.py .github/scripts

frontend-test:
	cd frontend && npm run test:ci

test: sdk-test frontend-test

build:
	uv build --package kfp --wheel
	cd frontend && npm run build
