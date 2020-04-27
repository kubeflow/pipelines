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
import { ViewerTensorboardConfig } from '../configs';
import { AuthServiceApi } from '../src/generated/apis/auth';

export const getTensorboardHandlers = (
  tensorboardConfig: ViewerTensorboardConfig,
  otherConfig: { apiServerAddress: string; authzEnabled: boolean },
): { get: Handler; create: Handler; delete: Handler } => {
  const { apiServerAddress, authzEnabled } = otherConfig;
  const authService = new AuthServiceApi({ basePath: apiServerAddress });
  /**
   * A handler which retrieve the endpoint for a tensorboard instance. The
   * handler expects a query string `logdir`.
   */
  const get: Handler = async (req, res) => {
    const { logdir, namespace } = req.query;
    if (!logdir) {
      res.status(404).send('logdir argument is required');
      return;
    }
    if (!namespace) {
      res.status(404).send('namespace argument is required');
      return;
    }

    try {
      if (authzEnabled) {
        await authService.authorize(namespace, 'VIEWERS', 'GET');
      }
      res.send(await k8sHelper.getTensorboardInstance(logdir, namespace));
    } catch (err) {
      console.error('Failed to list Tensorboard pods: ', err?.body || err);
      res.status(500).send(`Failed to list Tensorboard pods: ${err}`);
    }
  };

  /**
   * A handler which will create a tensorboard viewer CRD, waits for the
   * tensorboard instance to be ready, and return the endpoint to the instance.
   * The handler expects the following query strings in the request:
   * - `logdir`
   * - `tfversion`
   */
  const create: Handler = async (req, res) => {
    const { logdir, namespace, tfversion } = req.query;
    if (!logdir) {
      res.status(404).send('logdir argument is required');
      return;
    }
    if (!namespace) {
      res.status(404).send('namespace argument is required');
      return;
    }
    if (!tfversion) {
      res.status(404).send('tfversion (tensorflow version) argument is required');
      return;
    }

    try {
      if (authzEnabled) {
        await authService.authorize(namespace, 'VIEWERS', 'CREATE');
      }
      await k8sHelper.newTensorboardInstance(
        logdir,
        namespace,
        tensorboardConfig.tfImageName,
        tfversion,
        tensorboardConfig.podTemplateSpec,
      );
      const tensorboardAddress = await k8sHelper.waitForTensorboardInstance(
        logdir,
        namespace,
        60 * 1000,
      );
      res.send(tensorboardAddress);
    } catch (err) {
      console.error('Failed to start Tensorboard app: ', err?.body || err);
      res.status(500).send(`Failed to start Tensorboard app: ${err}`);
    }
  };

  /**
   * A handler that deletes a tensorboard viewer. The handler expects query string
   * `logdir` in the request.
   */
  const deleteHandler: Handler = async (req, res) => {
    const { logdir, namespace } = req.query;
    if (!logdir) {
      res.status(404).send('logdir argument is required');
      return;
    }
    if (!namespace) {
      res.status(404).send('namespace argument is required');
      return;
    }

    try {
      if (authzEnabled) {
        await authService.authorize(namespace, 'VIEWERS', 'DELETE');
      }
      await k8sHelper.deleteTensorboardInstance(logdir, namespace);
      res.send('Tensorboard deleted.');
    } catch (err) {
      console.error('Failed to delete Tensorboard app: ', err?.body || err);
      res.status(500).send(`Failed to delete Tensorboard app: ${err}`);
    }
  };

  return {
    get,
    create,
    delete: deleteHandler,
  };
};
