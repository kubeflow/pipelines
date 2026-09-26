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
  AuthorizeResourcesEnum,
  AuthorizeVerbEnum,
} from '../src/generated/apisv2beta1/auth/index.js';
import { parseError, isAllowedResourceName } from '../utils.js';
import { AuthorizeFn } from '../helpers/auth.js';
import { createTensorboardProxyPath } from './tensorboard-proxy.js';

/** Supply a default namespace only for unauthenticated standalone installations. */
export const getTensorboardHandlers = (
  tensorboardConfig: ViewerTensorboardConfig,
  authorizeFn: AuthorizeFn,
  defaultNamespace?: string,
): { get: Handler; create: Handler; delete: Handler } => {
  /**
   * Retrieves the scoped proxy path and image metadata for a TensorBoard instance.
   * The handler expects query strings `logdir` and `namespace`.
   */
  const get: Handler = async (req, res) => {
    const { logdir } = req.query;
    const namespace = req.query.namespace || defaultNamespace;
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
          verb: AuthorizeVerbEnum.GET,
          resources: AuthorizeResourcesEnum.VIEWERS,
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
   * Sanitize a client-supplied podTemplateSpec by extracting only safe
   * volume/volumeMount entries and merging them into the administrator-
   * configured base template.  Dangerous fields (hostPath, hostNetwork,
   * securityContext, privileged containers, etc.) are silently dropped.
   */
  function sanitizePodTemplateSpec(unsafe: any, base: any): any {
    if (!unsafe || typeof unsafe !== 'object' || !unsafe.spec) {
      return base;
    }
    const safe = JSON.parse(JSON.stringify(base || { spec: { containers: [{}] } }));
    safe.spec = safe.spec || {};
    safe.spec.containers = safe.spec.containers || [{}];

    if (Array.isArray(unsafe.spec.volumes)) {
      safe.spec.volumes = safe.spec.volumes || [];
      for (const v of unsafe.spec.volumes) {
        if (v.name && (v.persistentVolumeClaim || v.emptyDir)) {
          safe.spec.volumes.push(v);
        }
      }
    }

    if (Array.isArray(unsafe.spec.containers) && unsafe.spec.containers.length > 0) {
      const container = unsafe.spec.containers[0];

      if (Array.isArray(container.volumeMounts)) {
        safe.spec.containers[0].volumeMounts = safe.spec.containers[0].volumeMounts || [];
        for (const m of container.volumeMounts) {
          if (m.name && m.mountPath) {
            safe.spec.containers[0].volumeMounts.push(m);
          }
        }
      }

      if (Array.isArray(container.env)) {
        safe.spec.containers[0].env = safe.spec.containers[0].env || [];
        for (const e of container.env) {
          if (e.name && !e.valueFrom) {
            // Literal values without valueFrom are safe.
            // Secret references must be provided via the server's trusted
            // VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH configuration, rather
            // than caller-provided podTemplateSpec, to prevent privilege escalation.
            safe.spec.containers[0].env.push(e);
          }
        }
      }
    }

    return safe;
  }

  /**
   * Creates a TensorBoard viewer CRD, waits for the viewer to become ready,
   * and returns the scoped proxy path for that instance.
   * The handler expects the following query strings in the request:
   * - `logdir`
   * - `namespace`
   * - `tfversion`, optional. TODO: consider deprecating
   * - `image`, optional
   *
   * Volume mounts and environment variables may be supplied via a JSON
   * `podTemplateSpec` field in the POST body. Only safe volume types
   * (PVC, emptyDir) and standard metadata secrets are kept to prevent
   * privilege escalation.
   *
   * Either `image` or `tfversion` should be specified.
   */
  const create: Handler = async (req, res) => {
    const { logdir, tfversion, image } = req.query;
    const namespace = req.query.namespace || defaultNamespace;
    const unsafePodTemplateSpec = req.body?.podTemplateSpec;

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

    try {
      const authError = await authorizeFn(
        {
          verb: AuthorizeVerbEnum.CREATE,
          resources: AuthorizeResourcesEnum.VIEWERS,
          namespace: namespace as string,
        },
        req,
      );
      if (authError) {
        res.status(401).send(authError.message);
        return;
      }

      const mergedPodTemplateSpec = sanitizePodTemplateSpec(
        unsafePodTemplateSpec,
        tensorboardConfig.podTemplateSpec,
      );

      await k8sHelper.newTensorboardInstance(
        logdir as string,
        namespace as string,
        (image || tensorboardConfig.tfImageName) as string,
        (tfversion as string) || '',
        mergedPodTemplateSpec,
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
    const { logdir } = req.query;
    const namespace = req.query.namespace || defaultNamespace;
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
          verb: AuthorizeVerbEnum.DELETE,
          resources: AuthorizeResourcesEnum.VIEWERS,
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
