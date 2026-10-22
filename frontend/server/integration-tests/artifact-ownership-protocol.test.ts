// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import express from 'express';
import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';
import { createRequire } from 'node:module';
import requests from 'supertest';
import { BinaryWriter } from 'google-protobuf';

// This is a protocol fixture, not a live Kubernetes/MLMD installation. Namespace
// authorization and metadata responses are deterministic servers. The production
// authorization client, validator, protobufs, proxy, fetch and serving code run
// unmocked across real HTTP listeners. No Kubernetes calls are needed by this path.
vi.mock('../k8s-helper.js', () => ({ getK8sSecret: vi.fn(), getPod: vi.fn() }));

const require = createRequire(import.meta.url);
const servicePb =
  require('../../src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb.js') as typeof import('../../src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb.js');
const storePb =
  require('../../src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb.js') as typeof import('../../src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb.js');

type Artifact = { id: number; namespace?: string; unavailable?: boolean };

function grpcFrame(bytes: Uint8Array, trailer = false): Buffer {
  const header = Buffer.alloc(5);
  header[0] = trailer ? 0x80 : 0;
  header.writeUInt32BE(bytes.length, 1);
  return Buffer.concat([header, bytes]);
}

describe('artifact ownership compatibility over real HTTP and protobuf transport', () => {
  const namespace = 'team-a';
  const user = 'alice@example.test';
  const storageToken = 'synthetic-namespace-token';
  const servers: Server[] = [];
  let records: Map<string, Artifact[]>;
  let lookupUnavailable: boolean;
  let storageOrigin: string;
  let metadataOrigin: string;
  let metadataUris: string[];
  let contextIds: number[];
  let authorizationCalls: Array<{ namespace: string | null; user: string | undefined }>;
  let storageCalls: Array<{ path: string; authorization: string | undefined; user: unknown }>;
  let protocolErrors: string[];

  async function listen(server: Server): Promise<string> {
    servers.push(server);
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
    const address = server.address();
    if (!address || typeof address === 'string') throw new Error('Missing listener port');
    return `http://127.0.0.1:${address.port}`;
  }

  async function handleMetadata(req: IncomingMessage, res: ServerResponse) {
    const url = new URL(req.url!, 'http://fixture');
    if (url.pathname === '/apis/v1beta1/auth') {
      const requestedNamespace = url.searchParams.get('namespace');
      const identity = req.headers['user'];
      authorizationCalls.push({
        namespace: requestedNamespace,
        user: typeof identity === 'string' ? identity : undefined,
      });
      const allowed =
        requestedNamespace === namespace &&
        identity === user &&
        url.searchParams.get('resources') === 'VIEWERS' &&
        url.searchParams.get('verb') === 'GET';
      res.writeHead(allowed ? 200 : 403, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(allowed ? {} : { error: 'Namespace viewer access denied' }));
      return;
    }
    const chunks: Buffer[] = [];
    for await (const chunk of req) chunks.push(Buffer.from(chunk));
    const body = Buffer.concat(chunks);
    if (
      req.method !== 'POST' ||
      req.headers['content-type'] !== 'application/grpc-web+proto' ||
      body.length < 5 ||
      body[0] !== 0 ||
      body.readUInt32BE(1) !== body.length - 5
    ) {
      throw new Error('Invalid gRPC-web request framing');
    }
    let bytes: Uint8Array;
    if (url.pathname.endsWith('/GetArtifactsByURI')) {
      const request = servicePb.GetArtifactsByURIRequest.deserializeBinary(body.subarray(5));
      const uris = request.getUrisList();
      metadataUris.push(...uris);
      if (lookupUnavailable) {
        res.writeHead(503).end();
        return;
      }
      const response = new servicePb.GetArtifactsByURIResponse();
      for (const uri of uris) {
        for (const record of records.get(uri) || []) {
          const artifact = new storePb.Artifact();
          artifact.setId(record.id);
          artifact.setUri(uri);
          response.addArtifacts(artifact);
        }
      }
      bytes = response.serializeBinary();
    } else if (url.pathname.endsWith('/GetContextsByArtifact')) {
      const request = servicePb.GetContextsByArtifactRequest.deserializeBinary(body.subarray(5));
      const id = request.getArtifactId();
      contextIds.push(id);
      const record = [...records.values()].flat().find((artifact) => artifact.id === id);
      if (!record || record.unavailable) {
        res.writeHead(503).end();
        return;
      }
      const writer = new BinaryWriter();
      if (record.namespace) {
        const context = new storePb.Context();
        context.setType('system.PipelineRun');
        const value = new storePb.Value();
        value.setStringValue(record.namespace);
        // Encode the fixture's map entry directly: the generated legacy map
        // serializer is absent from google-protobuf 4. Production still uses
        // its generated Context/Value decoder for this actual wire payload.
        writer.writeMessage(1, context, (message, contextWriter) => {
          storePb.Context.serializeBinaryToWriter(message, contextWriter);
          contextWriter.writeMessage(5, value, (namespaceValue, mapWriter) => {
            mapWriter.writeString(1, 'namespace');
            mapWriter.writeMessage(2, namespaceValue, storePb.Value.serializeBinaryToWriter);
          });
        });
      }
      bytes = writer.getResultBuffer();
    } else {
      throw new Error('Unexpected metadata method');
    }
    res.writeHead(200, { 'Content-Type': 'application/grpc-web+proto' });
    res.end(Buffer.concat([grpcFrame(bytes), grpcFrame(Buffer.from('grpc-status: 0\r\n'), true)]));
  }

  beforeEach(async () => {
    vi.resetModules();
    vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', 'mlmd-then-prefix');
    vi.stubEnv('ARTIFACT_NAMESPACE_KEY_PREFIX', 'private-artifacts');
    records = new Map();
    lookupUnavailable = false;
    metadataUris = [];
    contextIds = [];
    authorizationCalls = [];
    storageCalls = [];
    protocolErrors = [];
    metadataOrigin = await listen(
      createServer((req, res) => {
        void handleMetadata(req, res).catch((error: Error) => {
          protocolErrors.push(error.message);
          res.writeHead(500).end();
        });
      }),
    );
    storageOrigin = await listen(
      createServer((req, res) => {
        storageCalls.push({
          path: req.url!,
          authorization: req.headers.authorization,
          user: req.headers['user'],
        });
        if (req.headers.authorization !== storageToken) {
          res.writeHead(403).end();
          return;
        }
        res.setHeader('Content-Type', 'text/html');
        res.end(`artifact:${req.url}`);
      }),
    );
  });

  afterEach(async () => {
    await Promise.all(
      servers.splice(0).map(
        (server) =>
          new Promise<void>((resolve, reject) => {
            server.close((error) => (error ? reject(error) : resolve()));
            server.closeAllConnections();
          }),
      ),
    );
    vi.unstubAllEnvs();
    expect(protocolErrors).toEqual([]);
  });

  async function app(proxy = true) {
    const { loadConfigs } = await import('../configs.js');
    const { getAuthorizeFn } = await import('../helpers/auth.js');
    const { getArtifactsAuthMiddleware, getArtifactsHandler, getArtifactsProxyHandler } =
      await import('../handlers/artifacts.js');
    const options = loadConfigs(['node', 'server', '/tmp', '3000'], {
      ENABLE_AUTHZ: 'true',
      KUBEFLOW_USERID_HEADER: 'user',
      KUBEFLOW_USERID_PREFIX: '',
      FRONTEND_SERVER_NAMESPACE: namespace,
      HTTP_BASE_URL: proxy ? '' : storageOrigin + '/',
      HTTP_AUTHORIZATION_KEY: 'Authorization',
      HTTP_AUTHORIZATION_DEFAULT_VALUE: proxy ? 'wrong-central-token' : storageToken,
    });
    let receiverOrigin = '';
    if (proxy) {
      const receiverOptions = loadConfigs(['node', 'server', '/tmp', '3000'], {
        FRONTEND_SERVER_NAMESPACE: namespace,
        HTTP_BASE_URL: storageOrigin + '/',
        HTTP_AUTHORIZATION_KEY: 'Authorization',
        HTTP_AUTHORIZATION_DEFAULT_VALUE: storageToken,
      });
      const receiver = express();
      receiver.get(
        '/artifacts/get',
        getArtifactsHandler({
          artifactsConfigs: receiverOptions.artifacts,
          options: receiverOptions,
          useParameter: false,
          tryExtract: false,
        }),
      );
      receiverOrigin = await listen(createServer(receiver));
    }
    const shared = express();
    shared.get(
      '/artifacts/get',
      getArtifactsAuthMiddleware(
        getAuthorizeFn(options.auth, { apiServerAddress: metadataOrigin }),
        true,
        'user',
        metadataOrigin,
      ),
      getArtifactsProxyHandler({
        enabled: proxy,
        allowedDomain: '^127\\.0\\.0\\.1$',
        namespacedServiceGetter: () => receiverOrigin,
      }),
      getArtifactsHandler({
        artifactsConfigs: options.artifacts,
        options,
        useParameter: false,
        tryExtract: false,
      }),
    );
    return shared;
  }

  function add(key: string, artifacts: Artifact[] = [{ id: 1, namespace }]) {
    const uri = `${storageOrigin}/${key}`;
    records.set(uri, artifacts);
    return uri;
  }

  async function read(key: string, { proxy = true, identity = user, ns = namespace } = {}) {
    const request = requests(await app(proxy))
      .get('/artifacts/get')
      .query({ source: 'http', bucket: new URL(storageOrigin).host, key, namespace: ns });
    if (identity) request.set('user', identity);
    return request;
  }

  it.each([
    ['legacy/model.txt', false],
    ['legacy/model.txt', true],
    ['projects/team-a-training/model.txt', true],
    ['legacy/model/', true],
  ] as const)('serves the exact tracked URI %s (proxy=%s)', async (key, proxy) => {
    const uri = add(key);
    const response = await read(key, { proxy });
    expect(response.status).toBe(200);
    expect(response.text || Buffer.from(response.body).toString()).toBe(`artifact:/${key}`);
    expect(response.headers['content-disposition']).toMatch(/^attachment/);
    expect(response.headers['x-content-type-options']).toBe('nosniff');
    expect(metadataUris).toEqual([uri]);
    expect(contextIds).toEqual([1]);
    expect(authorizationCalls).toEqual([{ namespace, user }]);
    expect(storageCalls).toEqual([
      { path: `/${key}`, authorization: storageToken, user: undefined },
    ]);
  });

  it('documents deferred exposure when matching metadata references an object readable with shared credentials', async () => {
    // Importers can associate another namespace's URI with their own context.
    // Compatibility deliberately does not claim that this proves storage ownership.
    const key = 'private-artifacts/team-b/model.txt';
    add(key, [{ id: 1, namespace }]);
    const response = await read(key, { proxy: false });
    expect(response.status).toBe(200);
    expect(storageCalls).toEqual([
      { path: `/${key}`, authorization: storageToken, user: undefined },
    ]);
  });

  it.each([false, true])(
    'denies a known foreign context with another failed lookup (%s)',
    async (foreignFirst) => {
      add('legacy/model.txt', [
        { id: 1, namespace: foreignFirst ? 'team-b' : namespace, unavailable: !foreignFirst },
        { id: 2, namespace: foreignFirst ? namespace : 'team-b', unavailable: foreignFirst },
      ]);
      expect((await read('legacy/model.txt')).status).toBe(403);
      expect(contextIds.sort()).toEqual([1, 2]);
      expect(storageCalls).toEqual([]);
    },
  );

  it.each([
    ['private-artifacts/team-a/model.txt', 'mlmd-then-prefix', 200],
    ['private-artifacts/team-b/model.txt', 'mlmd-then-prefix', 403],
    ['legacy/model.txt', 'mlmd-then-prefix', 403],
    ['private-artifacts/team-a/model.txt', 'mlmd-only', 403],
  ] as const)('retains the no-record decision for %s in %s', async (key, mode, status) => {
    vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', mode);
    expect((await read(key)).status).toBe(status);
    expect(contextIds).toEqual([]);
    expect(storageCalls).toHaveLength(status === 200 ? 1 : 0);
  });

  it.each(['lookup', 'context', 'missing-context'] as const)(
    'preserves legacy reads with %s unavailable',
    async (scenario) => {
      add('legacy/model.txt', [
        {
          id: 1,
          namespace: scenario === 'missing-context' ? undefined : namespace,
          unavailable: scenario === 'context',
        },
      ]);
      lookupUnavailable = scenario === 'lookup';
      expect((await read('legacy/model.txt')).status).toBe(200);
      expect(storageCalls).toHaveLength(1);
    },
  );

  it.each([
    ['', namespace, 401],
    ['mallory@example.test', namespace, 403],
    [user, 'team-b', 403],
  ] as const)(
    'denies unauthorized identity/namespace before metadata or storage (%s, %s)',
    async (identity, ns, status) => {
      add('legacy/model.txt');
      expect((await read('legacy/model.txt', { identity, ns })).status).toBe(status);
      expect(metadataUris).toEqual([]);
      expect(storageCalls).toEqual([]);
    },
  );
});
