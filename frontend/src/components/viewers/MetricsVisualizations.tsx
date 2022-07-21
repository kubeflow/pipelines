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

import HelpIcon from '@material-ui/icons/Help';
import React, { useRef, useState } from 'react';
import { useQuery } from 'react-query';
import { Array as ArrayRunType, Failure, Number, Record, String, ValidationError } from 'runtypes';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import { color, commonCss, padding } from 'src/Css';
import { Apis, ListRequest } from 'src/lib/Apis';
import { OutputArtifactLoader } from 'src/lib/OutputArtifactLoader';
import WorkflowParser, { StoragePath } from 'src/lib/WorkflowParser';
import { getMetadataValue } from 'src/mlmd/library';
import {
  filterArtifactsByType,
  filterLinkedArtifactsByType,
  getArtifactName,
  LinkedArtifact,
} from 'src/mlmd/MlmdUtils';
import { Artifact, ArtifactType, Execution } from 'src/third_party/mlmd';
import Banner from '../Banner';
import CustomTable, {
  Column,
  CustomRendererProps,
  Row as TableRow,
} from 'src/components/CustomTable';
import PlotCard from '../PlotCard';
import ConfusionMatrix, { ConfusionMatrixConfig } from './ConfusionMatrix';
import { HTMLViewerConfig } from './HTMLViewer';
import { MarkdownViewerConfig } from './MarkdownViewer';
import PagedTable from './PagedTable';
import ROCCurve, { ROCCurveConfig } from './ROCCurve';
import { PlotType, ViewerConfig } from './Viewer';
import { componentMap } from './ViewerContainer';
import Tooltip from '@material-ui/core/Tooltip';
import { Link } from 'react-router-dom';
import { RoutePage, RouteParams } from 'src/components/Router';
import { RunArtifact } from 'src/pages/CompareV2';

interface MetricsVisualizationsProps {
  linkedArtifacts: LinkedArtifact[];
  artifactTypes: ArtifactType[];
  execution: Execution;
  namespace: string | undefined;
}

/**
 * Visualize system metrics based on artifact input. There can be multiple artifacts
 * and multiple visualizations associated with one artifact.
 */
