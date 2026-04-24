import { vi, afterAll, beforeAll, beforeEach } from 'vitest';
import * as path from 'path';
import { fileURLToPath } from 'url';
import * as os from 'os';
import * as fs from 'fs';
import express from 'express';
import requests from 'supertest';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export function commonSetup(
  options: { commitHash?: string; tagName?: string; showLog?: boolean } = {},
): { argv: string[]; buildDate: string; indexHtmlPath: string; indexHtmlContent: string } {
  const indexHtmlPath = path.resolve(os.tmpdir(), 'index.html');
  const argv = ['node', 'dist/server.js', os.tmpdir(), '3000'];
  const buildDate = new Date().toISOString();
  const commitHash = options.commitHash || 'abcdefg';
  const tagName = options.tagName || '1.0.0';
  const indexHtmlContent = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT=null
  window.KFP_FLAGS.HIDE_SIDENAV=null
  </script>
  <script id="kubeflow-client-placeholder"></script>
</head>
</html>`;

  beforeAll(() => {
    console.log('beforeAll, writing files');
    fs.writeFileSync(path.resolve(__dirname, '..', 'BUILD_DATE'), buildDate);
    fs.writeFileSync(path.resolve(__dirname, '..', 'COMMIT_HASH'), commitHash);
    fs.writeFileSync(path.resolve(__dirname, '..', 'TAG_NAME'), tagName);
    fs.writeFileSync(indexHtmlPath, indexHtmlContent);
  });

  beforeEach(() => {
    vi.resetAllMocks();
    vi.restoreAllMocks();
  });

  if (!options.showLog) {
    beforeEach(() => {
      vi.spyOn(global.console, 'info').mockImplementation(() => {});
      vi.spyOn(global.console, 'log').mockImplementation(() => {});
    });
  }

  return { argv, buildDate, indexHtmlPath, indexHtmlContent };
}

export function buildQuery(queriesMap: { [key: string]: string | number | undefined }): string {
  const queryContent = Object.entries(queriesMap)
    .filter(
      (entry): entry is [string, string | number] => entry[1] !== undefined && entry[1] !== null,
    )
    .map(([key, value]) => `${key}=${encodeURIComponent(String(value))}`)
    .join('&');
  if (!queryContent) {
    return '';
  }
  return `?${queryContent}`;
}

export function mkTempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'kfp-test-'));
}

export function setupMockBackendTest(options: { showLog?: boolean } = {}): void {
  const frontendRoot = path.resolve(__dirname, '..', '..');
  const originalCwd = process.cwd();

  beforeAll(() => {
    process.chdir(frontendRoot);
  });

  afterAll(() => {
    process.chdir(originalCwd);
  });

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  if (!options.showLog) {
    beforeEach(() => {
      vi.spyOn(global.console, 'info').mockImplementation(() => {});
      vi.spyOn(global.console, 'log').mockImplementation(() => {});
    });
  }
}

export async function createMockBackendRequest(): Promise<ReturnType<typeof requests>> {
  vi.resetModules();
  const { default: mockApiMiddleware } = await import('../../mock-backend/mock-api-middleware.ts');
  const app = express();
  mockApiMiddleware(app as any);
  return requests(app);
}

export function asText(test: requests.Test): requests.Test {
  return test.buffer(true).parse((response, callback) => {
    response.setEncoding('utf8');
    let text = '';
    response.on('data', (chunk) => {
      text += chunk;
    });
    response.on('end', () => {
      callback(null, text);
    });
  });
}
