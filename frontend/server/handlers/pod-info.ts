// Copyright 2020 Google LLC
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

import { Handler } from 'express';
import * as k8sHelper from '../k8s-helper';

/**
 * podInfoHandler retrieves pod info and sends back as JSON format.
 */
export const podInfoHandler: Handler = async (req, res) => {
  const { podname, podnamespace } = req.query;
  if (!podname) {
    res.status(404).send('podname argument is required');
    return;
  }
  if (!podnamespace) {
    res.status(404).send('podnamespace argument is required');
    return;
  }
  const podName = decodeURIComponent(podname);
  const podNamespace = decodeURIComponent(podnamespace);

  try {
    const pod = await k8sHelper.getPod(podName, podNamespace);
    res.status(200).send(JSON.stringify(pod));
  } catch (err) {
    const message = `Could not get pod ${podName} in namespace ${podNamespace}`;
    console.error(message, err);
    const detailedMessage = err?.body?.message || err?.message || err;
    res.status(500).send(`${message}: ${detailedMessage}`);
  }
};
