// Copyright 2018 Google LLC
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

import * as Utils from './Utils';
import { ExperimentServiceApi } from '../apis/experiment';
import { JobServiceApi } from '../apis/job';
import { RunServiceApi } from '../apis/run';
import { PipelineServiceApi, ApiPipeline } from '../apis/pipeline';
import { StoragePath } from './WorkflowParser';
import { VisualizationServiceApi } from '../apis/visualization';

const v1beta1Prefix = 'apis/v1beta1';

export interface ListRequest {
  filter?: string;
  orderAscending?: boolean;
  pageSize?: number;
  pageToken?: string;
  sortBy?: string;
}

export interface BuildInfo {
  apiServerCommitHash?: string;
  apiServerReady?: boolean;
  buildDate?: string;
  frontendCommitHash?: string;
}

let customVisualizationsAllowed: boolean;

export class Apis {

  public static async areCustomVisualizationsAllowed(): Promise<boolean> {
    // Result is cached to prevent excessive network calls for simple request.
    // The value of customVisualizationsAllowed will only change if the
    // deployment is updated and then the entire pod is restarted.
    if (customVisualizationsAllowed === undefined) {
      const result = await this._fetch('visualizations/allowed');
      customVisualizationsAllowed = result === 'true';
    }
    return customVisualizationsAllowed;
  }

  /**
   * Get pod logs
   */
  public static getPodLogs(podName: string): Promise<string> {
    return this._fetch(`k8s/pod/logs?podname=${encodeURIComponent(podName)}`);
  }

  public static get basePath(): string {
    const path = location.protocol + '//' + location.host + location.pathname;
    // Trim trailing '/' if exists
    return path.endsWith('/') ? path.substr(0, path.length - 1) : path;
  }

  public static get experimentServiceApi(): ExperimentServiceApi {
    if (!this._experimentServiceApi) {
      this._experimentServiceApi = new ExperimentServiceApi({ basePath: this.basePath });
    }
    return this._experimentServiceApi;
  }

  public static get jobServiceApi(): JobServiceApi {
    if (!this._jobServiceApi) {
      this._jobServiceApi = new JobServiceApi({ basePath: this.basePath });
    }
    return this._jobServiceApi;
  }

  public static get pipelineServiceApi(): PipelineServiceApi {
    if (!this._pipelineServiceApi) {
      this._pipelineServiceApi = new PipelineServiceApi({ basePath: this.basePath });
    }
    return this._pipelineServiceApi;
  }

  public static get runServiceApi(): RunServiceApi {
    if (!this._runServiceApi) {
      this._runServiceApi = new RunServiceApi({ basePath: this.basePath });
    }
    return this._runServiceApi;
  }

  public static get visualizationServiceApi(): VisualizationServiceApi {
    if (!this._visualizationServiceApi) {
      this._visualizationServiceApi = new VisualizationServiceApi({ basePath: this.basePath });
    }
    return this._visualizationServiceApi;
  }

  /**
   * Retrieve various information about the build.
   */
  public static async getBuildInfo(): Promise<BuildInfo> {
    return await this._fetchAndParse<BuildInfo>('/healthz', v1beta1Prefix);
  }

  /**
   * Verifies that Jupyter Hub is reachable.
   */
  public static async isJupyterHubAvailable(): Promise<boolean> {
    const response = await fetch('/hub/', { credentials: 'same-origin' });
    return response ? response.ok : false;
  }

  /**
   * Reads file from storage using server.
   */
  public static readFile(path: StoragePath): Promise<string> {
    return this._fetch('artifacts/get' +
      `?source=${path.source}&bucket=${path.bucket}&key=${encodeURIComponent(path.key)}`);
  }

  /**
   * Gets the address (IP + port) of a Tensorboard service given the logdir
   */
  public static getTensorboardApp(logdir: string): Promise<string> {
    return this._fetch(`apps/tensorboard?logdir=${encodeURIComponent(logdir)}`);
  }

  /**
   * Starts a deployment and service for Tensorboard given the logdir
   */
  public static startTensorboardApp(logdir: string): Promise<string> {
    return this._fetch(
      `apps/tensorboard?logdir=${encodeURIComponent(logdir)}`,
      undefined,
      undefined,
      { headers: { 'content-type': 'application/json', }, method: 'POST', }
    );
  }

  /**
   * Uploads the given pipeline file to the backend, and gets back a Pipeline
   * object with its metadata parsed.
   */
  public static async uploadPipeline(
    pipelineName: string, pipelineData: File): Promise<ApiPipeline> {
    const fd = new FormData();
    fd.append('uploadfile', pipelineData, pipelineData.name);
    return await this._fetchAndParse<ApiPipeline>(
      '/pipelines/upload',
      v1beta1Prefix,
      `name=${encodeURIComponent(pipelineName)}`,
      {
        body: fd,
        cache: 'no-cache',
        method: 'POST',
      });
  }

  /*
   * Retrieves the name of the Kubernetes cluster if it is running in GKE, otherwise returns an error.
   */
  public static async getClusterName(): Promise<string> {
    return this._fetch('system/cluster-name');
  }

  /*
   * Retrieves the project ID in which this cluster is running if using GKE, otherwise returns an error.
   */
  public static async getProjectId(): Promise<string> {
    return this._fetch('system/project-id');
  }

  private static _experimentServiceApi?: ExperimentServiceApi;
  private static _jobServiceApi?: JobServiceApi;
  private static _pipelineServiceApi?: PipelineServiceApi;
  private static _runServiceApi?: RunServiceApi;
  private static _visualizationServiceApi?: VisualizationServiceApi;

  /**
   * This function will call this._fetch() and parse the resulting JSON into an object of type T.
   */
  private static async _fetchAndParse<T>(
    path: string, apisPrefix?: string, query?: string, init?: RequestInit): Promise<T> {
    const responseText = await this._fetch(path, apisPrefix, query, init);
    try {
      return JSON.parse(responseText) as T;
    } catch (err) {
      throw new Error(`Error parsing response for path: ${path}\n\n` +
        `Response was: ${responseText}\n\nError was: ${JSON.stringify(err)}`);
    }
  }

  /**
   * Makes an HTTP request and returns the response as a string.
   *
   * Use this._fetchAndParse() if you intend to parse the response as JSON into an object.
   */
  private static async _fetch(
    path: string, apisPrefix?: string, query?: string, init?: RequestInit): Promise<string> {
    init = Object.assign(init || {}, { credentials: 'same-origin' });
    const response = await fetch((apisPrefix || '') + path + (query ? '?' + query : ''), init);
    const responseText = await response.text();
    if (response.ok) {
      return responseText;
    } else {
      Utils.logger.error(`Response for path: ${path} was not 'ok'\n\nResponse was: ${responseText}`);
      throw new Error(responseText);
    }
  }
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum RunSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name'
}

// Valid sortKeys as specified by the backend.
export enum PipelineSortKeys {
  AUTHOR = 'id',
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
}

// Valid sortKeys as specified by the backend.
export enum JobSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
  PIPELINE_ID = 'pipeline_id'
}

// Valid sortKeys as specified by the backend.
export enum ExperimentSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
}
