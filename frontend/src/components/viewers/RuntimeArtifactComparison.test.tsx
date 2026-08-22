// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { ComponentProps, useState } from 'react';
import { ArtifactArtifactType } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import { StorageService } from 'src/lib/WorkflowParser';
import { CommonTestWrapper } from 'src/TestWrapper';
import { testBestPractices } from 'src/TestUtils';
import {
  createRuntimeArtifactComparisonSelectionState,
  RuntimeArtifactComparison,
  RuntimeComparisonArtifact,
  TEST_ONLY,
} from './RuntimeArtifactComparison';

function StatefulRuntimeArtifactComparison(
  props: Omit<
    ComponentProps<typeof RuntimeArtifactComparison>,
    'selectionState' | 'setSelectionState'
  >,
) {
  const [selectionState, setSelectionState] = useState(
    createRuntimeArtifactComparisonSelectionState,
  );
  return (
    <RuntimeArtifactComparison
      {...props}
      selectionState={selectionState}
      setSelectionState={setSelectionState}
    />
  );
}

function getRenderedRocColor(color: string): string {
  const hexMatch = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(color);
  if (hexMatch) {
    return hexMatch
      .slice(1)
      .map((channel) => Number.parseInt(channel, 16))
      .join(',');
  }
  const match = /^hsl\(([\d.]+)deg ([\d.]+)% ([\d.]+)%\)$/.exec(color);
  if (!match) return color;
  const hue = Number(match[1]);
  const saturation = Number(match[2]) / 100;
  const lightness = Number(match[3]) / 100;
  const chroma = (1 - Math.abs(2 * lightness - 1)) * saturation;
  const secondary = chroma * (1 - Math.abs(((hue / 60) % 2) - 1));
  const offset = lightness - chroma / 2;
  let channels: [number, number, number];
  if (hue < 60) channels = [chroma, secondary, 0];
  else if (hue < 120) channels = [secondary, chroma, 0];
  else if (hue < 180) channels = [0, chroma, secondary];
  else if (hue < 240) channels = [0, secondary, chroma];
  else if (hue < 300) channels = [secondary, 0, chroma];
  else channels = [chroma, 0, secondary];
  return channels.map((channel) => Math.round((channel + offset) * 255)).join(',');
}

vi.mock('./ROCCurve', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./ROCCurve')>();
  return {
    ...actual,
    default: ({ configs, colors }: { configs: unknown[]; colors?: string[] }) => (
      <div
        data-colors={colors?.join(',')}
        data-config-count={configs.length}
        data-testid='shared-roc-curve'
      />
    ),
  };
});

testBestPractices();

function classificationEntry(
  run: string,
  artifactId: string,
  metadata: Record<string, object>,
): RuntimeComparisonArtifact {
  return {
    artifact: {
      artifact_id: artifactId,
      name: 'evaluation',
      type: ArtifactArtifactType.ClassificationMetric,
      metadata,
    },
    key: `${run}:${artifactId}`,
    label: `${run} / Evaluate / evaluation`,
    namespace: 'team-a',
  };
}

