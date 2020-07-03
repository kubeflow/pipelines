/*
 * Copyright 2018 Google LLC
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

import {
  Api,
  Artifact,
  ArtifactType,
  Event,
  Execution,
  GetArtifactsByIDRequest,
  GetArtifactsByIDResponse,
  GetArtifactTypesRequest,
  GetArtifactTypesResponse,
  GetEventsByExecutionIDsRequest,
  GetEventsByExecutionIDsResponse,
} from '@kubeflow/frontend';
import { csvParseRows } from 'd3-dsv';
import { ApiVisualization, ApiVisualizationType } from '../apis/visualization';
import { ConfusionMatrixConfig } from '../components/viewers/ConfusionMatrix';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';
import { MarkdownViewerConfig } from '../components/viewers/MarkdownViewer';
import { PagedTableConfig } from '../components/viewers/PagedTable';
import { ROCCurveConfig } from '../components/viewers/ROCCurve';
import { TensorboardViewerConfig } from '../components/viewers/Tensorboard';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { Apis } from '../lib/Apis';
import { errorToMessage, logger } from './Utils';
import WorkflowParser, { StoragePath } from './WorkflowParser';

export interface PlotMetadata {
  format?: 'csv';
  header?: string[];
  labels?: string[];
  predicted_col?: string;
  schema?: Array<{ type: string; name: string }>;
  source: string;
  storage?: 'gcs' | 'inline';
  target_col?: string;
  type: PlotType;
}

type PlotMetadataContent = Omit<PlotMetadata, 'type'>;

export interface OutputMetadata {
  outputs: PlotMetadata[];
}

type SourceContentGetter = (source: string, storage?: PlotMetadata['storage']) => Promise<string>;

export class OutputArtifactLoader {
  public static async load(outputPath: StoragePath, namespace?: string): Promise<ViewerConfig[]> {
    let plotMetadataList: PlotMetadata[] = [];
    try {
      const metadataFile = await Apis.readFile(outputPath, namespace);
      if (metadataFile) {
        try {
          plotMetadataList = (JSON.parse(metadataFile) as OutputMetadata).outputs;
          if (plotMetadataList === undefined) {
            throw new Error('"outputs" field required by not found on metadata file');
          }
        } catch (e) {
          logger.error(`Could not parse metadata file at: ${outputPath.key}. Error: ${e}`);
          return [];
        }
      }
    } catch (err) {
      const errorMessage = await errorToMessage(err);
      logger.error('Error loading run outputs:', errorMessage);
      // TODO: error dialog
    }

    const getSourceContent: SourceContentGetter = async (source, storage) =>
      await readSourceContent(source, storage, namespace);

    const configs: Array<ViewerConfig | null> = await Promise.all(
      plotMetadataList.map(async metadata => {
        switch (metadata.type) {
          case PlotType.CONFUSION_MATRIX:
            return await this.buildConfusionMatrixConfig(metadata, getSourceContent);
          case PlotType.MARKDOWN:
            return await this.buildMarkdownViewerConfig(metadata, getSourceContent);
          case PlotType.TABLE:
            return await this.buildPagedTableConfig(metadata, getSourceContent);
          case PlotType.TENSORBOARD:
            return await this.buildTensorboardConfig(metadata, namespace);
          case PlotType.WEB_APP:
            return await this.buildHtmlViewerConfig(metadata, getSourceContent);
          case PlotType.ROC:
            return await this.buildRocCurveConfig(metadata, getSourceContent);
          default:
            logger.error('Unknown plot type: ' + metadata.type);
            return null;
        }
      }),
    );

    return configs.filter(c => !!c) as ViewerConfig[];
  }

  public static async buildConfusionMatrixConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<ConfusionMatrixConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.labels) {
      throw new Error('Malformed metadata, property "labels" is required.');
    }
    if (!metadata.schema) {
      throw new Error('Malformed metadata, property "schema" missing.');
    }
    if (!Array.isArray(metadata.schema)) {
      throw new Error('"schema" must be an array of {"name": string, "type": string} objects');
    }

    const content = await getSourceContent(metadata.source, metadata.storage);
    const csvRows = csvParseRows(content.trim());
    const labels = metadata.labels;
    const labelIndex: { [label: string]: number } = {};
    let index = 0;
    labels.forEach(l => {
      labelIndex[l] = index++;
    });

    if (labels.length ** 2 !== csvRows.length) {
      throw new Error(
        `Data dimensions ${csvRows.length} do not match the number of labels passed ${labels.length}`,
      );
    }

    const data = Array.from(Array(labels.length), () => new Array(labels.length));
    csvRows.forEach(([labelX, labelY, count]) => {
      const i = labelIndex[labelX.trim()];
      const j = labelIndex[labelY.trim()];
      // Note: data[i][j] means data(i, j) i on x-axis, j on y-axis
      data[i][j] = Number.parseInt(count, 10);
    });

    const columnNames = metadata.schema.map(r => {
      if (!r.name) {
        throw new Error('Each item in the "schema" array must contain a "name" field');
      }
      return r.name;
    });
    const axes = [columnNames[0], columnNames[1]];

    return {
      axes,
      data,
      labels,
      type: PlotType.CONFUSION_MATRIX,
    };
  }

  public static async buildPagedTableConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<PagedTableConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.header) {
      throw new Error('Malformed metadata, property "header" is required.');
    }
    if (!metadata.format) {
      throw new Error('Malformed metadata, property "format" is required.');
    }
    let data: string[][] = [];
    const labels = metadata.header || [];
    const content = await getSourceContent(metadata.source, metadata.storage);

    switch (metadata.format) {
      case 'csv':
        data = csvParseRows(content.trim()).map(r => r.map(c => c.trim()));
        break;
      default:
        throw new Error('Unsupported table format: ' + metadata.format);
    }

    return {
      data,
      labels,
      type: PlotType.TABLE,
    };
  }

  public static async buildTensorboardConfig(
    metadata: PlotMetadataContent,
    namespace?: string,
  ): Promise<TensorboardViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!namespace) {
      throw new Error('Namespace is required.');
    }
    WorkflowParser.parseStoragePath(metadata.source);
    return {
      type: PlotType.TENSORBOARD,
      url: metadata.source,
      namespace,
    };
  }

  public static async buildHtmlViewerConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<HTMLViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    return {
      htmlContent: await getSourceContent(metadata.source, metadata.storage),
      type: PlotType.WEB_APP,
    };
  }

  /**
   * @param reportProgress callback to report load progress, accepts [0, 100]
   * @throws error on exceptions
   * @returns config array, also returns empty array when expected erros happen
   */
  public static async buildTFXArtifactViewerConfig({
    namespace,
    execution,
    reportProgress = () => null,
  }: {
    namespace: string;
    execution: Execution;
    reportProgress: (progress: number) => void;
  }): Promise<HTMLViewerConfig[]> {
    // Error handling assumptions:
    // * Context/execution/artifact nodes are not expected to be in MLMD. Thus, any
    // errors associated with the nodes not being found are expected.
    // * RPC errors to MLMD are unexpected.
    // * Being unable to find an execution node with a matching argoPodName is expected, as this should only work on TFX >= 0.21.
    // * Once we have URIs for artifacts that we want to display, any errors displaying them are unexpected.
    //
    // With that in mind, buildTFXArtifactViewerConfig() returns an empty list for expected errors,
    // and throws/forwards for unexpected errors.

    // Since artifact types don't change per run, this can be optimized further so
    // that we don't fetch them on every page load.
    reportProgress(10);
    const [artifactTypes, artifacts] = await Promise.all([
      getArtifactTypes(),
      getOutputArtifactsInExecution(execution),
    ]);
    if (artifactTypes.length === 0 || artifacts.length === 0) {
      // There are no artifact types data or no artifacts.
      return [];
    }
    reportProgress(70);

    // TODO: Visualize non-TFDV artifacts, such as ModelEvaluation using TFMA
    let viewers: Array<Promise<HTMLViewerConfig>> = [];
    const exampleStatisticsArtifactUris = filterArtifactUrisByType(
      'ExampleStatistics',
      artifactTypes,
      artifacts,
    );
    exampleStatisticsArtifactUris.forEach(uri => {
      const evalUri = uri + '/eval/stats_tfrecord';
      const trainUri = uri + '/train/stats_tfrecord';
      viewers = viewers.concat(
        [evalUri, trainUri].map(async specificUri => {
          return buildArtifactViewerTfdvStatistics(specificUri, namespace);
        }),
      );
    });
    const schemaGenArtifactUris = filterArtifactUrisByType('Schema', artifactTypes, artifacts);
    viewers = viewers.concat(
      schemaGenArtifactUris.map(uri => {
        uri = uri + '/schema.pbtxt';
        const script = [
          'import tensorflow_data_validation as tfdv',
          `schema = tfdv.load_schema_text('${uri}')`,
          'tfdv.display_schema(schema)',
        ];
        return buildArtifactViewer({ script, namespace });
      }),
    );
    const anomaliesArtifactUris = filterArtifactUrisByType(
      'ExampleAnomalies',
      artifactTypes,
      artifacts,
    );
    viewers = viewers.concat(
      anomaliesArtifactUris.map(uri => {
        uri = uri + '/anomalies.pbtxt';
        const script = [
          'import tensorflow_data_validation as tfdv',
          `anomalies = tfdv.load_anomalies_text('${uri}')`,
          'tfdv.display_anomalies(anomalies)',
        ];
        return buildArtifactViewer({ script, namespace });
      }),
    );
    const EvaluatorArtifactUris = filterArtifactUrisByType(
      'ModelEvaluation',
      artifactTypes,
      artifacts,
    );
    viewers = viewers.concat(
      EvaluatorArtifactUris.map(uri => {
        const configFilePath = uri + '/eval_config.json';
        // The visualization of TFMA inside KFP UI depends a hack of TFMA widget js
        // For context and future improvement, please refer to
        // https://github.com/tensorflow/model-analysis/issues/10#issuecomment-587422929
        const script = [
          `import io`,
          `import json`,
          `import tensorflow as tf`,
          `import tensorflow_model_analysis as tfma`,
          `from ipywidgets.embed import embed_minimal_html`,
          `from IPython.core.display import display, HTML`,
          `config_file=tf.io.gfile.GFile('${configFilePath}', 'r')`,
          `config=json.loads(config_file.read())`,
          `featureKeys=list(filter(lambda x: 'featureKeys' in x, config['evalConfig']['slicingSpecs']))`,
          `columns=[] if len(featureKeys) == 0 else featureKeys[0]['featureKeys']`,
          `slicing_spec = tfma.slicer.SingleSliceSpec(columns=columns)`,
          `eval_result = tfma.load_eval_result('${uri}')`,
          `slicing_metrics_view = tfma.view.render_slicing_metrics(eval_result, slicing_spec=slicing_spec)`,
          `view = io.StringIO()`,
          `embed_minimal_html(view, views=[slicing_metrics_view], title='Slicing Metrics')`,
          `display(HTML(view.getvalue()))`,
        ];
        return buildArtifactViewer({ script, namespace });
      }),
    );
    // TODO(jingzhang36): maybe move the above built-in scripts to visualization server.

    return Promise.all(viewers);
  }

  public static async buildMarkdownViewerConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<MarkdownViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    return {
      markdownContent: await getSourceContent(metadata.source, metadata.storage),
      type: PlotType.MARKDOWN,
    };
  }

  public static async buildRocCurveConfig(
    metadata: PlotMetadataContent,
    getSourceContent: SourceContentGetter,
  ): Promise<ROCCurveConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.schema) {
      throw new Error('Malformed metadata, property "schema" is required.');
    }
    if (!Array.isArray(metadata.schema)) {
      throw new Error('Malformed schema, must be an array of {"name": string, "type": string}');
    }

    const content = await getSourceContent(metadata.source, metadata.storage);
    const stringData = csvParseRows(content.trim());

    const fprIndex = metadata.schema.findIndex(field => field.name === 'fpr');
    if (fprIndex === -1) {
      throw new Error('Malformed schema, expected to find a column named "fpr"');
    }
    const tprIndex = metadata.schema.findIndex(field => field.name === 'tpr');
    if (tprIndex === -1) {
      throw new Error('Malformed schema, expected to find a column named "tpr"');
    }
    const thresholdIndex = metadata.schema.findIndex(field => field.name.startsWith('threshold'));
    if (thresholdIndex === -1) {
      throw new Error('Malformed schema, expected to find a column named "threshold"');
    }

    const dataset = stringData.map(row => ({
      label: row[thresholdIndex].trim(),
      x: +row[fprIndex],
      y: +row[tprIndex],
    }));

    return {
      data: dataset,
      type: PlotType.ROC,
    };
  }
}

