# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: release
release:
	./hack/release-docker.sh

.PHONY: release-image
release-image:
	docker build -t kfp-release - < Dockerfile.release

.PHONY: push-release-image
push-release-image:
push: release-image
	docker tag kfp-release gcr.io/ml-pipeline-test/release:latest
	docker push gcr.io/ml-pipeline-test/release:latest
