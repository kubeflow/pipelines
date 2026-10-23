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

import * as minio from 'minio';
import { PassThrough, Readable } from 'stream';
import requests from 'supertest';
import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { MAX_ARTIFACT_PREVIEW_BYTES } from '../helpers/artifact-preview.js';
import { getK8sSecret } from '../k8s-helper.js';
import { commonSetup } from '../integration-tests/test-helper.js';

const MinioClient = minio.Client;
vi.mock('minio');
vi.mock('../k8s-helper.js');

const gcsDownload = vi.fn();
const gcsList = vi.fn();
vi.mock('../gcs-helper.js', () => ({
  getGCSClient: () => Promise.resolve({}),
  listGCSObjectNames: (...args: unknown[]) => gcsList(...args),
  downloadGCSObjectStream: (...args: unknown[]) => gcsDownload(...args),
}));

const validateArtifactNamespace = vi.fn();
vi.mock('../helpers/mlmd-validator.js', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../helpers/mlmd-validator.js')>()),
  validateArtifactNamespace: (...args: unknown[]) => validateArtifactNamespace(...args),
}));

const authorizeFetch = vi.fn();
vi.stubGlobal('fetch', authorizeFetch);

const content = '{"version":1,"kind":"model","hyperparameters":{"max_depth":3}}';
const minioEnv = {
  MINIO_ACCESS_KEY: 'minio',
  MINIO_HOST: 'minio-service',
  MINIO_NAMESPACE: 'kubeflow',
  MINIO_PORT: '9000',
  MINIO_SECRET_KEY: 'minio123',
  MINIO_SSL: 'false',
};

