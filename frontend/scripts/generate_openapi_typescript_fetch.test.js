const path = require('path');

const {
  normalizeGeneratedTypeScriptSource,
  normalizePrettier1IncompatibleTypeScript,
  resolvePrettierBin,
} = require('./generate_openapi_typescript_fetch.js');

describe('generate_openapi_typescript_fetch', () => {
  it('resolves the frontend-local prettier binary', () => {
    const repoRoot = path.resolve(__dirname, '..', '..');
    const prettierBin = resolvePrettierBin(repoRoot);

    expect(prettierBin).toMatch(
      /frontend[\\/]node_modules[\\/]prettier[\\/]bin-prettier\.js$/,
    );
  });

  it('renames v1 api symbols without touching unrelated identifiers', () => {
    const original = `
export interface ArchiveRunV1Request {}
const apiVersionV1 = 'v1';
async archiveRunV1Raw() {}
async archiveRunV1() {
  return this.archiveRunV1Raw();
}
throw new Error('Required parameter "id" was null or undefined when calling archiveRunV1().');
`;

    const updated = normalizeGeneratedTypeScriptSource(original, {
      renameV1ApiSymbols: true,
    });

    expect(updated).toContain('export interface ArchiveRunRequest {}');
    expect(updated).toContain('async archiveRunRaw() {}');
    expect(updated).toContain('async archiveRun() {');
    expect(updated).toContain('return this.archiveRunRaw();');
    expect(updated).toContain('calling archiveRun().');
    expect(updated).toContain("const apiVersionV1 = 'v1';");
  });

  it('rewrites server fetch types without touching other output', () => {
    const original = `
credentials?: RequestCredentials;
export type FetchAPI = WindowOrWorkerGlobalScope['fetch'];
private fetchApi = async (url: string, init: RequestInit) => {
`;

    const updated = normalizeGeneratedTypeScriptSource(original, {
      nodeCompatibleFetchTypes: true,
    });

    expect(updated).toContain("credentials?: RequestInit['credentials'];");
    expect(updated).toContain(
      'export type FetchAPI = (input: string, init: RequestInit) => Promise<Response>;',
    );
    expect(updated).toContain(
      'private fetchApi = async (url: string, init: RequestInit): Promise<Response> => {',
    );
  });

  it('strips syntax that the pinned prettier cannot parse', () => {
    const original = `
import type { Foo } from './Foo';

export class ResponseError extends Error {
  override name: "ResponseError" = "ResponseError";
}
`;

    const updated = normalizePrettier1IncompatibleTypeScript(original);

    expect(updated).toContain("import { Foo } from './Foo';");
    expect(updated).toContain('name: "ResponseError" = "ResponseError";');
    expect(updated).not.toContain('import type');
    expect(updated).not.toContain('override name');
  });
});
