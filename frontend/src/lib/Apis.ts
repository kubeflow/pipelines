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

import * as portableFetch from 'portable-fetch';
import { ExperimentServiceApi, FetchAPI } from '../apis/experiment';
import { JobServiceApi } from '../apis/job';
import { ApiPipeline, ApiPipelineVersion, PipelineServiceApi } from '../apis/pipeline';
import { RunServiceApi } from '../apis/run';
import { ApiVisualization, VisualizationServiceApi } from '../apis/visualization';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';
import { PlotType } from '../components/viewers/Viewer';
import * as Utils from './Utils';
import { StoragePath } from './WorkflowParser';
import { buildQuery } from './Utils';

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
  apiServerTagName?: string;
  apiServerReady?: boolean;
  buildDate?: string;
  frontendCommitHash?: string;
  frontendTagName?: string;
}

// Hack types from https://github.com/microsoft/TypeScript/issues/1897#issuecomment-557057387
export type JSONPrimitive = string | number | boolean | null;
export type JSONValue = JSONPrimitive | JSONObject | JSONArray;
export type JSONObject = { [member: string]: JSONValue };
export type JSONArray = JSONValue[];

let customVisualizationsAllowed: boolean;

// For cross browser support, fetch should use 'same-origin' as default. This fixes firefox auth issues.
// Refrence: https://github.com/github/fetch#sending-cookies
const crossBrowserFetch: FetchAPI = (url, init) =>
  portableFetch(url, { credentials: 'same-origin', ...init });

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

  public static async buildPythonVisualizationConfig(
    visualizationData: ApiVisualization,
    namespace?: string,
  ): Promise<HTMLViewerConfig> {
    const visualization = await Apis.visualizationServiceApi.createVisualization(
      namespace || '',
      visualizationData,
    );
    if (visualization.html) {
      const htmlContent = visualization.html
        // Fixes issue with TFX components (and other iframe based
        // visualizations), where the method in which javascript interacts
        // with embedded iframes is not allowed when embedded in an additional
        // iframe. This is resolved by setting the srcdoc value rather that
        // manipulating the document directly.
        .replace('contentWindow.document.write', 'srcdoc=');
      return {
        htmlContent,
        type: PlotType.WEB_APP,
      } as HTMLViewerConfig;
    } else {
      // This should never be thrown as the html property of a generated
      // visualization is always set for successful visualization generations.
      throw new Error('Visualization was generated successfully but generated HTML was not found.');
    }
  }

  /**
   * Get pod logs
   */
  public static getPodLogs(runId: string, podName: string, podNamespace: string): Promise<string> {
    let query = `k8s/pod/logs?podname=${encodeURIComponent(podName)}&runid=${encodeURIComponent(
      runId,
    )}`;
    if (podNamespace) {
      query += `&podnamespace=${encodeURIComponent(podNamespace)}`;
    }
    return this._fetch(query);
  }

  /**
   * Get pod info
   */
  public static async getPodInfo(podName: string, podNamespace: string): Promise<JSONObject> {
    const query = `k8s/pod?podname=${encodeURIComponent(podName)}&podnamespace=${encodeURIComponent(
      podNamespace,
    )}`;
    const podInfo = await this._fetch(query);
    return JSON.parse(podInfo);
  }

  /**
   * Get pod events
   */
  public static async getPodEvents(podName: string, podNamespace: string): Promise<JSONObject> {
    const query = `k8s/pod/events?podname=${encodeURIComponent(
      podName,
    )}&podnamespace=${encodeURIComponent(podNamespace)}`;
    const eventList = await this._fetch(query);
    return JSON.parse(eventList);
  }

  public static get basePath(): string {
    const path = window.location.protocol + '//' + window.location.host + window.location.pathname;
    // Trim trailing '/' if exists
    return path.endsWith('/') ? path.substr(0, path.length - 1) : path;
  }

  public static get experimentServiceApi(): ExperimentServiceApi {
    if (!this._experimentServiceApi) {
      this._experimentServiceApi = new ExperimentServiceApi(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._experimentServiceApi;
  }

  public static get jobServiceApi(): JobServiceApi {
    if (!this._jobServiceApi) {
      this._jobServiceApi = new JobServiceApi(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._jobServiceApi;
  }

  public static get pipelineServiceApi(): PipelineServiceApi {
    if (!this._pipelineServiceApi) {
      this._pipelineServiceApi = new PipelineServiceApi(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._pipelineServiceApi;
  }

  public static get runServiceApi(): RunServiceApi {
    if (!this._runServiceApi) {
      this._runServiceApi = new RunServiceApi(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._runServiceApi;
  }

  public static get visualizationServiceApi(): VisualizationServiceApi {
    if (!this._visualizationServiceApi) {
      this._visualizationServiceApi = new VisualizationServiceApi(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
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
  public static readFile(path: StoragePath, namespace?: string, peek?: number): Promise<string> {
    return this._fetch(this.buildReadFileUrl({ path, namespace, peek, isDownload: false }));
  }

  /**
   * Builds an url for the readFile API to retrieve a workflow artifact.
   * @param path object describing the artifact (e.g. source, bucket, and key)
   * @param isDownload whether we download the artifact as is (e.g. skip extracting from *.tar.gz)
   */
  public static buildReadFileUrl({
    path,
    namespace,
    peek,
    isDownload,
  }: {
    path: StoragePath;
    namespace?: string;
    peek?: number;
    isDownload?: boolean;
  }) {
    const { source, bucket, key } = path;
    if (isDownload) {
      return `artifacts/${source}/${bucket}/${key}${buildQuery({
        namespace,
        peek,
      })}`;
    } else {
      return `artifacts/get${buildQuery({ source, namespace, peek, bucket, key })}`;
    }
  }

  /**
   * Builds an url to visually represents a workflow artifact location.
   * @param param.source source of the artifact (e.g. minio, gcs, s3, http, or https)
   * @param param.bucket name of the bucket with the artifact (or host for http/https)
   * @param param.key key (i.e. path) of the artifact in the bucket
   */
  public static buildArtifactUrl({ source, bucket, key }: StoragePath) {
    // TODO see https://github.com/kubeflow/pipelines/pull/3725
    return `${source}://${bucket}/${key}`;
  }

  /**
   * Gets the address (IP + port) of a Tensorboard service given the logdir and tfversion
   */
  public static getTensorboardApp(
    logdir: string,
    namespace: string,
  ): Promise<{ podAddress: string; tfVersion: string }> {
    return this._fetchAndParse<{ podAddress: string; tfVersion: string }>(
      `apps/tensorboard?logdir=${encodeURIComponent(logdir)}&namespace=${encodeURIComponent(
        namespace,
      )}`,
    );
  }

  /**
   * Starts a deployment and service for Tensorboard given the logdir
   */
  public static startTensorboardApp(
    logdir: string,
    tfversion: string,
    namespace: string,
  ): Promise<string> {
    return this._fetch(
      `apps/tensorboard?logdir=${encodeURIComponent(logdir)}&tfversion=${encodeURIComponent(
        tfversion,
      )}&namespace=${encodeURIComponent(namespace)}`,
      undefined,
      undefined,
      { headers: { 'content-type': 'application/json' }, method: 'POST' },
    );
  }

  /**
   * Check if the underlying Tensorboard pod is actually up, given the pod address
   */
  public static async isTensorboardPodReady(path: string): Promise<boolean> {
    const resp = await fetch(path, { method: 'HEAD' });
    return resp.ok;
  }

  /**
   * Delete a deployment and its service of the Tensorboard given the URL
   */
  public static deleteTensorboardApp(logdir: string, namespace: string): Promise<string> {
    return this._fetch(
      `apps/tensorboard?logdir=${encodeURIComponent(logdir)}&namespace=${encodeURIComponent(
        namespace,
      )}`,
      undefined,
      undefined,
      { method: 'DELETE' },
    );
  }

  /**
   * Uploads the given pipeline file to the backend, and gets back a Pipeline
   * object with its metadata parsed.
   */
  public static async uploadPipeline(
    pipelineName: string,
    pipelineDescription: string,
    pipelineData: File,
  ): Promise<ApiPipeline> {
    const fd = new FormData();
    fd.append('uploadfile', pipelineData, pipelineData.name);
    return await this._fetchAndParse<ApiPipeline>(
      '/pipelines/upload',
      v1beta1Prefix,
      `name=${encodeURIComponent(pipelineName)}&description=${encodeURIComponent(
        pipelineDescription,
      )}`,
      {
        body: fd,
        cache: 'no-cache',
        method: 'POST',
      },
    );
  }

  public static async uploadPipelineVersion(
    versionName: string,
    pipelineId: string,
    versionData: File,
  ): Promise<ApiPipelineVersion> {
    const fd = new FormData();
    fd.append('uploadfile', versionData, versionData.name);
    return await this._fetchAndParse<ApiPipelineVersion>(
      '/pipelines/upload_version',
      v1beta1Prefix,
      `name=${encodeURIComponent(versionName)}&pipelineid=${encodeURIComponent(pipelineId)}`,
      {
        body: fd,
        cache: 'no-cache',
        method: 'POST',
      },
    );
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
    path: string,
    apisPrefix?: string,
    query?: string,
    init?: RequestInit,
  ): Promise<T> {
    const responseText = await this._fetch(path, apisPrefix, query, init);
    try {
      return JSON.parse(responseText) as T;
    } catch (err) {
      throw new Error(
        `Error parsing response for path: ${path}\n\n` +
          `Response was: ${responseText}\n\nError was: ${JSON.stringify(err)}`,
      );
    }
  }

  /**
   * Makes an HTTP request and returns the response as a string.
   *
   * Use this._fetchAndParse() if you intend to parse the response as JSON into an object.
   */
  private static async _fetch(
    path: string,
    apisPrefix?: string,
    query?: string,
    init?: RequestInit,
  ): Promise<string> {
    init = Object.assign(init || {}, { credentials: 'same-origin' });
    const response = await fetch((apisPrefix || '') + path + (query ? '?' + query : ''), init);
    const responseText = await response.text();
    if (response.ok) {
      return responseText;
    } else {
      Utils.logger.error(
        `Response for path: ${path} was not 'ok'\n\nResponse was: ${responseText}`,
      );
      throw new Error(responseText);
    }
  }
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum RunSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name',
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
  PIPELINE_ID = 'pipeline_id',
}

// Valid sortKeys as specified by the backend.
export enum ExperimentSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
}

// Valid sortKeys as specified by the backend.
export enum PipelineVersionSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name',
}
