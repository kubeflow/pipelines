/*
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

module.exports = async (page, scenario) => {
  const steps = scenario.steps;
  if (!steps || !steps.length) {
    return;
  }
  for (const s of steps) {
    console.log('performing action: ' + s.action + ' on: ' + s.selector);
    switch (s.action) {
      case 'click':
        await page.click(s.selector);
        break;
      case 'hover':
        await page.hover(s.selector);
        break;
      case 'waitFor':
        await page.waitFor(s.selector);
        break;
      case 'pause':
        await page.waitFor(300);
        break;
      default:
        throw new Error('Unknown action: ' + s.action);
    }
  }
};
