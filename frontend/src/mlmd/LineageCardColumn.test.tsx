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
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi, describe, expect, it, beforeEach, Mock } from 'vitest';
import { Artifact, Execution } from '../third_party/mlmd';
import { LineageCardColumn, LineageCardColumnProps, CardDetails } from './LineageCardColumn';
import { LineageRow } from './LineageTypes';
import { CARD_ROW_HEIGHT } from './LineageCss';
import { CARD_OFFSET } from './EdgeCanvas';
import { ControlledEdgeCanvas } from './ControlledEdgeCanvas';

// Hoist mock so LineageCardColumn's import of ControlledEdgeCanvas is replaced.
vi.mock('./ControlledEdgeCanvas', () => ({
  ControlledEdgeCanvas: vi.fn(() => null),
}));

const NEXT_ITEM_SAME_CARD_OFFSET = CARD_ROW_HEIGHT; // 54
const NEXT_ITEM_NEXT_CARD_OFFSET = CARD_ROW_HEIGHT + CARD_OFFSET; // 121

function buildArtifactRow(id: number, options: { next?: boolean } = {}): LineageRow {
  const artifact = new Artifact();
  artifact.setId(id);
  return {
    next: options.next,
    typedResource: { type: 'artifact', resource: artifact },
    resourceDetailsPageRoute: `/artifacts/${id}`,
  };
}

function buildExecutionRow(id: number): LineageRow {
  const execution = new Execution();
  execution.setId(id);
  return {
    typedResource: { type: 'execution', resource: execution },
    resourceDetailsPageRoute: `/executions/${id}`,
  };
}

const defaultProps: LineageCardColumnProps = {
  type: 'artifact',
  title: 'Artifacts',
  cards: [],
  columnWidth: 300,
  columnPadding: 20,
};

function renderColumn(props: Partial<LineageCardColumnProps> = {}) {
  return render(
    <MemoryRouter>
      <LineageCardColumn {...defaultProps} {...props} />
    </MemoryRouter>,
  );
}

