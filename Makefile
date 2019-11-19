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

all: build

build:
  # Create vendor directories with all dependencies.
	go mod vendor
	# Extract go licenses into a single file. This assume licext is install globally through
	# npm install -g license-extractor
	# See https://github.com/arei/license-extractor
	licext --mode merge --source vendor/ --target third_party/license.txt --overwrite
	# Delete vendor directory
	rm -rf vendor
