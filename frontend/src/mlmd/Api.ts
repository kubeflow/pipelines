/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { MetadataStoreServicePromiseClient } from 'src/third_party/mlmd';

/** Known Artifact properties */
export enum ArtifactProperties {
  ALL_META = '__ALL_META__',
  CREATE_TIME = 'create_time',
  DESCRIPTION = 'description',
  NAME = 'name',
  PIPELINE_NAME = 'pipeline_name',
  VERSION = 'version',
}

/** Known Artifact custom properties */
export enum ArtifactCustomProperties {
  WORKSPACE = '__kf_workspace__',
  RUN = '__kf_run__',
  NAME = 'name',
  DISPLAY_NAME = 'display_name',
  PIPELINE_NAME = 'pipeline_name', // TODO: Remove when switching to contexts
  RUN_ID = 'run_id', // TODO: Remove when switching to contexts
}

/** Known Execution properties */
export enum ExecutionProperties {
  NAME = 'name', // currently not available in api, use component_id instead
  COMPONENT_ID = 'component_id',
  PIPELINE_NAME = 'pipeline_name',
  RUN_ID = 'run_id',
  STATE = 'state',
}

/** Known Execution custom properties */
export enum ExecutionCustomProperties {
  NAME = 'name',
  WORKSPACE = '__kf_workspace__',
  RUN = '__kf_run__',
  RUN_ID = 'run_id', // TODO: Remove when switching to contexts
  TASK_ID = 'task_id',
}

/** Format for a List request */
export interface ListRequest {
  filter?: string;
  orderAscending?: boolean;
  pageSize?: number;
  pageToken?: string;
  sortBy?: string;
}

/**
 * Class to wrap backend APIs.
 */
export class Api {
  private static instance: Api;
  private metadataServicePromiseClient = new MetadataStoreServicePromiseClient('', null, null);

  /**
   * Factory function to return an Api instance.
   */
  public static getInstance(): Api {
    if (!Api.instance) {
      Api.instance = new Api();
    }
    return Api.instance;
  }

  get metadataStoreService() {
    return this.metadataServicePromiseClient;
  }
}
