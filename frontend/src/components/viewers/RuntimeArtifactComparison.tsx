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

import {
  Checkbox,
  FormControl,
  InputLabel,
  ListItemText,
  MenuItem,
  Select,
  SelectChangeEvent,
} from '@mui/material';
import { useMemo, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { V2beta1Artifact } from 'src/apisv2beta1/run';
import Banner from 'src/components/Banner';
import PlotCard from 'src/components/PlotCard';
import { padding } from 'src/Css';
import {
  getArtifactDisplayName,
  isClassificationMetricArtifact,
  isHtmlArtifact,
  isMarkdownArtifact,
} from 'src/lib/v2/RuntimeArtifactUtils';
import { stylesheet } from 'typestyle';
import { ConfusionMatrixConfig } from './ConfusionMatrix';
import ROCCurve, { lineColors, ROCCurveConfig } from './ROCCurve';
import {
  buildConfusionMatrixResult,
  buildRocCurves,
  ClassificationVisualization,
  expandClassificationMetrics,
  RuntimeArtifactVisualization,
} from './RuntimeMetricsVisualizations';

const MAX_SELECTED_ROC_CURVES = 10;
const DEFAULT_SELECTED_ROC_CURVES = 3;
const MAX_ROC_SELECTOR_OPTIONS = 100;

const css = stylesheet({
  comparisonGrid: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: 24,
    paddingTop: 16,
  },
  comparisonPanel: {
    flex: '1 1 420px',
    minWidth: 320,
  },
  curveLegend: {
    display: 'grid',
    gap: 6,
    listStyle: 'none',
    margin: '12px 0 0',
    padding: 0,
  },
  curveLegendItem: {
    alignItems: 'center',
    display: 'flex',
    gap: 8,
  },
  curveSwatch: {
    borderRadius: 2,
    display: 'inline-block',
    flex: '0 0 auto',
    height: 10,
    width: 10,
  },
  selector: {
    minWidth: 300,
    width: '100%',
  },
  selectorSection: {
    maxWidth: 720,
  },
});

export type RuntimeArtifactComparisonKind = 'classification' | 'html' | 'markdown';

export interface RuntimeComparisonArtifact {
  artifact: V2beta1Artifact;
  key: string;
  label: string;
  namespace?: string;
  sourceFinished?: boolean;
}

interface ComparisonPanelEntry {
  artifact?: V2beta1Artifact;
  configs?: ConfusionMatrixConfig[];
  key: string;
  label: string;
  namespace?: string;
  sourceFinished?: boolean;
}

interface RocComparisonEntry {
  config: ROCCurveConfig;
  key: string;
  label: string;
}

type ComparisonPanelKind = 'confusion matrix' | 'html' | 'markdown';
type PanelSelections = Record<ComparisonPanelKind, [string, string]>;

export interface RuntimeArtifactComparisonSelectionState {
  panelSelections: PanelSelections;
  rocColorByKey?: Record<string, string>;
  rocSelectedKeys?: string[];
}

export function createRuntimeArtifactComparisonSelectionState(): RuntimeArtifactComparisonSelectionState {
  return {
    panelSelections: {
      'confusion matrix': ['', ''],
      html: ['', ''],
      markdown: ['', ''],
    },
  };
}

