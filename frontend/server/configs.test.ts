// Copyright 2019 The Kubeflow Authors
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
import { randomBytes } from 'crypto';
import * as os from 'os';
import { vi } from 'vitest';
import { getConfigsForLogging, loadConfigs } from './configs.js';

vi.mock('crypto', { spy: true });

describe('loadConfigs', () => {
  it('should throw error if no static dir provided', () => {
    const argv = ['node', 'dist/server.js'];
    expect(() => loadConfigs(argv, {})).toThrowError();
  });

  it('default port should be 3000', () => {
    const tmpdir = os.tmpdir();
    const configs = loadConfigs(['node', 'dist/server.js', tmpdir], {});
    expect(configs.server.port).toBe(3000);
    expect(configs.server.staticDir).toBe(tmpdir);
  });

  it('default clusterDomain should be .svc.cluster.local', () => {
    const tmpdir = os.tmpdir();
    const configs = loadConfigs(['node', 'dist/server.js', tmpdir], {});
    expect(configs.viewer.tensorboard.clusterDomain).toBe('.svc.cluster.local');
  });

  it('clusterDomain should use CLUSTER_DOMAIN env var when set', () => {
    const tmpdir = os.tmpdir();
    const configs = loadConfigs(['node', 'dist/server.js', tmpdir], {
      CLUSTER_DOMAIN: 'cluster.corp',
    });
    expect(configs.viewer.tensorboard.clusterDomain).toBe('cluster.corp');
  });

  it('restricts GCS universes by default and parses explicit domains', () => {
    const tmpdir = os.tmpdir();
    expect(
      loadConfigs(['node', 'dist/server.js', tmpdir], {}).artifacts.allowedGcsUniverseDomains,
    ).toEqual(['googleapis.com']);
    expect(
      loadConfigs(['node', 'dist/server.js', tmpdir], {
        ALLOWED_GCS_UNIVERSE_DOMAINS: ' googleapis.com, GDC.EXAMPLE, ',
      }).artifacts.allowedGcsUniverseDomains,
    ).toEqual(['googleapis.com', 'gdc.example']);
  });

  it('generates a process-local tensorboard proxy signing secret when unset', () => {
    const tmpdir = os.tmpdir();
    const firstConfigs = loadConfigs(['node', 'dist/server.js', tmpdir], {
      MINIO_SECRET_KEY: 'shared-minio-secret',
    });
    const secondConfigs = loadConfigs(['node', 'dist/server.js', tmpdir], {
      MINIO_SECRET_KEY: 'another-minio-secret',
    });

    const signingSecret = firstConfigs.viewer.tensorboard.proxySigningSecret;
    expect(signingSecret).not.toBe('shared-minio-secret');
    expect(Buffer.from(signingSecret, 'base64url')).toHaveLength(32);
    expect(secondConfigs.viewer.tensorboard.proxySigningSecret).toBe(signingSecret);
  });

  it('tensorboard proxy signing secret uses TENSORBOARD_PROXY_SIGNING_SECRET when set', () => {
    const tmpdir = os.tmpdir();
    const signingSecret = 'dedicated-tensorboard-proxy-secret';
    const configs = loadConfigs(['node', 'dist/server.js', tmpdir], {
      MINIO_SECRET_KEY: 'shared-minio-secret',
      TENSORBOARD_PROXY_SIGNING_SECRET: signingSecret,
    });
    expect(configs.viewer.tensorboard.proxySigningSecret).toBe(signingSecret);
  });

  it('generates the fallback signing secret only when needed and reuses it', async () => {
    vi.resetModules();
    vi.mocked(randomBytes).mockClear();
    const { loadConfigs: freshLoadConfigs } = await import('./configs.js');
    expect(randomBytes).not.toHaveBeenCalled();

    const argv = ['node', 'dist/server.js', os.tmpdir()];
    const signingSecret = 'dedicated-tensorboard-proxy-secret';
    const configuredConfigs = freshLoadConfigs(argv, {
      TENSORBOARD_PROXY_SIGNING_SECRET: signingSecret,
    });
    expect(configuredConfigs.viewer.tensorboard.proxySigningSecret).toBe(signingSecret);
    expect(randomBytes).not.toHaveBeenCalled();

    const firstConfigs = freshLoadConfigs(argv, {});
    const secondConfigs = freshLoadConfigs(argv, {});
    expect(randomBytes).toHaveBeenCalledExactlyOnceWith(32);
    expect(secondConfigs.viewer.tensorboard.proxySigningSecret).toBe(
      firstConfigs.viewer.tensorboard.proxySigningSecret,
    );
  });

  it('redacts the tensorboard proxy signing secret from logged configs', () => {
    const tmpdir = os.tmpdir();
    const signingSecret = 'dedicated-tensorboard-proxy-secret';
    const configs = loadConfigs(['node', 'dist/server.js', tmpdir], {
      TENSORBOARD_PROXY_SIGNING_SECRET: signingSecret,
    });

    const loggedConfigs = getConfigsForLogging(configs);

    expect(JSON.stringify(loggedConfigs)).not.toContain(signingSecret);
    expect(loggedConfigs.viewer.tensorboard.clusterDomain).toBe('.svc.cluster.local');
    expect(loggedConfigs.viewer.tensorboard.proxySigningSecret).toContain('omitted');
  });

  it('rejects a tensorboard proxy signing secret reused from MINIO_SECRET_KEY', () => {
    const tmpdir = os.tmpdir();
    const sharedSecret = 'shared-secret-that-is-at-least-32-bytes';

    expect(() =>
      loadConfigs(['node', 'dist/server.js', tmpdir], {
        MINIO_SECRET_KEY: sharedSecret,
        TENSORBOARD_PROXY_SIGNING_SECRET: sharedSecret,
      }),
    ).toThrowError('must not reuse MINIO_SECRET_KEY');
  });

  it('rejects a short tensorboard proxy signing secret', () => {
    const tmpdir = os.tmpdir();

    expect(() =>
      loadConfigs(['node', 'dist/server.js', tmpdir], {
        MINIO_SECRET_KEY: 'different-minio-secret',
        TENSORBOARD_PROXY_SIGNING_SECRET: 'too-short',
      }),
    ).toThrowError('must be at least 32 bytes');
  });

  it.each([
    ['1', 1],
    [' 9000 ', 9000],
    ['65535', 65535],
  ])('parses valid AWS_S3_PORT configuration %s', (configuredPort, expectedPort) => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      AWS_S3_PORT: configuredPort,
    });
    expect(configs.artifacts.aws.port).toBe(expectedPort);
  });

  it.each(['', '   '])('treats empty AWS_S3_PORT configuration %s as unset', (port) => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], { AWS_S3_PORT: port });
    expect(configs.artifacts.aws.port).toBeUndefined();
  });

  it.each(['abc', '123abc', '0', '65536', '-1', '1.5'])(
    'rejects invalid AWS_S3_PORT configuration %s',
    (port) => {
      expect(() =>
        loadConfigs(['node', 'dist/server.js', os.tmpdir()], { AWS_S3_PORT: port }),
      ).toThrow('AWS_S3_PORT must be an integer between 1 and 65535');
    },
  );

  it('rejects conflicting embedded and separately configured AWS ports', () => {
    expect(() =>
      loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
        AWS_S3_ENDPOINT: 'objects.example.com:9000',
        AWS_S3_PORT: '9443',
      }),
    ).toThrow('AWS_S3_ENDPOINT port 9000 conflicts with AWS_S3_PORT value 9443');
  });

  it('rejects a separately configured AWS port that conflicts with explicit HTTPS port 443', () => {
    expect(() =>
      loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
        AWS_S3_ENDPOINT: 'https://s3.amazonaws.com:443',
        AWS_S3_PORT: '9443',
      }),
    ).toThrow('AWS_S3_ENDPOINT port 443 conflicts with AWS_S3_PORT value 9443');
  });

  it('normalizes an embedded AWS endpoint port for the storage client', () => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      AWS_S3_ENDPOINT: 'https://objects.example.com:9443',
    });

    expect(configs.artifacts.aws).toEqual(
      expect.objectContaining({
        endPoint: 'objects.example.com',
        port: 9443,
        useSSL: true,
      }),
    );
  });

  it.each([
    ['http://', 'true'],
    ['http://objects.example.com', 'true'],
    ['https://objects.example.com/path', 'true'],
  ])('rejects invalid AWS endpoint configuration %s', (endpoint, ssl) => {
    expect(() =>
      loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
        AWS_S3_ENDPOINT: endpoint,
        AWS_SSL: ssl,
      }),
    ).toThrow('AWS_S3_ENDPOINT must be a valid HTTP(S) origin consistent with AWS_SSL');
  });

  it('only enables official AWS endpoint trust when AWS_S3_ENDPOINT is explicit', () => {
    const defaults = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {});
    const explicit = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      AWS_S3_ENDPOINT: 's3.amazonaws.com',
    });
    expect(defaults.artifacts.allowOfficialAwsEndpoints).toBe(false);
    expect(explicit.artifacts.allowOfficialAwsEndpoints).toBe(true);
  });

  it.each([
    ['1', 1],
    ['65535', 65535],
  ])('parses valid MINIO_PORT configuration %s', (configuredPort, expectedPort) => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      MINIO_PORT: configuredPort,
    });
    expect(configs.artifacts.minio.port).toBe(expectedPort);
  });

  it.each(['abc', '0', '65536'])(
    'rejects invalid MINIO_PORT configuration %s',
    (configuredPort) => {
      expect(() =>
        loadConfigs(['node', 'dist/server.js', os.tmpdir()], { MINIO_PORT: configuredPort }),
      ).toThrow('MINIO_PORT must be an integer between 1 and 65535');
    },
  );

  it('normalizes explicitly allowed artifact origins', () => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      ALLOWED_ARTIFACT_ENDPOINTS: 'https://objects.example.com, http://minio.example.com:9000',
    });
    expect(configs.artifacts.allowedEndpoints).toEqual([
      'https://objects.example.com',
      'http://minio.example.com:9000',
    ]);
  });

  it('trusts the stock Argo service origins without additional allowlist entries', () => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {});

    expect(configs.artifacts.allowedEndpoints).toEqual([]);
    expect(configs.argo.artifactRepositoryEndpoints).toEqual([
      'http://seaweedfs.kubeflow.svc:9000',
      'http://seaweedfs.kubeflow.svc.cluster.local:9000',
    ]);
  });

  it.each(['.svc.cluster.corp', 'svc.cluster.corp'])(
    'derives Argo origins from operator storage settings and domain %s',
    (domain) => {
      const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
        MINIO_HOST: 'archive-store',
        MINIO_NAMESPACE: 'storage',
        MINIO_PORT: '9443',
        MINIO_SSL: 'true',
        CLUSTER_DOMAIN: domain,
        FRONTEND_SERVER_NAMESPACE: 'tenant',
      });

      expect(configs.argo.artifactRepositoryEndpoints).toEqual([
        'https://archive-store.storage.svc:9443',
        'https://archive-store.storage.svc.cluster.corp:9443',
      ]);
    },
  );

  it.each([
    { MINIO_HOST: 'objects.example.com', MINIO_NAMESPACE: '' },
    { MINIO_HOST: 'objects.example.com' },
    { MINIO_HOST: '127.0.0.1' },
    { MINIO_NAMESPACE: '' },
    { MINIO_NAMESPACE: 'invalid.namespace' },
  ])('does not generate service aliases for non-service settings %j', (env) => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], env);

    expect(configs.argo.artifactRepositoryEndpoints).toEqual([]);
  });

  it.each(['', '.', '.svc:9443', '.svc/path', '.svc@other.example.com'])(
    'does not derive an Argo origin from an invalid domain suffix %s',
    (domain) => {
      const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
        CLUSTER_DOMAIN: domain,
      });

      expect(configs.argo.artifactRepositoryEndpoints).toEqual([
        'http://seaweedfs.kubeflow.svc:9000',
      ]);
    },
  );

  it('deduplicates the Argo service origin when CLUSTER_DOMAIN is .svc', () => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      CLUSTER_DOMAIN: '.svc',
    });

    expect(configs.argo.artifactRepositoryEndpoints).toEqual([
      'http://seaweedfs.kubeflow.svc:9000',
    ]);
  });

  it('ignores blank explicitly allowed artifact endpoint entries', () => {
    const configs = loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
      ALLOWED_ARTIFACT_ENDPOINTS: 'https://objects.example.com, , http://minio.example.com:9000,',
    });
    expect(configs.artifacts.allowedEndpoints).toEqual([
      'https://objects.example.com',
      'http://minio.example.com:9000',
    ]);
  });

  it.each(['objects.example.com', 'ftp://objects.example.com', 'https://user@objects.example.com'])(
    'rejects invalid explicitly allowed artifact origin %s',
    (configuredEndpoint) => {
      expect(() =>
        loadConfigs(['node', 'dist/server.js', os.tmpdir()], {
          ALLOWED_ARTIFACT_ENDPOINTS: configuredEndpoint,
        }),
      ).toThrow('ALLOWED_ARTIFACT_ENDPOINTS entry must be an absolute HTTP(S) origin');
    },
  );
});
