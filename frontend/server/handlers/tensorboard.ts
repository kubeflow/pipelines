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
import { createTensorboardProxyPath } from './tensorboard-proxy.js';

export const getTensorboardHandlers = (
  tensorboardConfig: ViewerTensorboardConfig,
  authorizeFn: AuthorizeFn,
): { get: Handler; create: Handler; delete: Handler } => {
  /**
   * Retrieves the scoped proxy path and image metadata for a TensorBoard instance.
   * The handler expects query strings `logdir` and `namespace`.
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
      const tensorboardInstance = await k8sHelper.getTensorboardInstance(
        logdir as string,
        namespace as string,
      );
      res.send({
        proxyPath: tensorboardInstance.viewerName
          ? createTensorboardProxyPath(
              namespace as string,
              tensorboardInstance.viewerName,
              tensorboardConfig.proxySigningSecret,
            )
          : '',
        tfVersion: tensorboardInstance.tfVersion,
        image: tensorboardInstance.image,
      });
    } catch (err) {
      const details = await parseError(err);
      console.error(`Failed to list Tensorboard pods: ${details.message}`, details.additionalInfo);
      res.status(500).send(`Failed to list Tensorboard pods: ${details.message}`);
    }
  };

  /**
   * Creates a TensorBoard viewer CRD, waits for the viewer to become ready,
   * and returns the scoped proxy path for that instance.
   * The handler expects the following query strings in the request:
   * - `logdir`
   * - `namespace`
   * - `tfversion`, optional. TODO: consider deprecating
   * - `image`, optional
   * - `podtemplatespec`, optional
   *
   * Either `image` or `tfversion` should be specified.
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
    if (typeof namespace !== 'string' || !isAllowedResourceName(namespace as string)) {
      res.status(400).send('invalid namespace');
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
      );
      const viewerName = await k8sHelper.waitForTensorboardInstance(
        logdir as string,
        namespace as string,
        60 * 1000,
      );
      res.send(
        createTensorboardProxyPath(
          namespace as string,
          viewerName,
          tensorboardConfig.proxySigningSecret,
        ),
      );
    } catch (err) {
      const details = await parseError(err);
      console.error(`Failed to start Tensorboard app: ${details.message}`, details.additionalInfo);
      res.status(500).send(`Failed to start Tensorboard app: ${details.message}`);
    }
  };

  /**
   * Deletes a TensorBoard viewer. The handler expects query strings `logdir`
   * and `namespace`.
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
    if (typeof namespace !== 'string' || !isAllowedResourceName(namespace as string)) {
      res.status(400).send('invalid namespace');
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
      await k8sHelper.deleteTensorboardInstance(logdir as string, namespace as string);
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
