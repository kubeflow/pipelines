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
import * as fs from 'fs';
import * as path from 'path';
import { Deployments } from '../configs';

const DEFAULT_FLAG = 'window.KFP_FLAGS.DEPLOYMENT=null';
const KUBEFLOW_CLIENT_PLACEHOLDER = '<script id="kubeflow-client-placeholder"></script>';

/**
 * Returns a handler which retrieve and modify the index.html.
 * @param options.staticDir serve the static resources in this folder.
 * @param options.deployment whether this is a kubeflow deployment.
 */
export function getIndexHTMLHandler(options: {
  staticDir: string;
  deployment: Deployments;
}): Handler {
  const content = replaceRuntimeContent(loadIndexHtml(options.staticDir), options.deployment);

  return function handleIndexHtml(_, res) {
    if (content) {
      res.contentType('text/html');
      res.send(content);
    } else {
      res.sendStatus(404);
    }
  };
}

function loadIndexHtml(staticDir: string) {
  const filepath = path.resolve(staticDir, 'index.html');
  const content = fs.readFileSync(filepath).toString();
  // sanity checking
  if (!content.includes(DEFAULT_FLAG)) {
    throw new Error(
      `Error: cannot find default flag: '${DEFAULT_FLAG}' in index html. Its content: '${content}'.`,
    );
  }
  if (!content.includes(KUBEFLOW_CLIENT_PLACEHOLDER)) {
    throw new Error(
      `Error: cannot find kubeflow client placeholder: '${KUBEFLOW_CLIENT_PLACEHOLDER}' in index html. Its content: '${content}'.`,
    );
  }
  return content;
}

function replaceRuntimeContent(content: string | undefined, deployment: Deployments) {
  if (content && deployment === Deployments.KUBEFLOW) {
    return content
      .replace(DEFAULT_FLAG, 'window.KFP_FLAGS.DEPLOYMENT="KUBEFLOW"')
      .replace(
        KUBEFLOW_CLIENT_PLACEHOLDER,
        `<script id="kubeflow-client-placeholder" src="/dashboard_lib.bundle.js"></script>`,
      );
  }
  if (content && deployment === Deployments.MARKETPLACE) {
    return content.replace(DEFAULT_FLAG, 'window.KFP_FLAGS.DEPLOYMENT="MARKETPLACE"');
  }
  return content;
}
