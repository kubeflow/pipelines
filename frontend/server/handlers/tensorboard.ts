// Copyright 2019-2021 The Kubeflow Authors
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
import * as k8sHelper from '../k8s-helper.js';
import { ViewerTensorboardConfig } from '../configs.js';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';
import { parseError, isAllowedResourceName } from '../utils.js';
import { AuthorizeFn } from '../helpers/auth.js';

export const getTensorboardHandlers = (
  tensorboardConfig: ViewerTensorboardConfig,
  authorizeFn: AuthorizeFn,
): { get: Handler; create: Handler; delete: Handler } => {
  /**
   * A handler which retrieve the endpoint for a tensorboard instance. The
   * handler expects a query string `logdir`.
   */
  const get: Handler = async (req, res) => {
    const { logdir, namespace } = req.query;
    if (!logdir) {
      res.status(400).send('logdir argument is required');
      return;
    }
    if (!namespace) {
      res.status(400).send('namespace argument is required');
      return;
    }
    if (typeof namespace !== 'string' || !isAllowedResourceName(namespace as string)) {
      res.status(400).send('invalid namespace');
      return;
    }

    try {
      const authError = await authorizeFn(
        {
          verb: AuthorizeRequestVerb.GET,
          resources: AuthorizeRequestResources.VIEWERS,
          namespace: namespace as string,
        },
        req,
      );
      if (authError) {
        res.status(401).send(authError.message);
        return;
      }
      res.send(
        await k8sHelper.getTensorboardInstance(
          logdir as string,
          namespace as string,
          tensorboardConfig.clusterDomain,
        ),
      );
    } catch (err) {
      const details = await parseError(err);
      console.error(`Failed to list Tensorboard pods: ${details.message}`, details.additionalInfo);
      res.status(500).send(`Failed to list Tensorboard pods: ${details.message}`);
    }
  };

  /**
   * A handler which will create a tensorboard viewer CRD, waits for the
   * tensorboard instance to be ready, and return the endpoint to the instance.
   * The handler expects the following query strings in the request:
   * - `logdir`
   * - `tfversion`, optional. TODO: consider deprecate
   * - `image`, optional
   * - `podtemplatespec`, optional
   *
   * image or tfversion should be specified.
   */
  const create: Handler = async (req, res) => {
    const { logdir, namespace, tfversion, image, podtemplatespec: podTemplateSpecRaw } = req.query;
    if (!logdir) {
      res.status(400).send('logdir argument is required');
      return;
    }
    if (!namespace) {
      res.status(400).send('namespace argument is required');
      return;
    }
    if (!tfversion && !image) {
      res.status(400).send('missing required argument: tfversion (tensorflow version) or image');
      return;
    }
    if (tfversion && image) {
      res.status(400).send('tfversion and image cannot be specified at the same time');
      return;
    }
    let podTemplateSpec: any | undefined;
    if (podTemplateSpecRaw) {
      try {
        podTemplateSpec = JSON.parse(podTemplateSpecRaw as string);
      } catch (err) {
        res.status(400).send(`podtemplatespec is not valid JSON: ${err}`);
        return;
      }
    }

    try {
      const authError = await authorizeFn(
        {
          verb: AuthorizeRequestVerb.CREATE,
          resources: AuthorizeRequestResources.VIEWERS,
          namespace: namespace as string,
        },
        req,
      );
      if (authError) {
        res.status(401).send(authError.message);
        return;
      }
      await k8sHelper.newTensorboardInstance(
        logdir as string,
        namespace as string,
        (image || tensorboardConfig.tfImageName) as string,
        (tfversion as string) || '',
        podTemplateSpec || tensorboardConfig.podTemplateSpec,
        tensorboardConfig.clusterDomain,
      );
      const tensorboardAddress = await k8sHelper.waitForTensorboardInstance(
        logdir as string,
        namespace as string,
        60 * 1000,
        tensorboardConfig.clusterDomain,
      );
      res.send(tensorboardAddress);
    } catch (err) {
      const details = await parseError(err);
      console.error(`Failed to start Tensorboard app: ${details.message}`, details.additionalInfo);
      res.status(500).send(`Failed to start Tensorboard app: ${details.message}`);
    }
  };

  /**
   * A handler that deletes a tensorboard viewer. The handler expects query string
   * `logdir` in the request.
   */
  const deleteHandler: Handler = async (req, res) => {
    const { logdir, namespace } = req.query;
    if (!logdir) {
      res.status(400).send('logdir argument is required');
      return;
    }
    if (!namespace) {
      res.status(400).send('namespace argument is required');
      return;
    }

    try {
      const authError = await authorizeFn(
        {
          verb: AuthorizeRequestVerb.DELETE,
          resources: AuthorizeRequestResources.VIEWERS,
          namespace: namespace as string,
        },
        req,
      );
      if (authError) {
        res.status(401).send(authError.message);
        return;
      }
      await k8sHelper.deleteTensorboardInstance(
        logdir as string,
        namespace as string,
        tensorboardConfig.clusterDomain,
      );
      res.send('Tensorboard deleted.');
    } catch (err) {
      const details = await parseError(err);
      console.error(`Failed to delete Tensorboard app: ${details.message}`, details.additionalInfo);
      res.status(500).send(`Failed to delete Tensorboard app: ${details.message}`);
    }
  };

  return {
    get,
    create,
    delete: deleteHandler,
  };
};
