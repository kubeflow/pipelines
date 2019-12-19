// Copyright 2019 Google LLC
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

export function getGetTensorboardHandler(): Handler {
  return async (req, res) => {
    if (!k8sHelper.isInCluster) {
      res.status(500).send('Cannot talk to Kubernetes master');
      return;
    }
    const logdir = decodeURIComponent(req.query.logdir);
    if (!logdir) {
      res.status(404).send('logdir argument is required');
      return;
    }

    try {
      res.send(await k8sHelper.getTensorboardInstance(logdir));
    } catch (err) {
      res.status(500).send('Failed to list Tensorboard pods: ' + JSON.stringify(err));
    }
  };
}

export function getCreateTensorboardHandler(podTemplateSpec?: Object): Handler {
  return async (req, res) => {
    if (!k8sHelper.isInCluster) {
      res.status(500).send('Cannot talk to Kubernetes master');
      return;
    }
    const logdir = decodeURIComponent(req.query.logdir);
    if (!logdir) {
      res.status(404).send('logdir argument is required');
      return;
    }

    try {
      await k8sHelper.newTensorboardInstance(logdir, podTemplateSpec);
      const tensorboardAddress = await k8sHelper.waitForTensorboardInstance(logdir, 60 * 1000);
      res.send(tensorboardAddress);
    } catch (err) {
      res.status(500).send('Failed to start Tensorboard app: ' + JSON.stringify(err));
    }
  };
}
