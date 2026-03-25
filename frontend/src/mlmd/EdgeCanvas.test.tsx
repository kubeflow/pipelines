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
import { Artifact } from '../third_party/mlmd';
import { CARD_OFFSET, EdgeCanvas, EdgeCanvasProps, edgeCanvasCss } from './EdgeCanvas';
import { CardDetails } from './LineageCardColumn';
import { LineageRow } from './LineageTypes';
import { CARD_ROW_HEIGHT, CARD_TITLE_HEIGHT, CARD_ROW_CENTER_Y } from './LineageCss';

function buildArtifactRow(id: number, options: { next?: boolean } = {}): LineageRow {
  const artifact = new Artifact();
  artifact.setId(id);
  return {
    next: options.next,
    typedResource: { type: 'artifact', resource: artifact },
    resourceDetailsPageRoute: `/artifacts/${id}`,
  };
}

const defaultProps: Omit<EdgeCanvasProps, 'cards'> = {
  cardWidth: 200,
  edgeWidth: 40,
  reverseEdges: false,
};

function svgHeights(container: HTMLElement): number[] {
  return Array.from(container.querySelectorAll('svg')).map((svg) =>
    Number(svg.getAttribute('height')),
  );
}

function pathDs(container: HTMLElement): (string | null)[] {
  return Array.from(container.querySelectorAll('path')).map((p) => p.getAttribute('d'));
}

