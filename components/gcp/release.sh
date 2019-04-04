#!/bin/bash -e
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

# Update README.md from sample.ipynb
find ./ -type f -name "sample.ipynb" -exec jupyter nbconvert {} --to markdown --output README.md \;

# Stage updated README.md and sample.ipynb.
git add **/*.md
git add **/*.ipynb

# Generate pipeline zip file from notebook
find ./ -type f -name "sample.ipynb" -exec jupyter nbconvert --execute {} --to notebook --output executed-sample.ipynb \;

# Add component.yaml file to the pipeline zip file so it can 
# it can work as both pipeline or component.
shopt -s nullglob
for x in ./*/*/*.zip; do
    dir=$(dirname "$x")
    zip -r -j "$x" "$dir/component.yaml"
done