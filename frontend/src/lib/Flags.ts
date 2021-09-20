/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export enum Deployments {
  KUBEFLOW = 'KUBEFLOW',
  MARKETPLACE = 'MARKETPLACE',
}

const DEPLOYMENT_DEFAULT = undefined;
// Uncomment this to debug marketplace:
// const DEPLOYMENT_DEFAULT = Deployments.MARKETPLACE;

export const KFP_FLAGS = {
  DEPLOYMENT:
    // tslint:disable-next-line:no-string-literal
    window && window['KFP_FLAGS']
      ? // tslint:disable-next-line:no-string-literal
        window['KFP_FLAGS']['DEPLOYMENT'] === Deployments.KUBEFLOW
        ? Deployments.KUBEFLOW
        : // tslint:disable-next-line:no-string-literal
        window['KFP_FLAGS']['DEPLOYMENT'] === Deployments.MARKETPLACE
        ? Deployments.MARKETPLACE
        : DEPLOYMENT_DEFAULT
      : DEPLOYMENT_DEFAULT,
  HIDE_SIDENAV: window && window['KFP_FLAGS'] ? window['KFP_FLAGS']['HIDE_SIDENAV'] : false,
};