export function MetricsVisualizations({
  linkedArtifacts,
  artifactTypes,
  execution,
  namespace,
}: MetricsVisualizationsProps) {
  // There can be multiple system.ClassificationMetrics or system.Metrics artifacts per execution.
  // Get scalar metrics, confidenceMetrics and confusionMatrix from artifact.
  // If there is no available metrics, show banner to notify users.
  // Otherwise, Visualize all available metrics per artifact.
  const artifacts = linkedArtifacts.map(x => x.artifact);
  const classificationMetricsArtifacts = getVerifiedClassificationMetricsArtifacts(
    linkedArtifacts,
    artifactTypes,
  );
  const metricsArtifacts = getVerifiedMetricsArtifacts(artifacts, artifactTypes);
  const htmlArtifacts = getVertifiedHtmlArtifacts(linkedArtifacts, artifactTypes);
  const mdArtifacts = getVertifiedMarkdownArtifacts(linkedArtifacts, artifactTypes);
  const v1VisualizationArtifact = getV1VisualizationArtifacts(linkedArtifacts, artifactTypes);

  const {
    isSuccess: isV1ViewerConfigsSuccess,
    error: v1ViewerConfigError,
    data: v1ViewerConfigs,
  } = useQuery<ViewerConfig[], Error>(
    [
      'viewconfig',
      {
        artifact: v1VisualizationArtifact?.artifact.getId(),
        state: execution.getLastKnownState(),
        namespace: namespace,
      },
    ],
    () => getViewConfig(v1VisualizationArtifact, namespace),
    { staleTime: Infinity },
  );

  const { isSuccess: isHtmlDownloaded, error: htmlError, data: htmlViewerConfigs } = useQuery<
    HTMLViewerConfig[],
    Error
  >(
    [
      'htmlViewerConfig',
      {
        artifacts: htmlArtifacts.map(linkedArtifact => {
          return linkedArtifact.artifact.getId();
        }),
        state: execution.getLastKnownState(),
        namespace: namespace,
      },
    ],
    () => getHtmlViewerConfig(htmlArtifacts, namespace),
    { staleTime: Infinity },
  );

  const {
    isSuccess: isMarkdownDownloaded,
    error: markdownError,
    data: markdownViewerConfigs,
  } = useQuery<MarkdownViewerConfig[], Error>(
    [
      'markdownViewerConfig',
      {
        artifacts: mdArtifacts.map(linkedArtifact => {
          return linkedArtifact.artifact.getId();
        }),
        state: execution.getLastKnownState(),
        namespace: namespace,
      },
    ],
    () => getMarkdownViewerConfig(mdArtifacts, namespace),
    { staleTime: Infinity },
  );

  if (
    classificationMetricsArtifacts.length === 0 &&
    metricsArtifacts.length === 0 &&
    htmlArtifacts.length === 0 &&
    mdArtifacts.length === 0 &&
    !v1VisualizationArtifact
  ) {
    return <Banner message='There is no metrics artifact available in this step.' mode='info' />;
  }

  return (
    <>
      {/* Shows first encountered issue on Banner */}

      {(() => {
        if (v1ViewerConfigError) {
          return (
            <Banner
              message='Error in retrieving v1 metrics information.'
              mode='error'
              additionalInfo={v1ViewerConfigError.message}
            />
          );
        }
        if (htmlError) {
          return (
            <Banner
              message='Error in retrieving HTML visualization information.'
              mode='error'
              additionalInfo={htmlError.message}
            />
          );
        }
        if (markdownError) {
          return (
            <Banner
              message='Error in retrieving Markdown visualization information.'
              mode='error'
              additionalInfo={markdownError.message}
            />
          );
        }
        return null;
      })()}

      {/* Shows visualizations of all kinds */}
      {classificationMetricsArtifacts.map(linkedArtifact => {
        return (
          <React.Fragment key={linkedArtifact.artifact.getId()}>
            <ConfidenceMetricsSection linkedArtifacts={[linkedArtifact]} />
            <ConfusionMatrixSection artifact={linkedArtifact.artifact} />
          </React.Fragment>
        );
      })}
      {metricsArtifacts.map(artifact => (
        <ScalarMetricsSection artifact={artifact} key={artifact.getId()} />
      ))}
      {isHtmlDownloaded && htmlViewerConfigs && (
        <div key={'html'} className={padding(20, 'lrt')}>
          <PlotCard configs={htmlViewerConfigs} title={'Static HTML'} />
        </div>
      )}
      {isMarkdownDownloaded && markdownViewerConfigs && (
        <div key={'markdown'} className={padding(20, 'lrt')}>
          <PlotCard configs={markdownViewerConfigs} title={'Static Markdown'} />
        </div>
      )}
      {isV1ViewerConfigsSuccess &&
        v1ViewerConfigs &&
        v1ViewerConfigs.map((config, i) => {
          const title = componentMap[config.type].prototype.getDisplayName();
          return (
            <div key={i} className={padding(20, 'lrt')}>
              <PlotCard configs={[config]} title={title} />
            </div>
          );
        })}
    </>
  );
}

function getVerifiedClassificationMetricsArtifacts(
  linkedArtifacts: LinkedArtifact[],
  artifactTypes: ArtifactType[],
): LinkedArtifact[] {
  if (!linkedArtifacts || !artifactTypes) {
    return [];
  }
  // Reference: https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py#L124
  // system.ClassificationMetrics contains confusionMatrix or confidenceMetrics.
  const classificationMetricsArtifacts = filterLinkedArtifactsByType(
    'system.ClassificationMetrics',
    artifactTypes,
    linkedArtifacts,
  );

  return classificationMetricsArtifacts
    .map(linkedArtifact => ({
      name: linkedArtifact.artifact
        .getCustomPropertiesMap()
        .get('display_name')
        ?.getStringValue(),
      customProperties: linkedArtifact.artifact.getCustomPropertiesMap(),
      linkedArtifact: linkedArtifact,
    }))
    .filter(x => !!x.name)
    .filter(x => {
      const confidenceMetrics = x.customProperties
        .get('confidenceMetrics')
        ?.getStructValue()
        ?.toJavaScript();

      const confusionMatrix = x.customProperties
        .get('confusionMatrix')
        ?.getStructValue()
        ?.toJavaScript();
      return !!confidenceMetrics || !!confusionMatrix;
    })
    .map(x => x.linkedArtifact);
}

