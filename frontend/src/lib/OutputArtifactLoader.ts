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

import WorkflowParser, { StoragePath } from './WorkflowParser';
import { Apis } from '../lib/Apis';
import { ConfusionMatrixConfig } from '../components/viewers/ConfusionMatrix';
import { HTMLViewerConfig } from '../components/viewers/HTMLViewer';
import { MarkdownViewerConfig } from '../components/viewers/MarkdownViewer';
import { PagedTableConfig } from '../components/viewers/PagedTable';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { ROCCurveConfig } from '../components/viewers/ROCCurve';
import { TensorboardViewerConfig } from '../components/viewers/Tensorboard';
import { csvParseRows } from 'd3-dsv';
import { logger, errorToMessage } from './Utils';
import { ApiVisualization, ApiVisualizationType } from '../apis/visualization';
import {
  GetArtifactTypesRequest,
  GetArtifactsByIDRequest,
  GetContextByTypeAndNameRequest,
  GetExecutionsByContextRequest,
  GetEventsByExecutionIDsRequest,
} from '../generated/src/apis/metadata/metadata_store_service_pb';
import {
  Artifact,
  ArtifactType,
  Context,
  Event,
  Execution,
} from '../generated/src/apis/metadata/metadata_store_pb';

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

export interface OutputMetadata {
  outputs: PlotMetadata[];
}

