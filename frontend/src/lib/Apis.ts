// Copyright 2018 The Kubeflow Authors
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
import { ExperimentServiceApi, FetchAPI } from 'src/apis/experiment';
import { ExperimentServiceApi as ExperimentServiceApiV2 } from 'src/apisv2beta1/experiment';
import { JobServiceApi } from 'src/apis/job';
import { RecurringRunServiceApi } from 'src/apisv2beta1/recurringrun';
import { ApiPipeline, ApiPipelineVersion, PipelineServiceApi } from 'src/apis/pipeline';
import {
  V2beta1Pipeline,
  V2beta1PipelineVersion,
  PipelineServiceApi as PipelineServiceApiV2,
} from 'src/apisv2beta1/pipeline';
import { RunServiceApi as RunServiceApiV1 } from 'src/apis/run';
import { RunServiceApi as RunServiceApiV2 } from 'src/apisv2beta1/run';
import { ApiVisualization, VisualizationServiceApi } from 'src/apis/visualization';
import { HTMLViewerConfig } from 'src/components/viewers/HTMLViewer';
import { PlotType } from 'src/components/viewers/Viewer';
import * as Utils from './Utils';
import { buildQuery } from './Utils';
import { StoragePath, StorageService } from './WorkflowParser';

const v1beta1Prefix = 'apis/v1beta1';
const v2beta1Prefix = 'apis/v2beta1';

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
  apiServerMultiUser?: boolean;
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
  public static getPodLogs(
    runId: string,
    podName: string,
    podNamespace: string,
    createdAt: string,
  ): Promise<string> {
    let query = `k8s/pod/logs?podname=${encodeURIComponent(podName)}&runid=${encodeURIComponent(
      runId,
    )}`;
    if (podNamespace) {
      query += `&podnamespace=${encodeURIComponent(podNamespace)}`;
    }
    if (createdAt) {
      query += `&createdat=${encodeURIComponent(createdAt)}`;
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

  // TODO(jlyaoyuli): deprecrate v1 experimentServiceApi function after all integrations.
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

  // Add v2 experimentServiceV2 for partial integration
  public static get experimentServiceApiV2(): ExperimentServiceApiV2 {
    if (!this._experimentServiceApiV2) {
      this._experimentServiceApiV2 = new ExperimentServiceApiV2(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._experimentServiceApiV2;
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

  public static get recurringRunServiceApi(): RecurringRunServiceApi {
    if (!this._recurringRunServiceApi) {
      this._recurringRunServiceApi = new RecurringRunServiceApi(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._recurringRunServiceApi;
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

  public static get pipelineServiceApiV2(): PipelineServiceApiV2 {
    if (!this._pipelineServiceApiV2) {
      this._pipelineServiceApiV2 = new PipelineServiceApiV2(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._pipelineServiceApiV2;
  }

  public static get runServiceApi(): RunServiceApiV1 {
    if (!this._runServiceApiV1) {
      this._runServiceApiV1 = new RunServiceApiV1(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._runServiceApiV1;
  }

  public static get runServiceApiV2(): RunServiceApiV2 {
    if (!this._runServiceApiV2) {
      this._runServiceApiV2 = new RunServiceApiV2(
        { basePath: this.basePath },
        undefined,
        crossBrowserFetch,
      );
    }
    return this._runServiceApiV2;
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
  public static readFile({
    path,
    providerInfo,
    namespace,
    peek,
  }: {
    path: StoragePath;
    namespace?: string;
    providerInfo?: string;
    peek?: number;
  }): Promise<string> {
    let query = this.buildReadFileUrl({ path, namespace, providerInfo, peek, isDownload: false });
    return this._fetch(query);
  }

  /**
   * Builds an url for the readFile API to retrieve a workflow artifact.
   * @param path object describing the artifact (e.g. source, bucket, and key)
   * @param namespace the experiment namespace
   * @param providerInfo non default provider info to build session for the object store where the artifact is stored
   * @param peek the amount used to be read
   * @param isDownload whether we download the artifact as is (e.g. skip extracting from *.tar.gz)
   */
  public static buildReadFileUrl({
    path,
    namespace,
    providerInfo,
    peek,
    isDownload,
  }: {
    path: StoragePath;
    namespace?: string;
    providerInfo?: string;
    peek?: number;
    isDownload?: boolean;
  }) {
    const { source, bucket, key } = path;
    if (isDownload) {
      return `artifacts/${source}/${bucket}/${key}${buildQuery({
        namespace,
        providerInfo,
        peek,
      })}`;
    } else {
      return `artifacts/get${buildQuery({ source, namespace, providerInfo, peek, bucket, key })}`;
    }
  }

  /**
   * Builds an url to visually represents a workflow artifact location.
   * @param param.source source of the artifact (e.g. minio, gcs, s3, http, or https)
   * @param param.bucket name of the bucket with the artifact (or host for http/https)
   * @param param.key key (i.e. path) of the artifact in the bucket
   */
  public static buildArtifactLinkText({ source, bucket, key }: StoragePath) {
    // TODO see https://github.com/kubeflow/pipelines/pull/3725
    let platform: string = source;
    if (source === StorageService.GCS) {
      platform = 'gs'; // GCS link starts with gs://
    }
    return `${platform}://${bucket}/${key}`;
  }

  /**
   * Gets the address (IP + port) of a Tensorboard service given the logdir and tfversion
   */
  public static getTensorboardApp(
    logdir: string,
    namespace: string,
  ): Promise<{ podAddress: string; tfVersion: string; image: string }> {
    return this._fetchAndParse<{ podAddress: string; tfVersion: string; image: string }>(
      `apps/tensorboard${buildQuery({ logdir, namespace })}`,
    );
  }

  /**
   * Starts a deployment and service for Tensorboard given the logdir.
   */
  public static startTensorboardApp({
    logdir,
    namespace,
    image,
    podTemplateSpec,
  }: {
    logdir: string;
    namespace: string;
    image?: string;
    podTemplateSpec?: any;
  }): Promise<string> {
    return this._fetch(
      `apps/tensorboard${buildQuery({
        logdir,
        namespace,
        image,
        podtemplatespec: podTemplateSpec && JSON.stringify(podTemplateSpec),
      })}`,
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
    namespace?: string,
  ): Promise<ApiPipeline> {
    const fd = new FormData();
    fd.append('uploadfile', pipelineData, pipelineData.name);
    let query = `name=${encodeURIComponent(pipelineName)}&description=${encodeURIComponent(
      pipelineDescription,
    )}`;

    if (namespace) {
      query = `${query}&namespace=${encodeURIComponent(namespace)}`;
    }

    return await this._fetchAndParse<ApiPipeline>('/pipelines/upload', v1beta1Prefix, query, {
      body: fd,
      cache: 'no-cache',
      method: 'POST',
    });
  }

  public static async uploadPipelineVersion(
    versionName: string,
    pipelineId: string,
    versionData: File,
    description?: string,
  ): Promise<ApiPipelineVersion> {
    const fd = new FormData();
    fd.append('uploadfile', versionData, versionData.name);
    return await this._fetchAndParse<ApiPipelineVersion>(
      '/pipelines/upload_version',
      v1beta1Prefix,
      `name=${encodeURIComponent(versionName)}&pipelineid=${encodeURIComponent(pipelineId)}` +
        (description ? `&description=${encodeURIComponent(description)}` : ''),
      {
        body: fd,
        cache: 'no-cache',
        method: 'POST',
      },
    );
  }

  public static async uploadPipelineV2(
    pipelineName: string,
    pipelineDescription: string,
    pipelineData: File,
    namespace?: string,
  ): Promise<V2beta1Pipeline> {
    const fd = new FormData();
    fd.append('uploadfile', pipelineData, pipelineData.name);
    let query = `name=${encodeURIComponent(pipelineName)}&description=${encodeURIComponent(
      pipelineDescription,
    )}`;

    if (namespace) {
      query = `${query}&namespace=${encodeURIComponent(namespace)}`;
    }

    return await this._fetchAndParse<V2beta1Pipeline>('/pipelines/upload', v2beta1Prefix, query, {
      body: fd,
      cache: 'no-cache',
      method: 'POST',
    });
  }

  public static async uploadPipelineVersionV2(
    versionName: string,
    pipelineId: string,
    versionData: File,
    description?: string,
  ): Promise<V2beta1PipelineVersion> {
    const fd = new FormData();
    fd.append('uploadfile', versionData, versionData.name);
    return await this._fetchAndParse<V2beta1PipelineVersion>(
      '/pipelines/upload_version',
      v2beta1Prefix,
      `name=${encodeURIComponent(versionName)}&pipelineid=${encodeURIComponent(pipelineId)}` +
        (description ? `&description=${encodeURIComponent(description)}` : ''),
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
  private static _experimentServiceApiV2?: ExperimentServiceApiV2;
  private static _jobServiceApi?: JobServiceApi;
  private static _recurringRunServiceApi?: RecurringRunServiceApi;
  private static _pipelineServiceApi?: PipelineServiceApi;
  private static _pipelineServiceApiV2?: PipelineServiceApiV2;
  private static _runServiceApiV1?: RunServiceApiV1;
  private static _runServiceApiV2?: RunServiceApiV2;
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
  LAST_RUN_CREATED_AT = 'last_run_created_at',
}

// Valid sortKeys as specified by the backend.
export enum PipelineVersionSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name',
}