function getVerifiedMetricsArtifacts(
  artifacts: Artifact[],
  artifactTypes: ArtifactType[],
): Artifact[] {
  if (!artifacts || !artifactTypes) {
    return [];
  }
  // Reference: https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py#L104
  // system.Metrics contains scalar metrics.
  const metricsArtifacts = filterArtifactsByType('system.Metrics', artifactTypes, artifacts);

  return metricsArtifacts.filter(x =>
    x
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue(),
  );
}

function getVertifiedHtmlArtifacts(
  linkedArtifacts: LinkedArtifact[],
  artifactTypes: ArtifactType[],
): LinkedArtifact[] {
  if (!linkedArtifacts || !artifactTypes) {
    return [];
  }
  const htmlArtifacts = filterLinkedArtifactsByType('system.HTML', artifactTypes, linkedArtifacts);

  return htmlArtifacts.filter(x =>
    x.artifact
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue(),
  );
}

function getVertifiedMarkdownArtifacts(
  linkedArtifacts: LinkedArtifact[],
  artifactTypes: ArtifactType[],
): LinkedArtifact[] {
  if (!linkedArtifacts || !artifactTypes) {
    return [];
  }
  const htmlArtifacts = filterLinkedArtifactsByType(
    'system.Markdown',
    artifactTypes,
    linkedArtifacts,
  );

  return htmlArtifacts.filter(x =>
    x.artifact
      .getCustomPropertiesMap()
      .get('display_name')
      ?.getStringValue(),
  );
}

function getV1VisualizationArtifacts(
  linkedArtifacts: LinkedArtifact[],
  artifactTypes: ArtifactType[],
): LinkedArtifact | undefined {
  const systemArtifacts = filterLinkedArtifactsByType(
    'system.Artifact',
    artifactTypes,
    linkedArtifacts,
  );

  const v1VisualizationArtifacts = systemArtifacts.filter(x => {
    if (!x) {
      return false;
    }
    const artifactName = getArtifactName(x);
    // This is a hack to find mlpipeline-ui-metadata artifact for visualization.
    const updatedName = artifactName?.replace(/[\W_]/g, '-').toLowerCase();
    return updatedName === 'mlpipeline-ui-metadata';
  });

  if (v1VisualizationArtifacts.length > 1) {
    throw new Error(
      'There are more than 1 mlpipeline-ui-metadata artifact: ' +
        JSON.stringify(v1VisualizationArtifacts),
    );
  }
  return v1VisualizationArtifacts.length === 0 ? undefined : v1VisualizationArtifacts[0];
}

const ROC_CURVE_DEFINITION =
  'The receiver operating characteristic (ROC) curve shows the trade-off between true positive rate and false positive rate. ' +
  'A lower threshold results in a higher true positive rate (and a higher false positive rate), ' +
  'while a higher threshold results in a lower true positive rate (and a lower false positive rate)';

type ConfidenceMetric = {
  confidenceThreshold: string;
  falsePositiveRate: number;
  recall: number;
};

interface ConfidenceMetricsFilter {
  runArtifacts: RunArtifact[];
  selectedIds: string[];
  setSelectedIds: (selectedIds: string[]) => void;
}

interface ConfidenceMetricsSectionProps {
  linkedArtifacts: LinkedArtifact[];
  filter?: ConfidenceMetricsFilter;
}

const runNameCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return (
    <Tooltip title={props.value || ''} enterDelay={300} placement='top-start'>
      <Link
        className={commonCss.link}
        onClick={e => e.stopPropagation()}
        to={RoutePage.RUN_DETAILS.replace(':' + RouteParams.runId, props.id)}
      >
        {props.value}
      </Link>
    </Tooltip>
  );
};

const executionArtifactCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return (
    <Tooltip title={props.value || ''} enterDelay={300} placement='top-start'>
      <p>{props.value || ''}</p>
    </Tooltip>
  );
};

const curveLegendCustomRenderer: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return (
    <div
      style={{
        width: '2rem',
        height: '4px',
        backgroundColor: props.value,
      }}
    />
  );
};

function reload(request: ListRequest): Promise<string> {
  // TODO: Consider making an Api method for returning and caching types
  return Promise.resolve('');
}

// TODO: Can we place this somewhere else?
export const lineColors = [
  '#4285f4',
  '#efb4a3',
  '#684e91',
  '#d74419',
  '#7fa6c4',
  '#ffdc10',
  '#d7194d',
  '#6b2f49',
  '#f9e27c',
  '#633a70',
  '#5ec4ec',
];

const lineColorsStack = [...lineColors];

export const getRocCurveId = (linkedArtifact: LinkedArtifact): string =>
  `${linkedArtifact.event.getExecutionId()}-${linkedArtifact.event.getArtifactId()}`;

export function ConfidenceMetricsSection({
  linkedArtifacts,
  filter,
}: ConfidenceMetricsSectionProps) {
  const tableRef = useRef<CustomTable>(null); // TODO: Add refresh line.
  const [selectedIdColorMap, setSelectedIdColorMap] = useState<{[key: string]: string }>({});
  let confidenceMetricsDataList = linkedArtifacts
    .map(linkedArtifact => {
      const artifact = linkedArtifact.artifact;
      const customProperties = artifact.getCustomPropertiesMap();
      return {
        confidenceMetrics: customProperties
          .get('confidenceMetrics')
          ?.getStructValue()
          ?.toJavaScript(),
        name:
          customProperties.get('display_name')?.getStringValue() ||
          `artifact ID #${artifact.getId().toString()}`,
        id: getRocCurveId(linkedArtifact),
      };
    })
    .filter(confidenceMetricsData => confidenceMetricsData.confidenceMetrics);

  /*
    TODO(zpChris): This does not quite work.
    I need to keep the colors corresponding to each equal, and this only returns null if there are 
    no roc curves to display at all (not just that are not selected).
    I think I may need two separate pieces of logic for this.
  */
  if (confidenceMetricsDataList.length === 0) {
    // TODO: Should I display a note that there are no ROC Curves to display?
    return null;
  }

  let columns: Column[] = [];
  const rows: TableRow[] = [];
  if (filter) {
    confidenceMetricsDataList = confidenceMetricsDataList.filter(confidenceMetricsData => filter.selectedIds.includes(confidenceMetricsData.id));
    columns = [
      {
        customRenderer: executionArtifactCustomRenderer,
        flex: 1.5,
        label: 'Execution name > Run name',
      },
      // {
      //   customRenderer: runNameCustomRenderer,
      //   flex: 1,
      //   label: 'Run name',
      //   sortKey: RunSortKeys.NAME,
      // },
      { customRenderer: curveLegendCustomRenderer, flex: 0.5, label: 'Curve legend' },
    ];

    if (filter.selectedIds.length > 0 && Object.keys(selectedIdColorMap).length === 0) {
      filter.selectedIds.forEach(selectedId => {
        selectedIdColorMap[selectedId] = lineColorsStack.pop() || '';
      });
      setSelectedIdColorMap(selectedIdColorMap);
    }

    // I don't know how to get the correct line color. Or have a line color for each one.
    for (let i = 0; i < linkedArtifacts.length; i++) {
      const artifact = linkedArtifacts[i].artifact;
      const id = getRocCurveId(linkedArtifacts[i]);
      console.log(selectedIdColorMap);
      const row = {
        id,
        otherFields: [
          artifact
            .getCustomPropertiesMap()
            ?.get('display_name')
            ?.getStringValue() || '-',
            selectedIdColorMap[id],
        ] as any,
      };
      rows.push(row);
    }
  }

  const rocCurveConfigs: ROCCurveConfig[] = [];
  for (let i = 0; i < confidenceMetricsDataList.length; i++) {
    const confidenceMetrics = confidenceMetricsDataList[i].confidenceMetrics as any;
    const { error } = validateConfidenceMetrics(confidenceMetrics.list);

    // If an error exists with confidence metrics, return the first one with an issue.
    if (error) {
      const errorMsg =
        'Error in ' +
        confidenceMetricsDataList[i].name +
        " artifact's confidenceMetrics data format.";
      return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
    }

    rocCurveConfigs.push(buildRocCurveConfig(confidenceMetrics.list));
  }

  const updateSelection = (newSelectedIds: string[]): void => {
    if (filter) {
      newSelectedIds = newSelectedIds.slice(0, 10);
      const { selectedIds: oldSelectedIds, setSelectedIds } = filter;
      
      // Convert arrays to sets for quick lookup.
      const newSelectedIdsSet = new Set(newSelectedIds);
      const oldSelectedIdsSet = new Set(oldSelectedIds);

      // 2*O(n^2). This could actually be really large, conceivably.
      const addedIds = newSelectedIds.filter(selectedId => !oldSelectedIdsSet.has(selectedId));
      const removedIds = oldSelectedIds.filter(selectedId => !newSelectedIdsSet.has(selectedId));

      // I'll need this anyway to potentially limit how many new ones are selected.

      // Ok, I think I'll use a map here and pass it through filter.

      console.log('---------');
      console.log(addedIds);
      console.log(removedIds);
      console.log(lineColorsStack);

      setSelectedIds(newSelectedIds);
      removedIds.forEach(removedId => {
        lineColorsStack.push(selectedIdColorMap[removedId]);
        selectedIdColorMap[removedId] = '';
      });

      // Place a limit on the number of IDs that can be added.
      addedIds.forEach(addedId => {
        selectedIdColorMap[addedId] = lineColorsStack.pop() || '';
      });

      setSelectedIdColorMap(selectedIdColorMap);

      console.log(lineColorsStack);

      // Huh, I wonder if all of the new ones are added at the end. Maybe, but I don't want this to rely on the internal implementation of custom table.
      // Right now, could use the invariant that you can only remove one at a time (not true, remove all). However, I don't want to rely on that, in case the implementation changes for some reason.

      // It's better to have a separate, which represents removed. And going through twice? Lookup is immediate. So 2*O(n). No, that doesn't work either. If the IDs were tied to color, that'd be easiest.
      // I could use a set.
      // Using a conversion to Set, this becomes O(n). 4*O(n), specifically.
    }
    // Comparison between all the current selectedIds and the new selectedIds.
    /*
      - Find the change.
        - Do this by looping through new selectedIds? There's gotta be a faster way for this. Cause also have to loop through current.
        - Can use filter between the two.
        - Are the selectedIds order between the two? I can step through on each.
        - I suppose a better solution is to have a full map. With all possibilities.
        - Then that requires looping through all on each though.
        - Maybe just iterate through each array and use that to determine which should be removed or not.
        - So which are removed first (new array, what is not present).
        - But then no, I need 
    */
  }

  const colors: string[] = filter ? filter.selectedIds.map(selectedId => selectedIdColorMap[selectedId]) : [];
  return (
    <div className={padding(40, 'lrt')}>
      <div className={padding(40, 'b')}>
        <h3>
          {'ROC Curve: ' +
            (confidenceMetricsDataList.length === 1
              ? confidenceMetricsDataList[0].name
              : 'multiple artifacts')}{' '}
          <IconWithTooltip
            Icon={HelpIcon}
            iconColor={color.weak}
            tooltip={ROC_CURVE_DEFINITION}
          ></IconWithTooltip>
        </h3>
      </div>
      {/* TODO(zpChris): Introduce checkbox system that matches artifacts to curves. */}
      <ROCCurve configs={rocCurveConfigs} colors={colors} />
      {filter && (
        <>
          {filter.selectedIds.length === 10 ? <p>You have reached the maximum number of ROC Curves you can select at once. Deselect an item in order to select additional artifacts</p> : null}
          <CustomTable
            columns={columns}
            rows={rows}
            selectedIds={filter.selectedIds}
            // initialSortColumn={RunSortKeys.CREATED_AT}
            ref={tableRef}
            filterLabel='Filter artifacts'
            updateSelection={updateSelection}
            reload={reload}
            disablePaging={false}
            disableSorting={false}
            disableSelection={true}
            noFilterBox={false}
            emptyMessage='No artifacts found'
          />
        </>
      )}
    </div>
  );
}

