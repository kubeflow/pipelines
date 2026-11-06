import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);

test('frontend libraries declare Array.at without incidental Node ambient types', () => {
  const directory = mkdtempSync(path.join(tmpdir(), 'kfp-typescript-libs-'));
  try {
    writeFileSync(
      path.join(directory, 'arrays.ts'),
      `const pages: readonly { next_page_token?: string }[] = [];
const token: string | undefined = pages.at(-1)?.next_page_token;
const states: { state: string }[] = [];
const state: string | undefined = states.at(-1)?.state;
export { token, state };
`,
    );
    writeFileSync(
      path.join(directory, 'tsconfig.json'),
      JSON.stringify({
        extends: fileURLToPath(new URL('../tsconfig.json', import.meta.url)),
        compilerOptions: { types: [], noEmit: true },
        files: ['arrays.ts'],
        include: [],
        exclude: [],
      }),
    );
    const output = execFileSync(
      process.execPath,
      [require.resolve('typescript/bin/tsc'), '--project', path.join(directory, 'tsconfig.json')],
      { encoding: 'utf8', timeout: 30000 },
    );
    assert.equal(output, '');
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});