describe('EdgeCanvas', () => {
  // --- rendering counts ---

  it('renders no EdgeLines when no elements have a next connection', () => {
    const cards: CardDetails[] = [
      { title: 'card1', elements: [buildArtifactRow(1), buildArtifactRow(2)] },
    ];
    const { container } = render(<EdgeCanvas {...defaultProps} cards={cards} />);
    expect(container.querySelectorAll('svg').length).toBe(0);
  });

  it('renders one EdgeLine per connected element', () => {
    const cards: CardDetails[] = [
      {
        title: 'card1',
        elements: [
          buildArtifactRow(1, { next: true }),
          buildArtifactRow(2),
          buildArtifactRow(3, { next: true }),
        ],
      },
    ];
    const { container } = render(<EdgeCanvas {...defaultProps} cards={cards} />);
    expect(container.querySelectorAll('svg').length).toBe(2);
  });

  it('applies edgeCanvasReverse class when reverseEdges is true', () => {
    const cards: CardDetails[] = [{ title: 'card1', elements: [buildArtifactRow(1)] }];
    const { container } = render(
      <EdgeCanvas {...defaultProps} cards={cards} reverseEdges={true} />,
    );
    expect((container.firstChild as HTMLElement).classList.contains('edgeCanvasReverse')).toBe(
      true,
    );
  });

  it('does not apply edgeCanvasReverse class when reverseEdges is false', () => {
    const cards: CardDetails[] = [{ title: 'card1', elements: [buildArtifactRow(1)] }];
    const { container } = render(
      <EdgeCanvas {...defaultProps} cards={cards} reverseEdges={false} />,
    );
    expect((container.firstChild as HTMLElement).classList.contains('edgeCanvasReverse')).toBe(
      false,
    );
  });

  // --- viewHeight / pixel offset math ---

  it('first EdgeLine always starts at viewHeight=1 (canvas base)', () => {
    const cards: CardDetails[] = [{ title: 'c1', elements: [buildArtifactRow(1, { next: true })] }];
    const { container } = render(<EdgeCanvas {...defaultProps} cards={cards} />);
    expect(svgHeights(container)[0]).toBe(1);
  });

  it('second element on the same card advances viewHeight by CARD_ROW_HEIGHT', () => {
    // EdgeLine 1: height=1.  After push: viewHeight += CARD_ROW_HEIGHT → 55.
    // EdgeLine 2: height=55.
    const cards: CardDetails[] = [
      {
        title: 'c1',
        elements: [buildArtifactRow(1, { next: true }), buildArtifactRow(2, { next: true })],
      },
    ];
    const { container } = render(<EdgeCanvas {...defaultProps} cards={cards} />);
    expect(svgHeights(container)).toEqual([1, 1 + CARD_ROW_HEIGHT]);
  });

  it('first element on a second card advances viewHeight by CARD_ROW_HEIGHT + CARD_OFFSET', () => {
    // Card 1: EdgeLine at height=1.  After card: viewHeight = 1 + CARD_ROW_HEIGHT + CARD_OFFSET.
    // Card 2: EdgeLine at that height.
    const cards: CardDetails[] = [
      { title: 'c1', elements: [buildArtifactRow(1, { next: true })] },
      { title: 'c2', elements: [buildArtifactRow(2, { next: true })] },
    ];
    const { container } = render(<EdgeCanvas {...defaultProps} cards={cards} />);
    expect(svgHeights(container)).toEqual([1, 1 + CARD_ROW_HEIGHT + CARD_OFFSET]);
  });

  it('CARD_OFFSET is applied after each card even when it has no next elements', () => {
    // Card 1 has no next elements → no EdgeLine, but CARD_OFFSET is still consumed.
    // Card 2 first element: viewHeight = 1 + CARD_OFFSET (not 1).
    const cards: CardDetails[] = [
      { title: 'c1', elements: [buildArtifactRow(1)] },
      { title: 'c2', elements: [buildArtifactRow(2, { next: true })] },
    ];
    const { container } = render(<EdgeCanvas {...defaultProps} cards={cards} />);
    expect(svgHeights(container)).toEqual([1 + CARD_OFFSET]);
  });

  // --- canvas CSS positioning ---

  it('positions canvas container at left=cardWidth and top=CARD_TITLE_HEIGHT+CARD_ROW_CENTER_Y', () => {
    // edgeCanvasCss is a pure function: same arguments → same deterministic typestyle class name.
    // We call it with the values the component should use and verify the container carries that class.
    const expectedCss = edgeCanvasCss(
      /* left= */ defaultProps.cardWidth,
      /* top= */ CARD_TITLE_HEIGHT + CARD_ROW_CENTER_Y,
      /* width= */ defaultProps.edgeWidth,
    );
    const { container } = render(<EdgeCanvas {...defaultProps} cards={[]} />);
    expect((container.firstChild as HTMLElement).className).toContain(expectedCss.edgeCanvas);
  });

  // --- y-coordinate tracking and reverseEdges ---

  it('reverseEdges=false tracks y4; second card EdgeLine curves from flat start to raised end', () => {
    // Two cards each with one next element, edgeWidth=40.
    // EdgeLine 2: height=1+CARD_ROW_HEIGHT+CARD_OFFSET=122, y1=0, y4=CARD_ROW_HEIGHT+CARD_OFFSET=121.
    // startY = 122-0=122, endY = 122-121=1.
    // path: M0,122 C30,122 10,1 40,1
    const cards: CardDetails[] = [
      { title: 'c1', elements: [buildArtifactRow(1, { next: true })] },
      { title: 'c2', elements: [buildArtifactRow(2, { next: true })] },
    ];
    const { container } = render(
      <EdgeCanvas {...defaultProps} cards={cards} reverseEdges={false} />,
    );
    const paths = pathDs(container);
    expect(paths[1]).toBe('M0,122 C30,122 10,1 40,1');
  });

  it('reverseEdges=true tracks y1 instead of y4; second EdgeLine curves in opposite direction', () => {
    // Same setup, but lastNode=y1.
    // EdgeLine 2: height=122, y1=121, y4=0.
    // startY = 122-121=1, endY = 122-0=122.
    // path: M0,1 C30,1 10,122 40,122
    const cards: CardDetails[] = [
      { title: 'c1', elements: [buildArtifactRow(1, { next: true })] },
      { title: 'c2', elements: [buildArtifactRow(2, { next: true })] },
    ];
    const { container } = render(
      <EdgeCanvas {...defaultProps} cards={cards} reverseEdges={true} />,
    );
    const paths = pathDs(container);
    expect(paths[1]).toBe('M0,1 C30,1 10,122 40,122');
  });
});