export class OutputArtifactLoader {
  public static async load(outputPath: StoragePath): Promise<ViewerConfig[]> {
    let plotMetadataList: PlotMetadata[] = [];
    try {
      const metadataFile = await Apis.readFile(outputPath);
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

    const configs: Array<ViewerConfig | null> = await Promise.all(
      plotMetadataList.map(async metadata => {
        switch (metadata.type) {
          case PlotType.CONFUSION_MATRIX:
            return await this.buildConfusionMatrixConfig(metadata);
          case PlotType.MARKDOWN:
            return await this.buildMarkdownViewerConfig(metadata);
          case PlotType.TABLE:
            return await this.buildPagedTableConfig(metadata);
          case PlotType.TENSORBOARD:
            return await this.buildTensorboardConfig(metadata);
          case PlotType.WEB_APP:
            return await this.buildHtmlViewerConfig(metadata);
          case PlotType.ROC:
            return await this.buildRocCurveConfig(metadata);
          default:
            logger.error('Unknown plot type: ' + metadata.type);
            return null;
        }
      }),
    );
    return configs.filter(c => !!c) as ViewerConfig[];
  }

  public static async buildConfusionMatrixConfig(
    metadata: PlotMetadata,
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

    const path = WorkflowParser.parseStoragePath(metadata.source);
    const csvRows = csvParseRows((await Apis.readFile(path)).trim());
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
    csvRows.forEach(([target, predicted, count]) => {
      const i = labelIndex[target.trim()];
      const j = labelIndex[predicted.trim()];
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

  public static async buildPagedTableConfig(metadata: PlotMetadata): Promise<PagedTableConfig> {
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

    switch (metadata.format) {
      case 'csv':
        const path = WorkflowParser.parseStoragePath(metadata.source);
        data = csvParseRows((await Apis.readFile(path)).trim()).map(r => r.map(c => c.trim()));
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
    metadata: PlotMetadata,
  ): Promise<TensorboardViewerConfig | null> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    return null;
    //    WorkflowParser.parseStoragePath(metadata.source);
    //    return {
    //      type: PlotType.TENSORBOARD,
    //      url: metadata.source,
    //    };
  }

  public static async buildHtmlViewerConfig(metadata: PlotMetadata): Promise<HTMLViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    const path = WorkflowParser.parseStoragePath(metadata.source);
    const htmlContent = await Apis.readFile(path);

    return {
      htmlContent,
      type: PlotType.WEB_APP,
    };
  }

  public static async getMlmdContext(kfpPodName: string): Promise<Context> {
    if (kfpPodName.split('-').length < 3) {
      throw new Error('kfpPodName has fewer than 3 parts');
    }

    const pipelineName = kfpPodName
      .split('-')
      .slice(0, -2)
      .join('_');
    const runID = kfpPodName
      .split('-')
      .slice(0, -1)
      .join('-');
    const contextName = pipelineName + '.' + runID;

    const request = new GetContextByTypeAndNameRequest();
    request.setTypeName('run');
    request.setContextName(contextName);
    const {
      response: res,
      error: err,
    } = await Apis.getMetadataServicePromiseClient().getContextByTypeAndName(request);
    if (err) {
      throw new Error('Failed to getContextsByTypeAndName: ' + JSON.stringify(err));
    }

    const context = res && res.getContext();
    if (!context) {
      throw new Error("getContextByTypeAndName didn't have a context");
    }
    return context;
  }

  public static async getExecutionInContextWithPodName(
    kfpPodName: string,
    context: Context,
  ): Promise<Execution> {
    const contextId = context.getId();
    if (!contextId) {
      throw new Error('Context must have an ID');
    }

    const request = new GetExecutionsByContextRequest();
    request.setContextId(contextId);
    const {
      response: res,
      error: err,
    } = await Apis.getMetadataServicePromiseClient().getExecutionsByContext(request);
    if (err) {
      throw new Error('Failed to getExecutionsByContext: ' + JSON.stringify(err));
    }

    const executionList = (res && res.getExecutionsList()) || [];
    const foundExecution = executionList.find(execution => {
      const properties = execution.getPropertiesMap();
      const executionPodName = properties.get('kfp_pod_name');
      if (!executionPodName) {
        return false;
      }
      const executionState = properties.get('state');
      if (!executionState) {
        return false;
      }
      return (
        executionPodName.getStringValue() === kfpPodName &&
        executionState.getStringValue() === 'complete'
      );
    });
    if (!foundExecution) {
      logger.verbose("Couldn't find corresponding execution in context");
      throw new Error("Couldn't find corresponding execution in context");
    }
    return foundExecution;
  }

  public static async getOutputArtifactsInExecution(execution: Execution): Promise<Artifact[]> {
    const executionId = execution.getId();
    if (!executionId) {
      throw new Error('Execution must have an ID');
    }

    const request = new GetEventsByExecutionIDsRequest();
    request.addExecutionIds(executionId);
    const {
      response: res,
      error: err,
    } = await Apis.getMetadataServicePromiseClient().getEventsByExecutionIDs(request);
    if (err) {
      throw new Error('Failed to getExecutionsByExecutionIDs: ' + JSON.stringify(err));
    }

    const eventsList = (res && res.getEventsList()) || [];
    const outputArtifactIds: number[] = [];
    eventsList.forEach(event => {
      if (event.getType() === Event.Type.OUTPUT) {
        const artifactId = event.getArtifactId();
        if (artifactId) {
          outputArtifactIds.push(artifactId);
        }
      }
    });

    const artifactsRequest = new GetArtifactsByIDRequest();
    outputArtifactIds.forEach(artifactId => artifactsRequest.addArtifactIds(artifactId));
    const {
      response: artifactsRes,
      error: artifactsErr,
    } = await Apis.getMetadataServicePromiseClient().getArtifactsByID(artifactsRequest);
    if (artifactsErr) {
      throw new Error('Failed to getArtifactsByID: ' + JSON.stringify(artifactsErr));
    }

    const artifactsList = (artifactsRes && artifactsRes.getArtifactsList()) || [];
    return artifactsList;
  }

  public static async getArtifactTypes(): Promise<ArtifactType[]> {
    const request = new GetArtifactTypesRequest();
    const {
      response: res,
      error: err,
    } = await Apis.getMetadataServicePromiseClient().getArtifactTypes(request);
    if (err) {
      throw new Error('Failed to getArtifactTypes: ' + JSON.stringify(err));
    }
    const artifactTypes = (res && res.getArtifactTypesList()) || [];
    return artifactTypes;
  }

  public static filterTfdvArtifactsPaths(
    artifactTypes: ArtifactType[],
    artifacts: Artifact[],
  ): string[] {
    const tfdvArtifactTypes = artifactTypes.filter(artifactType => {
      return artifactType.getName() === 'ExampleStatistics';
      //return artifactType.getName() === 'Schema';
      //return artifactType.getName() === 'ExampleAnomalies';
    });
    const tfdvArtifactTypeIds = tfdvArtifactTypes.map(artifactType => {
      return artifactType.getId();
    });
    const tfdvArtifacts = artifacts.filter(artifact => {
      return tfdvArtifactTypeIds.includes(artifact.getTypeId());
    });

    const tfdvArtifactsPaths: string[] = [];
    tfdvArtifacts.forEach(tfdvArtifact => {
      const uri = tfdvArtifact.getUri();
      if (uri && uri.length > 0) {
        // tfdvArtifactsPaths.push(uri);
        const evalUri = uri + '/eval/stats_tfrecord';
        const trainUri = uri + '/train/stats_tfrecord';
        tfdvArtifactsPaths.push(evalUri);
        tfdvArtifactsPaths.push(trainUri);
      }
    });
    return tfdvArtifactsPaths;
  }

  public static getTfdvArtifactViewers(
    tfdvArtifactPaths: string[],
  ): Array<Promise<HTMLViewerConfig>> {
    return tfdvArtifactPaths.map(async artifactPath => {
      const script = [
        'import tensorflow_data_validation as tfdv',
        "stats = tfdv.load_statistics('" + artifactPath + "')",
        'tfdv.visualize_statistics(stats)',
        //        "stats = tfdv.load_schema_text('" + artifactPath + "')",
        //        'tfdv.display_schema(stats)',
        //        "anomalies = tfdv.load_anomalies_text('" + artifactPath + "/anomalies.pbtxt" + "')",
        //        'tfdv.display_anomalies(anomalies)',
      ];
      const specifiedArguments: any = JSON.parse('{}');
      specifiedArguments.code = script;
      const visualizationData: ApiVisualization = {
        arguments: JSON.stringify(specifiedArguments),
        source: '',
        type: ApiVisualizationType.CUSTOM,
      };
      const visualization = await Apis.buildPythonVisualizationConfig(visualizationData);
      if (!visualization.htmlContent) {
        throw new Error('Failed to build TFDV artifact visualization');
      }
      return {
        htmlContent: visualization.htmlContent,
        type: PlotType.WEB_APP,
      } as HTMLViewerConfig;
    });
  }

  public static filterTfmaArtifactsPaths(
    artifactTypes: ArtifactType[],
    artifacts: Artifact[],
  ): string[] {
    const tfmaArtifactTypes = artifactTypes.filter(artifactType => {
      return artifactType.getName() === 'ModelEvaluation';
    });
    const tfmaArtifactTypeIds = tfmaArtifactTypes.map(artifactType => {
      return artifactType.getId();
    });
    const tfmaArtifacts = artifacts.filter(artifact => {
      return tfmaArtifactTypeIds.includes(artifact.getTypeId());
    });

    const tfmaArtifactPaths: string[] = [];
    tfmaArtifacts.forEach(artifact => {
      const uri = artifact.getUri();
      if (uri && uri.length > 0) {
        tfmaArtifactPaths.push(uri);
      }
    });
    return tfmaArtifactPaths;
  }

  public static getTfmaArtifactViewers(
    tfmaArtifactPaths: string[],
  ): Array<Promise<HTMLViewerConfig>> {
    return tfmaArtifactPaths.map(async artifactPath => {
      const script = [
        'import tensorflow_model_analysis as tfma',
        "tfma_result = tfma.load_eval_result('" + artifactPath + "')",
        'tfma.view.render_slicing_metrics(tfma_result)',
      ];
      const specifiedArguments: any = JSON.parse('{}');
      specifiedArguments.code = script;
      const visualizationData: ApiVisualization = {
        arguments: JSON.stringify(specifiedArguments),
        source: '',
        type: ApiVisualizationType.CUSTOM,
      };
      const visualization = await Apis.buildPythonVisualizationConfig(visualizationData);
      if (!visualization.htmlContent) {
        throw new Error('Failed to build TFMA artifact visualization');
      }
      return {
        htmlContent: visualization.htmlContent,
        type: PlotType.WEB_APP,
      } as HTMLViewerConfig;
    });
  }

  public static async buildTFXArtifactViewerConfig(
    kfpPodName: string,
  ): Promise<HTMLViewerConfig[]> {
    // Since artifact types don't change per run, this can be optimized further so
    // that we don't fetch them on every page load.
    const artifactTypes = await this.getArtifactTypes();
    if (!artifactTypes) {
      throw new Error('Failed getting artifact types');
    }

    const context = await this.getMlmdContext(kfpPodName);
    if (!context) {
      throw new Error('Failed finding corresponding MLMD context');
    }

    const execution = await this.getExecutionInContextWithPodName(kfpPodName, context);
    if (!execution) {
      logger.verbose('Failed finding corresponding MLMD execution');
      return [];
    }

    const artifacts = await this.getOutputArtifactsInExecution(execution);
    if (!artifacts) {
      throw new Error('Failed finding output artifacts in execution');
    }

    const tfdvArtifactPaths = this.filterTfdvArtifactsPaths(artifactTypes, artifacts);
    //const tfmaArtifactPaths = this.filterTfmaArtifactsPaths(artifactTypes, artifacts);
    const tfdvArtifactViewerConfigs = this.getTfdvArtifactViewers(tfdvArtifactPaths);
    //const tfmaArtifactViewerConfigs = this.getTfmaArtifactViewers(tfmaArtifactPaths);
    //return Promise.all(tfdvArtifactViewerConfigs.concat(tfmaArtifactViewerConfigs));
    return Promise.all(tfdvArtifactViewerConfigs);
  }

  public static async buildMarkdownViewerConfig(
    metadata: PlotMetadata,
  ): Promise<MarkdownViewerConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    let markdownContent = '';
    if (metadata.storage === 'inline') {
      markdownContent = metadata.source;
    } else {
      const path = WorkflowParser.parseStoragePath(metadata.source);
      markdownContent = await Apis.readFile(path);
    }

    return {
      markdownContent,
      type: PlotType.MARKDOWN,
    } as MarkdownViewerConfig;
  }

  public static async buildRocCurveConfig(metadata: PlotMetadata): Promise<ROCCurveConfig> {
    if (!metadata.source) {
      throw new Error('Malformed metadata, property "source" is required.');
    }
    if (!metadata.schema) {
      throw new Error('Malformed metadata, property "schema" is required.');
    }
    if (!Array.isArray(metadata.schema)) {
      throw new Error('Malformed schema, must be an array of {"name": string, "type": string}');
    }

    const path = WorkflowParser.parseStoragePath(metadata.source);
    const stringData = csvParseRows((await Apis.readFile(path)).trim());

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
