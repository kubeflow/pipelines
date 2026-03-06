const path = require('path');

const {
  normalizeGeneratedTypeScriptSource,
  resolvePrettierModule,
} = require('./generate_openapi_typescript_fetch.js');

describe('generate_openapi_typescript_fetch', () => {
  it('resolves the frontend-local prettier module', () => {
    const repoRoot = path.resolve(__dirname, '..', '..');
    const prettier = resolvePrettierModule(repoRoot);

    expect(prettier).toBeTruthy();
    expect(typeof prettier.format).toBe('function');
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

  it('preserves modern TypeScript syntax under the current formatter', () => {
    const original = `
import type { Foo } from './Foo';

export class ResponseError extends Error {
  override name: "ResponseError" = "ResponseError";
}
`;

    const updated = normalizeGeneratedTypeScriptSource(original);

    expect(updated).toContain("import type { Foo } from './Foo';");
    expect(updated).toContain('override name: "ResponseError" = "ResponseError";');
  });
});
