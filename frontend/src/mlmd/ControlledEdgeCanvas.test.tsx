/*
 * Copyright 2021 The Kubeflow Authors
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

import * as React from 'react';
import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { CARD_OFFSET, ControlledEdgeCanvas } from './ControlledEdgeCanvas';
import { edgeCanvasCss } from './EdgeCanvas';
import { CARD_ROW_HEIGHT, CARD_TITLE_HEIGHT } from './LineageCss';

const defaultProps = {
  cardWidth: 200,
  edgeWidth: 40,
  reverseEdges: false,
  offset: 0,
  top: 0,
  outputExecutionToOutputArtifactMap: new Map<number, number[]>(),
};

function svgHeights(container: HTMLElement): number[] {
  return Array.from(container.querySelectorAll('svg')).map((svg) =>
    Number(svg.getAttribute('height')),
  );
}

function pathDs(container: HTMLElement): (string | null)[] {
  return Array.from(container.querySelectorAll('path')).map((p) => p.getAttribute('d'));
}

describe('ControlledEdgeCanvas', () => {
  // --- rendering counts ---

  it('renders one EdgeLine per artifactId', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10, 11, 12]}
        artifactToCardMap={
          new Map([
            [10, 0],
            [11, 0],
            [12, 0],
          ])
        }
      />,
    );
    expect(container.querySelectorAll('svg').length).toBe(3);
  });

  it('renders no EdgeLines when artifactIds is empty', () => {
    const { container } = render(
      <ControlledEdgeCanvas {...defaultProps} artifactIds={[]} artifactToCardMap={new Map()} />,
    );
    expect(container.querySelectorAll('svg').length).toBe(0);
  });

  it('applies edgeCanvasReverse class when reverseEdges is true', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10]}
        artifactToCardMap={new Map([[10, 0]])}
        reverseEdges={true}
      />,
    );
    expect((container.firstChild as HTMLElement).classList.contains('edgeCanvasReverse')).toBe(
      true,
    );
  });

  it('does not apply edgeCanvasReverse class when reverseEdges is false', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10]}
        artifactToCardMap={new Map([[10, 0]])}
      />,
    );
    expect((container.firstChild as HTMLElement).classList.contains('edgeCanvasReverse')).toBe(
      false,
    );
  });

  // --- viewHeight / pixel offset math ---

  it('single artifact: first EdgeLine starts at viewHeight=1 (canvas base)', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10]}
        artifactToCardMap={new Map([[10, 0]])}
      />,
    );
    expect(svgHeights(container)[0]).toBe(1);
  });

  it('uses the previous artifact in the sequence when ids are non-consecutive', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10, 42]}
        artifactToCardMap={
          new Map([
            [10, 0],
            [42, 0],
          ])
        }
      />,
    );
    expect(svgHeights(container)).toEqual([1, 1 + CARD_OFFSET]);
  });

  it('non-consecutive artifact ids on different cards use different-card spacing', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10, 42]}
        artifactToCardMap={
          new Map([
            [10, 0],
            [42, 1],
          ])
        }
      />,
    );
    expect(svgHeights(container)).toEqual([1, 1 + CARD_OFFSET + CARD_ROW_HEIGHT]);
  });

  it('non-zero offset shifts initial viewHeight and lastNodePosition', () => {
    // offset=10: viewHeight starts at 1+10=11, y4 starts at 10.
    // Single artifact: EdgeLine height=11.
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10]}
        artifactToCardMap={new Map([[10, 0]])}
        offset={10}
      />,
    );
    expect(svgHeights(container)[0]).toBe(11);
  });

  it('keeps rendered SVG heights positive when the starting offset is negative', () => {
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[2]}
        artifactToCardMap={new Map([[2, 1]])}
        offset={-54}
        reverseEdges={true}
      />,
    );
    expect(svgHeights(container)[0]).toBeGreaterThan(0);
  });

  // --- canvas CSS positioning ---

  it('positions canvas top at (top prop + CARD_TITLE_HEIGHT + CARD_ROW_HEIGHT / 2)', () => {
    // edgeCanvasCss is a pure function: same arguments → same deterministic typestyle class name.
    const topProp = 50;
    const expectedCss = edgeCanvasCss(
      /* left= */ defaultProps.cardWidth,
      /* top= */ topProp + CARD_TITLE_HEIGHT + CARD_ROW_HEIGHT / 2,
      /* width= */ defaultProps.edgeWidth,
    );
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        top={topProp}
        artifactIds={[]}
        artifactToCardMap={new Map()}
      />,
    );
    expect((container.firstChild as HTMLElement).className).toContain(expectedCss.edgeCanvas);
  });

  // --- y-coordinate tracking and reverseEdges ---

  it('reverseEdges=false tracks y4; second artifact EdgeLine curves from flat start to raised end', () => {
    // Two artifacts same card (ids=[10,11], both card 0), edgeWidth=40.
    // EdgeLine 2: height=68, y1=0, y4=67.
    // startY=68-0=68, endY=68-67=1.  path: M0,68 C30,68 10,1 40,1
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10, 11]}
        artifactToCardMap={
          new Map([
            [10, 0],
            [11, 0],
          ])
        }
        reverseEdges={false}
      />,
    );
    expect(pathDs(container)[1]).toBe('M0,68 C30,68 10,1 40,1');
  });

  it('reverseEdges=true tracks y1 instead of y4; second EdgeLine curves in opposite direction', () => {
    // Same setup.  EdgeLine 2: height=68, y1=67, y4=0.
    // startY=68-67=1, endY=68-0=68.  path: M0,1 C30,1 10,68 40,68
    const { container } = render(
      <ControlledEdgeCanvas
        {...defaultProps}
        artifactIds={[10, 11]}
        artifactToCardMap={
          new Map([
            [10, 0],
            [11, 0],
          ])
        }
        reverseEdges={true}
      />,
    );
    expect(pathDs(container)[1]).toBe('M0,1 C30,1 10,68 40,68');
  });
});
