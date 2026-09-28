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

import { Select as SelectPrimitive } from '@base-ui/react/select';
import { Check, ChevronDown } from 'lucide-react';
import { Button } from 'src/components/ui/button';
import { Input } from 'src/components/ui/input';
import { InspectionTabs } from 'src/components/modernization/InspectionTabs';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { V2beta1Artifact } from 'src/apisv2beta1/run';
import { VisualizationNotice } from './VisualizationNotice';
import PlotCard from 'src/components/PlotCard';
import {
  getArtifactDisplayName,
  isClassificationMetricArtifact,
  isHtmlArtifact,
  isMarkdownArtifact,
} from 'src/lib/v2/RuntimeArtifactUtils';
import './ComparisonViewers.css';
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

  const [selectedView, setSelectedView] = useState<'roc' | 'matrix'>('roc');
  const hasRoc = !!(rocEntries.length || rocErrors.length);
  const hasMatrix = !!(matrixEntries.length || matrixErrors.length);
  const activeView = hasRoc && hasMatrix ? selectedView : hasMatrix ? 'matrix' : 'roc';

  return (
    <>
      {(hasRoc || hasMatrix) && (
        <InspectionTabs
          ariaLabel='Classification visualization'
          tabs={[
            { label: 'ROC curves', disabled: !hasRoc },
            { label: 'Confusion matrix', disabled: !hasMatrix },
          ]}
          selectedTab={activeView === 'roc' ? 0 : 1}
          onSwitch={(index) => setSelectedView(index === 0 ? 'roc' : 'matrix')}
        >
          <div hidden={activeView !== 'roc'}>
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
              <VisualizationNotice
                message='The selected runs contain invalid ROC curve artifacts.'
                variant='error'
                details={rocErrors.join('\n')}
              />
            )}
          </div>
          <div hidden={activeView !== 'matrix'}>
            {!!matrixEntries.length && (
              <TwoPanelComparison
                entries={matrixEntries}
                kind='confusion matrix'
                selectedKeys={panelSelections}
                updatePanelSelection={updatePanelSelection}
              />
            )}
            {!!matrixErrors.length && (
              <VisualizationNotice
                message='The selected runs contain invalid confusion matrix artifacts.'
                variant='error'
                details={matrixErrors.join('\n')}
              />
            )}
          </div>
        </InspectionTabs>
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
  const [selectorFilter, setSelectorFilter] = useState('');
  const [portalContainer, setPortalContainer] = useState<HTMLElement>();
  const setSelectTrigger = useCallback((node: HTMLButtonElement | null) => {
    if (node) setPortalContainer(node.closest<HTMLElement>('.kfp-theme') || undefined);
  }, []);
  const [selectorPage, setSelectorPage] = useState(0);
  const [chartExpanded, setChartExpanded] = useState(false);
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
    const colors = allocateSelectedRocColors(
      selectedKeys,
      registry,
      Object.keys(colorState.colors),
    );
    currentColorState = {
      colors,
      keySetId: selectedKeySetId,
      registry: Object.fromEntries(registry),
    };
    // This guarded render-phase update preserves prior identity assignments without an effect-driven
    // state-reset chain. React immediately retries this component with the reconciled selection set.
    setColorState(currentColorState);
  }
  useEffect(() => {
    // Persist implicit query-refresh reconciliation in the parent-owned comparison state. This is
    // the lifecycle boundary that survives switching visualization tabs or collapsing the section;
    // selection events already write through synchronously below.
    updateSelectionState((current) =>
      areRocColorRegistriesEqual(current.rocColorByKey, currentColorState.registry)
        ? current
        : { ...current, rocColorByKey: currentColorState.registry },
    );
  }, [currentColorState.registry, updateSelectionState]);
  const selectedColors = currentColorState.colors;
  const getColor = (key: string) =>
    selectedColors[key] || currentColorState.registry[key] || getStableDefaultRocColor(key);
  const getSelectorColor = (key: string) => {
    if (selectedKeySet.has(key)) {
      return getColor(key);
    }
    const registry = new Map(Object.entries(currentColorState.registry));
    return allocateSelectedRocColors([...selectedKeys, key], registry, selectedKeys)[key];
  };
  const normalizedFilter = selectorFilter.trim().toLocaleLowerCase();
  const matchingEntries = normalizedFilter
    ? entries.filter(
        ({ key, label }) =>
          key.toLocaleLowerCase().includes(normalizedFilter) ||
          label.toLocaleLowerCase().includes(normalizedFilter),
      )
    : entries;
  const selectorPageCount = Math.max(
    1,
    Math.ceil(matchingEntries.length / MAX_ROC_SELECTOR_OPTIONS),
  );
  const visibleSelectorPage = Math.min(selectorPage, selectorPageCount - 1);
  const selectorStart = visibleSelectorPage * MAX_ROC_SELECTOR_OPTIONS;
  const selectorEntries = matchingEntries.slice(
    selectorStart,
    selectorStart + MAX_ROC_SELECTOR_OPTIONS,
  );
  const handleSelection = (value: string[], details: SelectPrimitive.Root.ChangeEventDetails) => {
    if (details.reason === 'none') {
      // Only this page of options is mounted. Preserve off-page selections when Base UI
      // reconciles removed options; validKeys above reconciles against the full artifact set.
      details.cancel();
      return;
    }
    const nextKeys = limitRocSelection(value);
    const registry = new Map(Object.entries(currentColorState.registry));
    const nextColorState = {
      colors: allocateSelectedRocColors(nextKeys, registry, selectedKeys),
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
    <section className='kfp-roc-section'>
      <h3 className='kfp-roc-heading'>Cross-run ROC curve comparison</h3>
      <div className='kfp-visualization-selectors'>
        <label className='kfp-visualization-field'>
          Search ROC curves
          <Input
            onChange={(event) => {
              setSelectorFilter(event.target.value);
              setSelectorPage(0);
            }}
            value={selectorFilter}
          />
        </label>
        <div className='kfp-visualization-field'>
          <span id='roc-comparison-label'>ROC curves</span>
          <SelectPrimitive.Root multiple value={selectedKeys} onValueChange={handleSelection}>
            <SelectPrimitive.Trigger
              ref={setSelectTrigger}
              className='kfp-visualization-select'
              aria-labelledby='roc-comparison-label'
            >
              {selectedKeys.length} curve{selectedKeys.length === 1 ? '' : 's'} selected
              <SelectPrimitive.Icon>
                <ChevronDown size={16} aria-hidden='true' />
              </SelectPrimitive.Icon>
            </SelectPrimitive.Trigger>
            <SelectPrimitive.Portal container={portalContainer}>
              <SelectPrimitive.Positioner
                className='kfp-visualization-popup-positioner'
                sideOffset={4}
              >
                <SelectPrimitive.Popup className='kfp-visualization-popup'>
                  <SelectPrimitive.List aria-label='ROC curves'>
                    {selectorEntries.map(({ key, label }) => (
                      <SelectPrimitive.Item
                        key={key}
                        value={key}
                        label={label}
                        className='kfp-visualization-option'
                        disabled={
                          selectedKeys.length >= MAX_SELECTED_ROC_CURVES && !selectedKeySet.has(key)
                        }
                      >
                        <SelectPrimitive.ItemIndicator className='kfp-visualization-check'>
                          <Check size={14} aria-hidden='true' />
                        </SelectPrimitive.ItemIndicator>
                        <span
                          aria-hidden='true'
                          className='kfp-curve-swatch'
                          style={{ backgroundColor: getSelectorColor(key) }}
                        />
                        <SelectPrimitive.ItemText>{label}</SelectPrimitive.ItemText>
                      </SelectPrimitive.Item>
                    ))}
                  </SelectPrimitive.List>
                </SelectPrimitive.Popup>
              </SelectPrimitive.Positioner>
            </SelectPrimitive.Portal>
          </SelectPrimitive.Root>
        </div>
        {!matchingEntries.length ? (
          <p aria-live='polite' role='status'>
            No ROC curves match this search.
          </p>
        ) : matchingEntries.length > MAX_ROC_SELECTOR_OPTIONS ? (
          <nav aria-label='ROC curve result pages'>
            <Button
              variant='secondary'
              disabled={visibleSelectorPage === 0}
              onClick={() => setSelectorPage((page) => Math.max(0, page - 1))}
              type='button'
            >
              Previous ROC curves
            </Button>
            <span aria-live='polite'>
              Showing {selectorStart + 1}–
              {Math.min(selectorStart + selectorEntries.length, matchingEntries.length)} of{' '}
              {matchingEntries.length} {normalizedFilter ? 'matching ' : ''}curves.
            </span>
            <Button
              variant='secondary'
              disabled={visibleSelectorPage >= selectorPageCount - 1}
              onClick={() => setSelectorPage((page) => Math.min(selectorPageCount - 1, page + 1))}
              type='button'
            >
              Next ROC curves
            </Button>
          </nav>
        ) : null}
      </div>
      {!!errors.length && (
        <VisualizationNotice
          message='Some ROC curve artifacts could not be displayed.'
          variant='error'
          details={errors.join('\n')}
        />
      )}
      {!!selectedEntries.length && (
        <ROCCurve
          colors={selectedEntries.map(({ key }) => getColor(key))}
          configs={selectedEntries.map(({ config }) => config)}
          labels={selectedEntries.map(({ label }) => label)}
          compactLegend
          maxChartHeight={chartExpanded ? undefined : 360}
          responsive
          disableAnimation
          forceLegend
        />
      )}
      {!!selectedEntries.length && (
        <Button
          variant='secondary'
          type='button'
          aria-expanded={chartExpanded}
          onClick={() => setChartExpanded((expanded) => !expanded)}
        >
          {chartExpanded ? 'Compact ROC chart' : 'Expand ROC chart'}
        </Button>
      )}
      {!selectedEntries.length && (
        <VisualizationNotice message='Select at least one ROC curve to compare.' variant='info' />
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
    <section className='kfp-visualization-section' aria-label={`Side-by-side ${kind} comparison`}>
      <div className='kfp-comparison-grid'>
        {activeSelectedKeys.map((selectedKey, panelIndex) => {
          const entry = entryByKey.get(selectedKey);
          const ordinal = panelIndex === 0 ? 'First' : 'Second';
          const labelId = `${kind.replace(/\s/g, '-')}-comparison-${panelIndex}`;
          return (
            <div className='kfp-comparison-panel' key={labelId}>
              <label className='kfp-visualization-field' htmlFor={labelId}>
                {ordinal} comparison artifact
                <select
                  id={labelId}
                  className='kfp-visualization-select'
                  value={selectedKey}
                  onChange={(event) => updatePanelSelection(kind, panelIndex, event.target.value)}
                >
                  <option value=''>Choose an artifact</option>
                  {entries.map((candidate) => (
                    <option key={candidate.key} value={candidate.key}>
                      {candidate.label}
                    </option>
                  ))}
                </select>
              </label>
              {entry?.configs && (
                <PlotCard configs={entry.configs} key={entry.key} title={entry.label} />
              )}
              {entry?.artifact && (
                <RuntimeArtifactVisualization
                  artifact={entry.artifact}
                  namespace={entry.namespace}
                  sourceFinished={entry.sourceFinished}
                  title={kind === 'html' ? 'HTML report' : 'Markdown report'}
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
  priorityKeys: string[] = [],
): Record<string, string> {
  const usedColors = new Set<string>();
  const colors: Record<string, string> = {};
  const keySet = new Set(keys);
  const prioritized = priorityKeys.filter(
    (key, index) => keySet.has(key) && priorityKeys.indexOf(key) === index,
  );
  const prioritizedSet = new Set(prioritized);
  const allocationKeys = [...prioritized, ...keys.filter((key) => !prioritizedSet.has(key)).sort()];
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

function areRocColorRegistriesEqual(
  left: Record<string, string> | undefined,
  right: Record<string, string>,
): boolean {
  if (!left) {
    return !Object.keys(right).length;
  }
  const rightEntries = Object.entries(right);
  return (
    Object.keys(left).length === rightEntries.length &&
    rightEntries.every(([key, color]) => left[key] === color)
  );
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
