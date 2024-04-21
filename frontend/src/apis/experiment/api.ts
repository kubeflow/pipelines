/// <reference path="./custom.d.ts" />
// tslint:disable
/**
 * backend/api/experiment.proto
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * OpenAPI spec version: version not set
 *
 *
 * NOTE: This class is auto generated by the swagger code generator program.
 * https://github.com/swagger-api/swagger-codegen.git
 * Do not edit the class manually.
 */

import * as url from 'url';
import * as portableFetch from 'portable-fetch';
import { Configuration } from './configuration';

const BASE_PATH = 'http://localhost'.replace(/\/+$/, '');

/**
 *
 * @export
 */
export const COLLECTION_FORMATS = {
  csv: ',',
  ssv: ' ',
  tsv: '\t',
  pipes: '|',
};

/**
 *
 * @export
 * @interface FetchAPI
 */
export interface FetchAPI {
  (url: string, init?: any): Promise<Response>;
}

/**
 *
 * @export
 * @interface FetchArgs
 */
export interface FetchArgs {
  url: string;
  options: any;
}

/**
 *
 * @export
 * @class BaseAPI
 */
export class BaseAPI {
  protected configuration: Configuration;

  constructor(
    configuration?: Configuration,
    protected basePath: string = BASE_PATH,
    protected fetch: FetchAPI = portableFetch,
  ) {
    if (configuration) {
      this.configuration = configuration;
      this.basePath = configuration.basePath || this.basePath;
    }
  }
}

/**
 *
 * @export
 * @class RequiredError
 * @extends {Error}
 */
export class RequiredError extends Error {
  name: 'RequiredError';
  constructor(public field: string, msg?: string) {
    super(msg);
  }
}

/**
 *
 * @export
 * @interface ApiExperiment
 */
export interface ApiExperiment {
  /**
   * Output. Unique experiment ID. Generated by API server.
   * @type {string}
   * @memberof ApiExperiment
   */
  id?: string;
  /**
   * Required input field. Unique experiment name provided by user.
   * @type {string}
   * @memberof ApiExperiment
   */
  name?: string;
  /**
   *
   * @type {string}
   * @memberof ApiExperiment
   */
  description?: string;
  /**
   * Output. The time that the experiment created.
   * @type {Date}
   * @memberof ApiExperiment
   */
  created_at?: Date;
  /**
   * Optional input field. Specify which resource this run belongs to. For Experiment, the only valid resource reference is a single Namespace.
   * @type {Array<ApiResourceReference>}
   * @memberof ApiExperiment
   */
  resource_references?: Array<ApiResourceReference>;
  /**
   * Output. Specifies whether this experiment is in archived or available state.
   * @type {ApiExperimentStorageState}
   * @memberof ApiExperiment
   */
  storage_state?: ApiExperimentStorageState;
}

/**
 *
 * @export
 * @enum {string}
 */
export enum ApiExperimentStorageState {
  UNSPECIFIED = <any>'STORAGESTATE_UNSPECIFIED',
  AVAILABLE = <any>'STORAGESTATE_AVAILABLE',
  ARCHIVED = <any>'STORAGESTATE_ARCHIVED',
}

/**
 *
 * @export
 * @interface ApiListExperimentsResponse
 */
export interface ApiListExperimentsResponse {
  /**
   * A list of experiments returned.
   * @type {Array<ApiExperiment>}
   * @memberof ApiListExperimentsResponse
   */
  experiments?: Array<ApiExperiment>;
  /**
   * The total number of experiments for the given query.
   * @type {number}
   * @memberof ApiListExperimentsResponse
   */
  total_size?: number;
  /**
   * The token to list the next page of experiments.
   * @type {string}
   * @memberof ApiListExperimentsResponse
   */
  next_page_token?: string;
}

/**
 *
 * @export
 * @enum {string}
 */
export enum ApiRelationship {
  UNKNOWNRELATIONSHIP = <any>'UNKNOWN_RELATIONSHIP',
  OWNER = <any>'OWNER',
  CREATOR = <any>'CREATOR',
}

/**
 *
 * @export
 * @interface ApiResourceKey
 */
