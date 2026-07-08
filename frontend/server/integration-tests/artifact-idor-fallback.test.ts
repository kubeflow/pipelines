// Copyright 2026 The Kubeflow Authors
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

// Unlike artifact-auth.test.ts, this suite does NOT mock the MLMD validator.
// It runs the real validator through the artifacts auth middleware, mocking
// only the gRPC-web transport, so it covers the wiring that decides what
// happens to objects the metadata store does not track (archived pod logs,
// v1 artifacts). That untested wiring is how the deny-all regression fixed
// by the prefix fallback shipped unnoticed.

import { vi, describe, it, expect, afterAll, afterEach, beforeEach } from 'vitest';
import { createRequire } from 'module';
import * as minio from 'minio';
import { PassThrough } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { commonSetup } from './test-helper.js';

const require = createRequire(import.meta.url);
// eslint-disable-next-line @typescript-eslint/no-var-requires
const servicePb = require('../../src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb.js');
// eslint-disable-next-line @typescript-eslint/no-var-requires
const storePb = require('../../src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb.js');

const MinioClient = minio.Client;
vi.mock('minio');
vi.mock('../k8s-helper.js');

const mockedFetch = vi.fn();
vi.stubGlobal('fetch', mockedFetch);

// Wraps a serialized proto message in gRPC-web framing: one data frame
// followed by a trailer frame carrying grpc-status 0.
function grpcWebResponseBuffer(messageBytes: Uint8Array): ArrayBuffer {
  const trailer = new TextEncoder().encode('grpc-status: 0\r\n');
  const framed = new Uint8Array(5 + messageBytes.length + 5 + trailer.length);
  const view = new DataView(framed.buffer);
  framed[0] = 0x00; // data frame
  view.setUint32(1, messageBytes.length, false);
  framed.set(messageBytes, 5);
  const trailerOffset = 5 + messageBytes.length;
  framed[trailerOffset] = 0x80; // trailer frame
  view.setUint32(trailerOffset + 1, trailer.length, false);
  framed.set(trailer, trailerOffset + 5);
  return framed.buffer;
}

function emptyArtifactsResponse(): ArrayBuffer {
  return grpcWebResponseBuffer(new servicePb.GetArtifactsByURIResponse().serializeBinary());
}

function trackedArtifactsResponse(artifactId: number): ArrayBuffer {
  const response = new servicePb.GetArtifactsByURIResponse();
  response.addArtifacts().setId(artifactId);
  return grpcWebResponseBuffer(response.serializeBinary());
}

function pipelineRunContextsResponse(namespace: string): ArrayBuffer {
  const response = new servicePb.GetContextsByArtifactResponse();
  const context = response.addContexts();
  context.setId(1);
  context.setType('system.PipelineRun');
  const namespaceValue = new storePb.Value();
  namespaceValue.setStringValue(namespace);
  context.getCustomPropertiesMap().set('namespace', namespaceValue);
  return grpcWebResponseBuffer(response.serializeBinary());
}