const ConfidenceMetricRunType = Record({
  confidenceThreshold: Number,
  falsePositiveRate: Number,
  recall: Number,
});
const ConfidenceMetricArrayRunType = ArrayRunType(ConfidenceMetricRunType);
function validateConfidenceMetrics(inputs: any): { error?: string } {
  try {
    ConfidenceMetricArrayRunType.check(inputs);
  } catch (e) {
    if (e instanceof ValidationError) {
      return { error: e.message + '. Data: ' + JSON.stringify(inputs) };
    }
  }
  return {};
}

function buildRocCurveConfig(confidenceMetricsArray: ConfidenceMetric[]): ROCCurveConfig {
  const arraytypesCheck = ConfidenceMetricArrayRunType.check(confidenceMetricsArray);
  return {
    type: PlotType.ROC,
    data: arraytypesCheck.map(metric => ({
      label: (metric.confidenceThreshold as unknown) as string,
      x: metric.falsePositiveRate,
      y: metric.recall,
    })),
  };
}

type AnnotationSpec = {
  displayName: string;
};
type Row = {
  row: number[];
};
type ConfusionMatrixInput = {
  annotationSpecs: AnnotationSpec[];
  rows: Row[];
};

interface ConfusionMatrixProps {
  artifact: Artifact;
}

