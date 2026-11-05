// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
const { chromium } = require('playwright');
const fs = require('node:fs');
(async () => {
  const browser = await chromium.launch({headless: true});
  const headers = process.env.TENSORBOARD_USER ? {'kubeflow-userid': process.env.TENSORBOARD_USER} : {};
  const page = await browser.newPage({extraHTTPHeaders: headers});
  const url = process.env.TENSORBOARD_URL || 'http://127.0.0.1:6006/tensorboard/viewer-acceptance/';
  const artifactPrefix = process.env.TENSORBOARD_ARTIFACT_PREFIX || 'tensorboard-browser';
  const failures = [];
  const successfulData = new Set();
  const consoleErrors = [];
  page.on('pageerror', error => consoleErrors.push(error.message));
  const prefix = new URL(url).pathname;
  page.on('response', r => {
    if (r.url().includes('/data/') && !r.ok()) failures.push(`${r.status()} ${r.url()}`);
    if (r.url().includes('/data/plugin/') && r.ok()) successfulData.add(r.url());
  });
  page.on('requestfailed', r => {
    if (r.url().includes('/data/')) failures.push(`${r.failure()?.errorText} ${r.url()}`);
  });
  try {
    await page.goto(url, {waitUntil:'networkidle'});
    // TensorBoard exposes stable plugin IDs on its navigation labels. Their
    // accessible names may include additional Material tab content.
    for (const plugin of ['scalars', 'images', 'graphs']) {
      await page.locator(`.plugin-name[data-plugin-id="${plugin}"]`).click();
      await page.waitForLoadState('networkidle');
      if (!new URL(page.url()).pathname.startsWith(prefix)) throw Error(`Escaped prefix: ${page.url()}`);
      if (await page.getByText('Failed to fetch runs', {exact:false}).count()) throw Error('Failed to fetch runs');
      const element = {scalars:'tf-scalar-dashboard', images:'tf-image-dashboard', graphs:'tf-graph-dashboard'}[plugin];
      await page.locator(element).waitFor({state:'visible'});
      const deadline = Date.now() + 30000;
      while (![...successfulData].some(url => url.includes(`/data/plugin/${plugin}/`))) {
        if (Date.now() >= deadline) throw Error(`No successful ${plugin} data request`);
        await page.waitForTimeout(100);
      }
    }
    if (failures.length) throw Error(JSON.stringify(failures));
    console.log('Browser scalar/image/graph navigation retained path prefix; all data requests passed');
  } catch (error) {
    fs.writeFileSync(`${artifactPrefix}.html`, await page.content());
    fs.writeFileSync(`${artifactPrefix}.json`, JSON.stringify({url:page.url(), failures, consoleErrors, successfulData:[...successfulData]}, null, 2));
    await page.screenshot({path:`${artifactPrefix}.png`, fullPage:true});
    console.error(await page.locator('body').innerText());
    throw error;
  } finally { await browser.close(); }
})().catch(e => { console.error(e); process.exit(1); });
