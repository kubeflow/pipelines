# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# compile all v2 samples to pipeline job JSON files
.PHONY: all
all:
	@for f in $$(find . -name "*.py"); do \
		compiled="$${f%.*}_pipeline.json"; \
		echo "compiling $${f} to $${compiled}"; \
		dsl-compile-v2 --py "$${f}" --out $${compiled}""; \
	done

# clean up all genereated pipeline job JSON files
.PHONY: clean
clean:
	@find . -name "*.json" -exec rm {} \;