const CONFUSION_MATRIX_DEFINITION =
  'The number of correct and incorrect predictions are ' +
  'summarized with count values and broken down by each class. ' +
  'The higher value on cell where Predicted label matches True label, ' +
  'the better prediction performance of this model is.';

export function ConfusionMatrixSection({ artifact }: ConfusionMatrixProps) {
  const customProperties = artifact.getCustomPropertiesMap();
  const name = customProperties.get('display_name')?.getStringValue();

  const confusionMatrix = customProperties
    .get('confusionMatrix')
    ?.getStructValue()
    ?.toJavaScript();
  if (confusionMatrix === undefined) {
    return null;
  }

  const { error } = validateConfusionMatrix(confusionMatrix.struct as any);

  if (error) {
    const errorMsg = 'Error in ' + name + " artifact's confusionMatrix data format.";
    return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
  }
  return (
    <div className={padding(40)}>
      <div className={padding(40, 'b')}>
        <h3>
          {'Confusion Matrix: ' + name}{' '}
          <IconWithTooltip
            Icon={HelpIcon}
            iconColor={color.weak}
            tooltip={CONFUSION_MATRIX_DEFINITION}
          ></IconWithTooltip>
        </h3>
      </div>
      <ConfusionMatrix configs={buildConfusionMatrixConfig(confusionMatrix.struct as any)} />
    </div>
  );
}

const ConfusionMatrixInputRunType = Record({
  annotationSpecs: ArrayRunType(
    Record({
      displayName: String,
    }),
  ),
  rows: ArrayRunType(Record({ row: ArrayRunType(Number) })),
});
function validateConfusionMatrix(input: any): { error?: string } {
  if (!input) return { error: 'confusionMatrix does not exist.' };
  try {
    const matrix = ConfusionMatrixInputRunType.check(input);
    const height = matrix.rows.length;
    const annotationLen = matrix.annotationSpecs.length;
    if (annotationLen !== height) {
      throw new ValidationError({
        message:
          'annotationSpecs has different length ' + annotationLen + ' than rows length ' + height,
      } as Failure);
    }
    for (let x of matrix.rows) {
      if (x.row.length !== height)
        throw new ValidationError({
          message: 'row: ' + JSON.stringify(x) + ' has different length of columns from rows.',
        } as Failure);
    }
  } catch (e) {
    if (e instanceof ValidationError) {
      return { error: e.message + '. Data: ' + JSON.stringify(input) };
    }
  }
  return {};
}