describe('LineageCardColumn', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // --- structural rendering ---

  it('renders column header with the given title', () => {
    renderColumn({ title: 'My Artifacts' });
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('My Artifacts');
  });

  it('renders one LineageCard per item in the cards array', () => {
    const cards: CardDetails[] = [
      { title: 'Card Alpha', elements: [buildArtifactRow(1)] },
      { title: 'Card Beta', elements: [buildArtifactRow(2)] },
      { title: 'Card Gamma', elements: [buildArtifactRow(3)] },
    ];
    renderColumn({ cards });
    const cardHeadings = screen.getAllByRole('heading', { level: 3 });
    expect(cardHeadings.length).toBe(3);
    expect(cardHeadings[0]).toHaveTextContent('Card Alpha');
    expect(cardHeadings[1]).toHaveTextContent('Card Beta');
    expect(cardHeadings[2]).toHaveTextContent('Card Gamma');
  });

  it('renders EdgeCanvas edges when outputExecutionToOutputArtifactMap is absent', () => {
    const cards: CardDetails[] = [
      { title: 'card1', elements: [buildArtifactRow(1, { next: true })] },
    ];
    const { container } = renderColumn({ cards });
    // EdgeCanvas is not mocked; it renders a real SVG per connected element.
    expect(container.querySelectorAll('svg').length).toBe(1);
  });

  it('renders ControlledEdgeCanvas (not EdgeCanvas) when outputExecutionToOutputArtifactMap is provided', () => {
    const artifactCards: CardDetails[] = [
      { title: 'artifact card', elements: [buildArtifactRow(1)] },
    ];
    const executionCards: CardDetails[] = [
      { title: 'execution card', elements: [buildExecutionRow(10)] },
    ];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([[10, [1]]]),
    });
    expect((ControlledEdgeCanvas as unknown as Mock).mock.calls.length).toBeGreaterThan(0);
  });

  it('renders no edges when skipEdgeCanvas is true', () => {
    const cards: CardDetails[] = [
      { title: 'card1', elements: [buildArtifactRow(1, { next: true })] },
    ];
    const { container } = renderColumn({ cards, skipEdgeCanvas: true });
    expect(container.querySelectorAll('svg').length).toBe(0);
  });

  // --- ControlledEdgeCanvas offset math ---

  it('single execution with single artifact: ControlledEdgeCanvas receives offset=0', () => {
    // Canvas is pushed before the artifact loop runs, so artifactOffset=0, executionOffset=0 → offset = 0.
    const artifactCards: CardDetails[] = [{ title: 'A', elements: [buildArtifactRow(1)] }];
    const executionCards: CardDetails[] = [{ title: 'E', elements: [buildExecutionRow(10)] }];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([[10, [1]]]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    expect(calls.length).toBe(1);
    expect(calls[0][0].offset).toBe(0);
  });

  it('single execution with two artifacts on the same card: ControlledEdgeCanvas receives offset=0', () => {
    // Canvas is pushed BEFORE artifacts are processed, so its offset is 0.
    // First artifact (new card): artifactOffset += NEXT_CARD_OFFSET (121).
    // Second artifact (same card): artifactOffset += SAME_CARD_OFFSET (54).
    const artifactCards: CardDetails[] = [
      { title: 'A', elements: [buildArtifactRow(1), buildArtifactRow(2)] },
    ];
    const executionCards: CardDetails[] = [{ title: 'E', elements: [buildExecutionRow(10)] }];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([[10, [1, 2]]]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    expect(calls.length).toBe(1);
    expect(calls[0][0].offset).toBe(0);
  });

  it('two executions on the same card each mapping to a different artifact card receive correct offsets', () => {
    // Execution 10 → artifact 1 (artifact card 0); execution 11 → artifact 2 (artifact card 1).
    // Both executions share the same execution card, so executionOffset advances by
    // NEXT_ITEM_SAME_CARD_OFFSET between them.
    //
    // Canvas 0 (exec 10): artifactOffset=0, executionOffset=0 → offset = 0.
    // After exec 10's artifact loop: artifact 1 is on a new card → artifactOffset += NEXT_CARD_OFFSET (121).
    // Canvas 1 (exec 11): artifactOffset=121, executionOffset=SAME_CARD_OFFSET (54)
    //   → offset = 121 - 54 = NEXT_CARD_OFFSET - SAME_CARD_OFFSET (67).
    const artifactCards: CardDetails[] = [
      { title: 'A', elements: [buildArtifactRow(1)] },
      { title: 'B', elements: [buildArtifactRow(2)] },
    ];
    const executionCards: CardDetails[] = [
      { title: 'E', elements: [buildExecutionRow(10), buildExecutionRow(11)] },
    ];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([
        [10, [1]],
        [11, [2]],
      ]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    expect(calls.length).toBe(2);
    expect(calls[0][0].offset).toBe(0);
    expect(calls[1][0].offset).toBe(NEXT_ITEM_NEXT_CARD_OFFSET - NEXT_ITEM_SAME_CARD_OFFSET);
  });

  it('execution canvas top tracks executionOffset correctly across multiple executions', () => {
    // Both executions on the same execution card.
    // exec 10: top=0.  exec 11: top=SAME_CARD_OFFSET (54).
    const artifactCards: CardDetails[] = [
      { title: 'A', elements: [buildArtifactRow(1)] },
      { title: 'B', elements: [buildArtifactRow(2)] },
    ];
    const executionCards: CardDetails[] = [
      { title: 'E', elements: [buildExecutionRow(10), buildExecutionRow(11)] },
    ];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([
        [10, [1]],
        [11, [2]],
      ]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    expect(calls[0][0].top).toBe(0);
    expect(calls[1][0].top).toBe(NEXT_ITEM_SAME_CARD_OFFSET);
  });

  it('calculates edgeWidth as columnPadding * 2 and passes it to ControlledEdgeCanvas', () => {
    const artifactCards: CardDetails[] = [{ title: 'A', elements: [buildArtifactRow(1)] }];
    const executionCards: CardDetails[] = [{ title: 'E', elements: [buildExecutionRow(10)] }];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([[10, [1]]]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    expect(calls[0][0].edgeWidth).toBe(defaultProps.columnPadding * 2); // 40
  });

  it('calculates cardWidth as columnWidth - edgeWidth and passes it to ControlledEdgeCanvas', () => {
    const artifactCards: CardDetails[] = [{ title: 'A', elements: [buildArtifactRow(1)] }];
    const executionCards: CardDetails[] = [{ title: 'E', elements: [buildExecutionRow(10)] }];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([[10, [1]]]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    const expectedEdgeWidth = defaultProps.columnPadding * 2; // 40
    expect(calls[0][0].cardWidth).toBe(defaultProps.columnWidth - expectedEdgeWidth); // 260
  });

  it('executions on different cards advance executionOffset by NEXT_CARD_OFFSET', () => {
    // Each execution on its own execution card.
    // exec 10: top=0.  exec 11: top=NEXT_CARD_OFFSET (121).
    const artifactCards: CardDetails[] = [
      { title: 'A', elements: [buildArtifactRow(1)] },
      { title: 'B', elements: [buildArtifactRow(2)] },
    ];
    const executionCards: CardDetails[] = [
      { title: 'E1', elements: [buildExecutionRow(10)] }, // card 0
      { title: 'E2', elements: [buildExecutionRow(11)] }, // card 1
    ];
    renderColumn({
      cards: artifactCards,
      connectedCards: executionCards,
      outputExecutionToOutputArtifactMap: new Map([
        [10, [1]],
        [11, [2]],
      ]),
    });
    const calls = (ControlledEdgeCanvas as unknown as Mock).mock.calls;
    expect(calls[0][0].top).toBe(0);
    expect(calls[1][0].top).toBe(NEXT_ITEM_NEXT_CARD_OFFSET);
  });
});