export function RuntimeArtifactComparison({
  artifacts,
  kind,
  selectionState,
  setSelectionState,
}: {
  artifacts: RuntimeComparisonArtifact[];
  kind: RuntimeArtifactComparisonKind;
  selectionState: RuntimeArtifactComparisonSelectionState;
  setSelectionState: Dispatch<SetStateAction<RuntimeArtifactComparisonSelectionState>>;
}) {
  const updatePanelSelection = (
    panelKind: ComparisonPanelKind,
    panelIndex: number,
    key: string,
  ) => {
    setSelectionState((current) => {
      const nextSelection = [...current.panelSelections[panelKind]] as [string, string];
      nextSelection[panelIndex] = key;
      return {
        ...current,
        panelSelections: { ...current.panelSelections, [panelKind]: nextSelection },
      };
    });
  };

  if (kind === 'classification') {
    const classificationArtifacts = artifacts.filter(({ artifact }) =>
      isClassificationMetricArtifact(artifact),
    );
    if (!classificationArtifacts.length) {
      return <p>There are no Classification Metrics available on the selected runs.</p>;
    }
    return (
      <ClassificationComparison
        artifacts={classificationArtifacts}
        panelSelections={selectionState.panelSelections['confusion matrix']}
        rocColorByKey={selectionState.rocColorByKey}
        rocSelectedKeys={selectionState.rocSelectedKeys}
        updateSelectionState={setSelectionState}
        updatePanelSelection={updatePanelSelection}
      />
    );
  }

  const fileArtifacts = artifacts.filter(({ artifact }) =>
    kind === 'html' ? isHtmlArtifact(artifact) : isMarkdownArtifact(artifact),
  );
  if (!fileArtifacts.length) {
    return (
      <p>There are no {kind === 'html' ? 'HTML' : 'Markdown'} available on the selected runs.</p>
    );
  }
  return (
    <TwoPanelComparison
      entries={fileArtifacts}
      kind={kind}
      selectedKeys={selectionState.panelSelections[kind]}
      updatePanelSelection={updatePanelSelection}
    />
  );
}

function ClassificationComparison({
  artifacts,
  panelSelections,
  rocColorByKey,
  rocSelectedKeys,
  updateSelectionState,
  updatePanelSelection,
}: {
  artifacts: RuntimeComparisonArtifact[];
  panelSelections: [string, string];
  rocColorByKey: Record<string, string> | undefined;
  rocSelectedKeys: string[] | undefined;
  updateSelectionState: Dispatch<SetStateAction<RuntimeArtifactComparisonSelectionState>>;
  updatePanelSelection: (kind: ComparisonPanelKind, panelIndex: number, key: string) => void;
}) {
  const visualizations = useMemo(
    () => buildComparisonClassificationVisualizations(artifacts),
    [artifacts],
  );
  const { entries: rocEntries, errors: rocErrors } = useMemo(
    () => buildRocComparisonEntries(visualizations),
    [visualizations],
  );
  const { matrixEntries, matrixErrors } = useMemo(() => {
    const result = buildConfusionMatrixResult(visualizations);
    return {
      matrixEntries: result.matrices.map(({ visualization, configs }) => ({
        configs,
        key: visualization.key,
        label: visualization.displayName,
      })),
      matrixErrors: result.errors,
    };
  }, [visualizations]);

  return (
    <>
      {!!rocEntries.length && (
        <RocCurveComparison
          entries={rocEntries}
          errors={rocErrors}
          explicitColorByKey={rocColorByKey}
          explicitSelectedKeys={rocSelectedKeys}
          updateSelectionState={updateSelectionState}
        />
      )}
      {!rocEntries.length && !!rocErrors.length && (
        <Banner
          message='The selected runs contain invalid ROC curve artifacts.'
          mode='error'
          additionalInfo={rocErrors.join('\n')}
        />
      )}
      {!!matrixEntries.length && (
        <TwoPanelComparison
          entries={matrixEntries}
          kind='confusion matrix'
          selectedKeys={panelSelections}
          updatePanelSelection={updatePanelSelection}
        />
      )}
      {!!matrixErrors.length && (
        <Banner
          message='The selected runs contain invalid confusion matrix artifacts.'
          mode='error'
          additionalInfo={matrixErrors.join('\n')}
        />
      )}
      {!rocEntries.length && !rocErrors.length && !matrixEntries.length && !matrixErrors.length && (
        <p>There are no ROC curves or confusion matrices available on the selected runs.</p>
      )}
    </>
  );
}

