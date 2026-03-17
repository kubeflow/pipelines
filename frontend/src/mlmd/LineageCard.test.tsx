import * as React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { LineageCard } from './LineageCard';
import { LineageRow } from './LineageTypes';
import { Artifact } from 'src/third_party/mlmd';
import { ArtifactProperties } from './Api';
import { stringValue } from './TestUtils';

function buildArtifactRow(name: string): LineageRow {
  const artifact = new Artifact();
  artifact.setId(1);
  artifact.getPropertiesMap().set(ArtifactProperties.NAME, stringValue(name));
  return {
    typedResource: { type: 'artifact', resource: artifact },
    resourceDetailsPageRoute: `/artifacts/${artifact.getId()}`,
    prev: false,
    next: false,
  };
}

function renderLineageCard(props: React.ComponentProps<typeof LineageCard>) {
  return render(
    <MemoryRouter>
      <LineageCard {...props} />
    </MemoryRouter>,
  );
}

function getContainerClasses(container: HTMLElement): string[] {
  return (container.firstElementChild as HTMLElement).className.split(' ').filter(Boolean);
}

describe('LineageCard', () => {
  it('renders card title in h3', () => {
    renderLineageCard({
      title: 'My Artifact Type',
      type: 'artifact',
      rows: [],
      addSpacer: false,
    });

    const heading = screen.getByRole('heading', { level: 3 });
    expect(heading.textContent).toBe('My Artifact Type');
  });

  it('applies different CSS classes for execution vs artifact type', () => {
    const { container: execContainer } = renderLineageCard({
      title: 'Executions',
      type: 'execution',
      rows: [],
      addSpacer: false,
    });

    const { container: artifactContainer } = renderLineageCard({
      title: 'Artifacts',
      type: 'artifact',
      rows: [],
      addSpacer: false,
    });

    expect(getContainerClasses(execContainer)).not.toEqual(getContainerClasses(artifactContainer));
  });

  it('adds a target CSS class when isTarget is true', () => {
    const { container: targetContainer } = renderLineageCard({
      title: 'Target',
      type: 'artifact',
      rows: [],
      addSpacer: false,
      isTarget: true,
    });

    const { container: nonTargetContainer } = renderLineageCard({
      title: 'Regular',
      type: 'artifact',
      rows: [],
      addSpacer: false,
      isTarget: false,
    });

    expect(getContainerClasses(targetContainer).length).toBeGreaterThan(
      getContainerClasses(nonTargetContainer).length,
    );
  });

  it('adds a spacer CSS class when addSpacer is true', () => {
    const { container: spacerContainer } = renderLineageCard({
      title: 'Spaced',
      type: 'artifact',
      rows: [],
      addSpacer: true,
    });

    const { container: noSpacerContainer } = renderLineageCard({
      title: 'Not Spaced',
      type: 'artifact',
      rows: [],
      addSpacer: false,
    });

    expect(getContainerClasses(spacerContainer).length).toBeGreaterThan(
      getContainerClasses(noSpacerContainer).length,
    );
  });

  it('renders one LineageCardRow per element in rows', () => {
    const rows = [buildArtifactRow('row-1'), buildArtifactRow('row-2'), buildArtifactRow('row-3')];

    const { container } = renderLineageCard({
      title: 'Multi Row',
      type: 'artifact',
      rows,
      addSpacer: false,
    });

    const cardBody = container.querySelector('.cardBody');
    expect(cardBody).not.toBeNull();
    expect(cardBody?.children.length).toBe(3);
  });

  it('passes setLineageViewTarget down to each row', () => {
    const setLineageViewTarget = vi.fn();
    const rows = [buildArtifactRow('clickable-row')];

    const { container } = renderLineageCard({
      title: 'Clickable Card',
      type: 'artifact',
      rows,
      addSpacer: false,
      setLineageViewTarget,
    });

    const cardRow = container.querySelector('.cardRow');
    expect(cardRow).not.toBeNull();
    fireEvent.click(cardRow!);
    expect(setLineageViewTarget).toHaveBeenCalledTimes(1);
  });
});
