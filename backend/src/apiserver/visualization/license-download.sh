#!/bin/bash
# Copyright 2020 Google LLC
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

# A script to download licenses based on
# a specified license table (3 columns: name,license_link,license_type) csv file.
#
# Usage:
#   ./license-download.sh ./third_party_licenses.csv ./third_party_licenses

mkdir -p $2

IFS=$'\n'
while IFS=, read -r col1 col2 col3
do
  DEST="$2/$col1.LICENSE"
  if [[ -f "$DEST" ]]; then
    echo "Skip downloaded license file $DEST."
  else
    wget -O $2/$col1.LICENSE $col2
  fi
done < $1