function RocCurveComparison({
  entries,
  errors,
  explicitColorByKey,
  explicitSelectedKeys,
  updateSelectionState,
}: {
  entries: RocComparisonEntry[];
  errors: string[];
  explicitColorByKey: Record<string, string> | undefined;
  explicitSelectedKeys: string[] | undefined;
  updateSelectionState: Dispatch<SetStateAction<RuntimeArtifactComparisonSelectionState>>;
}) {
  const validKeys = useMemo(() => new Set(entries.map(({ key }) => key)), [entries]);
  const explicitValidKeys = explicitSelectedKeys?.filter((key) => validKeys.has(key));
  const shouldUseDefaults =
    explicitSelectedKeys === undefined ||
    (!!explicitSelectedKeys.length && !explicitValidKeys?.length);
  const selectedKeys = shouldUseDefaults
    ? entries.slice(0, DEFAULT_SELECTED_ROC_CURVES).map(({ key }) => key)
    : explicitValidKeys || [];
  const selectedKeySet = new Set(selectedKeys);
  const selectedEntries = entries.filter(({ key }) => selectedKeySet.has(key));
  const selectedKeySetId = JSON.stringify([...selectedKeys].sort());
  const [colorState, setColorState] = useState<{
    colors: Record<string, string>;
    keySetId: string;
    registry: Record<string, string>;
  }>(() => {
    const registry = new Map(Object.entries(explicitColorByKey || {}));
    return {
      colors: allocateSelectedRocColors(selectedKeys, registry),
      keySetId: selectedKeySetId,
      registry: Object.fromEntries(registry),
    };
  });
  let currentColorState = colorState;
  if (colorState.keySetId !== selectedKeySetId) {
    const registry = new Map(Object.entries(colorState.registry));
    const colors = allocateSelectedRocColors(selectedKeys, registry);
    currentColorState = {
      colors,
      keySetId: selectedKeySetId,
      registry: Object.fromEntries(registry),
    };
    // This guarded render-phase update preserves prior identity assignments without an effect-driven
    // state-reset chain. React immediately retries this component with the reconciled selection set.
    setColorState(currentColorState);
  }
  const selectedColors = currentColorState.colors;
  const getColor = (key: string) =>
    selectedColors[key] || currentColorState.registry[key] || getStableDefaultRocColor(key);
  const initiallyVisibleEntries = entries.slice(0, MAX_ROC_SELECTOR_OPTIONS);
  const initiallyVisibleKeys = new Set(initiallyVisibleEntries.map(({ key }) => key));
  const selectorEntries = [
    ...selectedEntries.filter(({ key }) => !initiallyVisibleKeys.has(key)),
    ...initiallyVisibleEntries,
  ].slice(0, MAX_ROC_SELECTOR_OPTIONS);
  const handleSelection = (event: SelectChangeEvent<string[]>) => {
    const value = event.target.value;
    const nextKeys = limitRocSelection(typeof value === 'string' ? value.split(',') : value);
    const registry = new Map(Object.entries(currentColorState.registry));
    const nextColorState = {
      colors: allocateSelectedRocColors(nextKeys, registry),
      keySetId: JSON.stringify([...nextKeys].sort()),
      registry: Object.fromEntries(registry),
    };
    setColorState(nextColorState);
    updateSelectionState((current) => ({
      ...current,
      rocColorByKey: nextColorState.registry,
      rocSelectedKeys: nextKeys,
    }));
  };

  return (
    <section className={padding(20, 'lrt')}>
      <h3>Cross-run ROC curve comparison</h3>
      <div className={css.selectorSection}>
        <FormControl className={css.selector} variant='standard'>
          <InputLabel id='roc-comparison-label'>ROC curves</InputLabel>
          <Select
            labelId='roc-comparison-label'
            multiple
            value={selectedKeys}
            onChange={handleSelection}
            renderValue={(value) =>
              `${value.length} curve${value.length === 1 ? '' : 's'} selected`
            }
            inputProps={{ 'aria-label': 'ROC curves' }}
          >
            {selectorEntries.map(({ key, label }) => (
              <MenuItem
                disabled={
                  selectedKeys.length >= MAX_SELECTED_ROC_CURVES && !selectedKeySet.has(key)
                }
                key={key}
                value={key}
              >
                <Checkbox
                  checked={selectedKeySet.has(key)}
                  inputProps={{ 'aria-hidden': true }}
                  tabIndex={-1}
                />
                <span
                  aria-hidden='true'
                  className={css.curveSwatch}
                  style={{ backgroundColor: getColor(key) }}
                />
                <ListItemText primary={label} />
              </MenuItem>
            ))}
          </Select>
          {entries.length > selectorEntries.length && (
            <p>
              Showing {selectorEntries.length} of {entries.length} curves. Narrow the compared runs
              to choose from the remaining curves.
            </p>
          )}
        </FormControl>
      </div>
      {!!errors.length && (
        <Banner
          message='Some ROC curve artifacts could not be displayed.'
          mode='error'
          additionalInfo={errors.join('\n')}
        />
      )}
      {!!selectedEntries.length && (
        <>
          <ROCCurve
            colors={selectedEntries.map(({ key }) => getColor(key))}
            configs={selectedEntries.map(({ config }) => config)}
            disableAnimation
            forceLegend
          />
          <ul aria-label='Selected ROC curve provenance' className={css.curveLegend}>
            {selectedEntries.map(({ key, label }) => (
              <li className={css.curveLegendItem} key={key}>
                <span
                  aria-hidden='true'
                  className={css.curveSwatch}
                  style={{ backgroundColor: getColor(key) }}
                />
                {label}
              </li>
            ))}
          </ul>
        </>
      )}
      {!selectedEntries.length && (
        <Banner message='Select at least one ROC curve to compare.' mode='info' />
      )}
    </section>
  );
}

