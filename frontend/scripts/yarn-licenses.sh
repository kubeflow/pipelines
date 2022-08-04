#!/bin/bash

set -ex

# 1. Install yarn
npm install -D yarn

# 2. Set up yarn: It will convert from package.json to yarn.lock
npx yarn install

# 3. List all packages with their license
# npx yarn licenses list | less

# # 4. Summarize licenses
# yarn licenses list | grep License | sort -u | cut -d: -f2

# 5. Generate full license texts in one file
npx yarn licenses generate-disclaimer > dependency-licenses.txt

pushd server
npx yarn install
npx yarn licenses generate-disclaimer > dependency-licenses.txt
popd
 
cat dependency-licenses.txt >> server/dependency-licenses.txt
