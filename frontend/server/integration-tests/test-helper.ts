import * as path from 'path';
import * as os from 'os';
import * as fs from 'fs';

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
    jest.resetAllMocks();
    jest.restoreAllMocks();
  });

  if (!options.showLog) {
    beforeEach(() => {
      jest.spyOn(global.console, 'info').mockImplementation();
      jest.spyOn(global.console, 'log').mockImplementation();
    });
  }

  return { argv, buildDate, indexHtmlPath, indexHtmlContent };
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
