// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
// Run against Storybook: node --test scripts/subdag-interaction.smoke.mjs

import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';

const baseUrl = process.env.STORYBOOK_URL || 'http://localhost:6006';

test(
  'empty group bodies pan the view while headers, tasks, and artifacts still drag',
  {
    timeout: 60000,
  },
  async () => {
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
    try {
      const page = await browser.newPage({ viewport: { width: 1800, height: 1400 } });
      const errors = [];
      page.on('pageerror', (error) => errors.push(error.message));
      await page.goto(baseUrl + '/iframe.html?id=v2-groupeddag--nested-artifacts&viewMode=story');
      await page.getByText('Train model', { exact: true }).waitFor();
      const group = (name) =>
        page.getByRole('button', { name, exact: true }).locator('xpath=ancestor::*[@data-id][1]');
      const outer = group('Training pipeline');
      const inner = group('Training and evaluation');
      const leaf = group('Train model');
      const artifact = page.getByTestId(
        'rf__node-' + JSON.stringify(['root', 'workflow', 'fit', 'artifact.train.model']),
      );
      const outside = group('Deploy model');
      const settle = () =>
        page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
      const positions = async () =>
        Promise.all([outer, inner, leaf, artifact, outside].map((node) => node.boundingBox()));
      const close = (actual, expected, message) =>
        assert.ok(Math.abs(actual - expected) < 1.5, message + ': ' + actual + ' vs ' + expected);
      async function drag(start, verify, delta = { x: 30, y: 18 }) {
        await page.mouse.move(start.x, start.y);
        await page.mouse.down();
        await page.mouse.move(start.x + delta.x, start.y + delta.y, { steps: 10 });
        await settle();
        await verify();
        await page.mouse.up();
        await settle();
        await verify();
      }
      async function panFromBody(node) {
        const before = await positions();
        const bounds = await node.boundingBox();
        const header = await node.getByTestId('subdag-header').boundingBox();
        await drag({ x: bounds.x + 5, y: header.y + header.height + 5 }, async () => {
          const after = await positions();
          for (let index = 0; index < before.length; index++) {
            close(after[index].x - before[index].x, 30, 'Every node pans horizontally');
            close(after[index].y - before[index].y, 18, 'Every node pans vertically');
          }
        });
      }

      await panFromBody(outer);
      await panFromBody(inner);
      assert.equal(await page.getByText('Selected:', { exact: false }).count(), 0);

      // Selected groups must still allow body panning, without changing selection.
      await outer.getByRole('button', { name: 'Training pipeline', exact: true }).click();
      await page.getByText('Selected: root / Training pipeline', { exact: true }).waitFor();
      await panFromBody(outer);
      await page.getByText('Selected: root / Training pipeline', { exact: true }).waitFor();
      await page.getByRole('button', { name: 'Lock graph', exact: true }).click();
      await panFromBody(inner);
      await page.getByRole('button', { name: 'Unlock graph', exact: true }).click();

      const beforeHeaderDrag = await positions();
      const header = await outer
        .getByRole('button', { name: 'Training pipeline', exact: true })
        .boundingBox();
      await drag({ x: header.x + header.width / 2, y: header.y + header.height / 2 }, async () => {
        const after = await positions();
        const dx = after[0].x - beforeHeaderDrag[0].x;
        const dy = after[0].y - beforeHeaderDrag[0].y;
        assert.ok(dx > 0 && dy > 0, 'Header moves the group');
        for (let index = 1; index < 4; index++) {
          close(
            after[index].x - beforeHeaderDrag[index].x,
            dx,
            'Children follow header drag horizontally',
          );
          close(
            after[index].y - beforeHeaderDrag[index].y,
            dy,
            'Children follow header drag vertically',
          );
        }
        close(after[4].x, beforeHeaderDrag[4].x, 'Header drag does not pan outside nodes');
        close(after[4].y, beforeHeaderDrag[4].y, 'Header drag does not pan outside nodes');
      });

      const beforeInnerDrag = await positions();
      const innerHeader = await inner
        .getByRole('button', { name: 'Training and evaluation', exact: true })
        .boundingBox();
      await drag(
        { x: innerHeader.x + innerHeader.width / 2, y: innerHeader.y + innerHeader.height / 2 },
        async () => {
          const after = await positions();
          const dy = after[1].y - beforeInnerDrag[1].y;
          assert.ok(dy < 0, 'Nested header independently moves its group');
          for (const index of [2, 3])
            close(
              after[index].y - beforeInnerDrag[index].y,
              dy,
              'Nested children follow their group',
            );
          for (const index of [0, 4]) {
            close(
              after[index].x,
              beforeInnerDrag[index].x,
              'Nested drag keeps parent/viewport stationary',
            );
            close(
              after[index].y,
              beforeInnerDrag[index].y,
              'Nested drag keeps parent/viewport stationary',
            );
          }
        },
        { x: 0, y: -18 },
      );

      for (const node of [leaf, artifact]) {
        const before = await positions();
        const targetBefore = await node.boundingBox();
        const rect = await node.locator('button').first().boundingBox();
        await drag({ x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 }, async () => {
          const targetAfter = await node.boundingBox();
          assert.ok(
            targetAfter.x !== targetBefore.x || targetAfter.y !== targetBefore.y,
            'Actual child node moves',
          );
          const after = await positions();
          for (const index of [0, 1, 4]) {
            close(after[index].x, before[index].x, 'Child drag leaves groups/viewport stationary');
            close(after[index].y, before[index].y, 'Child drag leaves groups/viewport stationary');
          }
        });
      }
      await page.getByRole('button', { name: 'Collapse Training pipeline', exact: true }).click();
      await page.getByRole('button', { name: 'Expand Training pipeline', exact: true }).waitFor();
      assert.equal(await page.getByText('Train model', { exact: true }).count(), 0);
      const collapsedBefore = await outer.boundingBox();
      const outsideBefore = await outside.boundingBox();
      const collapsedHeader = await outer
        .getByRole('button', { name: 'Training pipeline', exact: true })
        .boundingBox();
      await drag(
        {
          x: collapsedHeader.x + collapsedHeader.width / 2,
          y: collapsedHeader.y + collapsedHeader.height / 2,
        },
        async () => {
          const after = await outer.boundingBox();
          assert.ok(
            after.x > collapsedBefore.x && after.y > collapsedBefore.y,
            'Collapsed groups remain draggable',
          );
          const outsideAfter = await outside.boundingBox();
          close(outsideAfter.x, outsideBefore.x, 'Collapsed group drag does not pan');
          close(outsideAfter.y, outsideBefore.y, 'Collapsed group drag does not pan');
        },
      );
      await page.getByRole('button', { name: 'Expand Training pipeline', exact: true }).click();
      await page.getByText('Train model', { exact: true }).waitFor();
      assert.deepEqual(errors, []);
    } finally {
      await browser.close();
    }
  },
);
