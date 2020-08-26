#!/bin/bash -e
# Copyright 2019 Google LLC
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


# A script to gather locally-installed python packages, and based on
# a specified license table (3 columns: name,license_link,license_type) csv file,
# download and save all license files into specified directory.
# Usage:
#   license.sh third_party_licenses.csv /usr/licenses


# Get the list of python packages installed locally.
IFS=$'\n'
INSTALLED_PACKAGES=($(pip freeze | sed s/=.*//))


# Get the list of python packages tracked in the given CSV file.
REGISTERED_PACKAGES=()
while IFS=, read -r col1 col2 col3
do
  REGISTERED_PACKAGES+=($col1)
done < $1

# Make sure all locally installed packages are covered.
DIFF=()
for i in "${INSTALLED_PACKAGES[@]}"; do
  skip=
  for j in "${REGISTERED_PACKAGES[@]}"; do
    [[ $i == $j ]] && { skip=1; break; }
  done
  [[ -n $skip ]] || DIFF+=("$i")
done

if [ -n "$DIFF" ]; then
  echo "The following packages are not found for licenses tracking."
  echo "Please add an entry in $1 for each of them."
  echo ${DIFF[@]}
  exit 1
fi

# Gather license files for each package. For packages with GPL license we mirror the source code.
mkdir -p $2/source
EXTRANEOUS=()
while IFS=, read -r col1 col2 col3
do
  if [[ " ${INSTALLED_PACKAGES[@]} " =~ " ${col1} " ]]; then
    # Download license is moved to part of `./license-download.sh`.
    # wget -O $2/$col1.LICENSE $col2
    # We check existence of the license file instead.
    if [[ -f "$2/${col1}.LICENSE" ]]; then
      # Verifies the LICENSE file is not empty.
      if [[ -s "$2/${col1}.LICENSE" ]]; then
        echo "OK: ${col1}'s license exists."
      else
        echo "Error: ${col1}'s license file exists, but is empty."
        exit 1
      fi
    else
      echo "Error: ${col1} package's license is missing."
      echo "Please rerun ./license-download.sh locally and commit into licenses folder."
      exit 1
    fi
    if [[ "${col3}" == *GPL* ]] || [[ "${col3}" =~ ^MPL ]]; then
      pip install -t "$2/source/${col1}" ${col1}
    fi
  else
    EXTRANEOUS+=("$col1")
  fi
done < $1

if [ -n "${EXTRANEOUS}" ]; then
  echo "Some libraries are part of third_party_licenses.csv, but not installed."
  echo "Please remove them from third_party_licenses.csv:"
  echo "${EXTRANEOUS[@]}"
fi