function TwoPanelComparison({
  entries,
  kind,
  selectedKeys,
  updatePanelSelection,
}: {
  entries: ComparisonPanelEntry[];
  kind: ComparisonPanelKind;
  selectedKeys: [string, string];
  updatePanelSelection: (kind: ComparisonPanelKind, panelIndex: number, key: string) => void;
}) {
  const entryByKey = new Map(entries.map((entry) => [entry.key, entry]));
  const activeSelectedKeys = selectedKeys.map((key) => (entryByKey.has(key) ? key : '')) as [
    string,
    string,
  ];

  return (
    <section className={padding(20, 'lrt')}>
      <h3>Side-by-side {kind} comparison</h3>
      <div className={css.comparisonGrid}>
        {activeSelectedKeys.map((selectedKey, panelIndex) => {
          const entry = entryByKey.get(selectedKey);
          const ordinal = panelIndex === 0 ? 'First' : 'Second';
          const labelId = `${kind.replace(/\s/g, '-')}-comparison-${panelIndex}`;
          return (
            <div className={css.comparisonPanel} key={labelId}>
              <FormControl className={css.selector} variant='standard'>
                <InputLabel id={labelId}>{ordinal} comparison artifact</InputLabel>
                <Select
                  labelId={labelId}
                  value={selectedKey}
                  onChange={(event) =>
                    updatePanelSelection(kind, panelIndex, event.target.value as string)
                  }
                  inputProps={{ 'aria-label': `${ordinal} comparison artifact` }}
                >
                  <MenuItem value=''>Choose an artifact</MenuItem>
                  {entries.map((candidate) => (
                    <MenuItem key={candidate.key} value={candidate.key}>
                      {candidate.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              {entry?.configs && (
                <PlotCard configs={entry.configs} key={entry.key} title={entry.label} />
              )}
              {entry?.artifact && (
                <RuntimeArtifactVisualization
                  artifact={entry.artifact}
                  namespace={entry.namespace}
                  sourceFinished={entry.sourceFinished}
                  title={entry.label}
                />
              )}
              {!entry && <p>The selected {kind} will be displayed here.</p>}
            </div>
          );
        })}
      </div>
    </section>
  );
}

function buildComparisonClassificationVisualizations(
  artifacts: RuntimeComparisonArtifact[],
): ClassificationVisualization[] {
  return artifacts.flatMap((entry) =>
    expandClassificationMetrics([entry.artifact]).map((visualization) => {
      const artifactDisplayName = getArtifactDisplayName(entry.artifact);
      const detailPrefix = `${artifactDisplayName} · `;
      const detail = visualization.displayName.startsWith(detailPrefix)
        ? visualization.displayName.slice(detailPrefix.length)
        : '';
      return {
        ...visualization,
        displayName: detail ? `${entry.label} / ${detail}` : entry.label,
        key: `${entry.key}:${visualization.key}`,
      };
    }),
  );
}

function buildRocComparisonEntries(visualizations: ClassificationVisualization[]): {
  entries: RocComparisonEntry[];
  errors: string[];
} {
  const entries: RocComparisonEntry[] = [];
  const errors: string[] = [];
  visualizations.forEach((visualization) => {
    const result = buildRocCurves([visualization]);
    if (result.error) {
      errors.push(result.error);
    }
    if (result.configs[0]) {
      entries.push({
        config: result.configs[0],
        key: visualization.key,
        label: visualization.displayName,
      });
    }
  });
  return { entries, errors };
}

function limitRocSelection(keys: string[]): string[] {
  return keys.slice(0, MAX_SELECTED_ROC_CURVES);
}

function getStableDefaultRocColor(key: string): string {
  // Use an identity-only fallback for unselected menu entries. Selected curves use the persistent
  // bounded allocation below so overlapping lines remain perceptually distinguishable.
  const unsignedHash = getStableRocHash(key);
  const hue = unsignedHash % 360;
  const saturation = 55 + ((unsignedHash >>> 9) % 36);
  const lightness = 32 + ((unsignedHash >>> 17) % 29);
  return `hsl(${hue}deg ${saturation}% ${lightness}%)`;
}

function allocateSelectedRocColors(
  keys: string[],
  registry: Map<string, string> = new Map(),
): Record<string, string> {
  const usedColors = new Set<string>();
  const colors: Record<string, string> = {};
  const allocationKeys = [...keys].sort();
  // Reserve every surviving assignment before considering new keys. Otherwise an inserted key at
  // the front could claim an existing curve's color and force that survivor to move.
  allocationKeys.forEach((key) => {
    const existingColor = registry.get(key);
    if (existingColor && !usedColors.has(existingColor)) {
      colors[key] = existingColor;
      usedColors.add(existingColor);
    }
  });
  allocationKeys.forEach((key) => {
    if (colors[key]) {
      return;
    }
    const startIndex = getStableRocHash(key) % lineColors.length;
    const color =
      Array.from(
        { length: lineColors.length },
        (_, offset) => lineColors[(startIndex + offset) % lineColors.length],
      ).find((candidate) => !usedColors.has(candidate)) || lineColors[startIndex];
    registry.set(key, color);
    colors[key] = color;
    usedColors.add(color);
  });
  return colors;
}

function getStableRocHash(key: string): number {
  let hash = 2166136261;
  for (let index = 0; index < key.length; index++) {
    hash ^= key.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  hash ^= hash >>> 16;
  hash = Math.imul(hash, 0x85ebca6b);
  hash ^= hash >>> 13;
  hash = Math.imul(hash, 0xc2b2ae35);
  hash ^= hash >>> 16;
  return hash >>> 0;
}

export const TEST_ONLY = {
  allocateSelectedRocColors,
  buildComparisonClassificationVisualizations,
  buildRocComparisonEntries,
  getStableDefaultRocColor,
  limitRocSelection,
};

export default RuntimeArtifactComparison;
