/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import css from './modernization.css?raw';
function luminance(hex: string) {
  const channels = [1, 3, 5]
    .map((index) => parseInt(hex.slice(index, index + 2), 16) / 255)
    .map((value) => (value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4));
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}
function contrast(a: string, b: string) {
  const [low, high] = [luminance(a), luminance(b)].sort((x, y) => x - y);
  return (high + 0.05) / (low + 0.05);
}

describe.each(['.kfp-theme', '.kfp-theme.dark'])('%s semantic text contrast', (selector) => {
  const block = css.slice(css.indexOf(`${selector} {`)).split('}')[0];
  const tokens = Object.fromEntries(
    [...block.matchAll(/--([\w-]+): (#[0-9a-f]{6});/g)].map((match) => [match[1], match[2]]),
  );
  const pairs = [
    ['foreground', 'background'],
    ['foreground-2', 'card'],
    ['muted-foreground', 'card'],
    ['muted-foreground', 'background'],
    ['muted-foreground', 'muted'],
    ['primary-foreground', 'primary'],
    ['status-failed-foreground', 'status-failed'],
    ...['succeeded', 'running', 'failed', 'warning', 'neutral'].map((status) => [
      `status-${status}`,
      `status-${status}-soft`,
    ]),
  ];
  it.each(pairs)('%s on %s meets 4.5:1 for small text', (foreground, background) => {
    expect(tokens[foreground]).toMatch(/^#[0-9a-f]{6}$/);
    expect(tokens[background]).toMatch(/^#[0-9a-f]{6}$/);
    expect(contrast(tokens[foreground], tokens[background])).toBeGreaterThanOrEqual(4.5);
  });
});
