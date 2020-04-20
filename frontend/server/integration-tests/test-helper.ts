import * as path from 'path';
import * as os from 'os';
import * as fs from 'fs';

export function commonSetup() {
  const indexHtmlPath = path.resolve(os.tmpdir(), 'index.html');
  const argv = ['node', 'dist/server.js', os.tmpdir(), '3000'];
  const buildDate = new Date().toISOString();
  const commitHash = 'abcdefg';
  const indexHtmlContent = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT=null
  </script>
  <script id="kubeflow-client-placeholder"></script>
</head>
</html>`;

  beforeAll(() => {
    fs.writeFileSync(path.resolve(__dirname, 'BUILD_DATE'), buildDate);
    fs.writeFileSync(path.resolve(__dirname, 'COMMIT_HASH'), commitHash);
    fs.writeFileSync(indexHtmlPath, indexHtmlContent);
  });

  afterAll(() => {
    fs.unlinkSync(path.resolve(__dirname, 'BUILD_DATE'));
    fs.unlinkSync(path.resolve(__dirname, 'COMMIT_HASH'));
    fs.unlinkSync(indexHtmlPath);
  });

  beforeEach(() => {
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  return { argv };
}

export function buildQuery(queriesMap: { [key: string]: string | undefined }): string {
  const queryContent = Object.entries(queriesMap)
    .filter((entry): entry is [string, string] => entry[1] != null)
    .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
    .join('&');
  if (!queryContent) {
    return '';
  }
  return `?${queryContent}`;
}
