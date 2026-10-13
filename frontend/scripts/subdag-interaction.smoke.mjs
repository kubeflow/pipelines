// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
// Run against Storybook: node --test scripts/subdag-interaction.smoke.mjs

import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';

const baseUrl = process.env.STORYBOOK_URL || 'http://localhost:6006';

test(
  'locked graphs pan with pointer and arrow keys; unlocking enables node movement',
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

      const canvas = page.getByRole('region', { name: 'Pipeline graph', exact: true });
      assert.equal(
        await page
          .getByRole('button', { name: 'Unlock graph', exact: true })
          .getAttribute('aria-pressed'),
        'true',
      );
      await panFromBody(outer);
      await panFromBody(inner);
      assert.equal(
        await canvas.evaluate((element) => document.activeElement === element),
        true,
        'Panning the background focuses keyboard navigation',
      );
      for (const target of [
        outer.getByRole('button', { name: 'Training pipeline', exact: true }),
        inner.getByRole('button', { name: 'Training and evaluation', exact: true }),
        leaf.locator('button').first(),
        artifact.locator('button').first(),
        outside.locator('button').first(),
        outer.getByRole('button', { name: 'Collapse Training pipeline', exact: true }),
      ]) {
        await page.getByRole('button', { name: 'Fit view', exact: true }).click();
        await settle();
        const before = await positions();
        const rect = await target.boundingBox();
        await drag({ x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 }, async () => {
          const after = await positions();
          for (let index = 0; index < before.length; index++) {
            close(after[index].x - before[index].x, 30, 'Locked node drag pans the whole canvas');
            close(after[index].y - before[index].y, 18, 'Locked node drag pans the whole canvas');
          }
        });
      }
      // Even the invisible hit area over an edge must not block canvas panning.
      await page.getByRole('button', { name: 'Fit view', exact: true }).click();
      await settle();
      const edgePoint = await page
        .locator('.react-flow__edge-path')
        .first()
        .evaluate((path) => {
          const point = path.getPointAtLength(path.getTotalLength() / 2);
          const screen = new DOMPoint(point.x, point.y).matrixTransform(path.getScreenCTM());
          return { x: screen.x, y: screen.y };
        });
      const beforeEdgePan = await positions();
      await drag(edgePoint, async () => {
        const after = await positions();
        for (let index = 0; index < after.length; index++) {
          close(after[index].x - beforeEdgePan[index].x, 30, 'Dragging an edge pans');
          close(after[index].y - beforeEdgePan[index].y, 18, 'Dragging an edge pans');
        }
      });
      assert.equal(await page.getByText('Selected:', { exact: false }).count(), 0);
      await canvas.focus();
      for (const [key, dx, dy] of [
        ['ArrowLeft', 40, 0],
        ['ArrowRight', -40, 0],
        ['ArrowUp', 0, 40],
        ['ArrowDown', 0, -40],
      ]) {
        const before = await positions();
        await page.keyboard.press(key);
        await settle();
        const after = await positions();
        for (let index = 0; index < before.length; index++) {
          close(after[index].x - before[index].x, dx, 'Arrow key pans all nodes horizontally');
          close(after[index].y - before[index].y, dy, 'Arrow key pans all nodes vertically');
        }
      }
      await canvas.evaluate((element) => {
        const input = document.createElement('input');
        input.setAttribute('aria-label', 'Graph text input');
        Object.assign(input.style, { position: 'fixed', top: '0', right: '0' });
        element.appendChild(input);
      });
      const input = page.getByRole('textbox', { name: 'Graph text input' });
      await input.fill('abc');
      await settle();
      const beforeTyping = await positions();
      await input.press('ArrowLeft');
      assert.equal(await input.evaluate((element) => element.selectionStart), 2);
      assert.deepEqual(await positions(), beforeTyping, 'Arrow keys in inputs do not pan');
      await input.evaluate((element) => element.remove());

      // Selected groups must still allow body panning, without changing selection.
      await outer.getByRole('button', { name: 'Training pipeline', exact: true }).click();
      await page.getByText('Selected: root / Training pipeline', { exact: true }).waitFor();
      await panFromBody(outer);
      await page.getByText('Selected: root / Training pipeline', { exact: true }).waitFor();
      await panFromBody(inner);
      // Arrow keys from a focused node also pan while locked, even if it is selected.
      await outer.focus();
      const beforeSelectedArrow = await positions();
      await page.keyboard.press('ArrowRight');
      await settle();
      const afterSelectedArrow = await positions();
      for (let index = 0; index < beforeSelectedArrow.length; index++) {
        close(
          afterSelectedArrow[index].x - beforeSelectedArrow[index].x,
          -40,
          'Locked selected node arrows pan',
        );
      }
      await page.getByRole('button', { name: 'Unlock graph', exact: true }).click();
      assert.equal(
        await page
          .getByRole('button', { name: 'Lock graph', exact: true })
          .getAttribute('aria-pressed'),
        'false',
      );
      await leaf.locator('button').first().click();
      await leaf.focus();
      const beforeNodeArrow = await positions();
      await page.keyboard.press('ArrowRight');
      await settle();
      const afterNodeArrow = await positions();
      assert.notEqual(
        afterNodeArrow[2].x,
        beforeNodeArrow[2].x,
        'Unlocked arrow key moves the selected component',
      );
      for (const index of [0, 1, 3, 4]) {
        close(
          afterNodeArrow[index].x,
          beforeNodeArrow[index].x,
          'Unlocked node movement does not pan',
        );
        close(
          afterNodeArrow[index].y,
          beforeNodeArrow[index].y,
          'Unlocked node movement does not pan',
        );
      }

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
      await page.getByRole('button', { name: 'Lock graph', exact: true }).click();
      await canvas.focus();
      const beforeRelock = await positions();
      await page.keyboard.press('ArrowRight');
      await settle();
      const afterRelock = await positions();
      for (let index = 0; index < beforeRelock.length; index++) {
        close(
          afterRelock[index].x - beforeRelock[index].x,
          -40,
          'Relocking restores arrow-key panning',
        );
      }
      await page.reload();
      await page.getByRole('button', { name: 'Unlock graph', exact: true }).waitFor();
      assert.deepEqual(errors, []);
    } finally {
      await browser.close();
    }
  },
);
