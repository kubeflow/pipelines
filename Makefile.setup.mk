SHELL := /bin/bash

.PHONY: setup-frontend-test
setup-frontend-test:
	npm cache clean --force && \
	cd ./frontend && npm ci