export interface ApiResourceKey {
  /**
   * The type of the resource that referred to.
   * @type {ApiResourceType}
   * @memberof ApiResourceKey
   */
  type?: ApiResourceType;
  /**
   * The ID of the resource that referred to.
   * @type {string}
   * @memberof ApiResourceKey
   */
  id?: string;
}

/**
 *
 * @export
 * @interface ApiResourceReference
 */
export interface ApiResourceReference {
  /**
   *
   * @type {ApiResourceKey}
   * @memberof ApiResourceReference
   */
  key?: ApiResourceKey;
  /**
   * The name of the resource that referred to.
   * @type {string}
   * @memberof ApiResourceReference
   */
  name?: string;
  /**
   * Required field. The relationship from referred resource to the object.
   * @type {ApiRelationship}
   * @memberof ApiResourceReference
   */
  relationship?: ApiRelationship;
}

/**
 *
 * @export
 * @enum {string}
 */
export enum ApiResourceType {
  UNKNOWNRESOURCETYPE = <any>'UNKNOWN_RESOURCE_TYPE',
  EXPERIMENT = <any>'EXPERIMENT',
  JOB = <any>'JOB',
  PIPELINE = <any>'PIPELINE',
  PIPELINEVERSION = <any>'PIPELINE_VERSION',
  NAMESPACE = <any>'NAMESPACE',
}

/**
 *
 * @export
 * @interface ApiStatus
 */
export interface ApiStatus {
  /**
   *
   * @type {string}
   * @memberof ApiStatus
   */
  error?: string;
  /**
   *
   * @type {number}
   * @memberof ApiStatus
   */
  code?: number;
  /**
   *
   * @type {Array<ProtobufAny>}
   * @memberof ApiStatus
   */
  details?: Array<ProtobufAny>;
}

/**
 * `Any` contains an arbitrary serialized protocol buffer message along with a URL that describes the type of the serialized message.  Protobuf library provides support to pack/unpack Any values in the form of utility functions or additional generated methods of the Any type.  Example 1: Pack and unpack a message in C++.      Foo foo = ...;     Any any;     any.PackFrom(foo);     ...     if (any.UnpackTo(&foo)) {       ...     }  Example 2: Pack and unpack a message in Java.      Foo foo = ...;     Any any = Any.pack(foo);     ...     if (any.is(Foo.class)) {       foo = any.unpack(Foo.class);     }   Example 3: Pack and unpack a message in Python.      foo = Foo(...)     any = Any()     any.Pack(foo)     ...     if any.Is(Foo.DESCRIPTOR):       any.Unpack(foo)       ...   Example 4: Pack and unpack a message in Go       foo := &pb.Foo{...}      any, err := anypb.New(foo)      if err != nil {        ...      }      ...      foo := &pb.Foo{}      if err := any.UnmarshalTo(foo); err != nil {        ...      }  The pack methods provided by protobuf library will by default use 'type.googleapis.com/full.type.name' as the type URL and the unpack methods only use the fully qualified type name after the last '/' in the type URL, for example \"foo.bar.com/x/y.z\" will yield type name \"y.z\".   JSON ==== The JSON representation of an `Any` value uses the regular representation of the deserialized, embedded message, with an additional field `@type` which contains the type URL. Example:      package google.profile;     message Person {       string first_name = 1;       string last_name = 2;     }      {       \"@type\": \"type.googleapis.com/google.profile.Person\",       \"firstName\": <string>,       \"lastName\": <string>     }  If the embedded message type is well-known and has a custom JSON representation, that representation will be embedded adding a field `value` which holds the custom JSON in addition to the `@type` field. Example (for message [google.protobuf.Duration][]):      {       \"@type\": \"type.googleapis.com/google.protobuf.Duration\",       \"value\": \"1.212s\"     }
 * @export
 * @interface ProtobufAny
 */