describe('artifact IDOR guard with the real MLMD validator', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  const artifactContent = 'hello world';
  const archivedLogKey =
    'private-artifacts/team-a/flip-coin-abc/2026/07/08/flip-coin-abc-1/main.log';
  const victimLogKey =
    'private-artifacts/victim-ns/flip-coin-def/2026/07/08/flip-coin-def-1/main.log';
  const unprefixedKey = 'some/other/object.txt';
  const trackedArtifactKey = 'private-artifacts/team-a/v2/artifacts/run-1/op/output';

  let getObjectMock: ReturnType<typeof vi.fn>;

  // Routes MLMD gRPC-web calls to canned proto responses and approves every
  // other fetch, which is the Subject Access Review against the API server:
  // the requester legitimately has RBAC access to the namespace they claim.
  const mockMlmd = ({
    artifacts,
    contexts,
  }: {
    artifacts: ArrayBuffer;
    contexts?: ArrayBuffer;
  }) => {
    mockedFetch.mockImplementation((url: unknown) => {
      const target = String(url);
      if (target.includes('/ml_metadata.MetadataStoreService/GetArtifactsByURI')) {
        return Promise.resolve({ ok: true, arrayBuffer: () => Promise.resolve(artifacts) });
      }
      if (target.includes('/ml_metadata.MetadataStoreService/GetContextsByArtifact')) {
        return Promise.resolve({
          ok: true,
          arrayBuffer: () => Promise.resolve(contexts ?? emptyArtifactsResponse()),
        });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(''),
      });
    });
  };

  const authEnabledConfigs = () => {
    const configurations = loadConfigs(argv, {
      MINIO_ACCESS_KEY: 'minio',
      MINIO_HOST: 'minio-service',
      MINIO_NAMESPACE: 'kubeflow',
      MINIO_PORT: '9000',
      MINIO_SECRET_KEY: 'minio123',
      MINIO_SSL: 'false',
      ML_PIPELINE_SERVICE_HOST: 'localhost',
      ML_PIPELINE_SERVICE_PORT: '8888',
      KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
      KUBEFLOW_USERID_PREFIX: '',
    });
    configurations.auth.enabled = true;
    return configurations;
  };

  beforeEach(() => {
    mockedFetch.mockReset();

    const servedKeys = new Set([archivedLogKey, victimLogKey, unprefixedKey, trackedArtifactKey]);
    getObjectMock = vi.fn(async (bucket: string, key: string) => {
      if (bucket === 'mlpipeline' && servedKeys.has(key)) {
        const objStream = new PassThrough();
        objStream.end(artifactContent);
        return objStream;
      }
      throw new Error(`Unable to retrieve ${bucket}/${key} artifact.`);
    });
    const mockedMinioClient = MinioClient as any;
    mockedMinioClient.mockImplementation(function () {
      return { getObject: getObjectMock };
    });
  });

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  afterEach(async () => {
    if (app) {
      await app.close();
    }
  });

  it('serves an archived pod log absent from MLMD when its object-key prefix matches the claimed namespace', async () => {
    // Regression guard: archived Argo pod logs are stored under the
    // private-artifacts/<namespace>/ keyFormat but are never registered as
    // MLMD artifacts, so a validator that denies everything unknown to MLMD
    // breaks all v1 log and artifact retrieval.
    mockMlmd({ artifacts: emptyArtifactsResponse() });

    app = new UIServer(authEnabledConfigs());

    await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=mlpipeline&key=${encodeURIComponent(
          archivedLogKey,
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'user@example.com')
      .expect(200, artifactContent);

    const mlmdLookup = mockedFetch.mock.calls.find((call) =>
      String(call[0]).includes('GetArtifactsByURI'),
    );
    expect(mlmdLookup).toBeDefined();
  });

  it('denies an artifact absent from MLMD when its object-key prefix names another namespace', async () => {
    mockMlmd({ artifacts: emptyArtifactsResponse() });

    app = new UIServer(authEnabledConfigs());

    const response = await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=mlpipeline&key=${encodeURIComponent(
          victimLogKey,
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'attacker@example.com')
      .expect(403);
    expect(response.text).toContain('does not belong to the requested namespace');
    expect(getObjectMock).not.toHaveBeenCalled();
  });

  it('denies an artifact absent from MLMD that carries no owning-namespace prefix', async () => {
    mockMlmd({ artifacts: emptyArtifactsResponse() });

    app = new UIServer(authEnabledConfigs());

    const response = await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=mlpipeline&key=${encodeURIComponent(
          unprefixedKey,
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'user@example.com')
      .expect(403);
    expect(response.text).toContain('does not belong to the requested namespace');
    expect(getObjectMock).not.toHaveBeenCalled();
  });

  it('denies a dot-segment key that spoofs the claimed namespace prefix', async () => {
    // The key is caller-controlled; "private-artifacts/team-a/../victim-ns/obj"
    // carries team-a's prefix while addressing victim-ns's object on stores
    // that normalize paths, so the fallback must deny non-normalized keys.
    mockMlmd({ artifacts: emptyArtifactsResponse() });

    app = new UIServer(authEnabledConfigs());

    const response = await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=mlpipeline&key=${encodeURIComponent(
          'private-artifacts/team-a/../victim-ns/some/object',
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'attacker@example.com')
      .expect(403);
    expect(response.text).toContain('does not belong to the requested namespace');
    expect(getObjectMock).not.toHaveBeenCalled();
  });

  it('denies a bucket-per-namespace v1 artifact absent from MLMD under the default prefix mode', async () => {
    // Documents the compatibility gap discussed on the fix PR: v1 deployments
    // that isolate namespaces with a dedicated bucket per namespace store
    // artifacts like "s3://team-a-bucket/<workflow>/<pod>/main.log" with no
    // owning-namespace key prefix, so the default mlmd-then-prefix mode denies
    // them. This pins the default-mode behavior; if the proposed opt-in
    // bucket-ownership mode lands, that mode needs its own allow/deny cases
    // while this default-mode denial must keep holding for shared buckets.
    mockMlmd({ artifacts: emptyArtifactsResponse() });

    app = new UIServer(authEnabledConfigs());

    const response = await requests(app.app)
      .get(
        `/artifacts/get?source=s3&bucket=team-a-bucket&key=${encodeURIComponent(
          'flip-coin-abc/flip-coin-abc-1/main.log',
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'user@example.com')
      .expect(403);
    expect(response.text).toContain('does not belong to the requested namespace');
    expect(getObjectMock).not.toHaveBeenCalled();
  });

  it('prefers MLMD namespace evidence over a matching object-key prefix', async () => {
    // The prefix fallback must only apply when MLMD has no record. When MLMD
    // attributes the artifact to another namespace, that evidence wins even
    // though the object key carries the claimed namespace's prefix.
    mockMlmd({
      artifacts: trackedArtifactsResponse(7),
      contexts: pipelineRunContextsResponse('victim-ns'),
    });

    app = new UIServer(authEnabledConfigs());

    const response = await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=mlpipeline&key=${encodeURIComponent(
          trackedArtifactKey,
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'attacker@example.com')
      .expect(403);
    expect(response.text).toContain('does not belong to the requested namespace');
    expect(getObjectMock).not.toHaveBeenCalled();
  });

  it('serves an MLMD-tracked artifact whose PipelineRun context matches the claimed namespace', async () => {
    mockMlmd({
      artifacts: trackedArtifactsResponse(7),
      contexts: pipelineRunContextsResponse('team-a'),
    });

    app = new UIServer(authEnabledConfigs());

    await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=mlpipeline&key=${encodeURIComponent(
          trackedArtifactKey,
        )}&namespace=team-a`,
      )
      .set('kubeflow-userid', 'user@example.com')
      .expect(200, artifactContent);
  });
});