function buildConfusionMatrixConfig(
  confusionMatrix: ConfusionMatrixInput,
): ConfusionMatrixConfig[] {
  return [
    {
      type: PlotType.CONFUSION_MATRIX,
      axes: ['True label', 'Predicted label'],
      labels: confusionMatrix.annotationSpecs.map(annotation => annotation.displayName),
      data: confusionMatrix.rows.map(x => x.row),
    },
  ];
}

interface ScalarMetricsSectionProps {
  artifact: Artifact;
}
function ScalarMetricsSection({ artifact }: ScalarMetricsSectionProps) {
  const customProperties = artifact.getCustomPropertiesMap();
  const name = customProperties.get('display_name')?.getStringValue();
  const data = customProperties
    .getEntryList()
    .map(([key]) => ({
      key,
      value: JSON.stringify(getMetadataValue(customProperties.get(key))),
    }))
    .filter(metric => metric.key !== 'display_name');

  if (data.length === 0) {
    return null;
  }
  return (
    <div className={padding(40, 'lrt')}>
      <div className={padding(40, 'b')}>
        <h3>{'Scalar Metrics: ' + name}</h3>
      </div>
      <PagedTable
        configs={[
          {
            data: data.map(d => [d.key, d.value]),
            labels: ['name', 'value'],
            type: PlotType.TABLE,
          },
        ]}
      />
    </div>
  );
}

async function getViewConfig(
  v1VisualizationArtifact: LinkedArtifact | undefined,
  namespace: string | undefined,
): Promise<ViewerConfig[]> {
  if (v1VisualizationArtifact) {
    return OutputArtifactLoader.load(
      WorkflowParser.parseStoragePath(v1VisualizationArtifact.artifact.getUri()),
      namespace,
    );
  }
  return [];
}

export async function getHtmlViewerConfig(
  htmlArtifacts: LinkedArtifact[] | undefined,
  namespace: string | undefined,
): Promise<HTMLViewerConfig[]> {
  if (!htmlArtifacts) {
    return [];
  }
  const htmlViewerConfigs = htmlArtifacts.map(async linkedArtifact => {
    const uri = linkedArtifact.artifact.getUri();
    let storagePath: StoragePath | undefined;
    if (!uri) {
      throw new Error('HTML Artifact URI unknown');
    }

    storagePath = WorkflowParser.parseStoragePath(uri);
    if (!storagePath) {
      throw new Error('HTML Artifact storagePath unknown');
    }

    // TODO(zijianjoy): Limit the size of HTML file fetching to prevent UI frozen.
    let data = await Apis.readFile(storagePath, namespace);
    return { htmlContent: data, type: PlotType.WEB_APP } as HTMLViewerConfig;
  });
  return Promise.all(htmlViewerConfigs);
}

export async function getMarkdownViewerConfig(
  markdownArtifacts: LinkedArtifact[] | undefined,
  namespace: string | undefined,
): Promise<MarkdownViewerConfig[]> {
  if (!markdownArtifacts) {
    return [];
  }
  const markdownViewerConfigs = markdownArtifacts.map(async linkedArtifact => {
    const uri = linkedArtifact.artifact.getUri();
    let storagePath: StoragePath | undefined;
    if (!uri) {
      throw new Error('Markdown Artifact URI unknown');
    }

    storagePath = WorkflowParser.parseStoragePath(uri);
    if (!storagePath) {
      throw new Error('Markdown Artifact storagePath unknown');
    }

    // TODO(zijianjoy): Limit the size of Markdown file fetching to prevent UI frozen.
    let data = await Apis.readFile(storagePath, namespace);
    return { markdownContent: data, type: PlotType.MARKDOWN } as MarkdownViewerConfig;
  });
  return Promise.all(markdownViewerConfigs);
}