export interface ProtobufAny {
  /**
   * A URL/resource name that uniquely identifies the type of the serialized protocol buffer message. This string must contain at least one \"/\" character. The last segment of the URL's path must represent the fully qualified name of the type (as in `path/google.protobuf.Duration`). The name should be in a canonical form (e.g., leading \".\" is not accepted).  In practice, teams usually precompile into the binary all types that they expect it to use in the context of Any. However, for URLs which use the scheme `http`, `https`, or no scheme, one can optionally set up a type server that maps type URLs to message definitions as follows:  * If no scheme is provided, `https` is assumed. * An HTTP GET on the URL must yield a [google.protobuf.Type][]   value in binary format, or produce an error. * Applications are allowed to cache lookup results based on the   URL, or have them precompiled into a binary to avoid any   lookup. Therefore, binary compatibility needs to be preserved   on changes to types. (Use versioned type names to manage   breaking changes.)  Note: this functionality is not currently available in the official protobuf release, and it is not used for type URLs beginning with type.googleapis.com.  Schemes other than `http`, `https` (or the empty scheme) might be used with implementation specific semantics.
   * @type {string}
   * @memberof ProtobufAny
   */
  type_url?: string;
  /**
   * Must be a valid serialized protocol buffer of the above specified type.
   * @type {string}
   * @memberof ProtobufAny
   */
  value?: string;
}

/**
 * ExperimentServiceApi - fetch parameter creator
 * @export
 */
