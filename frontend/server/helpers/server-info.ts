// Copyright 2019-2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as fs from 'fs';

const namespaceFilePath = '/var/run/secrets/kubernetes.io/serviceaccount/namespace';
let serverNamespace: string | undefined;
// The file path contains pod namespace when in Kubernetes cluster.
if (fs.existsSync(namespaceFilePath)) {
  serverNamespace = fs.readFileSync(namespaceFilePath, 'utf-8');
}

export function getServerNamespace(): string | undefined {
  // get ml-pipeline-ui pod namespace
  return serverNamespace;
}

export function getServerPodName(): string | undefined {
  // get ml-pipeline-ui pod name
  const { HOSTNAME: POD_NAME } = process.env;
  return POD_NAME;
}