describe('artifact preview endpoint', () => {
  const { argv } = commonSetup();
  let app: UIServer | undefined;
  let getObject: ReturnType<typeof vi.fn>;
  let listObjects: ReturnType<typeof vi.fn>;

  function request(auth = false, env: Record<string, string> = {}) {
    const options = loadConfigs(argv, {
      ...minioEnv,
      KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
      KUBEFLOW_USERID_PREFIX: '',
      ...env,
    });
    options.auth.enabled = auth;
    app = new UIServer(options);
    return requests(app.app);
  }

  beforeEach(() => {
    validateArtifactNamespace.mockReset().mockResolvedValue({ valid: true });
    authorizeFetch.mockReset().mockResolvedValue(new Response('{}', { status: 200 }));
    getObject = vi.fn(async () => Readable.from([content]));
    listObjects = vi.fn(() => {
      throw new Error('Must not list a preview directory');
    });
    (MinioClient as any).mockImplementation(function () {
      return { getObject, listObjects, listObjectsV2: listObjects };
    });
    gcsDownload.mockReset().mockImplementation(async () => Readable.from([content]));
    gcsList.mockReset().mockImplementation(async () => {
      throw new Error('Must not enumerate GCS previews');
    });
  });

  afterEach(async () => {
    await app?.close();
    app = undefined;
  });

  afterAll(() => vi.unstubAllGlobals());

  it.each(['minio', 's3'])('reads only _kfp_preview.json beneath a %s parent', async (source) => {
    const response = await request(false, {
      AWS_ACCESS_KEY_ID: 'aws',
      AWS_SECRET_ACCESS_KEY: 'aws-secret',
    })
      .get(`/artifacts/preview?source=${source}&bucket=bucket&key=v2%2Fartifacts%2Fmodel%2F`)
      .expect(200);
    expect(response.text).toBe(content);
    expect(response.headers['content-type']).toContain('application/json');
    expect(response.headers['cache-control']).toBe('no-store');
    expect(response.headers['x-content-type-options']).toBe('nosniff');
    expect(response.headers['content-disposition']).toContain('_kfp_preview.json');
    expect(getObject).toHaveBeenCalledWith('bucket', 'v2/artifacts/model/_kfp_preview.json');
    expect(listObjects).not.toHaveBeenCalled();
  });

  it('reads only _kfp_preview.json beneath a GCS parent', async () => {
    const response = await request()
      .get('/artifacts/preview?source=gcs&bucket=bucket&key=root%2Fdataset')
      .expect(200);
    expect(response.text).toBe(content);
    expect(gcsDownload).toHaveBeenCalledWith(
      expect.objectContaining({ bucket: 'bucket', objectName: 'root/dataset/_kfp_preview.json' }),
    );
    expect(gcsList).not.toHaveBeenCalled();
  });

  it.each([
    ['minio', 'v2/artifacts/model/'],
    ['s3', 'v2/artifacts/model'],
  ])('reads only the sibling preview beside a %s parent', async (source, key) => {
    const response = await request(false, {
      AWS_ACCESS_KEY_ID: 'aws',
      AWS_SECRET_ACCESS_KEY: 'aws-secret',
    })
      .get(
        `/artifacts/preview?source=${source}&bucket=bucket&key=${encodeURIComponent(key)}&storage=sibling`,
      )
      .expect(200);
    expect(response.text).toBe(content);
    expect(response.headers['content-disposition']).toContain('kfp-preview.json');
    expect(getObject).toHaveBeenCalledWith('bucket', 'v2/artifacts/model.kfp-preview.json');
    expect(listObjects).not.toHaveBeenCalled();
  });

  it('reads only the sibling preview beside a GCS parent', async () => {
    await request()
      .get('/artifacts/preview?source=gcs&bucket=bucket&key=root%2Fdataset&storage=sibling')
      .expect(200, content);
    expect(gcsDownload).toHaveBeenCalledWith(
      expect.objectContaining({ bucket: 'bucket', objectName: 'root/dataset.kfp-preview.json' }),
    );
  });

  it('rejects an unknown preview storage before reading storage', async () => {
    const response = await request()
      .get('/artifacts/preview?source=minio&bucket=bucket&key=model&storage=inline')
      .expect(400);
    expect(response.text).toContain('must be "directory" or "sibling"');
    expect(getObject).not.toHaveBeenCalled();
  });

  it('serves previews from the base-path route', async () => {
    const options = loadConfigs(argv, minioEnv);
    options.server.basePath = '/custom';
    app = new UIServer(options);
    await requests(app.app)
      .get('/custom/artifacts/preview?source=minio&bucket=bucket&key=model')
      .expect(200, content);
  });

  it('bounds GCS credential lookup by the preview deadline', async () => {
    vi.mocked(getK8sSecret).mockReturnValue(new Promise<string>(() => undefined));
    const timeout = new AbortController();
    const timeoutSpy = vi.spyOn(AbortSignal, 'timeout').mockReturnValue(timeout.signal);
    const providerInfo = encodeURIComponent(
      JSON.stringify({
        Provider: 'gs',
        Params: { fromEnv: 'false', secretName: 'gcs-secret', tokenKey: 'token' },
      }),
    );
    const pending = request()
      .get(`/artifacts/preview?source=gcs&bucket=bucket&key=model&providerInfo=${providerInfo}`)
      .then((response) => response);
    await vi.waitFor(() => expect(getK8sSecret).toHaveBeenCalled());
    timeout.abort(new DOMException('The operation timed out.', 'TimeoutError'));
    const response = await pending;
    timeoutSpy.mockRestore();
    expect(response.status).toBe(504);
    expect(gcsDownload).not.toHaveBeenCalled();
  });

  it('authorizes the parent artifact identity in multi-user mode', async () => {
    await request(true)
      .get('/artifacts/preview?source=minio&bucket=bucket&key=v2%2Fartifacts%2Fmodel')
      .set('kubeflow-userid', 'reader@example.com')
      .expect(400);
    await request(true)
      .get(
        '/artifacts/preview?source=minio&bucket=bucket&key=v2%2Fartifacts%2Fmodel&namespace=team-a',
      )
      .expect(401);

    await request(true)
      .get(
        '/artifacts/preview?source=minio&bucket=bucket&key=v2%2Fartifacts%2Fmodel&namespace=team-a',
      )
      .set('kubeflow-userid', 'reader@example.com')
      .expect(200, content);
    expect(validateArtifactNamespace).toHaveBeenCalledWith(
      expect.any(String),
      expect.stringMatching(/bucket\/v2\/artifacts\/model$/),
      'team-a',
    );

    getObject.mockClear();
    validateArtifactNamespace.mockResolvedValue({ valid: false, reason: 'another namespace' });
    await request(true)
      .get(
        '/artifacts/preview?source=minio&bucket=bucket&key=v2%2Fartifacts%2Fmodel&namespace=team-a',
      )
      .set('kubeflow-userid', 'reader@example.com')
      .expect(403);
    expect(getObject).not.toHaveBeenCalled();
  });

  it.each([
    // An empty key fails the shared request validation before the preview branch.
    ['', 500],
    ['parent%2F..%2Fother', 422],
    ['parent%2F*', 422],
    ['parent%2F%2Fchild', 422],
  ])('rejects the ambiguous parent key "%s"', async (key, status) => {
    await request().get(`/artifacts/preview?source=minio&bucket=bucket&key=${key}`).expect(status);
    expect(getObject).not.toHaveBeenCalled();
  });

  it.each([
    ['http', 'example.com'],
    ['volume', 'workspace'],
  ])('never reads a preview through the %s source', async (source, bucket) => {
    const response = await request().get(
      `/artifacts/preview?source=${source}&bucket=${bucket}&key=model`,
    );
    expect(response.status).toBeGreaterThanOrEqual(400);
    expect(response.status).toBeLessThan(500);
    expect(getObject).not.toHaveBeenCalled();
    expect(authorizeFetch).not.toHaveBeenCalled();
  });

  it('reports missing previews without treating them as empty data', async () => {
    getObject.mockRejectedValue(Object.assign(new Error('missing'), { code: 'NoSuchKey' }));
    const response = await request()
      .get('/artifacts/preview?source=minio&bucket=bucket&key=model')
      .expect(404);
    expect(response.text).toContain('The preview file is missing');

    gcsDownload.mockRejectedValue(Object.assign(new Error('missing'), { status: 404 }));
    await request().get('/artifacts/preview?source=gcs&bucket=bucket&key=model').expect(404);
  });

  it('rejects an oversized preview before sending success headers', async () => {
    const oversized = new PassThrough();
    oversized.end(Buffer.alloc(MAX_ARTIFACT_PREVIEW_BYTES + 1, 'a'));
    getObject.mockResolvedValue(oversized);
    const response = await request()
      .get('/artifacts/preview?source=minio&bucket=bucket&key=model')
      .expect(413);
    expect(response.headers['cache-control']).toBe('no-store');
  });

  it('hides unexpected storage failures behind a generic gateway error', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    getObject.mockRejectedValue(new Error('secret endpoint detail'));
    const response = await request()
      .get('/artifacts/preview?source=minio&bucket=bucket&key=model')
      .expect(502);
    expect(response.text).not.toContain('secret endpoint detail');
  });
});
