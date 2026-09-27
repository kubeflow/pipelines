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
   *
   * Credential boundary invariant:
   *   Caller-controlled code must not receive credentials the caller is
   *   not authorized to use, including the artifact Secret.
   *
   * - Supported images (tensorflow/tensorflow:*) keep the admin-configured
   *   service account and base-template credentials.
   * - Custom images get ALL credentials stripped from the final merged pod,
   *   automountServiceAccountToken set to false, and serviceAccountName
   *   cleared.
   * - ALL caller-supplied credential references (secretKeyRef, configMapKeyRef,
   *   envFrom with secretRef) are unconditionally rejected regardless of image.
   */

  function deepEqual(a: any, b: any): boolean {
    if (a === b) return true;
    if (typeof a !== 'object' || typeof b !== 'object' || a === null || b === null) return false;
    const keysA = Object.keys(a);
    const keysB = Object.keys(b);
    if (keysA.length !== keysB.length) return false;
    for (const key of keysA) {
      if (!keysB.includes(key) || !deepEqual(a[key], b[key])) return false;
    }
    return true;
  }

  function sanitizePodTemplateSpec(
    unsafe: any,
    base: any,
    isCustomImage: boolean,
    logdir: string,
  ): any {
    // 1. Validate logdir argument expansion
    if (logdir.startsWith('-') || /[;&$|`\n\r<>]/.test(logdir)) {
      throw new Error('Invalid logdir argument');
    }
    // 2. Merge base and unsafe
    const safe = JSON.parse(JSON.stringify(base || { spec: { containers: [{}] } }));
    safe.spec = safe.spec || {};
    safe.spec.containers = safe.spec.containers || [{}];

    if (unsafe && typeof unsafe === 'object' && unsafe.spec) {
      // Enforce explicit policy for caller volume types
      if (Array.isArray(unsafe.spec.volumes)) {
        safe.spec.volumes = safe.spec.volumes || [];
        for (const v of unsafe.spec.volumes) {
          if (v.name) {
            // ONLY allow safe volume types (strip hostPath, secret, configMap, etc.)
            if (!v.emptyDir && !v.persistentVolumeClaim) {
              continue;
            }
            const existing = safe.spec.volumes.find((ev: any) => ev.name === v.name);
            if (existing) {
              if (!deepEqual(existing, v)) {
                throw new Error('Conflicting volume: ' + v.name);
              }
            } else {
              safe.spec.volumes.push(v);
            }
          }
        }
      }

      if (Array.isArray(unsafe.spec.containers) && unsafe.spec.containers.length > 0) {
        const container = unsafe.spec.containers[0];

        if (Array.isArray(container.volumeMounts)) {
          safe.spec.containers[0].volumeMounts = safe.spec.containers[0].volumeMounts || [];
          for (const m of container.volumeMounts) {
            if (m.name && m.mountPath) {
              // Ensure the mount references a valid volume in the merged spec
              const volumeExists =
                safe.spec.volumes && safe.spec.volumes.find((v: any) => v.name === m.name);
              if (!volumeExists) {
                continue; // Strip mount if its volume was stripped or missing
              }

              const existingByName = safe.spec.containers[0].volumeMounts.find(
                (em: any) => em.name === m.name,
              );
              const existingByPath = safe.spec.containers[0].volumeMounts.find(
                (em: any) => em.mountPath === m.mountPath,
              );
              if (existingByName && !deepEqual(existingByName, m))
                throw new Error('Conflicting volumeMount name: ' + m.name);
              if (existingByPath && !deepEqual(existingByPath, m))
                throw new Error('Conflicting volumeMount path: ' + m.mountPath);
              if (!existingByName && !existingByPath) safe.spec.containers[0].volumeMounts.push(m);
            }
          }
        }

        // Reject ALL caller-supplied credential references unconditionally.
        // Credentials must come exclusively from the admin-controlled base template.
        if (Array.isArray(container.env)) {
          safe.spec.containers[0].env = safe.spec.containers[0].env || [];
          for (const e of container.env) {
            if (e.name) {
              if (e.valueFrom && (e.valueFrom.secretKeyRef || e.valueFrom.configMapKeyRef)) {
                // Reject: callers may not inject credential references
                continue;
              }

              const existing = safe.spec.containers[0].env.find((ee: any) => ee.name === e.name);
              if (existing) {
                if (!deepEqual(existing, e)) throw new Error('Conflicting env name: ' + e.name);
              } else {
                safe.spec.containers[0].env.push(e);
              }
            }
          }
        }

        // Unconditionally drop caller-supplied envFrom (secretRef / configMapRef)
        // Caller envFrom is never merged; only base-template envFrom is kept.
      }
    }

    // 3. Post-merge credential boundary enforcement for custom images.
    //    Custom images are caller-controlled code and must not receive ANY
    //    credentials — not even the base template's artifact Secret.
    if (isCustomImage) {
      // Strip secret volumes from the final merged pod
      if (Array.isArray(safe.spec.volumes)) {
        safe.spec.volumes = safe.spec.volumes.filter(
          (v: any) =>
            !v.secret &&
            !(v.projected && v.projected.sources && v.projected.sources.some((s: any) => s.secret)),
        );
      }

      // Strip credential env vars from the final merged container
      const container = safe.spec.containers[0];
      if (Array.isArray(container.env)) {
        container.env = container.env.filter(
          (e: any) => !(e.valueFrom && e.valueFrom.secretKeyRef),
        );
      }
      if (Array.isArray(container.envFrom)) {
        container.envFrom = container.envFrom.filter((e: any) => !e.secretRef);
        if (container.envFrom.length === 0) delete container.envFrom;
      }

      // Strip mounts that reference volumes we just removed
      if (Array.isArray(container.volumeMounts) && Array.isArray(safe.spec.volumes)) {
        const volumeNames = new Set(safe.spec.volumes.map((v: any) => v.name));
        container.volumeMounts = container.volumeMounts.filter((m: any) => volumeNames.has(m.name));
      }

      // Revoke the runtime service-account identity
      safe.spec.automountServiceAccountToken = false;
      safe.spec.serviceAccountName = '';
    }

    return safe;
  }

  /**
   * Returns true when the supplied image string refers to a supported
   * (trusted) TensorBoard image — i.e. one whose repository path starts
   * with the configured tfImageName (default: "tensorflow/tensorflow").
   *
   * A supported image keeps the administrator-configured service account
   * and base-template credentials.  Everything else is treated as
   * caller-controlled code that must not receive those credentials.
   */
  function isSupportedImage(image: string | undefined, tfImageName: string): boolean {
    if (!image) return true; // tfversion path — uses the default image
    // Exact match (no tag) or tagged variant of the configured image
    if (image === tfImageName) return true;
    if (image.startsWith(tfImageName + ':')) return true;
    return false;
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
   * (PVC, emptyDir) are kept from the caller to prevent privilege escalation.
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

      const isCustomImage = !isSupportedImage(
        image as string | undefined,
        tensorboardConfig.tfImageName,
      );
      const mergedPodTemplateSpec = sanitizePodTemplateSpec(
        unsafePodTemplateSpec,
        tensorboardConfig.podTemplateSpec,
        isCustomImage,
        logdir as string,
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
