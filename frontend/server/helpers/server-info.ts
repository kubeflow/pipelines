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
import { V1Pod } from '@kubernetes/client-node';
import { getPod } from '../k8s-helper';

const namespaceFilePath = '/var/run/secrets/kubernetes.io/serviceaccount/namespace';
let serverNamespace: string | undefined;
let hostPod: V1Pod | undefined;

// The file path contains pod namespace when in Kubernetes cluster.
if (fs.existsSync(namespaceFilePath)) {
  serverNamespace = fs.readFileSync(namespaceFilePath, 'utf-8');
}

// get ml-pipeline-ui host pod
export async function getHostPod(): Promise<[V1Pod | undefined, undefined] | [undefined, string]> {
  // use cached hostPod
  if (hostPod) {
    return [hostPod, undefined];
  }

  if (!serverNamespace) {
    return [undefined, "server namespace can't be obtained"];
  }

  // get ml-pipeline-ui server pod name
  const { HOSTNAME: POD_NAME } = process.env;
  if (!POD_NAME) {
    return [undefined, "server pod name can't be obtained"];
  }

  const [pod, err] = await getPod(POD_NAME, serverNamespace);

  if (err) {
    const { message, additionalInfo } = err;
    console.error(message, additionalInfo);
    return [undefined, `Failed to get host pod: ${message}`];
  }

  hostPod = pod;
  return [hostPod, undefined];
}
