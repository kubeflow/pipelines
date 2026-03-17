import * as React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { LineageCardRow } from './LineageCardRow';
import { LineageTypedResource } from './LineageTypes';
import { Artifact, Execution } from 'src/third_party/mlmd';
import { ArtifactProperties } from './Api';
import { stringValue } from './TestUtils';

function buildArtifactResource(name: string): LineageTypedResource {
  const artifact = new Artifact();
  artifact.setId(1);
  artifact.getPropertiesMap().set(ArtifactProperties.NAME, stringValue(name));
  return { type: 'artifact', resource: artifact };
}

function buildExecutionResource(): LineageTypedResource {
  const execution = new Execution();
  execution.setId(1);
  return { type: 'execution', resource: execution };
}

function buildDefaultProps(): React.ComponentProps<typeof LineageCardRow> {
  return {
    typedResource: buildArtifactResource('default'),
    resourceDetailsRoute: '/artifacts/1',
    leftAffordance: false,
    rightAffordance: false,
    isLastRow: false,
    hideRadio: false,
  };
}

function renderCardRow(overrides?: Partial<React.ComponentProps<typeof LineageCardRow>>) {
  return render(
    <MemoryRouter>
      <LineageCardRow {...buildDefaultProps()} {...overrides} />
    </MemoryRouter>,
  );
}

describe('LineageCardRow', () => {
  it('renders resource name as a link to resourceDetailsRoute', () => {
    renderCardRow({
      typedResource: buildArtifactResource('my-artifact'),
    });

    const link = screen.getByRole('link');
    expect(link.textContent).toBe('my-artifact');
    expect(link.getAttribute('href')).toBe('/artifacts/1');
  });

  it('renders left and right edge affordance dots when enabled', () => {
    const { container } = renderCardRow({
      leftAffordance: true,
      rightAffordance: true,
    });

    expect(container.querySelector('.edgeLeft')).not.toBeNull();
    expect(container.querySelector('.edgeRight')).not.toBeNull();
  });

  it('omits edge affordance dots when disabled', () => {
    const { container } = renderCardRow({
      leftAffordance: false,
      rightAffordance: false,
    });

    expect(container.querySelector('.edgeLeft')).toBeNull();
    expect(container.querySelector('.edgeRight')).toBeNull();
  });

  it('hides radio button when hideRadio is true', () => {
    const { container } = renderCardRow({ hideRadio: true });

    expect(container.querySelector('.noRadio')).not.toBeNull();
    expect(container.querySelector('.form-radio')).toBeNull();
  });

  it('calls setLineageViewTarget when an artifact row is clicked', () => {
    const setLineageViewTarget = vi.fn();
    const typedResource = buildArtifactResource('clickable');

    const { container } = renderCardRow({ typedResource, setLineageViewTarget });

    fireEvent.click(container.querySelector('.cardRow') as HTMLElement);
    expect(setLineageViewTarget).toHaveBeenCalledTimes(1);
    expect(setLineageViewTarget).toHaveBeenCalledWith(typedResource.resource);
  });

  it('does NOT call setLineageViewTarget when row is an execution type', () => {
    const setLineageViewTarget = vi.fn();

    const { container } = renderCardRow({
      typedResource: buildExecutionResource(),
      resourceDetailsRoute: '/executions/1',
      hideRadio: true,
      setLineageViewTarget,
    });

    fireEvent.click(container.querySelector('.cardRow') as HTMLElement);
    expect(setLineageViewTarget).not.toHaveBeenCalled();
  });

  it('does not throw when clicked without setLineageViewTarget provided', () => {
    const { container } = renderCardRow({
      typedResource: buildArtifactResource('no-handler'),
      setLineageViewTarget: undefined,
    });

    expect(() => {
      fireEvent.click(container.querySelector('.cardRow') as HTMLElement);
    }).not.toThrow();
  });
});