describe('RuntimeArtifactComparison', () => {
  it('combines ROC curves from different runs on one shared chart with provenance', () => {
    const artifacts = [
      classificationEntry('First run', 'roc-1', {
        confidenceMetrics: [{ confidenceThreshold: 0.8, falsePositiveRate: 0.1, recall: 0.9 }],
      }),
      classificationEntry('Second run', 'roc-2', {
        confidenceMetrics: [{ confidenceThreshold: 0.7, falsePositiveRate: 0.2, recall: 0.95 }],
      }),
    ];

    render(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={artifacts} kind='classification' />
      </CommonTestWrapper>,
    );

    expect(screen.getByTestId('shared-roc-curve')).toHaveAttribute('data-config-count', '2');
    const provenance = screen.getByRole('list', { name: 'Selected ROC curve provenance' });
    within(provenance).getByText('First run / Evaluate / evaluation');
    within(provenance).getByText('Second run / Evaluate / evaluation');
    expect(screen.getAllByRole('combobox', { name: 'ROC curves' })).toHaveLength(1);
  });

  it('uses default ROC selections when every explicit selection is unavailable', () => {
    const artifacts = [
      classificationEntry('First run', 'roc-1', {
        confidenceMetrics: [{ confidenceThreshold: 0.8, falsePositiveRate: 0.1, recall: 0.9 }],
      }),
      classificationEntry('Second run', 'roc-2', {
        confidenceMetrics: [{ confidenceThreshold: 0.7, falsePositiveRate: 0.2, recall: 0.95 }],
      }),
    ];
    const selectionState = {
      ...createRuntimeArtifactComparisonSelectionState(),
      rocSelectedKeys: ['removed-run:removed-curve'],
    };

    render(
      <CommonTestWrapper>
        <RuntimeArtifactComparison
          artifacts={artifacts}
          kind='classification'
          selectionState={selectionState}
          setSelectionState={vi.fn()}
        />
      </CommonTestWrapper>,
    );

    expect(screen.getByTestId('shared-roc-curve')).toHaveAttribute('data-config-count', '2');
    expect(screen.getByRole('combobox', { name: 'ROC curves' })).toHaveTextContent(
      '2 curves selected',
    );
  });

  it('preserves default ROC colors when a default curve is deselected', async () => {
    const artifacts = ['First', 'Second', 'Third'].map((run, index) =>
      classificationEntry(`${run} run`, `roc-${index + 1}`, {
        confidenceMetrics: [
          {
            confidenceThreshold: 0.8 - index * 0.1,
            falsePositiveRate: 0.1 + index * 0.1,
            recall: 0.9,
          },
        ],
      }),
    );

    const view = (kind: 'classification' | 'html') => (
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={artifacts} kind={kind} />
      </CommonTestWrapper>
    );
    const { rerender } = render(view('classification'));

    const initialColors = screen
      .getByTestId('shared-roc-curve')
      .getAttribute('data-colors')!
      .split(',');
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'ROC curves' }));
    fireEvent.click(await screen.findByRole('option', { name: /First run/ }));

    await waitFor(() =>
      expect(screen.getByTestId('shared-roc-curve')).toHaveAttribute('data-config-count', '2'),
    );
    const remainingColors = screen
      .getByTestId('shared-roc-curve')
      .getAttribute('data-colors')!
      .split(',');
    expect(remainingColors).toEqual(initialColors.slice(1));

    fireEvent.click(await screen.findByRole('option', { name: /First run/ }));
    const reselectedColors = screen
      .getByTestId('shared-roc-curve')
      .getAttribute('data-colors')!
      .split(',');
    expect(reselectedColors[0]).toBe(initialColors[0]);

    rerender(view('html'));
    rerender(view('classification'));
    expect(screen.getByTestId('shared-roc-curve').getAttribute('data-colors')!.split(',')).toEqual(
      reselectedColors,
    );
  });

  it('preserves default ROC colors across refreshes before explicit selection', () => {
    const artifacts = Array.from({ length: 3 }, (_, index) =>
      classificationEntry(`Run ${index}`, `roc-${index}`, {
        confidenceMetrics: [
          {
            confidenceThreshold: 0.8,
            falsePositiveRate: index / 100,
            recall: 0.9,
          },
        ],
      }),
    );
    const view = (visibleArtifacts: RuntimeComparisonArtifact[]) => (
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={visibleArtifacts} kind='classification' />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(artifacts));
    const initialColors = screen
      .getByTestId('shared-roc-curve')
      .getAttribute('data-colors')!
      .split(',');
    rerender(view([artifacts[2], artifacts[1]]));

    expect(screen.getByTestId('shared-roc-curve').getAttribute('data-colors')!.split(',')).toEqual([
      initialColors[2],
      initialColors[1],
    ]);
  });

  it('shows the color an unselected ROC curve will receive when selected', async () => {
    const artifacts = Array.from({ length: 5 }, (_, index) =>
      classificationEntry(`Run ${index}`, `roc-${index}`, {
        confidenceMetrics: [
          { confidenceThreshold: 0.8, falsePositiveRate: index / 100, recall: 0.9 },
        ],
      }),
    );

    render(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={artifacts} kind='classification' />
      </CommonTestWrapper>,
    );
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'ROC curves' }));
    const option = await screen.findByRole('option', { name: /Run 3/ });
    const swatchColor = option.querySelector<HTMLElement>('span[style]')!.style.backgroundColor;

    fireEvent.click(option);
    await waitFor(() =>
      expect(screen.getByTestId('shared-roc-curve')).toHaveAttribute('data-config-count', '4'),
    );
    const selectedColor = screen
      .getByTestId('shared-roc-curve')
      .getAttribute('data-colors')!
      .split(',')
      .at(-1)!;
    expect(swatchColor).toBe(`rgb(${getRenderedRocColor(selectedColor).replaceAll(',', ', ')})`);
  });

  it('keeps selector checkboxes out of the tab order and caps large option lists', async () => {
    const artifacts = Array.from({ length: 400 }, (_, index) =>
      classificationEntry(`Run ${index}`, `roc-${index}`, {
        confidenceMetrics: [
          { confidenceThreshold: 0.8, falsePositiveRate: index / 1_000, recall: 0.9 },
        ],
      }),
    );

    render(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={artifacts} kind='classification' />
      </CommonTestWrapper>,
    );
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'ROC curves' }));

    expect(await screen.findAllByRole('option')).toHaveLength(100);
    document.querySelectorAll<HTMLInputElement>('input[type="checkbox"]').forEach((checkbox) => {
      expect(checkbox.tabIndex).toBe(-1);
      expect(checkbox).toHaveAttribute('aria-hidden', 'true');
    });
    expect(screen.getByText(/Showing 100 of 400 curves/)).toBeVisible();
  });

  it('builds two independently selectable confusion-matrix panels', async () => {
    const firstMatrix = {
      annotationSpecs: [{ displayName: 'cat' }, { displayName: 'dog' }],
      rows: [{ row: [2, 0] }, { row: [1, 3] }],
    };
    const secondMatrix = {
      annotationSpecs: [{ displayName: 'cat' }, { displayName: 'dog' }],
      rows: [{ row: [12, 10] }, { row: [11, 13] }],
    };
    const artifacts = [
      classificationEntry('First run', 'matrix-1', { confusionMatrix: firstMatrix }),
      classificationEntry('Second run', 'matrix-2', { confusionMatrix: secondMatrix }),
    ];

    render(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={artifacts} kind='classification' />
      </CommonTestWrapper>,
    );

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'First comparison artifact' }));
    fireEvent.click(
      await screen.findByRole('option', { name: 'First run / Evaluate / evaluation' }),
    );
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Second comparison artifact' }));
    fireEvent.click(
      await screen.findByRole('option', { name: 'Second run / Evaluate / evaluation' }),
    );

    expect(screen.getByTitle('First run / Evaluate / evaluation')).toBeVisible();
    expect(screen.getByTitle('Second run / Evaluate / evaluation')).toBeVisible();

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'First comparison artifact' }));
    fireEvent.click(
      await screen.findByRole('option', { name: 'Second run / Evaluate / evaluation' }),
    );

    expect(screen.getAllByText('12')).toHaveLength(2);
  });

  it('downloads only selected file panels and isolates one artifact failure', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockImplementation(async ({ path }) => {
      if (path.key === 'first.html') {
        throw new Error('first report unavailable');
      }
      return '<h1>Second report</h1>';
    });
    const artifacts: RuntimeComparisonArtifact[] = [
      {
        artifact: {
          artifact_id: 'html-1',
          name: 'first report',
          type: ArtifactArtifactType.HTML,
          uri: 's3://reports/first.html',
        },
        key: 'run-1:html-1',
        label: 'First run / Report / first report',
        namespace: 'team-a',
      },
      {
        artifact: {
          artifact_id: 'html-2',
          name: 'second report',
          type: ArtifactArtifactType.HTML,
          uri: 's3://reports/second.html',
        },
        key: 'run-2:html-2',
        label: 'Second run / Report / second report',
        namespace: 'team-a',
      },
    ];

    render(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={artifacts} kind='html' />
      </CommonTestWrapper>,
    );

    expect(readFileSpy).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'First comparison artifact' }));
    fireEvent.click(
      await screen.findByRole('option', { name: 'First run / Report / first report' }),
    );
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Second comparison artifact' }));
    fireEvent.click(
      await screen.findByRole('option', { name: 'Second run / Report / second report' }),
    );

    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(2));
    expect(readFileSpy).toHaveBeenCalledWith({
      namespace: 'team-a',
      path: {
        bucket: 'reports',
        key: 'second.html',
        keyEncoding: 'storage',
        source: StorageService.S3,
      },
      providerInfo: undefined,
    });
    expect(await screen.findByText(/Unable to retrieve the selected visualization/)).toBeVisible();
    expect(screen.getByTitle('Second run / Report / second report')).toBeVisible();
  });

  it('refetches a selected file when its producer finishes', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Report</h1>');
    readFileSpy.mockClear();
    const artifact: RuntimeComparisonArtifact = {
      artifact: {
        artifact_id: 'live-html',
        name: 'report',
        type: ArtifactArtifactType.HTML,
        uri: 's3://reports/report.html',
      },
      key: 'run-1:live-html',
      label: 'First run / Report / report',
      namespace: 'team-a',
      sourceFinished: false,
    };
    const view = (entry: RuntimeComparisonArtifact) => (
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={[entry]} kind='html' />
      </CommonTestWrapper>
    );

    const { rerender } = render(view(artifact));
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'First comparison artifact' }));
    fireEvent.click(await screen.findByRole('option', { name: artifact.label }));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(1));

    rerender(view({ ...artifact, sourceFinished: true }));
    await waitFor(() => expect(readFileSpy).toHaveBeenCalledTimes(2));

    rerender(view({ ...artifact, sourceFinished: true }));
    expect(readFileSpy).toHaveBeenCalledTimes(2);
  });

  it('preserves selected artifacts while switching comparison tabs', async () => {
    vi.spyOn(Apis, 'readFile').mockResolvedValue('<h1>Report</h1>');
    const htmlEntry: RuntimeComparisonArtifact = {
      artifact: {
        artifact_id: 'html-1',
        name: 'report',
        type: ArtifactArtifactType.HTML,
        uri: 's3://reports/report.html',
      },
      key: 'run-1:html-1',
      label: 'First run / Report / report',
      namespace: 'team-a',
    };
    const markdownEntry: RuntimeComparisonArtifact = {
      artifact: {
        artifact_id: 'markdown-1',
        name: 'summary',
        type: ArtifactArtifactType.Markdown,
        uri: 's3://reports/summary.md',
      },
      key: 'run-1:markdown-1',
      label: 'First run / Report / summary',
      namespace: 'team-a',
    };
    const { rerender } = render(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={[htmlEntry, markdownEntry]} kind='html' />
      </CommonTestWrapper>,
    );
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'First comparison artifact' }));
    fireEvent.click(await screen.findByRole('option', { name: 'First run / Report / report' }));
    await screen.findByTitle('First run / Report / report');

    rerender(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={[htmlEntry, markdownEntry]} kind='markdown' />
      </CommonTestWrapper>,
    );
    rerender(
      <CommonTestWrapper>
        <StatefulRuntimeArtifactComparison artifacts={[htmlEntry, markdownEntry]} kind='html' />
      </CommonTestWrapper>,
    );

    expect(screen.getByRole('combobox', { name: 'First comparison artifact' })).toHaveTextContent(
      'First run / Report / report',
    );
  });

  it('keeps a maximum of ten selected ROC curves', () => {
    expect(
      TEST_ONLY.limitRocSelection(Array.from({ length: 11 }, (_, index) => `${index}`)),
    ).toEqual(Array.from({ length: 10 }, (_, index) => `${index}`));
  });

  it('assigns unique stable colors to the selected ROC curves', () => {
    const keys = Array.from({ length: 10 }, (_, index) => `run-${index + 1}:roc-${index + 1}`);
    const registry = new Map<string, string>();
    const colors = TEST_ONLY.allocateSelectedRocColors(keys, registry);
    const expandedColors = TEST_ONLY.allocateSelectedRocColors(
      ['new-curve', ...keys].slice(0, 10),
      registry,
    );

    expect(new Set(Object.values(colors)).size).toBe(keys.length);
    expect(TEST_ONLY.allocateSelectedRocColors([...keys].reverse())).toEqual(colors);
    const renderedColors = Object.values(colors).map((color) =>
      getRenderedRocColor(color).split(',').map(Number),
    );
    renderedColors.forEach((first, firstIndex) =>
      renderedColors
        .slice(firstIndex + 1)
        .forEach((second) =>
          expect(
            Math.sqrt(
              first.reduce(
                (sum, channel, channelIndex) => sum + (channel - second[channelIndex]) ** 2,
                0,
              ),
            ),
          ).toBeGreaterThan(40),
        ),
    );
    keys.slice(0, 9).forEach((key) => expect(expandedColors[key]).toBe(colors[key]));
  });

  it('maps hundreds of unselected identities directly without an allocation loop', () => {
    const keys = Array.from({ length: 400 }, (_, index) => `curve-${index}`);
    const colors = keys.map(TEST_ONLY.getStableDefaultRocColor);

    expect(colors).toHaveLength(keys.length);
    expect([...keys].reverse().map(TEST_ONLY.getStableDefaultRocColor).reverse()).toEqual(colors);
  });

  it.each([
    ['Run 1204:roc-1204:roc-1204', 'Run 1402:roc-1402:roc-1402'],
    ['Run 341:roc-341:roc-341', 'Run 521:roc-521:roc-521'],
    ['Run 199:roc-199:roc-199', 'Run 408:roc-408:roc-408'],
    ['Run 28:roc-28:roc-28', 'Run 84:roc-84:roc-84'],
  ])('keeps colliding selected ROC colors perceptually separated', (firstKey, secondKey) => {
    const allocatedColors = TEST_ONLY.allocateSelectedRocColors([firstKey, secondKey]);
    const renderedColors = [firstKey, secondKey]
      .map((key) => allocatedColors[key])
      .map((color) => getRenderedRocColor(color).split(',').map(Number));
    const rgbDistance = Math.sqrt(
      renderedColors[0].reduce(
        (sum, channel, index) => sum + (channel - renderedColors[1][index]) ** 2,
        0,
      ),
    );

    expect(rgbDistance).toBeGreaterThan(40);
  });
});