/**
 * @throws error when network error or invalid data
 */
async function getOutputArtifactsInExecution(execution: Execution): Promise<Artifact[]> {
  const executionId = execution.getId();
  if (!executionId) {
    throw new Error('Execution must have an ID');
  }

  const request = new GetEventsByExecutionIDsRequest();
  request.addExecutionIds(executionId);
  let res: GetEventsByExecutionIDsResponse;
  try {
    res = await Api.getInstance().metadataStoreService.getEventsByExecutionIDs(request);
  } catch (err) {
    err.message = 'Failed to getExecutionsByExecutionIDs: ' + err.message;
    throw err;
  }

  const outputArtifactIds = res
    .getEventsList()
    .filter(event => event.getType() === Event.Type.OUTPUT && event.getArtifactId())
    .map(event => event.getArtifactId());

  const artifactsRequest = new GetArtifactsByIDRequest();
  artifactsRequest.setArtifactIdsList(outputArtifactIds);
  let artifactsRes: GetArtifactsByIDResponse;
  try {
    artifactsRes = await Api.getInstance().metadataStoreService.getArtifactsByID(artifactsRequest);
  } catch (artifactsErr) {
    artifactsErr.message = 'Failed to getArtifactsByID: ' + artifactsErr.message;
    throw artifactsErr;
  }

  return artifactsRes.getArtifactsList();
}

