// @vitest-environment node
import { afterEach, describe, expect, it } from 'vitest';
import { mkdtemp, mkdir, readFile, realpath, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { build } from 'vite';
import { thirdPartyNoticesPlugin, writeThirdPartyNotices } from './third-party-notices.mjs';

const directories = [];
afterEach(async () => {
  await Promise.all(
    directories.splice(0).map((directory) => rm(directory, { recursive: true, force: true })),
  );
});
async function fixture() {
  const directory = await realpath(await mkdtemp(path.join(tmpdir(), 'kfp-notices-')));
  directories.push(directory);
  await file(
    directory,
    'node_modules/tailwindcss/package.json',
    JSON.stringify({ name: 'tailwindcss', version: '1.0.0' }),
  );
  await file(directory, 'node_modules/tailwindcss/LICENSE', 'Tailwind copyright and MIT license.');
  return directory;
}
async function file(root, name, content) {
  const filename = path.join(root, name);
  await mkdir(path.dirname(filename), { recursive: true });
  await writeFile(filename, content);
}

describe('third-party notices', () => {
  it('uses the Vite bundle inventory, including lazy chunks, supplemental notices and public fonts', async () => {
    const root = await fixture();
    await file(root, 'index.html', '<script type="module" src="/entry.js"></script>');
    await file(root, 'entry.js', 'import "included"; window.load = () => import("lazy");');
    for (const name of ['included', 'lazy', 'unused']) {
      await file(
        root,
        `node_modules/${name}/package.json`,
        JSON.stringify({ name, version: '1.0.0', main: 'index.js', license: 'MIT' }),
      );
      await file(root, `node_modules/${name}/index.js`, `console.log('${name}');`);
      await file(
        root,
        `node_modules/${name}/LICENSE`,
        `Copyright ${name}\nPermission for ${name}.`,
      );
    }
    await file(root, 'node_modules/included/NOTICE', 'Additional attribution from included.');
    await file(root, 'public/fonts/example/OFL.txt', 'Font copyright and full OFL fixture.');
    await build({
      root,
      configFile: false,
      logLevel: 'silent',
      plugins: [thirdPartyNoticesPlugin()],
      build: { license: { fileName: '.vite/third-party-licenses.json' } },
    });
    const notices = await readFile(path.join(root, 'dist/THIRD_PARTY_NOTICES.txt'), 'utf8');
    expect(notices).toContain('included@1.0.0 (MIT)');
    expect(notices).toContain('Copyright included\nPermission for included.');
    expect(notices).toContain('lazy@1.0.0');
    expect(notices).toContain('Additional attribution from included.');
    expect(notices).toContain('Font copyright and full OFL fixture.');
    expect(notices).toContain('Tailwind copyright and MIT license.');
    expect(notices).not.toContain('unused');
  });

  it('retains the pinned upstream Victory license and its bundled vendor licenses', async () => {
    const root = await fixture();
    const vendor = path.join(root, 'node_modules/victory-vendor');
    await file(
      root,
      '.vite/third-party-licenses.json',
      JSON.stringify([{ name: 'victory-vendor', version: '37.3.6', identifier: 'MIT AND ISC' }]),
    );
    await file(
      root,
      'third_party/victory-vendor/37.3.6/LICENSE',
      'Upstream Victory copyright and MIT license.',
    );
    await file(vendor, 'lib-vendor/d3-array/LICENSE', 'D3 copyright and ISC license.');
    await file(root, 'fonts/example/OFL.txt', 'Font copyright and OFL.');
    await writeThirdPartyNotices(root, new Map([['victory-vendor@37.3.6', vendor]]), root);
    const notices = await readFile(path.join(root, 'THIRD_PARTY_NOTICES.txt'), 'utf8');
    expect(notices).toContain('Upstream Victory copyright and MIT license.');
    expect(notices).toContain('D3 copyright and ISC license.');
  });

  it('fails the build with an actionable error if a bundled license is missing', async () => {
    const root = await fixture();
    await file(
      root,
      '.vite/third-party-licenses.json',
      JSON.stringify([{ name: 'missing-license', version: '1.0.0', identifier: 'MIT' }]),
    );
    await expect(writeThirdPartyNotices(root, new Map())).rejects.toThrow(
      'No license text for bundled dependency missing-license@1.0.0. Restore its license file',
    );
  });
});
