// Copyright 2020 The Kubeflow Authors
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
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
} from '../src/generated/apis/auth/index.js';
import { AuthorizeFn } from '../helpers/auth.js';

/**
 * Get pod info handlers.
 */
export function getPodInfoHandlers(authorizeFn: AuthorizeFn) {
  const podInfoHandler: Handler = async (req, res) => {
    const { podname, podnamespace } = req.query;
    if (!podname) {
      // 422 status code "Unprocessable entity", refer to https://stackoverflow.com/a/42171674
      res.status(422).send('podname argument is required');
      return;
    }
    if (!podnamespace) {
      res.status(422).send('podnamespace argument is required');
      return;
    }

    // Check access to namespace
    try {
      const authError = await authorizeFn(
        {
          verb: AuthorizeRequestVerb.GET,
          resources: AuthorizeRequestResources.VIEWERS,
          namespace: podnamespace as string,
        },
        req,
      );
      if (authError) {
        res.status(403).send('Access denied to namespace');
        return;
      }
    } catch (error) {
      console.error('Authorization error:', error);
      res.status(500).send('Authorization check failed');
      return;
    }

    const podName = decodeURIComponent(podname as string);
    const podNamespace = decodeURIComponent(podnamespace as string);

    const [pod, err] = await k8sHelper.getPod(podName, podNamespace);
    if (err) {
      const { message, additionalInfo } = err;
      console.error(message, additionalInfo);
      res.status(500).send(message);
      return;
    }
    res.status(200).send(JSON.stringify(pod));
  };

  const podEventsHandler: Handler = async (req, res) => {
    const { podname, podnamespace } = req.query;
    if (!podname) {
      res.status(422).send('podname argument is required');
      return;
    }
    if (!podnamespace) {
      res.status(422).send('podnamespace argument is required');
      return;
    }

    // Check access to namespace
    try {
      const authError = await authorizeFn(
        {
          verb: AuthorizeRequestVerb.GET,
          resources: AuthorizeRequestResources.VIEWERS,
          namespace: podnamespace as string,
        },
        req,
      );
      if (authError) {
        res.status(403).send('Access denied to namespace');
        return;
      }
    } catch (error) {
      console.error('Authorization error:', error);
      res.status(500).send('Authorization check failed');
      return;
    }

    const podName = decodeURIComponent(podname as string);
    const podNamespace = decodeURIComponent(podnamespace as string);

    const [eventList, err] = await k8sHelper.listPodEvents(podName, podNamespace);
    if (err) {
      const { message, additionalInfo } = err;
      console.error(message, additionalInfo);
      res.status(500).send(message);
      return;
    }
    res.status(200).send(JSON.stringify(eventList));
  };

  return { podInfoHandler, podEventsHandler };
}
