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
  rocColorByKey: Record<string, string>;
  rocSelectedKeys?: string[];
}

export function createRuntimeArtifactComparisonSelectionState(): RuntimeArtifactComparisonSelectionState {
  return {
    panelSelections: {
      'confusion matrix': ['', ''],
      html: ['', ''],
      markdown: ['', ''],
    },
    rocColorByKey: {},
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
  selectionState?: RuntimeArtifactComparisonSelectionState;
  setSelectionState?: Dispatch<SetStateAction<RuntimeArtifactComparisonSelectionState>>;
}) {
  const [internalSelectionState, setInternalSelectionState] = useState(
    createRuntimeArtifactComparisonSelectionState,
  );
  const isControlled = selectionState !== undefined && setSelectionState !== undefined;
  const activeSelectionState = isControlled ? selectionState : internalSelectionState;
  const updateSelectionState = isControlled ? setSelectionState : setInternalSelectionState;
  const updatePanelSelection = (
    panelKind: ComparisonPanelKind,
    panelIndex: number,
    key: string,
  ) => {
    updateSelectionState((current) => {
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
        panelSelections={activeSelectionState.panelSelections['confusion matrix']}
        rocColorByKey={activeSelectionState.rocColorByKey}
        rocSelectedKeys={activeSelectionState.rocSelectedKeys}
        updateSelectionState={updateSelectionState}
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
      selectedKeys={activeSelectionState.panelSelections[kind]}
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
  rocColorByKey: Record<string, string>;
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
          explicitSelectedKeys={rocSelectedKeys}
          rocColorByKey={rocColorByKey}
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
  explicitSelectedKeys,
  rocColorByKey,
  updateSelectionState,
}: {
  entries: RocComparisonEntry[];
  errors: string[];
  explicitSelectedKeys: string[] | undefined;
  rocColorByKey: Record<string, string>;
  updateSelectionState: Dispatch<SetStateAction<RuntimeArtifactComparisonSelectionState>>;
}) {
  const validKeys = useMemo(() => new Set(entries.map(({ key }) => key)), [entries]);
  const selectedKeys = (
    explicitSelectedKeys || entries.slice(0, DEFAULT_SELECTED_ROC_CURVES).map(({ key }) => key)
  ).filter((key) => validKeys.has(key));
  const selectedKeySet = new Set(selectedKeys);
  const selectedEntries = entries.filter(({ key }) => selectedKeySet.has(key));
  const selectedColorByKey = allocateRocColors(selectedKeys, rocColorByKey);
  const getColor = (key: string) => selectedColorByKey[key] || getStableRocColor(key);
  const handleSelection = (event: SelectChangeEvent<string[]>) => {
    const value = event.target.value;
    const nextKeys = limitRocSelection(typeof value === 'string' ? value.split(',') : value);
    updateSelectionState((current) => ({
      ...current,
      rocColorByKey: allocateRocColors(nextKeys, current.rocColorByKey),
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
            {entries.map(({ key, label }) => (
              <MenuItem
                disabled={
                  selectedKeys.length >= MAX_SELECTED_ROC_CURVES && !selectedKeySet.has(key)
                }
                key={key}
                value={key}
              >
                <Checkbox checked={selectedKeySet.has(key)} />
                <span
                  aria-hidden='true'
                  className={css.curveSwatch}
                  style={{ backgroundColor: getColor(key) }}
                />
                <ListItemText primary={label} />
              </MenuItem>
            ))}
          </Select>
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

function getStableRocColor(key: string): string {
  let hash = 0;
  for (let index = 0; index < key.length; index++) {
    hash = (hash * 31 + key.charCodeAt(index)) | 0;
  }
  return lineColors[Math.abs(hash) % lineColors.length];
}

function allocateRocColors(
  keys: string[],
  existingColors: Record<string, string> = {},
): Record<string, string> {
  const colorsByKey: Record<string, string> = {};
  const usedColors = new Set<string>();

  keys.forEach((key) => {
    const existingColor = existingColors[key];
    if (existingColor && lineColors.includes(existingColor) && !usedColors.has(existingColor)) {
      colorsByKey[key] = existingColor;
      usedColors.add(existingColor);
    }
  });

  keys.forEach((key) => {
    if (colorsByKey[key]) {
      return;
    }
    const preferredColorIndex = lineColors.indexOf(getStableRocColor(key));
    for (let offset = 0; offset < lineColors.length; offset++) {
      const candidate = lineColors[(preferredColorIndex + offset) % lineColors.length];
      if (!usedColors.has(candidate)) {
        colorsByKey[key] = candidate;
        usedColors.add(candidate);
        return;
      }
    }
  });

  return colorsByKey;
}

export const TEST_ONLY = {
  allocateRocColors,
  buildComparisonClassificationVisualizations,
  buildRocComparisonEntries,
  getStableRocColor,
  limitRocSelection,
};

export default RuntimeArtifactComparison;
