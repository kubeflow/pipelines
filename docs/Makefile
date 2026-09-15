# Copyright The Kubeflow Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Makefile for the Kubeflow Pipelines documentation website.
#
# Install the build dependencies first, from the repository root:
#
#     pip install -r docs/requirements.txt

SPHINXBUILD   ?= python -m sphinx
SOURCEDIR     := .
BUILDDIR      := _build

.PHONY: help docs docs-serve docs-test docs-linkcheck docs-clean

help:
	@echo "docs            Build the HTML documentation into $(BUILDDIR)/html"
	@echo "docs-serve      Serve the documentation locally with live reload"
	@echo "docs-test       Build with warnings treated as errors (as CI does)"
	@echo "docs-linkcheck  Check external links"
	@echo "docs-clean      Remove the build directory"

docs:
	$(SPHINXBUILD) -b html "$(SOURCEDIR)" "$(BUILDDIR)/html"

docs-serve:
	python -m sphinx_autobuild "$(SOURCEDIR)" "$(BUILDDIR)/html" --port 8000

docs-test:
	$(SPHINXBUILD) -W --keep-going -b html "$(SOURCEDIR)" "$(BUILDDIR)/html"

docs-linkcheck:
	$(SPHINXBUILD) -b linkcheck "$(SOURCEDIR)" "$(BUILDDIR)/linkcheck"

docs-clean:
	rm -rf "$(BUILDDIR)"