export const ExperimentServiceApiFetchParamCreator = function(configuration?: Configuration) {
  return {
    /**
     *
     * @summary Archives an experiment and the experiment's runs and jobs.
     * @param {string} id The ID of the experiment to be archived.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    archiveExperiment(id: string, options: any = {}): FetchArgs {
      // verify required parameter 'id' is not null or undefined
      if (id === null || id === undefined) {
        throw new RequiredError(
          'id',
          'Required parameter id was null or undefined when calling archiveExperiment.',
        );
      }
      const localVarPath = `/apis/v1beta1/experiments/{id}:archive`.replace(
        `{${'id'}}`,
        encodeURIComponent(String(id)),
      );
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'POST' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
    /**
     *
     * @summary Creates a new experiment.
     * @param {ApiExperiment} body The experiment to be created.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createExperiment(body: ApiExperiment, options: any = {}): FetchArgs {
      // verify required parameter 'body' is not null or undefined
      if (body === null || body === undefined) {
        throw new RequiredError(
          'body',
          'Required parameter body was null or undefined when calling createExperiment.',
        );
      }
      const localVarPath = `/apis/v1beta1/experiments`;
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'POST' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      localVarHeaderParameter['Content-Type'] = 'application/json';

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);
      const needsSerialization =
        <any>'ApiExperiment' !== 'string' ||
        localVarRequestOptions.headers['Content-Type'] === 'application/json';
      localVarRequestOptions.body = needsSerialization ? JSON.stringify(body || {}) : body || '';

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
    /**
     *
     * @summary Deletes an experiment without deleting the experiment's runs and jobs. To avoid unexpected behaviors, delete an experiment's runs and jobs before deleting the experiment.
     * @param {string} id The ID of the experiment to be deleted.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteExperiment(id: string, options: any = {}): FetchArgs {
      // verify required parameter 'id' is not null or undefined
      if (id === null || id === undefined) {
        throw new RequiredError(
          'id',
          'Required parameter id was null or undefined when calling deleteExperiment.',
        );
      }
      const localVarPath = `/apis/v1beta1/experiments/{id}`.replace(
        `{${'id'}}`,
        encodeURIComponent(String(id)),
      );
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'DELETE' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
    /**
     *
     * @summary Finds a specific experiment by ID.
     * @param {string} id The ID of the experiment to be retrieved.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getExperiment(id: string, options: any = {}): FetchArgs {
      // verify required parameter 'id' is not null or undefined
      if (id === null || id === undefined) {
        throw new RequiredError(
          'id',
          'Required parameter id was null or undefined when calling getExperiment.',
        );
      }
      const localVarPath = `/apis/v1beta1/experiments/{id}`.replace(
        `{${'id'}}`,
        encodeURIComponent(String(id)),
      );
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'GET' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
    /**
     *
     * @summary Finds all experiments. Supports pagination, and sorting on certain fields.
     * @param {string} [page_token] A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page.
     * @param {number} [page_size] The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page.
     * @param {string} [sort_by] Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default.
     * @param {string} [filter] A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)).
     * @param {'UNKNOWN_RESOURCE_TYPE' | 'EXPERIMENT' | 'JOB' | 'PIPELINE' | 'PIPELINE_VERSION' | 'NAMESPACE'} [resource_reference_key_type] The type of the resource that referred to.
     * @param {string} [resource_reference_key_id] The ID of the resource that referred to.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listExperiment(
      page_token?: string,
      page_size?: number,
      sort_by?: string,
      filter?: string,
      resource_reference_key_type?:
        | 'UNKNOWN_RESOURCE_TYPE'
        | 'EXPERIMENT'
        | 'JOB'
        | 'PIPELINE'
        | 'PIPELINE_VERSION'
        | 'NAMESPACE',
      resource_reference_key_id?: string,
      options: any = {},
    ): FetchArgs {
      const localVarPath = `/apis/v1beta1/experiments`;
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'GET' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      if (page_token !== undefined) {
        localVarQueryParameter['page_token'] = page_token;
      }

      if (page_size !== undefined) {
        localVarQueryParameter['page_size'] = page_size;
      }

      if (sort_by !== undefined) {
        localVarQueryParameter['sort_by'] = sort_by;
      }

      if (filter !== undefined) {
        localVarQueryParameter['filter'] = filter;
      }

      if (resource_reference_key_type !== undefined) {
        localVarQueryParameter['resource_reference_key.type'] = resource_reference_key_type;
      }

      if (resource_reference_key_id !== undefined) {
        localVarQueryParameter['resource_reference_key.id'] = resource_reference_key_id;
      }

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
    /**
     *
     * @summary Restores an archived experiment. The experiment's archived runs and jobs will stay archived.
     * @param {string} id The ID of the experiment to be restored.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    unarchiveExperiment(id: string, options: any = {}): FetchArgs {
      // verify required parameter 'id' is not null or undefined
      if (id === null || id === undefined) {
        throw new RequiredError(
          'id',
          'Required parameter id was null or undefined when calling unarchiveExperiment.',
        );
      }
      const localVarPath = `/apis/v1beta1/experiments/{id}:unarchive`.replace(
        `{${'id'}}`,
        encodeURIComponent(String(id)),
      );
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'POST' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
  };
};

/**
 * ExperimentServiceApi - functional programming interface
 * @export
 */
export const ExperimentServiceApiFp = function(configuration?: Configuration) {
  return {
    /**
     *
     * @summary Archives an experiment and the experiment's runs and jobs.
     * @param {string} id The ID of the experiment to be archived.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    archiveExperiment(
      id: string,
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<any> {
      const localVarFetchArgs = ExperimentServiceApiFetchParamCreator(
        configuration,
      ).archiveExperiment(id, options);
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
    /**
     *
     * @summary Creates a new experiment.
     * @param {ApiExperiment} body The experiment to be created.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createExperiment(
      body: ApiExperiment,
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<ApiExperiment> {
      const localVarFetchArgs = ExperimentServiceApiFetchParamCreator(
        configuration,
      ).createExperiment(body, options);
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
    /**
     *
     * @summary Deletes an experiment without deleting the experiment's runs and jobs. To avoid unexpected behaviors, delete an experiment's runs and jobs before deleting the experiment.
     * @param {string} id The ID of the experiment to be deleted.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteExperiment(
      id: string,
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<any> {
      const localVarFetchArgs = ExperimentServiceApiFetchParamCreator(
        configuration,
      ).deleteExperiment(id, options);
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
    /**
     *
     * @summary Finds a specific experiment by ID.
     * @param {string} id The ID of the experiment to be retrieved.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getExperiment(
      id: string,
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<ApiExperiment> {
      const localVarFetchArgs = ExperimentServiceApiFetchParamCreator(configuration).getExperiment(
        id,
        options,
      );
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
    /**
     *
     * @summary Finds all experiments. Supports pagination, and sorting on certain fields.
     * @param {string} [page_token] A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page.
     * @param {number} [page_size] The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page.
     * @param {string} [sort_by] Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default.
     * @param {string} [filter] A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)).
     * @param {'UNKNOWN_RESOURCE_TYPE' | 'EXPERIMENT' | 'JOB' | 'PIPELINE' | 'PIPELINE_VERSION' | 'NAMESPACE'} [resource_reference_key_type] The type of the resource that referred to.
     * @param {string} [resource_reference_key_id] The ID of the resource that referred to.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listExperiment(
      page_token?: string,
      page_size?: number,
      sort_by?: string,
      filter?: string,
      resource_reference_key_type?:
        | 'UNKNOWN_RESOURCE_TYPE'
        | 'EXPERIMENT'
        | 'JOB'
        | 'PIPELINE'
        | 'PIPELINE_VERSION'
        | 'NAMESPACE',
      resource_reference_key_id?: string,
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<ApiListExperimentsResponse> {
      const localVarFetchArgs = ExperimentServiceApiFetchParamCreator(configuration).listExperiment(
        page_token,
        page_size,
        sort_by,
        filter,
        resource_reference_key_type,
        resource_reference_key_id,
        options,
      );
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
    /**
     *
     * @summary Restores an archived experiment. The experiment's archived runs and jobs will stay archived.
     * @param {string} id The ID of the experiment to be restored.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    unarchiveExperiment(
      id: string,
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<any> {
      const localVarFetchArgs = ExperimentServiceApiFetchParamCreator(
        configuration,
      ).unarchiveExperiment(id, options);
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
  };
};

/**
 * ExperimentServiceApi - factory interface
 * @export
 */
export const ExperimentServiceApiFactory = function(
  configuration?: Configuration,
  fetch?: FetchAPI,
  basePath?: string,
) {
  return {
    /**
     *
     * @summary Archives an experiment and the experiment's runs and jobs.
     * @param {string} id The ID of the experiment to be archived.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    archiveExperiment(id: string, options?: any) {
      return ExperimentServiceApiFp(configuration).archiveExperiment(id, options)(fetch, basePath);
    },
    /**
     *
     * @summary Creates a new experiment.
     * @param {ApiExperiment} body The experiment to be created.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    createExperiment(body: ApiExperiment, options?: any) {
      return ExperimentServiceApiFp(configuration).createExperiment(body, options)(fetch, basePath);
    },
    /**
     *
     * @summary Deletes an experiment without deleting the experiment's runs and jobs. To avoid unexpected behaviors, delete an experiment's runs and jobs before deleting the experiment.
     * @param {string} id The ID of the experiment to be deleted.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    deleteExperiment(id: string, options?: any) {
      return ExperimentServiceApiFp(configuration).deleteExperiment(id, options)(fetch, basePath);
    },
    /**
     *
     * @summary Finds a specific experiment by ID.
     * @param {string} id The ID of the experiment to be retrieved.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    getExperiment(id: string, options?: any) {
      return ExperimentServiceApiFp(configuration).getExperiment(id, options)(fetch, basePath);
    },
    /**
     *
     * @summary Finds all experiments. Supports pagination, and sorting on certain fields.
     * @param {string} [page_token] A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page.
     * @param {number} [page_size] The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page.
     * @param {string} [sort_by] Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default.
     * @param {string} [filter] A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)).
     * @param {'UNKNOWN_RESOURCE_TYPE' | 'EXPERIMENT' | 'JOB' | 'PIPELINE' | 'PIPELINE_VERSION' | 'NAMESPACE'} [resource_reference_key_type] The type of the resource that referred to.
     * @param {string} [resource_reference_key_id] The ID of the resource that referred to.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    listExperiment(
      page_token?: string,
      page_size?: number,
      sort_by?: string,
      filter?: string,
      resource_reference_key_type?:
        | 'UNKNOWN_RESOURCE_TYPE'
        | 'EXPERIMENT'
        | 'JOB'
        | 'PIPELINE'
        | 'PIPELINE_VERSION'
        | 'NAMESPACE',
      resource_reference_key_id?: string,
      options?: any,
    ) {
      return ExperimentServiceApiFp(configuration).listExperiment(
        page_token,
        page_size,
        sort_by,
        filter,
        resource_reference_key_type,
        resource_reference_key_id,
        options,
      )(fetch, basePath);
    },
    /**
     *
     * @summary Restores an archived experiment. The experiment's archived runs and jobs will stay archived.
     * @param {string} id The ID of the experiment to be restored.
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    unarchiveExperiment(id: string, options?: any) {
      return ExperimentServiceApiFp(configuration).unarchiveExperiment(id, options)(
        fetch,
        basePath,
      );
    },
  };
};

/**
 * ExperimentServiceApi - object-oriented interface
 * @export
 * @class ExperimentServiceApi
 * @extends {BaseAPI}
 */
export class ExperimentServiceApi extends BaseAPI {
  /**
   *
   * @summary Archives an experiment and the experiment's runs and jobs.
   * @param {string} id The ID of the experiment to be archived.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof ExperimentServiceApi
   */
  public archiveExperiment(id: string, options?: any) {
    return ExperimentServiceApiFp(this.configuration).archiveExperiment(id, options)(
      this.fetch,
      this.basePath,
    );
  }

  /**
   *
   * @summary Creates a new experiment.
   * @param {ApiExperiment} body The experiment to be created.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof ExperimentServiceApi
   */
  public createExperiment(body: ApiExperiment, options?: any) {
    return ExperimentServiceApiFp(this.configuration).createExperiment(body, options)(
      this.fetch,
      this.basePath,
    );
  }

  /**
   *
   * @summary Deletes an experiment without deleting the experiment's runs and jobs. To avoid unexpected behaviors, delete an experiment's runs and jobs before deleting the experiment.
   * @param {string} id The ID of the experiment to be deleted.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof ExperimentServiceApi
   */
  public deleteExperiment(id: string, options?: any) {
    return ExperimentServiceApiFp(this.configuration).deleteExperiment(id, options)(
      this.fetch,
      this.basePath,
    );
  }

  /**
   *
   * @summary Finds a specific experiment by ID.
   * @param {string} id The ID of the experiment to be retrieved.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof ExperimentServiceApi
   */
  public getExperiment(id: string, options?: any) {
    return ExperimentServiceApiFp(this.configuration).getExperiment(id, options)(
      this.fetch,
      this.basePath,
    );
  }

  /**
   *
   * @summary Finds all experiments. Supports pagination, and sorting on certain fields.
   * @param {string} [page_token] A page token to request the next page of results. The token is acquried from the nextPageToken field of the response from the previous ListExperiment call or can be omitted when fetching the first page.
   * @param {number} [page_size] The number of experiments to be listed per page. If there are more experiments than this number, the response message will contain a nextPageToken field you can use to fetch the next page.
   * @param {string} [sort_by] Can be format of \&quot;field_name\&quot;, \&quot;field_name asc\&quot; or \&quot;field_name desc\&quot; Ascending by default.
   * @param {string} [filter] A url-encoded, JSON-serialized Filter protocol buffer (see [filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/v1beta1/filter.proto)).
   * @param {'UNKNOWN_RESOURCE_TYPE' | 'EXPERIMENT' | 'JOB' | 'PIPELINE' | 'PIPELINE_VERSION' | 'NAMESPACE'} [resource_reference_key_type] The type of the resource that referred to.
   * @param {string} [resource_reference_key_id] The ID of the resource that referred to.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof ExperimentServiceApi
   */
  public listExperiment(
    page_token?: string,
    page_size?: number,
    sort_by?: string,
    filter?: string,
    resource_reference_key_type?:
      | 'UNKNOWN_RESOURCE_TYPE'
      | 'EXPERIMENT'
      | 'JOB'
      | 'PIPELINE'
      | 'PIPELINE_VERSION'
      | 'NAMESPACE',
    resource_reference_key_id?: string,
    options?: any,
  ) {
    return ExperimentServiceApiFp(this.configuration).listExperiment(
      page_token,
      page_size,
      sort_by,
      filter,
      resource_reference_key_type,
      resource_reference_key_id,
      options,
    )(this.fetch, this.basePath);
  }

  /**
   *
   * @summary Restores an archived experiment. The experiment's archived runs and jobs will stay archived.
   * @param {string} id The ID of the experiment to be restored.
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof ExperimentServiceApi
   */
  public unarchiveExperiment(id: string, options?: any) {
    return ExperimentServiceApiFp(this.configuration).unarchiveExperiment(id, options)(
      this.fetch,
      this.basePath,
    );
  }
}
