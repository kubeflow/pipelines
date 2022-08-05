#!/bin/bash

set -ex

# 1. Install yarn
npm install -D yarn

# 2. Set up yarn: It will convert from package.json to yarn.lock
npx yarn install

# 3. Generate full license texts in one file
npx yarn licenses generate-disclaimer > dependency-licenses.txt

# 4. Generate full license texts for Frontend server
pushd server
npx yarn install
npx yarn licenses generate-disclaimer > dependency-licenses.txt
popd

# 5. Merge two licenses to one file in server/dependency-licenses.txt
cat dependency-licenses.txt >> server/dependency-licenses.txt

# Reference for usage of yarn:
# List all packages with their license
# $ npx yarn licenses list | less

# Summarize licenses
# $ npx yarn licenses list | grep License | sort -u | cut -d: -f2