async function getArtifactTypes(): Promise<ArtifactType[]> {
  const request = new GetArtifactTypesRequest();
  let res: GetArtifactTypesResponse;
  try {
    res = await Api.getInstance().metadataStoreService.getArtifactTypes(request);
  } catch (err) {
    err.message = 'Failed to getArtifactTypes: ' + err.message;
    throw err;
  }
  return res.getArtifactTypesList();
}

function filterArtifactUrisByType(
  artifactTypeName: string,
  artifactTypes: ArtifactType[],
  artifacts: Artifact[],
): string[] {
  const artifactTypeIds = artifactTypes
    .filter(artifactType => artifactType.getName() === artifactTypeName)
    .map(artifactType => artifactType.getId());
  const matchingArtifacts = artifacts.filter(artifact =>
    artifactTypeIds.includes(artifact.getTypeId()),
  );

  const tfdvArtifactsPaths = matchingArtifacts
    .map(artifact => artifact.getUri())
    .filter(uri => uri); // uri not empty
  return tfdvArtifactsPaths;
}

async function buildArtifactViewer({
  script,
  namespace,
}: {
  script: string[];
  namespace: string;
}): Promise<HTMLViewerConfig> {
  const visualizationData: ApiVisualization = {
    arguments: JSON.stringify({ code: script }),
    source: '',
    type: ApiVisualizationType.CUSTOM,
  };
  const visualization = await Apis.buildPythonVisualizationConfig(visualizationData, namespace);
  if (!visualization.htmlContent) {
    // TODO: Improve error message with details.
    throw new Error('Failed to build artifact viewer');
  }
  return {
    htmlContent: visualization.htmlContent,
    type: PlotType.WEB_APP,
  };
}

async function buildArtifactViewerTfdvStatistics(
  url: string,
  namespace: string,
): Promise<HTMLViewerConfig> {
  const visualizationData: ApiVisualization = {
    source: url,
    type: ApiVisualizationType.TFDV,
  };
  const visualization = await Apis.buildPythonVisualizationConfig(visualizationData, namespace);
  if (!visualization.htmlContent) {
    throw new Error('Failed to build artifact viewer, no value in visualization.htmlContent');
  }
  return {
    htmlContent: visualization.htmlContent,
    type: PlotType.WEB_APP,
  };
}

async function readSourceContent(
  source: PlotMetadata['source'],
  storage: PlotMetadata['storage'] | undefined,
  namespace: string | undefined,
): Promise<string> {
  if (storage === 'inline') {
    return source;
  }
  return await Apis.readFile(WorkflowParser.parseStoragePath(source), namespace);
}

export const TEST_ONLY = {
  readSourceContent,
};
