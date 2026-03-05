const path = require('path');

const {
  generateTargetsParallel,
  normalizeGeneratedTypeScriptSource,
  resolvePrettierModule,
  runPool,
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

  describe('runPool', () => {
    it('limits concurrency to the specified bound', async () => {
      let running = 0;
      let peak = 0;
      const tasks = Array.from({ length: 6 }, () => async () => {
        running++;
        peak = Math.max(peak, running);
        await new Promise((r) => setTimeout(r, 10));
        running--;
      });

      await runPool(tasks, 2);

      expect(peak).toBeLessThanOrEqual(2);
    });

    it('runs all tasks to completion', async () => {
      const results = [];
      const tasks = Array.from({ length: 5 }, (_, i) => async () => {
        results.push(i);
      });

      await runPool(tasks, 3);

      expect(results).toHaveLength(5);
      expect(results.sort()).toEqual([0, 1, 2, 3, 4]);
    });

    it('propagates the first task failure', async () => {
      const tasks = [
        async () => {},
        async () => {
          throw new Error('task 1 failed');
        },
        async () => {},
      ];

      await expect(runPool(tasks, 2)).rejects.toThrow('task 1 failed');
    });

    it('stops dequeuing new tasks after a failure', async () => {
      const started = [];
      const tasks = Array.from({ length: 6 }, (_, i) => async () => {
        started.push(i);
        if (i === 1) throw new Error('fail');
        await new Promise((r) => setTimeout(r, 50));
      });

      await expect(runPool(tasks, 2)).rejects.toThrow('fail');

      expect(started).toContain(0);
      expect(started).toContain(1);
      for (let i = 2; i < 6; i++) {
        expect(started).not.toContain(i);
      }
    });

    it('waits for in-flight tasks to complete before rejecting', async () => {
      const completed = [];
      const tasks = [
        async () => {
          await new Promise((r) => setTimeout(r, 50));
          completed.push(0);
        },
        async () => {
          throw new Error('fail');
        },
      ];

      await expect(runPool(tasks, 2)).rejects.toThrow('fail');

      expect(completed).toContain(0);
    });

    it('handles a single task', async () => {
      let called = false;
      await runPool(
        [
          async () => {
            called = true;
          },
        ],
        4,
      );
      expect(called).toBe(true);
    });

    it('handles empty task list', async () => {
      await expect(runPool([], 4)).resolves.toBeUndefined();
    });
  });

  describe('generateTargetsParallel', () => {
    let logSpy;
    let errorSpy;
    let stderrSpy;

    beforeEach(() => {
      logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
      errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      stderrSpy = vi.spyOn(process.stderr, 'write').mockImplementation(() => true);
    });

    afterEach(() => {
      vi.restoreAllMocks();
    });

    it('does not clean later targets when an early target fails', async () => {
      const prepared = [];

      await expect(
        generateTargetsParallel('/fake', ['v1:experiment', 'v1:job', 'v1:pipeline'], 1, {
          prepare: (_rr, tk) => prepared.push(tk),
          dockerGenerate: async (_rr, tk) => {
            if (tk === 'v1:experiment') throw new Error('gen fail');
          },
          postProcess: () => {},
          pullImage: () => {},
        }),
      ).rejects.toThrow('gen fail');

      expect(prepared).toEqual(['v1:experiment']);
    });

    it('post-processes successful targets even when a later target fails', async () => {
      const postProcessed = [];

      await expect(
        generateTargetsParallel('/fake', ['v1:experiment', 'v1:job', 'v1:pipeline'], 2, {
          prepare: () => {},
          dockerGenerate: async (_rr, tk) => {
            if (tk === 'v1:job') throw new Error('gen fail');
            await new Promise((r) => setTimeout(r, 10));
          },
          postProcess: (_rr, tk) => postProcessed.push(tk),
          pullImage: () => {},
        }),
      ).rejects.toThrow('gen fail');

      expect(postProcessed).toContain('v1:experiment');
      expect(postProcessed).not.toContain('v1:job');
      expect(postProcessed).not.toContain('v1:pipeline');
    });

    it('runs the full prepare-generate-postprocess lifecycle per target', async () => {
      const events = [];

      await generateTargetsParallel('/fake', ['v1:experiment', 'v1:job'], 1, {
        prepare: (_rr, tk) => events.push(`prepare:${tk}`),
        dockerGenerate: async (_rr, tk) => events.push(`generate:${tk}`),
        postProcess: (_rr, tk) => events.push(`postprocess:${tk}`),
        pullImage: () => events.push('pull'),
      });

      expect(events).toEqual([
        'pull',
        'prepare:v1:experiment',
        'generate:v1:experiment',
        'postprocess:v1:experiment',
        'prepare:v1:job',
        'generate:v1:job',
        'postprocess:v1:job',
      ]);
    });
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
