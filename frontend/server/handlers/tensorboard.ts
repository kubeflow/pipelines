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

/**
 * A handler which retrieve the endpoint for a tensorboard instance. The
 * handler expects a query string `logdir`.
 */
export const getTensorboardHandler: Handler = async (req, res) => {
  if (!k8sHelper.isInCluster) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }

  if (!req.query.logdir) {
    res.status(404).send('logdir argument is required');
    return;
  }

  const logdir = decodeURIComponent(req.query.logdir);

  try {
    res.send(await k8sHelper.getTensorboardInstance(logdir));
  } catch (err) {
    res.status(500).send('Failed to list Tensorboard pods: ' + JSON.stringify(err));
  }
};

/**
 * Returns a handler which will create a tensorboard viewer CRD, waits for the
 * tensorboard instance to be ready, and return the endpoint to the instance.
 * The handler expects the following query strings in the request:
 * - `logdir`
 * - `tfversion`
 * @param podTemplateSpec Custom pod template specification to be applied on the
 * tensorboard pod.
 */
export function getCreateTensorboardHandler(podTemplateSpec?: object): Handler {
  return async (req, res) => {
    if (!k8sHelper.isInCluster) {
      res.status(500).send('Cannot talk to Kubernetes master');
      return;
    }

    if (!req.query.logdir) {
      res.status(404).send('logdir argument is required');
      return;
    }

    if (!req.query.tfversion) {
      res.status(404).send('tensorflow version argument is required');
      return;
    }

    const logdir = decodeURIComponent(req.query.logdir);
    const tfversion = decodeURIComponent(req.query.tfversion);

    try {
      await k8sHelper.newTensorboardInstance(logdir, tfversion, podTemplateSpec);
      const tensorboardAddress = await k8sHelper.waitForTensorboardInstance(logdir, 60 * 1000);
      res.send(tensorboardAddress);
    } catch (err) {
      res.status(500).send('Failed to start Tensorboard app: ' + JSON.stringify(err));
    }
  };
}

/**
 * A handler that deletes a tensorboard viewer. The handler expects query string
 * `logdir` in the request.
 */
export const deleteTensorboardHandler: Handler = async (req, res) => {
  if (!k8sHelper.isInCluster) {
    res.status(500).send('Cannot talk to Kubernetes master');
    return;
  }

  if (!req.query.logdir) {
    res.status(404).send('logdir argument is required');
    return;
  }

  const logdir = decodeURIComponent(req.query.logdir);

  try {
    await k8sHelper.deleteTensorboardInstance(logdir);
    res.send('Tensorboard deleted.');
  } catch (err) {
    res.status(500).send('Failed to delete Tensorboard app: ' + JSON.stringify(err));
  }
};
